package schedule

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

var (
	ErrScheduleNotFound = errors.New("schedule not found")
	ErrInvalidCron      = errors.New("invalid cron expression")
)

// Store handles schedule persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new schedule store with SQLite backend
func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "schedules.db")
	// Enable WAL mode and busy timeout for better concurrent access
	db, err := sql.Open("sqlite", dbPath+"?_busy_timeout=5000&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS schedules (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		cron_expr TEXT NOT NULL,
		prompt TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		overlap_behavior TEXT NOT NULL DEFAULT 'skip',
		session_behavior TEXT NOT NULL DEFAULT 'resume',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_run_at DATETIME,
		next_run_at DATETIME,
		creator_token_id TEXT NOT NULL,
		creator_scope TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_schedules_enabled ON schedules(enabled);
	CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at);

	CREATE TABLE IF NOT EXISTS schedule_targets (
		id TEXT PRIMARY KEY,
		schedule_id TEXT NOT NULL,
		project_id TEXT NOT NULL,
		workspace_id TEXT,
		FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_targets_schedule ON schedule_targets(schedule_id);
	CREATE INDEX IF NOT EXISTS idx_targets_project ON schedule_targets(project_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// Create creates a new schedule with its targets
func (s *Store) Create(schedule *Schedule) error {
	// Validate cron expression before inserting
	if err := ValidateCron(schedule.CronExpr); err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if schedule.ID == "" {
		schedule.ID = "sched_" + uuid.New().String()[:8]
	}
	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	// Calculate next run time if not set
	if schedule.NextRunAt == nil && schedule.Enabled {
		nextRun, err := NextRun(schedule.CronExpr, now)
		if err == nil {
			schedule.NextRunAt = &nextRun
		}
	}

	_, err = tx.Exec(`
		INSERT INTO schedules (id, name, cron_expr, prompt, enabled, overlap_behavior, session_behavior, 
		                       created_at, updated_at, last_run_at, next_run_at, creator_token_id, creator_scope)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		schedule.ID, schedule.Name, schedule.CronExpr, schedule.Prompt,
		schedule.Enabled, schedule.OverlapBehavior, schedule.SessionBehavior,
		schedule.CreatedAt, schedule.UpdatedAt, schedule.LastRunAt, schedule.NextRunAt,
		schedule.CreatorTokenID, schedule.CreatorScope,
	)
	if err != nil {
		return fmt.Errorf("failed to insert schedule: %w", err)
	}

	for i := range schedule.Targets {
		target := &schedule.Targets[i]
		if target.ID == "" {
			target.ID = "tgt_" + uuid.New().String()[:8]
		}
		target.ScheduleID = schedule.ID

		_, err = tx.Exec(`
			INSERT INTO schedule_targets (id, schedule_id, project_id, workspace_id)
			VALUES (?, ?, ?, ?)`,
			target.ID, target.ScheduleID, target.ProjectID, target.WorkspaceID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert target: %w", err)
		}
	}

	return tx.Commit()
}

// Get retrieves a schedule by ID with its targets
func (s *Store) Get(id string) (*Schedule, error) {
	var schedule Schedule
	var lastRunAt, nextRunAt sql.NullTime
	var enabled int

	err := s.db.QueryRow(`
		SELECT id, name, cron_expr, prompt, enabled, overlap_behavior, session_behavior,
		       created_at, updated_at, last_run_at, next_run_at, creator_token_id, creator_scope
		FROM schedules WHERE id = ?`, id,
	).Scan(
		&schedule.ID, &schedule.Name, &schedule.CronExpr, &schedule.Prompt,
		&enabled, &schedule.OverlapBehavior, &schedule.SessionBehavior,
		&schedule.CreatedAt, &schedule.UpdatedAt, &lastRunAt, &nextRunAt,
		&schedule.CreatorTokenID, &schedule.CreatorScope,
	)
	if err == sql.ErrNoRows {
		return nil, ErrScheduleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule: %w", err)
	}

	schedule.Enabled = enabled != 0
	if lastRunAt.Valid {
		schedule.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		schedule.NextRunAt = &nextRunAt.Time
	}

	targets, err := s.getTargets(id)
	if err != nil {
		return nil, err
	}
	schedule.Targets = targets

	return &schedule, nil
}

func (s *Store) getTargets(scheduleID string) ([]ScheduleTarget, error) {
	rows, err := s.db.Query(`
		SELECT id, schedule_id, project_id, workspace_id
		FROM schedule_targets WHERE schedule_id = ?`, scheduleID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query targets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var targets []ScheduleTarget
	for rows.Next() {
		var target ScheduleTarget
		var workspaceID sql.NullString

		if err := rows.Scan(&target.ID, &target.ScheduleID, &target.ProjectID, &workspaceID); err != nil {
			return nil, fmt.Errorf("failed to scan target: %w", err)
		}
		if workspaceID.Valid {
			target.WorkspaceID = workspaceID.String
		}
		targets = append(targets, target)
	}

	return targets, rows.Err()
}

// List returns schedules matching the filter
func (s *Store) List(filter *ListFilter) ([]*Schedule, error) {
	query := `
		SELECT DISTINCT s.id, s.name, s.cron_expr, s.prompt, s.enabled, s.overlap_behavior, s.session_behavior,
		       s.created_at, s.updated_at, s.last_run_at, s.next_run_at, s.creator_token_id, s.creator_scope
		FROM schedules s`
	var args []interface{}
	var conditions []string

	if filter != nil {
		if filter.ProjectID != "" {
			query += ` LEFT JOIN schedule_targets t ON s.id = t.schedule_id`
			conditions = append(conditions, "t.project_id = ?")
			args = append(args, filter.ProjectID)
		}
		if filter.Enabled != nil {
			conditions = append(conditions, "s.enabled = ?")
			if *filter.Enabled {
				args = append(args, 1)
			} else {
				args = append(args, 0)
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY s.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		var lastRunAt, nextRunAt sql.NullTime
		var enabled int

		if err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.CronExpr, &schedule.Prompt,
			&enabled, &schedule.OverlapBehavior, &schedule.SessionBehavior,
			&schedule.CreatedAt, &schedule.UpdatedAt, &lastRunAt, &nextRunAt,
			&schedule.CreatorTokenID, &schedule.CreatorScope,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedule.Enabled = enabled != 0
		if lastRunAt.Valid {
			schedule.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			schedule.NextRunAt = &nextRunAt.Time
		}

		targets, err := s.getTargets(schedule.ID)
		if err != nil {
			return nil, err
		}
		schedule.Targets = targets

		schedules = append(schedules, &schedule)
	}

	return schedules, rows.Err()
}

// Update applies partial updates to a schedule
func (s *Store) Update(id string, update *ScheduleUpdate) error {
	// Validate cron expression if being updated
	if update.CronExpr != nil {
		if err := ValidateCron(*update.CronExpr); err != nil {
			return err
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Build dynamic update query
	var setClauses []string
	var args []interface{}
	var cronChanged bool

	if update.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *update.Name)
	}
	if update.CronExpr != nil {
		setClauses = append(setClauses, "cron_expr = ?")
		args = append(args, *update.CronExpr)
		cronChanged = true
	}
	if update.Prompt != nil {
		setClauses = append(setClauses, "prompt = ?")
		args = append(args, *update.Prompt)
	}
	if update.Enabled != nil {
		setClauses = append(setClauses, "enabled = ?")
		if *update.Enabled {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if update.OverlapBehavior != nil {
		setClauses = append(setClauses, "overlap_behavior = ?")
		args = append(args, *update.OverlapBehavior)
	}
	if update.SessionBehavior != nil {
		setClauses = append(setClauses, "session_behavior = ?")
		args = append(args, *update.SessionBehavior)
	}

	if len(setClauses) > 0 {
		setClauses = append(setClauses, "updated_at = ?")
		args = append(args, time.Now())
		args = append(args, id)

		query := "UPDATE schedules SET " + setClauses[0]
		for i := 1; i < len(setClauses); i++ {
			query += ", " + setClauses[i]
		}
		query += " WHERE id = ?"

		result, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to update schedule: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return ErrScheduleNotFound
		}
	}

	// Recalculate next_run_at if cron expression changed
	if cronChanged {
		nextRun, err := NextRun(*update.CronExpr, time.Now())
		if err == nil {
			_, err = tx.Exec("UPDATE schedules SET next_run_at = ? WHERE id = ?", nextRun, id)
			if err != nil {
				return fmt.Errorf("failed to update next_run_at: %w", err)
			}
		}
	}

	// Replace targets if provided
	if update.Targets != nil {
		_, err = tx.Exec("DELETE FROM schedule_targets WHERE schedule_id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete old targets: %w", err)
		}

		for i := range update.Targets {
			target := &update.Targets[i]
			if target.ID == "" {
				target.ID = "tgt_" + uuid.New().String()[:8]
			}
			target.ScheduleID = id

			_, err = tx.Exec(`
				INSERT INTO schedule_targets (id, schedule_id, project_id, workspace_id)
				VALUES (?, ?, ?, ?)`,
				target.ID, target.ScheduleID, target.ProjectID, target.WorkspaceID,
			)
			if err != nil {
				return fmt.Errorf("failed to insert target: %w", err)
			}
		}
	}

	return tx.Commit()
}

// Delete removes a schedule and its targets (CASCADE)
func (s *Store) Delete(id string) error {
	result, err := s.db.Exec("DELETE FROM schedules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrScheduleNotFound
	}

	return nil
}

// ListDue returns enabled schedules where next_run_at <= now
func (s *Store) ListDue(now time.Time) ([]*Schedule, error) {
	rows, err := s.db.Query(`
		SELECT id, name, cron_expr, prompt, enabled, overlap_behavior, session_behavior,
		       created_at, updated_at, last_run_at, next_run_at, creator_token_id, creator_scope
		FROM schedules
		WHERE enabled = 1 AND next_run_at IS NOT NULL AND next_run_at <= ?
		ORDER BY next_run_at ASC`, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list due schedules: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var schedules []*Schedule
	for rows.Next() {
		var schedule Schedule
		var lastRunAt, nextRunAt sql.NullTime
		var enabled int

		if err := rows.Scan(
			&schedule.ID, &schedule.Name, &schedule.CronExpr, &schedule.Prompt,
			&enabled, &schedule.OverlapBehavior, &schedule.SessionBehavior,
			&schedule.CreatedAt, &schedule.UpdatedAt, &lastRunAt, &nextRunAt,
			&schedule.CreatorTokenID, &schedule.CreatorScope,
		); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedule.Enabled = enabled != 0
		if lastRunAt.Valid {
			schedule.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			schedule.NextRunAt = &nextRunAt.Time
		}

		targets, err := s.getTargets(schedule.ID)
		if err != nil {
			return nil, err
		}
		schedule.Targets = targets

		schedules = append(schedules, &schedule)
	}

	return schedules, rows.Err()
}

// UpdateRunTimes updates last_run_at and next_run_at for a schedule
func (s *Store) UpdateRunTimes(id string, lastRun, nextRun time.Time) error {
	result, err := s.db.Exec(`
		UPDATE schedules SET last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ?`,
		lastRun, nextRun, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to update run times: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrScheduleNotFound
	}

	return nil
}
