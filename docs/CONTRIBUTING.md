# Contributing to Oubliette

Thank you for your interest in contributing to Oubliette!

## Development Setup

1. Fork and clone the repository
2. Copy config examples and configure:
   ```bash
   cp config/server.json.example config/server.json
   cp config/factory.json.example config/factory.json
   # Edit config/factory.json with your Factory API key
   ```
3. Install pre-commit hooks:
   ```bash
   pip install pre-commit
   pre-commit install
   ```
4. Build: `./build.sh`
5. Run tests: `go test ./... -race`

## Pre-commit Hooks

This repository uses pre-commit hooks to ensure code quality before commits:

- **golangci-lint** - Comprehensive Go linting (errcheck, govet, staticcheck, etc.)
- **gitleaks** - Secret scanning to prevent credential leaks
- **trailing-whitespace** - Removes trailing whitespace
- **check-added-large-files** - Prevents files >1MB from being committed

To run hooks manually:
```bash
pre-commit run --all-files
```

## Code Style

- Follow standard Go formatting (`gofmt`)
- Run `golangci-lint run` before committing (or rely on pre-commit hooks)
- Keep functions focused and under 50 lines when possible
- Add comments for exported functions and types

## Testing

- Run all tests: `go test ./... -race`
- Tests must pass with race detector enabled
- Add tests for new functionality
- Maintain or improve test coverage

## Pull Request Process

1. Create a feature branch from `main`
2. Make your changes with clear commit messages
3. Ensure all tests pass: `go test ./... -race`
4. Ensure code compiles: `go build ./...`
5. Run linters: `go vet ./...` and `staticcheck ./...`
6. Open a PR with a clear description of changes

## Commit Messages

Follow conventional commit format:
- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `test:` test additions or changes
- `refactor:` code restructuring
- `chore:` maintenance tasks

Example: `feat: add session index for O(1) lookups`

## Architecture Guidelines

- Keep handlers focused on a single responsibility
- Use dependency injection for testability
- Prefer explicit over implicit behavior
- Document architectural decisions in code comments or AGENTS.md

## Questions?

Open an issue for questions or discussions about potential contributions.
