package schedule

import (
	"context"
	"sync"
	"time"

	"github.com/HyphaGroup/oubliette/internal/logger"
)

// ExecutionFunc is called by the runner to execute a schedule target
// It should return the session ID(s) created and any error
type ExecutionFunc func(ctx context.Context, schedule *Schedule, target *ScheduleTarget) ([]string, error)

// Runner manages scheduled task execution
type Runner struct {
	store       *Store
	executeFunc ExecutionFunc
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// Track running executions per schedule for overlap handling
	running   map[string]int // schedule ID -> count of running executions
	runningMu sync.Mutex
}

// NewRunner creates a new schedule runner
func NewRunner(store *Store, executeFunc ExecutionFunc) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{
		store:       store,
		executeFunc: executeFunc,
		ctx:         ctx,
		cancel:      cancel,
		running:     make(map[string]int),
	}
}

// Start begins the scheduler loop
func (r *Runner) Start() {
	r.wg.Add(1)
	go r.loop()
	logger.Info("Schedule runner started")
}

// Stop gracefully stops the runner and waits for in-flight executions
func (r *Runner) Stop() {
	logger.Info("Stopping schedule runner...")
	r.cancel()
	r.wg.Wait()
	logger.Info("Schedule runner stopped")
}

// loop runs every minute to check for due schedules
func (r *Runner) loop() {
	defer r.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	r.checkDueSchedules()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.checkDueSchedules()
		}
	}
}

// checkDueSchedules finds and executes due schedules
func (r *Runner) checkDueSchedules() {
	now := time.Now()
	schedules, err := r.store.ListDue(now)
	if err != nil {
		logger.Error("Failed to list due schedules: %v", err)
		return
	}

	for _, schedule := range schedules {
		r.executeSchedule(schedule)
	}
}

// executeSchedule executes a single schedule respecting overlap behavior
func (r *Runner) executeSchedule(schedule *Schedule) {
	r.runningMu.Lock()
	runningCount := r.running[schedule.ID]

	// Handle overlap behavior
	switch schedule.OverlapBehavior {
	case OverlapSkip:
		if runningCount > 0 {
			r.runningMu.Unlock()
			logger.Info("Skipping schedule %s (%s): previous execution still running", schedule.ID, schedule.Name)
			r.recordSkippedExecutions(schedule, "previous execution still running")
			return
		}
	case OverlapQueue:
		if runningCount > 0 {
			r.runningMu.Unlock()
			// MVP: just skip with warning, full queue implementation later
			logger.Info("Skipping schedule %s (%s): previous execution still running (queue not yet implemented)", schedule.ID, schedule.Name)
			r.recordSkippedExecutions(schedule, "previous execution still running (queue not implemented)")
			return
		}
	case OverlapParallel:
		// Allow concurrent execution, no check needed
	default:
		// Default to skip behavior
		if runningCount > 0 {
			r.runningMu.Unlock()
			logger.Info("Skipping schedule %s (%s): previous execution still running", schedule.ID, schedule.Name)
			r.recordSkippedExecutions(schedule, "previous execution still running")
			return
		}
	}

	r.running[schedule.ID]++
	r.runningMu.Unlock()

	// Execute in goroutine to not block the ticker
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.runningMu.Lock()
			r.running[schedule.ID]--
			if r.running[schedule.ID] == 0 {
				delete(r.running, schedule.ID)
			}
			r.runningMu.Unlock()
		}()

		r.runSchedule(schedule)
	}()
}

// runSchedule executes the schedule for all targets
func (r *Runner) runSchedule(schedule *Schedule) {
	now := time.Now()
	logger.Info("Executing schedule %s (%s) with %d targets", schedule.ID, schedule.Name, len(schedule.Targets))

	for _, target := range schedule.Targets {
		sessionIDs, err := r.executeFunc(r.ctx, schedule, &target)
		if err != nil {
			logger.Error("Failed to execute schedule %s target %s: %v", schedule.ID, target.ProjectID, err)
			continue
		}
		logger.Info("Schedule %s executed for project %s, sessions: %v", schedule.ID, target.ProjectID, sessionIDs)
	}

	// Calculate next run time
	nextRun, err := NextRun(schedule.CronExpr, now)
	if err != nil {
		logger.Error("Failed to calculate next run for schedule %s: %v", schedule.ID, err)
		return
	}

	// Update run times in store
	if err := r.store.UpdateRunTimes(schedule.ID, now, nextRun); err != nil {
		logger.Error("Failed to update run times for schedule %s: %v", schedule.ID, err)
	}

	logger.Info("Schedule %s completed, next run at %s", schedule.ID, nextRun.Format(time.RFC3339))
}

// IsRunning returns the number of running executions for a schedule
func (r *Runner) IsRunning(scheduleID string) int {
	r.runningMu.Lock()
	defer r.runningMu.Unlock()
	return r.running[scheduleID]
}

// TriggerNow manually triggers a schedule immediately
func (r *Runner) TriggerNow(schedule *Schedule) ([]string, error) {
	logger.Info("Manually triggering schedule %s (%s)", schedule.ID, schedule.Name)

	var allSessionIDs []string
	var lastErr error

	for _, target := range schedule.Targets {
		sessionIDs, err := r.executeFunc(r.ctx, schedule, &target)
		if err != nil {
			logger.Error("Failed to execute schedule %s target %s: %v", schedule.ID, target.ProjectID, err)
			lastErr = err
			continue
		}
		allSessionIDs = append(allSessionIDs, sessionIDs...)
	}

	// Don't update run times for manual trigger - only for scheduled runs
	return allSessionIDs, lastErr
}

// recordSkippedExecutions records a skipped execution for each target
func (r *Runner) recordSkippedExecutions(schedule *Schedule, reason string) {
	now := time.Now()
	for _, target := range schedule.Targets {
		exec := &Execution{
			ScheduleID: schedule.ID,
			TargetID:   target.ID,
			ExecutedAt: now,
			Status:     ExecutionSkipped,
			Error:      reason,
		}
		if err := r.store.RecordExecution(exec); err != nil {
			logger.Error("Failed to record skipped execution for schedule %s: %v", schedule.ID, err)
		}
	}
}
