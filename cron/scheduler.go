package cron

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/store"
	"github.com/ai-gateway/pi-go/types"
	robfigcron "github.com/robfig/cron/v3"
)

// AgentBuilder creates a fresh Agent for the named agent config.
type AgentBuilder func(agentName string) (*agent.Agent, error)

// Scheduler manages cron jobs: scheduling, execution, and dynamic updates.
type Scheduler struct {
	mu      sync.Mutex
	store   *store.CronJobStore
	builder AgentBuilder

	cronEngine *robfigcron.Cron
	entries    map[string]robfigcron.EntryID // jobID -> cron entry (kind=cron)
	timers     map[string]*time.Timer        // jobID -> timer (kind=at)
	tickers    map[string]*tickerEntry       // jobID -> ticker (kind=every)
	running    map[string]bool               // overlap prevention
	cache      *scheduleCache

	ctx    context.Context
	cancel context.CancelFunc
}

// tickerEntry holds a ticker and a stop channel for the goroutine.
type tickerEntry struct {
	ticker *time.Ticker
	stop   chan struct{}
}

// New creates a Scheduler.
func New(st *store.CronJobStore, builder AgentBuilder) *Scheduler {
	return &Scheduler{
		store:   st,
		builder: builder,
		entries: make(map[string]robfigcron.EntryID),
		timers:  make(map[string]*time.Timer),
		tickers: make(map[string]*tickerEntry),
		running: make(map[string]bool),
		cache:   newScheduleCache(lruCacheSize),
	}
}

// Start loads enabled jobs from the store and begins scheduling.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cronEngine = robfigcron.New()
	s.cronEngine.Start()

	jobs, err := s.store.ListEnabled()
	if err != nil {
		return fmt.Errorf("cron: list enabled jobs: %w", err)
	}
	for i := range jobs {
		if err := s.scheduleJobLocked(&jobs[i]); err != nil {
			log.Printf("cron: schedule job %s (%s): %v", jobs[i].Name, jobs[i].ID, err)
		}
	}
	log.Printf("cron: started with %d enabled job(s)", len(jobs))
	return nil
}

// Stop cancels all scheduled jobs and shuts down.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}
	if s.cronEngine != nil {
		s.cronEngine.Stop()
	}
	for id, t := range s.timers {
		t.Stop()
		delete(s.timers, id)
	}
	for id, te := range s.tickers {
		te.ticker.Stop()
		close(te.stop)
		delete(s.tickers, id)
	}
}

// AddJob validates, persists a new job, and schedules it if enabled.
func (s *Scheduler) AddJob(job *store.CronJob) error {
	// Validate before persisting to avoid stale bad data.
	if job.Enabled {
		if err := validateSchedule(job); err != nil {
			return err
		}
	}
	if err := s.store.Create(job); err != nil {
		return err
	}
	if !job.Enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scheduleJobLocked(job)
}

// validateSchedule checks that a job's schedule data is valid for its kind.
func validateSchedule(job *store.CronJob) error {
	switch job.ScheduleKind {
	case "at":
		if job.Schedule.At == "" {
			return fmt.Errorf("'at' schedule requires a date/time")
		}
	case "every":
		if job.Schedule.EveryMs <= 0 {
			return fmt.Errorf("'every' schedule requires a positive interval")
		}
	case "cron":
		if job.Schedule.Expr == "" {
			return fmt.Errorf("'cron' schedule requires an expression")
		}
	}
	return nil
}

// UpdateJob updates a job, cancels old schedule, and reschedules if enabled.
func (s *Scheduler) UpdateJob(job *store.CronJob) error {
	if job.Enabled {
		if err := validateSchedule(job); err != nil {
			return err
		}
	}
	if err := s.store.Update(job); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelJobLocked(job.ID)
	if job.Enabled {
		return s.scheduleJobLocked(job)
	}
	return nil
}

// RemoveJob cancels and deletes a job.
func (s *Scheduler) RemoveJob(id string) error {
	s.mu.Lock()
	s.cancelJobLocked(id)
	s.mu.Unlock()
	return s.store.Delete(id)
}

// ToggleJob enables or disables a job.
func (s *Scheduler) ToggleJob(id string, enabled bool) error {
	if err := s.store.SetEnabled(id, enabled); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelJobLocked(id)
	if enabled {
		job, err := s.store.Get(id)
		if err != nil {
			return err
		}
		return s.scheduleJobLocked(job)
	}
	return nil
}

// TriggerNow runs a job immediately in a goroutine, regardless of its schedule.
// Returns an error if the job is not found; execution errors are logged asynchronously.
func (s *Scheduler) TriggerNow(id string) error {
	job, err := s.store.Get(id)
	if err != nil {
		return fmt.Errorf("cron: trigger: job %s not found: %w", id, err)
	}
	go s.executeJob(job.ID, job.Name, job.AgentName, job.Prompt)
	return nil
}

// Reload stops all schedules and reloads from the store.
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	// Cancel all existing.
	for id := range s.entries {
		s.cancelJobLocked(id)
	}
	for id := range s.timers {
		s.cancelJobLocked(id)
	}
	for id := range s.tickers {
		s.cancelJobLocked(id)
	}
	s.mu.Unlock()

	jobs, err := s.store.ListEnabled()
	if err != nil {
		return fmt.Errorf("cron: reload: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range jobs {
		if err := s.scheduleJobLocked(&jobs[i]); err != nil {
			log.Printf("cron: reload job %s (%s): %v", jobs[i].Name, jobs[i].ID, err)
		}
	}
	log.Printf("cron: reloaded %d enabled job(s)", len(jobs))
	return nil
}

// --- internal scheduling ---

// scheduleJobLocked adds the appropriate timer/ticker/cron for a job. Must hold s.mu.
func (s *Scheduler) scheduleJobLocked(job *store.CronJob) error {
	switch job.ScheduleKind {
	case "at":
		return s.scheduleAt(job)
	case "every":
		return s.scheduleEvery(job)
	case "cron":
		return s.scheduleCron(job)
	default:
		return fmt.Errorf("unknown schedule kind %q", job.ScheduleKind)
	}
}

func (s *Scheduler) scheduleAt(job *store.CronJob) error {
	if job.Schedule.At == "" {
		return fmt.Errorf("cron: 'at' schedule requires a non-empty time value")
	}
	d, err := nextFireAt(job.Schedule.At)
	if err != nil {
		return err
	}
	id := job.ID
	name := job.Name
	agentName := job.AgentName
	prompt := job.Prompt
	t := time.AfterFunc(d, func() {
		s.executeJob(id, name, agentName, prompt)
		// Auto-disable after execution.
		if err := s.store.SetEnabled(id, false); err != nil {
			log.Printf("cron: auto-disable job %s: %v", id, err)
		}
		s.mu.Lock()
		delete(s.timers, id)
		s.mu.Unlock()
	})
	s.timers[id] = t
	log.Printf("cron: scheduled at job %s (%s) in %s", name, id, d)
	return nil
}

func (s *Scheduler) scheduleEvery(job *store.CronJob) error {
	if job.Schedule.EveryMs <= 0 {
		return fmt.Errorf("cron: every_ms must be > 0")
	}
	every := time.Duration(job.Schedule.EveryMs) * time.Millisecond
	initialDelay := nextFireEvery(job.Schedule.EveryMs, job.Schedule.AnchorMs)

	id := job.ID
	name := job.Name
	agentName := job.AgentName
	prompt := job.Prompt
	ctx := s.ctx
	stopCh := make(chan struct{})

	go func() {
		// Wait for initial alignment.
		select {
		case <-time.After(initialDelay):
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		}
		s.executeJob(id, name, agentName, prompt)

		ticker := time.NewTicker(every)
		s.mu.Lock()
		if te, ok := s.tickers[id]; ok {
			te.ticker = ticker
		}
		s.mu.Unlock()

		for {
			select {
			case <-ticker.C:
				s.executeJob(id, name, agentName, prompt)
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-stopCh:
				ticker.Stop()
				return
			}
		}
	}()

	s.tickers[id] = &tickerEntry{ticker: nil, stop: stopCh}
	log.Printf("cron: scheduled every job %s (%s) every %s (initial delay %s)", name, id, every, initialDelay)
	return nil
}

func (s *Scheduler) scheduleCron(job *store.CronJob) error {
	sched, err := parseCronExpr(s.cache, job.Schedule.Expr, job.Schedule.Tz)
	if err != nil {
		return err
	}

	id := job.ID
	name := job.Name
	agentName := job.AgentName
	prompt := job.Prompt

	entryID := s.cronEngine.Schedule(sched, robfigcron.FuncJob(func() {
		s.executeJob(id, name, agentName, prompt)
	}))
	s.entries[id] = entryID
	log.Printf("cron: scheduled cron job %s (%s) expr=%s tz=%s", name, id, job.Schedule.Expr, job.Schedule.Tz)
	return nil
}

// cancelJobLocked cancels any active schedule for a job. Must hold s.mu.
func (s *Scheduler) cancelJobLocked(id string) {
	if entryID, ok := s.entries[id]; ok {
		s.cronEngine.Remove(entryID)
		delete(s.entries, id)
	}
	if t, ok := s.timers[id]; ok {
		t.Stop()
		delete(s.timers, id)
	}
	if te, ok := s.tickers[id]; ok {
		if te.ticker != nil {
			te.ticker.Stop()
		}
		close(te.stop)
		delete(s.tickers, id)
	}
}

// executeJob runs the agent and records history. Skips if already running (overlap prevention).
func (s *Scheduler) executeJob(jobID, jobName, agentName, prompt string) {
	s.mu.Lock()
	if s.running[jobID] {
		s.mu.Unlock()
		log.Printf("cron: skipping job %s (%s) — still running", jobName, jobID)
		return
	}
	s.running[jobID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, jobID)
		s.mu.Unlock()
	}()

	log.Printf("cron: executing job %s (%s) agent=%s", jobName, jobID, agentName)

	historyID, err := s.store.InsertHistory(jobID)
	if err != nil {
		log.Printf("cron: insert history for %s: %v", jobID, err)
	}

	ag, err := s.builder(agentName)
	if err != nil || ag == nil {
		errMsg := "build agent: returned nil"
		if err != nil {
			errMsg = fmt.Sprintf("build agent: %v", err)
		}
		log.Printf("cron: job %s: %s", jobName, errMsg)
		if historyID > 0 {
			_ = s.store.CompleteHistory(historyID, "error", "", errMsg)
		}
		return
	}

	ctx := s.ctx
	if err := ag.Prompt(ctx, prompt); err != nil {
		errMsg := fmt.Sprintf("prompt: %v", err)
		log.Printf("cron: job %s: %s", jobName, errMsg)
		if historyID > 0 {
			_ = s.store.CompleteHistory(historyID, "error", "", errMsg)
		}
		return
	}
	if err := ag.WaitForIdle(ctx); err != nil {
		errMsg := fmt.Sprintf("wait: %v", err)
		log.Printf("cron: job %s: %s", jobName, errMsg)
		if historyID > 0 {
			_ = s.store.CompleteHistory(historyID, "error", collectMessages(ag.State().Messages), errMsg)
		}
		return
	}

	finalState := ag.State()
	result := collectMessages(finalState.Messages)

	// If the agent loop itself errored (e.g. LLM auth failure), surface it.
	if finalState.Error != "" {
		log.Printf("cron: job %s: agent error: %s", jobName, finalState.Error)
		if historyID > 0 {
			_ = s.store.CompleteHistory(historyID, "error", result, finalState.Error)
		}
		return
	}

	// Truncate result to avoid blowing up the DB.
	const maxResult = 10000
	if len(result) > maxResult {
		result = result[:maxResult] + "...(truncated)"
	}

	if historyID > 0 {
		_ = s.store.CompleteHistory(historyID, "success", result, "")
	}
	log.Printf("cron: job %s (%s) completed successfully", jobName, jobID)
}

// collectMessages extracts a human-readable log from the agent's message history.
// It includes assistant text and tool result text (e.g. whipflow_run output).
func collectMessages(msgs []types.Message) string {
	var b strings.Builder
	for _, msg := range msgs {
		switch m := msg.(type) {
		case *types.AssistantMessage:
			for _, block := range m.Content {
				if tb, ok := block.(*types.TextContent); ok {
					if strings.TrimSpace(tb.Text) == "" {
						continue
					}
					if b.Len() > 0 {
						b.WriteString("\n")
					}
					b.WriteString(tb.Text)
				}
			}
		case *types.ToolResultMessage:
			for _, block := range m.Content {
				if tb, ok := block.(*types.TextContent); ok {
					if strings.TrimSpace(tb.Text) == "" {
						continue
					}
					if b.Len() > 0 {
						b.WriteString("\n")
					}
					b.WriteString(fmt.Sprintf("[%s] %s", m.ToolName, tb.Text))
				}
			}
		}
	}
	return b.String()
}
