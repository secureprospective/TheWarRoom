# RESUME — TheWarRoom core stages 4–8, overnight run (2026-10-04)

## 0. Next actions, in order
1. Read §5, then build the single rubric routine + settings; golden test against today's board.
2. Madden removal (separate commit, reported diff); assembly twin-file collapse.
3. Stage 5 live gate on Claude-OS (relaunch rules in §3); record in the plan; update this file.
4. Stages 6, 7, 8 per the plan; scratch history db for fitting at `~/scratch/twr-stage4/db/history.db`.
5. Stop at the README. Open one PR (do not merge); leave Christopher a numbered morning list.

## 1. What we are doing
- Christopher's goal (2026-10-03 night): work through Stages 4–8 without him. Do not merge to
  main. Stop at the README rewrite. He tests in the morning and merges.
- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`. R3 (Week-9 checkpoint) dropped.
- **Branch:** `session/core-stages-4-8` (cut from `session/week9-checkpoint`, which holds the
  Claude-OS CLAUDE.md change). One PR at the end; per-stage gate records in the plan.
- **Go:** `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB`
- **Gates:** `make lint`, `make verify`. Bloat baseline now comment 18, provenance 7, tiny 27, dupl 34.

## 2. Status
| Stage | State |
|---|---|
| 4 Signals | DONE, live gate PASS (v0.5.0-140). Commits a365ba3, 7e916e7, pushed. Gate record in the plan. |
| 5 Rubric as data | IN PROGRESS: analysis done, no code yet (see §5) |
| 6 Fit | — |
| 7 Measurables | — |
| 8 M2 | — |

## 3. Environment
- Claude-OS: `ssh claudeos` (localhost:2222 via passt). Leave it running. The app under test is
  `~/twr-gate/thewarroom`; launch: `cd ~/twr-gate && . ./cfbd.env && DISPLAY=:0 setsid -f ./thewarroom`.
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
- Put `CFBD_API_KEY` where his launcher sees it, or the college signal stays skipped on the Beelink.
- M2 shows "FINAL season complete" mid-season, and all-play 0-0: pre-existing; look at in Stage 8.

## 5. Stage 5 working notes (nothing written yet)
The 10 rubrics in `internal/engine/l4/{offense,defense,kicker}` are one routine with constants:
- film = Scurve(composite, infl, steep, cap) when HasFilm, else 1.0 (FilmRaw = NeutralNorm 0.5).
  K instead blends MaddenFilm 0.60 + NFLProduction 0.40 (each `curve.Present`), when either is present.
- RAS: off at QB and K (1.0). Else when HasRAS: 1 + weight*(Scurve(RAS/10, infl, steep, cap)-1).
- breakout: off at K (1.0). Else composite = wBA*breakoutAge + wST*schoolTier + wCS*collegeShare
  + wAT*ageTraj, through Scurve(infl 0.5, steep, cap 0.05).
  - SL-019 (strength >0 at TE 0.35, DE 0.35, CB 0.30, S 0.30) lifts breakoutAge (only when present)
    and ageTraj: `curve.SL019(v, RAS/10, strength, HasRAS)`.
  - DT: ageTraj = in.Cushion.Slow(ageTraj, NeutralNorm, RAS, HasRAS); no SL-019. The cushion is
    zero (off) for every other position via composition.
- Hooks the harness reads: `SL021Alpha(nflYear)` on DT (≤1 → 0.50 else 0.10) and DE (0.15 always);
  `HasNGSAnchor()` true on CB and S only. `defense.SL021Blend` lives in sl021.go.
- Constants differ by position: film (steep 12/cap .05, but DT/LB 10/.03, K 10/.03), RAS
  (RB 8/.04 w .6, TE 11/.08 w1, WR 10/.08 w1, DE 10/.08, DT 10/.08, LB 11/.04 w .6, CB 11/.08,
  S 10/.08), breakout steep 11 (CB 10), weights and three curves per position.
- Plan: one routine (`internal/engine/l4` single package) + per-position settings table stored as
  params (R7) — params store is `internal/store/params` (defaults.go ParamDef{Key, Position, Min,
  Max, Default}; Snapshot() Set; Set.GetGlobal). Curves need a representation in params
  (breakpoints as indexed keys) or stay as data in the settings table with scalars in params.
- Golden test first: board identical. Then Madden removal (R8) as a separate, reported change.
  Then collapse twin files in `internal/scouting/assembly` (measure before cutting).
- Users: harness_app.go imports l4/defense, kicker, offense; internal/harness/cases_eval_3g.go.
