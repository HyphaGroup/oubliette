# Oubliette Development Guide

## Hot Reload Options

### Option 1: Air (Recommended) ⭐

**What it is:** File watcher that rebuilds and restarts the server on changes.

**Pros:**
- Zero code changes needed
- Fast reload (1-2 seconds)
- MCP clients auto-reconnect
- Handles build errors gracefully
- Industry standard for Go development

**Cons:**
- Requires extra dependency (`air`)
- Brief disconnect during restart (~1 second)

**Usage:**
```bash
./dev.sh
```

Air watches all `.go` files and automatically:
1. Detects file changes
2. Rebuilds the binary
3. Gracefully stops old process
4. Starts new process
5. MCP client reconnects automatically

### Option 2: entr (Lightweight Alternative)

If you prefer minimal dependencies:

```bash
# Install entr (if not already installed)
brew install entr  # macOS
# or: apt-get install entr  # Linux

# Watch and rebuild
find . -name '*.go' | entr -r sh -c 'go build -o oubliette ./cmd/server && ./oubliette'
```

### Option 3: Make-based Reload

Create a `Makefile`:
```makefile
.PHONY: dev
dev:
	@while true; do \
		inotifywait -r -e modify,create,delete ./cmd ./internal; \
		make build && make run; \
	done
```

### Option 4: Plugin Architecture (Advanced)

For true hot reload without restart, you'd need to refactor handlers as plugins:

**Pros:**
- No server restart
- Zero downtime
- Keep active connections

**Cons:**
- Complex implementation
- Go doesn't support dynamic loading well
- Would require architectural changes

**Not recommended** for this project due to complexity vs benefit.

## Development Workflow

### Recommended Setup

1. **Terminal 1:** Run hot reload
   ```bash
   ./dev.sh
   ```

2. **Terminal 2:** Watch logs in real-time
   ```bash
   tail -f logs/oubliette-*.log
   ```

3. **Terminal 3:** Run tests on file changes
   ```bash
   find . -name '*.go' | entr -c go test ./...
   ```

4. **Editor:** Make changes to `.go` files

5. **Result:** Server automatically rebuilds and restarts within 1-2 seconds

### Testing Hot Reload

1. Start dev server: `./dev.sh`
2. Connect your MCP client to the server
3. Make a change to any handler in `internal/mcp/handlers.go`
4. Save the file
5. Watch the terminal - you'll see:
   ```
   building...
   running...
   ```
6. MCP client will briefly reconnect
7. Test your changes immediately!

## Why Air Works Well with MCP

**MCP Protocol Design:**
- Clients expect servers to restart
- Built-in reconnection logic
- Stateless request/response model
- Connection state separate from business logic

**What Happens During Reload:**
1. Air detects file change
2. Builds new binary (~1 second)
3. Sends SIGTERM to old process
4. Old process stops gracefully
5. New process starts (~0.5 seconds)
6. MCP client detects disconnect
7. Client auto-reconnects (~0.5 seconds)
8. **Total downtime:** ~2 seconds

**Active Sessions:**
- Gogol sessions are file-based (persistent)
- Container state is preserved
- Only the management server restarts
- Running Claude tasks continue unaffected

## Alternative: Systemd with Auto-Restart

For production-like development:

```ini
# /etc/systemd/system/oubliette.service
[Unit]
Description=Oubliette MCP Server
After=network.target docker.service

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/oubliette
ExecStart=/path/to/oubliette/oubliette
Restart=always
RestartSec=1

[Install]
WantedBy=multi-user.target
```

Then:
```bash
# Reload after code changes
sudo systemctl restart oubliette

# Watch logs
journalctl -u oubliette -f
```

## Debugging Tips

### Enable Verbose Logging

```bash
# Set environment variable
DEBUG=1 ./dev.sh
```

### Attach Debugger (Delve)

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug ./cmd/server --headless --listen=:2345 --api-version=2

# In VS Code, connect to port 2345
```

### Profile Performance

```bash
# Build with profiling
go build -o oubliette -pprof ./cmd/server

# Run and expose pprof endpoint
./oubliette

# In another terminal, analyze
go tool pprof http://localhost:6060/debug/pprof/heap
```

## Best Practices

1. **Use hot reload for handler development** - Fast iteration on MCP tool logic
2. **Use manual build for Docker changes** - Rebuilding images is slow anyway
3. **Run tests on save** - Catch errors before server restart
4. **Keep sessions short during dev** - Easier to test from scratch
5. **Use separate test projects** - Don't pollute your production projects

## Troubleshooting

**Air not rebuilding?**
- Check `.air.toml` exclude paths
- Verify file extensions are in `include_ext`
- Look at `tmp/build-errors.log`

**Server not starting after rebuild?**
- Check for port conflicts: `lsof -i :8080`
- Verify `.env` file exists
- Check logs: `tail -f logs/oubliette-*.log`

**MCP not reconnecting?**
- Check MCP client settings
- Verify server address is correct
- Try manual reconnect in your MCP client
