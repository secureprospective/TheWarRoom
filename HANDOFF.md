# Handoff

## Baton
Bee — 2026-10-23. Save profiling follow-up complete on `session/t487-close-books`; PR #113 remains unmerged.

## Where it stands
- D9 multi-season 10k profile: current-season rule validation dominates; `CheckRulesAppend` is 3.42/4.58 sampled CPU seconds on the save stack. Whole-profile GC is substantial (includes fixture setup).
- `internal/leaguerules/resolve.go` now constructs field descriptors once per resolution, not twice per change. Same checks, no shared cache.
- Differential complete-result/error tests in `internal/leaguerules/resolve_equivalence_test.go`; affected-package race tests and `go test ./...` pass.
- Three-run medians and method in `docs/data-layer/Database_Growth_Catalog.md`: 10k D9 0.627 → 0.507 s; D1 0.444 → 0.424 s. 1k essentially unchanged.
- Local profile, raw measurements and test logs: `.scratch/t487-profile/` (ignored).

## Next move
Review PR #113; further current-season rule decoding/allocation reduction if the strict 0.5 s budget is required. No GUI/live MFL testing performed.

## Blocked on
D9 still narrowly misses budget. No merge authorized.

## Tried and rejected
No broad replay cache or validation shortcuts: the measured catalogue allocation had a contained fix preserving all validation.
