# Ring 3 RESUME — 2026-10-12 17:25 CT (ring 3 code-complete; demo build frozen)

## Goal / deadline
/goal: complete ring 3 to excellence. Demo to league mates at MNF BUF @ LAR 7:15 PM CT. Build is FROZEN as
ring3-demo on Claude-OS (a1326f9). ~18:45 hand Christopher numbered demo steps; 19:15 watch live polling
(strip, followed bar, field, spine, MFL live scores). Endpoint index cards: LEAVE AS IS (Christopher 15:15).

## Branch state (session/ring-3, ~/work/TheWarRoom/.worktrees/ring-3, local, unpushed)
- bd75e80 C-01 asset board (negative-PPG live bug fixed). eaf0193 merge of session/ring-3-d (Sol s2b 01daed7,
  reviewed with mutant ageBand 24.5→25.5: goldens + P4 failed, 1,291 trade diffs; clean 0 diffs on 18 keys;
  bbaa629 AssetValue.Label/Range). a1326f9: fitAssetModel after warmArchive (7.7 s on Claude-OS), labAssets
  adapter, market wording (true minus, never −0.00, below-zero explained), qualifiers on own line, spec
  revision 3 (§4 League rename + label hacks removed, §14 built, §15 review/carried to ring 4), registry notes
  M-075/M-087/M-103/S-05. 118/118 kept ring-3 rows built. verify+build 0. Working tree clean.
- IR alert check done: tray shows no unavailable line on live data; no change needed.
- Claude-OS: ring3-demo running (backups: previous two).

## Remaining
1. ~18:45 numbered demo steps to Christopher; 19:15 watch BUF@LAR live polling; fix nothing mid-demo unless
   broken (then gate → deploy).
2. Christopher's click-through of ring 3; then push + PR (body ends with the Claude Code line) and squash-merge
   only after his OK: `~/.local/bin/gh pr merge <n> --squash`.
3. After merge: remove ring-3-d worktree via git worktree tooling (never mv); delete phase caches still present.

## Environment notes
- Gate env: PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
  GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1. Never commit/edit while a
  gate runs. Frontend lines ≤110 incl. tests.
- Deploy: gzip build → scp claudeos:warroom-ring1/thewarroom.new.gz → gunzip → deploy.sh <label>.
- passt outage fix: `passt --socket /run/user/1000/libvirt/qemu/run/passt/1-Claude-OS-net0.socket --tcp-ports 127.0.0.1/2222:22`.
- Clicks: nav Home (63,204) War Room (75,240) HQ (87,276) Trade (80,312) Pulse (87,348) Control (87,384);
  War Room tabs Valuations (430,86); Pulse tabs Now (338,86) Race (393,86) Archive (479,86); Control tabs
  Data and sources (429,86); Home Alert tray (415,86); Comms C (1256,90).
