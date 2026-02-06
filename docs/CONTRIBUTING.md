# Contributing

## Setup

```bash
git clone https://github.com/HyphaGroup/oubliette.git && cd oubliette
cp config/oubliette.jsonc.example config/oubliette.jsonc
# Edit with your API keys
./build.sh
go test ./... -short
```

## Code Style

- `gofmt` for formatting
- `golangci-lint run --enable gocritic` before committing
- Functions under 50 lines when possible

## Testing

```bash
go test ./... -short                        # Unit tests
cd test/cmd && go run . --test              # Integration tests
cd test/cmd && go run . --coverage-report   # Must be 100%
```

## Commit Messages

```
feat: add session index for O(1) lookups
fix: handle nil executor on resume
docs: update architecture for streaming changes
```

## Pull Requests

1. Branch from `main`
2. Ensure tests pass and lints clean
3. Clear description of changes
