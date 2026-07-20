# Handoff 42 — WarRoom Review-Week Harvest OUTCOME (executes handoff 41)

**From:** the review-harvest session, 2026-07-20, branch `session/review-harvest` (off main `ee891ef`).
**Full record:** `docs/reviews/warroom-review-week-2026-07/REVIEW_HARVEST_OUTCOME.md` — read it, it
has every lead's disposition. This handoff is the pickup for whoever merges / continues.

## What landed
- **GOAL 2 (PRIMARY, DONE on the Beelink):** arch-map skill v2 (deterministic classifier +
  real-depguard authority + coverage gate; Ornith reframed onto interpretation+uncertainty),
  measured 78→21 flagged / 0 build-violations on the identical run-1 input; ornith-lessons.md +
  glm-review-lessons.md written where the runs read them + wired into the review preamble; Hermes
  warroom-review skill reframed to leads-not-findings + lessons feed. NONE of this is in THIS repo —
  it's on the Beelink (`~/ai-workspace/skills/arch-map/`, `~/opencode/review_logs/warroom/`) and
  Hermes VM 502. If the Beelink skill dir is git-tracked, commit it there (independent backbone).
- **GOAL 1 TWR-2:** FALSE POSITIVE vs the documented Ship-3 money model (`TestIntegration_TagOffGridPriceSnapsInCap`
  enshrines "cell is KING, no rounding at rest; cap snaps at aggregation"). Fix drafted then REVERTED.
  A design-consistency flag (tag vs other write paths) is recorded for Christopher — not a bug.
- **GOAL 1 TWR-1:** FIXED in `app.go` — `context.WithTimeout(ctx, storeFloorTimeout=120s)` around
  `initStoreFloor` so a hung MFL can't black-screen OnStartup. Gate: build clean / lint 0 / race green.

## The ONLY blocker before merging TWR-1 to main
**Live Beelink FAILURE-PATH functional gate:** launch the app with MFL unreachable (e.g. block the
host / bad league id) and confirm a BOUNDED, surfaced startup error (via Ping / stderr) instead of a
black window — and confirm a NORMAL launch still comes up (init < 120s). Only after that → squash-merge
`session/review-harvest` to main. It's a startup-path change; do not merge on automated gates alone.

## Deferred leads (real, batched — NOT this session; see the outcome doc's table)
Go toolchain 1.26.5 + x/sys bump (safe, do together); vite multi-major jump (its OWN session, confirm
first); rulebook.Initialize tx-wrap; a log.Printf→slog sweep; app.go:213 Close-error; 3 frontend
try/catch (confirm TransactionsPanel isn't superseded by M4 first; ping.ts is dead code). Arch A4
(engine→store-via-interfaces) is a refactor lead, not a bug.

## Next
Either run the TWR-1 failure-path gate → merge, OR proceed to the Versioning & Releases phase
(handoff 40) which was the actual next-named thing. This harvest was a collaboration/training session,
not a build row.
