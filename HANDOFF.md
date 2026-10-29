# Handoff

## Baton
Review — 2026-10-28, `session/pick-value`. No push authorized or attempted.

## Where it stands
- Existing pick-value work through `cf5ae56d` is preserved. Native power composition still
  selects MFL; see `docs/architecture/pick-value.md` for the integration limits and evidence.
- U8 restores roster count on both Power Rankings reports: Whole roster, Starting lineup
  (default), Top N. N comes from the league starter count, as before; no numeric input existed.
- `TargetRace(week, team, mine, aggMode)` selects detached held boards; empty mode defaults
  to lineup in both views. Refresh calculates all six view/mode boards with one schedule read.
  Pick-setting changes rebuild every franchise mode without changing season/player values.
- Wails bindings and command argument/read schemas regenerated. Flow documented in
  `docs/architecture/system-narrative.md`.

## Evidence
- Targeted Go `-race`: 68 unique top-level tests PASS across Race, Season, Scoring,
  command catalog, binding generation and the composition slice. Final affected scoring /
  composition batch also PASS. Logs: `.tmp/u8/`.
- Calculation fixture proves sum 200 / lineup 110 / top-N 190 in BOTH views, and reads do not
  fetch. Forcing the composition edge back to default mode fails the regression.
- Targeted Vitest: 51 PASS. One full run: 1,580 PASS / 1 formatting failure; fixed five long
  lines, then 42 focused tests PASS including the formatting gate and Race interaction.
  No second full run. Typecheck PASS after the fix. `pnpm build` PASS before formatting fix.
- Mechanical UI detector: no findings. No GUI/live acceptance or aggregate verify claimed.

## Next move / blocked on
- Review and native GUI click-through on a disposable profile: open each Power Rankings
  report, confirm Starting lineup is selected, change each mode and check roster values /
  counting explanation update without a refresh. Other report controls must remain unchanged.
- CI/full verification remains unrun here by instruction. Existing native integration debt
  remains; initial untracked `.tmp-env.sh` and prior `.tmp/pickcalc` harness are untouched.

## Tried and rejected
- Re-fetching for a mode change: violates the held-read contract; refresh publishes all modes.
- Treating Top N as an arbitrary new number: the original used the league starter count.
- Inferring future pick slots from standings: existing pick-value ruling remains unchanged.
