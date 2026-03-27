package cron

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/store"
)

func openTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestSchedulerEvery(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var count atomic.Int32
	builder := func(agentName string) (*agent.Agent, error) {
		count.Add(1)
		return nil, nil // will cause executeJob to short-circuit with nil agent
	}

	job := &store.CronJob{
		ID:           "every-1",
		Name:         "every-test",
		ScheduleKind: "every",
		Schedule:     store.ScheduleData{EveryMs: 50}, // 50ms interval
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for a few ticks.
	time.Sleep(200 * time.Millisecond)
	sched.Stop()

	// Should have fired at least 2 times (initial + ticker).
	got := count.Load()
	if got < 2 {
		t.Errorf("expected at least 2 executions, got %d", got)
	}
}

func TestSchedulerAt(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var count atomic.Int32
	builder := func(agentName string) (*agent.Agent, error) {
		count.Add(1)
		return nil, nil
	}

	// Schedule 2s from now (RFC3339 has second resolution, need enough buffer).
	at := time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339)
	job := &store.CronJob{
		ID:           "at-1",
		Name:         "at-test",
		ScheduleKind: "at",
		Schedule:     store.ScheduleData{At: at},
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(3 * time.Second)
	sched.Stop()

	if got := count.Load(); got != 1 {
		t.Errorf("expected 1 execution, got %d", got)
	}

	// Verify auto-disable.
	got, err := cs.Get("at-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Enabled {
		t.Error("expected job to be auto-disabled after at execution")
	}
}

func TestSchedulerCron(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var count atomic.Int32
	builder := func(agentName string) (*agent.Agent, error) {
		count.Add(1)
		return nil, nil
	}

	// Use "every second" cron expression (robfig/cron standard parser uses 5-field).
	// For a 5-field expression, the finest resolution is 1 minute, which is too slow for tests.
	// Instead, let's just verify the cron expression is parsed and scheduled.
	job := &store.CronJob{
		ID:           "cron-1",
		Name:         "cron-test",
		ScheduleKind: "cron",
		Schedule:     store.ScheduleData{Expr: "* * * * *"}, // every minute
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Just verify it starts without error. We can't wait 60s for cron to fire.
	sched.Stop()
}

func TestSchedulerOverlapPrevention(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var started atomic.Int32
	var running atomic.Int32
	maxConcurrent := atomic.Int32{}

	builder := func(agentName string) (*agent.Agent, error) {
		started.Add(1)
		cur := running.Add(1)
		// Track max concurrency.
		for {
			old := maxConcurrent.Load()
			if cur <= old || maxConcurrent.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond) // simulate slow execution
		running.Add(-1)
		return nil, nil
	}

	job := &store.CronJob{
		ID:           "overlap-1",
		Name:         "overlap-test",
		ScheduleKind: "every",
		Schedule:     store.ScheduleData{EveryMs: 20}, // fire faster than execution
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	sched.Stop()

	if mc := maxConcurrent.Load(); mc > 1 {
		t.Errorf("overlap detected: max concurrent = %d (want 1)", mc)
	}
}

func TestSchedulerToggle(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var count atomic.Int32
	builder := func(agentName string) (*agent.Agent, error) {
		count.Add(1)
		return nil, nil
	}

	job := &store.CronJob{
		ID:           "toggle-1",
		Name:         "toggle-test",
		ScheduleKind: "every",
		Schedule:     store.ScheduleData{EveryMs: 30},
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Let it fire a few times.
	time.Sleep(100 * time.Millisecond)
	before := count.Load()

	// Disable.
	if err := sched.ToggleJob("toggle-1", false); err != nil {
		t.Fatalf("ToggleJob disable: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	after := count.Load()

	// Should not have fired more after disable.
	if after > before+1 { // +1 for possible in-flight
		t.Errorf("job fired after disable: before=%d after=%d", before, after)
	}

	sched.Stop()
}

func TestSchedulerAddRemove(t *testing.T) {
	db := openTestDB(t)
	cs := db.CronJobs()

	var count atomic.Int32
	builder := func(agentName string) (*agent.Agent, error) {
		count.Add(1)
		return nil, nil
	}

	sched := New(cs, builder)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := sched.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Add dynamically.
	job := &store.CronJob{
		ID:           "dyn-1",
		Name:         "dynamic",
		ScheduleKind: "every",
		Schedule:     store.ScheduleData{EveryMs: 30},
		AgentName:    "test-agent",
		Prompt:       "test prompt",
		Enabled:      true,
	}
	if err := sched.AddJob(job); err != nil {
		t.Fatalf("AddJob: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if count.Load() < 1 {
		t.Error("dynamically added job did not fire")
	}

	// Remove.
	if err := sched.RemoveJob("dyn-1"); err != nil {
		t.Fatalf("RemoveJob: %v", err)
	}
	before := count.Load()
	time.Sleep(100 * time.Millisecond)

	if count.Load() > before+1 {
		t.Error("job still firing after remove")
	}

	sched.Stop()
}

func TestLRUCache(t *testing.T) {
	c := newScheduleCache(2)

	s1, err := parseCronExpr(c, "0 9 * * MON", "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	s2, _ := parseCronExpr(c, "0 10 * * MON", "")
	s3, _ := parseCronExpr(c, "0 11 * * MON", "") // evicts s1

	// s1 should be evicted (cache cap=2).
	_ = s3
	_ = s2

	// Parsing s1 again should succeed (just re-parsed, not cached).
	s1b, err := parseCronExpr(c, "0 9 * * MON", "")
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if s1b == nil {
		t.Error("expected non-nil schedule")
	}
	_ = s1
}

func TestParseCronWithTimezone(t *testing.T) {
	c := newScheduleCache(10)
	s, err := parseCronExpr(c, "0 9 * * MON-FRI", "Asia/Shanghai")
	if err != nil {
		t.Fatalf("parse with tz: %v", err)
	}
	if s == nil {
		t.Error("expected non-nil schedule")
	}

	// Invalid timezone.
	_, err = parseCronExpr(c, "0 9 * * MON", "Invalid/Zone")
	if err == nil {
		t.Error("expected error for invalid timezone")
	}
}
