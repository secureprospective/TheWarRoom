# SQL DIGEST — the database behind the approved mocks (2026-10-30, planning only)

Source: every `CREATE` statement on origin/main 364379ba (120 tables/triggers, 10 named indexes, 4 views), `docs/data-layer/Database_Growth_Catalog.md` (measured baselines, thick3 fixture), `internal/db/pools.go`.
Target: **reads under 100 ms**, with no growth storms. Companion to DATA_FLOW_MAP.md (domains D01–D80).

## 1. The files and what lives in each
SQLite in WAL mode, one writer plus up to 10 read-only readers per file, `synchronous FULL`, foreign keys ON. Immutable tables carry `_no_update`/`_no_delete` triggers.

| Store | Tables (main ones) | What it holds | Feeds domains |
|---|---|---|---|
| **history** | `league_event` (the native ledger: league, stream, revision, kind, JSON payload, actor, supersedes), `league_season_summary` (closed books per season), `league_import`, `transfer_*` | **Native league truth.** Every command is one event; state is the fold of events. | T2/T3: D17–D23, D31, D34, D36, D41, D60–D67, D79 |
| history (stats) | `observations` (player × season × week × measure × source × as_of), `loads`, `unresolved_observations`, `measures`, `sources`, `source_fields`, `fetch_log`, `raw_archive` (every fetched body), views `week_features`/`season_features` | NFL stats from feeds, versioned per load | D07–D09, D12–D16, D54, D55, D75 |
| history (models) | `model_runs`, `model_scores`, `scoring_runs`, `run_scores`, `param_sets`, `compiled_snapshots`, `input_manifests` | Valuation runs, one row per player per run | D24, and home for D25–D28, D05, D06, D11, D35, D42 |
| **state** | `contracts`, `rosters`, `contract_years`, `dead_cap_ledger`, `cap_relief_ledger`, `contract_year_changes`, `mfl_salary_adjustments`, `player_status_events`, `calendar_events`, `season_phases`, `transaction_counts`, `trade_notes`, `whatif_sessions` | MFL-mirror league state + append-only side ledgers | D17–D20, D40, MFL path |
| **main** | `personal_entries`, `personal_offers`, `personal_offer_checks`, `move_envelopes`, `move_audit`, `bid_facts`, `bid_conditions`, `contract_facts`, `release_facts`, `league_mirror`, `param_defaults/overrides`, `rulebook_*` | Owner-private lists, MFL move lifecycle, bids, settings | D10, D21, D29, D34, D38, D39, D43–D45, D57, D58, D61, D71 |
| **identity** | `canonical_player`, `player_external_id`, `player_record` (birth, college, draft slot…), `player_ownership`, view `player_ids` | One record per real player (ruled 10-28) | T4: D26, D55, D78 |
| **accounts** | `accounts`, `account_sessions`, `account_totp`, `account_agent_tokens`, `account_enrollments`, `account_security_audit` | Logins, TOTP, AI tokens, invites | D64, D68, D69, D74 |
| ops | `audit_chain`, `league_recovery_log`, `league_roster_restore`, `server_release`, `schema_migrations` | Tamper chain, recovery, versions | D72 |

**How data gets in (four doors, and only four):**
1. **Commands → `league_event`.** Every owner and commish action, with actor and correlation ID. This is the only door for league truth.
2. **Feeds → `fetch_log` + `raw_archive` → `loads` → `observations`.** Stats, injuries, schedules, news, ADP.
3. **MFL mirror → `league_mirror` + state tables.** For MFL-hosted leagues and migration.
4. **Owner private writes → `personal_*`.** Plus accounts for login and security.

Live game data (D07–D09) is held **in memory** (`app/now` Held/Publish) and is not written per poll. That is right and must stay so; it is why game windows don't cause write storms.

## 2. Where reads stand against 100 ms (measured, thick3 = 32 teams, 7,398 events)
| Read | Today | Why | Under 100 ms? |
|---|---:|---|---|
| People page (D1) | 0.42 s at 10k events | folds the whole ledger per read | **No** |
| One save (D9) | 0.51 s at 10k | re-reads and replays every record to validate | **No** (budget 0.5 s) |
| Week results warm (D2) | 0.95 s | re-reads held stat copies + replays per starter | **No** |
| Archive lookup (D3) | 3–17 ms | URL substring scan | Yes |
| Growth | linear: 50k events → 3.9 s read, 7.1 s save | ~2,500 events per league-season | gets worse every season |

**The finding that matters:** the native ledger is the right design for correctness, but every screen that reads league state **re-derives it from events on each request**. Indexes can't fix that, because it is a compute cost, not a lookup cost. Closed books (`league_season_summary`) cap it to one season, which is still about 0.5 s.

## 3. Restructuring needed (priority order, each evidenced)
**R1 — Materialized projections for league state (the 100 ms enabler).**
Keep `league_event` as the only truth. Add projection tables that the write transaction updates, in the same transaction as the event:
- `proj_roster(league, franchise, player, slot, status)`
- `proj_contract(league, player, year, salary, end_type)`
- `proj_cap(league, franchise, year, committed, dead, room)`, which is the D20 runway, now one indexed row
- `proj_standings(league, season, franchise, …)`
- `proj_pick(league, season, round, slot, owner, via_chain)`
- `proj_bid_open(league, window, franchise, player, amount)`, with an index on (league, franchise) so D36 "open bids ≤ room" is one SUM
- `proj_waiver_claim`

Reads become indexed lookups (sub-millisecond to a few ms). Correctness guard, per ruling D9 (speed must prove the same decisions as full replay): a check that rebuilds projections from the ledger and compares byte-for-byte runs in CI and nightly. A mismatch means projection rebuild plus an alert, never silent drift.
Also fixes D9 saves: validation reads the projection, not a replay. Cost becomes one read plus one append, independent of history length.

**R2 — Personal store keyed by league.** `personal_entries`/`personal_offers` key on `mine` (franchise ID) with **no league column**. Franchise "0001" exists in every league, so once two leagues share a file (D8 still open) one league's owner reads another's private notes. Add `league` to every personal key and index. Shared by co-owners per franchise (ruled), isolated per league.

**R3 — Personal entries can't hold a thread.** The PK includes `at`, but the UNIQUE index `personal_marker(mine,kind,entity,entity_id)` allows only **one** entry per player per kind. D45 notes (dated threads), D29 versions and D39 plan steps need many. Restructure: one generic private table,
`personal_item(league, franchise, kind, entity, entity_id, item_id, version, payload JSON, created_at, updated_at, deleted_at)`, with
- PK `(league, franchise, kind, item_id)`
- index `(league, franchise, kind, entity, entity_id)`

It covers watch/stars/lanes, notes, plans, versions/pocket, scouting, snooze, layout and UI prefs, saved views, follow and queue: one isolation test for all of them (flow-map test 1).

**R4 — PAIR store for trade threads (D30).** `trade_thread(league, thread_id, a_franchise, b_franchise, status, turn, expires_at)` + `trade_round(thread_id, round, from, payload, at)`, append-only. Index `(league, a_franchise)` and `(league, b_franchise)`. The commish can read it after accept or appeal (ruled C6).

**R5 — Model results: latest-pointer + retention.** `model_runs`/`model_scores` and `run_scores` only grow; nothing deletes outside a migration quarantine. Today one valuation run writes about 2,000 player rows. The L2 models (D25–D28, D05, D06 nightly + per game final, D11, D35, D42) would multiply that by roughly 10 runs a day. Add:
- a `model_latest(league, model, run_id)` pointer, which screens read: one indexed hop
- small result tables per model keyed `(league, run_id, franchise|player)`
- **retention:** keep the latest run, each week's final run (needed for "why it moved" and history) and every run a replay proof pins; prune the rest weekly

**R6 — Stats "latest value" shape.** `observations` keeps every `as_of` version (correct for D5 integrity: pin the copy each scored week used). Screens need the latest only, so add a `obs_latest(player, season, week, measure)` table maintained at load time; per the growth catalog, scored weeks pin their version. This is what makes D2 week results fast and stops re-reading held copies.

**R7 — Raw archive retention (growth-catalog D6).** `raw_archive` keeps every body (224 MB already, 98 MB single CSV). Rule: keep bodies pinned by a scored week, a load or a replay fixture; age out the rest after N days; and compress large bodies (the gunzip limit is already in place).

**R8 — Ledger read indexes.** `league_event` has only `UNIQUE(league,stream,revision)` and `(league,correlation_id)`. The CR ledger filters (D23 actor, kind, via-AI, D62 rule history, CR-371 last commish action) need `(league, kind, recorded_at)` and `(league, actor, recorded_at)`. Projections handle state; these indexes handle the history views.

**R9 — Bid privacy at the row, not the screen.** Bids sit in `bid_facts` (main). With Q4 (the commish sees everything) and sealed-from-owners, the read query must filter by role in SQL (owner: own rows only; CMSH: all), never in the UI. D35 trains only from bids after the reveal, so it reads the ledger (revealed) and never `proj_bid_open`.

## 4. Storm risks and how the plan avoids them
| Risk | Where | Guard |
|---|---|---|
| Live polling writes during games | livegame/app/now | stay in memory; write once at final (today's design, keep it) |
| Nightly model fan-out | R5 | latest-pointer + retention; one transaction per model run, not per row |
| Playoff sim on request | D06 | precompute at game final and nightly; screens read stored rows (MODEL_METHODS review #5) |
| Save cost grows with history | D9 | R1 projections |
| One writer for many leagues | growth D8 | still open: one file vs one per league at the client/server panel. **R2 must land first either way** |
| Stats copies pile up | observations, raw_archive | R6 + R7 |
| Alert fan-out (32 teams × every injury/bid) | D56 | derive alerts from projections on read (cheap after R1); store only snooze/read marks (PRIV) |
| WAL growth during long reads | readers | backups already checkpoint; add a periodic `wal_checkpoint(PASSIVE)` with size watch (no checkpoint config is set today) |

## 5. Read map: each screen's 100 ms path after restructure
- **HQ:** proj_roster + proj_contract + proj_cap + player_record + obs_latest + model_latest (value, age) + personal_item (notes, stars). About 6 indexed reads.
- **War Room:** proj_cap, proj_pick, proj_bid_open (role-filtered), model_latest (Best 21, needs, posture, bid market), personal_item (versions, plans, queue, scouting), trade_thread. Lens and tune-it run per request on these rows (DERIV-PRIV), with no history scan.
- **Pulse:** proj_standings, model_latest (power, sim), in-memory live, league_event by (league, kind) for history, archive facts.
- **Home:** alerts derived from projections, personal_item (layout, snooze), obs_latest leaders, model_latest.
- **Control Room:** league_event with the R8 indexes, projections, governance, accounts, serverstatus.

## 6. Open / needs the panel or Christopher
- D8 one file vs one per league: already slated for the client/server panel. R2 makes either answer safe.
- Retention windows (R5/R7 "N days"): measure on the thick3 fixture before setting them. Never guess.
- R1 is the biggest structural change and touches the command write path. It is a durable, standard-setting decision, so it goes to an expert panel before the build (CLAUDE.md gate).

## 7. Rulings
- 10-30: **one database per league** (resolves growth-catalog D8). R2 drops from urgent leak to a guard: personal rows stay per franchise inside the league file; any shared/server-wide file must never hold league-private rows.
- 10-30: **retention approved** — forecasts: keep latest + each week's final + replay-pinned, delete the rest after 7 days; feed files: keep any used to score a week, delete the rest after 30 days. Check both on thick3 before locking.

## 8. Expert panel on R1 (10-30) — APPROVED WITH CHANGES
Seats: beam-free and glm-5.3-flash-free (free Zen), both APPROVE WITH CHANGES. nemotron-3.5-super-vl-free hung for 15 min with no output and was stopped; decided on two agreeing seats. Brief: `~/fleet/briefs/purge/panel-r1.md`; answers in `~/fleet/runs/glm-panel-{beam-free,mech}/stdout.log`.

R1 as built must include:
1. **Two-part proof.** (a) Rebuild-from-ledger compare in CI and nightly. (b) An independent check: golden fixtures of ledger-to-expected-state written from the rules, not from the fold code, because a fold bug shared by both paths would pass (a).
2. **Canonical compare.** Ordered rows, integer cents for all money, no floats in projections.
3. **Cursor at read time.** Each projection set stores the last ledger revision it folded. A read whose cursor ≠ the league's max revision refuses and triggers a rebuild, so stale data is caught on read, not overnight.
4. **Version stamp.** `projection_version` on the projection set; a rule or schema change forces a per-league re-project; reads refuse an old version.
5. **Corrections off the commit path.** A superseding event marks projections stale; re-projection runs as its own job in one atomic transaction with rollback. The save path stays at O(one event).
6. **Sequencing (GLM's simpler option, adopted as step 1).** First ship the fold as an in-memory per-league cache, rebuilt at startup and after supersede, which proves correctness. Then persist it as projection tables for server mode, where multiple reader processes can't share memory (Beam's point). Same fold code both steps.

## 9. Double-check pass (10-30) — corrections to sections 1–8
Every claim re-checked against origin/main. What held, what was wrong, and what the plan now says.

**Held (verified)**
- Live games are not written per poll. `composition/now_app.go:216` polls on `core.Network`; only `core.Fetches` (`app.go:213`, `archive.Transport`) archives, and live doesn't use it.
- Model runs have no retention (deletes only in a migration quarantine). Personal tables have no league column; one entry per player per kind (`personal_marker` UNIQUE). `league_event` indexes = UNIQUE(league,stream,revision) + partial `commissioner_request`.

**Wrong in sections 1/3 — corrected**
1. **Real file layout** (`app/platform/database_runtime.go:59–61`, `offsite/archive.go:49`):
   - `thewarroom.db`: main schema (personal, envelopes, bid_facts, params, rulebook, mirror, player identity, audit chain, recovery)
   - `history.db`: native ledger, closed books, stats, feed archive, models
   - `whatif.db`: the `state` schema
   - `server.db`: accounts (server mode)

   Section 1 named "stores" as if they were files; they aren't.
2. **The `state` tables (contracts, rosters, dead_cap_ledger, contract_years…) live in `whatif.db`. That's the MFL contract what-if sandbox (`composition/app.go:252`), not league truth.** Native D17–D20 come from ledger folds. The R1 projections are therefore new tables in the league file, not a reuse of `state`.
3. **R9 was aimed at the wrong table.** `bid_facts` is the MFL move-envelope bid record (keyed by the owner's own envelope). Native bids are ledger events read through `TargetNativeBids`/`TargetNativeBidAction`. Sealed filtering (owner sees own; commish sees all, per Q4) belongs in the native bid read and in `proj_bid_open`.
4. **A memory projection cache already exists:** `commissioner/replay_cache.go` (`WithReplayCache`, a bounded memory-only projection for compensation replay). Panel step 6 ("in-memory fold first") extends this cache to every domain fold. It doesn't start fresh. Reads are still slow because records are re-read from SQL and the other domain folds aren't memoized.
5. **WAL:** SQLite's default auto-checkpoint (1,000 pages) is active; section 4 overstated "no checkpoint config". The real risk is a long read transaction pinning the WAL. The guard is `journal_size_limit` plus a WAL size watch.

**New restructure item (missed before)**
- **R10 — split shared NFL data out of the league file.** One database per league (ruled) means `history.db` can't stay as it is: it mixes per-league truth (ledger, closed books, league models) with data every league shares (observations, loads, fetch_log, raw_archive at 224 MB, measures, sources) and player identity. Copying the shared data into each league file would multiply the biggest tables by the number of leagues. Target layout:
  - **league file (one per league):** ledger, closed books, R1 projections, personal items, trade threads, league-scored model results, envelopes/bid facts, rulebook, params
  - **shared NFL file (one per server):** observations + obs_latest, loads, fetch_log, raw_archive, measures/sources, player identity, league-independent model inputs
  - **server.db:** accounts, sessions, tokens, audit, server release

  Reads that need both use read-only attach or two pools; writes never span files.
- **R7 widened:** `fetch_log` writes a row for every archived fetch (`store/history/archive.go:18`), so it grows even when bodies dedupe. The same 30-day rule applies to unpinned `fetch_log` rows.

**Endpoint check (all `Target*` reads on main)**
| Group | Endpoints | Read path today | After plan |
|---|---|---|---|
| Native league (fold the ledger) | `TargetNative*` (~30: Standings, Lineup, Bids, Waivers, Trades, Draft, CapCheck, Results, Week…), `TargetCommissioner*` (Ledger, People, Market, Jobs, Preview, Rules, Schedule…) | re-fold from `league_event` each call | **R1 projections + R8 indexes.** This is every endpoint over 100 ms today. |
| Live / week | `TargetNow`, `TargetLiveGame`, `TargetRace`, `TargetSeason`, `TargetClock` | memory Held + history reads | unchanged, plus R6 obs_latest for week results |
| Personal | `TargetPersonal*`, `TargetBid*` (MFL path), `TargetTradeBuilder*`, `TargetWhatIf` | main / whatif | R3 `personal_item`; trade builder versions move there |
| Feeds/market | `TargetPool`, `TargetMarketReports`, `TargetAssetBoard`, `TargetDraftReports`, `TargetResearch*`, `TargetDataSources` | history stats + models | R5 model_latest, R10 shared file |
| Ops | `TargetBackupNow`, `TargetShellStatus`, `TargetCache*`, `TargetDataRefresh` | server | unchanged |

**Net:** the plan's shape holds (ledger stays truth; projections for speed; retention; private store). It now has **10 items (R1–R10)**. The two that change the build's foundation are **R1 (projections)** and **R10 (league file vs shared NFL file)**. Both must land before any new-screen endpoint is built on top.
