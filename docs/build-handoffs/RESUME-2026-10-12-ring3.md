# Ring 3 RESUME — 2026-10-12 15:50 CT (updated after the second compaction)

## Goal / deadline
/goal: complete ring 3 to excellence. **Christopher demos TheWarRoom to league mates at tonight's game
(MNF BUF @ LAR 7:15 PM CT).** Plan: stable build ready by ~18:45; Lab values (s2b/s3) are explicitly
after the demo (told him; their screens honestly say "not wired").

## State (branch session/ring-3, worktree ~/work/TheWarRoom/.worktrees/ring-3, local, unpushed)
HEAD 288e2da, deployed to Claude-OS as `ring3-p3h` at 14:43. All gates passed (785 frontend tests, 0 lint).
Commits this window:
- e5ced74 / a8533d6 / e5dd8ac: Research polish. Inspector facts as a list; Acts only on my franchise's
  players; a single as-of line; the redundant Last-week column dropped; week times grouped; archive
  lookups once per season (`archiveRead`); depth crosswalk via `history.SourceIDs` (one query: it was
  ~7 ms × thousands per read). Research now opens about 12 s after launch and search answers instantly.
- 1171e12 / d965844: **p3l2 done.** `livegame.Adapter.Final` reads a followed final once per follow,
  keeps it via `heldFinals` (history archive), and reads it back after a restart with no fetch; MFL↔ESPN
  team codes (`livegame.Canonical`); final shape (score, scoring plays by high-water mark, box, no field);
  "Game details as of / not read yet"; kickoff without seconds; X-02 week-wide spine + "Plays held for N
  of M games". Live-checked on Claude-OS (DET@ARI final, Chuck Clark/Jake Bates spine). Mutant caught.
- 288e2da: **p3h done** (Sol's partial reviewed and finished by Claude in ring-3-b, ff-merged).
  Archive reports distinct and in words; full-archive smoke via `TestArchiveReportsOnTheFullArchive`
  (opt-in, TWR_ARCHIVE_SMOKE=<out>). Champions 2014–2025 read right (ARI 2016, 2017; BUF 2025).
  Registry: 22 built; M-081/172/173/198 moved to ring 4. Owed: factexam proofs against MFL pages and the
  league_event review (Sol's events.go + tests exist; not yet read line by line).

## Next actions, in order
1. Live look at League Pulse › Pools / Archive on Claude-OS (first open compiles the facts, ~22 s
   there; check the status wording, the franchise selector defaulting to mine, series, pools 2015).
2. Once Sol is back (~15:45; check with a tiny dispatch), dispatch **s2b** in ring-3-d (`par/p3g2-s2b.md`;
   Sol's partial is uncommitted there on 4f65ea7). Do not let Sol touch ring-3.
3. Whole-app demo walkthrough on Claude-OS: Home, War Room (Research), Franchise HQ, Trade Floor, League
   Pulse (Now, Race, Archive), Control Room, Comms. Fix what a league mate would trip on.
4. ~18:45: tell Christopher the build is ready (numbered steps). 19:15: watch live polling on BUF@LAR
   (strip, followed bar, field, spine, MFL live scores).
5. Post-demo: s2b → s3 (+C-01, Inspector value panels), polish (Comms link affordance, grouped
   checked-facts rows, faster first-open archive query in `history.archivedBodies`), spec rev 3,
   registry gate, Christopher's click-through, PR squash-merge after his OK.

## Offer to Christopher (not yet asked)
The "Endpoint index" cards ("built, not yet landed live · ring N") show on every screen. They're spec'd
navigation; offer to hide them for the demo if he wants.

## Environment
See the prior summary for RUN, gates, deploy, clicks. Scroll outside inner tables (the box score has its
own scroll box: use y≈752). Sol: Codex limit hit 12:32, back about 15:45.

## 15:50 update
- Done since 14:45 (session/ring-3): def06ff (one Pools report; trophy names; survivor words; unset source
  = "not set up", not the red alarm), d2a87b8 (league_event reviewed), 445a80a (fact exam vs MFL standings
  pages: 160/160, 4 known open; pages in RUN/mfl-pages; opt-in TWR_MFL_PAGES), e8e2645 (archive warmed after
  launch). Uncommitted at 15:45, gates running: Comms drawer Acts as buttons; HQ lineup "Reading lineup from
  MFL…" + plain roster while pending; seasonChanged emitted right after lineups (season_app.go); snapshot
  workspaces read "Reading the league…" before the snapshot, not "Not wired".
- Christopher: Endpoint index cards stay as they are (15:15).
- Disk hygiene (Christopher 15:20): phase caches deleted (run dir 81 GB → ~1 GB); Claude-OS deploy.sh now
  keeps 2 backups. Claude-OS 22 GB coding-standards-verification is ClaudeBox's to clean, not ours.
- Sol back 15:47: s2b continuation dispatched in ring-3-d as phase p3g2-s2b3 (brief par/p3g2-s2b-cont.md;
  partial snapshot p3g2-s2b/partial-1232.tgz). Liveness: mtime of p3g2-s2b3/sessions/*.jsonl; done: sentinel.
- VITEST: always set BOTH VITEST_MAX_THREADS=2 and VITEST_MIN_THREADS=1, or vitest refuses to start.
- 15:55: b748979 committed + deployed (ring3-demo5): HQ loading at launch, seasonChanged after lineups,
  Comms drawer buttons. Gates 787 tests, 0 lint.
- 15:49:37 Claude-OS lost SSH: host passt (libvirt passt backend, --one-off) crashed ("Flow 1 (UDP flow):
  Failed to set SO_PEEK_OFF … Bad file descriptor"); libvirt's replacement died too; a manual foreground
  `passt -f` (no --one-off) restored it; libvirt's later passt (15:52:31) now serves 127.0.0.1:2222. VM
  and app never stopped. If it recurs: `pgrep -fa passt` shows no Claude-OS passt → start
  `passt --socket /run/user/1000/libvirt/qemu/run/passt/1-Claude-OS-net0.socket --tcp-ports 127.0.0.1/2222:22`.
