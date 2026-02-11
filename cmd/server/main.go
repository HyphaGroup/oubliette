package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	iofs "io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	agentopencode "github.com/HyphaGroup/oubliette/internal/agent/opencode"
	"github.com/HyphaGroup/oubliette/internal/auth"
	"github.com/HyphaGroup/oubliette/internal/backup"
	"github.com/HyphaGroup/oubliette/internal/cleanup"
	"github.com/HyphaGroup/oubliette/internal/config"
	"github.com/HyphaGroup/oubliette/internal/container"
	"github.com/HyphaGroup/oubliette/internal/container/applecontainer"
	"github.com/HyphaGroup/oubliette/internal/container/docker"
	"github.com/HyphaGroup/oubliette/internal/logger"
	"github.com/HyphaGroup/oubliette/internal/mcp"
	"github.com/HyphaGroup/oubliette/internal/project"
	"github.com/HyphaGroup/oubliette/internal/schedule"
	"github.com/HyphaGroup/oubliette/internal/session"
)

// Version is set at build time via -ldflags "-X main.Version=v1.0.0"
var Version = "dev"

func main() {
	// Check for subcommands before parsing flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cmdInit()
			return
		case "upgrade":
			cmdUpgrade(os.Args[2:])
			return
		case "mcp":
			cmdMCP(os.Args[2:])
			return
		case "token":
			cmdToken(os.Args[2:])
			return
		case "container":
			cmdContainer(os.Args[2:])
			return
		case "--version", "-v":
			fmt.Printf("oubliette %s\n", Version)
			return
		case "--help", "-h", "help":
			printUsage()
			return
		}
	}

	// Default: run server
	runServer()
}

func printUsage() {
	fmt.Printf(`Oubliette %s - Headless AI Agent Automation

Usage: oubliette [command] [options]

Commands:
  (default)    Start the MCP server
  init         Initialize Oubliette directory structure
  upgrade      Upgrade to latest version
  mcp          Configure MCP integration with AI tools
  token        Manage authentication tokens
  container    Manage containers (list, refresh, stop)

Server Options:
  --dir <path>       Oubliette home directory
  --daemon           Start server in background and exit when ready

Config Precedence (for server):
  1. --dir flag
  2. OUBLIETTE_HOME env var
  3. ./.oubliette (if initialized in current directory)
  4. ~/.oubliette (default)

Examples:
  oubliette                              Start the server (auto-detect config)
  oubliette --dir /path/to/oubliette     Start with specific config directory
  oubliette --daemon                     Start in background
  oubliette init                         Set up ~/.oubliette
  oubliette init --dir .                 Set up in current directory
  oubliette mcp --setup claude            Configure MCP for Claude Desktop
  oubliette mcp --setup claude-code      Configure MCP for Claude Code extension
`, Version)
}

func runServer() {
	// Parse command-line flags
	showVersion := flag.Bool("version", false, "Print version and exit")
	dirFlag := flag.String("dir", "", "Oubliette home directory (default: ~/.oubliette)")
	daemonFlag := flag.Bool("daemon", false, "Run in background and exit after server is ready")
	flag.Parse()

	if *showVersion {
		fmt.Printf("oubliette %s\n", Version)
		os.Exit(0)
	}

	// Daemon mode: re-exec in background and wait for health check
	if *daemonFlag {
		runDaemon(*dirFlag)
		return
	}

	// Determine oubliette directory with precedence:
	// 1. --dir flag
	// 2. OUBLIETTE_HOME env var
	// 3. ./.oubliette (current directory)
	// 4. ~/.oubliette (default)
	oublietteDir := resolveOublietteDir(*dirFlag)
	dataDir := filepath.Join(oublietteDir, "data")
	configDir := filepath.Join(oublietteDir, "config")

	// Check if initialized
	if _, err := os.Stat(filepath.Join(configDir, "oubliette.jsonc")); errors.Is(err, iofs.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "Oubliette not initialized. Run 'oubliette init' first.")
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.LoadAll(configDir)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate required configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Standard paths
	projectsDir := filepath.Join(dataDir, "projects")
	logDir := filepath.Join(dataDir, "logs")

	// Initialize logger
	if err := logger.Init(logDir); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.Println("🗝️  Oubliette - Headless AI Agent Automation")
	logger.Println("   \"The city remembered every one of its citizens...\"")
	logger.Println("")

	// Credentials and models are already loaded as part of cfg

	// Log model info
	if cfg.Models != nil && len(cfg.Models.Models) > 0 {
		logger.Printf("🤖 Loaded %d model(s)", len(cfg.Models.Models))
	}

	// Ensure projects directory exists
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		logger.Fatalf("Failed to create projects directory: %v", err)
	}

	// Initialize managers with config values
	projectMgr := project.NewManager(
		projectsDir,
		cfg.ProjectDefaults.MaxRecursionDepth,
		cfg.ProjectDefaults.MaxAgentsPerSession,
		cfg.ProjectDefaults.MaxCostUSD,
	)

	// Set model registry on project manager if available
	if cfg.Models != nil {
		projectMgr.SetModelRegistry(cfg.Models)
	}

	addr := cfg.Server.Address

	// Initialize container runtime based on preference
	var containerRuntime container.Runtime
	runtimePref := container.GetRuntimePreference()

	var baseRuntime container.Runtime
	switch runtimePref {
	case "docker":
		r, err := docker.NewRuntime()
		if err != nil {
			logger.Fatalf("Failed to initialize Docker runtime: %v", err)
		}
		baseRuntime = r
	case "apple-container":
		r, err := applecontainer.NewRuntime()
		if err != nil {
			logger.Fatalf("Failed to initialize Apple Container runtime: %v", err)
		}
		baseRuntime = r
	default: // "auto"
		// Try Apple Container first on macOS ARM64, then Docker
		if r, err := applecontainer.NewRuntime(); err == nil && r.IsAvailable() {
			baseRuntime = r
			logger.Println("🍎 Using Apple Container runtime")
		} else if r, err := docker.NewRuntime(); err == nil && r.IsAvailable() {
			baseRuntime = r
			logger.Println("🐳 Using Docker runtime")
		} else {
			logger.Fatalf("No container runtime available")
		}
	}

	// Wrap runtime with status caching (5 second TTL)
	// This reduces API calls when listing projects or checking status repeatedly
	containerRuntime = container.NewCachedRuntime(baseRuntime, 5*time.Second)
	defer func() { _ = containerRuntime.Close() }()

	// Initialize ImageManager with container mappings from config
	imageManager := container.NewImageManager(cfg.Containers, containerRuntime)

	// Set containers on project manager for image name resolution
	projectMgr.SetContainers(cfg.Containers)

	// Initialize agent runtime (OpenCode)
	agentRuntime := agentopencode.NewRuntime(containerRuntime)
	logger.Println("🤖 Agent runtime: OpenCode")
	if provCred, ok := cfg.Credentials.GetDefaultProviderCredential(); !ok || provCred.APIKey == "" {
		logger.Println("⚠️  WARNING: No API keys configured in oubliette.jsonc")
		logger.Println("   Sessions will fail until you add credentials.providers")
	}

	// Determine Oubliette MCP URL for session-specific configs
	oublietteMCPURL := fmt.Sprintf("http://localhost%s/mcp", addr)

	sessionMgr := session.NewManager(projectsDir, agentRuntime, oublietteMCPURL)

	// Load persistent session index
	if err := sessionMgr.LoadIndex(); err != nil {
		logger.Printf("⚠️  Failed to load session index: %v (will rebuild from disk)", err)
	}

	// Recover stale sessions from previous crashes (mark sessions active >30min as failed)
	if recovered, err := sessionMgr.RecoverStaleSessions(30 * time.Minute); err != nil {
		logger.Printf("⚠️  Failed to recover stale sessions: %v", err)
	} else if recovered > 0 {
		logger.Printf("🔄 Recovered %d stale sessions from previous crash", recovered)
	}

	// Verify container runtime is accessible
	ctx := context.Background()
	if err := containerRuntime.Ping(ctx); err != nil {
		logger.Fatalf("Failed to connect to container runtime: %v", err)
	}

	logger.Printf("✅ Connected to %s runtime\n", containerRuntime.Name())
	logger.Printf("📁 Projects directory: %s\n", projectsDir)
	logger.Printf("📝 Logs directory: %s\n", logDir)
	logger.Println("")

	// Initialize auth store
	authStore, err := auth.NewStore(dataDir)
	if err != nil {
		logger.Fatalf("Failed to initialize auth store: %v", err)
	}
	defer func() { _ = authStore.Close() }()
	logger.Printf("🔐 Auth database: %s/auth.db\n", dataDir)

	// Initialize schedule store
	scheduleStore, err := schedule.NewStore(dataDir)
	if err != nil {
		logger.Fatalf("Failed to initialize schedule store: %v", err)
	}
	defer func() { _ = scheduleStore.Close() }()
	logger.Printf("📅 Schedule database: %s/schedules.db\n", dataDir)

	// Sockets directory for gogol MCP connectivity (relay creates sockets inside containers)
	socketsDir := "/tmp/oubliette-sockets"

	// Create MCP server with default container resource limits
	server := mcp.NewServer(projectMgr, containerRuntime, sessionMgr, authStore, socketsDir, &mcp.ServerConfig{
		ContainerMemory: "4G",
		ContainerCPUs:   4,
		Credentials:     cfg.Credentials,
		ModelRegistry:   cfg.Models,
		ImageManager:    imageManager,
		AgentRuntime:    agentRuntime,
		ScheduleStore:   scheduleStore,
	})

	// Start resource cleanup with defaults
	cleaner := cleanup.New(cleanup.Config{
		ProjectsDir:      projectsDir,
		Interval:         5 * time.Minute,
		SessionRetention: 60 * time.Minute,
		DiskWarnPercent:  80,
		DiskErrorPercent: 90,
	})
	cleaner.Start()

	// Start backup automation if enabled
	var backupMgr *backup.Manager
	if cfg.ConfigDefaults.Backup.Enabled {
		backupDir := cfg.ConfigDefaults.Backup.Directory
		if !filepath.IsAbs(backupDir) {
			backupDir = filepath.Join(dataDir, backupDir)
		}
		backupMgr, err = backup.New(backup.Config{
			ProjectsDir: projectsDir,
			BackupDir:   backupDir,
			Retention:   cfg.ConfigDefaults.Backup.Retention,
			Interval:    time.Duration(cfg.ConfigDefaults.Backup.IntervalHours) * time.Hour,
		})
		if err != nil {
			logger.Printf("⚠️  Failed to initialize backup: %v", err)
		} else {
			backupMgr.Start()
			logger.Printf("📦 Backup automation enabled (dir=%s, retention=%d, interval=%dh)",
				backupDir, cfg.ConfigDefaults.Backup.Retention, cfg.ConfigDefaults.Backup.IntervalHours)
		}
	}

	logger.Println("🚀 Starting Oubliette MCP server...")
	logger.Printf("📡 Server address: http://localhost%s/mcp\n", addr)
	logger.Println("   Use project_*, container_*, session_* tools to manage resources")
	logger.Println("")

	// Setup graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve(addr)
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		logger.Fatalf("Server error: %v", err)
	case sig := <-shutdownChan:
		logger.Printf("⚠️  Received signal %v, initiating graceful shutdown...", sig)

		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Close active sessions
		logger.Println("   Closing active sessions...")
		server.Close()

		// Stop cleanup
		logger.Println("   Stopping cleanup...")
		cleaner.Stop()

		// Stop backup
		if backupMgr != nil {
			logger.Println("   Stopping backup...")
			backupMgr.Stop()
		}

		// Close runtime connection
		logger.Println("   Closing container runtime...")
		_ = containerRuntime.Close()

		// Close auth store
		logger.Println("   Closing auth database...")
		_ = authStore.Close()

		// Close schedule store
		logger.Println("   Closing schedule database...")
		_ = scheduleStore.Close()

		// Close logger
		logger.Println("✅ Shutdown complete")
		_ = logger.Close()

		cancel()
		os.Exit(0) //nolint:gocritic // intentional exit after manual cleanup
	}
}

func cmdInit() {
	// Parse init flags
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dirFlag := fs.String("dir", "", "Directory to initialize (default: ~/.oubliette)")
	_ = fs.Parse(os.Args[2:])

	var oublietteDir string
	if *dirFlag != "" {
		// Use specified directory
		absDir, err := filepath.Abs(*dirFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid directory: %v\n", err)
			os.Exit(1)
		}
		oublietteDir = absDir
	} else {
		// Default to ~/.oubliette
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not determine home directory: %v\n", err)
			os.Exit(1)
		}
		oublietteDir = filepath.Join(homeDir, ".oubliette")
	}

	configDir := filepath.Join(oublietteDir, "config")
	dataDir := filepath.Join(oublietteDir, "data")

	// Check if already initialized (look for config file, not just directory)
	configFile := filepath.Join(configDir, "oubliette.jsonc")
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("⚠️  %s is already initialized.\n", oublietteDir)
		fmt.Print("Overwrite? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	fmt.Println("🗝️  Initializing Oubliette")
	fmt.Println("")

	// Create directory structure
	dirs := []string{
		configDir,
		filepath.Join(dataDir, "projects"),
		filepath.Join(dataDir, "logs"),
		filepath.Join(dataDir, "backups"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("   Created %s\n", dir)
	}

	// Create unified oubliette.jsonc config
	unifiedConfig := `{
  // Oubliette Configuration

  "server": {
    "address": ":8080"
  },

  "credentials": {
    "github": {
      "credentials": {
        "default": {
          "token": "",
          "description": "GitHub token"
        }
      },
      "default": "default"
    },
    "providers": {
      "credentials": {},
      "default": ""
    }
  },

  "defaults": {
    "limits": {
      "max_recursion_depth": 3,
      "max_agents_per_session": 50,
      "max_cost_usd": 10.0
    },
    "agent": {
      "model": "sonnet",
      "autonomy": "off",
      "reasoning": "medium",
      "mcp_servers": {
        "oubliette-parent": {
          "type": "stdio",
          "command": "/usr/local/bin/oubliette-client",
          "args": ["/mcp/relay.sock"]
        }
      }
    },
    "container": {
      "type": "dev"
    },
    "backup": {
      "enabled": false,
      "directory": "data/backups",
      "retention": 7,
      "interval_hours": 24
    }
  },

  "containers": {
    "base": "ghcr.io/hyphagroup/oubliette-base:latest",
    "dev": "ghcr.io/hyphagroup/oubliette-dev:latest"
  },

  "models": {
    "models": {
      "sonnet": {
        "model": "claude-sonnet-4-5",
        "displayName": "Sonnet 4.5",
        "baseUrl": "https://api.anthropic.com",
        "maxOutputTokens": 64000,
        "provider": "anthropic"
      },
      "opus": {
        "model": "claude-opus-4-5",
        "displayName": "Opus 4.5",
        "baseUrl": "https://api.anthropic.com",
        "maxOutputTokens": 64000,
        "provider": "anthropic"
      }
    },
    "defaults": {
      "included_models": ["sonnet", "opus"],
      "session_model": "sonnet",
      "autonomy_mode": "auto-high",
      "reasoning_effort": "medium"
    }
  }
}
`
	configPath := filepath.Join(configDir, "oubliette.jsonc")
	if err := os.WriteFile(configPath, []byte(unifiedConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating oubliette.jsonc: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   Created %s\n", configPath)

	// Create admin token
	fmt.Println("")
	fmt.Println("Creating admin token...")
	authStore, err := auth.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing auth store: %v\n", err)
		os.Exit(1)
	}

	token, tokenID, err := authStore.CreateToken("admin", "admin", nil)
	if err != nil {
		_ = authStore.Close()
		fmt.Fprintf(os.Stderr, "Error creating token: %v\n", err)
		os.Exit(1)
	}
	_ = authStore.Close()

	fmt.Println("")
	fmt.Println("Admin token (save this - it cannot be retrieved later):")
	fmt.Printf("   %s\n", tokenID)

	// Pre-pull container images (skip in dev mode)
	if os.Getenv("OUBLIETTE_DEV") != "1" {
		fmt.Println("")
		fmt.Println("Pulling container images...")

		// Load config to get container definitions
		cfg, err := config.LoadAll(configDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load config for image pull: %v\n", err)
		} else {
			// Initialize runtime
			containerRT, err := initContainerRuntime()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not initialize container runtime: %v\n", err)
			} else {
				defer func() { _ = containerRT.Close() }()

				ctx := context.Background()
				for typeName, imageName := range cfg.Containers {
					fmt.Printf("   Pulling %s (%s)...\n", typeName, imageName)
					if err := containerRT.Pull(ctx, imageName); err != nil {
						fmt.Fprintf(os.Stderr, "   Warning: failed to pull %s: %v\n", imageName, err)
					} else {
						fmt.Printf("   ✅ %s ready\n", typeName)
					}
				}
			}
		}
	} else {
		fmt.Println("")
		fmt.Println("Dev mode: skipping image pull (use ./build.sh to build local images)")
	}

	fmt.Println("")
	fmt.Println("✅ Oubliette initialized!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Printf("   1. Edit %s with your API keys\n", configPath)
	fmt.Println("   2. Run 'oubliette mcp --setup <tool>' to configure your AI tool")
	fmt.Println("   3. Run 'oubliette' to start the server")

	_ = token // silence unused warning
}

func cmdUpgrade(args []string) {
	checkOnly := false
	for _, arg := range args {
		if arg == "--check" || arg == "-c" {
			checkOnly = true
		}
	}

	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	// Query GitHub API for latest release
	resp, err := http.Get("https://api.github.com/repos/HyphaGroup/oubliette/releases/latest")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode == 404 {
		_ = resp.Body.Close()
		fmt.Println("No releases found yet.")
		return
	}

	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		fmt.Fprintf(os.Stderr, "Error: GitHub API returned status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		_ = resp.Body.Close()
		fmt.Fprintf(os.Stderr, "Error parsing release info: %v\n", err)
		os.Exit(1)
	}
	_ = resp.Body.Close()

	latestVersion := release.TagName
	fmt.Printf("Latest version: %s\n", latestVersion)

	// Compare versions (simple string comparison, assumes semver format)
	currentVersion := Version
	if !strings.HasPrefix(currentVersion, "v") {
		currentVersion = "v" + currentVersion
	}

	if currentVersion == latestVersion {
		fmt.Println("")
		fmt.Println("✅ You are already on the latest version.")
		return
	}

	if checkOnly {
		fmt.Println("")
		fmt.Printf("Upgrade available: %s -> %s\n", Version, latestVersion)
		fmt.Println("Run 'oubliette upgrade' to install.")
		return
	}

	// Determine platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	binaryName := fmt.Sprintf("oubliette-%s-%s", goos, goarch)

	// Find download URLs
	var binaryURL, checksumsURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			binaryURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumsURL = asset.BrowserDownloadURL
		}
	}

	if binaryURL == "" {
		fmt.Fprintf(os.Stderr, "Error: No binary found for %s/%s\n", goos, goarch)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Printf("Downloading %s...\n", binaryName)

	// Download binary to temp file
	tmpFile, err := os.CreateTemp("", "oubliette-upgrade-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		os.Exit(1)
	}

	binaryResp, err := http.Get(binaryURL)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		fmt.Fprintf(os.Stderr, "Error downloading binary: %v\n", err)
		os.Exit(1)
	}

	if _, err := io.Copy(tmpFile, binaryResp.Body); err != nil {
		_ = binaryResp.Body.Close()
		_ = os.Remove(tmpFile.Name())
		fmt.Fprintf(os.Stderr, "Error saving binary: %v\n", err)
		os.Exit(1)
	}
	_ = binaryResp.Body.Close()
	_ = tmpFile.Close()

	// Verify checksum if available
	if checksumsURL != "" {
		fmt.Println("Verifying checksum...")
		checksumsResp, err := http.Get(checksumsURL)
		if err == nil {
			checksumsData, _ := io.ReadAll(checksumsResp.Body)
			_ = checksumsResp.Body.Close()

			// Find expected checksum
			var expectedChecksum string
			for _, line := range strings.Split(string(checksumsData), "\n") {
				if strings.Contains(line, binaryName) {
					parts := strings.Fields(line)
					if len(parts) >= 1 {
						expectedChecksum = parts[0]
						break
					}
				}
			}

			if expectedChecksum != "" {
				// Calculate actual checksum
				f, _ := os.Open(tmpFile.Name())
				h := sha256.New()
				_, _ = io.Copy(h, f)
				_ = f.Close()
				actualChecksum := fmt.Sprintf("%x", h.Sum(nil))

				if actualChecksum != expectedChecksum {
					_ = os.Remove(tmpFile.Name())
					fmt.Fprintf(os.Stderr, "Error: Checksum mismatch!\n")
					fmt.Fprintf(os.Stderr, "  Expected: %s\n", expectedChecksum)
					fmt.Fprintf(os.Stderr, "  Actual:   %s\n", actualChecksum)
					os.Exit(1)
				}
				fmt.Println("Checksum verified ✓")
			}
		}
	}

	// Get path to current binary
	currentBinary, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding current binary: %v\n", err)
		os.Exit(1)
	}
	currentBinary, _ = filepath.EvalSymlinks(currentBinary)

	// Replace binary
	fmt.Printf("Replacing %s...\n", currentBinary)

	// Make temp file executable
	_ = os.Chmod(tmpFile.Name(), 0o755)

	// Move temp file to replace current binary
	// First try rename (same filesystem)
	if err := os.Rename(tmpFile.Name(), currentBinary); err != nil {
		// Cross-filesystem, need to copy
		src, err := os.Open(tmpFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening temp file: %v\n", err)
			os.Exit(1)
		}

		dst, err := os.OpenFile(currentBinary, os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			_ = src.Close()
			fmt.Fprintf(os.Stderr, "Error opening binary for writing: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to run with sudo or adjust permissions.\n")
			os.Exit(1)
		}

		if _, err := io.Copy(dst, src); err != nil {
			_ = src.Close()
			_ = dst.Close()
			fmt.Fprintf(os.Stderr, "Error writing binary: %v\n", err)
			os.Exit(1)
		}
		_ = src.Close()
		_ = dst.Close()
	}

	fmt.Println("")
	fmt.Printf("✅ Upgraded from %s to %s\n", Version, latestVersion)
}

func cmdMCP(args []string) {
	// Parse mcp flags
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	setup := fs.String("setup", "", "Tool to configure: claude, claude-code")
	configFlag := fs.String("config", "", "Output MCP config file path (overrides tool default)")
	dirFlag := fs.String("dir", "", "Oubliette directory (default: ~/.oubliette)")
	_ = fs.Parse(args)

	if *setup == "" {
		fmt.Println("Usage: oubliette mcp --setup <tool> [options]")
		fmt.Println("")
		fmt.Println("Tools:")
		fmt.Println("  claude      Claude Desktop")
		fmt.Println("  claude-code Claude Code VS Code extension")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  --config <path>  Output MCP config file (overrides tool default)")
		fmt.Println("  --dir <path>     Oubliette directory (default: ~/.oubliette)")
		os.Exit(1)
	}

	tool := *setup

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not determine home directory: %v\n", err)
		os.Exit(1)
	}

	// Determine oubliette directory first (needed for config path resolution)
	var oublietteDir string
	if *dirFlag != "" {
		oublietteDir, err = filepath.Abs(*dirFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		oublietteDir = filepath.Join(homeDir, ".oubliette")
	}

	// Determine config file path
	var configPath string
	switch {
	case *configFlag != "":
		configPath, err = filepath.Abs(*configFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid config path: %v\n", err)
			os.Exit(1)
		}
	default:
		switch tool {
		case "claude":
			if runtime.GOOS == "darwin" {
				configPath = filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
			} else {
				configPath = filepath.Join(homeDir, ".config", "claude", "claude_desktop_config.json")
			}
		case "claude-code":
			configPath = filepath.Join(homeDir, ".config", "Code", "User", "globalStorage", "anthropic.claude-code", "settings.json")
		default:
			fmt.Fprintf(os.Stderr, "Unknown tool: %s\n", tool)
			fmt.Println("Supported tools: claude, claude-code")
			os.Exit(1)
		}
	}

	fmt.Printf("Setting up MCP for %s...\n", tool)
	fmt.Printf("Config file: %s\n", configPath)
	fmt.Println("")

	// Determine oubliette paths
	dataDir := filepath.Join(oublietteDir, "data")
	binaryPath := filepath.Join(oublietteDir, "bin", "oubliette")

	// Check if oubliette is initialized
	configDir := filepath.Join(oublietteDir, "config")
	if _, err := os.Stat(dataDir); errors.Is(err, iofs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Error: Oubliette is not initialized.\n")
		fmt.Fprintf(os.Stderr, "Run 'oubliette init' first.\n")
		os.Exit(1)
	}

	// Load oubliette config to get server address
	cfg, err := config.LoadAll(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	serverAddr := cfg.Server.Address
	if serverAddr == "" {
		serverAddr = ":8080"
	}
	// Extract port from address (e.g., ":8080" or "localhost:8080")
	port := serverAddr
	if idx := strings.LastIndex(serverAddr, ":"); idx >= 0 {
		port = serverAddr[idx+1:]
	}
	mcpURL := fmt.Sprintf("http://localhost:%s/mcp", port)

	// Create/get auth token
	authStore, err := auth.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening auth store: %v\n", err)
		os.Exit(1)
	}

	// Check for existing MCP token or create one
	tokens, err := authStore.ListTokens()
	if err != nil {
		_ = authStore.Close()
		fmt.Fprintf(os.Stderr, "Error listing tokens: %v\n", err)
		os.Exit(1)
	}

	var tokenID string
	for _, t := range tokens {
		if t.Name == "mcp-"+tool {
			tokenID = t.ID
			break
		}
	}

	if tokenID == "" {
		fmt.Printf("Creating auth token for %s...\n", tool)
		_, tokenID, err = authStore.CreateToken("mcp-"+tool, "admin", nil)
		if err != nil {
			_ = authStore.Close()
			fmt.Fprintf(os.Stderr, "Error creating token: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Using existing token for %s\n", tool)
	}
	_ = authStore.Close()

	// Read existing config or create new
	var mcpConfig map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &mcpConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing existing config: %v\n", err)
			os.Exit(1)
		}
	} else {
		mcpConfig = make(map[string]interface{})
	}

	// Ensure mcpServers key exists
	mcpServers, ok := mcpConfig["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
		mcpConfig["mcpServers"] = mcpServers
	}

	// Add/update oubliette entry (HTTP mode - oubliette is an HTTP MCP server)
	mcpServers["oubliette"] = map[string]interface{}{
		"type": "http",
		"url":  mcpURL,
		"headers": map[string]string{
			"Authorization": "Bearer " + tokenID,
		},
	}

	// Write config
	mcpConfigDir := filepath.Dir(configPath)
	if err := os.MkdirAll(mcpConfigDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	configData, err := json.MarshalIndent(mcpConfig, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting config: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, configData, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")
	fmt.Printf("✅ MCP configured for %s\n", tool)
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Printf("  1. Start the Oubliette server: %s\n", binaryPath)
	switch tool {
	case "claude":
		fmt.Println("  2. Restart Claude Desktop to pick up the new MCP server.")
	case "claude-code":
		fmt.Println("  2. Restart VS Code to pick up the new MCP server.")
	}
}

// cmdToken handles the 'token' subcommand for managing authentication tokens
func cmdToken(args []string) {
	if len(args) < 1 {
		printTokenUsage()
		os.Exit(1)
	}

	oublietteDir := resolveOublietteDir("")
	dataDir := filepath.Join(oublietteDir, "data")

	// Initialize auth store
	store, err := auth.NewStore(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing auth store: %v\n", err)
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "create":
		tokenCreate(store, cmdArgs)
	case "list":
		tokenList(store)
	case "revoke":
		tokenRevoke(store, cmdArgs)
	case "info":
		tokenInfo(store, cmdArgs)
	case "help", "-h", "--help":
		_ = store.Close()
		printTokenUsage()
		return
	default:
		_ = store.Close()
		fmt.Fprintf(os.Stderr, "Unknown token command: %s\n", cmd)
		printTokenUsage()
		os.Exit(1)
	}
	_ = store.Close()
}

func printTokenUsage() {
	fmt.Println(`Token Management

Usage: oubliette token <command> [options]

Commands:
  create    Create a new API token
  list      List all tokens
  revoke    Revoke a token
  info      Get token details
  help      Show this help

Scope Formats:
  admin              Full access to all tools and projects
  admin:ro           Read-only access to all tools and projects
  project:<uuid>     Full access to one project
  project:<uuid>:ro  Read-only access to one project

Examples:
  oubliette token create --name "Local Dev" --scope admin
  oubliette token create --name "Project Alpha" --scope project:abc-123-def
  oubliette token list
  oubliette token revoke oub_xxxx...
  oubliette token info oub_xxxx...`)
}

func tokenCreate(store *auth.Store, args []string) {
	fs := flag.NewFlagSet("token create", flag.ExitOnError)
	name := fs.String("name", "", "Human-readable token name (required)")
	scope := fs.String("scope", "", "Token scope: admin, admin:ro, project:<uuid>, or project:<uuid>:ro (required)")
	_ = fs.Parse(args)

	if *name == "" || *scope == "" {
		fmt.Fprintln(os.Stderr, "Error: --name and --scope are required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Validate scope
	if !isValidTokenScope(*scope) {
		fmt.Fprintf(os.Stderr, "Error: invalid scope '%s'\n", *scope)
		fmt.Fprintln(os.Stderr, "Valid scopes: admin, admin:ro, project:<uuid>, project:<uuid>:ro")
		os.Exit(1)
	}

	token, tokenID, err := store.CreateToken(*name, *scope, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Token created successfully!")
	fmt.Println()
	fmt.Printf("Token ID: %s\n", tokenID)
	fmt.Printf("Name:     %s\n", token.Name)
	fmt.Printf("Scope:    %s\n", token.Scope)
	fmt.Println()
	fmt.Println("IMPORTANT: Save this token now. It cannot be retrieved later.")
}

func tokenList(store *auth.Store) {
	tokens, err := store.ListTokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing tokens: %v\n", err)
		os.Exit(1)
	}

	if len(tokens) == 0 {
		fmt.Println("No tokens found.")
		fmt.Println()
		fmt.Println("Create one with: oubliette token create --name \"My Token\" --scope admin")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tSCOPE\tCREATED\tLAST USED")
	_, _ = fmt.Fprintln(w, "--\t----\t-----\t-------\t---------")

	for _, t := range tokens {
		lastUsed := "never"
		if t.LastUsedAt != nil {
			lastUsed = t.LastUsedAt.Format("2006-01-02 15:04")
		}
		maskedID := maskTokenID(t.ID)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			maskedID,
			t.Name,
			t.Scope,
			t.CreatedAt.Format("2006-01-02 15:04"),
			lastUsed,
		)
	}
	_ = w.Flush()
}

func tokenRevoke(store *auth.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: token ID required")
		fmt.Fprintln(os.Stderr, "Usage: oubliette token revoke <token_id>")
		os.Exit(1)
	}

	tokenID := args[0]
	err := store.RevokeToken(tokenID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error revoking token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Token %s revoked successfully.\n", maskTokenID(tokenID))
}

func tokenInfo(store *auth.Store, args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: token ID required")
		fmt.Fprintln(os.Stderr, "Usage: oubliette token info <token_id>")
		os.Exit(1)
	}

	tokenID := args[0]
	token, err := store.GetToken(tokenID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting token: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Token ID:    %s\n", maskTokenID(token.ID))
	fmt.Printf("Name:        %s\n", token.Name)
	fmt.Printf("Scope:       %s\n", token.Scope)
	fmt.Printf("Created:     %s\n", token.CreatedAt.Format("2006-01-02 15:04:05"))
	if token.LastUsedAt != nil {
		fmt.Printf("Last Used:   %s\n", token.LastUsedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Last Used:   never\n")
	}
	if token.ExpiresAt != nil {
		fmt.Printf("Expires:     %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Expires:     never\n")
	}
}

func isValidTokenScope(scope string) bool {
	// Admin scopes
	if scope == auth.ScopeAdmin || scope == auth.ScopeAdminRO {
		return true
	}
	// Project scopes: project:<uuid> or project:<uuid>:ro
	if strings.HasPrefix(scope, "project:") {
		rest := scope[8:]
		if rest == "" {
			return false
		}
		if strings.HasSuffix(rest, ":ro") {
			return len(rest) > 3
		}
		return true
	}
	return false
}

func maskTokenID(tokenID string) string {
	if len(tokenID) <= 12 {
		return "***"
	}
	return tokenID[:8] + "..." + tokenID[len(tokenID)-4:]
}

// cmdContainer handles the 'container' subcommand
func cmdContainer(args []string) {
	if len(args) < 1 {
		printContainerUsage()
		os.Exit(1)
	}

	cmd := args[0]

	switch cmd {
	case "list":
		containerList()
	case "refresh":
		containerRefresh(args[1:])
	case "stop":
		containerStop(args[1:])
	case "help", "-h", "--help":
		printContainerUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown container command: %s\n", cmd)
		printContainerUsage()
		os.Exit(1)
	}
}

func printContainerUsage() {
	fmt.Println(`Container Management

Usage: oubliette container <command> [options]

Commands:
  list                  List running containers
  refresh <project_id>  Pull latest image and restart container
  stop <project_id>     Stop a container
  stop --all            Stop all containers

Examples:
  oubliette container list
  oubliette container refresh proj_abc123
  oubliette container stop proj_abc123
  oubliette container stop --all`)
}

func containerList() {
	ctx := context.Background()

	// Initialize container runtime
	containerRT, err := initContainerRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = containerRT.Close() }()

	// Get projects directory
	homeDir, _ := os.UserHomeDir()
	projectsDir := filepath.Join(homeDir, ".oubliette", "data", "projects")

	// List project directories
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		fmt.Println("No projects found.")
		return
	}

	fmt.Println("Running containers:")
	fmt.Println()

	found := false
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "PROJECT ID\tCONTAINER\tSTATUS\tIMAGE")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectID := entry.Name()
		containerName := fmt.Sprintf("oubliette-%s", projectID[:min(8, len(projectID))])

		status, err := containerRT.Status(ctx, containerName)
		if err != nil {
			continue
		}

		if status == container.StatusRunning || status == container.StatusCreated {
			found = true
			info, _ := containerRT.Inspect(ctx, containerName)
			imageName := "unknown"
			if info != nil {
				imageName = info.Image
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", projectID[:min(12, len(projectID))]+"...", containerName, status, imageName)
		}
	}
	_ = w.Flush()

	if !found {
		fmt.Println("(none)")
	}
}

func containerRefresh(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: project_id required")
		fmt.Fprintln(os.Stderr, "Usage: oubliette container refresh <project_id>")
		os.Exit(1)
	}

	projectID := args[0]
	ctx := context.Background()

	// Load config
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".oubliette", "config")
	cfg, err := config.LoadAll(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize container runtime
	containerRT, err := initContainerRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Get project metadata
	projectsDir := filepath.Join(homeDir, ".oubliette", "data", "projects")
	metadataPath := filepath.Join(projectsDir, projectID, "metadata.json")

	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		_ = containerRT.Close()
		fmt.Fprintf(os.Stderr, "Error: project not found: %s\n", projectID)
		os.Exit(1)
	}

	var metadata struct {
		ImageName string `json:"image_name"`
	}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		_ = containerRT.Close()
		fmt.Fprintf(os.Stderr, "Error reading project metadata: %v\n", err)
		os.Exit(1)
	}

	imageName := metadata.ImageName
	if imageName == "" {
		// Use default based on container type
		imageName = cfg.Containers["dev"]
	}

	containerName := fmt.Sprintf("oubliette-%s", projectID[:min(8, len(projectID))])

	fmt.Printf("Refreshing container for project %s...\n", projectID[:min(12, len(projectID))])
	fmt.Printf("Pulling image %s...\n", imageName)

	if err := containerRT.Pull(ctx, imageName); err != nil {
		fmt.Fprintf(os.Stderr, "Error pulling image: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Stopping old container...")
	_ = containerRT.Stop(ctx, containerName)
	_ = containerRT.Remove(ctx, containerName, true)

	fmt.Printf("✅ Container refreshed. Start project to create new container.\n")
}

func containerStop(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: project_id or --all required")
		fmt.Fprintln(os.Stderr, "Usage: oubliette container stop <project_id>")
		fmt.Fprintln(os.Stderr, "       oubliette container stop --all")
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize container runtime
	containerRT, err := initContainerRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if args[0] == "--all" {
		// Stop all oubliette containers
		homeDir, _ := os.UserHomeDir()
		projectsDir := filepath.Join(homeDir, ".oubliette", "data", "projects")

		entries, err := os.ReadDir(projectsDir)
		if err != nil {
			_ = containerRT.Close()
			fmt.Println("No projects found.")
			return
		}

		stopped := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectID := entry.Name()
			containerName := fmt.Sprintf("oubliette-%s", projectID[:min(8, len(projectID))])

			status, err := containerRT.Status(ctx, containerName)
			if err != nil {
				continue
			}

			if status == container.StatusRunning {
				fmt.Printf("Stopping %s...\n", containerName)
				if err := containerRT.Stop(ctx, containerName); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: %v\n", err)
				} else {
					stopped++
				}
			}
		}

		_ = containerRT.Close()
		if stopped > 0 {
			fmt.Printf("✅ Stopped %d container(s)\n", stopped)
		} else {
			fmt.Println("No running containers to stop.")
		}
		return
	}

	// Stop specific project
	projectID := args[0]
	containerName := fmt.Sprintf("oubliette-%s", projectID[:min(8, len(projectID))])

	fmt.Printf("Stopping container %s...\n", containerName)
	if err := containerRT.Stop(ctx, containerName); err != nil {
		_ = containerRT.Close()
		fmt.Fprintf(os.Stderr, "Error stopping container: %v\n", err)
		os.Exit(1)
	}

	_ = containerRT.Close()
	fmt.Println("✅ Container stopped")
}

func initContainerRuntime() (container.Runtime, error) {
	runtimePref := container.GetRuntimePreference()

	switch runtimePref {
	case "docker":
		return docker.NewRuntime()
	case "apple-container":
		return applecontainer.NewRuntime()
	default:
		// Auto-detect
		if r, err := applecontainer.NewRuntime(); err == nil && r.IsAvailable() {
			return r, nil
		}
		return docker.NewRuntime()
	}
}

// resolveOublietteDir determines the oubliette home directory with precedence:
// 1. Explicit flag (if provided)
// 2. OUBLIETTE_HOME env var
// 3. ./.oubliette (current directory, if initialized)
// 4. ~/.oubliette (default)
func resolveOublietteDir(flagDir string) string {
	// 1. Explicit flag takes highest precedence
	if flagDir != "" {
		absDir, err := filepath.Abs(flagDir)
		if err != nil {
			log.Fatalf("Invalid directory: %v", err)
		}
		return absDir
	}

	// 2. OUBLIETTE_HOME env var
	if envDir := os.Getenv("OUBLIETTE_HOME"); envDir != "" {
		absDir, err := filepath.Abs(envDir)
		if err != nil {
			log.Fatalf("Invalid OUBLIETTE_HOME: %v", err)
		}
		return absDir
	}

	// 3. Check current directory for config/oubliette.jsonc (direct) or .oubliette/config/oubliette.jsonc
	cwd, err := os.Getwd()
	if err == nil {
		// Check for config directly in cwd (e.g., /path/to/oubliette_test/config/oubliette.jsonc)
		directConfig := filepath.Join(cwd, "config", "oubliette.jsonc")
		if _, err := os.Stat(directConfig); err == nil {
			return cwd
		}
		// Check for .oubliette subdirectory
		localDir := filepath.Join(cwd, ".oubliette")
		configFile := filepath.Join(localDir, "config", "oubliette.jsonc")
		if _, err := os.Stat(configFile); err == nil {
			return localDir
		}
	}

	// 4. Default to ~/.oubliette
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	return filepath.Join(homeDir, ".oubliette")
}

// runDaemon starts the server in background and waits for it to be ready
func runDaemon(dirFlag string) {
	// Get the path to this executable
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding executable: %v\n", err)
		os.Exit(1)
	}

	// Resolve config to get the server address for health check
	oublietteDir := resolveOublietteDir(dirFlag)
	configDir := filepath.Join(oublietteDir, "config")
	cfg, err := config.LoadAll(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	serverAddr := cfg.Server.Address
	if serverAddr == "" {
		serverAddr = ":8080"
	}
	// Extract port
	port := serverAddr
	if idx := strings.LastIndex(serverAddr, ":"); idx >= 0 {
		port = serverAddr[idx+1:]
	}
	healthURL := fmt.Sprintf("http://localhost:%s/health", port)

	// Check if already running
	resp, err := http.Get(healthURL)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("✅ Oubliette already running on port %s\n", port)
			os.Exit(0)
		}
	}

	// Build command string for nohup
	logFile := filepath.Join(oublietteDir, "data", "logs", "daemon.log")
	cmdStr := fmt.Sprintf("nohup %s", executable)
	if dirFlag != "" {
		cmdStr += fmt.Sprintf(" --dir %s", dirFlag)
	}
	cmdStr += fmt.Sprintf(" > %s 2>&1 &", logFile)

	// Start via shell with nohup
	cmd := exec.Command("sh", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting oubliette on port %s...\n", port)

	// Wait for health check to pass
	maxWait := 30 * time.Second
	checkInterval := 500 * time.Millisecond
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				fmt.Printf("✅ Oubliette running on port %s\n", port)
				os.Exit(0)
			}
		}
		time.Sleep(checkInterval)
	}

	fmt.Fprintf(os.Stderr, "Error: server failed to start within %v\n", maxWait)
	fmt.Fprintf(os.Stderr, "Check logs at: %s\n", logFile)
	os.Exit(1)
}
