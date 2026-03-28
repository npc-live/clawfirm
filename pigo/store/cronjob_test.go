package store_test

import (
	"testing"

	"github.com/ai-gateway/pi-go/store"
)

func TestCronJobCRUD(t *testing.T) {
	cs := openTestDB(t).CronJobs()

	job := &store.CronJob{
		ID:           "job-1",
		Name:         "test-job",
		ScheduleKind: "every",
		Schedule:     store.ScheduleData{EveryMs: 60000},
		AgentName:    "assistant",
		Prompt:       "do something",
		Enabled:      true,
	}

	// Create
	if err := cs.Create(job); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := cs.Get("job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "test-job" || got.ScheduleKind != "every" || got.Schedule.EveryMs != 60000 {
		t.Errorf("Get mismatch: %+v", got)
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}

	// Update
	got.Prompt = "updated prompt"
	if err := cs.Update(got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got2, _ := cs.Get("job-1")
	if got2.Prompt != "updated prompt" {
		t.Errorf("Update: want %q got %q", "updated prompt", got2.Prompt)
	}

	// Delete
	if err := cs.Delete("job-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = cs.Get("job-1")
	if err == nil {
		t.Error("expected error after Delete")
	}
}

func TestCronJobList(t *testing.T) {
	cs := openTestDB(t).CronJobs()

	_ = cs.Create(&store.CronJob{ID: "a", Name: "a", ScheduleKind: "cron", Schedule: store.ScheduleData{Expr: "* * * * *"}, AgentName: "ag", Prompt: "p", Enabled: true})
	_ = cs.Create(&store.CronJob{ID: "b", Name: "b", ScheduleKind: "at", Schedule: store.ScheduleData{At: "2099-01-01T00:00:00Z"}, AgentName: "ag", Prompt: "p", Enabled: false})

	all, err := cs.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("List: want 2 got %d", len(all))
	}

	enabled, err := cs.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(enabled) != 1 {
		t.Errorf("ListEnabled: want 1 got %d", len(enabled))
	}
}

func TestCronJobFindByName(t *testing.T) {
	cs := openTestDB(t).CronJobs()
	_ = cs.Create(&store.CronJob{ID: "x", Name: "unique-name", ScheduleKind: "every", Schedule: store.ScheduleData{EveryMs: 1000}, AgentName: "ag", Prompt: "p", Enabled: true})

	job, found, err := cs.FindByName("unique-name")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if !found || job.ID != "x" {
		t.Errorf("FindByName: found=%v id=%v", found, job)
	}

	_, found, err = cs.FindByName("nonexistent")
	if err != nil {
		t.Fatalf("FindByName missing: %v", err)
	}
	if found {
		t.Error("expected not found")
	}
}

func TestCronJobSetEnabled(t *testing.T) {
	cs := openTestDB(t).CronJobs()
	_ = cs.Create(&store.CronJob{ID: "e1", Name: "e1", ScheduleKind: "cron", Schedule: store.ScheduleData{Expr: "* * * * *"}, AgentName: "ag", Prompt: "p", Enabled: true})

	if err := cs.SetEnabled("e1", false); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	got, _ := cs.Get("e1")
	if got.Enabled {
		t.Error("expected disabled")
	}
}

func TestCronJobHistory(t *testing.T) {
	cs := openTestDB(t).CronJobs()
	_ = cs.Create(&store.CronJob{ID: "h1", Name: "h1", ScheduleKind: "every", Schedule: store.ScheduleData{EveryMs: 1000}, AgentName: "ag", Prompt: "p", Enabled: true})

	// Insert history
	hid, err := cs.InsertHistory("h1")
	if err != nil {
		t.Fatalf("InsertHistory: %v", err)
	}
	if hid <= 0 {
		t.Fatalf("InsertHistory: invalid id %d", hid)
	}

	// Complete history
	if err := cs.CompleteHistory(hid, "success", "result text", ""); err != nil {
		t.Fatalf("CompleteHistory: %v", err)
	}

	// List history for job
	hist, err := cs.ListHistory("h1", 10)
	if err != nil {
		t.Fatalf("ListHistory: %v", err)
	}
	if len(hist) != 1 {
		t.Fatalf("ListHistory: want 1 got %d", len(hist))
	}
	if hist[0].Status != "success" || hist[0].ResultText != "result text" {
		t.Errorf("ListHistory: %+v", hist[0])
	}

	// Insert another history for different job (should not appear)
	_ = cs.Create(&store.CronJob{ID: "h2", Name: "h2", ScheduleKind: "every", Schedule: store.ScheduleData{EveryMs: 1000}, AgentName: "ag", Prompt: "p", Enabled: true})
	hid2, _ := cs.InsertHistory("h2")
	_ = cs.CompleteHistory(hid2, "error", "", "something broke")

	// ListHistoryAll
	all, err := cs.ListHistoryAll(10)
	if err != nil {
		t.Fatalf("ListHistoryAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("ListHistoryAll: want 2 got %d", len(all))
	}

	// ListHistory for job "h1" only (should be 1, not 2).
	h1Hist, err := cs.ListHistory("h1", 10)
	if err != nil {
		t.Fatalf("ListHistory h1: %v", err)
	}
	if len(h1Hist) != 1 {
		t.Errorf("ListHistory h1: want 1 got %d", len(h1Hist))
	}
}
