# Oubliette Test Suite

## Unit Tests

```bash
go test ./... -race
```

## Load Tests

Load tests verify system behavior under high concurrency.

### Prerequisites
- Running Oubliette server
- Valid auth token (for authenticated tests)

### Quick Start
```bash
# Start server in another terminal
go run ./cmd/server

# Run load tests (default: 10 users, 30s duration)
./test/load/run_load_test.sh

# Custom parameters
./test/load/run_load_test.sh 60s 50  # 60 seconds, 50 users
```

### Environment Variables
- `OUBLIETTE_SERVER_URL` - Server URL (default: http://localhost:8080)
- `OUBLIETTE_AUTH_TOKEN` - Auth token for authenticated tests
- `OUBLIETTE_TEST_PROJECT` - Project ID for session tests
- `OUBLIETTE_LOAD_DURATION` - Test duration (default: 30s)
- `OUBLIETTE_LOAD_CONCURRENT` - Concurrent users (default: 10)

### Running with Profiling
```bash
cd test/load
go test -v -run TestHealthEndpoint -cpuprofile cpu.prof -memprofile mem.prof .
go tool pprof cpu.prof
```

## Chaos Tests

Chaos tests verify graceful degradation under failure conditions.

### Prerequisites
- Running Oubliette server
- Admin access (for some tests)
- Isolated environment recommended

### Running
```bash
# Basic chaos tests
go test -v ./test/chaos/...

# Docker restart test (requires privileges)
CHAOS_DOCKER_TEST=1 go test -v -run TestDockerDaemonRestart ./test/chaos/...

# Disk full test (DANGEROUS - fills disk)
CHAOS_DISK_TEST=1 go test -v -run TestDiskFull ./test/chaos/...
```

### Environment Variables
- `OUBLIETTE_SERVER_URL` - Server URL
- `OUBLIETTE_AUTH_TOKEN` - Auth token
- `OUBLIETTE_PROJECTS_DIR` - Projects directory path
- `OUBLIETTE_DATA_DIR` - Data directory path
- `CHAOS_DOCKER_TEST=1` - Enable Docker restart test
- `CHAOS_DISK_TEST=1` - Enable disk full test

## Test Results

Test results should show:
- 99%+ success rate for health endpoints under load
- Graceful error handling for corrupt/missing data
- Recovery after infrastructure failures
- No data loss during concurrent operations

## Known Limitations

1. Session tests require active project with working container
2. Docker tests require elevated privileges
3. Disk full tests may require manual cleanup
