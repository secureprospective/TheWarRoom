# APP-SET-1

## Baton
Sol → integrator, 2026-11-02, `build/SET-1`, based on `96ebc875`. Local only.
**Settings implemented; historical-score ruling remains blocked.**

## Settings and flow
- `lineup.lock`: “Each player’s game” / “Week’s first kickoff”. Default32 had no
  existing lock setting, so new native leagues use each player's game. MFL
  translation supplies first kickoff. Admission and lineup player readings use
  the resolved setting; commissioner lineup corrections still bypass locks.
- `league.startingAssets`: “Empty rosters and rules-defined auction lots” /
  “Held rosters, contracts and draft picks”. Native creation selects empty; MFL
  import selects held. O-Line lots still come from the league's O-Line rules.
- Catalogue → validated ordinary `RulesSet` ledger entry → `RulesInForce` →
  admission/readings. Existing commissioner settings UI and command registry expose
  both. Census ownership is CM-004 / CM-001 respectively. No new IPC door or count.
- Compatibility fills only absent settings on old creation records. Explicit
  settings win over origin, including after cutover. An otherwise empty rule
  stream does not trigger legacy-preset upgrades. Invalid values fail validation.

## Origin branches removed
1. Starting snapshot admission (`internal/commissioner/native_seed.go`).
2. Empty-start admission (`internal/commissioner/native_start.go`).
3. Held capture versus empty start (`internal/app/commissioner/native_seed_api.go`).
4. Seed job wording (`internal/commissioner/jobs_league.go`).
5. Cutover admission (`internal/commissioner/migration.go`).
6. Old import schedule backfill (`internal/leaguemigration/settings.go`).
Also removed the source-derived ban on O-Line cards in held starting assets.
Source/preset decoding remains only as historical metadata/default compatibility,
not operational seed/cutover decisions. No source branch remains in those paths.

## Evidence
- Targeted Go `-race -p 2`, test discovery first: final counts below.
- Broader touched-package batch: 2,797 PASS, eight failures found and corrected;
  one optional disposable-profile test skipped. No full suite.
- Four native preset creation/reading digest oracles remain unchanged after
  excluding the two intentional new setting surfaces (and the previously blank
  internal ID of a completely empty rule stream). Lineup cached/uncached readings
  are byte-identical with the selected policy. Eleven imported fixture leagues
  round-trip without source-field loss under the existing comparison.
- Real composition/API test changes both settings twice and checks ordinary
  ledger receipts, current values, citations and revision changes.
- Vitest, desktop build, lint and commit hooks: final results below. Build only;
  no app launch, live profile access, MFL writes, or remote operations.

## Blocked: historical recorded results
A disposable regression probe supplied an MFL week with scores `105.25` and
`99.00` to `mflsource.Schedule`. It FAILED: the output retained only game ID,
home and away. `ScheduleGame` has no recorded score; the import plan retains no
team-result snapshot. `NativeResultsReading` always calls `franchiseWeekScore`
without a cutover boundary. Thus the requested preservation test cannot pass.
The failing probe was not committed; no scoring behavior was changed because this
ruling is explicitly test-only. Next slice must retain source results and define
which week crosses cutover, then add a passing import/result regression test.
This is an existing functional gap, not verified preservation.

## Tried and rejected
- Updating historical scores or introducing a new result store here: exceeds the
  test-only ruling and the wire/expose-only scope.
- Refreshing preset digest constants: instead preserved the existing byte-level
  oracles, excluding only the new settings.
- Applying legacy upgrades to empty leagues merely because defaults now exist:
  caught by the old digest oracle and corrected.

## Final checks
- Go acceptance batch: **56 PASS** (19 top-level tests), including all eight
  corrected regressions. Fixture migration: **12 PASS** (11 fixture cases plus
  parent). Command registry: **44 PASS**. Composition settings/wire checks:
  **3 PASS**. Total **115 PASS** counting subtests and their parents. A final reference
  backfill edge check adds **35 PASS**, including a native held-assets league
  with no MFL reference and the real backfill/cutover composition path.
- Vitest: **23 PASS**, three files (rules, rule-change form, rule draft).
- `CI=true make build`: **PASS**; regenerated Wails models, no new method/binding.
  Wails emits existing missing-type diagnostics but completes frontend and binary.
- Targeted golangci-lint: **0 issues**; final commit runs the configured hooks.
- Scope measurement against base: affected handwritten production Go files,
  **3,485 → 3,622 physical lines**, **126,327 → 131,300 Unicode code points**
  including LF. Tests, fixtures, docs and generated models counted separately.
- Optional disposable-profile lineup-equivalence test remains unexercised; the
  normal in-repo byte-equivalence test passes. No desktop walk requested.

Native held starting assets without an MFL identity are a no-op for historical
reference backfill, not an error and never an invented reference. Commit hooks
passed for the settings implementation; the follow-up commit runs them again.
