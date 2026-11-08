# Client/server step 1b — contract map (2026-11-08)

Evidence, not a plan. Gate (`docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md:50`): *required locked-screen +
authorized CLI/MCP workflows exercise same service with identical outcomes/rejections.*
Parity test: `internal/serverhttp/contract_parity_test.go` (`TestContractParityScreenAndCLI`).

## The three doors as they exist today

| Door | Route | Reaches |
|---|---|---|
| **Wails, local** | `frontend/src/app/shell/localWorkflow.ts:47` `TargetWorkflow` → `internal/app/commissioner/commissioner_preview_api.go:67-80` | `catalog.RunWorkflow` → `leagueservice.Service` (local host). Ten reads bypass it: `localWorkflow.ts:20-44` call a direct Wails binding and send `native.assets.read` only for the authority label. Control forms call `TargetCommissionerPreview/Execute` (`commissioner/proxy.ts:37-38` → `commissioner_api.go:13`, `commissioner_preview_api.go:16`) → `Service.Preview/Execute`, not the catalog. |
| **Hosted (runNative desktop / browser)** | `serverclient/desktop_handler.go:151` → `serverhttp` `WorkflowPath` → `workflow.go:56,68` | `catalog.RunWorkflow` → `leagueservice.NewHosted` (`hosted.go:232`) with football **pinned** (`hosted.go:227,240-249`). Only `WorkflowCommands()` (`commandcatalog/workflow_commands.go:20-34,57-106`) are served. The hosted desktop has no general commissioner preview/execute action (`serverclient/desktop.go:79-95` action list). |
| **CLI / MCP** | `cmd/twr/main.go:191-200`, `cmd/twr/mcp.go:45-77` → `serverclient/agent.go` `Dispatch` | **Catalog commands** (`serverhttp.WorkflowCommand`): `Agent.workflow` reads `native.assets.read` for the pinned football digest, then sends the command on `WorkflowPath` → `token_policy.go` `agent` (football pinned via `workflowHost`) → `catalog.RunWorkflow`, the screens' path. **Other commissioner commands** still go `DispatchPath` → `Service.Preview/Execute` (debt, ruling 1 below). `DispatchPath` refuses catalog commands (400, "update twr"). Tokens are commissioner-only (`commissioner/token_scope.go:251`). |

## Per-screen map

Screen → reads file attribution is by importer (`HomeBrief.tsx`, `NativeFranchiseHQ.tsx`, `PulseNow.tsx`/`NativeLeaguePulse.tsx`,
`WarRoom*.tsx`, `control/NativeAccount.tsx` → `commissioner/HostedLeague.tsx`). Market is a War Room tab (`WarRoom.tsx:20,98`).
Wails column: **cat** = `TargetWorkflow` → catalog; **direct** = own binding (`localWorkflow.ts:<line>`); **cmd** = `TargetCommissionerExecute`.

| Command / read | Screens | Wails | Hosted serverhttp | CLI/MCP | Mark |
|---|---|---|---|---|---|
| `native.assets.read` | Home, HQ, War Room (Market, Trade), Control | cat | catalog | none | missing in CLI |
| `trades.read` | Home, War Room (Drawers, Plans, Trade) | cat | catalog | none | missing in CLI |
| `standings.read` | HQ, Pulse, War Room Trade, Control | cat | catalog | none | missing in CLI |
| `results.read` | HQ, Pulse, Control | cat | catalog | none | missing in CLI |
| `lineup.read` | HQ, Pulse, Control | cat | catalog | none | missing in CLI |
| `draft.read` | Home, War Room Draft, Control | cat | catalog | none | missing in CLI |
| `commish.rules.read` | HQ, War Room Market, Control | cat | catalog | none | missing in CLI |
| `bids.read` | Home, War Room Market, Control | cat | catalog | none | missing in CLI |
| `waivers.read` | War Room Market, Control | cat | catalog | none | missing in CLI |
| `value.read` | HQ, War Room Inspector | cat | catalog | none | missing in CLI |
| `native.reserve.check` | HQ, Control | cat | catalog | none | missing in CLI |
| `moves.read` | HQ | cat | catalog | none | missing in CLI |
| `odds.read` | Pulse | cat | catalog | none | missing in CLI |
| `commish.market.read` | Control | cat | catalog | none | missing in CLI |
| `native.cap.check` | Home, HQ, War Room Trade | direct `:21` | — | none | different path (direct binding `TargetNativeCapCheck`); missing on server; missing in CLI |
| `projections.read` | HQ, Pulse, War Room Inspector | direct `:43` | — | none | different path (`TargetNativeProjections`); missing on server; missing in CLI |
| `week.read` | HQ, Pulse | direct `:28` | — | none | different path (`TargetNativeWeek`); missing on server; missing in CLI |
| `lineupdelta.read` | War Room Inspector, War Room Trade | direct `:30` | — | none | different path (`TargetNativeLineupDelta`); missing on server; missing in CLI |
| `calendar.read` | War Room header, Home | direct `:25` | — | none | different path (`TargetLeagueCalendar`); missing on server; missing in CLI |
| `contract.quote` | HQ | direct `:23` | — | none | different path (`TargetNativeContractQuote`); missing on server; missing in CLI |
| `native.directory.read` | Pulse | direct `:32` | — | none | different path (`TargetNativeDirectory`); missing on server; missing in CLI |
| `bracket.read` | Pulse | direct `:41` | — | none | different path (`TargetNativeBracket`); missing on server; missing in CLI |
| `commish.jobs.read` | Home | direct `:39` | — | none | different path (`TargetCommissionerJobs`); missing on server; missing in CLI |
| `commish.ledger.filtered.read` | Home | direct `:36` | — (unfiltered `LedgerPath` only) | unfiltered ledger `agent.go:49` | different path (`TargetCommissionerLedgerFiltered`); missing on server (filter); CLI has unfiltered only |
| `commish.lineup.set` | HQ (`hqNativeLineup.tsx:27`), Control (`HostedLeague.tsx:81`) | cat → `SetLineup` (`workflow_commands.go:133`) | catalog → `SetLineup` | catalog (`WorkflowPath`, pinned) | **same service** (test-proven: screen and CLI both record) |
| `roster.reserve` | HQ (`hqNativeLineup.tsx:36`), Control | cat | catalog, football pinned | catalog (`WorkflowPath`, pinned) | **same service** (test-proven) |
| `trade.offer` | War Room Trade (`warroomTradeReads.ts:80`) | cat | catalog | catalog, commissioner token only | **same service** (test-proven; owners have no token, ruling 2 of part 2) |
| `draft.pick` | Control (`HostedLeague.tsx:149`) | cat | catalog (`runWorkflowPick`) | catalog refuses: the pick entry takes the on-clock franchise's `DraftPickInput`, the CLI sends a `RawCommand` | refused by the catalog, state unchanged (test-proven; correct behaviour) |
| `draft.timer`, `draft.skip` | Control (`HostedLeague.tsx:81`) | cat | catalog | catalog (`WorkflowPath`, pinned) | **same service** (test-proven) |
| `commish.rules.set` | Control (`RuleChangeForm.tsx:61`) | cmd | none on hosted desktop (no action in `desktop.go:79-95`); `ExecutePath` exists | `Service.Execute` | missing on server (hosted screen route); CLI same service via generic dispatch |
| `commish.notice.publish`, `commish.whatsnew.post` | Control (`NoteForm.tsx:11,14`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| `commish.league.create` | Control (`StartLeagueForm.tsx:34`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| `commish.correction.revert` | Control (`UndoForm.tsx:19`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| Personal notes / targets, trade-builder plans | War Room Market/Inspector/Trade/Plans, HQ inspector (`offers/api`, `trades/api` importers) | own bindings | none | none | missing on server (personal plans move in step 5); missing in CLI |

## Counts (rows above; a row may carry two marks)

- same service via catalog on Wails + hosted: 14 reads, plus `commish.lineup.set`, `roster.reserve`, `trade.offer`, `draft.pick`, `draft.timer/skip` on those two doors. The CLI joins on the four mutation rows `commish.lineup.set`, `roster.reserve`, `trade.offer`, `draft.timer/skip`; `draft.pick` is refused by the catalog for the CLI by design.
- **different path:** 10 Wails direct-binding reads + `commish.ledger.filtered.read` (11). CLI: 0 catalog rows (was 4). The four non-catalog Control rows still reach `Service.Execute` through `DispatchPath` (debt below).
- **missing on server:** 11 direct-binding reads; 4 Control form rows (hosted desktop has no preview/execute action); personal plans (1) = 16.
- **missing in CLI:** every read except the unfiltered ledger = 25 rows (`native.assets.read` is used internally for the football digest but not offered as a command). `draft.pick` is no longer counted: the catalog refuses it for the CLI by design. Owners have no CLI at all (commissioner-only tokens).

## What the parity test proves (`TestContractParityScreenAndCLI`)

Same fixture league (`hostedfixture.CreateWorkflowLeague`), fresh per door, same steps, real TLS `serverhttp.Hosted` handler;
screen door = `WorkflowPath` with a human session, CLI door = `serverclient.Agent.Dispatch` with an agent token
(scope: the command, `commish.ledger.read`, `native.directory.read`, `native.assets.read`).

Rows: `roster.reserve`, `commish.lineup.set`, `trade.offer`, `draft.timer`, `draft.skip`. Every step asserts the same
allowed/changed/state for both doors; every refusal leaves the league unchanged.

| Step | Outcome (all five rows) |
|---|---|
| preview | both accept, no change |
| stale expected revision | both refuse, unchanged |
| unknown command | both refuse, unchanged |
| ineligible argument | both refuse, unchanged |
| wrong role (owner, franchise.02) | both refuse, unchanged (screen: catalog role gate; CLI: token issuance refused) |
| missing confirmation (CLI, G2/G3 only) | refused, unchanged |
| execute (confirmed) | **both record, same state** |

Also: `TestContractParityDraftPick` (CLI pick refused by the catalog, unchanged), `TestContractDispatchRefusesCatalogCommands`
(catalog command on `DispatchPath` → 400 whose body says to update twr), `TestContractCLIRefusalWording` (a missing read is
named; a refused preview reads as a preview refusal and is `ErrPreviewRefused`).

## Rulings 11-08

Step 1b, part 2 (Christopher):
1. YES: CLI/MCP commands go through the same catalog path as the screens (`catalog.RunWorkflow`, pinned football). Done for every catalog command.
2. NOT NOW: owner agent tokens. `TokenScopePermission` stays commissioner-only. Untouched.
3. The 11 direct-binding reads and the Control forms on the server move to step 2. Left as rows above.

Step 1b, part 3 (Christopher + head brain):
1. Non-catalog commissioner commands (`commish.rules.set`, `commish.notice.publish`/`commish.whatsnew.post`, `commish.league.create`, `commish.correction.revert`, …) keep `DispatchPath` for now. They join the catalog in step 2 with the hosted Control forms, and the bypass is deleted then.
2. Option A: a CLI catalog command needs its token to also cover `native.directory.read` and `native.assets.read`. Desktop-issued tokens already carry both (`platform.DelegableCommands`). If a token lacks one, the CLI names the missing read and says to ask the commissioner for a new token.
3. A refused catalog preview reads as a preview refusal (`serverclient.ErrPreviewRefused`), not "did not record this change". Execute refusals stay `ErrRefused`. Screens are unchanged.
4. An older `twr` that sends a catalog command to `DispatchPath` is not caught by the version gate: it declares the same protocol, so the server answers 400. The 400 body now says to update twr. Caveat: older clients do not show the body (`agent.go` `post` default branch prints "the league refused this request (400); this token may not cover it"). See Needs ruling in the part-3 report.
5. The failing goldens `hosted.wire.testdata.json` / `rollover.wire.testdata.json` were stale, not clock flakes and not a production bug. `b825060eb` (11-05) added the Market pool join (names, minimums, points) and `poolPointsNote` to `native.assets.read`, and the goldens were never regenerated. The fixture's evidence holds only season 2026, with every kickoff at now+24h, so the offseason fallback asks for 2025 (rollover: 2027) and records the note. That output is the same on any date. Regenerated with `TWR_UPDATE_HOSTED_WIRE=1`; the diff is the pool fields and the note, plus timestamps and ids that the comparison ignores.

## Debt

- `DispatchPath` → `Service.Preview/Execute` stays for the non-catalog commissioner commands (+50 production LOC in step 1b part 2, +19 more in part 3 for the refusal wording). Payoff: in step 2 those commands join the catalog with the hosted Control forms, `DispatchPath` is deleted, and so are `WorkflowCommand`'s dispatch guard and `Agent.Dispatch`'s branch.
- Each CLI catalog command costs one extra round trip (`native.assets.read` for the football digest).
