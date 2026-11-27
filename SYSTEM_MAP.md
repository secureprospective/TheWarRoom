# System Map

A feature directory: find any feature in two hops.

**How to find things**
1. Find the feature in the table below; the slug in the first column is its name everywhere else.
2. The desktop root, its composition adapters, and desktop-only app packages were removed in desktop-removal PR-D and PR-E; remaining backend services live under `internal/` (`docs/architecture/feature-extraction.md`).
3. Frontend code is under `frontend/src/app/<dir>/`, with its tests colocated as `*.test.ts(x)`.
4. To find a file's feature, look it up in `docs/architecture/placement.tsv` (every `internal/` package, one row each; the root holds no Go files).
5. Layers, the hosted command surface, data, external services and the native league are written up at length in `docs/architecture/system-narrative.md`.

6. Building a new server feature? Its plug-in points are the `FUTURE-SEAM` markers listed in `docs/architecture/future-seams.md`; extend those, never build beside them.
7. Data stack, data flow, sandboxes and the CLI/AI-first path (today, should-have, future): `docs/architecture/infra-map.md`.

Paths are relative to the repo root. `frontend/src/app/` is shortened to `app/`. A `*` is a glob.

| Feature | Backend entry | `internal/` packages | Frontend | Tests | Spec | Data |
|---|---|---|---|---|---|---|
| `research` | — | `internal/ingestion/espnnews`, `internal/ingestion/injuries` | `app/research` | `internal/ingestion/espnnews/*_test.go` | `docs/ui/Research_Surface.md` | `internal/ingestion/espnnews/testdata` |
| `race` | — | `internal/app/season`, `internal/app/season/seasontest`, `internal/engine/race`, `internal/ingestion/leaguestandings`, `internal/ingestion/playoffs` | `app/race` | `internal/engine/race/*_test.go` | `docs/ui/Target_UI_Spec_2026-10.md` §14 | `internal/ingestion/playoffs/testdata`, `internal/ingestion/leaguestandings/testdata` |
| `now` | — | `internal/app/now`, `internal/app/season`, `internal/app/season/seasontest`, `internal/app/now/nowtest`, `internal/ingestion/livegame`, `internal/ingestion/livestats`, `internal/ingestion/weeklyresults`; hosted acquisition: `cmd/twr-server/live_*.go`, `internal/store/history/live_overlay.go` | `app/now` | `internal/app/now/*_test.go` | `docs/ui/Target_UI_Spec_2026-10.md` §14 | `internal/app/now/livegame_sources.json`, `internal/ingestion/livegame/testdata`, `internal/ingestion/weeklyresults/testdata` |
| `market` | — | `internal/pickvalue`, `internal/ingestion/marketfeeds` | `app/market`, `app/reports`, `app/assets` | `internal/pickvalue/*_test.go` | `docs/ui/Target_UI_Spec_2026-10.md` §10 | `internal/ingestion/marketfeeds/testdata` |
| `draft` | — | `internal/ingestion/draftreports` | `app/draftreports` (transport in `api.ts`), `app/draftroom` | `internal/ingestion/draftreports/*_test.go` | `docs/ui/target-draft/TheWarRoom_Target_Draft_2026-10.html` | `internal/ingestion/draftreports/testdata` |
| `offers` (offers, personal, pool, bids) | — | `internal/app/mflref/bids`, `internal/engine/bid`, `internal/ingestion/freeagents`, `internal/store/personal` | `app/offers` (MFL reference UI code), `app/commands/mfl.ts` + `mfl.gen.ts` (Go-catalog preflight), `app/personal`, `app/pool` | `internal/engine/bid/*_test.go` | `docs/transactions/Free_Agency_Design.md`, `docs/ui/Target_UI_Spec_2026-10.md` §10 | tables: `internal/store/personal` |
| `contracts` (contracts, roster actions, what-if) | — | `internal/app/mflref/contracts` (frozen MFL API, Host and plan logic), `internal/mflref/whatif` (session model and lifecycle), `internal/store/state/whatif_session.go` (shared migration and session wire records), `internal/app/mflref/contracts/contractstest`, `internal/engine/contractacts`, `internal/engine/eligibility`, `internal/mflref/lineup`, `internal/transactions` (+ `acquisitions`, `contracts`, `deadcap`, `freeagency`), `internal/ingestion/salaryadjustments` | `app/contracts`, `app/data` | `internal/app/mflref/contracts/*_test.go` | `docs/transactions/Salary_Ledger_Design.md`, `docs/transactions/Transaction_Reference.md`, `docs/league-rules/Official_Rulebook.md` | `internal/engine/eligibility/testdata`, `internal/mflref/lineup/testdata` |
| `sources` (data sources, crosswalk) | — | `internal/app/sources`, `internal/archive`, `internal/measures`, `internal/ingestion/college`, `internal/ingestion/contracts`, `internal/ingestion/crosswalk`, `internal/ingestion/feeds`, `internal/ingestion/leaguearchive`, `internal/ingestion/players` | no screen; command-only refresh and settings adapters retained | `internal/app/sources/*test.go` | `docs/sources/Approved_Sources.md`, `docs/data-layer/Measure_Dictionary.md` | `internal/measures/measures.csv`, `internal/measures/sources.csv`, `internal/measures/feeds.csv`, `internal/ingestion/players/testdata` |
| `trades` | — | `internal/app/mflref/trades`, `internal/ingestion/pendingtrades` | `app/trades` (transport in `api.ts`), `app/shell` (trade desk), `app/data` | `internal/app/mflref/trades/*_test.go` | `docs/transactions/Transaction_Reference.md`, `docs/ui/Target_UI_Spec_2026-10.md` §6 | — |
| `scoring` (rankings M1, power rankings M2) | — | `internal/app/scoring`, `internal/engine`, `internal/model`, `internal/model/fit`, `internal/modelrun`, `internal/appvalue` (league-scored read over the unchanged model), `internal/m2service`, `internal/powerrankings`, `internal/ingestion/playerscores` | none (no screen yet) | `internal/app/scoring/*_test.go` | `docs/scoring-engine/Engine_Specification.md`, `docs/modules/M2_Power_Rankings_Design.md` | `internal/store/params/fitted.json`, `docs/fit/Fit_Report.md` |
| `control` (league room, refresh, clock, MFL key) | — | `internal/leagueclock`, `internal/leaguefeed`, `internal/leagueweek`, `internal/mflkey`, `internal/store/params`, `internal/store/rulebook`, `internal/ingestion/league`, `internal/ingestion/leagueschedule`, `internal/ingestion/livescoring`, `internal/ingestion/nflschedule`, `internal/ingestion/rosters`, `internal/ingestion/transactions` | `app/control` (MFL reference labels; hosted startup fails closed), `app/clock`, `app/commands` | `internal/leagueclock/*_test.go` | `docs/transactions/Season_Phase_Design.md`, `docs/data-layer/MFL_API_Specification.md` | `internal/ingestion/league/testdata`, `internal/ingestion/nflschedule/testdata`, tables: `internal/store/rulebook` |
| `commissioner` (commissioner + native league) | — | `internal/commissioner`, `internal/commissioner/market` (market judgment over staged ledger Frames), `internal/governance`, `internal/leaguerules`, `internal/nativecheck`, `internal/fantasyscore`, `internal/fantasyscore/source`, `internal/engine/rookiedraft`, `internal/engine/competition` (tie-breakers, playoff fields and brackets, schedule generation) | `app/commissioner` (hosted franchise: `HostedLeague.tsx`, in-memory workflow: `hostedRoom.ts`; shared hosted rule editor/review: `HostedRules.tsx`, `HostedReview.tsx`; teamless commissioner view: `HostedCommissionerLeague.tsx`; bids/waivers/trades/results/rollover forms: `HostedWorkflows.tsx`; checked readings: `hostedMarkets.ts`), `app/draftroom`, `app/governed` | `internal/commissioner/*_test.go` | `docs/commissioner/README.md`, `docs/ui/Target_UI_Spec_2026-10.md` §16, `docs/transactions/Season_Rollover_Design.md`, `docs/architecture/native-scoring.md` (native scoring and the 32-team default) | `internal/commissioner/testdata`, `internal/leaguerules/testdata`, `internal/fantasyscore/testdata`, `internal/fantasyscore/source/testdata`, `docs/commissioner/Commissioner_Governs_Edges.tsv` |
| `platform` (startup, shutdown, locks, logging, hand-offs, fixture HTTPS) | — | `internal/app/mflref/planwatch` (local plan polling and landing), `internal/app/report`, `internal/leagueservice` (shared native reads/previews; release matrix: catalog mutation fixtures in `cmd/twr/testdata/release-mutations.json`, real CLI/MCP matrix in `cmd/twr/release_matrix_test.go`), `tools/hostedfixture` (shared disposable league setup), `internal/serverversion` (server-first compatibility and read-only preflight), `internal/serverhttp`, `internal/serverclient` (hosted owner workflow: `internal/serverhttp/workflow.go`, `internal/commandcatalog/workflow_commands.go` (42 league workflows plus 3 account-preference workflows; parity fixture in `internal/serverhttp/testdata/workflow-before.json`), `internal/serverhttp/market_workflow_test.go`; entry/client: `cmd/twr-server`; agent CLI/MCP: `cmd/twr`), `internal/accounts`, `internal/store/accounts` (account bootstrap, hosted preferences, no league grants), `internal/db`, `internal/offsite` (encrypted hosted R2 recovery, coverage gate and restore seam; `deploy/server/BACKUPS.md`; spare-host restore and incident runbook: `deploy/server/RESTORE.md`), `internal/serverstatus` (console-only health dashboard over an owner-only socket; daily/away routine: `deploy/server/CHECKLIST.md`), `internal/releasetrust` (public bootstrap and persistent TUF verification; `internal/releasetrust/README.md`), `internal/leaguerecovery` (offline fresh-database roster/contract recovery; no authority grants), `internal/leaguetransfer` (encrypted replay-verified exports, sealed activation and fenced reverse moves; `internal/leaguetransfer/README.md`), `internal/offline` (shared network-failure wording), `internal/headshot`, `internal/store/mflref/moves` | `app/shell`, `app/cards`, `app/look`, `app/registry`, `app/data/query.ts` (shared typed read/cache hook; design in `docs/architecture/frontend-reads.md`) | — | `docs/backend/Backend_Architecture.md`, `docs/agent-codex.md` | `docs/ui/endpoint-registry.csv` |
| `core` (app state, shared domain, transport, stores) | — | `internal/commandcatalog`, `internal/domain`, `internal/playerid`, `internal/numeric`, `internal/feedwire` (acquisition-free held feed envelopes), `internal/mflref/envelope`, `internal/snapshot`, `internal/normalize`, `internal/mfl`, `internal/ingestion`, `internal/store/state`, `internal/store/history`, `internal/playerrecord` | — | in each package; command registry in `internal/commandcatalog` (native commands, hosted workflows, browser exposure and an offline operator-console exposure; shared JSON schema); `frontend/scripts/nativegen` generates CLI verbs, command-bar candidates and unified TS argument maps and the audited held-read projection in `app/commands/reads.gen.ts`; generated browser list and frozen browser fixture in `app/commands` | `docs/data-layer/MFL_API_Specification.md`, `docs/data-layer/MFL_Scoring_Rules_Decode.md` | `internal/ingestion/testdata`, tables: `internal/store/state`, `internal/store/history` |


Native cap runway: `internal/commissioner/native_runway.go` composes the existing cap counter for four
seasons per franchise; exposed additively by `TargetNativeCapCheck`. Flow and limits:
`docs/architecture/cap-runway.md`.

Native creation/start: `internal/commissioner/native_start.go` builds source-selected starting assets.
The app's `history.ExecuteCommissionerWithFacts` records native Create + start atomically;
`ledgerJobs` supplies the start job only to unstarted leagues. Flow and evidence:
`docs/architecture/native-scoring.md` § Main integration — part 8.
O-Line auction close and pending minimum-contract assignments: `internal/commissioner/oline_close.go`,
`internal/store/history/native_oline_clock.go`; flow: `docs/architecture/oline-auction-close.md`.

Roster/contract recovery: `internal/leaguerecovery` (offline orchestration),
`internal/store/history/roster_restore.go` and `internal/store/history/stand_up.go` (atomic capture/rebuild),
`internal/commissioner/roster_restore.go` (checked private format),
`internal/db/league_recovery.go` (point/log SQL). History v10 keeps permanent player identity in SQL through the migration
helpers in `internal/db/player_identity.go`; legacy MFL reads use a compatibility
view. Stand-up produces a checked **read-only** league, never authority grants.
Evidence, responsibility split and justified debt:
`docs/architecture/storage-recovery-identity.md` and `internal/store/history/README.md`.

The Lab (lab, facts, labcompile, manifest, labparity) moved out to its own LeagueLab repo on 2026-11-04.

Tooling outside the features: `scripts/value_board.py` (offline current-engine valuation triage; tagged harness in `internal/modelrun/value_board_test.go`, method in `docs/architecture/valuation-review.md`), `tools/fit` (writes the fitted params), `tools/stresscheck` (lab-only read-only integrity and ledger reader), `tools/twr-practice` (development-only hosted HTTPS sandbox launcher; `tools/twr-practice/README.md`), `tools/ifaceguard`, `tools/boundarycheck` (function/source boundary ratchet), `scripts/`.

Current boundary-extraction inventory and resume point: `docs/architecture/app-boundary.md`.
Finished-score finality is in `internal/app/now/nfl_reading.go`. Desktop-only app packages were removed in PR-E;
remaining MFL reference packages live under `internal/app/mflref/`.

Reference schedule/lineup refresh policy lives in `internal/app/mflref/migration`.
Reference schedule following is `schedule.follow` in `internal/leaguerules/competition.go`.
Shared roster counting lives in `internal/powerrankings/lineup.go`, not the MFL-backed M2 service.
Resolved player listing facts live in `internal/domain/types.go`; `normalize.PlayerFacts` is a compatibility alias.

## Where new code goes

The React PWA calls the commands exposed by `internal/serverhttp`. `cmd/twr-server`
wires the hosted services; `cmd/twr` supplies CLI/MCP access.

- A new root Go file needs a row in `docs/architecture/placement.tsv`, and the root count may
  not rise above `docs/architecture/root-go-baseline.txt`. Both are checked by `make lint` and CI
  (`scripts/check-placement.sh`, `scripts/check-root-ratchet.sh`).
- A new `internal/` package needs a manifest row and a mention in this table's row for its feature.
- A new feature needs a row here before its first file lands.

## Rules CI enforces

| Guard | Failure and fix |
|---|---|
| Root-file ratchet (`make placement`) | Root Go file count exceeds `docs/architecture/root-go-baseline.txt`: extract logic and tests into their feature; lower the baseline as files leave, never raise it to hide growth. |
| Placement (`make placement`) | A root file or internal package is missing, duplicated, stale or unindexed in `docs/architecture/placement.tsv`: correct its feature row and this map; remove obsolete paths. |
| Import boundaries (`make lint`, `.golangci.yml` depguard) | Non-app internals cannot import `internal/app`. App features may use their declared subtree and shared services (`internal/app/report`, `internal/app/season`, and the catalog where declared), not sibling features. Existing engine, transport and store rules still apply. |
| Frontend transport (`make frontend-transport`) | Do not reintroduce generated desktop imports or the legacy global runtime into frontend source or tests. |

Move handoff/withdrawal, evidence refresh/watch scheduling and roster-verdict rebasing policy
live in `internal/app/mflref/planwatch`.

MFL migration reading/translation lives in `internal/leaguemigration/mflsource`; the native
`internal/leaguemigration` service owns its `Source`/`Candidate` port and destination writes.
`internal/leaguemigration/migrationtest` is fenced MFL-only fixture support.

Contracts and trade package names, method names and stored wire keys stay unchanged. Session models
and replay baselines belong to `internal/mflref/whatif`; the shared whatif database
migration and raw records remain in `internal/store/state/whatif_session.go`.

`docs/architecture/boundary-mfl.tsv` declares exact reference fences, direct legacy
import pairs (each with its remaining-ownership reason), and native domain roots.
The boundary gate checks every production dependency below those native roots,
including indirect paths through legacy debt. The migration source adapter is fenced,
not an exemption for the destination writer. Mixed application adapters still
listed as debt are not claimed to be isolated native domain packages.

Held MFL seed translation lives in `internal/app/mflref/seed`; local trade-plan SQL
lives in `internal/store/mflref/tradeplans`. Shared state migrations retain the unchanged DDL.

MFL-only bid-plan checks live in `internal/app/mflref/bids`; exact arithmetic stays in
`internal/engine/bid` because transaction pricing also uses its salary floor.

The frozen reference surface and compatible stored-plan names are documented in
`docs/architecture/mfl-reference-fence.md`.

MFL import uses `twr-server league-load`, `internal/leaguemigration`, and its MFL source.
See `cmd/twr-server/README.md` for the operator workflow.

League-scoped values are prepared by `internal/appvalue`.
