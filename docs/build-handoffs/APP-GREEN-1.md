# APP-GREEN-1 — known failing tests made green

Base `build/INTEGRATE-1102` 2a2bc5bb → branch `build/GREEN-1`.

## Per failure (cause commit → test or code → fix)
| Failure | Cause | Wrong | Fix |
|---|---|---|---|
| TestNativeResultsStandingsWire, TestLedgerAuditAdoptsExistingHistoryWithoutChangingReceipts, TestNativeScoreWireFixtureMatchesGo, TestNativeLeagueWireMatchesGolden (score part) | — | — | Already pass on base 2a2bc5bb (fixed by an earlier merge); nothing to do |
| TestFrozenDoorMetadata | SET-2 f8d769d0 (`schedule.*` args gain `recorded`), STALL-9 1b4e8187 (`stats.queue.read` gains `request`; TargetCancelRead put in the fixture as `desktop`, but the code says `both`) | test | regenerated the 4 schema hashes (×2 doors); TargetCancelRead availability set to `both` |
| TestCommissionerWireMatchesGolden (rules, controls), TestCommissionerReplacedRulesWire, TestNativeLeagueWireMatchesGolden (lineup.json) | SET-1 8978f859 | test | goldens regenerated; diff is ONLY the new `league.startingAssets` and `lineup.lock` rows/labels |
| TestCommissionerPresets | SET-1 | test | an empty league now has Set `league.startingAssets`=empty and `lineup.lock`=each player's game (SET-1 handoff: native creation selects these); asserted exactly |
| check-frontend-transport (3) | 9d735caa roomReads, 5110ff1d/bc95f10e localWorkflow.test | code | roomReads takes `ShellStatus`/`ParamsResult` from `control/api.ts` (new `ShellStatus` alias); the test's `desktop()` returns its fake commissioner API instead of reading `window.go.commissioner` |

## Failures that were NOT in the brief (they fail on base too; found by bisect)
| Failure | Cause | Wrong | Fix |
|---|---|---|---|
| TestLegacyCommandVersionsReplayAndRetry | SET-1: `Prepared()` appends starting-setting defaults, so retrying a pre-SET-1 Create gets refused as a different change | **code** | `MatchesRetry` accepts a Create retry whose seed is exactly the stored seed plus the same `startingDefaults` the ledger already reads it with (`seedChanges`) |
| TestLaunchAutomaticallyImportsPlayedLineups | SET-2: the recorded-results backfill error reused `lineupErr`, so "import week not known" blocked the played-lineups import | **code** | separate `resultsErr` in `ScheduleSync.syncPlayedLineups`; when recorded-results capture happens is unchanged |
| TestNativeGovernedCatalogueHasNoOrphanedFields | SET-1 cited `lineup.lock` in the lineup reading but not in the lineup check | **code** | `native_lineup.go` cites `lineup.lock`; new round-trip proof `TestLineupLockIsTheLeaguesChoice`; lineup-check goldens (+1 rule line only) |
| TestLockedPlayerCannotBeMovedInOrOutAfterKickoff | SET-1: the fixture is an import, so it locks at first kickoff; the refusal wording was generalised | test | the test sets `lineup.lock=each_player_game` explicitly (it tests per-game timing) and expects the new wording; no assertion removed |
| TestNativeLineupSetThroughTheGateReadingAndRestart, TestNativeDenialsMatchGolden | SET-1 wording | test | new wording |
| TestEmptySeedAdmissionBySource/legacy_nfl | SET-1 de-origin wording | test | new wording |
| TestCutoverIsOnceOnlyAndPermanent | SET-1 origin gate → starting-assets gate (still refused) | test | new wording |

No scoring or valuation results changed. No assertion was deleted or skipped.

## Verify (`-race -p 2`)
composition 518 PASS / 0 FAIL; store/history 525 PASS / 4 SKIP (pre-existing); commissioner 636 PASS / 1 SKIP;
commandcatalog 44 PASS; app/mflref/migration 9 PASS. Frontend: transport check holds, tsc clean, vitest
(shell/localWorkflow, control, commissioner) 572 PASS. The full suite was not run (Claude runs it after merge).

## Debt
- The empty preset is still labelled "every rule Needs a ruling", but two settings are now Set. That copy is stale.
- The brief's list of known failures was out of date: 5 had already been fixed, and 9 unlisted SET-1/SET-2 failures existed. SET-1 and SET-2 merged without these packages green.
