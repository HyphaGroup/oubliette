# Consolidate Token CLI into Main Binary

## Problem

Token management is currently a separate binary (`oubliette-token`) requiring users to know about and use two different commands. This is inconsistent with the unified CLI design introduced in `add-install-and-mcp-setup` where `init`, `upgrade`, and `mcp` are subcommands of the main `oubliette` binary.

## Solution

Move token management into the main `oubliette` binary as `oubliette token <subcommand>`:

```bash
# Before
oubliette-token create --name "My Token" --scope admin
oubliette-token list
oubliette-token revoke <id>
oubliette-token info <id>

# After
oubliette token create --name "My Token" --scope admin
oubliette token list
oubliette token revoke <id>
oubliette token info <id>
```

## Changes

1. **Add `token` subcommand** to `cmd/server/main.go` with create/list/revoke/info actions
2. **Delete `cmd/token/`** directory (the separate binary)
3. **Update build.sh** to remove `oubliette-token` build
4. **Update documentation** (README.md, DEPLOYMENT.md, OPERATIONS.md)
5. **Update tests** to use new command path
6. **Update CI workflow** to remove separate token binary build

## Out of Scope

- Changing token functionality (scopes, storage, validation)
- Adding new token features
