# App boundary — stacked revamp B

## Resume here

Part 2 in progress: **48 units / 2,212 baseline function LOC moved in Part 4**, including
35 committed before the usage-limit resume and thirteen units completed afterward.
Cumulative **116/126 units / 4,078/4,534 baseline function LOC**; 10 units / 456 LOC remain.
Frontend plan Part 3 shared read/cache work has not started.
Branch: `session/revamp-B-app-boundary`. No PR/merge. HANDOFF.md untouched.

Next: remaining Control read unit 102; Platform lifecycle/entry/resource
units 86, 90–92, 95–96, 98–100; then core policy audit/extraction and negative boundary ratchet.
Keep readiness-dependent captures lazy, original lock spans and refusal text. Use filtered
composition race batches with command-door parity; broad fixture runs need >200 seconds.

Still required for Part 2 completion: core Host policy extraction and negative composition/core
ratchet, full CI and disposable-profile live review. No permission/role/binding/schema change here.

## Boundary and preserved slice

Desktop Wails → existing feature API → composition capture → transport-free feature function.
The service receives values or narrow resource-read callbacks, not App/core/Wails. Composition
retains resource lifetime and locks; services may enrich caller-owned detached values and read
local archives through ports, but do not acquire shared state or change door admission/bindings. HTTP/CLI may call the same feature function,
but no new exposure is added in this part. Existing shared native commands already use
`internal/leagueservice`; this is not evidence that the desktop MFL reads exist over HTTP.

| Moved responsibility | Input → output | State/effects and failure boundary |
|---|---|---|
| `now.FinishedScoreCutoff` | held NFL schedule + observation timestamp + now → latest finished instant/known | Pure; retains kickoff/finality/4-hour behavior, empty/ambiguous/playing/future refusal. Composition still owns previous-week refresh, first-finish pinning, locks, pacing and held publication. |
| `platform.DatabaseChecksReading` | ordered named routine-check observations → report | Pure; zero observation means unchecked; failed beats empty/healthy regardless of order. Composition still reads each pool's LastCheck in original order, including missing pool as zero. No changes to actual database checking/backup schedules. |

**Done looks like:** CI permits only an explicit composition adapter/lifecycle allowlist and rejects
new domain-data branches there; feature services import neither composition/core nor Wails;
command-door parity stays unchanged. A ratchet must also cover core host decision code,
not merely empty composition by relocating it into core. This part documents the gate target;
it does not claim a new enforced ratchet.

## Inventory conventions

126 cohesive move units; 223 named functions; **4534 physical function LOC**
(including braces/comments inside functions, excluding imports/declarations/outside comments).
Baseline handwritten composition: 48 production files, 7445 physical LOC.
This is relocation scope, **not a deletion/savings claim**. A unit lists every contributing
function; decisions inside callbacks belong to the containing unit. Line numbers below are
stable references to `dda866b8`, not the current shortened files.

Doors: W = local Wails feature API (named after colon), B = background/local startup worker,
H = HTTP hosted, C = agent CLI. Unless explicitly H/C, local MFL reads are **not** assumed
hosted/CLI-exposed. Internal cross-feature consumers share the same local path. Runtime-only
policy is included conservatively; OS/transport argument dispatch may remain in the final
adapter allowlist, not be mislabeled domain logic.

Move bands, smallest risk first: **1** pure decisions; **2** held-read/wording orchestration;
**3** refresh/cache/evidence policy; **4** mutation/authorization/transaction lifecycles;
**5** resource/platform/transport policy (separate service decisions from legitimate adapters).
The first two rows are complete; row order within subsequent bands is advisory.

| # | Band | Baseline file and function:line | LOC | Owner under `internal/app/` | Actual doors/consumers | Decision(s) to move |
|---|---|---|---:|---|---|---|
| 1 | 1 | `internal/composition/finished_scores.go` — `finishedScoreCutoff`:14 | 25 | `now` | B,W | Week must have begun; final flag uses observation time, absent flag allows four hours, playing/future games refuse; latest finish wins. MOVED. |
| 2 | 1 | `internal/composition/database_checks_app.go` — `databaseChecksReading`:140, `databaseCheckWords`:165 | 33 | `platform` | W:sources | Unchecked/healthy/failed precedence, integrity and FK verdicts, report rows and failure wording. MOVED. |
| 3 | 1 | `internal/composition/archive_app.go` — `archiveOpponentFallback`:15 | 14 | `archive` | W:archive,comms | Find matching week's game and choose the other franchise, otherwise blank.  MOVED (part 2, batch 1). |
| 4 | 1 | `internal/composition/asset_board.go` — `finishedScores`:42, `onField`:70 | 34 | `research` | W:research,market,offers,trades | Count only prior held weeks, accumulate per-player points/games and newest timestamp; define total finished-week scope for Market.  MOVED (part 2, batch 2). |
| 5 | 1 | `internal/composition/contracts_host.go` — `scorePeriod`:60 | 18 | `contracts` | W:contracts | Accept only requested score period, reject archive without that period; preserve parse errors.  MOVED (part 2, batch 1). |
| 6 | 2 | `internal/composition/control_alerts.go` — `lineupAlert`:43 | 37 | `control` | W:control | Unknown week/feed/rules becomes unavailable; no alert after last lock or for full/legal lineup; otherwise next-lock urgency and trimmed problems.  MOVED (part 2, batch 2). |
| 7 | 2 | `internal/composition/control_alerts.go` — `tradeAlert`:81 | 25 | `control` | W:control | Reject unavailable evidence; count only incoming nonexpired offers; earliest nonzero expiry controls urgency.  MOVED (part 2, batch 2). |
| 8 | 2 | `internal/composition/control_alerts.go` — `irAlerts`:107 | 41 | `control` | W:control | Require current roster and live injuries; deduplicate my IR players; warn only on ineligible verdict with name fallback and IR target.  MOVED (part 2, batch 2). |
| 9 | 1 | `internal/composition/control_alerts.go` — `byeInputsNote`:194, `byeRoster`:211 | 39 | `control` | W:control | Require held byes/current week/current own roster; join player identity and active status to team bye weeks.  MOVED (part 2, batch 2). |
| 10 | 3 | `internal/composition/control_alerts.go` — `byeAlerts`:149, `byeWeekLock`:235 | 60 | `control` | W:control | Rules/week failures override availability note; unknown active-player facts abort; alert only current/next week shortfalls with local archived lock or unavailable-lock note.  MOVED (part 2, batch 2). |
| 11 | 2 | `internal/composition/control_alerts.go` — `Alerts`:24 | 18 | `control` | W:control | Acquire lineup then trades, stop on either failure; gather IR, lineup, bye, trade alerts in order with one clock reading. MOVED (part 3, batch 3). |
| 12 | 1 | `internal/composition/draft_reports_app.go` — `DraftNext`:23 | 12 | `draft` | W:draft | No what-if store means no date; otherwise clock failure text or next configured date.  MOVED (part 2, batch 3). |
| 13 | 3 | `internal/composition/engine_refresh.go` — `refreshEngine`:16 | 20 | `scoring` | B | Skip missing params/busy model; load signals and points before pool refresh and due check; point failure aborts; health event remains under original span. MOVED (part 3, batch 4). |
| 14 | 4 | `internal/composition/engine_refresh.go` — `checkEngine`:38 | 32 | `scoring` | B | Read prior run/input changes, apply EngineDue and dirty flag, fetch directory; clear dirty before run and restore on failure so concurrent changes survive. MOVED (part 3, batch 4). |
| 15 | 5 | `internal/composition/engine_refresh.go` — `engineFinished`:71, `sourceHealthChanged`:80 | 13 | `scoring` | B,W | Log run totals/duration and emit health event only if callback exists (runtime policy). MOVED (part 3, batch 4). |
| 16 | 2 | `internal/composition/engine_refresh.go` — `modelFreeAgents`:94 | 9 | `scoring` | B,W:scoring | Capture held free-agent IDs as engine input under season lock.  MOVED (part 2, batch 5). |
| 17 | 3 | `internal/composition/finished_scores.go` — `refreshFinishedScores`:40 | 47 | `research` | B | Previous market week only; hydrate missing score archive; pin first finished instant per week; reread week and YTD once if scores predate cutoff with paced cancellation. MOVED (part 4, batch 1). |
| 18 | 3 | `internal/composition/market_feeds_app.go` — `refreshMarketFeeds`:41 | 19 | `market` | B,W:market,offers | Skip fresh daily feeds unless forced, but projections must match week; pace reads and stop on cancellation. MOVED (part 3, batch 4). |
| 19 | 3 | `internal/composition/market_feeds_app.go` — `refreshMarketFeed`:61 | 38 | `market` | B,W:market,offers | Scope projection archive by week; reject wrong-week parsed feed; discard prior week's projection before publishing held/live-or-archive result. MOVED (part 3, batch 4). |
| 20 | 1 | `internal/composition/moves_app.go` — `validateMoveURL`:58 | 12 | `platform` | W:platform,contracts,trades | Allow only HTTPS MFL hostname/suffix without user info; preserve existing URL behavior.  MOVED (part 2, batch 1). |
| 21 | 1 | `internal/composition/moves_app.go` — `movePredicate`:87 | 14 | `platform` | B,W:platform | Intent-to-landing-predicate registry, unknown intent has no predicate.  MOVED (part 2, batch 1). |
| 22 | 1 | `internal/composition/moves_app.go` — `watchableMove`:441, `skipMoveRefresh`:445, `saveMoveEvidence`:476 | 16 | `platform` | B,W:platform | Require unchanged listed state; don't watch handed-off contract; roster refresh covers contracts; save state changes or new trade audit evidence only.  MOVED (part 2, batch 1). |
| 23 | 4 | `internal/composition/moves_app.go` — `handOffMove`:19 | 38 | `platform` | W:platform | Ready/startup and own-franchise checks; refuse trade-desk/release-only acts; validate URL, persist handoff before browser/event/wake. MOVED (part 4, batch 3). |
| 24 | 4 | `internal/composition/moves_app.go` — `withBidFacts`:103, `moveSources`:121 | 35 | `platform` | B,W:platform | Load predicate-specific contract/bid facts; union required sources; log each unknown intent once. MOVED (part 4, batch 3). |
| 25 | 4 | `internal/composition/moves_app.go` — `checkMoves`:140, `refreshedObservation`:200 | 69 | `platform` | B,W:platform | Refresh only with awaiting work and known sources; reread under moves lock, skip stale observations/state races, save changed evidence; emit once if changed. MOVED (part 4, batch 3). |
| 26 | 3 | `internal/composition/moves_app.go` — `refreshMoveSources`:211, `refreshMoveSource`:251, `recentMoveSource`:305 | 80 | `platform` | B,W:platform | Latest handoff sets evidence floor; fixed refresh order/spacing; skip recent successful post-handoff feeds and duplicate contracts; injectable refresh or normal source routes, log roster failure. MOVED (part 4, batch 3). |
| 27 | 4 | `internal/composition/moves_app.go` — `moveObservation`:273 | 29 | `platform` | B,W:platform | Build roster/season/contract evidence; successful unchanged roster refresh advances observation time without lying about content change. MOVED (part 4, batch 5). |
| 28 | 4 | `internal/composition/moves_app.go` — `revalidateReadyRosterMoves`:330, `revalidateReleaseAfterRefresh`:430 | 57 | `platform` | B,W:control,sources | Revalidate release/contract plans, then only ready IR/taxi candidates; rebuild eligibility, batch state changes and notify. MOVED (part 4, batch 5). |
| 29 | 4 | `internal/composition/moves_app.go` — `planDecision`:380, `returnPlanToDraft`:398 | 46 | `platform` | B,W:control,sources | Select taxi/IR verdict only for still-ready held player; ineligible only follows invalidate→rebase→save, unknown verdict leaves plan unchanged. MOVED (part 4, batch 3). |
| 30 | 4 | `internal/composition/moves_app.go` — `withdrawRosterMove`:449 | 26 | `platform` | W:platform | Require startup and ownership; withdraw through envelope, save before audit/event. MOVED (part 4, batch 3). |
| 31 | 5 | `internal/composition/moves_app.go` — `requestMovesPass`:71, `logMove`:81 | 12 | `platform` | W:platform;B | Ready gate before wake; log last audit transition (runtime adapter policy). MOVED (part 4, batch 3). |
| 32 | 4 | `internal/composition/moves_watcher.go` — `watchMoves`:17, `movesFastUntil`:57 | 62 | `platform` | B | Restart pass; wake/poll share minute rate limit; consume queued wake in same pass; fast mode lasts 30 minutes after latest handoff; cancellation stops timer. MOVED (part 4, batch 3). |
| 33 | 4 | `internal/composition/native_gate.go` — `nativeOwnerActions`:32, `startContractAction`:57 | 28 | `commissioner` | W:commissioner | Register only priced contract actions with owner window; before starting require action enabled in held ledger. MOVED (part 3, batch 1). |
| 34 | 4 | `internal/composition/native_gate.go` — `runNativeAction`:79 | 25 | `commissioner` | W:commissioner | Reject unknown action/startup failure; replay ledger, ask calendar at owner time/zone; only permit executes; preserve closed/refused error chain. MOVED (part 3, batch 1). |
| 35 | 5 | `internal/composition/native_gate.go` — `ownerZone`:70 | 6 | `commissioner` | W:commissioner;B | Injected owner zone else OS local zone. MOVED (part 3, batch 1). |
| 36 | 4 | `internal/composition/native_owner_app.go` — `recordOwnerAction`:24, `tradeFamily`:47, `waiverFamily`:62, `bidFamily`:77 | 49 | `commissioner` | W:commissioner | Decode owner command, require family args and matching permit league/franchise/action, then shared service execution with existing refusal text. MOVED (part 3, batch 1). |
| 37 | 4 | `internal/composition/native_draft_app.go` — `runDraftPick`:27, `closeFinishedDrafts`:52 | 29 | `commissioner` | W:commissioner | Decode/validate pick, resolve identity, record under permit/deadline; close finished drafts best-effort without undoing pick. MOVED (part 3, batch 1). |
| 38 | 4 | `internal/composition/native_lineup_app.go` — `runLineupSet`:30, `LineupFacts`:63 | 39 | `commissioner` | W:commissioner | Decode lineup and require matching permit; construct private immediate command; find selected week's held NFL locks, no network. MOVED (part 3, batch 1). |
| 39 | 2 | `internal/composition/native_schedule_app.go` — `nativeScheduleDates`:18 | 27 | `commissioner` | W:commissioner;desktop registry service host (H/C use separate hosts) | Read reference NFL schedules only for requested season; ignore bad bodies/conversion; first valid row per week wins.  MOVED (part 2, batch 2). |
| 40 | 3 | `internal/composition/native_schedule_app.go` — `BidEnvironment`:48 | 35 | `commissioner` | W:commissioner;store callbacks | Replay season, derive week/offseason fallback; only known week loads playoff bracket/directory; build shared bid environment. MOVED (part 3, batch 1). |
| 41 | 4 | `internal/composition/native_seed_app.go` — `nativeSeedCommand`:12, `heldNativeSnapshot`:33 | 54 | `commissioner` | W:commissioner;desktop registry service host (H/C use separate hosts) | Retry must match original seed/expected/reason; require season-aligned held rosters/settings, parse roster date, assemble/validate snapshot with held draft and directory facts. MOVED (part 3, batch 5). |
| 42 | 3 | `internal/composition/now_app.go` — `refreshNowWeek`:58, `feedFresh`:152 | 44 | `now` | B,W:now,research | Five-minute schedule and daily bye TTL, forced refresh override; preserve last good copy; pace schedule before bye fetch. MOVED (part 3, batch 2). |
| 43 | 2 | `internal/composition/now_app.go` — `scheduleFeed`:100, `checkScheduleWeek`:145 | 27 | `now` | B,W:now,research | Both live and archive must match requested week; scope archived URL by W. MOVED (part 3, batch 2). |
| 44 | 3 | `internal/composition/now_app.go` — `seedClockSchedules`:122 | 22 | `now` | startup | Seed current week plus nonzero clock rule weeks from archive only; log failed reads and skip missing. MOVED (part 3, batch 2). |
| 45 | 3 | `internal/composition/now_app.go` — `NowRefresh`:175 | 31 | `now` | W:now | Race refresh first; choose market week for zero; only current or listed league week accepted; positive week triggers schedule refresh under original lock. MOVED (part 3, batch 2). |
| 46 | 1 | `internal/composition/now_app.go` — `KnownFranchise`:157 | 4 | `now` | W:now | Rulebook membership determines known franchise.  MOVED (part 2, batch 2). |
| 47 | 4 | `internal/composition/now_app.go` — `LiveGame`:235 | 37 | `now` | W:now | Initialize adapter; reset on week change retaining fetch/finals; seed empty board; reject absent follow ID; refresh/final and MFL poll, attach detail failure without losing view. MOVED (part 3, batch 2). |
| 48 | 3 | `internal/composition/now_app.go` — `startLiveGame`:273, `pollLiveMFL`:288 | 30 | `now` | W:now | Lazy endpoint/cache setup; active due MFL polling only; success resets failures, failure uses exponential delay capped at 16×30 seconds. MOVED (part 3, batch 2). |
| 49 | 2 | `internal/composition/now_app.go` — `liveGameSpines`:327, `spinePlayers`:356, `inspectLiveBox`:381 | 72 | `now` | W:now | No detail work without held/followed games; require history/crosswalk; only known ESPN IDs and directory facts become spines/photos; attach box entities for matched IDs only. MOVED (part 3, batch 2). |
| 50 | 3 | `internal/composition/offers_reading.go` — `recordOffers`:103 | 14 | `offers` | B,W:control | Skip failed reads/missing pools; save pending offers with transaction-ending evidence for archive owner; history save failure is separate from MFL read failure. MOVED (part 4, batch 1). |
| 51 | 1 | `internal/composition/offers_reading.go` — `personalSnapshot`:122 | 12 | `offers` | W:offers,contracts,trades | Require loaded snapshot and franchise membership; otherwise exact personal refusal.  MOVED (part 2, batch 1). |
| 52 | 3 | `internal/composition/pool_app.go` — `refreshFreeAgents`:20, `restoreFreeAgents`:55 | 62 | `offers` | B,W:offers | Restore archive before first refresh for cross-launch comparison; parse/normalize/sort IDs, set model dirty only for successful changed list; retain good held data on failure. MOVED (part 3, batch 4). |
| 53 | 3 | `internal/composition/pool_app.go` — `refreshOffersPool`:87 | 18 | `offers` | W:offers | Startup gate; fresh unforced pool still refreshes Market reports; otherwise pool then Market under refresh lock. MOVED (part 3, batch 4). |
| 54 | 2 | `internal/composition/pool_app.go` — `poolAssetValues`:106 | 14 | `offers` | W:offers | Hydrate Research, value requested IDs in Market model, index board values and preserve source note. MOVED (part 3, batch 3). |
| 55 | 1 | `internal/composition/power_rankings_app.go` — `remainingSchedule`:120, `ages`:130, `seasonWeeks`:146 | 29 | `scoring` | B,W:race | Schedule failure removes column; missing directory/birthdate means NaN age; age uses 365.2425-day years; unreadable season length gives zero.  MOVED (part 2, batch 2). |
| 56 | 3 | `internal/composition/race_app.go` — `refreshRace`:22 | 46 | `race` | B,W:race,now | Read/publish weeks→brackets→standings with spacing; successful standings builds season/franchise power using one cached remaining schedule; publish power only on success. MOVED (part 3, batch 5). |
| 57 | 3 | `internal/composition/race_app.go` — `scopedBodies`:71, `bracketsOrArchive`:85 | 47 | `race` | B,W:race,now,research | Archive must match URL scope; bracket list plus each bracket required; paced failure aborts all; last nonlive part sets aggregate freshness. MOVED (part 3, batch 5). |
| 58 | 2 | `internal/composition/race_app.go` — `raceSnapshot`:123 | 15 | `race` | W:race,now | Capture joint live/held state; detect unplayed live week; apply live overlay only to matching week. MOVED (part 3, batch 2). |
| 59 | 1 | `internal/composition/refresh_app.go` — `seasonGuess`:44 | 6 | `control` | startup,B,W:sources | Use held season else wall-clock year for discovery.  MOVED (part 2, batch 2). |
| 60 | 4 | `internal/composition/refresh_app.go` — `refreshLeague`:53, `refreshLeagueLocked`:70, `fetchLeague`:104 | 78 | `control` | startup,B,W:sources;move watcher | Discover current season, normalize roster/money; stage rules+mirror and atomically commit; release prepared rules on mirror-stage failure; update checked times/model dirty; successful outer refresh updates draft feeds/revalidates plans. MOVED (part 4, batch 7). |
| 61 | 5 | `internal/composition/refresh_app.go` — `refreshInBackground`:141, `launchRefresh`:172 | 44 | `control` | startup,B | Start due clock and move watcher once after readiness; optional launch refresh then hourly work; stopping prevents worker admission; report failures while preserving league. MOVED (part 4, batch 7). |
| 62 | 2 | `internal/composition/research_archive.go` — `archivedResearchFeed`:24 | 7 | `research` | B,W:research,market,offers | Weekly feed selects scoped archive; unscoped kinds read all archived bodies.  MOVED (part 2, batch 4). |
| 63 | 3 | `internal/composition/research_archive.go` — `hydrateResearchContext`:41 | 49 | `research` | W:research,market,offers,trades | One archive hydrate at a time; missing held feeds/weekly scores/depth/injuries/news only; roster+extra IDs define news scope; archived injury marked pending; merge without overwriting live data.  MOVED (part 2, batch 4). |
| 64 | 2 | `internal/composition/research_archive.go` — `firstArchiveRead`:92, `hydrateResearchFeed`:106 | 25 | `research` | W:research,market,offers,trades | Each archive key tried once; only missing feeds hydrate; live failure reason overrides archived pending note.  MOVED (part 2, batch 3). |
| 65 | 3 | `internal/composition/research_archive.go` — `mergeResearchArchive`:117, `hydrateArchivedDepth`:169 | 65 | `research` | W:research,market,offers,trades | Refresh links when crosswalk version changes; fill only absent held data; old-day news becomes pending; preserve earlier live errors over archive fallback.  MOVED (part 2, batch 4). |
| 66 | 2 | `internal/composition/research_reading.go` — `researchRows`:33, `snapshot`:45 | 26 | `research` | W:research,market,offers,trades | Use first feed's rows; choose now if unset; finish-score totals and held cut charges become research input, with detached indexes.  MOVED (part 2, batch 3). |
| 67 | 3 | `internal/composition/research_reading.go` — `PrepareResearchDepth`:73 | 8 | `research` | W:research | No fetch transport means no depth preparation; otherwise bounded nonforced refresh. MOVED (part 4, batch 1). |
| 68 | 3 | `internal/composition/research_runtime.go` — `refreshResearchFeed`:33, `researchLinks`:69, `refreshResearchDepth`:84 | 77 | `research` | W:research | Five-minute feed/daily depth freshness unless forced; weekly archive scoping; link only known MFL IDs for gsis/espn; publish links/errors and held feeds. MOVED (part 4, batch 1). |
| 69 | 3 | `internal/composition/research_runtime.go` — `refreshResearch`:111 | 43 | `research` | W:research | Read kinds/YTD then weekly scores through market week with status/pacing; injuries, current schedule, depth follow; startup/cancellation errors preserve text and clear status. MOVED (part 4, batch 1). |
| 70 | 3 | `internal/composition/research_runtime.go` — `refreshResearchNews`:176 | 45 | `research` | W:research | Validate optional mine and franchise; reject concurrent refresh; order roster IDs with directory; bounded news batches, merge result even on error. MOVED (part 4, batch 1). |
| 71 | 3 | `internal/composition/research_runtime.go` — `researchDepthOrArchive`:227 | 9 | `research` | W:research | Live depth success returns live freshness; otherwise log and use fallback archive preserving original error. MOVED (part 4, batch 1). |
| 72 | 3 | `internal/composition/season_app.go` — `refreshSeason`:31 | 49 | `control` | B | Fixed transactions/comms/shell/lineups/trades/injuries/draft/market/race/firsts/current-schedule order; pace/cancel, notify early for lineups, revalidate plans, always wake watcher. MOVED (part 4, batch 2). |
| 73 | 3 | `internal/composition/season_app.go` — `refreshInjuries`:83, `heldInjuries`:116, `refreshSeasonTransactions`:129 | 71 | `control` | B,W:research,platform | Live/archive parse and convert; publish last-good on error; injury ID→status/provenance view; injury export is leagueless. MOVED (part 4, batch 2). |
| 74 | 3 | `internal/composition/season_app.go` — `refreshSeasonLineups`:158, `lineupWeekParam`:208 | 55 | `control` | B,W:now | Zero week uses blank W; live and archived results scoped to selected week; reject mismatches and clear old week before publishing. MOVED (part 4, batch 2). |
| 75 | 4 | `internal/composition/season_app.go` — `refreshPendingTrades`:215, `clearPendingTrades`:258 | 44 | `control` | B,W:control | Keyring read precedes even host discovery; generation fences key deletion; no-key clears data, unavailable/timeout retains data with failure; observe offers separately before hold publication. MOVED (part 4, batch 2). |
| 76 | 5 | `internal/composition/season_app.go` — `refreshPendingTradesInBackground`:264 | 14 | `control` | W:control | Key-change refresh uses cancellable worker context else fallback; refresh lock then season notification. MOVED (part 4, batch 2). |
| 77 | 3 | `internal/composition/shell_feeds_app.go` — `readShellFeed`:16, `dailyShellReading`:33, `refreshDailyShellFeed`:43 | 40 | `control` | B,W:control,offers | Read archive only when absent; old local date marks pending; at most one network attempt per export/local day (also skip today's successful held read); publish result under season lock. MOVED (part 4, batch 2). |
| 78 | 2 | `internal/composition/shell_feeds_app.go` — `mergeMFLCalendar`:89 | 44 | `control` | W:platform,control | Map held event timestamps/title fallback; discard past deadlines; sort urgency/time/ID; append calendar provenance/unavailability note.  MOVED (part 2, batch 2). |
| 79 | 3 | `internal/composition/snapshot_app.go` — `takeSnapshot`:23, `targetDirectory`:44, `targetInjuries`:85, `injuriesHeld`:94 | 70 | `platform` | W:platform | Ready gate, serialized directory/injuries snapshot; first-use injury refresh; live directory time uses original lookup timestamp, archive/fail provenance explicit; publish only built snapshot. MOVED (part 4, batch 5). |
| 80 | 3 | `internal/composition/snapshot_app.go` — `targetClock`:100 | 20 | `control` | W:platform,control,draft,trades | Require ready/healthy what-if; derive clock from phase/calendar/NFL week windows; merge held MFL calendar, no fetch. MOVED (part 4, batch 5). |
| 81 | 1 | `internal/composition/snapshot_app.go` — `rosterMoveTarget`:121 | 11 | `platform` | W:platform,contracts,offers,trades | Unknown discovered host gives unmapped target/block; otherwise encode league/options URL, never submit.  MOVED (part 2, batch 1). |
| 82 | 4 | `internal/composition/snapshot_app.go` — `draftRosterMove`:133, `listMoves`:186 | 71 | `platform` | W:platform | Require own franchise/player/held snapshot; taxi vs IR chooses target/ID prefix/envelope; save local plan only; rail labels trade receipts but omits bid/release/contract plans. MOVED (part 4, batch 5). |
| 83 | 3 | `internal/composition/sources_runtime.go` — `compileFactsInBackground`:47, `warmArchive`:79 | 38 | `sources` | startup | Only existing manifest permits republish/compile/warm/model-fit; failures stop later steps; bounded worker and archive cache lock. MOVED (part 4, batch 4). |
| 84 | 2 | `internal/composition/sources_runtime.go` — `readSettings`:102, `writeSettings`:117 | 28 | `sources` | startup,W:sources | Missing file means defaults; JSON decode errors remain visible; write indented owner-only temp file then rename atomically. MOVED (part 3, batch 3). |
| 85 | 3 | `internal/composition/asset_board.go` — `fitAssetModel`:133 | 17 | `market` | startup | No verified cache means no fit; compile error logs without publishing; success publishes model and season event. MOVED (part 4, batch 4). |
| 86 | 5 | `internal/composition/platform_host.go` — `startHeadshots`:61, `photoHandler`:83 | 30 | `platform` | startup,W photo HTTP | History/crosswalk failures preserve error; lazy cache plus photo-off setting; unavailable cache returns 503, off switch denies request. |
| 87 | 5 | `internal/composition/database_checks_app.go` — `checkDatabases`:31, `startDatabaseChecks`:43 | 38 | `platform` | startup,B | Skip missing pools/canceled context; log findings; start one ready-gated hourly worker with bounded check/backup cycle and joined shutdown. MOVED (part 4, batch 8). |
| 88 | 4 | `internal/composition/database_checks_app.go` — `newDatabaseBackups`:71, `backupDatabases`:83, `backupNow`:99, `pruneBackups`:125 | 55 | `platform` | B,W:commissioner;restore CLI | Build-stamped dev/prod matched-set schema limits; daily capture before retention; manual capture requires local account and configured backups; private logs/public generic failure; prune findings best-effort. MOVED (part 4, batch 8). |
| 89 | 3 | `internal/composition/app.go` — `directory`:73 | 17 | `sources` | startup,B,W many | Cache season directory after first successful normalized player fetch; failure never publishes partial lookup. MOVED (part 4, batch 4). |
| 90 | 5 | `internal/composition/app.go` — `startup`:126, `settleStartupFailure`:180, `finishStartup`:187, `initStoreFloor`:255 | 131 | `platform` | startup/probe | Logging failure nonfatal; acquire protected DB floor before client; initialize ordered stores; refresh only empty mirror; coordinator owns Writer; publish complete floor/readiness before revalidation; plain offline failure; launch refresh only if not seeded. |
| 91 | 5 | `internal/composition/app.go` — `openDatabases`:209, `logMigrationFindings`:319 | 48 | `platform` | startup/probe | Instance lock before recovery and pre-migration backup; open matched stores at schema versions, archive subsequent fetches; log only present pools. |
| 92 | 5 | `internal/composition/app.go` — `shutdown`:334, `startWorker`:355 | 32 | `platform` | shutdown,B | Mark stopping/cancel workers under admission lock; join before closing pools/releasing instance lock; no new worker after stopping. |
| 93 | 1 | `internal/composition/app.go` — `configDir`:370, `isDevBuild`:386, `dbFileName`:390 | 21 | `platform` | startup,CLI | Owner-only config dir; dev stamp routes to separate dev databases, never production ledger.  MOVED (part 2, batch 3). |
| 94 | 5 | `internal/composition/logging.go` — `setupLogging`:19, `pruneLogs`:37 | 25 | `platform` | startup | Owner-only per-launch log plus stderr; keep ten newest chronological launch files, best-effort pruning. MOVED (part 4, batch 4). |
| 95 | 5 | `internal/composition/instance_lock_unix.go` — `acquireInstanceLock`:14, `releaseInstanceLock`:29 | 20 | `platform` | startup,CLI | Nonblocking exclusive OS lock; close on lock failure; nil-safe unlock/close. |
| 96 | 5 | `internal/composition/instance_lock_windows.go` — `acquireInstanceLock`:15, `releaseInstanceLock`:32 | 22 | `platform` | startup,CLI | Windows exclusive lock equivalent; nil-safe unlock/close. |
| 97 | 1 | `internal/composition/version.go` — `buildLabel`:14 | 6 | `platform` | startup,CLI,W stamps | Append parenthesized commit only for stamped builds; ldflags globals must stay compatible.  MOVED (part 2, batch 1). |
| 98 | 5 | `internal/composition/restore_cli.go` — `listBackups`:20, `restoreBackup`:51, `restoreFailure`:86 | 80 | `platform` | offline CLI | List manifest directories sorted; lock before basename-confined matched restore; distinguish newer journal/no prior data; private technical log with temp fallback, public recovery wording/exit codes. |
| 99 | 5 | `internal/composition/run.go` — `Main`:37, `runFlag`:86, `probe`:103, `packageSmoke`:115 | 96 | `platform` | CLI/desktop entry | Known argument shapes only, unknown args never fall into production GUI; probe exits based on startup; smoke requires absolute existing disposable dir/lock and no stores/credentials/workers. |
| 100 | 5 | `internal/composition/run.go` — `helperCommands`:155, `runNative`:167, `nativeUpdateLock`:209 | 61 | `platform` | hosted desktop entry | Only delegable helper catalog IDs; hosted startup excludes local App; HTTPS/league admission; update-cache lock retries 100×100ms; unsigned trust failure does not prevent screen; dev-only practice update flag. |
| 101 | 2 | `internal/composition/cache_app.go` — `cacheScopes`:23, `ClearCache`:63 | 28 | `sources` | W:sources | Ordered five-scope labels/actions; scope selects corresponding forced feature refresh; unknown scope returns not found without effects. MOVED (part 3, batch 3). |

| 102 | 3 | `internal/composition/control_host.go` — `FetchRuleNames`:51, `FetchSiteNews`:65, `SourceHealth`:79 | 36 | `control` | W:control | Bounded live/archive rule dictionary and news; source health fails explicitly rather than fabricating state. |
| 103 | 2 | `internal/composition/control_host.go` — `OpenMFLAccounting`:102 | 16 | `control` | W:control | Require startup; discovered host or empty host passes through AccountingURL admission before browser open. MOVED (part 3, batch 3). |
| 104 | 4 | `internal/composition/control_host.go` — `Params`:119, `SetParam`:149, `ResetParam`:162 | 53 | `control` | W:control | Read definitions/current overrides, fail on missing position value; word labels/groups and sort group→label→position; bounded admin override/reset stores return explicit success/failure. MOVED (part 3, batch 3). |
| 105 | 4 | `internal/composition/control_host.go` — `keyStatus`:175, `KeyStatus`:183, `SetKey`:201, `DeleteKey`:234, `lookupMyFranchise`:255 | 84 | `control` | W:control;startup | Trim/plausibility 16–256-byte key, verify before storage, distinguish rejected/unreachable/keyring unavailable; store and resolve franchise only after verification, bump generation and refresh. Delete clears identity/verified time/trades and fences older work. Lookup failure denies owner actions. MOVED (part 4, batch 7). |
| 106 | 4 | `internal/composition/commissioner_host.go` — `Principal`:25, `commissionerPrincipal`:33, `commissionerSeat`:49, `commissionerSeatSignIn`:63 | 45 | `commissioner` | W:commissioner;local service host | Nonempty installation plus OS uid hashes stable local identity; injected session overrides fallback; reject blank identity. Ready-gated local seat/status/sign-in preserve account aliases and re-read status after sign-in. MOVED (part 4, batch 4). |
| 107 | 2 | `internal/composition/commissioner_host.go` — `nativeFacts`:90 | 13 | `commissioner` | W:commissioner;store callbacks | Held directory contributes player facts only when loaded; only held injuries supply NFL designations; no network.  MOVED (part 2, batch 5). |
| 108 | 2 | `internal/composition/archive_app.go` — `readArchiveFacts`:32 | 13 | `archive` | W:archive,comms;startup | Read verified snapshot with curated readings/compiler/manifest; retain cached snapshot only when returned SHA is nonempty; preserve service errors. MOVED (part 3, batch 3). |
| 109 | 4 | `internal/composition/trades_app.go` — `DraftTradeAccept`:42 | 25 | `trades` | W:trades | Build request/check from held offer; missing MFL host blocks; draft accept, supersede older moves, then save receipt under moves lock. MOVED (part 4, batch 6). |
| 110 | 4 | `internal/composition/trades_app.go` — `TradeBuilder`:69 | 44 | `trades` | W:trades | Validate saved-plan ownership before any action; capture state/catalog, preview/save/reject/revoke dispatch; run remove/handoff then return current saved plans under what-if→refresh locks. MOVED (part 4, batch 6). |
| 111 | 2 | `internal/composition/trades_app.go` — `tradeState`:115, `tradeBuilderTeams`:133, `builderPickValue`:250 | 34 | `trades` | W:trades,contracts | Held lineup selection validates snapshot/franchise; gather pending trades and valued player/pick assets; missing model gives explicit Not valued cell; failed board yields no values. MOVED (part 3, batch 3). |
| 112 | 4 | `internal/composition/trades_app.go` — `tradeID`:148, `keepTradePlan`:155 | 32 | `trades` | W:trades | Random trade ID; construct propose request, missing-host block; save what-if plan then envelope then event (preserve existing ordering, no atomicity change). MOVED (part 4, batch 6). |
| 113 | 4 | `internal/composition/trades_app.go` — `draftTradeExit`:180, `savedTradePlans`:220 | 68 | `trades` | W:trades | Require still-pending offer and correct owner direction; stale pending/unknown host block; supersede prior exits before save. Saved plans require own franchise, omit withdrawn and recompute preview before attaching receipt. MOVED (part 4, batch 6). |
| 114 | 4 | `internal/composition/trades_app.go` — `removeTradePlan`:259, `tradeBuilderPlanAction`:281, `requireTradePlanFranchise`:513 | 43 | `trades` | W:trades | Dispatch remove/handoff only; saved ID actions recheck ownership; remove verifies mine and transitions via WithdrawTrade before save/audit/event. MOVED (part 4, batch 6). |
| 115 | 4 | `internal/composition/trades_app.go` — `tradeSimulation`:298, `tradeProjectionInput`:499 | 64 | `trades` | W:trades | Own what-if vs counterparty mirror; fresh isolated generations; merge only currently-owned players/adjustments, seed separate preview store; derive before/seed/live cap inputs without mutating authoritative mirror. MOVED (part 4, batch 6). |
| 116 | 4 | `internal/composition/trades_app.go` — `tradeProjection`:349 | 54 | `trades` | W:trades | Detect newer MFL stamps with reset reasons; collect clock/roster/pick/leg blocks; execute local trade only with nonempty valid legs; project each franchise's rich roster/cap and deal text. MOVED (part 4, batch 6). |
| 117 | 4 | `internal/composition/trades_app.go` — `tradeBuilderHandOff`:403, `checkTradeExitPending`:454 | 65 | `trades` | W:trades | Recompute saved plan preview; refuse blocks/out-of-scope receipt/nonpending exit/unsafe URL; save handoff before browser/event/wake; proposal skips pending-exit check. MOVED (part 4, batch 6). |
| 118 | 2 | `internal/composition/trades_app.go` — `tradePickReason`:470, `tradeClockReasons`:484 | 20 | `trades` | W:trades | FP_ token requires held fresh ownership; unreadable clock blocks with explicit reason; otherwise shared deadline rule.  MOVED (part 2, batch 5). |
| 119 | 3 | `internal/composition/week_app.go` — `refreshWeeksInBackground`:21, `refreshWeek`:56 | 55 | `control` | B | Hourly week→season→clock-rule schedules→finished scores→engine; re-launch league refresh on tick. Week failure preserves prior week/time with failure note; success may derive phase; always revalidate release and notify. MOVED (part 4, batch 2). |
| 120 | 1 | `internal/composition/week_app.go` — `lineupWeekFrom`:85 | 22 | `control` | B,startup | Fetch default week then advance until lineup week available; reject >18/mismatched requested week; final week18 ends with no lineup; wrap fetch/conversion errors unchanged.  MOVED (part 2, batch 1). |
| 121 | 3 | `internal/composition/week_app.go` — `seedCurrentWeek`:109 | 35 | `control` | startup | Apply identical lineup-week algorithm to archived schedules; require exactly one matching scoped result; log failure without replacing held state. MOVED (part 4, batch 2). |
| 122 | 4 | `internal/composition/week_app.go` — `deriveWeekPhase`:145 | 30 | `control` | B | Parse configured bounds, stop on invalid/season-over phase; only changed valid phase runs coordinator AdvancePhase with NFL-week/fetched evidence note. MOVED (part 4, batch 2). |
| 123 | 2 | `internal/composition/week_app.go` — `lastRegularWeek`:188, `windowSnapshot`:196, `timeNow`:215 | 26 | `control` | W:control,platform;B | Last-regular week must parse within 2–18 else zero; snapshot only rule weeks with detached matchups; injected clock else wall time.  MOVED (part 2, batch 5). |
| 124 | 3 | `internal/composition/shell_feeds_app.go` — `refreshShellFeeds`:63 | 25 | `control` | B | Pace calendar before trade bait; stop on pause cancellation; season/league-specific live fetch and reference archive for both daily feeds. MOVED (part 4, batch 2). |
| 125 | 3 | `internal/composition/now_app.go` — `refreshScoreFirsts`:42 | 13 | `now` | B,W:now | Read missing first scores from held archive; failed read logs and leaves old holder; publish only successful result under season lock. MOVED (part 3, batch 2). |
| 126 | 2 | `internal/composition/asset_board.go` — `assetInput`:120 | 9 | `market` | W:market,offers,trades | Hydrate Research before input capture; add every rostered mirror ID to board's value scope, even without finished score. MOVED (part 3, batch 3). |

## Wiring / representation adapters inspected but not counted as decision units

These functions construct/bind/forward, detach shared state, or map representations. Conditional
nil-preserving serialization, resource-error propagation and snapshot copying are not new
business policy. Re-evaluate them when the caller's service moves; do not add forwarding layers
just to make composition look small. Constants/type aliases and generated `wails_surface.gen.go`
are excluded from business-decision LOC. `command_bindings.go` is registry wiring only.

- `internal/composition/platform_host.go`: `CheckMoves`, `Clock`, `DraftRosterMove`, `HandOff`, `Moves`, `PhotosOff`, `PlayerName`, `Snapshot`, `WithdrawRoster`, `platformAPI`.
- `internal/composition/research_archive.go`: `hydrateResearch`, `researchFeedLabel`, `researchKinds`.
- `internal/composition/database_checks_app.go`: `databaseChecksReport`, `databaseFiles`.
- `internal/composition/control_host.go`: `KeyUnavailable`, `Now`, `RulebookMarkdown`, `ScoringConfig`, `SetWindow`, `Week`, `Zone`, `controlAPI`.
- `internal/composition/native_schedule_app.go`: `Now`, `Trusted`, `nativeWaiverClock`.
- `internal/composition/sources_runtime.go`: `compileFacts`, `countText`, `healthInput`, `loadSignals`, `pluralNoun`, `publishLabManifest`, `researchCrosswalkVersion`, `settingsPath`, `sourceStanding`, `wireSources`.
- `internal/composition/mfl_feed_cache.go`: `fallbackParent`, `leagueScheduleOrArchive`, `liveOrArchive`, `readArchivedFeed`, `sameExport`, `standingsOrArchive`.
- `internal/composition/native_draft_app.go`: `draftPickAction`.
- `internal/composition/report_reading.go`: `proseName`, `reportShell`, `reportTime`, `wireBoard`, `wireChoices`, `wireRange`, `wireReading`, `wireReport`, `wireRows`, `wireSlice`.
- `internal/composition/shell_feeds_app.go`: `shellFeedTime`.
- `internal/composition/offers_reading.go`: `OffersAssetValues`, `OffersBuildStamp`, `OffersDirectory`, `OffersEngineValues`, `OffersHistoryContext`, `OffersInjuries`, `OffersLab`, `OffersLog`, `OffersMarketFacts`, `OffersProvenance`, `OffersRosterTarget`, `OffersTradeBait`, `OffersWake`, `RefreshOffersPool`, `archivedOnly`, `counterProvenance`, `heldCutCharge`, `offersMarketFacts`, `poolMarketLines`.
- `internal/composition/draft_reports_app.go`: `DraftValues`, `draftReportValues`, `engineRunText`, `measureMeanings`.
- `internal/composition/cache_app.go`: `CacheScopes`, `sourcesAPI`.
- `internal/composition/commissioner_host.go`: `CloseFinishedDrafts`, `CommissionerAuthorization`, `CommissionerBackupNow`, `CommissionerContext`, `CommissionerNow`, `CommissionerPreviewRegistry`, `CommissionerPrincipal`, `CommissionerRecords`, `CommissionerSeat`, `CommissionerSeatSignIn`, `CommissionerZone`, `NativeFacts`, `NativeScheduleDates`, `NativeSeedCommand`, `RunNativeOwnerAction`, `TransferStatus`, `leagueRecords`, `newCommissionerAPI`.
- `internal/composition/native_lineup_app.go`: `PlayerFacts`, `lineupSetAction`.
- `internal/composition/run.go`: `bindings`.
- `internal/composition/market_feeds_app.go`: `buildMarketReports`, `heldMarket`, `heldMarketForWeek`, `marketArchiveScope`, `marketFacts`, `marketFactsAt`, `marketHeld`, `marketKinds`, `marketWeek`, `marketWeekText`, `wireFact`.
- `internal/composition/engine_refresh.go`: `runModelLocked`.
- `internal/composition/research_runtime.go`: `heldResearchInput`, `researchKey`, `researchStatus`.
- `internal/composition/freshness.go`: `liveFreshness`.
- `internal/composition/app.go`: `BrowserOpenURL`, `EventsEmit`, `NewApp`, `attachSeat`, `domReady`, `ready`.
- `internal/composition/pool_app.go`: `sortFreeAgentIDs`.
- `internal/composition/trades_app.go`: `TradeState`, `TradesBuildStamp`, `labelTradeReceipts`, `tradeCentsCorrection`, `tradeDeadlineReasons`, `tradeFigure`, `tradesAPI`.
- `internal/composition/power_rankings_app.go`: `defaultAgg`, `describeMeasures`, `loadSeasonPoints`, `powerRankings`, `wirePower`.
- `internal/composition/refresh_app.go`: `Fetch`.
- `internal/composition/archive_app.go`: `configureArchive`, `readCommsArchive`.
- `internal/composition/race_app.go`: `raceFeed`, `racePower`.
- `internal/composition/native_owner_app.go`: `bidAction`, `tradeAction`, `waiverAction`.
- `internal/composition/moves_app.go`: `wakeMoves`.
- `internal/composition/command_bindings.go`: `CommandRegistry`, `archiveCommandBindings`, `commissionerCommandBindings`, `commsCommandBindings`, `contractsCommandBindings`, `controlCommandBindings`, `counterpartiesCommandBindings`, `draftCommandBindings`, `marketCommandBindings`, `newCommandAPIs`, `nowCommandBindings`, `offersCommandBindings`, `platformCommandBindings`, `raceCommandBindings`, `registryInputs`, `researchCommandBindings`, `sourcesCommandBindings`, `tradesCommandBindings`.
- `internal/composition/moves_watcher.go`: `newMovesTimer`.
- `internal/composition/season_app.go`: `holdSeasonFeed`, `seasonPause`.
- `internal/composition/native_gate.go`: `contractActionName`.
- `internal/composition/contracts_host.go`: `AcquisitionCorrection`, `AcquisitionSnapshot`, `AcquisitionTeams`, `ContractsSeason`, `ReleaseTarget`, `RosterFeeds`, `RosterSnapshot`, `ValidateMoveURL`, `contracts`.
- `internal/composition/research_reading.go`: `HydrateResearch`, `OpenBrowser`, `Ready`, `RefreshResearch`, `RefreshResearchNews`, `ResearchDestination`, `ResearchNames`, `ResearchNews`, `ResearchSnapshot`, `feed`, `player`, `publishResearch`, `snapshotResearch`.
- `internal/composition/asset_board.go`: `AssetInput`, `LabStamp`, `assetBoard`, `buildAssetBoard`, `dynastyValueCells`, `newMarketAPI`, `onFieldNow`.
- `internal/composition/now_app.go`: `NowInput`, `heldFeed`, `heldMatchupStarters`, `liveGameReading`, `liveGameWindows`, `liveWeek`, `liveWeekUnplayed`, `nowAPI`, `nowFeed`, `possessive`, `seedLiveGames`.
- `internal/composition/week_app.go`: `fetchLineupWeek`, `weekClock`, `weekSnapshot`, `windowInputs`.
- `internal/composition/lab_app.go`: `lab`, `labStamp`.

## Core-style glue follow-up (not included in composition counts)

`internal/app/core` still contains 2,225 handwritten production LOC. Its `State` carries
locks/stores and many Host implementations; it is not the final transport-free service boundary.
Known follow-up: `host_control.go` SetLeagueWindow/setTradeWindow/setCalendarWindow,
`fallback.go` archive/live fallback decisions, `host_offers.go` own-franchise/target policy,
and host refresh/read callbacks for every feature. Audit those before claiming Part 2 complete.
Keep raw state/lock ownership stable while extracting ports; never let feature packages import
core or siblings to evade the existing depguard. Shared view contracts already live in
`internal/app/season` and wording in `internal/app/report`.

## Part 1 evidence / remaining work

- Frontend prerequisite: `pnpm -C frontend install && pnpm -C frontend build` passed
  (existing large-chunk warning; no frontend source/generated bindings changed).
- Moved `TestFinishedScoresDue` and its helper from composition into Now; added exact-cutoff
  cases for absent/empty/ambiguous schedules, final observation, latest mixed cutoff and future observation.
- Added Platform report cases for unchecked/healthy/error/integrity/FK findings and status
  precedence in both orders. Retained composition integration tests for real pool checking,
  serialized report field, Sources reading and joined worker shutdown.
- `go test -race ./internal/app/now ./internal/app/platform` passed in the initial batch.
- Initial broad composition package run hit the 200-second tool timeout without a result;
  it is **not** counted as passing. No remaining test process observed afterward.
- Targeted composition race tests passed: BackgroundFinishedScoresRereadsOnce,
  DatabaseChecksReportAllFilesAndFailures, DatabaseChecksWorkerJoinsOnShutdown,
  DataSourcesIncludesLocalDatabaseChecks, EveryCommandDoorRoleParity (including its negative cases).
- `go vet ./internal/app/now ./internal/app/platform ./internal/composition` passed.
- `internal/composition/command_matrix_test.go` untouched; no permission/role/binding/schema changes.
- First commit hook caught two newly unused composition forwarding helpers (`nflGames`,
  `weekKickedOff`); removed them rather than suppressing lint. No remaining callers.
- Final implementation rerun: Now/Platform package race tests, the five targeted composition
  race tests and vet all passed. Commit hooks passed golangci-lint, gitleaks and ifaceguard;
  `scripts/check-placement.sh` passed (126 paths / 17 features).
- Measured relocation: cutoff 25 baseline function LOC → 38-line Now production file;
  database report/wording 33 baseline function LOC → 47-line Platform production file plus
  retained composition capture. Removed two unused forwarding functions (6 LOC).
- Whole affected handwritten production set (`composition/{database_checks_app,finished_scores,now_app}.go`,
  `app/now/finished_scores.go`, `app/platform/database_checks.go`): 683 → 711 physical LOC,
  22,895 → 23,609 Unicode code points including LF. This grows by 28 LOC / 714 code points;
  the gain is two decisions with inward ownership, not source compression. Tests/docs excluded.
- No full `make verify`, push, PR, merge, GUI or live-data operations this part.

Estimated remaining one-hour parts: Part 2 **8–12** (including core audit, lock-aware lifecycle
moves and final gate); Part 3 **4–6** (registry-typed read/cache primitive, staged store migration,
read invalidation/event behavior, parity/regression checks). Estimates, not commitments;
frontend stores were not inventoried this part.

Owner ruling: none needed for these two moves. If a later move requires changing public bindings,
permissions, Raw validation, lock ordering or schema, stop and surface it rather than expanding scope.

Debt left: all open inventory units; core Host policy audit; composition ratchet implementation
and negative test; broad/final verification and live review; all shared frontend read-tool work.

## Part 2 — batch 1

Finding/proposal: pure opponent, score-period, move, roster admission, label and lineup-week
selection decisions still lived in composition. Move them into their existing feature packages;
composition captures held resources and forwards values/callbacks. Preserve error text, raw input
validation, lock spans, callback order, URL behavior and every command door. No new exposure.

- Units 3/5/20/21/22/51/81/97/120: 125 baseline function LOC relocated (unit 5 retains
  the archive adapter; unit 51 retains locked capture; unit 120 accepts the same read callback).
- Moved URL allow/deny and taxi registration tests to Platform, retained invalid-envelope
  construction integration in composition. Build-label test now tests values inward without
  mutating ldflags globals. Added opponent, period, personal-snapshot, lineup-week selection,
  move intent registry, watch/evidence/refresh and target URL checks.
- Frontend install/build passed (unchanged large-chunk warning); no frontend sources changed.
- Full Archive/Contracts/Control/Offers feature race suites passed; Archive took 148 seconds.
  Initial combined run failed compiling a new Platform test (wrong state constant), corrected.
  Initial move-evidence test fixture omitted required scope; corrected to validated restored
  envelopes with legal audits. Final targeted Platform/Control race tests passed.
- Targeted composition race tests passed for handoff, roster targets, week refresh, personal
  reads, held score periods, Archive/Contracts wire goldens and command-door parity.
- First lint found four newly cross-package error returns; preserve exact refusal text with
  narrowly explained wrapcheck annotations (no blanket suppression or changed error chains).

## Part 2 — batch 2

Finding/proposal: composition still selected alerts, finished-score scope, calendar deadlines,
local native schedules and scoring fallback columns. Move those decisions to feature functions
on published values and narrow archive/clock callbacks. Typed Control alert inputs avoid sibling
feature imports; composition converts Contracts/Trades representations without policy.

Preserved slice: composition takes each original lock and archive read, in original order →
Control applies lineup/trade/IR/bye eligibility, precedence and horizon policy → composition
still returns the same unsorted reading for the existing API to sort. IR target reads stay lazy,
only for known ineligible own players. Bye rules parse and failed-freshness choice are inward;
invalid rules still short-circuit the freshness callback. Other-week lock reads keep the same
archive URL and first-body/conversion policy; current-week locks do not read the archive.

- Eleven units, 346 baseline function LOC. Finished scores are shared by Research and the
  Market adapter without feature-to-feature imports. Market retains only conversion into its
  Scored values. Research snapshot enrichment (unit 66) is still open, not double-counted.
- Calendar merge preserves fallback titles, urgency/time/ID sorting, unknown-date behavior,
  provenance text and local-zone wording. Native schedule selection preserves first valid row
  per week and the existing compatibility allowance for old reference URLs carrying this league's L.
- Scoring age time is still sampled after directory I/O; missing directory/date stays NaN,
  schedule read failures omit the column, and parseable out-of-range season lengths stay accepted.
- Added inward cases for alert refusals/precedence, locks/horizon, IR dedup/ownership/name fallback,
  finished-week scope/counts/latest/zero scores/first-feed-only/detachment, calendar ordering,
  native schedule season/duplicates/invalid bodies/read errors, ages/schedule/season fallback,
  calendar-year discovery and franchise-key membership (not display-name availability).
- Final targeted feature race tests passed (Control, Research, Scoring, Native, Now).
  Targeted composition race batch passed (5.6 seconds): TargetAlerts, ByeAlert*, CalendarMerge*,
  DailyShell*, AssetBoardOnField*, PlayerForm*, ResearchWire*, MarketWire*, Race*, ScoringWire*,
  NativeSchedule*, EveryCommandDoorRoleParity*. Follow-up bye-rule inward extraction also passed
  all inward alert and composition TargetAlerts/ByeAlert tests.
- Test fixture corrections, not production changes: time.Time instant comparisons use Equal;
  native test now expects the pre-existing legacy-L URL acceptance; new Now test needs its own
  package declaration. No changed permissions, binding methods, schemas or role data.

## Part 2 — batch 3

Four further units (84 baseline function LOC): draft clock/date availability, read-once archive
markers and feed hydration, Research snapshot enrichment, build/config isolation. Platform owns
owner-only config directory creation/tightening and dev database names; ldflags variables remain
at their original composition symbols. Draft preserves no-store/no-clock/no-now refusal order.
Research accepts the existing shared cut-charge function as a port, rather than copying that
rule or importing a sibling feature; it owns default observation time and new enrichment maps.
The first-feed rows rule is now shared inward, with a compatibility forwarding helper for the
remaining composition adapters/tests. No business rule was forked.

- Moved standalone database filename test inward; retained config/log integration in composition.
  Added Platform mode-tightening/blocker-preservation/dev-stamp cases, Draft read-order/failure
  cases, Research time/enrichment isolation and archive read-once/live-error precedence cases.
- Targeted Platform/Draft/Research/composition race tests passed. Composition batch included
  DBFileName/ConfigDir, DraftWire, PlayerForm, ResearchWire/ResearchHeld and command-door parity.
- Initial new hydration fixture named a nonexistent Feed field; fixed fixture, not production.
  Lint found the deliberate historical 0750 test chmod; narrowly documented its purpose.
- Pure snapshot enrichment no longer lives in composition. Joint held capture/publication,
  archive resource ownership and the original locks still do.

## Part 2 — batch 4

Three archive units (121 baseline function LOC): weekly archive scoping, hydrate-missing/read-order
policy and latest-held merge/depth fallback policy. Research now owns them via ArchiveFeeds,
ArchiveSource and ArchiveInput. Composition still owns History lifetime, ResearchArchiveMu,
SeasonMu, LookupMu, the exact capture/re-read/publication spans and the crosswalk read callback.
Hydration never fetches; merge runs on the latest detached capture so live feeds/depth/injuries/news
win over archive copies. Existing per-row local-date clock sampling and error precedence remain.
Season/History are startup-owned; reading those into the service source does not introduce a
mutable-season capture (traced: production Season assignment occurs only at startup).

- Added inward scope-selection, complete hydration call-order/absent-read markers/injury-pending,
  news-failure handling, retry suppression, depth live-error preservation, crosswalk-version
  refresh, latest-live-wins and other-held-state preservation tests. Existing real-fixture
  cold-open, full Research refresh, concurrent read/news and joint-capture composition tests pass.
- HydrateArchive returns newly archived news, not a replacement for held live news. Its test now
  exercises the real merge before the next read, rather than incorrectly treating that output
  as already published. Removed an unused wrapcheck annotation caught by lint.
- make lint caught tiny-file growth (33 versus 24). Fixed the footprint, not the baseline:
  pure helpers joined their existing responsibility files (table below); startup stamp wiring
  joined app.go (ldflags package symbols unchanged); shared archive adapter joined sources_runtime.go,
  beside its existing archive compile/warm work. Offers' existing API now calls the same personal
  snapshot admission function instead of retaining an identical copy of the rule.

Current responsibility homes after consolidation (tests keep their descriptive file names):

| Previous file | Current home |
|---|---|
| app/archive/opponent.go | internal/app/archive/board.go |
| app/contracts/score_period.go | internal/app/contracts/roster.go |
| app/control/lineup_week.go | internal/app/control/clock.go |
| app/offers/snapshot.go | internal/app/offers/personal_api.go |
| app/now/finished_scores.go | internal/app/now/nfl_reading.go |
| app/scoring/context.go | internal/app/scoring/power.go |
| app/commissioner/native/schedules.go | internal/app/commissioner/native/season.go |
| composition/archive_app.go | internal/composition/sources_runtime.go |
| composition/version.go | internal/composition/app.go |

All previous-file entries in this table are relative to internal/. Inventory baseline paths and
Part 1 measurement sets above are historical references, not stale current placement claims.

## Part 2 — four-batch checkpoint (pushed before batch 5)

- All 27 moved units this part total **676 baseline function LOC**. Cumulative: 29/126 units,
  734/4,534 baseline function LOC; all band 1 units done. Remaining composition scope: 97 units,
  3,800 baseline function LOC, plus the separately identified core host decision work.
- Full `make lint` passed after the cohesion/placement fixes: ifaceguard, file-length report,
  bloat (comment share 5%, provenance 0, tiny files 24/24, duplication 0), placement (126 paths /
  17 features), root ratchet, frontend transport guard and golangci-lint (zero issues).
  Only existing >600-line files reported: internal/leaguerules/catalogue.go and internal/facts/columns.go;
  neither was changed. No baseline was raised.
- Final affected-feature race batch passed for Archive/Contracts/Control/Draft/Native/Now/Offers/
  Platform/Research/Scoring, filtered to the extracted responsibilities and their regression cases.
- Final composition race batch passed in 12.3 seconds, filtered to all moved callers and relevant
  wire goldens, refresh, cancellation, archive/live publication, Research concurrent/joint capture,
  native schedule lifecycle and EveryCommandDoorRoleParity. No permission metadata changed.
- Whole affected handwritten production union against `98ea8cf7`: **39 paths**, **6,871 → 7,098
  physical LOC**, **247,784 → 257,100 Unicode code points including LF** (+227 LOC / +9,316 code
  points). Tests/docs/generated output excluded. This is relocation/ownership, not compression.
- Prior broad composition timeout is still not diagnosed by these passing filtered runs; the
  full Archive feature suite took 148 seconds earlier, so large fixture work is a plausible cost,
  not a proven explanation for that separate timeout. Do not label it a deadlock without a stack.

Estimated remaining one-hour parts: Part 2 **8–10**, Part 3 **4–6** (still not started).
Owner ruling: none. No changed role/permission/binding/schema/lock ordering, live MFL fetch,
GUI, production profile, PR or merge. No new package or dependency.
Debt left: remaining 97 composition units; core decision audit and extraction; composition/core
negative ratchet; full hook/CI and live review; shared frontend read/cache tool and store migration.

## Part 2 — batch 5 / last implementation batch

Four further joint-read units (68 baseline function LOC): model free-agent ID scope, native
held-directory/injury availability, trade pick/clock evidence refusals, configured-week capture
and injected/wall clock selection. Each original lock/read/time-sampling span stays in composition;
feature functions own the availability/range/scope choices. Leading zeros, duplicates, nil/empty
semantics and clock-read failure ordering stay unchanged. Native directory transformation still
runs under LookupMu, and injury state is read afterward, only when observed as held.

- Added inward tests for model ID order/duplicates/nil, loaded native facts/lazy injury callbacks,
  pick-only ownership reads and missing-date precedence, failed-clock refusal without sampling
  time, configured-week selection/deep slice detachment, last-regular-week bounds and clock fallback.
- Targeted feature race batch passed (Scoring, Native, Trades, Control); composition race batch
  passed (94.6 seconds) for actual model refresh/read, native legality/lineup, trade deadline,
  configured windows and command-door parity. This slower filtered batch demonstrates real
  expensive fixture/model work, but still does not identify the prior broad timeout's exact cause.
- Full make lint passed on this candidate: all guards, bloat 24/24, zero lint issues.
- The four-batch checkpoint e101f00f was pushed successfully; its pre-push changed-package lint,
  shuffled tests/importer tests and binding generation passed. No PR was opened.

Final relocation total for this session: **31 units / 744 baseline function LOC**. Cumulative:
**33/126 units / 802/4,534 baseline function LOC**. Whole affected production union against
`98ea8cf7`: **44 paths**, **7,405 → 7,671 physical LOC**,
**267,190 → 277,927 Unicode code points including LF**. Tests/docs/generated output
excluded. Relocation is not savings; footprint grows by **266 LOC / 10,737 code points**.

Next owner verification remains live review on a disposable profile: same Control alert labels,
Research/Market finished-week values and trade deadline refusals before/after refresh. No live
GUI/MFL/prod-profile operation was performed here. Remaining work is the Resume list plus the
other open inventory rows, core host policy/ratchet work and all Part 3 frontend read-tool work.

## Part 3 — batch 1: native owner boundary

Finding: native owner gate/permit checks, command-family selection, lineup/pick construction,
bid-season evidence and owner-zone policy remained in composition. Proposal/preserved slice:
composition retains original locks, readiness callback, identity and bounded History adapters;
Native now owns gate order and refusal identity, permit matching, command construction and
held schedule/bidding decisions. Ledger validation and transaction authorization remain unchanged.
Seven units (33–38, 40), 211 baseline function LOC. No binding/schema/role changes.
Targeted Native/composition race batch passed (50.4 seconds), including command-door parity,
all registered gate actions, lineup recording/restart, pick, owner waiver/bid admission and bid
season/playoff/offseason cases. New inward tests cover refusal ordering, no execution on bad
family/permit, lazy identity, best-effort draft closing, held locks and local-zone fallback.

## Part 3 — batch 2: Now refresh and live-game boundary

Nine units (42–45, 47–49, 58, 125), 291 baseline function LOC. Now owns schedule/bye
freshness and pacing, archived clock-week seeding, requested-week admission, first-score
failure/publication policy, live week reset/follow admission, poll backoff and spine/box evidence
selection. Composition retains original joint captures, lock spans, network/store constructors,
read/publish callbacks and per-player photo-off sampling. Race consumes the shared Now live overlay
function; no sibling feature import was introduced. No public door, permission or schema changed.

Targeted Now and composition race batch passed (13.7 / 8.4 seconds), including Now held-refresh,
concurrent/joint snapshot, Race goldens/capture/refresh, live/pregame wire, held failure and box
identity, and command-door parity. New inward tests pass for exact freshness cutoffs, force,
canceled pacing, MFL exponential backoff/cap/reset, lazy initialization/failure, week reset and
transport retention, invalid-follow refusal, detail failure retaining board, spine early exits,
week scope, race overlay and first-score failure preserving held data.

## Part 3 — batch 3: joint reads and Sources/Control writes

Nine units (11, 54, 84, 101, 103, 104, 108, 111, 126), 213 baseline function LOC.
Control owns ordered alert acquisition/assembly, Accounting admission and parameter reads/override
results. Startup-dependent resources remain lazy ports: readiness finishes before capturing the
parameter store/context or Accounting host/season. Sources owns settings persistence and the
ordered held-copy catalogue/dispatch; composition supplies each existing feature's refresh.
Archive owns verified-only cache replacement under the caller's original FactsMu. Market owns
pool indexing/pick model fallback, Trades owns held-state read order and failed-board value refusal,
and Research owns unscored roster-ID scope. Cross-feature hydration/conversion stays in composition.
No sibling imports, permission/schema changes or new dependency/package.

Added inward tests: settings defaults/roundtrip/0600/malformed/blocked-temp preservation; cache
order/unknown no-effects/forced selected error; parameter startup/read/value/write deadline/error;
Accounting startup/unknown-host no-open; alert order/single clock; pool leading-zero IDs/dedup,
missing pick model; roster scope order/detachment/empty; archive empty/error retain-good;
trade selection failure avoids pending read; failed-board assets remain unvalued. Targeted feature
race tests and composition alert/value/trade landing/cache/parameter/parity batch passed.

### Part 3 footprint consolidation

After batch 3, bloat caught five newly tiny composition adapters (29 versus baseline 24).
Consolidated by responsibility without raising the baseline; tests keep descriptive names:

| Old path | Current home |
|---|---|
| internal/composition/native_owner_app.go | internal/composition/native_gate.go |
| internal/composition/native_draft_app.go | internal/composition/native_gate.go |
| internal/composition/native_lineup_app.go | internal/composition/native_gate.go |
| internal/composition/native_schedule_app.go | internal/composition/commissioner_host.go |
| internal/composition/cache_app.go | internal/composition/sources_runtime.go |

These are adapter-only relocations; inventory paths remain historical baseline references.

## Part 3 — batch 4: background model and market/pool refresh

Seven units (13–15, 18–19, 52–53), 202 baseline function LOC. Scoring owns model admission,
input ordering, due/evidence checks, dirty consumption/restoration and completion wording/events.
Original model/refresh mutex spans and dirty atomics remain caller-owned ports; new inward tests
exercise success/failure and changes arriving during a run. Market owns per-kind daily/weekly
refresh admission, pacing, archive projection-week validation and wrong-week clearing before
publication. Pool owns archived first-good restore, sorted/normalized IDs, changed-list dirtiness,
last-good retention and fresh-pool Market-only refresh. No new transport, authority or schema.

Initial filtered composition batch found the old source-text order test still looking for calls
now owned inward. Kept worker/startup ordering assertions and adapter-port wiring assertions;
new inward runtime tests now prove actual execution order rather than source spelling. Other
new tests cover model early failure and health event, pool scope/no-fetch-on-held, free-agent
failure preservation/dedup, freshness/force paths and Market wrong-week failure behavior.

Batch 4 consolidation: `internal/composition/freshness.go` →
`internal/composition/mfl_feed_cache.go`, beside its live/archive read adapters; the last unused
SameExport forwarding helper was removed. No rule was copied and no ratchet baseline changed.

## Part 3 — batch 5: native seed and Race refresh

Three units (41, 56–57), 147 baseline function LOC. Commissioner owns retry identity and lazy
seed capture; Native owns held-store/season/date admission and seed construction/validation.
Composition still holds RefreshMu and the original separate season/target snapshot captures;
invalid or retry requests never acquire later inputs. Race owns week→bracket→standing publication
order, cancellation, success-only power replacement and the one lazy schedule shared by both
power views. Bracket reads preserve all-or-nothing errors and last-nonlive freshness precedence.
The generic URL-parameter archive filter lives in shared Report rather than introducing a sibling
feature dependency; live/archive transport and permission checks are unchanged.

Added inward tests for retry mismatch/no-recapture/error/new-command fields, missing/mixed/bad-date
seed early exits, Race read/publication order and one schedule, failed standings no power replace,
bracket aggregate freshness/partial refusal/cancellation, and scoped archive acceptance.
Feature and filtered composition race batches passed (6.2 seconds composition), including real
seed capture/retry/missing-input tests, Race goldens/concurrent/joint snapshots and command-door
parity. The single-result power representation adapter is production code; its map fixture adapter
now lives with Race's composition tests. No public wire, permission, schema or dependency change.

## Part 3 — final implementation checkpoint

- This part: **35/126 units, 1,064 baseline function LOC**. Cumulative **68/126,
  1,866/4,534 LOC**; remaining **58 units / 2,668 LOC**, plus core policy/ratchet work.
- Whole affected handwritten production union against `1b0e5059`: **42 paths**, **5,523 →
  6,037 physical LOC**, **205,021 → 228,273 Unicode code points including LF**: growth of
  **514 LOC / 23,252 code points**, not compression. Tests separately: **20 paths**, **1,432 →
  2,417 LOC** (+985). Generated/frontend/docs excluded from both measurements.
- Final affected-feature race batch passed in all 13 packages: Commissioner, Native, Now,
  Control, Market, Pool, Race, Report, Research, Scoring, Sources, Trades and Archive; filtered
  to all newly extracted responsibilities and regression cases. Final composition race batch:
  **85 test/subtest results passed, 160.776 seconds**, including command-door parity, real native
  owner/seed writes, Now/Race concurrent snapshots, pool/market publication, source events,
  cache wiring and real trade-builder ownership/deadline/handoff paths.
- Timeout evidence: that filtered run's slowest individual test was
  `TestTradeBuilderDeadlineOwnershipPersistenceAndSafeHandoff` (**106.63 s**); other real trade
  fixture tests took 22.54–36.82 s, registered gate test 26.96 s. These are measured expensive
  integration paths, not a demonstrated deadlock. The earlier broad run's exact timeout cause
  is still unknown; do not claim these timings prove its full explanation. Keep filtered runs.
- Full `make lint` passed on the final implementation: ifaceguard, placement (126 paths /
  17 features), root/frontend transport ratchets, bloat (comment 5%, provenance 0, tiny 24/24,
  duplication 0) and golangci-lint (zero issues). The two pre-existing >600-line files were
  untouched. A separate attempt and one commit hook collided with another worktree's linter;
  retried normally after it released the lock, no bypass or baseline change.
- Review found the lazy live adapter must retain an existing Finals port when History is absent.
  Preserved it and added `TestLiveAdapterKeepsExistingFinalsWithoutHistory`. The moved gate
  sentinel now has one owner inward; removed the composition global alias rather than adding
  a new global. New test references the same service sentinel and still checks error identity.
- Two one-at-a-time mutation checks were rejected: dirty restoration set false after failed
  model run; lazy adapter discarded existing Finals. Sources were restored before the final
  passing batches. No mutant was committed. Native family execution lost an unnecessary
  curried forwarding closure during review; permit/refusal behavior is unchanged.
- Frontend prerequisite install/build passed (existing large-chunk warning); no frontend,
  generated binding, command registry, command bindings or command-door parity test changes.
  No live MFL, GUI, production-profile operation, PR or merge. Push/hook evidence, if run,
  belongs in the next checkpoint; not claimed here.

Estimated remaining one-hour parts: Part 2 **6–8**, including core extraction/ratchet and final
verification; frontend plan Part 3 **4–6**, not started and still not inventoried this part.
Owner ruling: none. Christopher's next live verification on a disposable profile: unchanged
Now refresh/week selection and held game detail, native closed/open owner actions, Sources
read-again scopes and saved trade deadline refusals/handoff. Passing means same labels/refusals
and held-data retention, no network or writes caused by a refused action.

Debt left: 58 open composition units; core Host policy audit/extraction; negative composition/core
ratchet; final full CI and disposable-profile live review; all registry-driven frontend read/cache
work and store migration. No permission/role/binding/schema or lock-order change was needed.

### Part 3 relocation ledger (baseline decision-function LOC, adapters retained)

Paths below are relative to `internal/`; named groups include the contributing files.

| Source in composition | Service home in app | Units | Baseline LOC |
|---|---|---:|---:|
| native_gate, native_owner_app, native_draft_app, native_lineup_app, native_schedule_app, native_seed_app | commissioner/native/{owner_actions,season,seed}.go; commissioner/native_seed_api.go | 8 | 265 |
| now_app.go | now/{live_runtime,firsts}.go | 8 | 276 |
| race_app.go | race/refresh.go; now/live_runtime.go; report/held.go | 3 | 108 |
| control_alerts.go; control_host.go | control/{alert_reading,reading,parameters}.go | 3 | 87 |
| engine_refresh.go | scoring/model.go | 3 | 65 |
| market_feeds_app.go | market/reports.go | 2 | 57 |
| pool_app.go | offers/pool/refresh.go; market/assets.go | 3 | 94 |
| sources_runtime.go; cache_app.go | sources/settings.go | 2 | 56 |
| sources_runtime.go (baseline archive_app.go) | archive/read.go | 1 | 13 |
| trades_app.go | trades/builder.go; market/assets.go | 1 | 34 |
| asset_board.go | research/reading.go | 1 | 9 |

Read-only final source trace checked readiness before lazy store/context/host capture, exact
model defer order (cancel → health event → unlock), conditional dirty Load and failure restoration,
latest-held re-read before publication, archive scope and last-good retention, per-player photo
switch sampling, lazy live transport/finals, native permit and ledger revalidation, and Race's
shared lazy schedule. Core Season/History and SeasonChanged are startup-owned in production.
All five feature API constructors used by cache wiring and leagueservice.New are allocation-only;
eagerly forming their refresh callbacks does not perform admission, I/O or a write. No additional
behavior defect was found after the Finals retention correction. This is source trace plus the
named exercised evidence, not a substitute for the deferred live review.

### Push checkpoint

`47f12f24` and all six commits this part were pushed to
`origin/session/revamp-B-app-boundary`. The real pre-push hook passed binding generation,
full lint and shuffled changed-package/importer tests. No generated/tracked build change remained.
Git status was clean and local/remote branch refs matched. No PR or merge.
Full CI for `47f12f24` **passed**: verify run `40415875098` (checks, all four race
shards, release-tools and aggregate verify) and desktop packages run `40415875107`
(Linux/Windows/macOS packages and installation jobs). These are automated package/install checks,
not Christopher's deferred live feature review. The closing commit after this checkpoint is docs-only.

Full composition package timings from those CI race jobs: **194.260, 261.870, 411.862 and
478.298 seconds**; Archive took **287.340 s** in one shard. This establishes that a 200-second
tool timeout can cut off legitimate passing fixture work. Together with the locally measured
106.63-second real trade-builder test, the evidence supports an undersized broad-run budget,
not a demonstrated deadlock. It does not reconstruct the exact earlier timed-out process.

Next batch's first concrete slice: `finished_scores.go` unit 17 and `offers_reading.go` unit 50,
then Research units 67–71. Preserve cancellation/pacing, first-finish pinning and the original
latest-held re-read/publication spans; callbacks must not accidentally capture startup-dependent
stores before readiness or replace live data with a stale detached capture. After that, move
local plan save/handoff workflows with their existing ownership checks and save-before-browser order.

## Part 4 — batch 1: finished scores, Offers history and Research

Finding/proposal: refresh admission, ordered reads, first-finish pinning, news concurrency/scope
and depth fallback still lived in composition. Move those rules inward; composition retains
RefreshMu/NewsMu/SeasonMu and the latest-held publication spans, actual network/archive wiring,
and startup-dependent resources. Finished scores capture the held score timestamp and schedule
under the same original lock; first finish merges against current held state. News publishes
partial results even on read failure and refuses concurrent/scope-invalid work before capture.
Seven units (17, 50, 67–71), 243 baseline function LOC. Shared pause selection/cancellation now
has its single owner in Report; Core keeps a compatibility forwarder. No public/schema/role change.

Targeted feature and composition race batch passed: inward pin/cancellation, feed last-good and
force, depth links/error retention, ordered refresh/cancellation, lazy transport preparation,
news admission/failure publication and Offers read/save-error separation; existing real Research
refresh/news concurrency, finished-score reread, pending history/save-failure and door parity.
Initial compile errors (name collision with existing FinishedScores and unused context import)
were corrected before the passing batch. No skipped test is counted as passing.
Frontend prerequisite install/build passed (existing large-chunk warning). Pull --ff-only failed
because this stacked branch tracks origin/main; no merge/rebase/upstream rewrite performed.

## Part 4 — batch 2: seasonal Control refresh and week policy

Ten units (72–77, 119, 121–122, 124), 518 baseline function LOC. Control owns fixed refresh
ordering, cancellation and early lineup notification; injury/transaction conversion and held
injury views; requested-lineup week admission/reset; key-first pending reads, generation fencing,
key failure wording and separate history-save logging; daily shell archive/attempt admission;
bounded weekly worker scheduling, week publication/phase derivation and held-week seed admission.
Composition retains exactly the original RefreshMu, KeyMu/SeasonMu and WeekMu spans, tickers,
worker joining and raw transport/archive adapters. A stale key generation never enters SeasonMu.
No network with no key, no stale generation republish, and no history failure relabeled as MFL loss.

New inward order/cancellation, no-key/no-fetch, stale-generation/no-publication, key failure
retention, daily retry/last-good, week reset/phase refusal, worker schedule and absent/ambiguous
seed tests pass. Filtered Control/composition race batch passed (11.3 seconds composition),
including key deletion during fetch, shutdown cancellation, real refresh order, lineup archive
isolation, weekly refresh transitions and command-door parity. Initial worker test incorrectly
assumed no windows for missing regular-season bounds; corrected to preserve the fixed Week 9
trade deadline. Removed the now-unused dailyShellReading composition forwarder.

## Part 4 — batch 3: local move actions and evidence watcher

Eight units (23–26, 29–32), 368 baseline function LOC. Platform owns ownership/preflight and
save-before-browser handoff/withdrawal, predicate-specific facts and unknown-intent logging,
refresh-before-locked-reread evidence observation, source order/pacing/recent evidence, ready-plan
verdict/rebase, wake admission/audit wording and restart/fast-poll scheduling. Store ports expose
only the local envelope ledger; they never write MFL. Composition keeps startup/context/resource
admission and supplies the original locks, cross-feature roster/season observation and release/
contract adapters. Refused ownership or failed persistence causes no browser/event/wake effect.

Added inward real-envelope/memory-ledger tests for save-before-effects/failed-save retention,
ownership refusal, withdrawal effects, declared source order/contract dedup/spacing cancellation,
post-handoff recency, facts failure/unknown once, unknown eligibility retention and exact
Invalidate→Rebase route, fast-mode cutoff and ready-gated wake. Existing real-store composition
handoff/watch/revalidation/cancellation/parity race batch passed (3.6 seconds composition).
An invalid unknown-intent fixture omitted its required player/status; corrected the fixture.

Bloat found the shortened watcher adapter became tiny; consolidated by responsibility:
internal/composition/moves_watcher.go → internal/composition/moves_app.go. Updated placement/map;
removed unused source/fast-mode forwarders, without changing a baseline. Inventory paths are
historical references. Unit 27 observation assembly and unit 28 cross-feature plan revalidation
remain open, not counted as moved.

## Part 4 — batch 4: startup evidence, identity and logs

Five units (83, 85, 89, 94, 106), 142 baseline function LOC. Sources owns imported-only startup
compile/warm/fit ordering and publish-only-normalized directory caching; Market owns verified-only
model fit and success-only publication/event; Commissioner owns stable local identity and local
seat read/sign-in/re-read admission; Platform owns private per-launch logging/retention.
All resource captures remain lazy across readiness, especially the local seat created by startup.
FactsMu still spans archive read and both success/failure logging. Logging installation remains
an App startup adapter; source file internal/composition/logging.go is now internal/app/platform/logging.go.
Moved the standalone log-retention test inward; composition keeps config/log permissions integration.

New inward startup short-circuit/order, directory failed-fetch/failed-normalize preservation/cache,
verified-fit failure/no-publication/order, identity UID isolation/lazy fallback and sign-in/re-read/
startup-refusal tests pass. Feature/composition filtered race batch passed (1.8 seconds composition),
including real local identity, logging permissions/retention, Sources lock and door parity. The first
build found a callback type mismatch (Model interface versus LabModel); corrected the service port.

## Part 4 — batch 5: held snapshots and plan revalidation

Five units (27–28, 79–80, 82), 247 baseline function LOC. Platform owns roster confirmation
freshness, ordered snapshot acquisition/success-only publication, directory provenance and
first-use injuries, roster-draft admission/IR-versus-taxi envelope construction/rail filtering,
and release→contract→eligible-roster plan revalidation. Control owns ready/state-gated held clock
construction and calendar merge. Composition retains the original resource/lock/context spans,
including Build using the app context (not the bounded directory context), lookup timestamp
sampling under LookupMu, and save/notification under MovesMu. Cross-feature contract evidence,
season conversion and eligibility snapshot construction remain representation/resource adapters.

Added inward capture failure/no-publish, directory original-time/archive/failure provenance,
injury lazy/bounded refresh, roster authorization/player/snapshot early refusal, rail filtering,
confirmed-versus-failed roster evidence, candidate admission/revalidation order and held-clock
startup/state/merge tests. Filtered Platform/Control/Research/composition race batch passed
(3.4 seconds composition), including real snapshot cache/archive, roster writes/restart,
concurrent draft, eligibility rebasing, weekly clock and door parity. Full make lint passed:
all guards, bloat 24/24 and zero golangci-lint issues. Source review also preserved news pause
selection before the held news capture, matching its original sampling order.

## Part 4 — batch 6: trade plan lifecycle and isolated projection (usage-limit resume)

Eight units (109–110, 112–117), 395 baseline function LOC. Trades owns accept/propose/exit
construction, save/supersession, saved-plan admission, builder dispatch, removal, recomputed
handoff refusals and save-before-browser ordering. Simulation/Projection own session selection,
isolated generations, currently-owned asset merge, stale baseline/reset reasons, leg admission,
local transaction and rich cap/roster projection. Composition retains what-if→refresh and MovesMu
spans plus cross-feature WhatIf, asset, clock, authorization and URL ports. No authoritative
mirror mutation, remote submission, permission/binding/schema change or new package/dependency.

The resume began with uncommitted plans.go and trades_app.go from the interrupted run; completed
and source-traced them against HEAD before testing. Added inward save-failure/order, missing/wrong
exit direction, blocked/stale handoff, receipt persistence before browser/event/wake, owner refusal,
partial builder output and simulation/projection early-error tests. Full Trades race suite passed;
filtered real composition trade batch plus command-door parity passed (130.844 seconds), including
exact cents, private-counterparty isolation, IR/taxi/removed assets, pick-only/next season, local
removal/supersession, wire goldens and safe handoff. Existing integration tests remain outward;
new tests exercise the inward workflows rather than duplicating the real DB fixture harness.

Frontend install/build passed (existing chunk warning). git pull --ff-only again refused divergent
origin/main tracking; no merge/rebase/upstream change. Initial lint found misplaced/now-unused
wrapcheck annotations and import grouping; corrected, affected-package golangci-lint passed.
No skipped test counted as passing. Timeout evidence remains the prior CI 194–478-second passing
composition runs and locally measured expensive fixtures, not a demonstrated deadlock.

## Part 4 — batch 7: atomic league refresh and credentials

Three units (60–61, 105), 206 baseline function LOC. Control owns ordered discovery, roster/money
normalization, staged rule/mirror commit and failure release, success-only checked/dirty publication,
post-refresh revalidation, ready-gated worker dispatch and launch-result wording. KeyWorkflow owns
trim/length admission, verify-before-store, refusal classification, lazy startup store capture,
identity lookup failure, successful verification publication and delete-before-clear/refresh.
Composition retains RefreshMu, WeekStart/WorkerMu cancellation/admission, KeyMu and the nested
SeasonMu clear span, concrete network/keyring adapters and generation/identity state assignments.

New inward readiness/no-early-capture, rejection/unreachable/no-store, exact verify→store→lookup→
publish→refresh, delete removal, fail-closed identity, stage release, failed-commit/no-publication,
unchanged/no-dirty and worker/launch order tests pass. Filtered composition race batch passes
(3.482 seconds), including key leak/panic/season gates, atomic two-store rollback, manual failure
health event, Control golden/concurrent reads, worker shutdown and door parity. The source-wiring
test still asserted the pre-batch-2 weekly function body; changed it to assert the actual service
port wiring, retaining inward runtime WeekWorker order coverage. Affected-package lint passed.
No role, binding, schema, authorization, lock-order or remote-write change.

## Part 4 — batch 8: routine checks and matched backups

Two units (87–88), 93 baseline function LOC. Platform owns absent/canceled database admission,
ready-gated bounded check→backup cycles, build-matched backup schema configuration, daily capture
before retention and manual sign-in/configuration admission. Public backup failure now has its
single sentinel owner inward; technical causes remain private logs. Composition retains once/
worker joining, live resource capture, commissioner result conversion and the existing sign-in
refusal identity supplied as a port; no sibling service import or new global alias.

Added inward no-sign-in/no-capture, missing configuration, private failure/no-retention, bounded
manual capture, basename/public wording, best-effort retention, failed daily capture/no-prune,
ready refusal and same-context ordered maintenance tests. Platform and filtered composition race
batches passed (4.579 seconds composition), including actual daily capture, before-migration
matched backup, manual restore into a never-opened profile, joined checks/shutdown and door parity.
No schema version, backup retention, authorization or public response change.
