# APP-MERGE-7 — done

Bee, 2026-11-02. Branch `build/INTEGRATE-1102`; started at `3e8fe2729`.
Local only: no remote contacted, no push, no full repository Go suite.

## 1. Boundary regression — DONE

Commit `152274c88` — separate held roster evidence from acquisition.

The history import was a regression introduced by SCHED, not inherited architecture debt:
`git log -1 b51768473` names the Nov 1 unrostered-zero change;
`git log -1 981564993` names the Nov 1 dated-membership bye change.
Removing contracts alone exposed the second route through researchfeeds, then nflschedule.
All production history imports of ingestion packages are now gone.

- Historical depth rows/validation and decoding now belong to the existing acquisition-free
  `internal/fantasyscore/source` package. History consumes its plain `RawDepth` values;
  the decoder retains all dated snapshots, unlike research's latest-snapshot projection.
- Bye values/validation/held decoding and team-code normalization live beside those facts.
  The existing ingestion APIs delegate and retain type aliases, avoiding duplicate business rules.
- New `internal/feedwire` holds the existing singleton/array envelope decoder and error-payload
  rejection, without HTTP/MFL dependencies. Indexed in placement, system map and narrative.
- Historical selection, completeness, no-roster, no-snap and bye-zero admission are unchanged.
  A new decoder regression proves that older snapshots survive and an invalid older row or
  malformed Parquet is rejected. Existing schedule/absence tests exercise the moved helpers.
- No boundary rules, fingerprints, fences, exemptions or guards were relaxed.

Final `go run ./tools/boundarycheck`: exit **0**, **170** frozen functions/sources.

## 2. Home controls and long lines — DONE

Commit `5110ff1d3` — route Home actions through shared controls.

HomeBrief and homeTiles use shared `Act` controls and named surface-command handling.
Tile selection is a sibling button, not a clickable article containing nested controls.
Ledger rows are real buttons; navigation remains `nav.open`. Customize, pin, resize,
overflow/restore, drawers, transaction cards, tabs and news filters retain their existing actions.
Escape and shortcut handling remain document listeners with cleanup. The locked CSS remains;
small scoped rules preserve full-width alert/ledger layout and route pointer input to sibling controls.
Tests observe navigation command receipts rather than spying on the pre-conversion callback.
A new regression observes tile selection, shared-button semantics, absence of nested buttons,
alert opening and Escape closing. JSX spacing is explicitly retained where wrapping required it.

All **86** original long lines across **10** shell files were wrapped; strict controls/length
checks are unchanged. Files: HomeBrief, homeTiles, homeBrief.test, HQPanels, PulseNow,
homeReads, leaguePicker.interaction.test, localWorkflow.test, pulseLive.test and pulseReads.test.

Local-main ancestry was checked rather than assumed:
- Only PulseNow exists on local `main` (`b4b341804`).
- `git log main -- frontend/src/app/shell/PulseNow.tsx` identifies `b6fb23600`
  (2026-10-07). That main version has no lines over 110 characters.
- Its current replacement is `dbf6f576c` (2026-11-01); all other listed files were added
  after local main. No file was skipped based on ancestry. No remote main was fetched.

Final `pnpm vitest run src/app`: **180 files, 1,832 tests PASS**, zero failures, 12.37s.
`pnpm tsc --noEmit`: exit **0**. Mechanical UI detector: `[]`; not browser acceptance proof.

## 3. Final-tree verification — DONE

Code tree: `5110ff1d3`; subsequent changes are reports only.
Environment: `GOTOOLCHAIN=go1.26.10`, worktree-local `GOLANGCI_LINT_CACHE`, specified Go/pnpm
PATH; Go checks/build used `GOMEMLIMIT=1500MiB`, `GOMAXPROCS=8`.
Capacity checks preceded heavy jobs; jobs were serialized. Commits used `git commit -F` with hooks.
Go commit: golangci-lint, gitleaks, ifaceguard PASS. Frontend commit: gitleaks PASS,
Go-only hooks correctly skipped. An initial hook refusal for unwrapped delegation errors was fixed.

Discovery first: `go test -list .` across the packages below, **1,006** test declarations.
Then `go test -race -p 1 -timeout 30m -json` on the same explicit package list:

| Package under internal/ | PASS tests/subtests | SKIP | Package elapsed |
|---|---:|---:|---:|
| store/history | 465 | 4 | 249.688s |
| fantasyscore | 218 | 0 | 1.473s |
| fantasyscore/source | 130 | 4 | 30.004s |
| feedwire | no test files | — | — |
| ingestion | 39 | 0 | 1.163s |
| ingestion/researchfeeds | 7 | 0 | 1.736s |
| ingestion/nflschedule | 13 | 0 | 1.046s |
| ingestion/livegame | 24 | 0 | 0.001s |
| composition | 846 | 0 | 898.214s |
| **Total** | **1,742** | **8** | |

The eight skips are existing opt-in cache/fixture/measurement cases:
SleeperUnknownIDReport, FullRealSeasonLoadsAsWeekParts, KeptBigLeagueIdentityMigration,
ReplayCost, E5Diagnostic, StudyEdgeConversions, RTLStudy, StudyWeeksUnderTheLegacyUpgrade.
No required zero regression skipped: dated membership, complete no-roster coverage,
unrostered zero without a game/event rejection, historical-team schedule byes,
legacy-ID rejection, no-snap proof coverage and schedule/event contradiction tests all PASS.

Plain `make build`: exit **0**, packaged `build/bin/thewarroom`, 24.718s.
Wails still reports missing snapshot model aliases while generating bindings, but exits successfully;
no tracked generated files changed. The production binary was not launched.
`git diff --check`: exit **0**.

Raw command logs and ancestry evidence remain worktree-local in `.task-evidence/` (untracked).
The pre-existing `.golangci-cache/` remains untracked.

## Debt left

- No blocker remains in this brief's requested gates.
- Independent review/mutant check and disposable-profile visual/keyboard acceptance remain unverified.
  Test Home tile selection/navigation, drag/resize/pin, overflow restore, news filters, alert/ledger
  drawers and Escape; passing means the same locked appearance and actions, with keyboard-operable
  shared controls and no nested buttons. The DOM tests/build do not prove pixel parity or drag behavior.
- Carry forward APP-MERGE-4's unresolved operator rulings and transaction-follow activation debt;
  APP-MERGE-6's delayed-worker live shutdown/publication acceptance still needs its live check.
- Full repository verification is reserved for the operator, as requested.
