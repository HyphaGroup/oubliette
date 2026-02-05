// Package load provides load testing for the Oubliette MCP server.
//
// Run with: go test -v -tags=load ./test/load/... -timeout 45m
// Enable pprof: go test -v -tags=load ./test/load/... -cpuprofile cpu.prof -memprofile mem.prof
package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Config for load tests
type Config struct {
	ServerURL       string
	AuthToken       string
	ConcurrentUsers int
	Duration        time.Duration
	RampUpTime      time.Duration
}

// Stats tracks load test metrics
type Stats struct {
	RequestsSent    int64
	RequestsSuccess int64
	RequestsFailed  int64
	TotalLatencyMs  int64
	MinLatencyMs    int64
	MaxLatencyMs    int64
	Errors          sync.Map
}

func (s *Stats) RecordRequest(latencyMs int64, err error) {
	atomic.AddInt64(&s.RequestsSent, 1)
	atomic.AddInt64(&s.TotalLatencyMs, latencyMs)

	if err != nil {
		atomic.AddInt64(&s.RequestsFailed, 1)
		s.Errors.Store(err.Error(), struct{}{})
	} else {
		atomic.AddInt64(&s.RequestsSuccess, 1)
	}

	// Update min/max with CAS
	for {
		min := atomic.LoadInt64(&s.MinLatencyMs)
		if min == 0 || latencyMs < min {
			if atomic.CompareAndSwapInt64(&s.MinLatencyMs, min, latencyMs) {
				break
			}
		} else {
			break
		}
	}
	for {
		max := atomic.LoadInt64(&s.MaxLatencyMs)
		if latencyMs > max {
			if atomic.CompareAndSwapInt64(&s.MaxLatencyMs, max, latencyMs) {
				break
			}
		} else {
			break
		}
	}
}

func (s *Stats) Summary() string {
	sent := atomic.LoadInt64(&s.RequestsSent)
	success := atomic.LoadInt64(&s.RequestsSuccess)
	failed := atomic.LoadInt64(&s.RequestsFailed)
	totalLatency := atomic.LoadInt64(&s.TotalLatencyMs)
	minLatency := atomic.LoadInt64(&s.MinLatencyMs)
	maxLatency := atomic.LoadInt64(&s.MaxLatencyMs)

	avgLatency := float64(0)
	if sent > 0 {
		avgLatency = float64(totalLatency) / float64(sent)
	}

	successRate := float64(0)
	if sent > 0 {
		successRate = float64(success) / float64(sent) * 100
	}

	summary := fmt.Sprintf(`
=== Load Test Results ===
Total Requests:  %d
Successful:      %d (%.2f%%)
Failed:          %d
Latency (ms):
  Min: %d
  Max: %d
  Avg: %.2f
`, sent, success, successRate, failed, minLatency, maxLatency, avgLatency)

	// Collect unique errors
	var errors []string
	s.Errors.Range(func(key, _ interface{}) bool {
		errors = append(errors, key.(string))
		return true
	})
	if len(errors) > 0 {
		summary += "\nErrors:\n"
		for _, e := range errors {
			summary += fmt.Sprintf("  - %s\n", e)
		}
	}

	return summary
}

// MCPClient for load testing
type MCPClient struct {
	baseURL   string
	authToken string
	client    *http.Client
}

func NewMCPClient(baseURL, authToken string) *MCPClient {
	return &MCPClient{
		baseURL:   baseURL,
		authToken: authToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *MCPClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      time.Now().UnixNano(),
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      method,
			"arguments": params,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", result.Error.Code, result.Error.Message)
	}

	return result.Result, nil
}

// TestHealthEndpoint tests the health endpoint under load
func TestHealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	cfg := getConfig()
	stats := &Stats{}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < cfg.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			client := &http.Client{Timeout: 5 * time.Second}

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				start := time.Now()
				resp, err := client.Get(cfg.ServerURL + "/health")
				latency := time.Since(start).Milliseconds()

				if err != nil {
					stats.RecordRequest(latency, err)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					stats.RecordRequest(latency, fmt.Errorf("status %d", resp.StatusCode))
				} else {
					stats.RecordRequest(latency, nil)
				}

				time.Sleep(100 * time.Millisecond) // Rate limit
			}
		}(i)
	}

	wg.Wait()
	t.Log(stats.Summary())

	// Assertions
	sent := atomic.LoadInt64(&stats.RequestsSent)
	success := atomic.LoadInt64(&stats.RequestsSuccess)
	if sent == 0 {
		t.Fatal("No requests sent")
	}
	successRate := float64(success) / float64(sent)
	if successRate < 0.99 {
		t.Errorf("Success rate %.2f%% below 99%% threshold", successRate*100)
	}
}

// TestProjectList tests the project_list endpoint under load
func TestProjectList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	cfg := getConfig()
	if cfg.AuthToken == "" {
		t.Skip("OUBLIETTE_AUTH_TOKEN not set")
	}

	stats := &Stats{}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < cfg.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				start := time.Now()
				_, err := client.Call(ctx, "project_list", map[string]interface{}{})
				latency := time.Since(start).Milliseconds()
				stats.RecordRequest(latency, err)

				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	t.Log(stats.Summary())
}

// TestConcurrentSessions simulates concurrent session operations
func TestConcurrentSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	cfg := getConfig()
	if cfg.AuthToken == "" {
		t.Skip("OUBLIETTE_AUTH_TOKEN not set")
	}

	stats := &Stats{}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	// This test requires an existing project
	projectID := os.Getenv("OUBLIETTE_TEST_PROJECT")
	if projectID == "" {
		t.Skip("OUBLIETTE_TEST_PROJECT not set")
	}

	var wg sync.WaitGroup
	for i := 0; i < cfg.ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
			workspaceID := fmt.Sprintf("load-test-user-%d", userID)

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Send a message to workspace (creates or resumes session)
				start := time.Now()
				_, err := client.Call(ctx, "session_message", map[string]interface{}{
					"project_id":       projectID,
					"workspace_id":     workspaceID,
					"message":          "Load test ping",
					"create_workspace": true,
				})
				latency := time.Since(start).Milliseconds()
				stats.RecordRequest(latency, err)

				time.Sleep(1 * time.Second) // Slower rate for sessions
			}
		}(i)
	}

	wg.Wait()
	t.Log(stats.Summary())
}

// BenchmarkHealthEndpoint benchmarks the health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	cfg := getConfig()
	client := &http.Client{Timeout: 5 * time.Second}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(cfg.ServerURL + "/health")
			if err != nil {
				b.Error(err)
				continue
			}
			resp.Body.Close()
		}
	})
}

// BenchmarkProjectList benchmarks the project_list MCP call
func BenchmarkProjectList(b *testing.B) {
	cfg := getConfig()
	if cfg.AuthToken == "" {
		b.Skip("OUBLIETTE_AUTH_TOKEN not set")
	}

	client := NewMCPClient(cfg.ServerURL+"/mcp", cfg.AuthToken)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Call(ctx, "project_list", map[string]interface{}{})
			if err != nil {
				b.Error(err)
			}
		}
	})
}

func getConfig() Config {
	serverURL := os.Getenv("OUBLIETTE_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	duration := 30 * time.Second
	if d := os.Getenv("OUBLIETTE_LOAD_DURATION"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			duration = parsed
		}
	}

	concurrent := 10
	if c := os.Getenv("OUBLIETTE_LOAD_CONCURRENT"); c != "" {
		fmt.Sscanf(c, "%d", &concurrent)
	}

	return Config{
		ServerURL:       serverURL,
		AuthToken:       os.Getenv("OUBLIETTE_AUTH_TOKEN"),
		ConcurrentUsers: concurrent,
		Duration:        duration,
		RampUpTime:      5 * time.Second,
	}
}
