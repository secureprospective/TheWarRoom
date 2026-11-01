# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/INTEGRATE-1101. APP-VALUE-5 partially fixed; local commits only, no remote operations.

## Where it stands
- Copied-profile desktop composition/workflow returns Bills (`franchise.01`) values: Best 21 = 243.262845927, rank 3/32, 35/35 roster players valued. No native read error reproduced, including on b7726819.
- Removed War Room's second HQ read. HQ/War Room publish independent regions while values load; late publications are scope-fenced. Cap runway held: 2026 $1.08M / 2027 $45.58M / 2028 $69.94M.
- One in-memory league valuation reused across headers/inspectors, invalidated by DB commits, input/param changes, clock rewind and future observation visibility. No engine/model math or persistent schema changes.
- Harness before: value cold/repeat 18,890/16,252 ms; cap 23/21, rules 8/8, assets 1/1. After: value cold/repeat 18,786/31 ms; cap 29/21, rules 9/9, assets 2/2; league-wide inspector 32 ms. Errors all nil. Timings include workflow/JSON; exclude database initialization.
- PASS: parity 25 players, max Now/dynasty difference 0; TypeScript; Vitest 159 files / 1,729 tests; targeted race Go 108 top-level / 295 named outcomes; plain make build. Hooks passed; two cache/revision mutants rejected.
- Affected handwritten production: 2,025 → 2,215 physical lines; tests: 1,401 → 1,650 (against b7726819). No reduction claimed.
- Ignored evidence in `.tmp/app-value5/`; throwaway harness removed after measurement. No desktop launch, live profile access, network fetch or remote operation.

## Next move / blocked on
- Cold `value.read` still FAILS the 200 ms budget. Cache hits are not cold-read proof. Any history commit or another league can cause another expensive pass. Review refresh-owned preparation/materialization and its freshness contract before choosing durable storage; start at `internal/appvalue/source.go` and `internal/store/playerrecord/store.go:Book`.
- Real desktop visual confirmation, league-specific age arc and real-profile lineup-fit availability remain unverified. Server-only player-record/params plumbing, copy-scoped valuation and free-agent fit remain inherited debt.
- Growth finding D10 recorded in `docs/data-layer/Database_Growth_Catalog.md`. Do not accept this as full APP-VALUE-5 completion.

## Tried and rejected
- Unwired desktop host hypothesis: actual `TargetWorkflow` and directly bound reads both return values.
- Re-reading all HQ data inside War Room delayed/discarded the already-loaded header; reuse the scoped reading.
- No fabricated zeros, model-math changes, league-origin branches, stale TTL cache or hidden warm-up counted as a cold read.
