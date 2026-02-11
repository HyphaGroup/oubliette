package audit

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Operation represents the type of auditable operation
type Operation string

const (
	OpProjectCreate Operation = "project.create"
	OpProjectDelete Operation = "project.delete"
	OpTokenCreate   Operation = "token.create"
	OpTokenRevoke   Operation = "token.revoke"
)

// Event represents an audit log entry
type Event struct {
	Timestamp   time.Time              `json:"timestamp"`
	Operation   Operation              `json:"operation"`
	TokenID     string                 `json:"token_id,omitempty"`
	TokenScope  string                 `json:"token_scope,omitempty"`
	ProjectID   string                 `json:"project_id,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	WorkspaceID string                 `json:"workspace_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// Logger handles audit logging
type Logger struct {
	logger  *slog.Logger
	enabled bool
	mu      sync.RWMutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default audit logger
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(true)
	})
	return defaultLogger
}

// New creates a new audit logger
func New(enabled bool) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &Logger{
		logger:  slog.New(handler),
		enabled: enabled,
	}
}

// SetEnabled enables or disables audit logging
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// Log records an audit event
func (l *Logger) Log(event *Event) {
	l.mu.RLock()
	enabled := l.enabled
	l.mu.RUnlock()

	if !enabled {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	attrs := []any{
		slog.String("audit", "true"),
		slog.String("operation", string(event.Operation)),
		slog.Bool("success", event.Success),
	}

	if event.TokenID != "" {
		attrs = append(attrs, slog.String("token_id", maskToken(event.TokenID)))
	}
	if event.TokenScope != "" {
		attrs = append(attrs, slog.String("token_scope", event.TokenScope))
	}
	if event.ProjectID != "" {
		attrs = append(attrs, slog.String("project_id", event.ProjectID))
	}
	if event.SessionID != "" {
		attrs = append(attrs, slog.String("session_id", event.SessionID))
	}
	if event.WorkspaceID != "" {
		attrs = append(attrs, slog.String("workspace_id", event.WorkspaceID))
	}
	if event.RequestID != "" {
		attrs = append(attrs, slog.String("request_id", event.RequestID))
	}
	if event.Error != "" {
		attrs = append(attrs, slog.String("error", event.Error))
	}
	if event.Details != nil {
		detailsJSON, _ := json.Marshal(event.Details)
		attrs = append(attrs, slog.String("details", string(detailsJSON)))
	}

	l.logger.Info("AUDIT", attrs...)
}

// LogSuccess records a successful operation
func (l *Logger) LogSuccess(op Operation, tokenID, tokenScope, projectID string) {
	l.Log(&Event{
		Operation:  op,
		TokenID:    tokenID,
		TokenScope: tokenScope,
		ProjectID:  projectID,
		Success:    true,
	})
}

// LogFailure records a failed operation
func (l *Logger) LogFailure(op Operation, tokenID, tokenScope, projectID string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	l.Log(&Event{
		Operation:  op,
		TokenID:    tokenID,
		TokenScope: tokenScope,
		ProjectID:  projectID,
		Success:    false,
		Error:      errMsg,
	})
}

func maskToken(tokenID string) string {
	if len(tokenID) <= 12 {
		return "***"
	}
	return tokenID[:8] + "..."
}

// Convenience functions using default logger

func Log(event *Event) {
	Default().Log(event)
}

func LogSuccess(op Operation, tokenID, tokenScope, projectID string) {
	Default().LogSuccess(op, tokenID, tokenScope, projectID)
}

func LogFailure(op Operation, tokenID, tokenScope, projectID string, err error) {
	Default().LogFailure(op, tokenID, tokenScope, projectID, err)
}
