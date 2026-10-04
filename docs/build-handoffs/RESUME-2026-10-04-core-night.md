# RESUME — TheWarRoom core stages 4–8, overnight run (2026-10-04)

## 0. State: ALL STAGES DONE — stopped at the README, waiting on Christopher
Stages 4–8 are built, live-gated on Claude-OS and recorded in the plan. **PR #7**
(https://github.com/secureprospective/TheWarRoom/pull/7, `session/core-stages-4-8` → main) is
open and mergeable; NOT merged. Claude-OS is running the app at `v0.5.0-173-geb005c6`
(`~/twr-gate/thewarroom-s8`, sha256 fa57a0ac…), the gate DB at dynasty.discount 0.85, board #11,
model run #8. Nothing else running. Next actions are Christopher's (§4); Claude's, after his go:
1. When Christopher has tested: merge PR #7 only on his explicit go-ahead.
2. Rewrite the README's "How It Gets Built" section with him (it still credits the retired
   GLM/Gemini/DeepSeek/Ornith council).

## 1. What we are doing
- Christopher's goal (2026-10-03 night): work through Stages 4–8 without him. Do not merge to
  main. Stop at the README rewrite. He tests in the morning and merges.
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. R3 (Week-9 checkpoint) dropped.
- **Branch:** `session/core-stages-4-8` (cut from `session/week9-checkpoint`, which holds the
  Claude-OS CLAUDE.md change). One PR at the end; per-stage gate records in the plan.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify`. Bloat baseline now comment 13, provenance 0, tiny 22, dupl 0.

## 2. Status
| Stage | State |
|---|---|
| 4 Signals | DONE, live gate PASS (v0.5.0-140). Commits a365ba3, 7e916e7, pushed. Gate record in the plan. |
| 5 Rubric as data | DONE, gate PASS. 1159105 (routine), 17efc6c (Madden out), 6719a69 (assembly twins, CFBD season fix). Gate record in the plan. |
| 6 Fit | DONE, gate PASS. 835f3df (model, fit, cmd/fit), 58f71c5 (holdout rule + real fit), 0548566 (Admin filter), e8ef223 (reproducible), fb87530 (gate record; model under depguard purity). Athleticism × age tested and rejected (R6-9). |
| 7 Measurables | DONE, gate PASS (holdout: talent 9/10, value even, rookies model-only; kickers fail, recorded). 8cbe2b7…7fe4ab1. |
| 8 M2 | DONE, gate PASS (v0.5.0-172/173). 21719e4, 372c856, ce690b5 (results named by what the blend read), 5eb0e1b (column gap, name floors), eb005c6 (runs record only their own params), 4daf45f (gate record R8-1…R8-6). |

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

## 4. Morning items for Christopher (in PR #7 too)
1. Test on Claude-OS (app already running there): M1 Assets (Now / Dyn columns), M2 Pulse
   (This season / The franchise), Control → Engine Admin (filter "dynasty").
2. Decide the dynasty horizon: `dynasty.seasons` (5) and `dynasty.discount` (0.85).
3. Set the CFBD key on the Beelink (safe now: only school tier calls CFBD, R7-9).
4. Optional: turn on all-play in the MFL league settings (the blend reads points for until then).
5. Merge go-ahead for PR #7, then the README rewrite together.
- Periphery note: the Transact roster pane is narrower than its table (status and salary scroll).

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
