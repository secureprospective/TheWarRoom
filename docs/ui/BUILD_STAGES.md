# BUILD STAGES — loop-ready queue (2026-10-30)

Source of truth for scope: `docs/ui/data-catalog/` (DATA_FLOW_MAP, MODEL_METHODS_DRAFT with review + rulings, SQL_DIGEST R1–R10 + panel), approved mocks in `docs/ui/mocks/`, rulings in `docs/ui/UI_Direction_v2_2026-10-30.md`.
The mocks are fixed. Correctness first: every speed path proves the same decisions as full replay.

## How the loop runs (one slice per turn)
1. Take the first `☐` slice whose `needs` are all `☑`.
2. Write its brief file from the slice card. The brief has a fixed output, write-as-you-go steps, an honesty clause and a "Debt left" section. It goes in `~/fleet/briefs/build/<id>.md`.
3. Dispatch to the builder named on the card on its own worktree/branch, and arm a done-watcher in the same step.
4. On done: Claude reviews the diff against the card's **Done =** line, runs the one targeted test (`go test -list` first), and fixes or re-dispatches.
5. Mark `☑ <date> <commit>` here and commit. Next slice.
6. **Stop and ask Christopher** only for: a ruling, a visible-screen click test (end of each G slice), or an evidenced problem.

**Sizing rule:** a slice touches ≤ ~8 files of production code plus its tests, and one domain or one table family. Anything larger gets split before dispatch.
**Branches:** one stacked branch per phase (`build/A-data`, `build/B-agg`, …). Full suite + push + CI once per phase. Squash-merge on green for invisible phases (A–F). Visible phases (G) merge after his click test.
**Builders (as quota allows):** Terra/Sol when GPT is back. Until then Tom (Opus, medium) for hard multi-file slices, and GLM 5.3 free or DeepSeek free for tight, well-specified slices. Claude only reviews.
**Every phase ends with:** the full suite, `make bench-league` against the 100 ms read and 0.5 s save budgets on thick3, and a resume update.

---

## Phase A — data foundation (must land before anything is built on top)
| ☐ | ID | Slice | Needs | Builder | Done = |
|---|---|---|---|---|---|
| ☐ | A1 | **R10 file split, schema side:** league file / shared NFL file / server.db. Open paths, pools, migrations, backup + restore + recovery list all three. | — | Tom | fresh install creates 3 kinds of file; backup/restore round-trip test passes for each |
| ☐ | A2 | **R10 migration:** existing history.db + thewarroom.db → per-league file + shared NFL file, idempotent, with a dry run. Recovery log row. | A1 | Tom | migrating a copy of the real DBs gives byte-equal ledger folds before/after; dry run writes nothing |
| ☐ | A3 | **R1 step 1:** extend `commissioner/replay_cache.go` to every domain fold (roster, contract, cap, standings, picks, bids, waivers), keyed by league + max revision. | A2 | Tom | D1 people and D9 save benches on thick3 show the change; fold output equals uncached for every fixture |
| ☐ | A4 | **Golden fixtures:** ledger → expected state, written from the rules (not the fold code) for each domain in A3. Panel condition 1b. | A3 | GLM free | ≥ 1 fixture per domain incl. corrections/supersede; all pass |
| ☐ | A5 | **R1 step 2, tables:** `proj_*` tables (roster, contract, cap-by-year, standings, pick, bid_open, waiver_claim) in integer cents with `projection_version` + cursor; written in the event's transaction. | A3, A4 | Tom | after every fixture command, proj tables equal the fold (canonical compare) |
| ☐ | A6 | **R1 guards:** read-time cursor check (refuse + rebuild), version-mismatch rebuild, nightly + CI rebuild-compare, corrections off the commit path (stale mark + rebuild job). | A5 | Tom | a forced stale cursor refuses then heals; supersede doesn't lengthen save time |
| ☐ | A7 | **Switch native reads to projections:** `TargetNative*` + `TargetCommissioner*` read proj tables. | A6 | Tom | every native endpoint < 100 ms on thick3; save < 0.5 s; growth bench flat from 1k to 50k |
| ☐ | A8 | **R8 ledger indexes** (league,kind,recorded_at) + (league,actor,recorded_at); **D23 actor + "via AI" stamp** on every event. | A2 | GLM free | ledger filter queries use the index (EXPLAIN); new events carry actor/via |
| ☐ | A9 | **R6 obs_latest** maintained at load; scored weeks pin their copy (growth D5). | A2 | DeepSeek free | week results < 100 ms warm; re-scoring a past week uses its pinned copy |
| ☐ | A10 | **R5 model_latest + retention** (latest + weekly final + replay-pinned; rest after 7 days) and **R7 raw_archive + fetch_log retention** (scored-week pins; rest after 30 days). Measure on thick3 first. | A2 | DeepSeek free | prune job is idempotent; pinned rows survive; sizes reported before/after |
| ☐ | A11 | **R3 `personal_item`** (league, franchise, kind, item_id, version, JSON) + migrate `personal_entries`; **R4 trade_thread/trade_round** (PAIR). One isolation table test for PRIV/PAIR. | A2 | Tom | isolation tests 1 and 3 from the flow map pass; old personal data migrated |
| ☐ | A12 | **Bids:** fix `market/bid_execute.go:136` withdraw (R3 "a bid is a bid"); open bids ≤ cap via `proj_cap` + `proj_bid_open` SUM; sealed read (owner: own; commish: all, Q4); Outbid alert producer. | A7 | Tom | withdraw refused for standing bids; over-cap bid refused; isolation test 2 passes |

## Phase B — aggregators (L1)
| ☐ | ID | Slice | Needs | Builder | Done = |
|---|---|---|---|---|---|
| ☐ | B1 | D20 cap runway read (multi-year committed/dead/room) from proj_cap | A7 | GLM free | HQ/WR/Home/plan/bid all call one function |
| ☐ | B2 | D03 tiebreak "step used" from engine/race | A7 | GLM free | every tie in fixtures names its step |
| ☐ | B3 | D08 full play list + drives, D09 compact stat line (memory only, no per-poll writes) | — | DeepSeek free | live replay of a recorded game shows every play; no DB writes during polling |
| ☐ | B4 | D21 native move-envelope lifecycle | A7 | Tom | native moves go planned → landed / needs you |
| ☐ | B5 | D55 bio in one read, D59 leaders, D78 how-acquired exposed | A9 | GLM free | single read each, < 100 ms |
| ☐ | B6 | CR-328 correction "would this flip a result" replay preview | A6 | Tom | preview matches the result after applying |

## Phase C — models (L2), one per slice, in method order
Each slice applies the review fixes in MODEL_METHODS_DRAFT §13, ships the replay proof with its pass bar, writes through `model_latest`, and is computed off-request (nightly / game final).
| ☐ | ID | Model | Needs |
|---|---|---|---|
| ☐ | C1 | D26 age curve (true age, no look-ahead) | A10 |
| ☐ | C2 | D25 Best 21 + rank | C1 |
| ☐ | C3 | D27 needs/surplus (max of Best-21 gap and starting-slot shortfall) | C2 |
| ☐ | C4 | D28 posture: owner-set, public; model suggests; truth-table bands; lens | C3 |
| ☐ | C5 | D11 native projections (injury constants fitted, not invented) | A9 |
| ☐ | C6 | D33 lineup delta: this week + rest of season, equal size | C5, B1 |
| ☐ | C7 | D32 tune-it | C4, C6 |
| ☐ | C8 | D05 power components + why-moved; fitted weights + commish sliders | C5, C2 |
| ☐ | C9 | D06 playoff sim (precomputed), two magic numbers | C8, B2 |
| ☐ | C10 | D35 bid market (revealed only), rival names, player bid history | A12, C3 |
| ☐ | C11 | D42 prospect tiers + still-there odds | A9 |
Builders: Tom for C9/C10, GLM free for the rest. If a replay bar fails, stop and report; never loosen a bar to pass.

## Phase D — private features (L3) on `personal_item`
D1 watch lanes + stars · D2 note threads · D3 plans (steps on calendar events + C1/C2 checks) · D4 trade versions/pocket · D5 scouting + trait taxonomy · D6 draft queue + server-side auto-pick · D7 snooze + read marks · D8 Home layout + UI prefs · D9 followed games + saved stat views. Needs A11. GLM free / DeepSeek free; one slice each.

## Phase E — content (L4)
E1 D13 usage + def-vs-pos · E2 D15 return week · E3 D53 depth charts · E4 D52 news relevance · E5 D54 stat tables + CSV · E6 D80 college context (free CFBD key) · E7 D48 eras/lineage · E8 D49 trophies/records · E9 D50 weekly story (frozen). Needs A2 (shared NFL file). E7–E9 need A7.

## Phase F — commissioner office (L5)
F1 D65 permission sets + single `mayRun` (first; every later write goes through it) · F2 D60 the 7 missing job producers (feed/backup show pass/fail to CMSH) · F3 D62 rule presets + history · F4 D66 exceptions in force · F5 D67 pick grant/swap/forfeit/condition/void · F6 D68 TOTP reset · F7 D69 last seen · F8 D70 AI notice draft · F9 D72 telemetry · F10 D73 support/admin roles · F11 D74 AI agent connections · F12 D76 CR search. Needs A7, A8.

## Phase G — screens to the mocks (visible; his click test closes each)
Each screen is split by region so a slice fits one window; every drawer binds to the selected entity (open item 4).
- **G-HQ:** rail/header/cap strip · roster table · inspector · moves/envelopes
- **G-WR:** trade builder + lens + tune-it · negotiation threads · FA/bids (sealed copy, Outbid) · plans · draft room + queue + scouting
- **G-LP:** live strip + matchup + play-by-play · race/power/sim · history/archive
- **G-HM:** tiles + layout · alerts + Outbid · news/depth/stats
- **G-CR:** jobs · rules · season/fix/picks · people/permission sets · league/server

Each needs its domains from A–F. A G slice is done only when Claude has walked it in the browser (CDP) at 1920×1080 and 2550×1352, every element is under 100 ms, and Christopher has clicked it.

## Exit gate (whole build)
All five screens match the mocks; every isolation test passes; every native endpoint is under 100 ms and every save under 0.5 s on thick3 with a 50k-event growth run flat; every model's replay bar passes; and Christopher has used it in the real environment.
