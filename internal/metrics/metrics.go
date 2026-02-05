package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RequestsTotal counts total HTTP requests
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oubliette_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// RequestDuration tracks request latency
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oubliette_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// ActiveSessions tracks currently active sessions
	ActiveSessions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oubliette_active_sessions",
			Help: "Number of active sessions",
		},
		[]string{"project_id"},
	)

	// ContainersRunning tracks running containers
	ContainersRunning = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oubliette_containers_running",
			Help: "Number of running containers",
		},
	)

	// SessionDuration tracks how long sessions run
	SessionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "oubliette_session_duration_seconds",
			Help:    "Session duration in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600, 1800, 3600},
		},
		[]string{"project_id", "status"},
	)

	// EventBufferDrops tracks dropped events due to buffer overflow
	EventBufferDrops = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oubliette_event_buffer_drops_total",
			Help: "Total number of events dropped due to buffer overflow",
		},
		[]string{"session_id"},
	)

	// ProjectsTotal tracks total number of projects
	ProjectsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oubliette_projects_total",
			Help: "Total number of projects",
		},
	)

	// ToolCalls tracks MCP tool invocations
	ToolCalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oubliette_tool_calls_total",
			Help: "Total number of MCP tool calls",
		},
		[]string{"tool", "status"},
	)
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE support
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Middleware creates an HTTP middleware that records metrics
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)

		RequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(wrapped.statusCode)).Inc()
		RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// normalizePath normalizes URL paths to avoid high cardinality
func normalizePath(path string) string {
	switch path {
	case "/health", "/ready", "/mcp", "/mcp/", "/metrics":
		return path
	default:
		if len(path) > 5 && path[:5] == "/mcp/" {
			return "/mcp"
		}
		return "other"
	}
}

// Handler returns the Prometheus metrics HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordSessionStart increments active session gauge
func RecordSessionStart(projectID string) {
	ActiveSessions.WithLabelValues(projectID).Inc()
}

// RecordSessionEnd decrements active session gauge and records duration
func RecordSessionEnd(projectID, status string, durationSeconds float64) {
	ActiveSessions.WithLabelValues(projectID).Dec()
	SessionDuration.WithLabelValues(projectID, status).Observe(durationSeconds)
}

// RecordToolCall records an MCP tool invocation
func RecordToolCall(tool, status string) {
	ToolCalls.WithLabelValues(tool, status).Inc()
}

// SetContainersRunning sets the running container count
func SetContainersRunning(count float64) {
	ContainersRunning.Set(count)
}

// SetProjectsTotal sets the total project count
func SetProjectsTotal(count float64) {
	ProjectsTotal.Set(count)
}

// RecordEventDrop records an event buffer drop
func RecordEventDrop(sessionID string) {
	EventBufferDrops.WithLabelValues(sessionID).Inc()
}
