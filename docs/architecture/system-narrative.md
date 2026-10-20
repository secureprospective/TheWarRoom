# System narrative

The long account of what exists in TheWarRoom: layers, IPC surface, data, external services and the
native league. Moved from the root `SYSTEM_MAP.md` on 2026-10-20; that file is now a one-page feature
directory and `docs/architecture/placement.tsv` assigns every root file and package to a feature.
Update this narrative in the same commit as any new package, IPC method or external service. Current as of 2026-10-19: ring 4 merged (`main`
5739c5c9), then the clean sweep checked every path, symbol and IPC method named here against the
code.

## Layers and packages

Import rules marked **(depguard)** are build errors in `.golangci.yml`, not conventions.

| Layer | Package | Job |
|---|---|---|
| App | repo root (`package main`) | Wails entrypoint (`main.go`), the `App` composition root (`app.go`) and the IPC adapters (`*_app.go`). Adapters validate, route and format; no business logic, no SQL. |
| Composition | `internal/commissioner` | Typed app-native command validation, session/authorization ports, receipts, compensation replay and privacy-filtered ledger/people readings. Imports the pure governance policy; the production authorizer replays the transaction's existing records and uses census permissions, without I/O or re-entry. `history.ExecuteCommissioner` trial-folds the proposed change and appends once in `history.db`'s `league_event`; no what-if or MFL writes. Creation takes `{name, preset: legacy_nfl|empty, franchiseCount}`; Go expands `franchise.01`..`NN` (named "Franchise N") and, for Legacy NFL, the whole typed preset seed into the durable record, so replay never depends on later preset code; the creator becomes commissioner. `commish.rules.set` changes one `internal/leaguerules` catalogue field: `rules.go` maps every field to exactly one census row (authorization and gravity; apply-now on a competitive field is G3), `rules_season.go` resolves rules by season and instant; `next_rollover` changes stay pending until the recorded rollover instant arrives, and ended-season reads ignore later compensation, and the trial resolve refuses incoherent rule sets before append. `rules_reading.go` is the public sectioned rules reading. CM-018..021 people changes, rule changes and correction reversal share the one append path; reversal also requires the target's original permission. Root `commissioner_app.go` exposes execute/ledger/people/rules/native-leagues IPC, using only the injected local session. Rule rows also carry the in-force value as plain JSON (`leaguerules.Current`), a percent unit and the safeguard Go applies for each timing; the people reading says what this principal's role may start (`yourRole`, `mayChangeRules`, `mayGrantRoles`); a ledger entry says where this principal may undo it (`undo`: league or private). The frontend lives in `frontend/src/app/commissioner/`, all lazy under Control Room › League: one shared state (`proxy.ts`, `leagueState`) holds the league picker, the ledger/rules/people readings and the outcome of the last form, and reloads all three on a receipt; typed parsers `rules.ts`, `people.ts`; screens `LeagueRules`, `LeaguePeople`, `LeagueLedger` (with `UndoEntry`) and `LeaguePicker` and the tray forms `StartLeagueForm`, `RuleChangeForm`, `RoleForm`. Forms only build the typed body and send it through the `commissioner.execute` verb, with a request key that repeats on a retry of the same submit (`ChangeBar`, `forms.ts`); G2 confirms by `HoldToFire` (a self-contained hold timer, the one reviewed keyboard exception besides the command bar). Rule values are edited through `ruleDraft.ts` (drafts to and from Go's plain JSON) and `RuleInput.tsx`; list and map add/remove go through the `commissioner.draft` verb. Go stays the authority. Succession forms and ballots are not built. CM-022 succession, inactivity, elections and vote commands (`succession.go`) use recorded ledger times and the same append path: CM-022 is a composite census row, so each command names its gate (commissioner only for the inactivity and voting periods and the activity check-in; co-commissioner or DOT to open votes; a vote's target may not close it) and the governance trial fold decides the rest. Every recorded change counts as its author's activity; an inactivity takeover waits out the longer of the two periods while season dates need a ruling. People readings show public vote counts and succession status, with voter participation private to C/CC; ballot ledger entries are private, tie rulings public. |
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
| Layer 1 | `internal/ingestion/players`, `rosters`, `leagueschedule`, `leaguestandings` | MFL's league-scoped player database (it includes commissioner-created players; once a day), every franchise's roster (salary and contract status left raw for normalize), the league's own weekly matchups and the standings, all as raw records. |
| Layer 1 | `internal/ingestion/salaryadjustments` | MFL's dead-cap ledger. An empty reply is valid (no dead cap), so MFL's error envelope is checked first. |
| Layer 1 | `internal/ingestion/playerscores` | Season fantasy-point totals for the last completed season as a history batch (`outcome.fantasy_points`), the M1 base-points proxy. |
| Layer 1 | `internal/ingestion/sitenews` | MFL's site news (What's New) RSS from the api host. |
| Layer 1 | `internal/ingestion/crosswalk` | MFL id → nflverse gsis from DynastyProcess `db_playerids.csv`, with a pfr index for PFR sources. One MFL id with two gsis fails the fetch. |
| Layer 1 | `internal/normalize` | Raw records → domain types: the players lookup and roster join. |
| Leaf | `internal/domain`, `internal/playerid`, `internal/numeric` | Value types. `playerid.New` is the only way to build a `PlayerID`. |
| Leaf | `internal/measures` | The measure registry: `measures.csv`, `sources.csv`, `source_fields.csv` and `feeds.csv`, embedded and validated. Generates `docs/data-layer/Measure_Dictionary.md` (`make measure-dictionary`). Adding a source is a CSV row, not code. |
| Leaf | `internal/nativecheck` | Pure native active-roster limits, per-position/group lineup checks, invalid-lineup penalty, missed flag, IR/taxi verdicts and salary-cap counting. Cap counting applies season rules, reserve shares and signed recorded charges with the league's money increment; `CapAllows` judges a net counted move delta (including conditional drops), refusing hard-cap overages and flagging soft-cap overages with a plain-word deadline in a supplied viewer zone or Needs a ruling. `NewWaiverPolicy` consumes resolved CM-012 settings without inventing missing values; `Close` dates an unprocessed window from its original cut and `Award` chooses among already-eligible franchises using a supplied priority-first standings or rolling order (rolling moves the winner last). The native commissioner ledger consumes this policy for cuts, private claims and atomic awards; no imported league or hand-off state is changed. `NewBidPolicy` does the same for CM-013 free-agent bids (points, limits, steps, phase rules, tie-aware winner). Rules and held football facts arrive as values; no MFL policy, I/O or stores. |
| Leaf | `internal/leaguerules` | Typed commissioner-owned season rules, explicit unset values, cited Legacy NFL preset, pure ordered change resolver and preview diffs. `catalogue.go` exports the read-only field catalogue (label, section, value schema, competitive), plain-JSON `DecodeSetting`, whole-value stored encoding for durable seeds, and plain-words rendering. Imports only the standard library and pure domain value types; no store, transport, ingestion or I/O. |
| Store | `internal/db` | SQLite pools: one write connection, many read-only ones, one WAL file. `Backups` owns matched three-file manifests, fixed-order writer reservation for daily capture, startup pre-migration capture, offline journaled restore/recovery and verified-set retention. Call recovery before opening application pools; restore requires offline ownership of the data-directory instance lock. |
| Store | `internal/store/rulebook` | League rules from MFL as immutable versions with one active pointer, plus commissioner overrides. |
| Store | `internal/store/params` | Engine calibration: shipped defaults plus admin overrides, league-wide or per position. The Layer 4 settings are seeded from `l4.Defaults`; the model's fitted values from the embedded `fitted.json`, marked calibrated. Shipped defaults are upserted at start-up; overrides are never touched. A set splits into the board's part and the model's (`model.*`, `dynasty.*`); each run records only its own. |
| Store | `internal/store/moves` | Append-only envelope specs and replayable audit entries in `thewarroom.db`; restart-safe move receipts and awaiting evidence queries. `bid_facts` holds current bid figures; `bid_conditions` keeps the conditional drop by audit version, keyed by an envelope's correlation id: the owner's own bid figures, no franchise column, frozen by a trigger once the bid is handed off. |
| Store | `internal/store/state` | Two things behind one `Reader`. **`Mirror`**: the league as MFL states it (season, rosters with contracts, salary adjustments), replaced whole by a refresh; every score surface reads it. **`Store`**: the what-if league in `whatif.db`, seeded from the mirror (rosters and MFL salary adjustments, so its cap starts equal): rosters, contracts, the contract-year ledger, dead cap, cap relief, phases, feed, calendar. Append-only ledgers; the transaction coordinator holds the only `Writer`. |
| Store | `internal/store/history` | `history.db`: everything the app cannot rebuild. The fetch archive (`raw_archive`, `fetch_log`), facts per measure appended on change (`observations`, read as of a date through `Features`), source health, and model runs with the param set, engine and inputs they used. Append-only, enforced by triggers. The `scoring_runs` and `run_scores` tables stay in the schema and in existing databases, holding the retired board runs; no code reads or writes them. |
| Analytics | `internal/facts` | Pure pinned-input compiler for season, franchise, standings, rules, calendar, player identity/history/conflicts, source provenance, loader notes, curated trade-note readings, weekly games and lineups, player weeks, injuries, weekly rosters, NFL games, points allowed, playoffs, contracts, cap charges, the draft, picks owned, transactions and their assets, trade bait and MFL-wide ADP/AAV: every Lab relation except the checks; typed in-memory columns only. `Facts.Views()` computes the Lab's named readings (played_game, regular_week, all_play_week, team_season, trade_asset, offer, roster_move) from the columns on demand. `internal/facts/parity_test.go` compares the independent Python golden in verify; the full archive is opt-in. No snapshot publication or database tables. |
| Analytics | `internal/labcompile` | Pure facts-to-Lab compiler: calendar, production, contracts, trades/draft, directories, ledger/source proofs and full wins/replacement, five-season forecast, pick curves, market and rolling backtest analysis. `Build` produces the complete document and an immutable `Model` whose `Player`/`Pick` API uses the trade-leg valuer at a supplied moment, with explicit unvalued reasons. Passive Python kernel/method/analysis goldens and reduced conservative smoke run in verify; opt-in full archive parity covers all 18 keys and player API/leg equality, while recorded-analysis document parity remains isolated. The app fits the model after launch (`fitAssetModel`, after `warmArchive`) and C-01 reads `Model.Player` through `labAssets`; the Lab screens still read the frozen document. |
| Analytics | `internal/lab` | Frozen fingerprint-checked compiled Lab facts, typed columnar decoder, pure Market fitting, delivery, calendar, Draft slot returns, team-side Counterparties profiles and owner-scoped offer endings. Offline JS goldens in `tools/labparity` pin the reference engine and data hashes; Node is never required at runtime or by verify. Exact missing-input exceptions live in `testdata/missing-inputs.csv`; `facts-reference.py` independently rebuilds the Python facts oracle in a disposable directory, with its source-row proof and provenance under `internal/lab/testdata/facts`. The reduced three-season golden retains every reference relation; full parity is opt-in. Reference relations no screen reads (price list, contender trades, draft map, the evidence grade) live in `reference_test.go`. |
| Policy | `internal/governance` | Pure typed-event fold for principals, seats, private invites, abilities, role permissions, DOT recusal and succession; standard library only, no I/O or stores. Census permission data, a single co-commissioner seat and explicit activity/ballot evidence; locked all-franchise election majority, timed recorded runoff closure and candidate-recused tie rulings. |
| Engine | `internal/engine` | L3 age decay (`ApplyDecay`, `PeakLimit`), which `cmd/fit` reads. The L1–L6 pipeline and its types were removed (`docs/architecture/removed-code.md`). |
| Engine | `internal/engine/rookiedraft` | Pure native rookie-draft setup, slot contracts, order/ownership planning, selection preview and fixed-instant clock replay. `FinalStandings.FirstRound` is the typed worst-first standings seam; `MissingFinalStandings` refuses absent results with "The draft order needs the final standings, which are not built yet." Unset later patterns and tie rules never default. Rules, owned picks, accepted selections (each carrying the contract fixed when it was made) and native roster/cap facts arrive as values; a pick is refused only for pushing a roster over a limit, never for a roster still short of its minimum. The ledger, calendar and gate wiring is in `internal/commissioner` (below). |
| Engine | `internal/engine/bid` | The free-agent auction's arithmetic and rules (rulebook §6) as pure functions: bid points in exact thousandths, the minimum salary by experience (`freeagency` reads the same table), eleven named rule checks with Go-written reasons (including cap coverage by one conditional drop), and the ProBoards post text. |
| Engine | `internal/engine/race` | The league race as pure functions over weekly results: records, all-play, streaks, games back, MFL's standingsSort applied step by step with a trace, playoff seeding (division leaders first), the opening round and the bubble. Proved against the real 2025 and 2026 seasons in `race_test.go`. |
| Engine | `internal/engine/eligibility` | MFL's IR and taxi rules as pure verdicts: per player and direction, eligible / ineligible (reason) / unknown (reason), from held feeds only (injuries status, players draft year, roster status, league capacities). The captured 2026-10-10 owner pages are its committed test oracle (`testdata/answer-key.csv`). |
| Engine | `internal/model` | The measurable's model: a player's league-points percentile blended with a prior from pre-NFL facts, the talent and survival arcs, and `Params`, the fitted values per position (`model.*@POS`). Pure. |
| Engine | `internal/model/fit` | Fits `model.Params` from stored seasons, scoring every choice on a holdout season. Pure; `cmd/fit` runs it. |
| Engine | `internal/engine/l4` | Layer 4's per-position settings table only. Its adjustable numbers are params (`l4.film.cap@WR` and so on), listed in the Data and sources parameter editor (R7) under areas marked "inert, no run reads it": the rubric math that read them was removed with the layer 1–4 engine. |
| Composition | `internal/app/scoring`, `internal/m2service`, `internal/powerrankings` | M2: counts each roster's model values (on-field-now or dynasty, league points per game) as the best legal lineup, the top N or the whole roster, and z-blends them: this season with the results (all-play when MFL reports it, otherwise points for) at 4 ÷ (4 + weeks played) or the slider's weight; the franchise with the roster's age. `m2service.Outlooks` adds the context columns (cap room, schedule luck, projected record), never in the score. |
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
seconds. A bid's late-bid mark is `nativecheck.BidPolicy`'s configured duration. The imported (MFL league) clock keeps Rulebook §14's week 9 first-kickoff deadline.
`internal/commissioner/calendar.go` folds public exception Acts (`commish.calendar.exception`, CM-026,
G3) and rule compensation into the reading. The existing history transaction authorizes, validates,
retries and appends the exception. `league_calendar_app.go` exposes `TargetLeagueCalendar` for an
explicit app league, season, franchise, action and owner time zone; it reads only durable local events.
This read port does not yet replace the what-if transaction phase gate or the imported owner screens.
`internal/commissioner/schedule_generator.go` generates complete division-weighted rounds for 2–32
franchises, with one rotating bye in odd leagues. `schedule.generate`, `schedule.edit` and
`schedule.publish` share the authorized/versioned/idempotent append path (CM-015, G2; published
week edits are G3). Generation takes explicit weeks with the current division alignment; an omitted
pattern uses the in-force rule, and an unset rule refuses to guess. Equal even-sized divisions use
complete within-division rounds; reciprocal meetings and oriented remaining trails balance home games. Each game has a stable week/slot id. Edits patch one game's home/away
franchises, validating the complete resulting week. Generated weeks are frozen in the ledger record;
publication replay never reruns the generator or resolves today's schedule rules. `native_league_schedule_app.go` exposes `TargetLeagueSchedule`: only the published
projection, replayed from local events and compensations. Draft changes never replace that reading.
`native_schedule_app.go` exposes the franchise season and current-week owner readings over those
publications. Opponents are native; kickoff and finality come only from locally archived NFL weeks,
localized for the owner, with missing dates explicit. Ended seasons stop at their rollover boundary;
new seasons do not inherit a publication. Until a native seed supplies a season, the first schedule
season is the creation record's year. No imported fantasy schedule or network fetch is used.
`internal/commissioner/windows.go` holds the named window doors, `calendar.window.set`, `.move` and
`.remove` (CM-023, G2, co-commissioner): each replays into the same `calendar.windows` value a
`rules.set` writes (`windowChange` feeds the season resolver's `seasonChanges`), so the two doors read one projection and a
reversal that would strand a later window change is refused. `gate.go` holds `NativeActionGate`, the one
decision every native owner action asks: it answers through `ActionCalendar.Decide` (exceptions and the
exact trade deadline included), says in the owner's local time which window opens a closed action, and
mints the `Permit` an action needs to run. Root `native_gate.go` is the registry and its only door,
`runNativeAction`; `native_gate_test.go` runs every registered action shut and open (an action may refuse
for want of its own details, but not for a closed gate). The registry (`nativeOwnerActions`) holds the lineup
(`lineup.set`), the rookie pick (`draft.pick`), the native cut and claim (`roster.cut`, `waivers.claim`) and
one `contract.<action>` start for each contract action an owner takes under a window (tag, extension,
restructure, buyout, RFA tender, re-sign, release), built from `nativecheck.ContractActions`. Each action
takes the owner's input and returns its receipt; a contract start only confirms the league has that action
on, through the same `nativecheck.Availability` the quote reads, and records nothing until choosing the
contract and paying its price are built. The waiver append transaction rechecks the gate and the owner's
seat, including callers below the IPC door. Now/Race integration and the remaining census owner-edge wiring
are still pending.

## IPC surface

| File | Methods |
|---|---|
| `league_calendar_app.go` | `TargetLeagueCalendar(RawLeagueCalendarScope)`: app-owned action windows, live close countdowns, exact chosen trade deadline and public scoped exceptions, replayed from the league ledger and localized for the owner. |
| `native_league_schedule_app.go` | `TargetLeagueSchedule(RawLeagueScheduleScope)`: published native head-to-head schedule only; draft changes are not owner readings. |
| `native_schedule_app.go` | `TargetNativeWeek(RawNativeWeekScope)`, `TargetNativeFranchiseSchedule(RawNativeScheduleScope)`: validated league/franchise/season/owner-zone scopes and explicit week reading time; published opponents, venue, bye, local first NFL kickoff and next week. Unknown held NFL dates never become guessed weeks. |
| `market_surfaces_app.go`, `market_profile_app.go` | `TargetMarketLists`, `TargetPickClock`, `TargetMarketStretches`, `TargetAssetProfile`: validated historical scopes and asset kinds; typed Go-computed display sections in `Reading[MarketSurface]`, fitted ranges and archive disclosure, no spread grade. |
| `counterparties_app.go` | `TargetCounterparties`, `TargetCounterpartTeam`: validated historical standing/window/posture scopes; tier cards, trends, pair matrix, sortable team values and season/asset-flow/partner Inspector readings. Stateless construction lives in `internal/app/counterparties`; the adapter supplies the shared Lab loader, build stamp and presentation vocabulary, and converts service values to the unchanged main wire types. Lazy frontend in `frontend/src/app/counterparties`, franchise ids resolved against the shared snapshot. |
| `lab_app.go` | `TargetMarket` validates season/phase input and returns a typed historical Market payload in the generic `Reading` shell extending `snapshot.Sourced`; dated provenance, count/threshold and Go-authored counting disclosure, no spread grade. |
| `whatif_app.go` | `TargetWhatIf` validates read/cut/buyout/undo/reset intents; per-franchise persistent baseline and ordered moves in `whatif.db`. Generation-scoped coordinator ledgers replay on mutation, publish after success, keep the mirror read-only, block stale edits until Reset. Lazy Contracts panel shows Go-authored before/after cents and rules. |
| `native_legality_app.go` | `TargetNativeCapCheck(league, viewerZone)`, `TargetNativeRosterCheck`, `TargetNativeReserveCheck`, `TargetNativeLineupCheck`, `TargetNativeHouseRules`, `TargetNativeContractQuote(RawContractQuote)` (one contract action priced on one recorded native contract at the app clock: allowed or refused with the plain reason, new salary and years, dead money by season, cap effect and how each number was reached; reads the app league ledger and held player positions only and records nothing): app-owned recorded rosters and season-effective rules, with held player positions, NFL designations and draft years. No refresh, MFL acceptance promise, score-feed wiring or owner-screen migration. |
| `native_seed_app.go` | `TargetNativeSeed(RawNativeSeedRequest)` records one held Legacy NFL starting snapshot in the commissioner event stream: rosters, contract terms, reserve status, franchises (with each division's conference, `heldConferences`), current/future picks and salary adjustments. Go captures the inputs without network or client-supplied assets; retries retain the original receipt after mirror refresh. `TargetNativeAssets` replays that app-owned snapshot, separate from MFL and what-if. `TargetNativeDirectory` reads the native franchise directory: recorded seed names/abbreviations, later active identity edits, current governance seats, and season-resolved alignment (held seed divisions carried into the shared rules resolver). `TargetCommissionerPeople` shares this directory projection; pending alignment changes do not apply, and compensation restores the recorded basis. No MFL directory or hand-off is changed. Empty leagues have no seed. |
| `contracts_app.go` | `TargetContracts` validates team/position/sort scopes and returns separate typed `Reading` shells for league contracts, my-franchise cap, literal MFL salary adjustments. `TargetPlayerContract` validates player IDs and returns observed current salary / explicitly flat-filled inferred future years. Both read a coherent `Mirror.ContractsView`, never the what-if ledger. Lazy frontend in `frontend/src/app/contracts`; every player Inspector includes contract years. |
| `market_feeds_app.go`, `market_feeds_reading.go`, `report_reading.go` | `TargetMarketReportsRefresh(force)` and `TargetMarketReports`: held week-scoped projections and daily MFL-wide market feeds, archive fallback; shared Go Report and lazy React ReportView used by draft and player-market reports; Go-written pool and Inspector market facts. |
| `pool_app.go` | `TargetPoolRefresh(force bool)` fetches MFL's free-agent list (skipped when fresh); `TargetPool(RawPoolScope)` reads the held list with no network: name, position, team, age, experience and the §6 minimum per row, filtered and sorted in Go, in a `Reading[Pool]` that is failed before the first fetch, empty (thin) when MFL lists nobody, stale when a refresh fails. |
| `bids_app.go`, `bids_plans_app.go` | Bid plans (intent `bid.submit`): `TargetBidPreview(RawBidInput)` checks typed figures against the rules without saving; `TargetBidDraft`, `TargetBidRevise`, `TargetBidHandOff` (records the copied ProBoards post; refused if the rules no longer pass), `TargetBidPosted` (the owner says it is up), `TargetBidWithdraw` (all nonterminal local plans; never removes a ProBoards post); `TargetBidPlans(franchiseID)` lists the owner's plans with Go's words, live rules and the actions on offer. The landing watcher looks for the player on the owner's roster with the planned contract. `TargetMoves` leaves bid plans out. |
| `moves_app.go` | `TargetHandOff(correlationID string) (envelope.Receipt, error)` saves before browser hand-off; `TargetWithdrawRoster(correlationID)` ends a local IR/taxi plan without changing MFL; `TargetCheckMoves() error` wakes landing checks; `target:moves` signals a hand-off or changed move. |
| `moves_watcher.go` | Cancellable landing watcher; shared 60-second wake/fast-mode rate limit and persisted restart recovery. |
| `livegame_app.go`, `livegame_spine.go` | `TargetLiveGame(open, follow, mine)`: demand-driven Now poll on held NFL windows; MFL liveScoring at 30 s, ESPN board at 30 s/details at 15 s with capped backoff; separate held detail time, strict ESPN crosswalk starter matching, cached personal-use photos. Source URLs are data in `livegame_sources.json`. Lazy `frontend/src/app/now/LiveGame.tsx`; `game.follow id=` command. |
| `season_app.go` | Held season feeds (transactions, lineups, pendingTrades, injuries); `TargetSeason` reads without network; `target:season` signals refreshes and key changes. Ready roster plans whose held eligibility turned ineligible return to Draft after a refresh (`revalidateReadyRosterMoves`, in `moves_app.go`). |
| `snapshot_app.go` | `TargetSnapshot` (initialized mirror and rulebook, cached live directory with archive fallback, held injuries feed fetched on first use; shared fixture/live contract; keeps the last good build); `TargetClock` (phase log and commissioner calendar from the what-if store; no network); `TargetDraftIR`, `TargetDraftTaxi`, `TargetMoves` (draft `roster.ir` / `roster.taxi` against the kept build into the persistent move store; no network) |
| `mflkey_app.go` | `MFLKeyStatus`, `SetMFLKey`, `DeleteMFLKey` (keyring presence and verified replacement) |
| `version.go` | The build stamp (`buildLabel`); no binding. Every `Reading` carries it as its build line, and `-version` prints it. |
| `refresh_app.go` | `refreshLeague` (pull the league from MFL into the mirror); no binding. It runs at startup on an empty mirror, at launch from `domReady`, and from `TargetDataRefresh("league")`. |
| `crosswalk_app.go` | `LoadCrosswalk`, `GetCrosswalkReport` (the player directory and its match rates) |
| `rankings_app.go` | `loadSeasonPoints` (MFL's season totals into history, for the engine refresh); no binding |
| `power_rankings_app.go` | `powerRankings`, behind the race readings; no binding |
| `transactions_app.go` | `TransactionRequest` and `buildRequest`, shared by the what-if ledger and release rebase; no binding |
| `admin_app.go` | `GetParams`, `SetParam`, `ResetParam` (the Engine Admin console; reset returns a parameter to its shipped default) |
| `control_room_app.go` | `TargetLeagueSettings` (M-091, M-097), `TargetAccounting` and `TargetOpenMFLAccounting` (M-098, M-197; the app only opens MFL's page), `TargetBylaws` (M-185, the league rulebook by section), `TargetSiteNews` (M-141, refreshed when an hour old) |
| `data_sources_app.go`, `league_archive_app.go` | `TargetDataSources` (what history holds about every source and the checked facts), `TargetDataRefresh(scope)` (renews one scope and says what it did), `TargetCrosswalk` (C-08 match rates), `SetLeagueArchiveDir(dir)` (the archive's `raw/` folder, one folder per season). `internal/app/sources` owns synchronous source-health, signal coverage/loading, crosswalk construction/linking, archive import, manifest publication and checked-facts compilation behind narrow history interfaces. Root adapters retain declared wire types, settings shared with photos, directory/roster snapshots, shared locks, held-result publication and background scheduling. |
| `archive_app.go`, `asset_board.go` | `TargetArchive(season, team, other)` (the league-history board), `TargetAssetBoard` (C-01 from held weekly scores and, once compiled, the Lab value model; never fetches) |
| `draft_reports_app.go` | `TargetDraftReports` (held exports and model scores only), `TargetDraftReportsRefresh(force)`, `TargetOpenMFLDraftRoom`, `TargetNFLDraft` (held detailed players only) |
| `comms_app.go`, `research_app.go` | `TargetCommsRefresh(force)`, `TargetResearchStatus` |
| `acquisitions_app.go`, `contract_acts_app.go`, `release_app.go`, `trade_builder_app.go` | `TargetAcquisition(RawAcquisition)`, `TargetContractActs(RawContractAct)`, `TargetReleasePlans(RawReleaseRequest)`, `TargetTradeBuilder(RawTradeBuilder)`: validated what-if readings in the `Reading` shell |
| `commissioner_app.go` | `TargetCommissionerLeagues`, `TargetCommissionerLedger(league)`, `TargetCommissionerRules(league)` (see the Composition row) |
| `windows_app.go` | `TargetSetWindow(kind, at)` sets one league window |

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
- **Database checks:** `internal/db/routine_checks.go` retains the last attempted integrity/FK
  checks per pool. `database_checks_app.go` runs them after the window opens and weekly while
  open, through a cancellable, joined worker. Data Sources exposes `databaseChecks` using
  the existing report renderer. Results are process-local; restarting runs checks again.
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

Nothing at present. The scouting fetchers listed here were removed with the layer 1–4 engine
on 2026-10-20.

## Built but not wired

Found by the 2026-10-19 sweep. Each works and is tested, but nothing in the shipped app reaches it.
Wire it or retire it when its ring comes up; do not count it as landed.

- **Native owner readings with no screen yet.** Go serves these over IPC and the frontend never calls
  them: `TargetNativeWeek`, `TargetNativeFranchiseSchedule`, `TargetLeagueCalendar`,
  `TargetLeagueSchedule`, `TargetNativeLineup`, `TargetNativeLineupSet`, `TargetNativeResults`,
  `TargetNativeStandings`, `TargetNativeScorePreview`, `TargetNativePlayerScore`, the
  `TargetNative*Check` legality readings, `TargetNativeContractQuote`, `TargetNativeBidAction`,
  `TargetNativeTradeAction`, `TargetNativeWaiverAction` and `TargetCorrections`. They are the owner
  side of the app-native league.
- **IPC kept only as test observation points:** `TargetSeason`, `TargetCheckMoves` and `TargetMarket`.
  No screen calls them; the Go tests read season, watcher and Lab market state through them.
- **IR and taxi entry checks:** `nativecheck.CanEnterIR` and `CanEnterTaxi` (with the CM-038
  exception parameter) have no production caller. The stay/must-leave verdicts are wired; placement
  onto IR or taxi in a native league is not.
- **Play-by-play scoring:** `fantasyscore/source` `ReadPlays` and `AttachPlays` attach yardage and
  touchdown distances. The golden test needs them to match MFL's scores; no production path reads
  plays, so whether distance bonuses reach native scores is unverified.
- **The `scoring_runs` / `run_scores` tables** stay in existing history databases, holding the
  retired board runs, unread. Their writer and readers were removed 2026-10-20
  (`docs/architecture/removed-code.md`); the schema is unchanged, so no migration was needed.
- **The `transaction_corrections` table** stays in existing state databases, empty and unread. Its
  writer was never reachable from the app and was removed; commissioner corrections live in
  `history.db`.

### Scoring orchestration

`internal/app/scoring` reads the current and previous model runs, builds the power boards,
refreshes season fantasy-point totals and executes synchronous model passes through explicit
store interfaces and readers. `power_rankings_app.go` retains the wire declarations and converts
service results without changing nil/empty slices. The retired `ScoreLeague`, `GetRankings`,
`GetPlayerScore` and `GetPowerRankings` bindings are not restored; Race consumes the surviving
root power-board entry point. `engine_refresh.go` retains scheduling, `modelMu`, dirty-flag
publication and health notifications. Directory ages and free-agent snapshots stay root because
they read shared held state under its existing locks; callbacks preserve their original read
order. Root wire goldens and integration tests complement implementation tests in the service.

## What does not exist, on purpose

- No ORM: parameterized `database/sql` only, inside the data layer.
- No DI framework: constructor injection from `app.go`.
- No package-level mutable state (`gochecknoglobals`).
- No cgo SQLite: `modernc.org/sqlite`.
- No `interface{}`/`any` in exported signatures (`ifaceguard`).
- No logging framework: the standard library.
- No inbound HTTP server.

The target UI data contract and fixture/live providers live in `frontend/src/app/data/`;
`look/` holds the shared visual channels (gravity, urgency,
signal) and glyphs; `assets/` is the C-01 asset board; `specimen/` is a dev-only page that loads
only with `?specimen` in a dev build. `snapshot.NewSource` composes the initialized mirror and rulebook; the directory carries its own fetch provenance. Money in the snapshot is exact cents; absent held fields are omitted.

The target shell lives in `frontend/src/app/shell/`; its six-node table and hash routes are data.
`frontend/src/app/commands/` owns the immutable verb registry, sole dispatch path, keyboard
bindings and in-memory recent-command log. `Act` is the only clickable primitive in the target
UI. `frontend/src/TargetMount.tsx` mounts the target shell; the Alpha harness UI was removed (`docs/architecture/removed-code.md`).

`docs/ui/registry/normalize.py` normalizes and validates `docs/ui/endpoint-registry.csv`.
`frontend/scripts/gen-endpoints.mjs` validates the shared owner/commissioner CSV and generates
`endpoints.gen.ts` (owner rows only) and `commissioner.gen.ts` (CM-001..CM-056 with owner `governs`
edges). The commissioner module is not imported by the app entry graph; future commissioner screens
must load it lazily. Both maps are frozen and freshness-tested;
`frontend/src/app/registry/` owns resolution, indexed placements, compact not-wired rows and the
pure two-route gate. `endpoint.open` navigates and highlights; merged rows target kept ids or places.

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
`snapshot.BuildClock` is the one clock builder: `TargetClock` (in `snapshot_app.go`) passes the what-if store, and
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

`snapshot_app.go`: `TargetDraftTaxi(franchiseID, playerID string) (envelope.Receipt, error)` saves a taxi plan.
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
`counterparties_filters.go` owns display treatment for filter-fixed posture shares and excluded tiers
in both Counterparties and its team Inspector.

`internal/ingestion/draftreports` validates MFL draftResults and futureDraftPicks into typed held
picks (string PlayerID, epoch instants) and ownership; one adapter family, two export fetchers.
`draft_reports_app.go` refreshes both with archive fallback and holds them; its bindings copy the
held feeds and call `internal/app/draft`, which reads local model values without network
(`held.go`) and authors nine report tables, grid, timing and value-distance bars (`reports.go`),
built as `internal/app/report` values the root converts to its wire types. The Draft Lab
(`draft_app.go`) stays at root on the Market surface types.
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
`TargetComms` snapshots existing season transactions, current mirror contracts and the held directory.
`internal/app/comms` reads weekly contract history through an explicit archive reader and builds
sentences shared by C-05 and M-032 with Go-validated filters. Refresh planning takes explicit
callbacks; key access, feed publication, archive caching and their shared locks stay on App.
Wire types remain declared at root and convert the service's shared `internal/app/report` values.
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
`roster_reports_app.go` exposes `TargetRosterReports` from held exports; `draft_reports_app.go`
exposes `TargetNFLDraft` from validated detailed player draft selections (`internal/app/draft`); `internal/lineup/byes.go`
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
points and the held free-agent list; it is the only writer of the model run. A held mutex makes a pass skip; changed-value loads,
league/list changes or a 24-hour-old run make it due. `target:season` refreshes open readers.
C-07 lists the engine run and its age; completed background checks append their run timestamp
without changing any scoring formula or using ownership to find points.

League Pulse › Archive: `archive_app.go` is the binding and keeps the checked archive snapshot on
`App`; `internal/app/archive` reads that snapshot (`read.go`) and authors the historical reports
(`board.go`, `records.go`), built as `internal/app/report` values the root converts to its wire
types (see `docs/architecture/feature-extraction.md`). The lazy
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

## Season rollover

`season.rollover` (`internal/commissioner/rollover_command.go`, CM-027, G3, commissioner only) moves a
native league to the next NFL season in one ledger entry. `Rollover` (`rollover.go`, `rollover_rules.go`,
`rollover_carry.go`) is the pure plan: contracts age a year, expiring players stay on their franchise marked
pending with an RFA or UFA route only where the rookie round is recorded, IR and taxi return to active, dead
money carries and season charges end, the champion is archived, waiting rule changes apply (the newest per
rule only) and every dated calendar edge moves one civil year. Picks are not issued. A rule it needs that is
unset blocks the plan with the reason. `PrepareRollover` builds the before-state (`RolloverBefore`: assets,
rules in force with their calendar, waiting changes) and the plan under the ledger transaction; a dry run
returns the plan and writes nothing. The one entry stores the before-state and the plan, and
`validateRolloverRecord` checks the plan's digest against the stored before-state on every read; replay
applies the stored plan and never rebuilds it. Once a ledger holds a rollover entry, entries (not dates) number
seasons (`rollover_season.go`): the closed season keeps the rules captured before the entry, the new one starts
from the plan's rules, and a compensating revert restores both. `CheckRolloverAppend` refuses reversing a
change the rollover captured, or a rollover with later changes. A league's first season is the NFL season its
creation records (`firstSeason`); a ledger recorded before that existed names no season until it is seeded.

## Native rookie draft

`internal/commissioner/draft_commands.go` holds the CM-041/CM-042 commands (`draft.order.auto`, `.set`,
`.pattern`, `.tiebreak` at G2, `draft.start` at G1) on the existing append path; setup (CM-040) is the
`draft.*` rule fields through `rules.set`. An order change made after the draft's start must say the clock
runs, which makes it G3 with a reason; `CheckDraftAppend` verifies that claim and refuses an order change
that would leave a made pick without its place. `draft_overlay.go` lays one draft's recorded order over the
rules in force, and `projectDraftCalendar` gives the calendar the draft's own start: `CalendarReading`, the
native gate and `draft_reading.go` all read the start through it, so the calendar's `rookie_draft` window
and the draft cannot disagree (a window command on that window is refused once a start is recorded).
`draft_reading.go` (`readDraft`) replays the ledger into `internal/engine/rookiedraft`; pick ownership comes
from the starting snapshot's future picks and is never recreated. A pick is the one owner action recorded in
the commissioner stream without commissioner permission: root `native_draft_app.go` registers it, so
`runNativeAction` asks the calendar first, and `history.RecordDraftPick` (`internal/store/history/draft.go`)
checks the franchise's seated owner, replays the draft, judges roster and cap and appends in one immediate
transaction. Go fixes the slot and contract in the record; `TargetCommissionerExecute` refuses `draft.pick`.
`native_projector.go` gives a drafted rookie his contract in the native assets. `TargetNativeDraft` reads
the order, picks, clock and the viewer's next pick in the viewer's time zone; a season of zero reads the
first draft nobody has closed (`draftSeasonNext`).

Round one follows the final standings of the season before the draft's season. `readDraft` takes a
`DraftStandings` seam (`draft_standings.go`) because the standings need the result sources only the store
holds: `history.draftStandings` (`internal/store/history/draft.go`) builds it over
`NativeStandingsReading`, read once per season and instant. It refuses in words while any regular-season game
of that season has no result, orders worst first, and applies the draft's tie rule (`draft.order.tiebreak`,
"League franchise order" or its reverse) only to franchises the standings leave level, refusing with "Needs a
ruling" when the rule is unset and a tie exists. Once the draft starts, round one is read as the standings
stood at the start, so a result corrected later cannot move it.

The live draft (CM-043..045) is recorded acts replayed by `rookiedraft.ReplayActs` (`acts.go`): `draft.skip`,
`.assign`, `.insert`, `.change` and `.timer` (pause, resume, set) at G3 with a typed reason, `draft.revert to`
at G3 and `draft.archive` at G2 (`draft_tray.go`). An expired clock skips its pick when the draft is read, at
that pick's own deadline, and a commissioner may still give it a rookie; an insertion is always a person's;
`change` names the rookie the commissioner saw, so a pick that moved is refused rather than retargeted. Go
fixes each rookie's contract in `DraftOutcome` when the act is recorded. A restore (`draft_revert.go`) marks
inactive the live entries after its checkpoint; nothing is deleted, and reading the ledger as it stood before
the restore gives the other state. A close (`draft_close.go`) records the undrafted rookies, who join the
native free-agent pool, and the picks issued for the next two seasons (every franchise, the draft's rounds,
tradable at once). The app closes a draft that has used its last pick, as `native.draft.clock`, from the
waiver clock (`processNativeDraftsDue`); a person may close it sooner. `checkLiveDraft` tries every live
change against the draft as it would stand, so the replay that reads it later is the one that refuses it
now, and `DraftPreview` (`draft_preview.go`) words each change for the tray.

`frontend/src/app/draftroom/` is the lazy Draft room above the Draft lab in Trade Floor › Draft room: the
clock with a countdown and the pick's close in the viewer's own time (`DraftClock`), the board by round
(`DraftBoard`), the owner's own pick (`YourPick`, `TargetNativeDraftPick`), what a close did
(`DraftOutcome`) and, for a co-commissioner or above, the shared side tray (`DraftTray`, `trayAction.ts`).
`draftRoom` (`state.ts`) holds the reading; `draft.wire.testdata.json` is Go-written
(`TestTheDraftRoomWireIsWhatGoSends`).

## Native contract action settings

`internal/commissioner/contract_actions.go` validates `contract.action.set` for the nine CM-009 actions.
The commissioner-only named command replaces one action's parameters and enable switch. Its replay
projects onto the same typed `contracts.*` field as `commish.rules.set`; current rules, pending changes,
replacement history, compensation and season-effective ordering all share that projection. Omitted
parameters explicitly need a ruling, rather than borrowing preset numbers. The command uses G2 at the
next rollover and the binding competitive-rule G3 safeguard for apply-now changes. Existing MFL owner
actions are unchanged.

## Native contract pricing

`internal/nativecheck/contract_*.go` is the pure price: `PriceContract(Input, ContractAction)` takes the rules
in force, one recorded contract (flat salary through its last season), what the franchise and contract have
already used (`Trail`) and what was asked, and returns `Pricing`: a plain refusal (off in this league, a limit
used, the contract does not qualify, a length over the maximum), each ruling still Needed, the new salary and
years, the season-by-season salary after, and the dead money by the season it lands in. Percentage math is
taken to cents first and then snapped to the league's increment (`scaleMoney`, shared with cap counting);
every rate, floor, tier, premium and limit is read from the catalogue, none is written in the code. A
contract's years left are the seasons after the current one (`Expiration - season`), the count a release and
a buyout charge by. A release charges the restructured rate when the contract has been restructured.
`internal/commissioner/native_contract_quote.go` is the one check both the preview and the later owner action
call (`QuoteContract` -> `checkContract`): it asks `NativeActionGate` for the action's window (a retirement,
death and cap relief are found by the commissioner and have none), prices, then counts the franchise's cap
before and after with `nativecheck.CheckCap` and `CapAllows` (`native_contract_cap.go`); only a move that
raises this season's counted salary can be refused for the cap. The verdict order is: off or unruled or not
qualifying, then the calendar, then the cap. A refused quote still carries its price when the cause is only
the calendar or the cap. `TargetNativeContractQuote(RawContractQuote)` (`native_legality_app.go`) reads
the ledger and held player facts and records nothing. Limits count only contract actions recorded in the app
league; none is recorded yet (the owner action that records them comes later), so the used counts are empty
and the quote says so.

## Native fantasy scoring

`internal/fantasyscore` is a pure leaf: exact ten-thousandths, commissioner catalogue values,
true-position tackle/assist rates, per-play stacking bonuses, explicit unused stats and Needs a ruling.
It imports only the standard library and `internal/leaguerules`; no store, ingestion, I/O or scouting.
`StatLine` records canonical tackles/assists once; positional catalogue entries are rates, not extra stats.
`internal/fantasyscore/source` validates held nflverse weekly CSV and play-by-play, retains exact
long-play distances and return-touchdown credits (returning team only), and adapts them without fetching or store writes.
`source/testdata/comparison.csv` pins two complete held Legacy roster weeks against MFL totals,
with an independent native-points reference and every mismatch classified; the Legacy preset follows the held league’s MFL scoring settings, with optional per-stat rounding
and once-per-week counting. Eight source-credit differences remain explicitly unresolved. The app-owned player score reading composes these packages; existing MFL score surfaces remain unchanged.
`source.StatPort` is the read-only player-week revision port; `history.Store.PlayerWeekStats` implements
it over the exact registered weekly-file URL in the existing fetch archive. Distinct body hashes are
returned newest first with held timestamps, exact quantities and the feed position; only a refetch of the
body just before is collapsed, so a feed that goes back to an earlier line is a new revision. A missing player
stays absent rather than reusing an older line. Play-by-play is not a registered Signals feed, so each
weekly revision explicitly reports that missing evidence; consumers must not treat it as a complete
native scoring line. No fixture or network fallback is installed.

`source.IdentityPort` reads the existing `history.Store.SourceID` crosswalk; `ReadNativeEvidence`
validates a league player/week and returns every version of the player's week the held files gave
(`source.LineRevision`, oldest first, equal consecutive lines collapsed), plus the newest file; which version
counts is the league's (see Native statistics queue). Missing identity, week and play-by-play stay explicit absence. It does not fetch or manufacture a zero stat line.
`internal/commissioner/score_commands.go` adds CM-030 position set/clear and CM-032 season/week-scoped
player/franchise deltas through the existing command ledger, with CC+ permission, reasons and G2/G3.
`score_changes.go` folds compensation before selecting native scoring position and adding each delta
once, retaining linked ledger provenance; overflow is refused on append and compensation.
`ResultFlipCheck` compares published native games before and after a proposed score change. These utilities
feed `TargetNativePlayerScore` in `native_legality_app.go` through `history.Store.NativePlayerScore`.
`commissioner.NativePlayerScoreReading` combines the counted held version of the week, native position
and adjustment provenance, and season-bound `RulesInForce`. Complete totals alone have `points`;
incomplete evidence has explicitly labelled `partialPoints`, and missing identity/week or unsettled
rules have neither. Missing play-by-play stays explicit. The Go-written wire fixture lives at
`internal/commissioner/testdata/player-score-wire.json`. Existing MFL owner readings and scouting
positions are unchanged.

## Native statistics queue

CM-031. A native score never moves because the NFL feed changed: it counts the player's week at the
league's *baseline*, the first version the held files gave, until the ledger approves a later one.
`source/revisions.go` numbers a week's versions (`LineRevision`, digest over position and quantities) and
`LineBook` gathers them for a whole season in one pass; `history.Store.heldWeeklyFiles` is the one place the
archive is walked. `stat_commands.go` holds the three G2 commands (`commish.stat.queue.approve`,
`.reject`, `commish.stat.correct`; CC+, public, effective now, reason optional) and `StatItemID`
(`stat.<season>.<week>.<player>.<version>.<digest>`, parsed once at the boundary). `stat_ledger.go` folds the
active commands, compensation included: the last active approval is the baseline, an active rejection hides
that exact item, a correction sets one stat of the week. Replay never reads the feed to decide a past
approval; it only looks the approved version up in the archive and refuses if it is gone.
`NativePlayerScoreReading` counts baseline + corrections. `stat_queue.go` (`StatQueueReading`) makes one item
per player-week whose newest version differs from the baseline, priced by two real score readings (before,
and with the approval laid in a trial ledger) and by `ResultFlipCheck` with that same trial command;
`stat_decisions.go` lists decided items with who and when; `stat_preview.go` previews any of the three
(points and flip stated, a flip is a warning). `CheckStatAppend` refuses an item that is no longer the newest
version, is decided, or counts the same, and a correction without a source, with no effect or for a stat the
league does not score. Root `TargetStatQueue(league, season)` reads `history.Store.StatQueue`; the preview
goes through `TargetNativeScorePreview`. A week the newest file drops makes no item and unscores nobody.
Wire fixture: `internal/commissioner/testdata/stat-queue-wire.json`.

## Native results and standings

`native_results_app.go` exposes `TargetNativeResults(league, season, week)`,
`TargetNativeStandings(league, season)` and read-only `TargetNativeScorePreview(body)`.
`internal/commissioner/native_week_score.go` folds supplied weekly starters through the native
player-score reading, then adds franchise deltas once; an incomplete starter leaves no final total.
`native_results.go` scores only published games, preserves ties and freezes a closed season's
publication. `standings_rank.go` ranks exact records under the season's ordered criteria with plain
reasons and explicit missing-data skips. `standings_reading.go` counts regular-season weeks only,
complete weekly points (including byes) and whole-complete-week all-play comparisons.
`standings_commands.go` folds commissioner-only G3 record/order overrides and clear commands beside,
never over, computed standings; existing authorization, append, request-key and compensation paths own
all changes. `result_flip.go` compares before/after games for score previews: yes, no, or unknown with
reasons; a flip warns, never blocks recording. Go owns the results/standings wire fixture in
`internal/commissioner/testdata/native-results-standings-wire.json`.
`internal/store/history/native_results.go` composes held evidence and the `LineupSource`, which defaults
to the league's own ledger (see Native lineups); it never substitutes MFL starters. The existing held-stat
adapter still lacks play-by-play. The conference of each division now comes from the starting snapshot's
`Conferences` (see Native playoffs); while a league holds none, the conference criterion passes with no
data, as does the native power rank, which has no input yet. No owner screen is migrated and no scouting
or imported MFL reading changes.

A result-flip warning names only the games of the franchises the change touches (a player's score touches
the franchise that starts him), once each and by franchise name: "We can't tell yet whether this changes
A v B: neither score is complete."

## Native playoffs

`native_results_app.go` adds `TargetNativeBracket(league, season)`, a read-only reading of a season's playoffs.
`internal/commissioner/playoff_rules.go` resolves the playoff rules the season runs under
(`RulesInForce`, plus a bracket shape from the league's own graph, the built-in Legacy graph or a saved
shape) into `PlayoffRules`, or into a plain "Needs a ruling: ..." sentence when they do not fit together (a
field that does not add up, a shape that names a seed that does not exist, a shape that pairs by the seeds
left while reseeding is off, weeks that disagree). `playoff_conference.go` reads each team's conference from
the division alignment and the snapshot's `Conferences` (a one-conference league needs none).
`playoff_field.go` seeds each conference from the final standings: division winners first, wild cards next,
byes, and a seed set by hand replaces one slot. `playoff_slots.go` and `playoff_game.go` follow the shape
round by round, so a round pairs from the teams that reached it (the highest seed left plays the lowest when
reseeding is on); a game's winner is the higher total of the franchises' native weekly scores, the home team
adding the home field points; a tie has no winner until the commissioner advances a team. These are pure and
tested without a store. `bracket_reading.go` and `bracket_build.go` compose them with
`NativeStandingsReading` (the effective order: the commissioner's, when set) and the native weekly score of
each playoff week; the bracket is shown only once every regular-season game has a result.

Commands (all through the one execute path; typed, expected version, request key, compensation and replay):
`bracket.build`, `.edit`, `.seed`, `.preset.save` (CM-046, G2 until the first playoff game kicks off, G3 with
a typed reason after), `bracket.advance` (CM-047, always G3), `title.set` (CM-048, G1; the label is stamped
with the season's own year and a label naming another year is refused) and `history.link` (CM-049, G2,
co-commissioner or above). `bracket_state.go` folds the active commands of a season; `playoff_check.go` is the
append check. Whether the playoffs have begun is judged from the held NFL kickoffs of the first playoff week
(`store/history.playoffFactsFor`, the lineup environment); while they are not held the command is taken as
stated. `history_link.go` reads links: the champions archived with the league are the starting state, a link set
by hand replaces one and shows as set by the commissioner, and the bracket's champion and
`FranchiseHistoryReading` read through them. The season rollover does not yet take its champion from the
bracket. The Go-written wire fixture is `internal/commissioner/testdata/native-bracket-wire.json`. No
frontend screen reads it yet.

## Native lineups

`internal/commissioner/lineup_commands.go` adds `commish.lineup.set` (an owner, G2, private) and
`commish.lineup.correct` (CM-029, co-commissioner or above, G3, typed reason, shown to the league). A lineup
is the whole set of starters for one franchise, season and week; the latest active one stands, so
compensation restores the one before it (`RecordedLineupFor`, `LedgerWeekLineup` in `lineup_week.go`).
`AuthorizeRecords` lets only the seated owner of the franchise, with the `lineups` ability, set its lineup;
a commissioner changes someone else's only by the correction, and an owner cannot overwrite a correction.
`CheckLineupAppend` (run inside the append transaction through `Store.ExecuteCommissioner`) requires an
owner's lineup to be this season's, on the active roster, legal under `nativecheck.JudgeLineup` and not to
move any player whose game has kicked off, in or out; with the week's kickoffs not held it refuses rather
than guess. A correction skips the lock and the legality refusal: an illegal one is penalised when scored.
`LineupFacts` (names, positions, NFL teams, the week's kickoffs, the owner's zone) comes from the
`history.LineupEnvironment` the app installs at startup (`appLineupEnv` in `native_lineup_app.go`, reading the
held directory and held NFL schedule, no fetch); without it lineups cannot be set or corrected.
`franchiseWeekScore` judges a complete lineup with the same `nativecheck` functions: an invalid one loses
what the league's penalty rule says (Legacy: the top starter), a missing one is a flagged missed lineup
(`NativeResults.Missed`) and is never penalised. `NativeLineupReading` / `TargetNativeLineup(league,
franchise, week, at)` read starters, bench, who is locked and when (owner's zone) and the verdict.
`TargetNativeLineupSet` is the only door for an owner: it runs `lineup.set` through `runNativeAction`, the
first entry in `nativeOwnerActions()`. The action opens on the calendar action `lineup`, so a window (or
exception) that opens `lineup` must exist: the Legacy preset has a `lineups` window open all league year, since each player locks at his own kickoff. `lineup.correct` is sent through
`TargetCommissionerExecute` and previewed through `TargetNativeScorePreview` with the result-flip check.
Go owns `internal/commissioner/testdata/native-lineup-wire.json`.


Commissioner ledger filters: `TargetCommissionerLedgerFiltered(league, body)` decodes a
`commissioner.RawLedgerFilter`, then the history store builds the authorized reading before Go filters
by author, command kind and recorded time (inclusive start, exclusive end). Filters never change the
league version or expose private rows. `LedgerFilters` presents that reading, converts viewer-local
bounds to RFC3339, ignores superseded answers and reapplies on refresh. `DensityScreen` wraps the four
existing commissioner screens and uses the current shell's `connectDensity` and `createShellState` for independent memory;
`screenDensity` binds that projection outside React, through the `commissioner.surface` Act;
Narrative labels report cells, Tactical keeps the existing tables, Matrix compresses spacing.

Commissioner League landing: `TargetCommissionerJobs` reads native league records through
`history.CommissionerJobs`; `commissioner.JobsReading` derives ordered jobs from resolved rules.
`commissioner.JobCalendar` (`RolloverAt`, `Window`) is the typed replacement seam for the native
ActionCalendar; today's private `ruleCalendar` accepts only recorded exact instants, never inventing
season dates from month/day presets. Unbuilt jobs stay explicit. `TargetCommissionerControls` returns
all 56 rows' grouping, usable status and owned rule labels; questions reuse the generated commissioner
registry. Both readings have Go-written frontend wire fixtures. The lazy `LeagueLanding` replaces the
four stacked screens in `control/LeagueRoom`: jobs, question/rule search and eight collapsible groups;
existing setup/role forms and screens live inside their groups, MFL references stay below.
Non-rule search opens and focuses through `commissioner.surface` and `revealWorkSection`. `JobsStrip` reuses `clock.connectClock`'s DOM
projection with an optional plain-word formatter, so counting never rerenders React. Dates are viewer-local.

Commissioner rule dry runs: `TargetCommissionerPreview(body)` accepts the execute command body;
`history.PreviewCommissioner` holds a read-only snapshot and calls the same preparation, permission,
request-retry, version, governance and rule checks as execute, stopping before append. Refusals become
plain-word blockers, while storage failures remain errors. `commissioner.RulePreview` supplies the
question, frozen expected league/version, current/proposed values and provenance, timing and Go's
safeguard. The root embeds the existing endpoint registry; `ReadPreviewCatalogue` derives affected
owner-reading names and their catalogue-row count from its `governs` edges, never a player/team count.
No edges means a null count and "Not known yet". The lazy `RuleTray` uses the shared `SideTray` focus
lifecycle and `PreviewReview` anatomy. Landing Change links and rule search open it for the selected
rule; the background is inert until close. `RuleChangeForm` reads previews for the exact execute body
and request key, discards superseded responses and blocks confirmation until a matching preview is
ready. `preview.ts` parses the boundary; `ChangeBar` retains Go's confirm/hold/reason safeguards.
Setup, role changes and undo use the same lazy `ActionTray` and `PreviewReview` as rule changes;
`TrayLauncher` replaces inline confirms. `useChangePreview` shares exact-body/request matching across all
four forms. Go describes each command and refuses unchanged rules and roles in the shared append checks.
`CommandSafeguards` writes the command/safeguard fixture used by the frontend click/keypress guard.
The Go-written fixture is
`frontend/src/app/commissioner/preview.wire.testdata.json`.

League ops: notifications are two ordinary rules (`notify.transactions`, `notify.reminders`, owned by CM-051 and
not competitive, so a change needs one confirm). Notices and league notes are recorded acts
(`commish.notice.publish`, `commish.whatsnew.post`: plain text, pinned or not, shown to every owner, posted now);
`TargetCommissionerNotices` reads them from the ledger alone through `commissioner.NotesReading`, so a notice
exists for owners only once its entry is recorded and stops when a later entry withdraws it. The same reading
carries the app's own release notes (a Go table, nobody posts them) and the mail log (M-093), which says the
app sends no mail and lists every notice recorded, withdrawn ones marked. `TargetCommissionerDigests` writes one
plain sentence for each ledger entry the viewer may read that took a hold or a typed reason, through the same
privacy as the ledger. `commissioner.Governed` (who, when, which change) rides on each rule row Go shows as set
in the ledger and on each note; the reusable `app/governed/GovernedGlyph` shows it in the viewer's own time.
The commissioner package imports nothing that can reach a network or a mail system, and a test enforces it.
Ledger filters gain the kind "notices". Control Room › Data and sources gains "Read data again":
`TargetCacheScopes` lists the held copies and `TargetCacheClear` runs the refresh each screen already has, asked
to ignore the copy held (`cache_app.go`); the refresh scopes there are the existing S-09 Acts. The lazy
`LeagueOps` screen shows notices, What's New, the mail log and the digest in League ops; `NoteForm` posts in the
shared side tray with Go's preview.

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

## Native free-agent bids

`internal/nativecheck/bids.go` is the pure CM-013 policy: `NewBidPolicy` reads the resolved rules (modes, year
multipliers, limits, tiers, steps, late-bid mark, phase weeks, point decimals, minimum salaries, money
increment) and refuses any value still waiting for a ruling. `Check` judges one proposal in a fixed order and
returns the first refusal as one plain sentence; `ChooseBidWinner` takes the most points and reports equal top
bids as a tie, because the rules name no tie-break. In open bidding the bid to beat is the leader and a bid
placed after the late mark must beat it by the late step; in blind bidding it is the owner's own earlier bid
and there is no visible clock, so nothing about another owner's bid is revealed by a refusal.
`internal/commissioner/bid_command.go` holds the five commands: `bid.place`, `bid.raise` and `bid.withdraw` are
owner actions (calendar action `ufa_bid`, private in the ledger); `bids.process` and `bids.award.override` are
CM-036 commissioner commands at G2 for a co-commissioner or above. `bid_execute.go` judges them inside the append
transaction (`PrepareBidAppend`): bid rules, then roster and cap legality through the same check a waiver claim
uses, a player on waivers or released this season by the bidder is refused, and `bids.process` judges every bid
again at the decision. `bid_projector.go` replays bids and decisions with the native assets (the winner signs the
contract his live bid named, flat, expiring in `season + years - 1`), so a compensating revert undoes a decision
and a reopen replays it. `bid_reading.go` gives `TargetNativeBids(league, franchise)`: blind bids are visible only
to their owner (even their existence), open bids show the leader and the late-bid countdown, awards are public and
a losing bid is shown only to its owner; the mode read is the rule in force now. `bids.standFor` dates each
open auction's close in the viewer's zone; every new bid restarts it.
`commissioner.DueBids` uses the same decision policy as manual processing, leaving ties and players on
waivers undecided. `internal/store/history/native_bid_clock.go` submits due awards through the append
transaction with a process-only authorizer and rechecks eligibility under its lock. The managed waiver
clock also runs the bid worker, resuming persisted auctions on startup. `BidEnvironmentSource` supplies
the held NFL week and the native bracket's playoff field. Before the first held kickoff, or after a held
rollover and before September when no schedule is held, week-specific limits do not apply; other missing
weeks remain refused. The jobs reading reports the native waiver and bid queues as built without exposing
blind auction counts or times. `native_bids_app.go` exposes `TargetNativeBidAction(RawCommand)` and
`TargetNativeBids(league, franchise)`, and registers all three owner actions with the native gate.
Restricted free-agent bids wait for the tender step, which the app does not yet record.

## Native roster and asset corrections

`internal/commissioner/correction_command.go` validates CM-028's `correct.roster`,
`correct.transaction`, `correct.provenance` and `correct.pick`, and CM-039's `rule.exception`.
All are public-now G3 changes requiring a typed reason and co-commissioner permission, through
`ExecuteCommissioner` with version, request-key retry and compensation. `correction_projector.go`
changes cloned native assets only; an add uses the player's last recorded contract. Transaction
corrections compensate one active cut, waiver award or draft selection; dependent actions must be
reversed first. Draft consistency is checked on both correction and compensation. Draft ownership
reads the projected picks, never the original owners alone.

Rule exceptions match exactly one franchise, player or recorded transaction and stop counting at
their expiry instant. The shared native roster, cap, IR and taxi checks consume matched exceptions;
waiver eligibility (including rechecking at processing), bidding and rookie-selection legality pass
those same values to those same checks. Supported references are `roster.limits`,
`roster.positionBounds`, `cap.enforcement`, `ir.eligible`, `taxi.maximumYears`, and
`calendar.<action>` (franchise-scoped). Calendar exceptions feed `ActionCalendar.Decide` itself,
including permanent exceptions, rather than patching a second decision after it. Other rule references
are refused, never silently accepted. Rules and the imported MFL owner readings are unchanged.

`TargetCorrections(league)` (`commissioner_app.go`, `history.Corrections`) lists active corrections
newest first with author, recorded time, public reason, scope and named asset descriptions.
Native assets, roster legality, cap, reserves, waivers and draft readings carry the public corrections;
individual corrected players and picks also carry their reason. Go owns
`internal/commissioner/testdata/corrections-wire.json`; Wails generates the binding.
The rollover append lock resolves `NativeBracketReading` through history links before preparing its
frozen plan, so the archived champion is the decided season winner, not an unset snapshot field.
## Native trades

`native_trades_app.go` exposes `TargetNativeTradeAction` (offer, accept, decline, withdraw) through the native
owner registry and calendar gate, and `TargetNativeTrades(league, viewer)` (viewer is the seated franchise).
`internal/commissioner/trade_rules.go` resolves CM-014 without defaulting unset settings; an offer records
its review policy and expiry. `trade_execute.go` checks ownership, future-pick eligibility, all-party roster
and cap legality under the append lock, again at acceptance and approval. `trade_projector.go` folds offers,
votes, automatic party recusals, one appeal, commissioner/co-commissioner rulings and public reasoned
overrides into the same native assets as cuts, bids and draft picks. Compensation checks the prior trade
state and asset dependencies; replay never votes or re-decides an old trade. Compensatory picks are marked
on `NativePick` and never transferable. `draft_reading.go` reads projected pick ownership; reversing a trade
cannot strand a made draft selection. `trade_reading.go` gives all seated owners the public review trail,
frozen offered contract details and viewer-local times; expiry lapses an unaccepted offer at its instant.
The Go-written fixture is `internal/commissioner/testdata/native-trades-wire.json`. No imported MFL reading
or frontend screen is migrated.

## Commissioner act previews

`history.PreviewCommissionerWithFacts` uses the execute checks plus the shared
`prepareCommissionerTrial` (waivers, bids, trades, draft and rollover), with the app's held player
facts and owner zone. It holds a read-only ledger transaction and never appends a trial event.
`commissioner/preview_operations.go` compares typed owner readings over a private trial ledger;
`history/commissioner_preview_readings.go` adds held scores, queue decisions, results and standings.
Ledger labels supply the proposed act; roster/contract/cap/pick, people, schedule, calendar and draft
readings supply current values. Operational affected words name franchises; `affects.count` retains
its existing reading-count meaning. Permission refusals return blockers only, never private fields.
An already-recorded request previews its unchanged receipt, without a blocker or another write.

Starting assets use **`TargetCommissionerPreview(JSON.stringify(seedRequest))`**, where `seedRequest`
is the identical `{league, expected, request, reason}` sent to `TargetNativeSeed`. No `command` or
`args` is sent for this case. Go captures the same held snapshot through `nativeSeedCommand`, including
retry identity. No new Wails binding or client-supplied starting assets are needed. Every other act
uses the normal execute command body. Controls A–E and G (except CM-055), plus the built bracket and
advance commands (CM-046/047), are marked built; the rest of F keeps its prior flags.

## Commissioner market and competition pickers

`commissioner_market_app.go` exposes `TargetCommissionerMarket(league, command)` over held ledger
records. `commissioner.MarketReading` uses the exact command permission and emits only named target
choices: unprocessed waiver windows, processed claims, anonymous undecided auction targets, decided
bids and eligible trade reviews/appeals/overrides. Private claims and undecided bid identities or
amounts never travel. `TargetCommissionerSchedule` exposes authorized draft and published games
separately; owners retain publication-only readings. `TargetCommissionerActor` returns the local
session identity for self-only people acts. The lazy commissioner tray uses those doors, and the
existing `TargetNativeBracket`, for schedule and bracket acts with named game/round/seed/team pickers.
Go-authored market and competition wire fixtures pin the frontend bodies and previews.

## File migrations

`internal/db` owns the shared file runner and the complete main-file DDL. `user_version`
steps and their markers commit together after `VACUUM INTO <file>.pre-v<N>.bak`.
Main and what-if files know v1 (current-schema adoption); history knows v2 (typed
`quarantine_model_runs` / `quarantine_model_scores` repair, gated by FK findings).
The state store retains its older `schema_migrations` owner/version history inside
file adoption. History registry refreshes use DML, not startup DDL. Openers supply
the file's newest known version before any writer opens. `Pools.MigrationFindings`
returns both check passes; startup logs them, including unresolved unrelated FKs.
Repeated initialization at the same file version reuses completed checks within that
pool; `db.Check` is the explicit fresh-check path.
Existing backup paths are never overwritten: a collision blocks the pending upgrade.
