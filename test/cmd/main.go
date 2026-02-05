package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HyphaGroup/oubliette/test/pkg/client"
	"github.com/HyphaGroup/oubliette/test/pkg/coverage"
	"github.com/HyphaGroup/oubliette/test/pkg/repl"
	"github.com/HyphaGroup/oubliette/test/pkg/suites"
	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

func main() {
	// Parse flags
	serverURL := flag.String("server", "http://localhost:8080/mcp", "Oubliette MCP server URL")
	authToken := flag.String("token", "", "Bearer token for authentication (or set OUBLIETTE_TOKEN env var)")
	interactive := flag.Bool("interactive", false, "Start interactive REPL mode")
	interactiveShort := flag.Bool("i", false, "Start interactive REPL mode (shorthand)")
	testMode := flag.Bool("test", false, "Run automated tests")
	coverageReport := flag.Bool("coverage-report", false, "Show test coverage report")
	testFilter := flag.String("filter", "", "Filter tests by name (substring match)")
	testTags := flag.String("tags", "", "Filter tests by tags (comma-separated)")
	excludeTags := flag.String("exclude-tags", "", "Exclude tests with these tags (comma-separated)")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	listTools := flag.Bool("list-tools", false, "List all available tools")
	tool := flag.String("tool", "", "Tool name to invoke")
	params := flag.String("params", "{}", "Tool parameters as JSON")
	flag.Parse()

	// Get auth token from flag or environment
	token := *authToken
	if token == "" {
		token = os.Getenv("OUBLIETTE_TOKEN")
	}

	// Create client
	mcpClient := client.NewMCPClient(*serverURL)
	if token != "" {
		mcpClient.SetAuthToken(token)
	}

	// Test connection
	if err := mcpClient.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to server: %v\n", err)
		os.Exit(1)
	}

	if !*jsonOutput {
		fmt.Printf("✓ Connected to Oubliette MCP server at %s\n\n", *serverURL)
	}

	// Show coverage report if requested
	if *coverageReport {
		// Get test directory (parent of cmd directory)
		testDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get working directory: %v\n", err)
			os.Exit(1)
		}
		// If we're in test/cmd, go up one level to test/
		if strings.HasSuffix(testDir, "/cmd") || strings.HasSuffix(testDir, "\\cmd") {
			testDir = filepath.Dir(testDir)
		}

		analyzer := coverage.NewAnalyzer(mcpClient, testDir)
		report, err := analyzer.Analyze()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to analyze coverage: %v\n", err)
			os.Exit(1)
		}

		report.PrintReport()

		// Print suggestions for untested tools
		if len(report.UntestedList) > 0 {
			fmt.Println("💡 Suggestions:")
			for _, toolName := range report.UntestedList {
				suite := coverage.SuggestTestSuite(toolName)
				fmt.Printf("  • Add test for %s to pkg/suites/%s\n", toolName, suite)
			}
		}

		// Exit with error code if coverage is below 100%
		if report.CoveragePercent < 100.0 {
			os.Exit(1)
		}
		return
	}

	// Run tests if requested
	if *testMode {
		runner := testpkg.NewTestRunner(mcpClient)
		runner.SetVerbose(*verbose)
		runner.SetJSONOutput(*jsonOutput)

		// Parse filter
		filter := testpkg.TestFilter{
			NamePattern: *testFilter,
		}

		if *testTags != "" {
			filter.Tags = strings.Split(*testTags, ",")
		}

		if *excludeTags != "" {
			filter.ExcludeTags = strings.Split(*excludeTags, ",")
		}

		runner.SetFilter(filter)

		// Add test suites
		runner.AddTests(suites.GetBasicTests())
		runner.AddTests(suites.GetProjectTests())
		runner.AddTests(suites.GetWorkspaceTests())     // Workspace management tests
		runner.AddTests(suites.GetContainerTests())
		runner.AddTests(suites.GetSessionTests())
		runner.AddTests(suites.GetRecursionTests())
		runner.AddTests(suites.GetMessagingTests())
		runner.AddTests(suites.GetAuthTests())          // Authentication tests
		runner.AddTests(suites.GetScheduleTests())      // Scheduled tasks tests
		runner.AddTests(suites.GetOpenSpecTests())      // OpenSpec integration tests
		runner.AddTests(suites.GetComprehensiveTests()) // Comprehensive E2E tests
		runner.AddTests(suites.GetManagerTests())       // Manager.sh command tests
		runner.AddTests(suites.GetCLITests())           // CLI binary tests

		// Run tests
		_ = runner.Run()

		// Exit with appropriate code
		os.Exit(runner.ExitCode())
	}

	// Start interactive REPL if requested
	if *interactive || *interactiveShort {
		replInstance := repl.NewREPL(mcpClient)
		if err := replInstance.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "REPL error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// List tools if requested
	if *listTools {
		tools, err := mcpClient.ListTools()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list tools: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Available tools (%d):\n", len(tools))
		for _, t := range tools {
			fmt.Printf("  - %s\n", t.Name)
			if t.Description != "" {
				fmt.Printf("    %s\n", t.Description)
			}
		}
		return
	}

	// Invoke tool if specified
	if *tool != "" {
		// Parse parameters
		var toolParams map[string]interface{}
		if err := json.Unmarshal([]byte(*params), &toolParams); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse parameters: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Invoking tool: %s\n", *tool)
		fmt.Printf("Parameters: %s\n\n", *params)

		// Invoke tool
		result, err := mcpClient.InvokeTool(*tool, toolParams)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to invoke tool: %v\n", err)
			os.Exit(1)
		}

		// Display result
		if result.IsError {
			fmt.Println("❌ Tool returned error:")
		} else {
			fmt.Println("✓ Tool succeeded:")
		}

		content := result.GetToolContent()
		fmt.Println(content)

		if result.IsError {
			os.Exit(1)
		}
		return
	}

	// No action specified
	fmt.Println("Usage:")
	fmt.Println("  Test mode:     oubliette-test --test [--filter <pattern>] [--tags <tags>] [--verbose] [--json]")
	fmt.Println("  Coverage:      oubliette-test --coverage-report")
	fmt.Println("  Interactive:   oubliette-test -i")
	fmt.Println("  List tools:    oubliette-test --list-tools")
	fmt.Println("  Invoke tool:   oubliette-test --tool <name> --params '{\"key\":\"value\"}'")
	fmt.Println("\nExamples:")
	fmt.Println("  oubliette-test --test                          # Run all tests")
	fmt.Println("  oubliette-test --coverage-report               # Show test coverage")
	fmt.Println("  oubliette-test --test --filter connection      # Run tests matching 'connection'")
	fmt.Println("  oubliette-test --test --tags smoke             # Run tests tagged 'smoke'")
	fmt.Println("  oubliette-test --test --verbose                # Run with verbose logging")
	fmt.Println("  oubliette-test --test --json                   # Output as JSON")
	fmt.Println("  oubliette-test -i                              # Start interactive REPL")
	fmt.Println("  oubliette-test --list-tools                    # List all tools")
	fmt.Println("  oubliette-test --tool project_list")
	fmt.Println("  oubliette-test --tool project_create --params '{\"project_name\":\"test-api\"}'")
}
