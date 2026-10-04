# RESUME — TheWarRoom core stages 4–8, overnight run (2026-10-04)

## 0. Next actions, in order
1. Stage 6: the fit tool (read the plan's Stage 6 and reasoning §4a first). Data: the scratch
   history db `~/scratch/twr-stage4/db/history.db` (2021–2026, 1.03 M values).
2. Stage 7: the two measurables, the case set, the holdout, retire the harness.
3. Stage 8: M2 on the new numbers.
4. Stop at the README. Open one PR (do not merge); leave Christopher a numbered morning list.

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
| 5 Rubric as data | DONE, gate PASS. 1159105 (routine), 17efc6c (Madden out), 6719a69 (assembly twins, CFBD season fix). Gate record in the plan. |
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
  Only after Stage 7 lands: today's Score League college path times out with a key.
- M2 shows "FINAL season complete" mid-season, and all-play 0-0: pre-existing; look at in Stage 8.

## 5. Stage 5 facts worth keeping
- `l4.Rubric` + `l4.Defaults`; knobs are params `l4.<component>.<name>@POS`; `composition.Rubrics`
  builds them from a run's params. `params.DefaultSet()` is the shipped set (tests use it).
- Claude-OS gate DB now holds board runs 1–5 (run 3 = Stage 5 golden, run 5 = Madden-free). The
  gate binaries are `~/twr-gate/thewarroom-{s4,s5,r8,s5c}`; sqlite3 is installed in the VM.
- With a CFBD key the old Score League college path times out (six live seasons); Stage 7 must
  read college from the history store instead.
