# Handoff

## Baton
Bee → ClaudeBox / review — 2026-11-01, build/ROSTER-1. Local commits only.

## Where it stands
- APP-ROSTER-4 checks lineup active membership at the week's first held kickoff,
  once reached, using the effective-time projection. Before kickoff or without a
  held schedule it uses today's roster; player-specific lineup locks are unchanged.
- Eight table cases pass, including cuts, additions, inclusive boundaries, earliest
  kickoff and schedule fallback. Targeted commissioner race: 574 PASS, 1 inherited
  FAIL, 0 SKIP. `internal/commissioner/draft_startup_test.go:315` still expects
  90ca21fd230f0aa6 and gets 8a6936145d37309e, reproduced before this work as recorded
  in `docs/build-handoffs/APP-ROSTER-3-evidence.md`. No race diagnostic.
- Plain `make build` PASS with existing Wails binding warnings; binary not launched.
  Logs: `.tmp/roster4/`. No GUI test or independent review.

## Next move / blocked on
- Review lineup-at-lock; test a disposable historical week with a cut and an add
  after its first kickoff: the cut starter should be accepted, the added starter
  refused. Pre-observation kickoff remains refused, not guessed.
- Investigate inherited startup digest separately. Pre-seed reconstruction,
  unverified imported contracts, transaction translation/idempotency, 32-team
  replay equality and the prior 76 lineup refusals remain open or unmeasured.

## Tried and rejected
- Current roster for historical lineups loses valid cut players and admits late
  additions. No importer, contracts, player-lock policy or bindings changed.
- `git pull --ff-only` has no upstream; no push attempted. Pre-existing untracked
  `.golangci-cache/` was left untouched.
