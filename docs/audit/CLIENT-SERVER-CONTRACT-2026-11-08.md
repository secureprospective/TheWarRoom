# Client/server step 1b — contract map (2026-11-08)

Evidence, not a plan. Gate (`docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md:50`): *required locked-screen +
authorized CLI/MCP workflows exercise same service with identical outcomes/rejections.*
Parity test: `internal/serverhttp/contract_parity_test.go` (`TestContractParityScreenAndCLI`).

## The three doors as they exist today

| Door | Route | Reaches |
|---|---|---|
| **Wails, local** | `frontend/src/app/shell/localWorkflow.ts:47` `TargetWorkflow` → `internal/app/commissioner/commissioner_preview_api.go:67-80` | `catalog.RunWorkflow` → `leagueservice.Service` (local host). Ten reads bypass it: `localWorkflow.ts:20-44` call a direct Wails binding and send `native.assets.read` only for the authority label. Control forms call `TargetCommissionerPreview/Execute` (`commissioner/proxy.ts:37-38` → `commissioner_api.go:13`, `commissioner_preview_api.go:16`) → `Service.Preview/Execute`, not the catalog. |
| **Hosted (runNative desktop / browser)** | `serverclient/desktop_handler.go:151` → `serverhttp` `WorkflowPath` → `workflow.go:56,68` | `catalog.RunWorkflow` → `leagueservice.NewHosted` (`hosted.go:232`) with football **pinned** (`hosted.go:227,240-249`). Only `WorkflowCommands()` (`commandcatalog/workflow_commands.go:20-34,57-106`) are served. The hosted desktop has no general commissioner preview/execute action (`serverclient/desktop.go:79-95` action list). |
| **CLI / MCP** | `cmd/twr/main.go:191-200`, `cmd/twr/mcp.go:45-77` → `serverclient/agent.go:63-71` → `serverhttp` `DispatchPath` → `hosted.go:251-297` | `Service.Preview` / `Service.Execute` directly — **not** `catalog.RunWorkflow`, and football is **not** pinned (`workflowHost(host, false)`, `hosted.go:227`). The only read is the ledger (`agent.go:49`). Tokens are commissioner-only (`commissioner/token_scope.go:251`). |

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
| `commish.lineup.set` | HQ (`hqNativeLineup.tsx:27`), Control (`HostedLeague.tsx:81`) | cat → `SetLineup` (`workflow_commands.go:133`) | catalog → `SetLineup` | `Service.Execute` → refused `leagueservice/mutations.go:33` | **different path** (DispatchPath bypasses catalog); CLI cannot set a lineup (test-proven) |
| `roster.reserve` | HQ (`hqNativeLineup.tsx:36`), Control | cat | catalog, football pinned | `Service.Execute`, football unpinned | **different path** (DispatchPath, no pin); CLI execute refused "the NFL's listing of him is not held" even with human confirmation (test-proven) |
| `trade.offer` | War Room Trade (`warroomTradeReads.ts:80`) | cat | catalog | `Service.Execute`, commissioner token only | different path (DispatchPath bypasses catalog; owners have no token) |
| `draft.pick` | Control (`HostedLeague.tsx:149`) | cat | catalog (`runWorkflowPick`) | none (`Agent.Dispatch` sends `RawCommand` only, `agent.go:71`; `PickPath` unused by CLI) | missing in CLI |
| `draft.timer`, `draft.skip` | Control (`HostedLeague.tsx:81`) | cat | catalog | `Service.Execute` | different path (DispatchPath) |
| `commish.rules.set` | Control (`RuleChangeForm.tsx:61`) | cmd | none on hosted desktop (no action in `desktop.go:79-95`); `ExecutePath` exists | `Service.Execute` | missing on server (hosted screen route); CLI same service via generic dispatch |
| `commish.notice.publish`, `commish.whatsnew.post` | Control (`NoteForm.tsx:11,14`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| `commish.league.create` | Control (`StartLeagueForm.tsx:34`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| `commish.correction.revert` | Control (`UndoForm.tsx:19`) | cmd | as above | `Service.Execute` | missing on server (hosted screen route) |
| Personal notes / targets, trade-builder plans | War Room Market/Inspector/Trade/Plans, HQ inspector (`offers/api`, `trades/api` importers) | own bindings | none | none | missing on server (personal plans move in step 5); missing in CLI |

## Counts (rows above; a row may carry two marks)

- same service via catalog on Wails + hosted: 14 reads, plus `commish.lineup.set`, `roster.reserve`, `trade.offer`, `draft.pick`, `draft.timer/skip` on those two doors.
- **different path:** 10 Wails direct-binding reads + `commish.ledger.filtered.read` (11); CLI DispatchPath for `commish.lineup.set`, `roster.reserve`, `trade.offer`, `draft.timer/skip` (4 rows).
- **missing on server:** 11 direct-binding reads; 4 Control form rows (hosted desktop has no preview/execute action); personal plans (1) = 16.
- **missing in CLI:** every read except the unfiltered ledger (25 rows) and `draft.pick` = 26. Owners have no CLI at all (commissioner-only tokens).

## What the parity test proves (`TestContractParityScreenAndCLI`)

Same fixture league (`hostedfixture.CreateWorkflowLeague`), fresh per door, same steps, real TLS `serverhttp.Hosted` handler;
screen door = `WorkflowPath` with a human session, CLI door = `serverclient.Agent.Dispatch` with an agent token.

| Step | `roster.reserve` | `commish.lineup.set` |
|---|---|---|
| preview | both accept, no change | both accept, no change |
| stale expected revision | both refuse, unchanged | both refuse, unchanged |
| unknown command | both refuse, unchanged (screen 400; CLI 400) | same |
| ineligible (`0099` not injured / `9999` not rostered) | both refuse, unchanged (CLI reason differs: missing football evidence) | both refuse, unchanged |
| wrong role (owner, franchise.02) | both refuse, unchanged (screen: own-franchise gate; CLI: token issuance refused) | same |
| missing confirmation (CLI only, G2/G3) | refused, unchanged | refused, unchanged |
| execute (confirmed) | **screen records; CLI refused** | **screen records; CLI refused** |

The last row is asserted as the current divergence; it flips to parity only after a ruling (below).
