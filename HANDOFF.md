# Handoff

## Baton
Review — 2026-10-29, `session/fix-fresh`. F1 v2 built here; no push authorized.

## Where it stands
- Incomplete seasons are skipped, not entire players. No usable production after an
  incomplete season means unavailable with a stored reason. Explicit zero points with no
  games is an unused empty season; a rookie keeps his prior, labelled reduced.
- Snap-only seasons remain outside scales. The points-based kicker rule uses the latest
  usable past season or current pace; no production means unavailable, not invented zero.
- Input status survives model-score storage, Market player/pick cells, pick calibration,
  team rows/reports and draft numeric reads. Unavailable entries never enter value sums.
- F2–F4 remain in the preceding branch commits, unchanged by this task.

## Evidence
- Requested held replay: 2,258 valued vs 2,278; 1,891 movers; 20 unavailable (8 rostered,
  12 free agents), each with a reason. Richardson, Ty Simpson, Bradley Chubb and Ty Okada
  are valued and reduced. 107 reduced values; zero zero-valued outputs.
- Go filters listed first: affected nine packages 369 top-level PASS, 4 opt-in skips;
  composition slice 14 PASS; final runs `-race -count=1`. Replay also ran under `-race`.
- Asset Vitest 8 PASS; `pnpm build`, `go build ./...`, `make lint` PASS. Bindings
  regenerated without a diff. UI detector: no findings. No full verify or GUI claim.
- Evidence: `.tmp/f1-v2/`; replay: `.tmp/value-board/`. Production physical lines:
  12 affected handwritten files, 3,083 → 3,342 against `bd2a5083` (tests/docs separate).

## Next move / blocked on
- Independent review/mutant and disposable-profile click gate. Player and pick cells
  must say reduced inputs with reasons; all-unavailable players must say Not valued;
  team reports must label partial totals and preserve reasons in cell titles.
- Old arrays remain readable, but older binaries cannot decode new status-bearing
  model-input JSON. Review downgrade compatibility before deployment.
- Initial `git pull --ff-only` refused divergent upstream; no merge/rebase attempted.
- Untracked `.lintcache/` is generated lint cache; cleanup was tool-blocked, not committed.

## Tried and rejected
- Rejecting a player for one unusable season: discards other valid production.
- Treating zero points/no games as broken production: incorrectly removes Ty Simpson's prior.
- Missing-value zeros in sums/pick calibration without a label: hides unavailable evidence.
