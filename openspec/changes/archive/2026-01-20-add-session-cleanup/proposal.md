# Proposal: Add Session Cleanup

## Summary

Add functionality to clean up old session metadata files from disk, preventing unbounded growth of the sessions directory.

## Motivation

Session metadata files (`.json`) persist in `projects/<id>/sessions/` after sessions end. These files accumulate over time:
- A bug caused sessions to be marked completed on every turn, creating ~100 session files in a day
- Even with the fix, long-running projects will accumulate session files
- No mechanism exists to clean up old sessions

Current state of example project:
```
$ ls projects/8be6ef85-.../sessions/ | wc -l
97
```

## Proposed Solution

1. Add `CleanupOldSessions(projectID string, maxAge time.Duration)` to session manager
2. Expose as `session_cleanup` MCP tool for manual cleanup
3. Optionally add automatic cleanup on server startup or periodic background task

## Scope

- Session manager: Add cleanup method
- MCP tools: Add `session_cleanup` tool
- No changes to active session management (already has idle cleanup)

## Alternatives Considered

1. **Automatic cleanup only** - Less control, might delete sessions users want to keep
2. **Keep all sessions forever** - Current behavior, leads to disk bloat
3. **TTL per session file** - More complex, overkill for this use case

## Decision

Manual cleanup via MCP tool, with optional auto-cleanup on startup for sessions older than a configurable threshold.
