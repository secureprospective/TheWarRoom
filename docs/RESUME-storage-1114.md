# TheWarRoom RESUME #9 — storage block + one database per league (2026-11-14 ~03:15 UTC / 9:15 PM CT 11-13)

Supersedes RESUME #8 and its UPDATE blocks (kept below in /root/TheWarRoom-RESUME.md for history).

## 1. WHAT WE ARE DOING
Storage block before trade-desk work resumes: (1) backups off the writer + skip unchanged,
(3) one league watcher instead of per-device polling, (2) REDEFINED 11-14: one dedicated database
per league. Tom (Opus, low effort) builds on the Beelink; I review lean, assemble, check on the lab.
- Repo: Beelink `chris@192.168.1.191` (key `/root/.ssh/beelink`) `~/work/TheWarRoom`, worktree `.worktrees/storage`.
- Branches: `session/storage-backup-snapshot` (pushed, DRAFT PR #229, tip `6776f8848`) and
  `session/storage-league-files` (LOCAL ONLY, stacked on it, tip `77629ee98` + this resume commit).
- Lab VM: from Beelink `ssh -i ~/.ssh/id_ed25519 root@192.168.1.182`; scripts in `/root/twr-lab/`.

## 2. AGENTS + HARNESSES
- Tom launcher (one slot, serial): `ssh -f ... 'TOM_WT=storage TOM_BRIEF=<file> TOM_TIMEOUT=<s> [TOM_SCOPE="..."] nohup ~/fleet/bin/launch-tom-build.sh >/dev/null 2>&1 &'`
- Briefs: `~/fleet/briefs/purge/`. Run dir (REUSED each launch, copy the report out first): `~/fleet/runs/tom-storage/` — done when `completion.txt` non-empty; report = `stdout.log`.
- Alive: `pgrep -f "[t]imeout <TOM_TIMEOUT> claude"`; transcript newest in `~/.claude/projects/-home-chris-work-TheWarRoom--worktrees-storage/` (`ls -lt`). A long quiet transcript can be one blocking command — check the lab CT.
- Go toolchain is `GOTOOLCHAIN=go1.26.11` now (1.26.10 refused by the module).

## 3. IN FLIGHT (check first)
**Tom lab stall run** — launched ~03:15 UTC 11-14, `TOM_TIMEOUT=10800`, brief `tom-storage-labstall-1114.md` (3,651 B with addendum).
- His container: **CT 257 `tomlab-257`** on the lab VM (he must destroy it; seed copy `/root/seed.db` inside is REAL LEAGUE DATA and must go with it). Log inside: `pct exec 257 -- cat /root/all.log`.
- Untracked `internal/db/snapshot_measure_test.go` in the worktree is his; he must delete it.
- CT105 watcher (dies at compaction — RE-ARM): on done, copies report to `~/fleet/runs/tom-storage-labstall-report.md`.
- After it ends verify: `pct list` shows no 257; `git status --short` empty; CTs 254/255/256 stopped.

## 4. THE CURRENT PROBLEM (PR #229 blocker)
Commit `21b2de0fe` replaced `sync_file_range` with `dest.Sync()` (fsync) per snapshot chunk to get
the tiny-files lint ratchet back to 24. Lab, 474 MB seed, CT 2 cores/4 GB (OBSERVED in all.log):
| run | max stall | p99 | p50 | snapshot time | RSS |
|---|---|---|---|---|---|
| fsync 1 | 1,063 ms | 642 ms | 162 ms | **747 s** | 553 MB |
| fsync 2 | 1,151 ms | 651 ms | 162 ms | **745 s** | 554 MB |
| fsync 3 | running at capture time | | | | |
- Gate (<1 s) FAILS narrowly; far worse: **12.4 minutes per snapshot vs the off-site manager's
  2-minute attempt deadline** → at this size no backup would ever finish and writes would stop at
  14 minutes. 11-08 `sync_file_range` figures were 75–82 s total, 0.47–0.81 s max.
- Tom is now running the `sync_file_range` variant (`/root/sfr.test`) 3× for the direct comparison.
- Leading fix (NOT yet decided): restore `snapshot_flush_linux.go` / `snapshot_flush_other.go` from
  `da120a34b`. Caveat: that puts tiny_files at 26 vs baseline 24; the ratchet rule is "cut it back,
  don't raise the baseline", so either find a home for the two functions in existing files or ask
  Christopher to accept 26. Tom said folding needs unrelated files/padding/hardcoded syscall numbers.
- Even 75–82 s is only for the snapshot; tar+gzip+age+upload+verify also sit inside the 2 minutes.
  Unmeasured. Per-league files make league sets small, which is the real cure.

## 5. GATES / STATUS
| Piece | State |
|---|---|
| Item 1 backups (`da120a34b`) | built, reviewed by me (manager.go, hosted_backup.go, pools.Revision): logic sound, RPO rules intact |
| Item 3 watcher (`6776f8848`) | built, reviewed (league_watch.go); 104 idle streams 1,040 → 44.8 store calls/s (fakes); revoke close ≤5 s |
| PR #229 | DRAFT; CI ALL PASS; **blocked by §4** |
| League-files map | done: `~/fleet/runs/tom-storage2-map-report.md` (19,988 B) |
| Groundwork slices 1–3 | done, local: `80ace033d` accounts schema_version · `7c0085c63` identity-first · `60c3bfc41` resolver · `b8d301cd6` · `77629ee98`; `make verify` exit 0; report `~/fleet/runs/tom-storage2-groundwork-report.md`; I read identity_first.go + stand_up diff |
| Slice 4 server file | brief READY, not launched: `tom-storage2-serverfile-1114.md` (7,003 B) |
| Slices 5–8 | not briefed (5 reference file, 6 per-league files, 7 conversion tool, 8 transfer as one file) |
| Idle SQL rate / real backup set size on a live lab server | not measured |

## 6. REFUTED / SETTLED — DO NOT RETEST
- "One shared file for all leagues" (my + Sol's 11-13 advice): OVERRULED by Christopher 11-14. Do not reopen.
- fsync per chunk as a substitute for sync_file_range: measured, fails (§4).
- Superseded brief: `tom-storage2b-split-1114.SUPERSEDED.md` (two-file authority/evidence split) — do not launch.
- Browser has no reconnect of the invalidation stream (no EventSource / stream reader in `frontend/src`).
- No SQL statement or transaction writes accounts + league tables together (map, fast read; build briefs re-check).

## 7. DECISIONS (Christopher)
- 11-13 "I approve": storage block items 1–3 before trade desk.
- 11-14: "Each league needs its own dedicated, segregated database that doesn't mix from one league to the next." Judge storage by practical future use. (memory: project_warroom_one_database_per_league.md)
- 11-14 "You have the ok": shut down CT 256 (done; stopped, NOT destroyed). Destroying 254/255/256 is NOT approved; a classifier denied my first shutdown attempt before his OK.
- 11-14 "Go" (after I said slice 4 depends on accounts being server-wide): I proceed on accounts server-wide. My reading; restated to him.
- Standing: 15-min RPO not weakened; restart begins paused; no work on main; never --no-verify; squash merges; ratchet baselines not raised.
- OPEN, asked, unanswered: Q1 restoring one league signs out only that league (rec yes) · Q2 end the shared audit chain once, one per league, old head recorded (rec yes) · Q3 football pin per league (rec yes) · Q4 deleted league archived 30 days (rec yes) · NFL data = shared reference, not league data (my assumption) · lab R2 bucket with existing Cloudflare token.

## 8. LEDGER STATE
- Pushed: `session/storage-backup-snapshot` @ `6776f8848` (PR #229).
- Local only: `session/storage-league-files` (5 groundwork commits + resume commit).
- Reports on Beelink `~/fleet/runs/`: tom-storage1-report.md, tom-storage3-report.md, tom-storage2-map-report.md, tom-storage2-groundwork-report.md, tom-storage-labstall-report.md (first, unmeasured attempt; overwritten when the current run ends).
- Scratchpad briefs mirror `~/fleet/briefs/purge/`.

## 9. NEXT ACTIONS, IN ORDER
1. Re-arm a watcher on the lab run; when done copy the report, verify cleanup (§3).
2. Read the sfr comparison. If sfr passes (<1 s, ~80 s): brief Tom to restore sync_file_range on `session/storage-backup-snapshot` WITHOUT raising the baseline if a clean home exists; otherwise ask Christopher to accept tiny_files 26. Then one `make verify`, push, CI, mark #229 ready, squash with `--match-head-commit`.
3. Tell Christopher plainly: 474 MB one-file backups cannot meet the 2-minute attempt window with fsync, and are tight even with sfr; per-league files are the cure.
4. After #229 merges: `git rebase --onto origin/main 6776f8848 session/storage-league-files` (after the fix commit, use the new PR tip instead), push, PR, CI.
5. Launch slice 4: `TOM_WT=storage TOM_BRIEF=tom-storage2-serverfile-1114.md TOM_TIMEOUT=14400` (worktree on `session/storage-league-files`, clean). Arm a watcher in the same step.
6. Brief slices 5–8 from the map as answers to Q1–Q4 arrive.
7. Lab check on a live lab server: idle SQL rate with ~100 streams; real backup set size; idle server uploads nothing.
8. Report only what ran; name what Christopher should test.

## 10. ENVIRONMENT NOTES
- Compound remote commands that shut down/destroy lab containers can be refused by the permission classifier; don't work around — ask.
- `launch-tom-build.sh` refuses a second run while one holds the slot (flock) — serial only.
- Hermes task list: `ssh -i /root/.ssh/hermes hermes@192.168.1.222`, `~/work/command-center/bin/cc`.
- Times for Christopher in Central.

## 11. HONEST STATUS
Nothing is merged. Item 1 as pushed would break backups at real size (§4); the fix is known in
shape but not built or measured on this branch. Groundwork has passed `make verify` locally but
has not seen CI. The map was a fast pass; its citations are re-checked only where slices 1–3
touched them. No end-to-end run on a lab server has happened this session.
