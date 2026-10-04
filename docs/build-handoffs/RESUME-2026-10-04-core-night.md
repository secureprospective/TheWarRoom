# RESUME — TheWarRoom core stages 4–8, overnight run (2026-10-04)

## 0. Next actions, in order
Stage 8 is BUILT, not yet gated. Commits 21719e4 (M2 two views on the model run, Δ against the
previous model run, roster-value naming) and 372c856 (results side reads points for when MFL
reports no all-play, which this league never does; weeks from head-to-head vs
lastRegularSeasonWeek; team column min 150 px). Nothing running; Claude-OS is up with the app
`~/twr-gate/thewarroom-s8` (built from 21719e4: rebuild and recopy for 372c856).
1. **Stage 8 live gate on Claude-OS** (copy with `scp -l 40000`; two unlimited copies reset
   the passt link): rebuild, recopy as thewarroom-s8, launch, PULSE (172,360 unmaximized).
   Screenshot both views (View chips "This season" / "The franchise"), with team names showing
   and the label naming "points for".
2. **Param edit → new run → changed board:** Control (172,533) → Engine Admin (513,130) →
   filter (497,190) "dynasty.discount" → set 0.6 → Apply; Assets (172,302) → Score League
   (395,126) → model run #4; PULSE → The franchise: Δ against model run #3 shows moves; old run
   stays readable (Δ is computed from it; `model_scores` run 3 in the VM history.db). Then set
   dynasty.discount back to 0.85 and Score League again.
3. Record the Stage 8 design (R8-1 views, R8-2 franchise = roster alone, R8-3 Δ vs previous
   model run, R8-4 points-for fallback, R8-5 phase from MFL standings) and gate in the plan.
4. Stop at the README; open one PR (session/core-stages-4-8 → main) with `gh pr create`, do
   not merge; morning list (§4) as numbered steps plus a decision matrix.
5. Update this file (three paths) and T373.

## 1. What we are doing
- Christopher's goal (2026-10-03 night): work through Stages 4–8 without him. Do not merge to
  main. Stop at the README rewrite. He tests in the morning and merges.
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. R3 (Week-9 checkpoint) dropped.
- **Branch:** `session/core-stages-4-8` (cut from `session/week9-checkpoint`, which holds the
  Claude-OS CLAUDE.md change). One PR at the end; per-stage gate records in the plan.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify`. Bloat baseline now comment 15, provenance 3, tiny 24, dupl 11.

## 2. Status
| Stage | State |
|---|---|
| 4 Signals | DONE, live gate PASS (v0.5.0-140). Commits a365ba3, 7e916e7, pushed. Gate record in the plan. |
| 5 Rubric as data | DONE, gate PASS. 1159105 (routine), 17efc6c (Madden out), 6719a69 (assembly twins, CFBD season fix). Gate record in the plan. |
| 6 Fit | DONE, gate PASS. 835f3df (model, fit, cmd/fit), 58f71c5 (holdout rule + real fit), 0548566 (Admin filter), e8ef223 (reproducible), fb87530 (gate record; model under depguard purity). Athleticism × age tested and rejected (R6-9). |
| 7 Measurables | DONE, gate PASS (holdout: talent 9/10, value even, rookies model-only; kickers fail, recorded). 8cbe2b7…7fe4ab1. |
| 8 M2 | — |

## 3. Environment
- Claude-OS: `ssh claudeos` (localhost:2222 via passt). Leave it running. The app under test is
  `~/twr-gate/thewarroom`; launch: `cd ~/twr-gate && . ./cfbd.env && DISPLAY=:0 setsid -f ./thewarroom`.
  Stage 6 gate binary: `~/twr-gate/thewarroom-s6` (v0.5.0-151 + e8ef223 fit). Unmaximized window:
  CONTROL (172,533), Engine Admin tab (513,130), Admin filter box (497,190).
  Window maximized: nav HOME (60,243) ASSETS (60,302) PULSE (60,360) CONTROL (60,534); Control
  tabs at y=123: Crosswalk (513), Signals (603). Sudo works in the VM.
- passt segfaults under some traffic. Recovery without reboot: stop the app through the guest agent
  (`virsh -c qemu:///session qemu-agent-command Claude-OS` guest-exec `/usr/bin/pkill -x thewarroom`),
  then `virsh detach-device/attach-device --live ~/scratch/twr-stage4/nic.xml`.
- CFBD key lives in CT105's `/root/.bashrc` (`ssh claudebox 'bash -ic "printf %s \"\$CFBD_API_KEY\""'`).
  The Beelink has none: Christopher's own launches skip college loads (morning item).
- Scratch: `~/scratch/twr-stage4/` (real-data backfill test `scratch_backfill_test.go`, tag
  `scratch`, plus a full 2021–2026 history db at `db/history.db` for fitting work).

## 4. Morning items for Christopher (collect here)
- Set the dynasty horizon in Control → Engine Admin: `dynasty.seasons` (5) and
  `dynasty.discount` (0.85). They are his product calls (how much later seasons count).
- CFBD key: Score League no longer fetches college stats live (R7-9); with a key only school
  tier calls CFBD. Safe to set now.
- M2 "FINAL" mid-season and all-play 0-0: FIXED in 372c856 (phase from MFL standings; MFL reports no all-play for this league, so the blend reads points for). Christopher may want to enable all-play in the MFL league settings.

## 5b. Stage 6/7 working notes
- Scale: within-position percentile of league fantasy points per game played, among the
  season's regulars (4+ games). Games = weeks with any snap or a league score, weeks ≤ the
  league's last week (17 every year 2021–2025; MFL echoes week 17 for W=18).
- MFL history: league 14432 exists 2021+ on www47. Season path must be the season itself
  (`/2021/export?TYPE=playerScores&W=YTD`); `/2026/...&YEAR=2021` returns nothing. Weekly W=n
  works. MFL 429s after ~70 calls at 0.5 rps; 0.2 rps with backoff gets through slowly.
- Scratch db `~/scratch/twr-stage4/db/history.db` now has MFL YTD 2021–2025 and weekly
  2021–2025 (2025 completing). `fitdebug.db` there is a partial backup used to debug (delete it
  after the real fit; 300 MB).
- Fit design decisions (record in the plan's Stage 6 design table): k_now by one-way ANOVA on
  weekly points, then ÷(1−R²_true) to measure against the prior (R²_true = prior R² ÷ mean
  reliability; conservative min(train, holdout) when holdout n<30); dynasty k and arc fitted
  jointly on what the blend leaves (first try, arc on raw deltas with exits imputed at the 10th
  percentile, double-counted regression to the mean and attrition — rejected); survivors
  weighted 1/P(survive) capped at 5; Z shape chosen by holdout; recency grid vs Marcel; generic
  nflverse "DB" excluded from the fit.
- Debug run (holdout 2024, partial data): blend beats last-season-alone and prior-alone at 9/10
  positions; prior R² holdout 0.21–0.43 (K ≈ 0); survival beats base rate everywhere; arc helps
  6/10; recency fitted vs Marcel mixed. Report honestly.
- **Stage 7 plan:** runtime production = MFL YTD points / games (no weekly needed at runtime);
  the app must load YTD 2021..season (closed seasons once, current each Score League — extend
  `loadBasePoints`, path year = the season). On-field-now = Z(e_eff, k_now)·est + (1−Z)·prior,
  est = recency-weighted, arc-adjusted seasons S, S−1, S−2; dynasty start uses k_dynasty;
  dynasty = Σ_{t=1..5} d^t·P(on field)·talent along the arc (discount d a non-fitted global
  param, e.g. 0.85). Rostered players use their MFL position for params. Store as a new run kind
  with its own append-only table (on_field_now, dynasty, prior, production, z's, input flags).
  Case set + holdout (2025 from ≤2024 vs today's board: 2024 points × age pull) + Spearman vs
  today's board; then retire the harness (internal/harness, Rookie Sandbox and Architectural
  Tests tabs, bindings). Stage 8: M2 (internal/m2service, powerrankings) reads the measurables
  run; two views; param edit → new run → changed board, old board readable.

## 5. Stage 5 facts worth keeping
- `l4.Rubric` + `l4.Defaults`; knobs are params `l4.<component>.<name>@POS`; `composition.Rubrics`
  builds them from a run's params. `params.DefaultSet()` is the shipped set (tests use it).
- Claude-OS gate DB now holds board runs 1–5 (run 3 = Stage 5 golden, run 5 = Madden-free). The
  gate binaries are `~/twr-gate/thewarroom-{s4,s5,r8,s5c}`; sqlite3 is installed in the VM.
- With a CFBD key the old Score League college path times out (six live seasons); Stage 7 must
  read college from the history store instead.
