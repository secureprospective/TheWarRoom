# TheWarRoom UI/UX + data planning — RESUME #8 (2026-10-30 late, compact-safe)

## 1. What we are doing
Approved mocks (HQ v10, War Room v6, Pulse v14, Home v5, Control Room v1) are **fixed** going into the build. This session turned them into a complete data plan: catalog → flow map → model methods → SQL digest → staged build queue. **Planning only: no production code was written.** Next: get the docs onto main, then run the build loop.
Where: Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`, worktree `.worktrees/uiux`, branch `session/ui-ux-groundwork` (LOCAL ONLY, not pushed). Key `/root/.ssh/beelink`. Main checkout belongs to another session (`session/league-history-pwa`); don't touch it. Code truth: `origin/main` 364379ba, read with `git show origin/main:<path>`. Commit with `GOTOOLCHAIN=go1.26.10`, never `--no-verify`, trailers Co-Authored-By Claude Opus 5.5 + Claude-Session.

## 2. Agents / harnesses
- **GPT (Terra/Sol) is OUT of tokens.** Christopher: "lay off the GPT". Don't dispatch them until he says the quota reset.
- **Free OpenCode:** `~/fleet/bin/run-glm.sh <wt> <brief> <run> <model>` → `~/fleet/runs/glm-<run>/` (stdout.log, completion.txt). Models: `opencode/glm-5.3-flash-free` (he asked to confirm 5.3 free: yes), `opencode/deepseek-v4.1-flash-free`, `opencode/beam-free`. `opencode/nemotron-3.5-super-vl-free` HUNG 15 min with no output; avoid it. Launch with `ssh -f … 'nohup … &'` and arm a background until-loop watcher.
- Tom (Opus medium) for hard multi-file work. Claude reviews only.
- Christopher asked for low/medium thinking to save tokens: keep briefs tight and replies short.
- Terra chain script pattern: scratchpad `tie-chain.sh` (per-screen briefs, `launch-terra-reorg.sh`, run dir `~/fleet/runs/terra-uiux/`).

## 3. Status
| Item | State |
|---|---|
| Data catalogs (Claude 330 rows + Terra 1,890) | done, 84bee24f |
| DATA_FLOW_MAP (isolation classes, chains, D01–D80, layers, tie-out, field check) | done |
| MODEL_METHODS_DRAFT (GLM) + Claude review §13 + rulings §14 | done |
| SQL_DIGEST (R1–R10, panel, double-check §9) | done |
| BUILD_STAGES queue A–G | done, 89837a80 |
| Push docs to main | **ASKED, not answered** |
| Build loop | not started |

## 4. Artifacts (worktree `docs/ui/`)
- `data-catalog/claude-catalog-2026-10-30.md`, `terra-{HQ,WR,LP,HM,CR}.md`, `COMPARE.md`
- `data-catalog/tie-{HQ,WR,LP,HM,CR}.md`: Terra tied every one of the 1,890 items to a D# (input = output count on every screen)
- `data-catalog/DATA_FLOW_MAP.md`: §0 verification, §1 isolation classes, §2 trunks T1–T6, §3 chains C1–C11, §4 domain table D01–D80, §5 layers, §6 isolation tests, §7 open items, §8 tie-out triage, §9 rulings log, §10 half-built field check
- `data-catalog/MODEL_METHODS_DRAFT.md`: 11 model methods (GLM), §13 Claude review, §14 rulings
- `data-catalog/SQL_DIGEST.md`: §1 files, §2 latency, §3 R1–R9, §4 storms, §5 read map, §6 open, §7 rulings, §8 panel, §9 double-check corrections + R10 + endpoint map
- `BUILD_STAGES.md`: the loop queue
- Commits this window: a4f62445 map · 49e6bd55 · 829c0f70 tie-out · 85ea64cc field check · b8b34c24 · 071e5add methods · 0ded4a1b · 9b189b72 · (posture) · 558be732 SQL digest · 00a50852 · f0830f76 · fdf941ae panel · e872b2fa double-check · 89837a80 build stages

## 5. Key understanding (the "middle" of the session; don't re-derive)
**Isolation classes:**
- PUB: league-public
- LEDGER: append-only public record, actor + via-AI
- PRIV: per franchise, shared by co-owners
- SEALED: owner + commish office until reveal
- PAIR: the two trade parties; commish after accept/appeal
- CMSH: permission-set holders via one `mayRun`
- OPS: server admin, no league rows
- DERIV-PRIV: computed per request, never cached
- CLIENT: unsaved in-memory edits

The key rule: a PUB model never reads PRIV/SEALED input.

**Trunks:** T1 clock/identity · T2 roster+contracts · T3 ledger · T4 player record · T5 live feed · T6 valuation.

**Chains:**
- C1 cap: contracts + deadcap → D20 runway, the single source of "room"; bids, trades and plans all call it.
- C2 value: record → D24 → D26 age → D25 Best 21 → D05 dynasty → D28 → D27 → D32.
- C3 contend: D11 → power → D06 sim.
- C4 live: in memory.
- C5 market: sealed → reveal → ledger → D35 trained on revealed only.
- C6 trade: PRIV versions → PAIR thread → DOT review → ledger.
- C7 plans · C8 draft · C9 archive · C10 office · C11 owner shell.

**Corrections found by checking code:**
- store/personal already has kinds watch/bait/note/draft, keyed by franchise (no league column).
- D77 rookie slot salary EXISTS (`engine/rookiedraft/setup.go` slotSalary).
- D79 waivers EXIST (commissioner/market waiver_*).
- D80 college context PARTIAL (app/sources college, needs a free CFBD key).
- `AcquiredVia` exists in native_seed.
- `TargetCommissionerJobs`/`Notices` exist.
- `OnFieldNow` exists.
- livegame espn.go has down/distance/yardLine.

**Database truth (verified):**
- Files:
  - `thewarroom.db`: main (personal, envelopes, bid_facts = MFL envelope bids, params, rulebook, identity, audit, recovery)
  - `history.db`: `league_event` ledger, closed books, observations, raw_archive 224 MB, fetch_log, model runs
  - `whatif.db`: `state` schema = MFL contract what-if sandbox, NOT truth
  - `server.db`: accounts
- Native state is a FOLD of `league_event` on every read and save: 0.42 s read / 0.51 s save at 10k events, linear.
- `commissioner/replay_cache.go` is an existing memory projection cache (compensation replay only).
- Live polling uses `core.Network`, not the archiving `core.Fetches`, so there are no per-poll writes (verified).
- fetch_log writes a row per archived fetch.
- Default SQLite autocheckpoint is active.
- model_runs/scores have no retention.

**Restructure R1–R10:**
- R1 projections (panel conditions below)
- R2 league key (moot with one DB per league; keep as a guard)
- R3 `personal_item` generic table (the current UNIQUE allows one entry per player per kind)
- R4 trade_thread/round (PAIR)
- R5 model_latest + retention
- R6 obs_latest + pinned scored weeks
- R7 raw_archive + fetch_log retention
- R8 ledger indexes (league,kind,recorded_at), (league,actor,recorded_at)
- R9 sealed filter in the native bid read + proj_bid_open (NOT bid_facts)
- R10 split into per-league file / shared NFL file / server.db

**Panel (beam + GLM free, approve with changes), required for R1:**
1. Two proofs: rebuild-compare plus golden fixtures written from the rules.
2. Canonical compare in integer cents.
3. Read-time cursor check.
4. projection_version.
5. Corrections re-project off the commit path.
6. Extend the memory cache first, then persist.

**Endpoints:** ~40 `TargetNative*`/`TargetCommissioner*` fold the ledger and are all over 100 ms; they move to projections. Live/personal/feed/ops reads have other fixes.

**Model review fixes (§13):** true age not experience; no look-ahead; order D26→D25→D27→D28 · D11→D33→D32 · D11→D05→D06 · D35 · D42; D28 truth table; D06 precomputed (not per request); injury constants fitted; needs = max(Best-21 gap, starting-slot shortfall).

## 6. Refuted / don't redo
- Section-1 "store = file" naming was wrong; the `state` tables aren't league truth; R9 on bid_facts was wrong. All corrected in SQL_DIGEST §9.
- Terra's NEW counts are inflated; trust the Claude source map.
- Nemotron panel seat hangs.
- Earlier UI refutations from resume #7 still stand (classes need the cr- prefix, etc.).

## 7. Decisions (10-30, Christopher)
- **Bids:**
  - Outbid owner sees only "You've been outbid" until close; Home gets an Outbid alert; WR v6 outbid card copy changes.
  - The commish office sees EVERYTHING of open bids and waiver claims (Q4); CR mock copy "private even to commish" changes.
- **Screens:**
  - Owner screens are identical for every role; office powers only in Control Room.
  - Co-owners share one private space per franchise.
- **Models:**
  - D35: show rival names; clicking a player opens bid history.
  - D06: two magic numbers (playoffs and bye).
  - D05: one fitted weight set + commish sliders.
  - D33: this week and rest of season, equal size.
  - D28 posture: owner sets their own, the league sees it (PUB).
- **Database:**
  - **One database per league.**
  - Retention: forecasts keep latest + weekly final + replay-pinned, rest after 7 days; feed files keep scored-week pins, rest after 30 days (check on thick3 first).
  - R1 projections approved by panel with changes.
- **Build:** staged so each slice fits a context window; he will have me "run them on a loop and get everything done".
- Earlier rulings (R1–R3, P1/P2, permission sets, picks fluid) are in resume #7 and UI_Direction_v2.

## 8. Ledger
Everything above is committed on the local branch through 89837a80 plus this resume. Nothing is pushed.

## 9. Next actions
1. Get Christopher's answer: may I push `session/ui-ux-groundwork` and squash-merge the docs to main so builders see the plan? (Docs only.) If yes: push, open the PR, merge on green.
2. Start the loop at BUILD_STAGES A1 (R10 file split): write `~/fleet/briefs/build/A1.md`, dispatch Tom (GPT out), arm a watcher, review against "Done =".
3. Continue the queue in order; stop only for rulings, evidenced problems, or his G-phase click tests.
4. Optional: write `docs/data-layer` updates into Database_Growth_Catalog (D8 resolved: one DB per league; new rows R1–R10).

## 10. Environment
- Scratchpad `/tmp/claude-0/-root/ceeceeaf-c38e-4b3f-bddd-137d1de3f99a/scratchpad/`.
- Copy files with scp; `ssh -n` for commands; pipe content via `cat | ssh 'cat >> file'`.
- Big-league fixture: `~/logs/TheWarRoom/bigleague-fixtures/thick3-*.db`; `make bench-league`.
- Run `~/fleet/bin/heavy-ok.sh` before heavy jobs.

## 11. Honest status
The plan is complete and reviewed but **untested in code**. The latency numbers are measured baselines; the projected "<100 ms after R1" is a design expectation, not yet a measurement. The R10 split is the riskiest slice: it migrates real databases, so it needs a dry run and recovery-log proof. GPT quota is out; the build pace depends on Tom and the free models.
