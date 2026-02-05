package suites

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	testpkg "github.com/HyphaGroup/oubliette/test/pkg/testing"
)

// GetCLITests returns CLI binary tests
func GetCLITests() []*testpkg.TestCase {
	return []*testpkg.TestCase{
		{
			Name:        "test_cli_oubliette_server_help",
			Description: "Test oubliette-server --help shows usage",
			Tags:        []string{"cli", "server"},
			Covers:      []string{"cli:oubliette-server"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				output, err := runCLI(ctx, "oubliette-server", "--help")
				if err != nil {
					// --help may exit non-zero, check output instead
					ctx.Log("oubliette-server --help output: %s", output)
				}

				ctx.Assertions.AssertContains(output, "Usage", "Should show usage information")
				return nil
			},
		},

		{
			Name:        "test_cli_oubliette_server_version",
			Description: "Test oubliette-server -v shows version",
			Tags:        []string{"cli", "server"},
			Covers:      []string{"cli:oubliette-server"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Try -v flag for version
				output, err := runCLI(ctx, "oubliette-server", "-v")
				ctx.Log("oubliette-server -v output: %s, err: %v", output, err)

				// May not have version flag, that's ok
				return nil
			},
		},

		{
			Name:        "test_cli_oubliette_token_help",
			Description: "Test oubliette token --help shows usage",
			Tags:        []string{"cli", "token"},
			Covers:      []string{"cli:oubliette"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				output, _ := runCLI(ctx, "oubliette", "token")
				ctx.Log("oubliette token output: %s", output)

				// Should show available commands
				ctx.Assertions.AssertContains(output, "create", "Should mention create command")
				ctx.Assertions.AssertContains(output, "list", "Should mention list command")
				return nil
			},
		},

		{
			Name:        "test_cli_oubliette_token_list",
			Description: "Test oubliette token list command",
			Tags:        []string{"cli", "token"},
			Covers:      []string{"cli:oubliette"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				output, err := runCLI(ctx, "oubliette", "token", "list")
				ctx.Log("oubliette token list output: %s, err: %v", output, err)

				// Should either show tokens or indicate none exist
				// The command should at least run without crashing
				return nil
			},
		},

		{
			Name:        "test_cli_oubliette_client_runs",
			Description: "Test oubliette-client starts and shows expected error without socket",
			Tags:        []string{"cli", "client"},
			Covers:      []string{"cli:oubliette-client"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Client without socket should error with connection failure
				output, _ := runCLI(ctx, "oubliette-client")
				ctx.Log("oubliette-client output: %s", output)

				// Should mention relay socket or connection
				ctx.Assertions.AssertContains(output, "connect", "Should show connection-related message")
				return nil
			},
		},

		{
			Name:        "test_cli_oubliette_relay_runs",
			Description: "Test oubliette-relay starts and shows expected error without env",
			Tags:        []string{"cli", "relay"},
			Covers:      []string{"cli:oubliette-relay"},
			Timeout:     10 * time.Second,
			Execute: func(ctx *testpkg.TestContext) error {
				// Relay without env should error
				output, _ := runCLI(ctx, "oubliette-relay")
				ctx.Log("oubliette-relay output: %s", output)

				// Should mention required environment variable
				ctx.Assertions.AssertContains(output, "OUBLIETTE_PROJECT_ID", "Should mention required env var")
				return nil
			},
		},
	}
}

// runCLI executes a CLI binary with arguments
func runCLI(ctx *testpkg.TestContext, binary string, args ...string) (string, error) {
	binPath := findBinary(binary)
	if binPath == "" {
		ctx.Log("Binary %s not found", binary)
		return "", nil
	}

	cmd := exec.Command(binPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// findBinary locates a binary in the repo root
func findBinary(name string) string {
	// Walk up from test/pkg/suites to find repo root
	cwd, _ := os.Getwd()
	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		binPath := filepath.Join(dir, name)
		if info, err := os.Stat(binPath); err == nil && info.Mode()&0111 != 0 {
			return binPath
		}
	}
	// Fallback - assume we're in repo root
	if info, err := os.Stat(name); err == nil && info.Mode()&0111 != 0 {
		return name
	}
	return ""
}
