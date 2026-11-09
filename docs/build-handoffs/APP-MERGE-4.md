# APP-MERGE-4

Local merge: `build/ROSTER-1` at `6dcf0224d` into `build/INTEGRATE-1102`
at `5e38e4e2e`. No remote operations or other-worktree changes.

## Conflict resolutions

- `HANDOFF.md`: replace competing batons with this integration's status; keep prior evidence by reference.
- `internal/leaguemigration/mflsource/settings.go`: initialize both the empty schedule-follow setting and enabled transaction-follow setting; neither replaces the other.

Automatic merge of `internal/composition/migration_app.go` preserves the inline
current-week refresh, single delayed played-week worker, running/started guards,
five-minute launch delay and ten-minute deadline. Transaction import runs within
that worker before played lineups. Transaction failure retains held history and
returns without importing lineups from potentially stale membership.

`internal/composition/live_stats_app.go`, frontend Pulse files and the cutover
rule fold remain identical to the integration parent. Cutover still clears
`schedule.follow` atomically; the integration's blessed schedule/scoring goldens
are not re-blessed.

## Semantic reconciliation

The roster line's timestamp filtering exposed a current-reading contradiction:
a rollover receipt deliberately advances just beyond prior receipts, so a slow
clock hid the recorded rollover from assets and roster checks. Current store
assets and `ReadNativeBase` use existing `ReadingInstant`; explicit historical
`NativeAssetsReading` and `NativeAssetsAsOf` retain their exact observation time.
The current/historical unit regression and rollover/reopen integration exercise
both contracts. A broader attempted clock adjustment inside the projector was
rejected by effective-time and historical-pick tests and reverted, not retained.

An absent `Transactions` rule group is omitted from JSON, preserving the frozen
pre-conferences rollover's original digest. Configured transaction-follow values
remain serialized. Existing historical-rollover validation exercises compatibility.
Old seeds remain unconfigured for transaction following; new Legacy seeds are off.
The seed-upgrade comparison now explicitly asserts that distinction.

Source/launch fake transports now answer the new read-only transactions request.
Catalogue and preset digest expectations account for the new setting without
changing competitive rules or weakening admission checks. Historical legality
fixtures pin their seed observation to their held October 13 data rather than
the wall-clock mirror replacement time. This preserves kickoff-lock and illegal
lineup assertions; the historical observation guard itself is not weakened.

## Generated contracts

- `internal/leaguerules/testdata/legacy.json`: `LEAGUERULES_WRITE_GOLDEN=1`; adds the disabled transaction-follow rule.
- `internal/commissioner/testdata/rollover-preview.wire.json`: `TWR_WRITE_WIRE=1`; adds the rule and resulting before-state digest.
- `internal/commissioner/testdata/rollover-entry.wire.json`: same writer; records the rule in the rollover before-state and resulting digests.
- `internal/commandcatalog/testdata/doors-before.json`: schema-only `TWR_WRITE_WIRE=1` writer added to the existing parity test; exactly five correction schemas change for roster status, historical pick/slot and provenance evidence. Inventory, permissions, modes and unrelated schemas remain frozen.
- `frontend/wailsjs/go/models.ts`: plain `make build`; historical picks/evidence/slots, nullable missing contracts, correction status/history and transaction rules. No binding hand edits.
- `testdata/wire/commissioner/controls.json`: `TWR_WRITE_WIRE=1`; exposes the new transaction-follow setting under CM-038.
- `testdata/wire/commissioner/rules.json`: same writer; adds the noncompetitive transaction-follow row.
- `frontend/src/app/commissioner/rules.wire.testdata.json`: same writer; adds the row to an empty league's reading.
- `frontend/src/app/commissioner/rules.legacy.wire.testdata.json`: same writer; adds the disabled row to the Legacy reading.
- `frontend/src/app/commissioner/replaced.wire.testdata.json`: same writer; retains the row after reverting another setting.
- `frontend/src/app/commissioner/jobs.wire.testdata.json`: same writer; one more unconfigured rule and its owning control.
- `internal/commissioner/presets_test.go`: digest literals refreshed from test-produced hashes for the added transaction rule/reading; existing empty-league creation hash remains unchanged through zero omission.

## Verification

`go build ./...` and plain `make build` pass on the final production code.
Discovery (`go test -list .`) preceded race batches, all with `-timeout 30m`.
Final counts (top-level + subtest PASS, not counting package PASS events):

| Package | Top-level | Subtests | Total PASS | SKIP |
| --- | ---: | ---: | ---: | ---: |
| internal/composition | 604 | 243 | 847 | 0 |
| internal/commissioner | 297 | 302 | 599 | 0 |
| internal/store/history | 275 | 190 | 465 | 4 |
| internal/leaguemigration | 6 | 27 | 33 | 0 |
| internal/leaguemigration/mflsource | 14 | 28 | 42 | 0 |
| internal/leaguerules | 40 | 73 | 113 | 0 |

`internal/leaguemigration/migrationtest` has no test files. All six tested
packages pass; no final failures. The four history skips are opt-in tests.
Other packages' final evidence is `final2-race.jsonl`; after correcting only
composition fixtures, full composition passes in `final3-composition.jsonl`.
Final shell Vitest: 47 files / 332 tests PASS. `pnpm exec tsc --noEmit -p .` PASS.
Logs are local ignored artifacts under `.tmp/app-merge-4/`.

## Debt and limits

No repository-wide suite, independent review, GUI click-through or combined
worker timing proof. First-kickoff rulings for 26 conflicting rows remain with
Christopher. Older-import transaction-follow activation and off-roster
missing-contract jobs remain prior roster-line debt. The transaction import is
still reached through schedule following; this merge does not redesign its
activation or increase the worker deadline. Live network/feed acceptance is not
claimed.
