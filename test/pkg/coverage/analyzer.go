package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HyphaGroup/oubliette/test/pkg/client"
)

// ItemCoverage represents coverage information for a single item (tool, command, or binary)
type ItemCoverage struct {
	Name        string
	Description string
	TestCount   int
	TestedBy    []string
}

// CategoryCoverage represents coverage for a category (MCP, Manager, CLI)
type CategoryCoverage struct {
	Items       map[string]*ItemCoverage
	Total       int
	Tested      int
	Untested    int
	Percent     float64
	UntestedList []string
}

// ToolCoverage is an alias for backwards compatibility
type ToolCoverage = ItemCoverage

// CoverageReport contains the full coverage analysis
type CoverageReport struct {
	// MCP Tools (backwards compatible fields)
	TotalTools      int
	TestedTools     int
	UntestedTools   int
	ToolCoverage    map[string]*ToolCoverage
	UntestedList    []string
	CoveragePercent float64

	// Extended coverage categories
	MCP     *CategoryCoverage
	Manager *CategoryCoverage
	CLI     *CategoryCoverage

	// Overall coverage
	TotalItems      int
	TestedItems     int
	OverallPercent  float64
}

// Analyzer analyzes test coverage for external interfaces
type Analyzer struct {
	client    *client.MCPClient
	testDir   string
	suitesDir string
	repoRoot  string
}

// NewAnalyzer creates a new coverage analyzer
func NewAnalyzer(mcpClient *client.MCPClient, testDir string) *Analyzer {
	// Find repo root (parent of test/)
	repoRoot := filepath.Dir(testDir)
	return &Analyzer{
		client:    mcpClient,
		testDir:   testDir,
		suitesDir: filepath.Join(testDir, "pkg", "suites"),
		repoRoot:  repoRoot,
	}
}

// Analyze performs coverage analysis for all external interfaces
func (a *Analyzer) Analyze() (*CoverageReport, error) {
	report := &CoverageReport{
		ToolCoverage: make(map[string]*ToolCoverage),
		UntestedList: []string{},
	}

	// Discover and analyze MCP tools
	mcpCoverage, err := a.analyzeMCPTools()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze MCP tools: %w", err)
	}
	report.MCP = mcpCoverage

	// Backwards compatibility - populate legacy fields
	report.TotalTools = mcpCoverage.Total
	report.TestedTools = mcpCoverage.Tested
	report.UntestedTools = mcpCoverage.Untested
	report.CoveragePercent = mcpCoverage.Percent
	report.UntestedList = mcpCoverage.UntestedList
	report.ToolCoverage = mcpCoverage.Items

	// Manager commands removed - no longer used
	report.Manager = &CategoryCoverage{
		Items:        make(map[string]*ItemCoverage),
		UntestedList: []string{},
		Total:        0,
		Tested:       0,
		Percent:      100,
	}

	// Discover and analyze CLI binaries
	cliCoverage, err := a.analyzeCLIBinaries()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze CLI binaries: %w", err)
	}
	report.CLI = cliCoverage

	// Calculate overall coverage
	report.TotalItems = report.MCP.Total + report.Manager.Total + report.CLI.Total
	report.TestedItems = report.MCP.Tested + report.Manager.Tested + report.CLI.Tested
	if report.TotalItems > 0 {
		report.OverallPercent = float64(report.TestedItems) / float64(report.TotalItems) * 100
	}

	return report, nil
}

// analyzeMCPTools discovers and analyzes MCP tool coverage
func (a *Analyzer) analyzeMCPTools() (*CategoryCoverage, error) {
	tools, err := a.client.ListTools()
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	coverage := &CategoryCoverage{
		Items:        make(map[string]*ItemCoverage),
		UntestedList: []string{},
	}

	for _, tool := range tools {
		coverage.Items[tool.Name] = &ItemCoverage{
			Name:        tool.Name,
			Description: tool.Description,
			TestCount:   0,
			TestedBy:    []string{},
		}
	}

	// Scan for MCP tool invocations in tests
	if err := a.scanMCPToolUsage(coverage.Items); err != nil {
		return nil, err
	}

	// Calculate stats
	a.calculateCategoryStats(coverage)
	return coverage, nil
}



// analyzeCLIBinaries discovers and analyzes CLI binary coverage
func (a *Analyzer) analyzeCLIBinaries() (*CategoryCoverage, error) {
	binaries, err := a.discoverCLIBinaries()
	if err != nil {
		return nil, err
	}

	coverage := &CategoryCoverage{
		Items:        make(map[string]*ItemCoverage),
		UntestedList: []string{},
	}

	for _, bin := range binaries {
		coverage.Items[bin] = &ItemCoverage{
			Name:      bin,
			TestCount: 0,
			TestedBy:  []string{},
		}
	}

	// Scan for Covers annotations in tests
	if err := a.scanCoversAnnotations(coverage.Items, "cli:"); err != nil {
		return nil, err
	}

	a.calculateCategoryStats(coverage)
	return coverage, nil
}



// discoverCLIBinaries finds oubliette-* executables in repo root
func (a *Analyzer) discoverCLIBinaries() ([]string, error) {
	binaries := []string{}

	entries, err := os.ReadDir(a.repoRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to read repo root: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "oubliette-") && !entry.IsDir() {
			// Check if it's executable
			info, err := entry.Info()
			if err == nil && info.Mode()&0111 != 0 {
				binaries = append(binaries, name)
			}
		}
	}

	// Also include the main oubliette binary if it exists
	if _, err := os.Stat(filepath.Join(a.repoRoot, "oubliette")); err == nil {
		// Not including "oubliette" itself as it's just a symlink/copy of oubliette-server
	}

	sort.Strings(binaries)
	return binaries, nil
}

// scanCoversAnnotations scans tests for Covers field annotations
func (a *Analyzer) scanCoversAnnotations(coverage map[string]*ItemCoverage, prefix string) error {
	return filepath.Walk(a.suitesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		return a.scanFileForCovers(path, coverage, prefix)
	})
}

// scanFileForCovers scans a Go file for Covers annotations
func (a *Analyzer) scanFileForCovers(filename string, coverage map[string]*ItemCoverage, prefix string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	currentTest := ""

	ast.Inspect(node, func(n ast.Node) bool {
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Check if this is a TestCase literal:
		// 1. Explicit type: testpkg.TestCase{...}
		// 2. Implicit type inside []*testpkg.TestCase{...}
		isTestCase := false
		if sel, ok := comp.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "TestCase" {
			isTestCase = true
		} else if comp.Type == nil {
			// Implicit type - check if it has typical TestCase fields
			hasNameField := false
			hasCoversField := false
			for _, elt := range comp.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						if id.Name == "Name" {
							hasNameField = true
						}
						if id.Name == "Covers" {
							hasCoversField = true
						}
					}
				}
			}
			isTestCase = hasNameField && hasCoversField
		}

		if !isTestCase {
			return true
		}

		// Extract Name and Covers fields
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}

			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}

			if key.Name == "Name" {
				if lit, ok := kv.Value.(*ast.BasicLit); ok {
					currentTest = strings.Trim(lit.Value, "\"")
				}
			}

			if key.Name == "Covers" {
				// Parse Covers slice
				if compLit, ok := kv.Value.(*ast.CompositeLit); ok {
					for _, item := range compLit.Elts {
						if lit, ok := item.(*ast.BasicLit); ok {
							coverItem := strings.Trim(lit.Value, "\"")
							if strings.HasPrefix(coverItem, prefix) {
								itemName := strings.TrimPrefix(coverItem, prefix)
								if cov, exists := coverage[itemName]; exists {
									cov.TestCount++
									if currentTest != "" && !contains(cov.TestedBy, currentTest) {
										cov.TestedBy = append(cov.TestedBy, currentTest)
									}
								}
							}
						}
					}
				}
			}
		}

		return true
	})

	return nil
}

// calculateCategoryStats calculates coverage statistics for a category
func (a *Analyzer) calculateCategoryStats(coverage *CategoryCoverage) {
	coverage.Total = len(coverage.Items)
	coverage.Tested = 0
	coverage.UntestedList = []string{}

	for name, item := range coverage.Items {
		if item.TestCount > 0 {
			coverage.Tested++
		} else {
			coverage.UntestedList = append(coverage.UntestedList, name)
		}
	}

	coverage.Untested = len(coverage.UntestedList)
	if coverage.Total > 0 {
		coverage.Percent = float64(coverage.Tested) / float64(coverage.Total) * 100
	}
	sort.Strings(coverage.UntestedList)
}

// scanMCPToolUsage scans Go test files for MCP tool invocations
func (a *Analyzer) scanMCPToolUsage(coverage map[string]*ItemCoverage) error {
	// Walk through test suite directory
	return filepath.Walk(a.suitesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .go files
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		return a.scanFile(path, coverage)
	})
}

// scanFile scans a single Go file for tool invocations
func (a *Analyzer) scanFile(filename string, coverage map[string]*ToolCoverage) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	// Track which test we're in
	currentTest := ""

	// Visit all nodes in the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			// Look for TestCase struct literals
			if sel, ok := x.Type.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "TestCase" {
					// Extract test name from Name field
					for _, elt := range x.Elts {
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == "Name" {
								if lit, ok := kv.Value.(*ast.BasicLit); ok {
									currentTest = strings.Trim(lit.Value, "\"")
								}
							}
						}
					}
				}
			}

		case *ast.CallExpr:
			// Look for InvokeTool calls
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "InvokeTool" && len(x.Args) > 0 {
					// First argument is the tool name
					if lit, ok := x.Args[0].(*ast.BasicLit); ok {
						toolName := strings.Trim(lit.Value, "\"")
						if cov, exists := coverage[toolName]; exists {
							cov.TestCount++
							if currentTest != "" && !contains(cov.TestedBy, currentTest) {
								cov.TestedBy = append(cov.TestedBy, currentTest)
							}
						}
					}
				}
			}
		}
		return true
	})

	return nil
}

// contains checks if a string slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrintReport prints a human-readable coverage report
func (r *CoverageReport) PrintReport() {
	fmt.Println("========================================")
	fmt.Println("     External Surface Coverage Report    ")
	fmt.Println("========================================")
	fmt.Println()

	// Overall summary
	fmt.Println("OVERALL COVERAGE")
	fmt.Printf("  Total Items:    %d\n", r.TotalItems)
	fmt.Printf("  Tested Items:   %d\n", r.TestedItems)
	fmt.Printf("  Coverage:       %.1f%%\n", r.OverallPercent)
	fmt.Println()

	// MCP Tools section
	if r.MCP != nil {
		r.printCategoryReport("MCP Tools", r.MCP)
	}

	// Manager Commands section
	if r.Manager != nil {
		r.printCategoryReport("Manager Commands", r.Manager)
	}

	// CLI Binaries section
	if r.CLI != nil {
		r.printCategoryReport("CLI Binaries", r.CLI)
	}
}

// printCategoryReport prints coverage for a single category
func (r *CoverageReport) printCategoryReport(name string, cat *CategoryCoverage) {
	fmt.Println("----------------------------------------")
	fmt.Printf("%s: %d/%d (%.1f%%)\n", name, cat.Tested, cat.Total, cat.Percent)
	fmt.Println("----------------------------------------")

	if cat.Untested > 0 {
		fmt.Println("Untested:")
		for _, itemName := range cat.UntestedList {
			item := cat.Items[itemName]
			fmt.Printf("  - %s", itemName)
			if item.Description != "" {
				fmt.Printf(" - %s", item.Description)
			}
			fmt.Println()
		}
	}

	// Show tested items
	testedItems := []string{}
	for itemName, item := range cat.Items {
		if item.TestCount > 0 {
			testedItems = append(testedItems, itemName)
		}
	}
	sort.Strings(testedItems)

	if len(testedItems) > 0 {
		fmt.Println("Tested:")
		for _, itemName := range testedItems {
			item := cat.Items[itemName]
			fmt.Printf("  + %s (%d test(s))", itemName, item.TestCount)
			if len(item.TestedBy) > 0 {
				fmt.Printf(": %s", strings.Join(item.TestedBy, ", "))
			}
			fmt.Println()
		}
	}
	fmt.Println()
}

// PrintSummary prints a concise summary
func (r *CoverageReport) PrintSummary() {
	fmt.Printf("Overall: %d/%d (%.1f%%) | MCP: %d/%d | Manager: %d/%d | CLI: %d/%d\n",
		r.TestedItems, r.TotalItems, r.OverallPercent,
		r.MCP.Tested, r.MCP.Total,
		r.Manager.Tested, r.Manager.Total,
		r.CLI.Tested, r.CLI.Total)
}

// GetGaps returns a list of tools that need tests
func (r *CoverageReport) GetGaps() []string {
	gaps := []string{}
	for _, toolName := range r.UntestedList {
		cov := r.ToolCoverage[toolName]
		gap := fmt.Sprintf("%s - %s", toolName, cov.Description)
		gaps = append(gaps, gap)
	}
	return gaps
}

// SuggestTestSuite suggests which test suite should contain tests for a tool
func SuggestTestSuite(toolName string) string {
	name := strings.ToLower(toolName)

	if strings.Contains(name, "project") && (strings.Contains(name, "create") || strings.Contains(name, "delete") || strings.Contains(name, "list") || strings.Contains(name, "get")) {
		return "project.go"
	}

	if strings.Contains(name, "spawn") || strings.Contains(name, "stop") || strings.Contains(name, "exec") || strings.Contains(name, "logs") || strings.Contains(name, "rebuild") {
		return "container.go"
	}

	if strings.Contains(name, "session") || strings.Contains(name, "task") {
		return "session.go"
	}

	if strings.Contains(name, "child") || strings.Contains(name, "recursion") || strings.Contains(name, "depth") || strings.Contains(name, "limit") {
		return "recursion.go"
	}

	if strings.Contains(name, "message") || strings.Contains(name, "interactive") {
		return "messaging.go"
	}

	return "basic.go"
}
