# TheWarRoom RESUME #8: 2026-11-14 (storage block; newest on top; #7 below)

## WHAT WE ARE DOING
Christopher asked (11-13) for a database architecture evaluation before resuming TheWarRoom. Done; he APPROVED a "storage block" (items 1-3 below) ahead of the trade-desk work. Tom builds on the Beelink, I review, assemble ONE branch and check it ("protect your context and assemble the final product and check it").
- Repo: Beelink `~/work/TheWarRoom`, worktree `.worktrees/storage`, branch `session/storage-backup-snapshot` (cut from origin/main `ca488e1ab`, #228). All three items stack on this branch.
- Hosts: Beelink `ssh chris@192.168.1.191`; Claude-OS `ssh claudeos` (Christopher logged in 11-14, idle, ~5 GB free: use it for built-app checks); lab via `/root/bin/twr-lab`.

## IN FLIGHT
- **Tom, item 1 (backups).** Launched 11-14 with a 4 h timeout: `TOM_WT=storage TOM_BRIEF=tom-storage1-backup-1114.md TOM_TIMEOUT=14400 ~/fleet/bin/launch-tom-build.sh`.
  - Alive: `ssh chris@192.168.1.191 'pgrep -af "timeout 14400 claude"'`. Transcript (poll its mtime): `~/.claude/projects/-home-chris-work-TheWarRoom--worktrees-storage/9de0997e-6d3a-4e4c-bf5f-58638df318c1.jsonl`.
  - Done when `~/fleet/runs/tom-storage/completion.txt` is non-empty; report = `~/fleet/runs/tom-storage/stdout.log`. Committed so far: `da120a34b` "backups: snapshot without holding the writer; skip unchanged sets" (was running end gates).
  - **The watcher dies with this context. Re-arm one on resume.**
  - NOTE: the launcher reuses run dir `tom-storage`; copy `stdout.log` to `~/fleet/runs/tom-storage1-report.md` BEFORE launching item 3 (the launcher overwrites it).

## AGENTS + HARNESSES
- Tom (Opus 5.5 low): `~/fleet/bin/launch-tom-build.sh`, briefs in `~/fleet/briefs/purge/`, one slot at a time (flock). Christopher 11-13: fixes/coding go to Tom on the Beelink.
- Sol 6.1 high second opinion: `~/fleet/bin/launch-sol61-super.sh <brief>`; briefs `~/fleet/briefs/`.
- Launch with `ssh -f`; scp briefs and check `wc -c`.

## STATUS
| Item | State |
|---|---|
| DB evaluation | DONE, delivered 11-13. Sol 6.1 report: Beelink `~/fleet/runs/sol61-sol-twr-db-arch-2026-11-13/stdout.log` (36,643 B). Brief: `~/fleet/briefs/sol-twr-db-arch-2026-11-13.md` |
| 1 Backups: paced snapshot off the writer + skip when unchanged | Tom building; one commit; NOT reviewed, NOT measured |
| 3 One league watcher instead of per-device polling + reconnect jitter | Brief STAGED `~/fleet/briefs/purge/tom-storage3-streams-1114.md` (4,254 B); launch after item 1 |
| 2 Split league (authority) data from research (evidence) data | NOT briefed. Largest. Follows `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` rows 22-28 |
| 4-6 (roster rewrite per move; hosted integrity check/session cleanup/WAL limit/disk throttle; audit check one query per row) | Not started; after 1-3 |

## KEY FINDINGS (verified in source on origin/main 17fa30b7f; do not re-derive)
- Core is sound: one writer, WAL, synchronous FULL, append-only `league_event` with hash chain, newer files refused, checked migrations. One file per server (not per league) is right.
- `internal/offsite/manager.go` + `internal/db/hosted_backup.go`: full `VACUUM INTO` on the write connection every 5 min, unconditional; writes refused when coverage > 14 min; caps 128 MiB ciphertext / 512 MiB plain. Old capture froze writes 5-22 s at 474 MB (11-08 notes).
- Paced snapshot exists on `session/s2-map` (`cc55f1877`, `1d866570c`, `internal/db/snapshot.go`), max stall 0.47-0.81 s, never on main. Costs ~DB size in RAM (+460 MB at 474 MB).
- Measured seed copy `~/fleet/lab-seed/twr-seed-20261108/history.db` (474 MB, opened immutable): raw_archive 135 MB, observations + 3 indexes ~260 MB, 0 league events in that copy. `gzip -1` = 193 MB (app uses default gzip level: real set size UNMEASURED).
- `internal/serverhttp/invalidations.go`: per stream, head read every 250 ms + session read every 500 ms; budget 3/seat + 8 = 104 streams -> ~620 idle queries/s. Client reconnect fixed 1 s.
- `internal/db/player_identity.go:100` ReplaceRosterOwnership deletes+reinserts every owned row per roster move, before the unchanged check.
- Hosted server never calls `Pools.CheckDue` (only `internal/app/platform/database_runtime.go`, desktop). No `DELETE FROM account_sessions` anywhere. No WAL pragmas set. `serverstatus/source.go` verifies both audit chains on every status request. `history/features.go:220` uses `MAX(rowid) FROM observations` (blocks a WITHOUT ROWID change).
- Live-score poller is NOT started by the hosted server today (desktop only).

## DO NOT RETEST / SETTLED
- Sol confirmed the findings; corrections folded in above (gzip figure not proof of cap breach; oversize fails the backup, writes stop via the age rule; live poller not hosted).
- Per-league files: rejected. WITHOUT ROWID on observations: not now.

## DECISIONS (Christopher)
- 11-13: "I approve" the storage block, items 1-3 first, each gated by a lab measurement: write stall < 1 s during backup; idle query rate near zero; backup under the cap.
- 11-13: fixes/coding go to Tom on the Beelink; protect my context (memory updated).
- 11-14: assemble the final product and check it; Claude-OS is available.
- Standing: 15-minute RPO not weakened; restart starts paused until a fresh verified set.

## LEDGER STATE
- Nothing pushed. Branch `session/storage-backup-snapshot` is local on the Beelink only.
- This RESUME is NOT yet committed in the repo: Tom is mid-run in the only relevant worktree. Commit it to `docs/RESUME-storage-1114.md` on that branch once Tom's run ends.

## NEXT ACTIONS
1. Re-arm the Tom watcher; when done, save his report to `~/fleet/runs/tom-storage1-report.md`.
2. Review item 1 lean: `git -C .worktrees/storage show --stat da120a34b` then the diff of `internal/offsite/manager.go` and `internal/db/hosted_backup.go`; confirm test names exist and ran (`go test -list`), and that restart-paused and the 14-minute cutoff are intact.
3. Launch item 3: `ssh -f chris@192.168.1.191 'TOM_WT=storage TOM_BRIEF=tom-storage3-streams-1114.md TOM_TIMEOUT=14400 nohup ~/fleet/bin/launch-tom-build.sh >/dev/null 2>&1'`; arm a watcher.
4. Write and launch the item 2 brief (split), from the plan rows 22-28 + Sol's ordering rule (evidence commits first; authority references only durable evidence; ownership FK to canonical identity must stay in the authority file). Tell Tom to stop and report if it needs a ruling.
5. Assemble: one branch, `make verify` once, commit this RESUME, push, PR, CI green.
6. Check on the lab (and Claude-OS for the built app): write stall during backup < 1 s; idle server uploads nothing; idle SQL rate with ~100 streams; real backup set size vs the 128 MiB cap.
7. Report to Christopher with only what was run; name what he should test.

## HONEST STATUS
- Nothing in the storage block is reviewed or measured yet. Item 2 is the big one and unscoped in hours.
- Real encrypted backup size, ten-season save time and memory on current main are unmeasured.
