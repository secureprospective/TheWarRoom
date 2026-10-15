# System Map

What exists in TheWarRoom and where new code belongs. Update it in the same commit as any new
package, IPC method or external service. Current as of 2026-10-04 (Stage 7 of
`docs/build-handoffs/Core_Build_Plan_2026-10.md`).

## Layers and packages

Import rules marked **(depguard)** are build errors in `.golangci.yml`, not conventions.

| Layer | Package | Job |
|---|---|---|
| App | repo root (`package main`) | Wails entrypoint (`main.go`), the `App` composition root (`app.go`) and the IPC adapters (`*_app.go`). Adapters validate, route and format; no business logic, no SQL. |
| Composition | `internal/commissioner` | Typed app-native command validation, session/authorization ports, receipts, compensation replay and privacy-filtered ledger/people readings. Imports the pure governance policy; the production authorizer replays the transaction's existing records and uses census permissions, without I/O or re-entry. `history.ExecuteCommissioner` trial-folds the proposed change and appends once in `history.db`'s `league_event`; no what-if or MFL writes. Creation takes `{name, preset: legacy_nfl|empty, franchiseCount}`; Go expands `franchise.01`..`NN` (named "Franchise N") and, for Legacy NFL, the whole typed preset seed into the durable record, so replay never depends on later preset code; the creator becomes commissioner. `commish.rules.set` changes one `internal/leaguerules` catalogue field: `rules.go` maps every field to exactly one census row (authorization and gravity; apply-now on a competitive field is G3), `rules_season.go` resolves rules by season and instant; `next_rollover` changes stay pending until the recorded rollover instant arrives, and ended-season reads ignore later compensation, and the trial resolve refuses incoherent rule sets before append. `rules_reading.go` is the public sectioned rules reading. CM-018..021 people changes, rule changes and correction reversal share the one append path; reversal also requires the target's original permission. Root `commissioner_app.go` exposes execute/ledger/people/rules/native-leagues IPC, using only the injected local session. Rule rows also carry the in-force value as plain JSON (`leaguerules.Current`), a percent unit and the safeguard Go applies for each timing; the people reading says what this principal's role may start (`yourRole`, `mayChangeRules`, `mayGrantRoles`); a ledger entry says where this principal may undo it (`undo`: league or private). The frontend lives in `frontend/src/app/commissioner/`, all lazy under Control Room › League: one shared state (`proxy.ts`, `leagueState`) holds the league picker, the ledger/rules/people readings and the outcome of the last form, and reloads all three on a receipt; typed parsers `rules.ts`, `people.ts`; screens `LeagueRules`, `LeaguePeople`, `LeagueLedger` (with `UndoEntry`) and `LeagueDesk` (picker plus the forms `StartLeagueForm`, `RuleChangeForm`, `RoleForm`). Forms only build the typed body and send it through the `commissioner.execute` verb, with a request key that repeats on a retry of the same submit (`ChangeBar`, `forms.ts`); G2 confirms by `HoldToFire` (a self-contained hold timer, the one reviewed keyboard exception besides the command bar). Rule values are edited through `ruleDraft.ts` (drafts to and from Go's plain JSON) and `RuleInput.tsx`; list and map add/remove go through the `commissioner.draft` verb. Go stays the authority. Succession forms and ballots are not built. CM-022 succession, inactivity, elections and vote commands (`succession.go`) use recorded ledger times and the same append path: CM-022 is a composite census row, so each command names its gate (commissioner only for the inactivity and voting periods and the activity check-in; co-commissioner or DOT to open votes; a vote's target may not close it) and the governance trial fold decides the rest. Every recorded change counts as its author's activity; an inactivity takeover waits out the longer of the two periods while season dates need a ruling. People readings show public vote counts and succession status, with voter participation private to C/CC; ballot ledger entries are private, tie rulings public. |
| Credentials | `internal/mflkey` | OS keyring store; redacted credential type, bounded operations. |
| Layer 1 | `internal/ingestion/pendingtrades` | Keyed pending-trade feed and candidate-key verification; no franchise inference. |
| Transport | `internal/mfl` | MFL HTTP client: rate limit, host discovery, 429 backoff. No domain types. |
| Transport | `internal/archive` | The HTTP transport every outbound request goes through: it records each response body (sha256, gzip) and each attempt to a `Sink`. Leaf. |
| Layer 1 | `internal/ingestion` | Fetchers returning raw `Raw*` records. Shared helpers in the root package (`LeagueExport`, `FetchLeagueExport`, the CSV and CFBD plumbing); one subpackage per source. |
| Layer 1 | `internal/ingestion/feeds` | The table-driven loader: reads any file in `feeds.csv` and maps its columns through `source_fields.csv` into a batch for the measure store. A new signal from such a file is registry rows, not code. |
| Layer 1 | `internal/ingestion/college` | CFBD season stats as a batch for the measure store: one call per college season, each team's totals summed for the share denominators. `Map` reads an archived body, so a season is re-read when the registry maps a measure it lacks. |
| Layer 1 | `internal/ingestion/contracts` | NFL contracts (OverTheCap via nflverse, Parquet) as a batch: each player's contract terms under the season signed. Read every 30 days. |
| Layer 1 | `internal/ingestion/injuries` | MFL's NFL injury designations (the league-less `injuries` export, fetched without L from api.myfantasyleague.com), the IR rule's input. Empty bodies and MFL's HTTP-200 error envelope both fail the fetch: a glitch must read "not checked", never "nobody is injured". |
| Layer 1 | `internal/ingestion/weeklyresults` | Validated MFL weeklyResults for the whole season (W=YTD): every week's games, scores in exact hundredths, MFL's results and each side's started and benched players; unscored weeks keep their set lineups. |
| Layer 1 | `internal/ingestion/playoffs` | Validated MFL playoffBrackets and playoffBracket (BRACKET_ID) exports: each bracket's games, seeds, scores and where an unknown side comes from. |
| Layer 1 | `internal/ingestion/livegame` | Personal-use ESPN scoreboard, summary and core participant plays; Yahoo score/last-play failover, timed held copy and window-scoped polite demand polling. Injected transport; fixtures only. |
| Layer 1 | `internal/ingestion/marketfeeds` | Validated MFL week projections and MFL-wide ranks, all-period ADP and auction-budget shares; fixture-only tests. |
| Layer 1 | `internal/ingestion/freeagents` | MFL's `freeAgents` league export: the players on no roster, the pool bid plans are made from. MFL's HTTP-200 error envelope and a reply with no `freeAgents` section both fail the fetch; an empty pool is a valid answer. |
| Layer 1 | `internal/normalize` | Raw records → domain types: the players lookup and roster join. |
| Leaf | `internal/domain`, `internal/playerid`, `internal/numeric`, `internal/scouting` | Value types. `playerid.New` is the only way to build a `PlayerID`. `scouting` holds the Layer 4 input types. |
| Leaf | `internal/measures` | The measure registry: `measures.csv`, `sources.csv`, `source_fields.csv` and `feeds.csv`, embedded and validated. Generates `docs/data-layer/Measure_Dictionary.md` (`make measure-dictionary`). Adding a source is a CSV row, not code. |
| Leaf | `internal/nativecheck` | Pure native active-roster limits, per-position/group lineup checks, constrained best-lineup selection, invalid-lineup penalty, missed flag, IR/taxi verdicts and salary-cap counting. Cap counting applies season rules, reserve shares and signed recorded charges with the league's money increment; `CapAllows` judges a net counted move delta (including conditional drops), refusing hard-cap overages and flagging soft-cap overages with a plain-word deadline in a supplied viewer zone or Needs a ruling. `NewWaiverPolicy` consumes resolved CM-012 settings without inventing missing values; `Close` dates an unprocessed window from its original cut and `Award` chooses among already-eligible franchises using a supplied priority-first standings or rolling order (rolling moves the winner last). The native commissioner ledger consumes this policy for cuts, private claims and atomic awards; no imported league or hand-off state is changed. Rules and held football facts arrive as values; no MFL policy, I/O or stores. |
| Leaf | `internal/leaguerules` | Typed commissioner-owned season rules, explicit unset values, cited Legacy NFL preset, pure ordered change resolver and preview diffs. `catalogue.go` exports the read-only field catalogue (label, section, value schema, competitive), plain-JSON `DecodeSetting`, whole-value stored encoding for durable seeds, and plain-words rendering. Imports only the standard library and pure domain value types; no store, transport, ingestion or I/O. |
| Store | `internal/db` | SQLite pools: one write connection, many read-only ones, one WAL file. |
| Store | `internal/store/rulebook` | League rules from MFL as immutable versions with one active pointer, plus commissioner overrides. |
| Store | `internal/store/params` | Engine calibration: shipped defaults plus admin overrides, league-wide or per position. The Layer 4 settings are seeded from `l4.Defaults`; the model's fitted values from the embedded `fitted.json`, marked calibrated. Shipped defaults are upserted at start-up; overrides are never touched. A set splits into the board's part and the model's (`model.*`, `dynasty.*`); each run records only its own. |
| Store | `internal/store/moves` | Append-only envelope specs and replayable audit entries in `thewarroom.db`; restart-safe move receipts and awaiting evidence queries. `bid_facts` holds current bid figures; `bid_conditions` keeps the conditional drop by audit version, keyed by an envelope's correlation id: the owner's own bid figures, no franchise column, frozen by a trigger once the bid is handed off. |
| Store | `internal/store/state` | Two things behind one `Reader`. **`Mirror`**: the league as MFL states it (season, rosters with contracts, salary adjustments), replaced whole by a refresh; every score surface reads it. **`Store`**: the what-if league in `whatif.db`, seeded from the mirror (rosters and MFL salary adjustments, so its cap starts equal): rosters, contracts, the contract-year ledger, dead cap, cap relief, phases, feed, calendar. Append-only ledgers; the transaction coordinator holds the only `Writer`. |
| Store | `internal/store/history` | `history.db`: everything the app cannot rebuild. The fetch archive (`raw_archive`, `fetch_log`), facts per measure appended on change (`observations`, read as of a date through `Features`), source health, and scoring runs with the param set, engine and inputs they used. Append-only, enforced by triggers. |
| Analytics | `internal/facts` | Pure pinned-input compiler for season, franchise, standings, rules, calendar, player identity/history/conflicts, source provenance, loader notes, curated trade-note readings, weekly games and lineups, player weeks, injuries, weekly rosters, NFL games, points allowed, playoffs, contracts, cap charges, the draft, picks owned, transactions and their assets, trade bait and MFL-wide ADP/AAV: every Lab relation except the checks; typed in-memory columns only. `Facts.Views()` computes the Lab's named readings (played_game, regular_week, all_play_week, team_season, trade_asset, offer, roster_move) from the columns on demand. `internal/facts/parity_test.go` compares the independent Python golden in verify; the full archive is opt-in. No snapshot publication or database tables. |
| Analytics | `internal/labcompile` | Pure facts-to-Lab compiler: calendar, production, contracts, trades/draft, directories, ledger/source proofs and full wins/replacement, five-season forecast, pick curves, market and rolling backtest analysis. `Compile` produces the complete document; `Build` also returns an immutable `Model` whose `Player`/`Pick` API uses the trade-leg valuer at a supplied moment, with explicit unvalued reasons. Passive Python kernel/method/analysis goldens and reduced conservative smoke run in verify; opt-in full archive parity covers all 18 keys and player API/leg equality, while recorded-analysis document parity remains isolated. The app fits the model after launch (`fitAssetModel`, after `warmArchive`) and C-01 reads `Model.Player` through `labAssets`; the Lab screens still read the frozen document. |
| Analytics | `internal/lab` | Frozen fingerprint-checked compiled Lab facts, typed columnar decoder, pure Market fitting, delivery, price lists, calendar, Draft slot returns, team-side Counterparties profiles and owner-scoped offer endings. Offline JS goldens in `tools/labparity` pin the reference engine and data hashes; Node is never required at runtime or by verify. Exact missing-input exceptions live in `testdata/missing-inputs.csv`; `facts-reference.py` independently rebuilds the Python facts oracle in a disposable directory, with its source-row proof and provenance under `internal/lab/testdata/facts`. The reduced three-season golden retains every reference relation; full parity is opt-in. |
| Policy | `internal/governance` | Pure typed-event fold for principals, seats, private invites, abilities, role permissions, DOT recusal and succession; standard library only, no I/O or stores. Census permission data, a single co-commissioner seat and explicit activity/ballot evidence; locked all-franchise election majority, timed recorded runoff closure and candidate-recused tie rulings. |
| Engine | `internal/engine` | The scoring pipeline as pure functions (L1, L3, L4 dispatch, L5, L6). |
| Engine | `internal/engine/bid` | The free-agent auction's arithmetic and rules (rulebook §6) as pure functions: bid points in exact thousandths, the minimum salary by experience (`freeagency` reads the same table), eleven named rule checks with Go-written reasons (including cap coverage by one conditional drop), and the ProBoards post text. |
| Engine | `internal/engine/race` | The league race as pure functions over weekly results: records, all-play, streaks, games back, MFL's standingsSort applied step by step with a trace, playoff seeding (division leaders first), the opening round and the bubble. Proved against the real 2025 and 2026 seasons in `race_test.go`. |
| Engine | `internal/engine/eligibility` | MFL's IR and taxi rules as pure verdicts: per player and direction, eligible / ineligible (reason) / unknown (reason), from held feeds only (injuries status, players draft year, roster status, league capacities). The captured 2026-10-10 owner pages are its committed test oracle (`testdata/answer-key.csv`). |
| Engine | `internal/model` | The measurable's model: a player's league-points percentile blended with a prior from pre-NFL facts, the talent and survival arcs, and `Params`, the fitted values per position (`model.*@POS`). Pure. |
| Engine | `internal/model/fit` | Fits `model.Params` from stored seasons, scoring every choice on a holdout season. Pure; `cmd/fit` runs it. |
| Engine | `internal/engine/l4` | Layer 4: one rubric routine driven by a per-position settings table. The adjustable numbers in the table are params (`l4.film.cap@WR` and so on); composition reads them back for each run. |
| Composition | `internal/composition` | Engine inputs from the stores plus per-player facts. |
| Composition | `internal/rankings` | M1: scores every rostered player from a params snapshot and history features, and writes one scoring run. |
| Composition | `internal/m2service`, `internal/powerrankings` | M2: counts each roster's model values (on-field-now or dynasty, league points per game) as the best legal lineup, the top N or the whole roster, and z-blends them: this season with the results (all-play when MFL reports it, otherwise points for) at 4 ÷ (4 + weeks played) or the slider's weight; the franchise with the roster's age. `m2service.Outlooks` adds the context columns (cap room, schedule luck, projected record), never in the score. |
| Composition | `internal/scouting/assembly` | Builds today's board's scouting profiles: RAS, coverage and school tier from their feeds, college share and breakout age from history. |
| Composition | `internal/modelrun` | The measurables: values every rostered player and held MFL free agent exactly once with `internal/model` and the run's params, and writes a model run beside the board. `AtLeaguePositions` puts every player MFL lists at MFL's position before the scales are built; `cmd/fit` does the same from MFL's players export. |
| Composition | `internal/snapshot` | Pure, deterministic target UI snapshot builder over a read-only source; shared freshness is `domain.Freshness`. Its `eligibility` section holds the IR/taxi verdicts for every rostered player plus each input feed's provenance. |
| Tooling | `cmd/fixtures` | Copies league and history snapshots to disposable databases, initializes the app stores offline and exports the shared snapshot. |
| Mutation | `internal/transactions` | The `Coordinator`: every league-state change runs here, in one transaction. Handler subpackages (`acquisitions`, `contracts`, `deadcap`, `freeagency`) are reachable only through it. |
| Tooling | `cmd/fit` | Reads a history database, runs `model/fit`, and writes `internal/store/params/fitted.json` (shipped as calibrated defaults) and `docs/fit/Fit_Report.md`. |
| Tooling | `tools/ifaceguard` | Vet tool: no `interface{}`/`any` in exported signatures. |

**(depguard)**
- `mfl`, `ingestion` and `normalize` never import `engine`, `store`, `transactions` or
  `database/sql`.
- `engine` and `model` import no store, transport, ingestion, normalize, `database/sql`, `net` or `os`.
- The four stores (`rulebook`, `state`, `params`, `history`) never import each other.
- `measures` imports nothing of the app's but `domain`; `archive` imports nothing of the app's.
- `database/sql` is confined to `db` and `store`.
- `transactions/*` handler packages are imported only by `transactions`.

## App-owned calendar reading

`internal/leagueclock/action_calendar.go` dates commissioner action windows at civil midnight in
league time, assigns shared dates to the chosen side, and provides one default-deny action decision
with exact trade cutoff and franchise/time-scoped exceptions. It carries live closes and countdown
seconds. `Countdown` calculates a bid's late-bid mark from its configured duration; bid-plan wiring
is still pending. The imported (MFL league) clock keeps Rulebook §14's week 9 first-kickoff deadline.
`internal/commissioner/calendar.go` folds public exception Acts (`commish.calendar.exception`, CM-026,
G3) and rule compensation into the reading. The existing history transaction authorizes, validates,
retries and appends the exception. `league_calendar_app.go` exposes `TargetLeagueCalendar` for an
explicit app league, season, franchise, action and owner time zone; it reads only durable local events.
This read port does not yet replace the what-if transaction phase gate or the imported owner screens.
`internal/commissioner/schedule_generator.go` generates complete division-weighted rounds for 2–32
franchises, with one rotating bye in odd leagues. `schedule.generate`, `schedule.edit` and
`schedule.publish` share the authorized/versioned/idempotent append path (CM-015, G2; published
week edits are G3). Generation reads the current schedule rules and division alignment; an unset
pattern refuses to guess. `league_schedule_app.go` exposes `TargetLeagueSchedule`: only the published
projection, replayed from local events and compensations. Draft changes never replace that reading.
`internal/commissioner/windows.go` holds the named window doors, `calendar.window.set`, `.move` and
`.remove` (CM-023, G2, co-commissioner): each replays into the same `calendar.windows` value a
`rules.set` writes (`windowChange` feeds the season resolver's `seasonChanges`), so the two doors read one projection and a
reversal that would strand a later window change is refused. `gate.go` holds `NativeActionGate`, the one
decision every native owner action asks: it answers through `ActionCalendar.Decide` (exceptions and the
exact trade deadline included), says in the owner's local time which window opens a closed action, and
mints the `Permit` an action needs to run. Root `native_gate.go` is the registry and its only door,
`runNativeAction`; `native_gate_test.go` runs every registered action shut and open. The registry now contains `roster.cut` (cut window) and `waivers.claim` (free-agent window);
`native_waivers_app.go` binds their native append callbacks. The append transaction rechecks the gate
and current owner seat, including callers below the IPC door. Now/Race integration and the remaining census
owner-edge wiring are still pending.

## IPC surface

| File | Methods |
|---|---|
| `league_calendar_app.go` | `TargetLeagueCalendar(RawLeagueCalendarScope)`: app-owned action windows, live close countdowns, exact chosen trade deadline and public scoped exceptions, replayed from the league ledger and localized for the owner. |
| `league_schedule_app.go` | `TargetLeagueSchedule(RawLeagueScheduleScope)`: published native head-to-head schedule only; draft changes are not owner readings. |
| `market_surfaces_app.go`, `market_profile_app.go` | `TargetMarketLists`, `TargetPickClock`, `TargetMarketStretches`, `TargetAssetProfile`: validated historical scopes and asset kinds; typed Go-computed display sections in `Reading[MarketSurface]`, fitted ranges and archive disclosure, no spread grade. |
| `counterparties_app.go`, `counterpanel_app.go` | `TargetCounterparties`, `TargetCounterpartTeam`: validated historical standing/window/posture scopes; tier cards, trends, pair matrix, sortable team values and season/asset-flow/partner Inspector readings. Lazy frontend in `frontend/src/app/counterparties`, franchise ids resolved against the shared snapshot. |
| `lab_app.go` | `TargetMarket` validates season/phase input and returns a typed historical Market payload in the generic `Reading` shell extending `snapshot.Sourced`; dated provenance, count/threshold and Go-authored counting disclosure, no spread grade. |
| `whatif_app.go` | `TargetWhatIf` validates read/cut/buyout/undo/reset intents; per-franchise persistent baseline and ordered moves in `whatif.db`. Generation-scoped coordinator ledgers replay on mutation, publish after success, keep the mirror read-only, block stale edits until Reset. Lazy Contracts panel shows Go-authored before/after cents and rules. |
| `native_legality_app.go` | `TargetNativeCapCheck(league, viewerZone)`, `TargetNativeRosterCheck`, `TargetNativeReserveCheck`, `TargetNativeLineupCheck`, `TargetNativeHouseRules`: app-owned recorded rosters and season-effective rules, with held player positions, NFL designations and draft years. No refresh, MFL acceptance promise, score-feed wiring or owner-screen migration. |
| `native_seed_app.go` | `TargetNativeSeed(RawNativeSeedRequest)` records one held Legacy NFL starting snapshot in the commissioner event stream: rosters, contract terms, reserve status, franchises, current/future picks and salary adjustments. Go captures the inputs without network or client-supplied assets; retries retain the original receipt after mirror refresh. `TargetNativeAssets` replays that app-owned snapshot, separate from MFL and what-if. `TargetNativeDirectory` reads the native franchise directory: recorded seed names/abbreviations, later active identity edits, current governance seats, and season-resolved alignment (held seed divisions carried into the shared rules resolver). `TargetCommissionerPeople` shares this directory projection; pending alignment changes do not apply, and compensation restores the recorded basis. No MFL directory or hand-off is changed. Empty leagues have no seed. |
| `contracts_app.go` | `TargetContracts` validates team/position/sort scopes and returns separate typed `Reading` shells for league contracts, my-franchise cap, literal MFL salary adjustments. `TargetPlayerContract` validates player IDs and returns observed current salary / explicitly flat-filled inferred future years. Both read a coherent `Mirror.ContractsView`, never the what-if ledger. Lazy frontend in `frontend/src/app/contracts`; every player Inspector includes contract years. |
| `market_feeds_app.go`, `market_feeds_reading.go`, `report_reading.go` | `TargetMarketReportsRefresh(force)` and `TargetMarketReports`: held week-scoped projections and daily MFL-wide market feeds, archive fallback; shared Go Report and lazy React ReportView used by draft and player-market reports; Go-written pool and Inspector market facts. |
| `pool_app.go` | `TargetPoolRefresh(force bool)` fetches MFL's free-agent list (skipped when fresh); `TargetPool(RawPoolScope)` reads the held list with no network: name, position, team, age, experience and the §6 minimum per row, filtered and sorted in Go, in a `Reading[Pool]` that is failed before the first fetch, empty (thin) when MFL lists nobody, stale when a refresh fails. |
| `bids_app.go`, `bids_plans_app.go` | Bid plans (intent `bid.submit`): `TargetBidPreview(RawBidInput)` checks typed figures against the rules without saving; `TargetBidDraft`, `TargetBidRevise`, `TargetBidHandOff` (records the copied ProBoards post; refused if the rules no longer pass), `TargetBidPosted` (the owner says it is up), `TargetBidWithdraw` (all nonterminal local plans; never removes a ProBoards post); `TargetBidPlans(franchiseID)` lists the owner's plans with Go's words, live rules and the actions on offer. The landing watcher looks for the player on the owner's roster with the planned contract. `TargetMoves` leaves bid plans out. |
| `moves_app.go` | `TargetHandOff(correlationID string) (envelope.Receipt, error)` saves before browser hand-off; `TargetWithdrawRoster(correlationID)` ends a local IR/taxi plan without changing MFL; `TargetCheckMoves() error` wakes landing checks; `target:moves` signals a hand-off or changed move. |
| `moves_watcher.go` | Cancellable landing watcher; shared 60-second wake/fast-mode rate limit and persisted restart recovery. |
| `livegame_app.go`, `livegame_spine.go` | `TargetLiveGame(open, follow, mine)`: demand-driven Now poll on held NFL windows; MFL liveScoring at 30 s, ESPN board at 30 s/details at 15 s with capped backoff; separate held detail time, strict ESPN crosswalk starter matching, cached personal-use photos. Source URLs are data in `livegame_sources.json`. Lazy `frontend/src/app/now/LiveGame.tsx`; `game.follow id=` command. |
| `season_app.go` | Held season feeds (transactions, lineups, pendingTrades, injuries); `TargetSeason` reads without network; `target:season` signals refreshes and key changes. Ready roster plans whose held eligibility turned ineligible return to Draft after a refresh (`revalidateReadyRosterMoves`, in `moves_app.go`). |
| `target_app.go` | `TargetSnapshot` (initialized mirror and rulebook, cached live directory with archive fallback, held injuries feed fetched on first use; shared fixture/live contract; keeps the last good build); `TargetClock` (phase log and commissioner calendar from the what-if store; no network); `TargetDraftIR`, `TargetDraftTaxi`, `TargetMoves` (draft `roster.ir` / `roster.taxi` against the kept build into the persistent move store; no network) |
| `mflkey_app.go` | `MFLKeyStatus`, `SetMFLKey`, `DeleteMFLKey` (keyring presence and verified replacement) |
| `version.go` | `AppInfo` (version, commit, startup error for the banner) |
| `refresh_app.go` | `RefreshLeague` (pull the league from MFL into the mirror) |
| `crosswalk_app.go` | `LoadCrosswalk`, `GetCrosswalkReport` (the player directory and its match rates) |
| `signals_app.go` | `LoadSignals`, `GetSignals` (load every due signal file; source health, freshness and coverage) |
| `m1_app.go`, `model_app.go`, `m1_player_score_app.go` | `ScoreLeague` (board run and model run), `GetRankings` (with the measurables), `GetPlayerScore` |
| `m2_app.go` | `GetPowerRankings` |
| `leagueschedule_app.go` | `GetLeagueSchedule` |
| `m4_app.go` | `GetFranchises`, `GetRoster`, `GetFreeAgentPool`, `GetLegalOps` |
| `transactions_app.go` | `ExecuteTransaction`, `PreviewTransaction`, `GetCurrentPhase` |
| `transactions_feed_app.go`, `transactions_calendar_app.go` | `GetFeed`, `GetCalendarEvents` |
| `rulebook_app.go` | `GetLeagueSetting`, `SetLeagueSettingOverride` |
| `admin_app.go` | `GetParams`, `SetParam` (the Engine Admin console) |

Every method that takes frontend input validates it before acting. Bindings in
`frontend/wailsjs/` are generated (`wails generate module`); never hand-edit them.

## Data and logs

- **Databases,** in `~/.config/TheWarRoom/`, split by lifecycle:
  - `thewarroom.db`: the MFL mirror and app settings. Rebuildable from MFL.
  - `history.db`: the fetch archive, observations and scoring runs. Not rebuildable; back it up.
  - `whatif.db`: the what-if league the transaction screens work on (R2). Deleting it reseeds it
    from the mirror at the next launch.
  - A dev build (no version stamp) uses `thewarroom-dev.db`, `history-dev.db` and
    `whatif-dev.db`, so development never touches the real league. One instance at a time
    (`.lock` file).
- **Season:** read from MFL (`league.Discover`: the newest year in the league's history), held in
  the mirror. Nothing in the code names a year.
- **Refresh:** after the window opens, and from "Refresh from MFL" in the rail. An empty mirror is
  filled during startup.
- **Migrations:** `internal/store/state/migrations.go`, forward-only and versioned; a
  `VACUUM INTO` backup is taken before any pending migration runs.
- **Logs:** one file per launch in `~/.config/TheWarRoom/logs/`, also on stderr. Startup logs
  each step and its time; a failed startup shows a banner in the app.
- **Headless check:** `thewarroom -probe` runs the full startup without a window and exits
  non-zero on failure. `-version` prints the build label. Any other argument exits 2, so a
  mistyped flag never opens a window on the real database.

## External services

- Every outbound request goes through `internal/archive`, so every body received is kept in
  `history.db`. A URL with no row in `sources.csv` is logged as `unregistered`.
- **MFL API** (league 14432), outbound only, through `internal/mfl`. The league host is
  discovered at runtime. The players endpoint is limited to once a day.
- **nflverse and DynastyProcess** (static CSVs on GitHub), **CFBD** (bearer token in
  `CFBD_API_KEY`). EA Madden was removed from the prior (R8); its `sources.csv` row is retired so
  its archived fetches stay labelled. See `docs/sources/Approved_Sources.md`.

## Scheduled for replacement

These work and are tested, but the core plan rewrites or deletes them. Do not extend them.

- `internal/scouting/assembly`, `m1_scouting.go` and the CSV/CFBD fetchers (`pfrcoverage`, `ras`,
  `schooltier`): today's board still reads them. The college share and breakout age already read
  the college seasons `college` stores in history, not CFBD live. They go when their consumer,
  today's engine, does.

## What does not exist, on purpose

- No ORM: parameterized `database/sql` only, inside the data layer.
- No DI framework: constructor injection from `app.go`.
- No package-level mutable state (`gochecknoglobals`).
- No cgo SQLite: `modernc.org/sqlite`.
- No `interface{}`/`any` in exported signatures (`ifaceguard`).
- No logging framework: the standard library.
- No inbound HTTP server.

The target UI data contract and fixture/live providers live in `frontend/src/app/data/`;
`components/` remains the current harness. `snapshot.NewSource` composes the initialized mirror and rulebook; the directory carries its own fetch provenance. Money in the snapshot is exact cents; absent held fields are omitted.

The target shell lives in `frontend/src/app/shell/`; its six-node table and hash routes are data.
`frontend/src/app/commands/` owns the immutable verb registry, sole dispatch path, keyboard
bindings and in-memory recent-command log. `Act` is the only clickable primitive in the target
UI. `frontend/src/TargetMount.tsx` switches between the target shell and the unchanged harness.

`docs/ui/registry/normalize.py` normalizes and validates `docs/ui/endpoint-registry.csv`.
`frontend/scripts/gen-endpoints.mjs` validates the shared owner/commissioner CSV and generates
`endpoints.gen.ts` (owner rows only) and `commissioner.gen.ts` (CM-001..CM-056 with owner `governs`
edges). The commissioner module is not imported by the app entry graph; future commissioner screens
must load it lazily. Both maps are frozen and freshness-tested;
`frontend/src/app/registry/` owns resolution, indexed placements, compact not-wired rows and the
pure two-route gate. `endpoint.open` navigates and highlights; merged rows target kept ids or places.
The harness is lazy-loaded only on `harness.open`; it is not in the target entry chunk.

`internal/ingestion/nflschedule` fetches and validates MFL's league-independent NFL schedule (with home side and scores) and the season's bye weeks (`byes.go`),
then converts it into dated `internal/leagueweek` facts at a supplied instant.
`internal/leagueweek` derives the lineup week, per-team kickoff locks and bounded season phase;
after the season ends it requires explicit rollover approval.
`internal/leagueclock` reads the held phase, commissioner calendar and optional lineup week/first lock:
deadlines soonest first with U0–U3 and the pin/promote ordering rule; league windows stay explicitly
unknown.
`internal/envelope` owns immutable move states (one transition map), checks, observation predicates
and per-feed observations, including unmapped IR drafts; `internal/store/moves` owns append-only
SQLite persistence. Neither core does I/O.
`snapshot.BuildClock` is the one clock builder: `TargetClock` (in `target_app.go`) passes the what-if store, and
`cmd/fixtures -whatif` passes a backup of it. Move fixtures stay outside the entry chunk.

`frontend/src/app/clock/` renders the league clock; one DOM ticker owns countdowns and urgency,
independently of React. Clock and snapshot providers load in parallel, never falling back live to fixtures.

`frontend/src/app/shell/MFLConnection.tsx` is the lazy Control Room › App keyring surface.
`mflKeyField.ts` owns only the mounted uncontrolled input ref; no command arg or state holds a key.
`frontend/src/app/commands/mflKey.ts` holds status independently of shell renders; the registry
alone writes it and runs connect/forget. The provider lazily imports generated Wails bindings;
`parseMFLKeyStatus` validates their five-state JSON before any status reaches the surface.

`internal/leaguefeed` holds typed season transactions, assets, lineups, pending trades, weekly results and playoff brackets; pure leaf.
`internal/ingestion/transactions` validates and converts the unkeyed MFL season transaction feed.
`internal/ingestion/livescoring` validates and converts MFL saved weekly lineups and live totals.

`internal/lineup` parses MFL starter bounds and checks full/partial saved lineups without I/O.
`lineup_app.go`: `TargetLineup` reads held starters, active-roster bench and rulebook legality; no network.

HQ › Roster: `LineupRoster.tsx` shows starters, bench and Go legality; `lineups.ts` caches and refreshes.

`lineupset_app.go`: `TargetCheckLineup(franchiseID string, starters []string) (lineup.Result, error)` checks locally.
`lineupset_app.go`: `TargetDraftLineup(franchiseID string, starters []string) (envelope.Receipt, error)` saves a plan.
`internal/envelope`: `lineup.set` checks active starters and saved baseline; watches explicit-week MFL lineups.

HQ lineup editing: `LineupEditor.tsx` checks starter toggles through Go and shows saved Start/Bench plans.
Hand-off Act: `move.handoff` opens a Ready plan; `target:moves` reloads receipts and the held HQ lineup.

`trades_app.go`: `TargetTrades(franchiseID string) (TradeReading, error)` reads held offers locally.
`trades_app.go`: `TargetDraftTradeAccept(franchiseID, tradeID string) (envelope.Receipt, error)` saves an accept plan.
`internal/envelope`: `trade.accept` watches post-hand-off TRADE asset sets; disappearance waits in DOTReview.

Trade desk: `shell/TradeDesk.tsx` groups held offers; lazy `data/tradeProvider.ts` parses Go readings.
Accept plan: `commands/tradeAccept.ts` drafts `trade.accept`; `shell/TradePlan.tsx` tracks desk hand-off.

`alerts_app.go`, `bye_alerts_app.go`: `TargetAlerts(franchiseID string) (AlertReading, error)` reads
lineup, incoming-offer and current/next-week position-minimum bye alerts from held inputs only.
`internal/leaguefeed`, `internal/ingestion/livescoring`: matchups, playing counts and per-player live scores.

Home seasonal card: `shell/SeasonalCard.tsx` lazily renders the held clock and shared deadline ticker.
Home alert tray: `shell/AlertTray.tsx` lazily reads ordered franchise alerts and routes to their workspaces.

`target_app.go`: `TargetDraftTaxi(franchiseID, playerID string) (envelope.Receipt, error)` saves a taxi plan.
`internal/envelope`: `roster.taxi` derives direction from held status and watches post-hand-off rosters.
IR and taxi open MFL owner pages without submitting; unavailable held taxi rules are labelled unchecked.
Inspector: `shell/PlayerAct.tsx` drafts IR/taxi by held roster status and hands off inline.

`frontend/src/app/market/` owns lazy historical Market screens and asset Inspector content.
`provider.ts` validates the Reading shell and typed sections; `state.ts` re-queries Go on filters,
retains prior readings until replacements arrive, and rejects stale responses. `ReadingView.tsx`
shares range geometry and the counting disclosure. The `market` command owns filters/inspection;
`endpoint.open` and War Room › Market are the two entry routes. `data/parse.ts` exposes the shared
freshness decoder used by both live and archive provenance. Player Inspector code is unchanged.

`internal/lab/draft.go` and `draft_position.go` compute slot-relative draft returns, season cells,
monotone value curves, plays-like picks, stretch-matched starter rates, known year-2 retention
and same-season draft-or-buy comparisons. `draft_app.go` validates `RawDraftScope` at IPC and
returns `Reading[MarketSurface]` from `TargetDraftLab`; no live ownership or player forecast.
`frontend/src/app/market/Draft.tsx` is a second lazy chunk reached from Market's Draft lab door
or L-15–L-22. `draftState.ts` retains prior scopes during reads and rejects stale replies; pick
assets reuse the Inspector for historical round cohorts. Draft goldens additionally stamp
`draftlab.js`; Node remains offline reference tooling only.

`offers_app.go`: `TargetOffers` validates season/direction/window/team scope and returns Go-authored owner disclosure, counts, eight stretch outcomes and acceptance comparisons in `Reading[OffersView]`. `internal/lab/offers.go` ports the reference offerStretches with exhaustive offline scope goldens; lazy `frontend/src/app/offers` lives above the pending trade desk, independent of the selected live franchise.

Bid plans (War Room › Valuations and pool): `frontend/src/app/pool/` is one lazy chunk reached from the
Market component. `state.ts` re-queries `TargetBidPreview` after each form change and keeps the last answer
on screen until the next lands; the `market` command carries the `pool-*` fields (no new verb). My moves
and the free-agent Inspector (`pool:<id>` asset) load `BidPlans`/`PoolInspector` from the same chunk, so the
entry bundle holds none of it. `wire.testdata.json` is generated by `TestPoolWireFixtureMatchesGo`
(`TWR_WRITE_WIRE=1`) and the frontend parsers are tested against it.
`internal/envelope`: `bid.submit` plans use a `manual` target (no URL; the owner posts on ProBoards), the
reserved `bid_pending` state, and the terminal `awarded_off_app` state for a bid that went elsewhere.

Cut/buyout plans: `release_app.go` drafts from persisted what-if moves; `release_rebase.go`
re-prices through the coordinator on isolated whatif.db generations and invalidates against
scoped mirror revisions. `internal/envelope/release.go` owns named checks, irreversible cut
confirmation and roster-export landing. `internal/store/moves/releasefacts.go` joins facts and
append-only envelope history atomically, enforcing one open release per player. The lazy
Contracts/My moves release cards share `RailStrip`; no mirror write or MFL submission.

Contract Act plans: `contract_acts_app.go` exposes validated preview/draft/read and manual hand-off, posted, withdraw and rebase commands. `internal/engine/contractacts` owns pure §7/§9–11 price and term previews. `internal/envelope/contract.go` checks missing rules/evidence and declares the Contracts observation source. `internal/store/moves/contractfacts.go` stores priced baselines with envelope history atomically. Annual/expiration exports do not prove inferred future cells. Contract windows reuse coordinator LegalOps; RFA tender/re-sign are offseason only. Unknown history is noted for commissioner checking. Team projections install priced cells through the coordinator on isolated simulation generations and reuse the What-if result builder. Lazy menus and cards live in `frontend/src/app/contracts/ContractActs.tsx`.

`frontend/src/app/shell/workScroll.ts` confines card reveals to `.wbody`, opens enclosing disclosures
and focuses endpoint sections, and holds table rows through asynchronous panel changes.
Draft Lab return maps lead with all-position summaries and closed, counted position breakdowns.
`counter_filters.go` owns display treatment for filter-fixed posture shares and excluded tiers
in both Counterparties and its team Inspector.

`internal/ingestion/draftreports` validates MFL draftResults and futureDraftPicks into typed held
picks (string PlayerID, epoch instants) and ownership; one adapter family, two export fetchers.
`draft_reports_app.go` refreshes both with archive fallback and reads local model values without
network; `draft_reports_reading.go` authors nine report tables, grid, timing and value-distance bars.
`frontend/src/app/draftreports/` is lazy inside Draft room alongside the unchanged Draft lab;
`draft.reports` refreshes or opens the discovered MFL room without submitting a pick.

League Pulse › Race: `race_app.go` holds weeklyResults, the playoff brackets, MFL's standings and both M2 boards, refreshed with the hourly season refresh or on open (`TargetRaceRefresh`). `TargetRace(week, team, mine)` reads held data only and validates the choices; `race_reading.go`, `race_playoffs_reading.go` and `race_points_reading.go` write every report (M-002 standings on top, then M-008/C-02, M-090, M-170, M-074, M-014, M-161, M-071, M-076, M-077, M-011–M-013, M-009, M-010) through the shared `Report` shell. The lazy screen is `frontend/src/app/race/Race.tsx`. The league's divisions, conferences and standingsSort arrive with the rulebook (`league.RawConfig`).

League Pulse › Now: `now_app.go` holds each opened week's NFL schedule, the season's bye weeks and, per week, the first held copy of its scores (read from the fetch archive, newer copies only). `TargetNowRefresh(week, force)` refreshes the race's feeds, then that week's NFL schedule and the byes (5-minute freshness, archive fallback scoped by `W`); `TargetNow(week, reportWeek, mine)` reads held data only and validates the week (0: the current NFL week; reportWeek 0: the last scored week). `now_reading.go` (week choice, held live scores for a week MFL has not scored, X-03 Classic matchup, M-005, M-003, M-004), `now_recap_reading.go` (M-007, M-021, M-022, M-088) and `now_nfl_reading.go` (M-015, M-016, M-072, M-018) write the reports on the race's view. The lazy screen is `frontend/src/app/now/Now.tsx`.

### Data and sources

- `internal/ingestion/leaguearchive`: offline, prevalidated MFL file-tree import through the archive Sink.
- `internal/manifest`: pinned-input identity, body resolution and last-good snapshot standing; no I/O except the injected body reader.
- `league_archive_app.go`, `data_sources_app.go`, `data_sources_reading.go`: settings, scoped refresh, manifest publishing and Go-written Control Room reports.
- `internal/store/history/inputs.go`: archive locators, input manifests and append-only compile attempts (metadata, not fact tables); `snapshot_checks` keeps each attempt's check results beside it.
- `facts_app.go`: compiles the published manifest (archive pins plus the curated `note_readings.csv`, pinned by its own sha256) with `internal/facts`, runs the nine checks against the identity-pinned exceptions (`internal/lab/testdata/check-exceptions.csv`), and records the attempt; the same manifest, compiler and rules never compile twice, and a failure keeps the last good snapshot, stale. Runs after an archive import and at launch.
- `frontend/src/app/datasources`: lazy Control Room Data and sources, scope Acts, player crosswalk and engine parameter overrides.

## League communications

`comms_app.go` holds authenticated messageBoard/polls exports with archive fallback.
`TargetComms` reads existing season transactions, current mirror contracts and the held directory.
`comms_reading.go` writes sentences shared by C-05 and M-032 with Go-validated filters.
Drops use the last held weekly roster before the move, dated by the last NFL kickoff.
`TargetOpenComms` opens ProBoards or the discovered MFL board/polls pages only.
`internal/ingestion/comms.go` validates posts. Lazy `frontend/src/app/comms` serves
Comms and Home › Digests through the shared Report shell.

## Player photos (ring 3, p3k)

`internal/headshot` fetches a player's photo from ESPN's resizer (96×70), then Sleeper, keyed through the
player directory's `espn` and `sleeper` ids (`history.Store.SourceID`). It paces requests at four a second,
accepts only real PNG/JPEG bytes, treats 403/404 and known silhouettes as "no photo" (rechecked after seven
days), and caches files in `<data dir>/headshots/` under a 200 MB least-recently-used cap. Photos are personal
use only (spec §7): never bundled, never in git, never in `history.db`, so its HTTP client bypasses the
archiving transport on purpose. The webview loads `/headshot/<mfl id>` through the Wails asset handler
(`photos_app.go`); `TargetPhoto` gives a card Go's address and alt text, or none when photos are off
(`settings.json` `photosOff`, `SetPhotos`). `cards/PlayerPhoto.tsx` reserves the box and shows the NFL team
when there is no photo.

## Trade Acts and acquisition costing (ring 3, p3t)

Trade Acts and acquisition costing: `trade_builder_app.go` validates the IPC boundary and persists
local proposals in `internal/store/state/trade_plans.go` on whatif.db. `trade_projection.go` projects
two pinned what-if rosters through the coordinator, retaining exact cents, and rechecks hand-offs.
`internal/envelope/trade_acts.go` watches full asset sets; rejection/withdrawal disappearance stays
ambiguous, never Landed. `acquisitions_app.go` prices held incoming contracts or saved bid terms
and an optional unclaimed cut through the same coordinator. Lazy Trade desk and Contracts chunks
host `shell/TradeBuilder.tsx` and `contracts/Acquisitions.tsx`; all controls use the command registry.
Trade receipts carry Go-authored asset labels using the desk/builder pick formatter. Proposal, rejection
and withdrawal plans share local removal before hand-off; accept plans are unchanged. New pool bid forms
release the old table anchor, reveal within the work column and focus the salary field after mounting.

`internal/store/personal` owns watch, bait, dated plain-text notes and ordered draft entries in
 the app's main SQLite pool (the existing dev/prod split). `personal_app.go` exposes validated
`TargetPersonal`, `TargetPersonalChange` and read-only `TargetPersonalHandOff`; no MFL submission.
`internal/ingestion/shellfeeds.go` validates calendar, trade bait and player-score exports.
`roster_reports_app.go` exposes `TargetRosterReports` from held exports; `nfl_draft_app.go`
exposes `TargetNFLDraft` from validated detailed player draft selections; `internal/lineup/byes.go`
checks position-minimum bye shortfalls. `calendar_app.go` merges held MFL calendar events into C-04.
`shell_feeds_app.go` holds calendar and tradeBait, refreshed once per local day with archive fallback;
screen reads never fetch. Their fetchers live beside the parsers in `internal/ingestion/shellfeeds.go`.
`frontend/src/app/personal` is lazy list editing and persistence; `shell/StatusStrip.tsx` lazily
reads source health and held gameday counts via `TargetShellStatus`.

The snapshot's player directory now also includes named unrostered players for global search and
private draft/watch lists; eligibility remains roster-only. `history.ReferenceBodies` admits imported
league-archive responses on read-only surfaces; live fallbacks still exclude imports.

## Player research

`internal/ingestion/researchfeeds` validates display-only MFL playerScores (week and YTD,
including team defenses), the five MFL-wide activity exports, pointsAllowed and playerProfile.
It reads the newest nflverse depth-chart snapshot using `contracts.ReadRows` (shared Parquet
reader), with the captured latest CSV as the fixture oracle. None of these readings write
model observations or scouting inputs.
`research_app.go` holds these feeds with archive fallback, resolves depth-chart identities
through held gsis/ESPN crosswalk links, and caches rostered-player profiles per local day with
one-second spacing. `TargetResearch`, `TargetResearchRefresh`, `TargetResearchNewsRefresh`,
`TargetResearchPlayer` and `TargetOpenResearchArticle` validate the IPC surface; article links
use the discovered MFL host and require an article held for that player.
`research_reading.go` and `research_context_reading.go` author all Research reports through the
shared Report shell. `frontend/src/app/research` is lazy, with one screen for research, a news
section on Home › Digests, and a read-only player Inspector for directory search results.
M-050 is explicitly ring 4: weekly NFL team box-score totals require an nflverse team-stat
adapter; fantasy points and NFL game scores are not those statistics. M-055 resolves to the
existing Race lineup detail's best-lineup reading.

`engine_refresh.go` scores after the background launch/hourly refresh has loaded signals, league
points and the held free-agent list. A shared mutex excludes the manual path; changed-value loads,
league/list changes or a 24-hour-old run make it due. `target:season` refreshes open readers.
C-07 lists the engine run and its age; completed background checks append their run timestamp
without changing any scoring formula or using ownership to find points.

League Pulse › Archive: `archive_app.go` rebuilds only the checked archive facts and caches them;
`archive_reading.go` and `archive_records_reading.go` author historical reports. The lazy
`frontend/src/app/archive/Archive.tsx` has no MFL Pools report. MFL pool exports remain accepted
by the offline importer and preserved in the raw archive; native pools and mini games are deferred.
The shared shell route resolver reads legacy `pools-archive` links as `archive`.

`finished_scores.go` re-reads only the last finished week and YTD after the held NFL schedule
first proves completion, preserving its completion time across hourly passes. With no final flag,
four hours from kickoff is the allowance. Player research prices unclaimed cuts through
`transactions.CutCharge`, the coordinator’s existing rule, and names the last three finished weeks.
Refresh completion emits `target:season` even after failed or unchanged source reads, so the strip
and mounted research Inspector re-read held facts. M-025 is retired and has no route.

Bid conditional drops: `bids_conditional.go` prices held roster choices through `transactions.CutCharge`,
shared with the research Inspector. `bid_conditions` versions the named drop and its post label atomically
with envelope history; no change after hand-off. The lazy bid form reveals the choice only over cap room.

Offers history freshness: `offers_live_app.go` merges the archive owner's proposals with persisted
`personal_offers` and `personal_offer_checks` in the existing GM store (`internal/store/personal`).
Successful pendingTrades refreshes capture proposal/first/last-seen times; held completed transactions
prove acceptance inside the proposal-to-first-miss interval. Other disappearances remain unknown.
`internal/labcompile.OfferTiming` shares the archive's Eastern-date season/stretch rules. Open offers
are excluded from ending rates. The lazy Offers surface reloads its selected scope on season events.

## Native asset corrections

`internal/commissioner/native_projector.go` replays active contract corrections, season-scoped cap
adjustments and player cap relief in ledger order over cloned seed assets. Roster, reserve, cap and
directory readings share this projection. `native_override.go` validates CM-034 (G2) and CM-038 (G3)
commands on the existing commissioner append path, requiring a public reason and co-commissioner
permission; compensation uses the same replay. `PendingTradeCapCheck` is the future trade-escalation
seam, currently reporting no pending trades. Reserve exceptions match only their player and bucket,
expire at the supplied instant and explain the commissioner reason; no MFL or what-if state changes.

## Native fantasy scoring

`internal/fantasyscore` is a pure leaf: exact ten-thousandths, commissioner catalogue values,
true-position tackle/assist rates, per-play stacking bonuses, explicit unused stats and Needs a ruling.
It imports only the standard library and `internal/leaguerules`; no store, ingestion, I/O or scouting.
`StatLine` records canonical tackles/assists once; positional catalogue entries are rates, not extra stats.
`internal/fantasyscore/source` validates held nflverse weekly CSV and play-by-play, retains exact
long-play distances and return-touchdown credits (returning team only), and adapts them without fetching or store writes.
`source/testdata/comparison.csv` pins two complete held Legacy roster weeks against MFL totals,
with an independent native-points reference and every mismatch classified; the Legacy preset follows the held league’s MFL scoring settings, with optional per-stat rounding
and once-per-week counting. Eight source-credit differences remain explicitly unresolved. Neither package is connected to runtime score surfaces yet.

## Native waiver wire

`internal/commissioner/waiver_command.go` validates CM-035 commands and owner cut/claim arguments;
`waiver_execute.go` computes decisions under the history append lock, using held player facts and the
resolved roster, cap, contract and CM-012 rules. `waiver_legality.go` prices each remaining flat paid
season at the configured release rate and money increment, and rechecks claim eligibility at processing.
`waiver_projector.go` replays frozen cuts, private claims, orders and awards together with native asset
corrections; compensations cannot strand dependent awards. `waiver_reading.go` gives owners their own
claims, localized closes and priority provenance. Commissioner ledger claims remain hidden until processing.
`internal/store/history/native_waivers.go` owns the `StandingsOrder(ctx, league, season, at)` typed seam:
native recorded priority in `RawNativeSnapshot.StandingsOrder`, never imported or inferred standings.
Without that input, automatic order selection explicitly needs the unfinished native standings source.
Commissioner manual order is separate and can supply priority without inventing standings.
`ExecuteCommissionerWithFacts` retains the existing authorized, versioned, idempotent append transaction;
its server-computed decisions are not accepted from IPC. `ProcessDueWaivers` submits only closed-window
processing through that same path. The cancellable `nativeWaiverClock` runs with the app's managed
background workers and resumes held windows after restart; no network feed is involved.
`native_waivers_app.go` exposes `TargetNativeWaiverAction(RawCommand)` and
`TargetNativeWaivers(league, franchise, at)`. Neither port migrates the imported owner screens.
