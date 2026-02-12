package testing

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HyphaGroup/oubliette/test/pkg/client"
)

// TestRunner runs a collection of test cases
type TestRunner struct {
	client  *client.MCPClient
	tests   []*TestCase
	filter  TestFilter
	verbose bool
	jsonOut bool
	results []*TestResult
}

// TestFilter defines filtering criteria for tests
type TestFilter struct {
	NamePattern string   // Match test name (substring)
	Tags        []string // Match tests with any of these tags
	ExcludeTags []string // Exclude tests with any of these tags
}

// NewTestRunner creates a new test runner
func NewTestRunner(mcpClient *client.MCPClient) *TestRunner {
	return &TestRunner{
		client:  mcpClient,
		tests:   []*TestCase{},
		results: []*TestResult{},
		verbose: false,
		jsonOut: false,
	}
}

// AddTest adds a test case to the runner
func (r *TestRunner) AddTest(test *TestCase) {
	r.tests = append(r.tests, test)
}

// AddTests adds multiple test cases to the runner
func (r *TestRunner) AddTests(tests []*TestCase) {
	r.tests = append(r.tests, tests...)
}

// SetFilter sets the test filter
func (r *TestRunner) SetFilter(filter TestFilter) {
	r.filter = filter
}

// SetVerbose enables verbose output
func (r *TestRunner) SetVerbose(verbose bool) {
	r.verbose = verbose
}

// SetJSONOutput enables JSON output
func (r *TestRunner) SetJSONOutput(jsonOut bool) {
	r.jsonOut = jsonOut
}

// shouldRunTest determines if a test should be run based on filters
func (r *TestRunner) shouldRunTest(test *TestCase) bool {
	// Check name pattern
	if r.filter.NamePattern != "" {
		if !strings.Contains(strings.ToLower(test.Name), strings.ToLower(r.filter.NamePattern)) {
			return false
		}
	}

	// Check excluded tags
	for _, excludeTag := range r.filter.ExcludeTags {
		for _, testTag := range test.Tags {
			if testTag == excludeTag {
				return false
			}
		}
	}

	// Check included tags (if specified, test must have at least one)
	if len(r.filter.Tags) > 0 {
		hasTag := false
		for _, includeTag := range r.filter.Tags {
			for _, testTag := range test.Tags {
				if testTag == includeTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

// Run executes all tests that match the filter
func (r *TestRunner) Run() *TestSummary {
	start := time.Now()
	summary := &TestSummary{
		Total:   0,
		Passed:  0,
		Failed:  0,
		Skipped: 0,
		Results: []*TestResult{},
	}

	if !r.jsonOut {
		fmt.Println("🧪 Oubliette Integration Test Suite")
		fmt.Println()
	}

	// Count tests to run
	testsToRun := []*TestCase{}
	for _, test := range r.tests {
		if r.shouldRunTest(test) {
			testsToRun = append(testsToRun, test)
		} else {
			summary.Skipped++
		}
	}

	summary.Total = len(testsToRun)

	if !r.jsonOut {
		fmt.Printf("Running %d test(s)...\n\n", summary.Total)
	}

	// Run each test
	for i, test := range testsToRun {
		if !r.jsonOut {
			fmt.Printf("[%d/%d] %s", i+1, summary.Total, test.Name)
			if test.Description != "" {
				fmt.Printf(": %s", test.Description)
			}
			fmt.Println()
		}

		result := test.Run(r.client)
		summary.Results = append(summary.Results, result)
		r.results = append(r.results, result)

		if result.Passed {
			summary.Passed++
			if !r.jsonOut {
				fmt.Printf("  ✓ PASSED (%.2fs, %d assertions)\n", result.Duration.Seconds(), result.Assertions)
			}
		} else {
			summary.Failed++
			if !r.jsonOut {
				fmt.Printf("  ❌ FAILED (%.2fs, failed at: %s)\n", result.Duration.Seconds(), result.FailedAt)
				if result.Error != nil {
					fmt.Printf("     Error: %v\n", result.Error)
				}
			}
		}

		// Show logs in verbose mode
		if r.verbose && !r.jsonOut && len(result.Logs) > 0 {
			fmt.Println("  Logs:")
			for _, log := range result.Logs {
				fmt.Printf("    %s\n", log)
			}
		}

		if !r.jsonOut {
			fmt.Println()
		}
	}

	summary.Duration = time.Since(start)

	// Print summary or JSON output
	if r.jsonOut {
		r.printJSONOutput(summary)
	} else {
		r.printSummary(summary)
	}

	return summary
}

// printSummary prints a text summary of test results
func (r *TestRunner) printSummary(summary *TestSummary) {
	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("Test Summary:\n")
	fmt.Printf("  Total:   %d\n", summary.Total)
	fmt.Printf("  Passed:  %d\n", summary.Passed)
	fmt.Printf("  Failed:  %d\n", summary.Failed)
	fmt.Printf("  Skipped: %d\n", summary.Skipped)
	fmt.Printf("  Duration: %.2fs\n", summary.Duration.Seconds())

	if summary.Failed > 0 {
		fmt.Println("\nFailed tests:")
		for _, result := range summary.Results {
			if !result.Passed {
				fmt.Printf("  ❌ %s", result.TestName)
				if result.Error != nil {
					fmt.Printf(": %v", result.Error)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println("─────────────────────────────────────────")

	if summary.Failed == 0 {
		fmt.Println("✓ All tests passed!")
	} else {
		fmt.Printf("❌ %d test(s) failed\n", summary.Failed)
	}
}

// printJSONOutput prints test results as JSON
func (r *TestRunner) printJSONOutput(summary *TestSummary) {
	output := map[string]interface{}{
		"total":    summary.Total,
		"passed":   summary.Passed,
		"failed":   summary.Failed,
		"skipped":  summary.Skipped,
		"duration": summary.Duration.Seconds(),
		"results":  []map[string]interface{}{},
	}

	for _, result := range summary.Results {
		resultMap := map[string]interface{}{
			"name":       result.TestName,
			"passed":     result.Passed,
			"duration":   result.Duration.Seconds(),
			"assertions": result.Assertions,
			"failed_at":  result.FailedAt,
		}

		if result.Error != nil {
			resultMap["error"] = result.Error.Error()
		}

		if r.verbose {
			resultMap["logs"] = result.Logs
		}

		output["results"] = append(output["results"].([]map[string]interface{}), resultMap)
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
		return
	}

	fmt.Println(string(jsonBytes))
}

// ExitCode returns the appropriate exit code based on test results
func (r *TestRunner) ExitCode() int {
	for _, result := range r.results {
		if !result.Passed {
			return 1
		}
	}
	return 0
}

// TestSummary contains aggregate results of a test run
type TestSummary struct {
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Results  []*TestResult
}
