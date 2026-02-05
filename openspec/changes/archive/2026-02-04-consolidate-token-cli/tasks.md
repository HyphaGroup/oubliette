# Tasks: Consolidate Token CLI

**COMPLETED** - Commit `5aade04`

## Phase 1: Move Token Code

- [x] Add `token` subcommand to `cmd/server/main.go`
  - Route to `cmdToken(args)` function
  - Implement create/list/revoke/info subcommands
  - Reuse logic from `cmd/token/main.go`
- [x] Delete `cmd/token/` directory

## Phase 2: Update Build

- [x] Update `build.sh` to remove `oubliette-token` build line
- [x] Update `.github/workflows/ci.yml` to remove token binary build step
- [x] Update `Dockerfile.server` to remove token binary

## Phase 3: Update Documentation

- [x] Update `README.md` token examples
- [x] Update `docs/DEPLOYMENT.md` token references
- [x] Update `docs/OPERATIONS.md` token commands
- [x] Update `docs/INSTALLATION.md` if token is mentioned

## Phase 4: Update Tests

- [x] Update `test/pkg/suites/cli.go` to use `oubliette token` instead of `oubliette-token`
- [x] Update `test/pkg/coverage/analyzer.go` if it references token binary
