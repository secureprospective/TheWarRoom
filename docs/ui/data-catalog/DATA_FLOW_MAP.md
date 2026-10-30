# DATA FLOW MAP — approved mocks → data (2026-10-30)

Input: `claude-catalog-2026-10-30.md` (rows + SOURCE MAP D01–D78), Terra per-screen catalogs, origin/main 364379ba.
The approved mocks are fixed (HQ v10, War Room v6, Pulse v14, Home v5, Control Room v1). This map says what feeds each element, who may see it, and the order to build it in.

## 0. Verification pass (this map)
- Opened every package claimed EXISTS on origin/main; all real: engine/race, powerrankings, app/season, ingestion/livegame, app/now, app/control, transactions/deadcap, app/offers, store/personal, app/market, governance, leagueclock, app/draft, app/archive, app/report, serverstatus, commandcatalog, playerrecord.
- **Corrections to the source map:**
  - D38 Watchlist and D45 Private notes: NEW → **PARTIAL**. `store/personal` already stores kinds `watch | bait | note | draft` per franchise ("the GM's private lists, separate from MFL's mirror"). It needs lanes/stars (D38) and per-player note threads (D45), not a new store.
  - D77 Rookie slot salary: ? → **EXISTS**. `engine/rookiedraft/setup.go` slotSalary over `leaguerules.SlotPrice`; returns "needs a ruling" when the league hasn't priced slots, so the tile must show that state.
- **Isolation finding:** `store/personal` is keyed by franchise (`Mine`), not by account. Fine for one owner per team; co-owners would share one private space. Logged as Q3 below.

## 1. Isolation classes (every domain gets exactly one)
| Class | Who reads | Who writes | Rule |
|---|---|---|---|
| **PUB** league-public | every member | engines/feeds only | Same bytes for every viewer. Cache once per league. |
| **LEDGER** public record | every member (filtered by Audience) | commands only, via ledger | Append-only. Actor + "via AI" stamped. Commish corrections are new rows, never edits. |
| **PRIV** per-franchise private | that franchise only | that franchise | Never in a league cache, never in a commish view, never in search, never in exports other than its owner's. |
| **SEALED** private until an event | the owner + commish office (Q4); everyone at the reveal event | the owner | Becomes PUB/LEDGER at the reveal (window close, trade accept). Bids are SEALED. |
| **CMSH** commissioner office | permission-set holders | commish office | Gated by the single `mayRun` check. Never leaks into owner screens except via ledger. |
| **OPS** server operator | server-admin (R10) | server | League data never flows here, only health. |
| **DERIV-PRIV** | owner | computed | Model output run on PRIV input (e.g. a lens applied to *my* plan). Same isolation as PRIV; compute per request, never cache league-wide. |

The rule that matters most: **a PUB model must never read a PRIV or SEALED input.** For example, team needs (D27) must not read my watchlist, and bid-market rivals (D35) must not count sealed bids before the reveal.

## 2. Spine — the shared trunks (build once, everything reads them)
```
T1 League identity/clock  D01, D40 ─────────────────────────────┐
T2 Roster + contracts     D17, D18, D78 ───┐                    │
T3 Ledger                 D22, D23 ────────┤ (every command writes T3; T2 is
                                            │  the projection of T3)            
T4 Player record          D12, D14, D16, D55, playerid ─────────┤
T5 Live feed              D07 (truth ladder ESPN→nflverse)      │
T6 Valuation              D24 (value is f(player record)) ──────┘
```
T1–T6 are all EXISTS or near. Everything new hangs off them. No new screen may invent its own copy of a trunk.

## 3. Feed chains (what feeds the next)
**C1 Cap chain:** T2 contracts + D19 dead cap → **D20 cap runway** (NEW aggregator) → HQ cap strip, WR trade cap check, Home cap tile, D39 plan checks, D36 bid ≤ cap.
- D20 is the single source of the phrase "room". The bid check, the trade check and the plan check all call it. Today's "caller judges cap" (`nativecheck/bids.go:186`) gets its caller here.

**C2 Value chain:** T4 record → D24 value → **D26 age curve** → **D25 Best 21** (value is value, any position) → D05 power DYNASTY horizon → D28 posture lens → D27 team needs/surplus → D32 tune-it (TRIM/ASK/SWEETEN) → WR shelf/verdict.
- D26 is upstream of D25, so the age curve lands first.

**C3 Contend chain:** D17 lineup + D11 projections → powerrankings.Lineup/ExpectedPoints → D05 power (FORM/LINEUP/HEALTH/ALL-PLAY/RUN-IN) → WinProbability → **D06 playoff sim** (odds, magic number, bye %, "this week moves it") → LP race, Home race tile, HQ standing.
- D02 seeding + D03 tiebreak narrative feed the sim's tie resolution. The sim must call `engine/race` Seeding, never re-implement it.

**C4 Live chain:** T5 livegame scoreboard → D08 play-by-play (PARTIAL) → D09 fantasy box per slot (scoring rules × stat deltas) → LP live strip, matchup, HQ "playing now", Home live tile → D59 ticker → (post-final) D51 stat changes → D12 points.
- Corrections flow backward: nflverse final ≠ held → CompareAlerts → D51 → CR stat queue → T3 ledger.

**C5 Market chain:** D34 pool + bids (SEALED) → D36 cap gate (C1) → window close → reveal → T3 ledger → T2 roster → **D35 bid-market model** (PUB, built from *revealed* history only: recent winning bids, est. win price, trend, win odds, rivals = owners with room + need).
- "Rivals" is computed from public needs + cap, **not** from sealed bids.

**C6 Trade chain:** D29 builder versions (PRIV) → D33 lineup delta + C1 cap check + C2 lens (DERIV-PRIV) → offer sent → D30 thread (shared by the 2 parties only = PRIV-pair) → accept → D31 DOT review (CMSH/DOT) → T3 ledger → T2.
- The thread is a new class, **PAIR**: it is visible to the two franchises, and to the commish only after accept or appeal.

**C7 Plan chain:** D39 plans (PRIV) steps anchored on D40 calendar events + checks via C1/C2 → plan "conflict" flags → Home plan tile, WR plan rail. Plans read PUB and their own PRIV data only.

**C8 Draft chain:** D41 order + traded-via (PUB) → D42 prospects/ADP (PUB) → D43 my queue (PRIV) → auto-pick (reads queue at pick time, server-side) → D44 scouting (PRIV) → D46 mock runner (DERIV-PRIV, local only).

**C9 Archive chain:** T3 ledger + D47 archive facts → D48 eras/lineage/stories → D49 trophies/records → D50 week story/recaps → LP history, Home headline.
- D50 is generated once per week, frozen, PUB.

**C10 Office chain:** D60 jobs ← producers (alerts, governance, feeds, backups, votes, invites, softcap) → CR job feed → command (D63) → `mayRun` (D65 permission sets) → T3 ledger.
- D61 rules → change form → preview → pending rollover → D62 history (= ledger filter).

**C11 Owner-shell chain:** D56 alerts (PUB-derived, per franchise) → D57 snooze (PRIV) → Home alert band. D58 layout (PRIV) decides which tiles render. D75 freshness stamps every tile.

## 4. Domain table — owner · class · feeds from → feeds to · refresh · layer
Layer = build order: L0 trunks exist, L1 aggregators, L2 models, L3 private stores, L4 narrative/content, L5 office.

| # | Domain | Status | Owner pkg/store | Class | Feeds from | Feeds to (screens) | Refresh | L |
|---|---|---|---|---|---|---|---|---|
| D01 | Identity/phase/week | EXISTS | leagueclock, app/now | PUB | T1 | all | phase change | 0 |
| D02 | Standings/seeding | EXISTS | engine/race | PUB | scores | LP, HM, HQ, CR, D06 | game final | 0 |
| D03 | Tiebreak "step used" | PARTIAL | engine/race | PUB | D02 Order | LP | game final | 1 |
| D04 | All-play, positions, grid, brackets | EXISTS | app/season | PUB | scores | LP, D05 | game final | 0 |
| D05 | Power + components + why-moved | PARTIAL | powerrankings | PUB | D04, D11, D14, D17, D25 | LP, HQ, HM, D06 | weekly + injury | 2 |
| D06 | Playoff sim/magic/bye | NEW | new `engine/sim` | PUB | D02, D05, schedule | LP, HM | game final, nightly | 2 |
| D07 | Live scoreboard | EXISTS | ingestion/livegame | PUB | ESPN→nflverse | LP, HQ | live poll | 0 |
| D08 | Play-by-play/drives | PARTIAL | ingestion/livegame | PUB | ESPN summary | LP | live poll | 1 |
| D09 | Fantasy box per slot | PARTIAL | app/now | PUB | D08, scoring rules | LP, HM | live poll | 1 |
| D10 | Followed games | PARTIAL | store/personal (new kind) | PRIV | — | LP | on change | 3 |
| D11 | Projections/ranges | PARTIAL | app/market (MFL) → native | PUB | feeds | HQ, HM, D05 | daily | 2 |
| D12 | Weekly points/form | EXISTS | app/season, fantasyscore | PUB | D09 final | HQ, HM, LP | final | 0 |
| D13 | Usage/def-vs-pos | NEW | ingestion (nflverse) | PUB | nflverse | HQ, HM | weekly | 4 |
| D14 | Injuries | EXISTS | ingestion/injuries | PUB | feeds | HQ, D05, D56 | hourly | 0 |
| D15 | Return week | NEW | ingestion/injuries | PUB | feeds | HQ | hourly | 4 |
| D16 | Schedule/byes | EXISTS | ingestion/nflschedule | PUB | feeds | HQ, HM, D56 | weekly | 0 |
| D17 | Rosters/slots/taxi/IR | EXISTS | leaguerules, store/state | PUB | T3 | HQ, WR, D20 | on move | 0 |
| D18 | Contracts | EXISTS | store/state | PUB | T3 | HQ, WR, D20 | on move | 0 |
| D19 | Dead cap | EXISTS | transactions/deadcap | PUB | T3 | HQ, HM, D20 | on move | 0 |
| D20 | Cap runway | PARTIAL | new `transactions/runway` | PUB | D18, D19 | HQ, WR, HM, D36, D39 | on move | 1 |
| D21 | Move envelopes | EXISTS (MFL) | mflref/envelope | PRIV until landed → LEDGER | owner | HQ | on move | 1 |
| D22 | Franchise history | EXISTS | store/history | LEDGER | T3 | HQ, HM, LP | on move | 0 |
| D23 | League ledger + via-AI + kinds | PARTIAL | store/history + governance Audience | LEDGER | all commands | CR, LP | on write | 1 |
| D24 | Value/pick value | EXISTS | app/market | PUB | T4 | HQ, WR, HM, C2 | nightly | 0 |
| D25 | Best 21 | NEW | app/market | PUB | D24, D26 | WR, LP, D05 | nightly | 2 |
| D26 | Age curve | NEW | playerrecord/app/market | PUB | T4 | HQ, WR, D25 | nightly | 2 |
| D27 | Needs/surplus | NEW | app/market | PUB | D17, D25, D26 | WR, HQ, D35 | nightly | 2 |
| D28 | Posture + lens | NEW | app/market | PUB label / DERIV-PRIV lens | D05, D25, D27 | WR | nightly | 2 |
| D29 | Builder versions/pocket | NEW | store/personal (kind) | PRIV | owner | WR | on change | 3 |
| D30 | Negotiation thread | NEW | new store (pair) | PAIR | parties | WR, HM | on change | 3 |
| D31 | DOT review/appeal | EXISTS | governance | CMSH(DOT) → LEDGER | accept | CR, HQ | on vote | 0 |
| D32 | Tune-it | NEW | app/market | DERIV-PRIV | D27, D28, D29 | WR | on request | 2 |
| D33 | Lineup delta | PARTIAL | powerrankings.Lineup | DERIV-PRIV | D17, D11, D29 | WR, HQ | on request | 2 |
| D34 | FA pool + bids | EXISTS (MFL) | app/offers, commissioner/market | PUB pool / SEALED bids | feeds, owner | WR, HM | on change | 0 |
| D35 | Bid market model | NEW | app/offers | PUB (revealed history only) | T3 bids revealed, D20, D27 | WR | window close | 2 |
| D36 | Bids ≤ cap; binding | PARTIAL+BUG | nativecheck + market | SEALED | D20, D34 | WR | on bid | 1 |
| D37 | Outbid notice content | RULING | — | SEALED | D34 | WR, HM | on bid | — |
| D38 | Targets/watch/stars | PARTIAL | store/personal `watch` | PRIV | owner | WR, HQ, HM | on change | 3 |
| D39 | Plans | NEW | store/personal (new kind) | PRIV | D40, D20, C2 | WR, HM | on change | 3 |
| D40 | Calendar events | EXISTS | leagueclock | PUB | rules | WR, HM, CR, D39 | on rule change | 0 |
| D41 | Draft order/chain/clock | EXISTS | app/draft | PUB | T3 | WR, CR | live in draft | 0 |
| D42 | Prospects/ADP/tiers | PARTIAL | ingestion/marketfeeds | PUB | feeds | WR | daily | 2 |
| D43 | My queue + auto-pick | PARTIAL | store/personal `draft` | PRIV (read by server at pick) | owner | WR | on change | 3 |
| D44 | Scouting | NEW | store/personal (kind) | PRIV | owner | WR | on change | 3 |
| D45 | Player notes | PARTIAL | store/personal `note` | PRIV | owner | HQ, WR | on change | 3 |
| D46 | Mock draft | NEW | frontend (local) | DERIV-PRIV | D41, D42, D43 | WR | on request | 3 |
| D47 | Archive facts | EXISTS | app/archive | PUB | history | LP | on season close | 0 |
| D48 | Eras/lineage/stories | NEW | app/archive | PUB | T3, D47 | LP | on season close | 4 |
| D49 | Trophies/records/awards | NEW/PARTIAL | app/archive | PUB | D47, D48 | LP | on season close | 4 |
| D50 | Week story/recaps | NEW | new `app/story` | PUB (frozen) | D02, D06, D09, T3 | LP, HM | weekly | 4 |
| D51 | Stat changes + queue | EXISTS | app/control | PUB / CMSH queue | C4 | LP, CR | final | 0 |
| D52 | News + relevance | PARTIAL | ingestion/sitenews | PUB feed, relevance DERIV per franchise | feeds, D17 | HM | hourly | 4 |
| D53 | Depth charts | NEW | ingestion | PUB | feeds | HM | daily | 4 |
| D54 | Stat tables / saved views | PARTIAL | fantasyscore; views personal | PUB / PRIV views | nflverse | HM | weekly | 4 |
| D55 | Bio + career | PARTIAL | playerrecord | PUB | T4 | HM | weekly | 1 |
| D56 | Alerts (+ Outbid, ruled 10-30) | EXISTS + new producer | app/control | per-franchise derived (PUB inputs; Outbid carries no rival data) | D14, D16, D17, D34, bid execute | HM | on input | 0 |
| D57 | Snooze | NEW | store/personal (kind) | PRIV | owner | HM | on change | 3 |
| D58 | Home layout/pins | NEW | store/personal (kind) | PRIV | owner | SH, HM | on change | 3 |
| D59 | Leaders ticker | PARTIAL | fantasyscore | PUB | D09/D12 | HM | live | 1 |
| D60 | Jobs + producers | PARTIAL | app/control + governance | CMSH | 7 missing producers | CR | on input | 5 |
| D61 | Rules + form + preview | EXISTS | leaguerules | PUB read / CMSH write | — | CR, D40 | on change | 0 |
| D62 | Rule history/presets | PARTIAL | ledger filter + params | LEDGER/CMSH | T3 | CR | on change | 5 |
| D63 | Command registry | EXISTS | commandcatalog | PUB | — | SH, CR | build | 0 |
| D64 | Roles/succession/invites | EXISTS | governance | CMSH (roles PUB) | T3 | CR | on change | 0 |
| D65 | Permission sets | NEW | governance | CMSH (grants private) | — | CR, all writes via mayRun | on change | 5 |
| D66 | Exceptions in force | PARTIAL | history + market | LEDGER | T3 | CR | on change | 5 |
| D67 | Pick ops | PARTIAL | commissioner | CMSH → LEDGER | — | CR, D41 | on change | 5 |
| D68 | TOTP status/reset | PARTIAL | store/accounts | CMSH (status only, never secret) | — | CR | on change | 5 |
| D69 | Last seen | PARTIAL | store/personal + governance | CMSH | sessions | CR | on login | 5 |
| D70 | Notices/AI draft | PARTIAL | app/commissioner | CMSH → PUB on send | — | CR, HM | on send | 5 |
| D71 | Rulebook | EXISTS | app/core, leaguerules | PUB | — | CR | on change | 0 |
| D72 | Server status/backups | EXISTS/PARTIAL | serverstatus, app/platform | OPS | — | CR | poll | 5 |
| D73 | Support/admin roles | NEW | governance | OPS | — | CR | on change | 5 |
| D74 | AI agent connections | NEW | store/accounts | CMSH per actor | — | CR | on change | 5 |
| D75 | Freshness | EXISTS | app/report | PUB | every feed | every tile | per feed | 0 |
| D76 | CR search | NEW | frontend over CMSH data | CMSH | CR data only | CR | on load | 5 |
| D77 | Rookie slot salary | EXISTS | engine/rookiedraft | PUB | leaguerules SlotPrice | HM, D41 | on rule change | 0 |
| D79 | Waivers: claims, priority, processing | EXISTS | commissioner/market (waiver_execute/legality/reading) | SEALED claims until processing; priority PUB | owner claims, D61 rules | WR, CR jobs | on process | 0 |
| D80 | College game context for prospects | PARTIAL | app/sources college (CollegeFootballData, needs free key) | PUB | college feed | WR scouting | weekly in season | 4 |
| D78 | How acquired | PARTIAL | transactions | PUB | T3 | HM, LP | on move | 1 |

## 5. Build layers (dependency order)
- **L0 — trunks (exist):** wire the screens to them first. Every screen can render real skeleton data from L0 alone.
- **L1 — aggregators:** D20 runway (unblocks C1 everywhere + the bid cap bug), D03, D08/D09, D23 via-AI/kinds, D55, D59, D78, D21 native lifecycle, D36 cap gate + withdraw fix.
- **L2 — models (need algorithms, open item 4):** D26 → D25 → D27 → D28/D32/D33; D11 native; D05 components → D06 sim; D35 bid market; D42 tiers. Each needs a written method + a replay proof (correctness #1).
- **L3 — private stores:** extend `store/personal` with new kinds (follow, plan, version/pocket, scouting, snooze, layout, saved view, stars/lanes). Pair store for D30. One isolation test suite covers all of them.
- **L4 — content/narrative:** D13, D15, D48–D50, D52–D54.
- **L5 — office:** D65 permission sets first (every CR write goes through `mayRun`), then D60 producers, D62, D66–D70, D72–D74, D76.

The order L1 → L2 → L3 is fixed by data dependencies. L4 and L5 can run in parallel with L2/L3.

## 6. Isolation tests the build must carry (one table test each)
1. PRIV kinds: franchise A cannot read B through any API, cache, search, export or commish view.
2. SEALED bids: before window close nobody else sees amount, count or existence (D35 rivals included); after close they appear in the ledger.
3. PAIR threads: only the two parties; commish only after accept/appeal.
4. PUB models never take a PRIV/SEALED input. Enforced by type: model inputs come only from PUB stores.
5. CMSH: every write goes through one `mayRun`; LOCKED commands are never delegable.
6. OPS: no league rows reach server-admin views.
7. DERIV-PRIV results are never written to a shared cache.

## 7. Open items — status
1. **Bid visibility (D37): RULED 10-30 — SEALED.** Outbid owner sees "You've been outbid", with no rival name or amount until the window closes; then the result is in the ledger. Plan items: (a) the WR v6 outbid card shows name + amount, so its copy changes to the sealed text (this is the one ruled change to an approved mock); (b) **Home alert**: an "Outbid" alert joins D56 (producer = bid execute event, per franchise, urgent while the window is open, deep-links to the WR bid), and D57 snooze applies to it; (c) isolation test 2 covers the alert payload (no name or amount).
2. **Role variants: RULED 10-30 — identical.** Owner screens render the same for every role; the commish sees their own team there; office powers live only in Control Room and its deep links. Build rule: no role checks in owner-screen code.
3. **Co-owners and private space (new):** RULING NEEDED. `store/personal` is per franchise. Should co-owners share one private space (the current code) or have one each?
4. **Mock inspectors half-bind:** build rule, no ruling. Every drawer binds to the selected entity, the WR partner selector swaps shelf and direction, and plan checks run per plan.
5. **Models without algorithms (L2):** each gets a method doc and replay proof before it ships. The listed order is the build order.
6. **Engine leads:** `market/bid_execute.go:136` withdraw() (R3 bug), and `nativecheck/bids.go:186` cap, which D20 resolves. The 7 jobs with no producer are L5 D60.

## 8. Tie-out — every catalog item mapped (Terra pass, Claude triage, 10-30)
Terra mapped all 1,890 items (`tie-{HQ,WR,LP,HM,CR}.md`; input = output count on every screen). 135 SPLIT, ~54 UNMAPPED, ~10 CLASS-CONFLICT. Claude triage:

**Real gaps → added to the map**
- D79 Waivers (CR-023/024/219–226, CR-222): built on main, missing from the map.
- D80 College game context (WR-334/340): a new data source, L4.
- Private UI preferences (HQ-211 drawer, HM-010 palette, HM-175, WR-359 rail shortcuts): fold into D58, renamed "Layout + UI prefs", PRIV.
- Unread/attention state (WR-003/005): PRIV read-marks on D30 threads and D56 alerts.
- Owner time zone (CR-199): account profile field (store/accounts), PRIV/CMSH.
- Server telemetry (CR-470–474, 481): the D72 NEW part, already noted; now named per row.

**Mapping misses, not gaps (domain already covers them)**
- Fantasy schedule (LP-109/149/154/248–254, CR-256–258) → D04 ScheduleGrid/ScheduleByWeek.
- Live NFL box stats (LP-265–273) → D08 (livegame ParseSummary).
- Archive verification checks (LP-192/210/220) → D47. Week-by-week standings (LP-310) → D02 replayed per week.
- Alert check failure/retry (HM-027/128/129) → D56. Clipped count (HM-044) → UI.
- League creation date, migration provenance (CR-429/430) → governance LeagueCreated + commandcatalog MigrationEntries.
- Last commish action, last command (CR-371/550) → D23 ledger filter. Auction duration (CR-238) → D61.
- Previews (CR-187/194/328/535) → D61 backend preview + command dry-run. **The correction "would this flip a result" preview (CR-328) is PARTIAL**: it needs a replay of the week, and joins L1.
- Competitive window seasons (WR-012) → D28. Trait vocabulary (WR-335) → D44, a static taxonomy.

**Class conflicts resolved**
- HQ-036/038/075 unsaved lineup edits: a new class, **CLIENT** (in-memory draft, never stored). Not a conflict.
- HQ-184 envelope shows my own bid: the owner seeing their own sealed bid is allowed. Not a conflict.
- WR-245/246, HM-203 outbid name/amount: covered by the 10-30 sealed ruling (mock copy change).
- CR-034/035 feed and backup status in the commish job feed: the commish job shows a pass/fail summary (CMSH); the detail stays OPS. On a self-hosted league the commish is often also the admin. CLAIMED, my call.
- CR-472 auth trouble per team: that's support for a team, so it's CMSH, not OPS.
- **CR-237 sealed bid count shown to the commish, and CR-219–226 pending waiver claims → RULING NEEDED** (Q4 below).

## 9. Rulings log
- Q1 bid visibility: SEALED + Home outbid alert (10-30).
- Q2 role variants: identical owner screens (10-30).
- Q3 co-owner private space: RULED 10-30 — co-owners SHARE one private space per franchise (matches store/personal keyed by franchise; no change).
- Q4 commish and sealed bids/claims: RULED 10-30 — EVERYTHING. The commish office sees every open bid and claim (amounts, teams) before reveal. SEALED now means sealed from other owners, not from CMSH. Plan items: CR mock copy "bids are private even to the commish" changes; CR-237 and CR-219–226 resolve to CMSH; isolation test 2 becomes "no OWNER other than the bidder sees it before close; the commish office does, and every commish view of an open bid is ledgered as a view" (proposed, to confirm).

## 10. Half-built areas — what exists vs what to add (Claude, origin/main, 10-30)
Method: a targeted git grep for the missing field, then the hit opened to confirm it. "0 hits" = the whole field is to build.

| # | Exists (confirmed) | To add |
|---|---|---|
| D03 | race Order/HeadToHead | "which step broke the tie" output (0 hits) |
| D05 | Blend, ExpectedWins, WinProbability | components FORM/LINEUP/HEALTH/ALL-PLAY/RUN-IN, weekly history, why-moved (0 hits) |
| D08 | livegame/espn.go current play: down, distance, yardLine | stored list of every play + drive summary + per-play fantasy delta |
| D09 | market.OnFieldNow, app/now MatchSpine/InspectLiveBox | compact stat line per slot |
| D10 | livegame Followed (league-level) | per-owner follow kind in store/personal |
| D11 | weekly projections via marketfeeds (MFL-sourced) | native ranges (floor/ceiling), rank vs position, rest of season |
| D20 | contracts + deadcap inputs | the runway aggregator itself (0 hits) |
| D21 | mflref/envelope (MFL path only) | native-mode lifecycle states (0 native hits) |
| D23 | ledger + governance Audience | actor + "via AI" stamp on entries (0 hits in commissioner/ledger.go), kind filter |
| D30 | app/offers RecordOffers | rounds/counters/turn/expiry (no real hits) |
| D33 | powerrankings.Lineup | per-trade before/after lineup delta |
| D36 | bid legality per bid | sum of open bids vs cap room (0 hits) + withdraw() fix |
| D38 | store/personal kind `watch` (+ app/offers personal_api) | lanes + stars |
| D42 | ADP, ranks, AAV feeds (marketfeeds) | tiers, "likely there at #N" (0 hits) |
| D43 | AutoPick rule menu | per-owner draft queue (0 real hits; "Queue" = stat queue) |
| D45 | store/personal kind `note` | per-player thread shape (multiple dated notes) |
| D52 | sitenews feed, core SourcesRostered | relevance score per franchise |
| D54 | fantasyscore + nflverse stats | saved views (0 hits), CSV export |
| D55 | playerrecord career, app/sources college | bio fields ht/wt/draft slot in one read |
| D59 | weekly points (D12) | leaders query (0 hits); trivial |
| D60 | TargetCommissionerJobs read | the 7 missing producers |
| D62 | presets exist for brackets only | rule-set presets apply/compare; history = ledger filter |
| D66 | exceptions scattered (archive, sources, market) | one unified "in force" read with expiry |
| D67 | correct.pick (move/award correction) | grant/swap/forfeit/condition/void |
| D68 | accounts TOTP | commish reset (0 hits) |
| D69 | store/personal LastSeen, governance LastActivity | one per-owner read for CR |
| D70 | TargetCommissionerNotices (notices, notes, mail log) | AI draft |
| D72 | serverstatus Checks, backups | telemetry p95/requests (0 hits) |
| D78 | commissioner native_seed AcquiredVia | expose per player on HM/LP |
| D80 | college source (needs a free CFBD key) | game-context read per prospect |
| CR-328 | correction preview | "would this flip a result" replay (0 hits) |

Pattern: the half-built areas are mostly **one missing read or one missing field on a built store**. The heavy new work stays where the map already put it: D20 runway, D05/D06 models, D30 threads, D36 cap sum, D23 actor stamp.
