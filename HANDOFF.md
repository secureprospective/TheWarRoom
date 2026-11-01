# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-1; local commits only, no remote contact.

## Where it stands
- New imports read all MFL schedule weeks and map IDs in the importer. Explicit fixtures use ordinary schedule.generate / schedule.publish and shared ledger reads; no source-dependent engine path.
- Existing leagues: `go run ./cmd/twr schedule-backfill --help`. Explicit destination, commissioner identity and reason; recorded franchise mapping; atomic publication; unchanged re-sync/re-import writes nothing. Instructions: docs/architecture/schedule-import.md.
- SQLite-backup proof under .tmp/sched/profile: 17 weeks, two appends, retry zero. Bills week 8 away at Patriots. Native matchup confirms it. Starting assets preserved; real supplied profile and live config untouched.
- Comprehensive targeted-package race run: 542 top-level / 1,108 named PASS, four skips, one inherited startup-draft digest failure. Baseline schedule/owner source overlay reproduces identical failure. Final affected-behavior batch: 25 top-level / 65 named PASS. tsc, plain make build and targeted golangci-lint pass. Commit hooks not bypassed.
- Affected handwritten Go file set vs 330f33b1: production 2,070 → 2,418 physical lines; tests 689 → 963. No reduction claimed.

## Next move / blocked on
- Played standings are NOT available: prior import has no weekly lineups/results; live MFL schedule scores blank. Native standings returns 0 decided / 208 pending. Displayed first three Bills, Patriots, Jets all 0–0, not historical rankings.
- week.read contains real week-8 opponent but Current remains unset: only five distinct NFL week-date exports held; existing selector requires every scheduled week's dates. No dates guessed or results zeroed. No GUI test performed.
- make lint stops at inherited bloat tiny-files 26 vs baseline 25 (no increase retained). Standalone boundary fails untouched (*App).startValuePreparation fingerprint. Neither gate weakened.
- Profile proof/logs remain in ignored .tmp/sched/. Untracked .golangci-cache/ is the brief-required lint cache.

## Tried and rejected
- Separate fixture helper file increased tiny-file count; folded into the existing schedule responsibility instead.
- Do not turn source schedule score strings into fabricated native lineups or zero-point results. Scoring engines unchanged.
