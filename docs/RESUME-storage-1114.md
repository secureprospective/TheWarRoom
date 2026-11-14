# RESUME #9 UPDATE 6 (compact-safe) — 2026-11-14 ~10:10 AM CT — read this first

Christopher: "push this session to completion; do not burn wall clock." I (head brain, Opus 5.5)
build slice 5c MYSELF ("take the wheel instead of assigning it to Sol"). Tom is OUT of weekly quota.

## 1. WHAT WE ARE DOING
Storage block, one database per league. Slice 5 (football reference file) + 5c (his rulings: league
comes back when football lags; live pin change) → one PR → merge on green. Then slices 6 (per-league
files), 7 (conversion tool), 8 (transfer as one file) — Sol builds, I review/merge; his live test at end.
- Beelink `chris@192.168.1.191` (key /root/.ssh/beelink), repo `~/work/TheWarRoom`.
- Branch `session/storage-reference-file` @ 0f1dfc21e, in worktree `.worktrees/storage` (clean).
- MY 5c WORK: scratch worktree `.worktrees/storage-5c`, DETACHED at 0f1dfc21e, 18 uncommitted
  changes (code+tests+docs), NOT committed yet. frontend/dist copied in (ignored).

## 2. AGENTS
- Tom: no quota. Sol: `SOL_WT=<wt> SOL_BRIEF=<file in ~/fleet/briefs/purge/> SOL_THINK=medium|high nohup ~/fleet/bin/launch-sol-build.sh` via `ssh -f`; run dir ~/fleet/runs/sol-<wt>/ (completion.txt, stdout.log). Watcher loop on CT105, re-arm after compaction.
- Unlaunched/superseded: sol-storage5c-lag-pin-1114.md (I built it instead).

## 3. GATES / STATUS
| Item | State |
|---|---|
| PR #229, #230, #231 (debt sweep) | MERGED; main = e34d7e24f |
| Slice 5 head 0f1dfc21e | make verify exit 0, 125 ok (~/fleet/runs/storage5b-verify.log) |
| 5c code | builds + vets; targeted tests PASS (below); race run IN FLIGHT |
| 5c mutants | conflict check (history) KILLED; change gate (serverhttp) KILLED |
| make verify on 5c | NOT RUN |
New/rewritten tests (all PASS non-race): history TestRecoverLeagueIdentitiesFromRestorePoints (2 cases), TestPutArchiveBodyChecksItsName; cmd/twr-server TestRestoreThreeFilesReportsMissingPin (3 cases: together / pinned version only in evidence object comes back / gone everywhere → reported, league restores), TestMissingPinnedReferenceStopsHostedSelection (now: server starts, other league unmoved, dashboard names it, retry after 1 min picks evidence up), TestPinFootballLiveThroughSocket (guard holds it, unknown version refused, live apply, other league unmoved, directory row written); serverhttp TestMissingFootballHoldsChangesAndKeepsReadings (assets read OK, Writable false, bid held 409 plain words, bid runs after), TestRequestKeepsThePinItFirstRead.
The two old tests asserting "refuse restore"/"stop server" were REWRITTEN per his ruling (justified; say so in PR).

## 4. 5c CHANGES (storage-5c worktree)
- NEW internal/store/history/reference_recover.go: RecoverLeagueIdentities(league, reference) from league_roster_restore.payload.identities (rewritten every league txn, events.go:145); conflict = outside id already another player → reported, never merged/minted. PutArchiveBody(key, stored gzip) verifies sha256.
- cmd/twr-server/restore_reference.go: attachReference never refuses for lacks; full match wins, else nearest set that opens, else empty reference file (PrepareReferenceFile); fillReference = pinned bodies from evidence objects + identities; plain report. Wrong key/newer set still stop.
- cmd/twr-server/hosted.go: recoverPlayers at start before serving; pinRoute on status socket.
- cmd/twr-server/football_pins.go: rewritten — mutex, LeaguePin.Missing on ErrNoBody (store = base.Sibling(), no football), retry ≤1/min, missingLeagues(), setPin (directory row through guarded sign-in pools first, then swap), console `-status-socket` (running) XOR `-db` (stopped).
- internal/serverhttp: LeaguePin.Missing + cacheKey; withRequestPin/pinFor (one pin per request) in hosted.go, workflow.go, reply_cache.go, token_policy.go; workflow refuses changes for Missing (missingFootball words), Writable false.
- internal/serverstatus: Route + Serve(..., routes...); Source.MissingFootball → "Football data" Problem line. cmd/twr-server/status.go wires both.
- Docs: deploy/server/BACKUPS.md, docs/architecture/storage-recovery-identity.md.
- Known limit (tell him, done in last reply): while football missing, ALL that league's changes wait (roster moves/bids validate against pinned player facts). A past-only player (league_player, not in a restore point) whose facts were lost stays missing (reported).

## 5. IN FLIGHT
- Race run of history, serverstatus, serverhttp, cmd/twr-server in storage-5c (CT105 bg task; output lines "<pkg> rc= pass= fail="; JSON in Beelink /tmp/5c/*.json). Alive: `pgrep -fa "go test -race"` on Beelink.

## 6. REFUTED / DO NOT RETEST
- Deleting raw_archive rows in tests: forbidden (append-only trigger). Use never-held evidence (footballEvidence) instead.
- `ssh -n` + heredoc → empty file. Write locally, scp, run python patch remotely.
- Single quotes inside `ssh '...'` python blocks break quoting — scp a script file.
- Multi-statement ExecContext binds args to first statement only (modernc) — split.

## 7. DECISIONS (all in memory project_warroom_one_database_per_league.md)
Q1–Q5 yes; slice-4 answers stand; lag → league comes back, football waits; pin change live via admin socket; R2 test bucket APPROVED for final drill (league-history copy, list for him to delete). Vocabulary: "test server" (VM 503), "league history"; never "lab data".

## 8. NEXT ACTIONS
1. Read race-run result; fix any failure at its cause.
2. In storage-5c: `git add -A` (not frontend/dist) → commit on top of 0f1dfc21e with message + `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` + Claude-Session line; include this resume in docs/RESUME-storage-1114.md (prepend).
3. In `.worktrees/storage`: `git merge --ff-only <5c sha>`; remove scratch worktree storage-5c.
4. heavy-ok → `make verify` once (storage wt) → push → PR (body: slice 5 + 5b + 5c, rewritten tests named, limits) → CI → squash `--match-head-commit`.
5. Brief slice 6 for Sol (high): per-league files, own writer/backup/restore each, Q1 sign-out only that league's owners, Q2 per-league audit chain (close shared once, head recorded), Q4 30-day archive, credential-revoking sign-in writes off the league writer, directory readers; reference file stays shared. Then 7, 8.
6. Final drill on test server (copy of CT 256 21 MB) incl. real R2 test bucket upload timing; then his live test.

## 9. ENV
Go: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOTOOLCHAIN=go1.26.11 GOMEMLIMIT=1500MiB GOLANGCI_LINT_CACHE=$HOME/.cache/golangci-storage`. gh at ~/.local/bin. Times to him in CT.

## 10. HONEST STATUS
5c has passed only targeted tests (non-race) — no race result, no make verify, no CI, no real-server drill yet. Live pin over socket is tested in-process only.
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
