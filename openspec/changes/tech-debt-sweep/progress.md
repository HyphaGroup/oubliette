# Progress: Tech Debt Sweep

## Status: Not Started

## Discovery Summary

Codebase scan on 2026-02-11 found:

| Category | Items | Est. Lines Removed |
|----------|-------|-------------------|
| Dead files | 4 files + 1 directory | ~170 |
| Legacy ScopeReadOnly | 18 refs across 8 files | ~30 |
| Legacy tool Scope | 25 refs across 5 files | ~40 |
| Dead .factory/ code | 4 functions + struct | ~80 |
| Dead session mode system | type + 6 fields + function + paths | ~100 |
| Unused audit ops | 8 constants | ~10 |
| Unused metrics | 4 functions + 4 vars | ~30 |
| Unused backup exports | 2 functions | ~40 |
| os.IsNotExist anti-pattern | 24 locations, 9 files | ~0 (replacements) |
| Stale comments | 5 locations | ~10 |
| **Total** | | **~510** |
