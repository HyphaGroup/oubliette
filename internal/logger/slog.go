package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var (
	slogger *slog.Logger
	logFile *os.File
)

// InitSlog initializes the slog-based logger
// If jsonOutput is true, logs are formatted as JSON for production
func InitSlog(logDir string, jsonOutput bool) error {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}

	// Create log file with timestamp
	logFileName := "oubliette-" + time.Now().Format("2006-01-02") + ".log"
	logFilePath := filepath.Join(logDir, logFileName)

	var err error
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	// Write to both stdout and file
	writer := io.MultiWriter(os.Stdout, logFile)

	var handler slog.Handler
	if jsonOutput {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	slogger = slog.New(handler)
	slog.SetDefault(slogger)

	return nil
}

// CloseSlog closes the slog log file
func CloseSlog() error {
	if logFile != nil {
		return logFile.Close()
	}
	return nil
}

// Slog returns the slog.Logger instance for structured logging
func Slog() *slog.Logger {
	if slogger == nil {
		return slog.Default()
	}
	return slogger
}

// WithContext returns a logger with context fields
func WithContext(ctx context.Context) *slog.Logger {
	logger := Slog()

	// Extract common fields from context if available
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		logger = logger.With("request_id", requestID)
	}
	if sessionID := ctx.Value(ContextKeySessionID); sessionID != nil {
		logger = logger.With("session_id", sessionID)
	}
	if projectID := ctx.Value(ContextKeyProjectID); projectID != nil {
		logger = logger.With("project_id", projectID)
	}

	return logger
}

// Context keys for structured logging
type contextKey string

const (
	ContextKeyRequestID contextKey = "request_id"
	ContextKeySessionID contextKey = "session_id"
	ContextKeyProjectID contextKey = "project_id"
)

// InfoContext logs an info message with context
func InfoContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// ErrorContext logs an error with context
func ErrorContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// WarnContext logs a warning with context
func WarnContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// DebugContext logs debug info with context
func DebugContext(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}
