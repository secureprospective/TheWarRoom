# Ring 3 RESUME — 2026-10-12 17:40 CT (ring 3 MERGED; next: land rings 0–3 live)

## What we are doing
Ring 3 merged to main: PR #18 squash 5a1c329 (Christopher OK 17:3x, "looks great, push and merge").
Christopher's next instruction (17:40): "when we come back from compaction we need to address all of the loose
ends like C-03 Home seasonal card, and the rest of them from ring 0 to ring 3".
Tonight still: demo to league mates at MNF BUF @ LAR 7:15 PM CT on Claude-OS build ring3-demo (= merged code).
~18:45 hand Christopher numbered demo steps; 19:15 watch live polling (strip, followed bar, field, spine, MFL
live scores). Do NOT redeploy Claude-OS between 18:30 and the end of the game unless something is broken.

## What "loose ends" means (evidence, 17:40)
- Every endpoint card in the app's Endpoint index says "built, not yet landed live"
  (frontend/src/app/registry/EndpointIndex.tsx:13). Registry status `landed` = proved live on Claude-OS;
  only ONE row is landed (M-027 Submit Lineup). The spec never defines `landed` — define it in the spec.
- Rows to land, rings 0–3 (status built): ring 0: 5 (incl. C-03 Home seasonal card), ring 1: 6, ring 2: 50,
  ring 3: 122 → 183 rows. Query:
  python3 csv over docs/ui/endpoint-registry.csv where ring in 0–3 and status == built.
- Merged rows with lazy ring_note "with target" (M-145, M-146, M-147, M-150, M-152, M-158, M-160, M-162,
  M-163, M-165, M-166, M-168, M-171, M-175, M-177, M-178, M-180, M-181, M-187, M-188, M-190): write the real
  merged_into/merged_place reason each.
- M-027 ring_note is a bare "1".

## Plan for the next window (NEXT ACTIONS, in order)
1. After the demo (or before 18:30 only for read-only prep): branch from main, e.g.
   `git -C ~/work/TheWarRoom worktree add .worktrees/ring-3-land -b session/ring-3-land origin/main`
   (never work on main; never touch the main checkout ~/work/TheWarRoom except via git worktree commands).
2. Define `landed` in spec §6/§14: "seen working on Claude-OS against the live snapshot, with its data, its
   empty state and its Act (if any); screenshot in the run dir".
3. Walk every built row node by node on Claude-OS (Home first: C-03 seasonal card), screenshot each to
   RUN/shots/land-<id>.png, fix what is wrong (gate → deploy → recheck), then set status `landed` with a
   ring_note naming the proof. Batch by workspace. Regenerate endpoints.gen.ts (node scripts/gen-endpoints.mjs);
   check the bundle cap 321,818 B (status words ship in the bundle — 'landed' already exists, so no growth).
4. Fix the 21 "with target" merged notes and M-027's note.
5. Gate (verify+build), deploy, Christopher click-through, PR, squash-merge after his OK.
6. Cleanup after that merge: remove worktrees ring-3 and ring-3-d with `git worktree remove` and log the
   removal in ~/MOVED.md, ~/archive/MANIFEST.md and CT105 context.md.

## Environment
- Gate env: PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
  GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1; make … VERSION=dev
  COMMIT=<ring> BUILD_DATE=<ring>. Never commit/edit while a gate runs (pre-commit stashes). Frontend lines ≤110.
- Deploy: gzip build/bin/thewarroom → scp claudeos:warroom-ring1/thewarroom.new.gz → gunzip → deploy.sh <label>.
  App log ~/warroom-ring1/run.log on Claude-OS ("value model fitted in ~7.7s" after "league archive read").
- RUN = ~/fleet/runs/warroom-ring3-2026-10-11 (shot.sh <name>, click.sh x y). Clicks: nav Home (63,204) War Room
  (75,240) HQ (87,276) Trade (80,312) Pulse (87,348) Control (87,384); War Room tabs Valuations (430,86) Research
  (540,86); Pulse tabs Now (338,86) Race (393,86) Archive (479,86); Control tabs Data and sources (429,86);
  Home Alert tray (415,86); Comms C (1256,90). Scroll: xdotool mousemove 550 500 click --repeat N 5.
- passt outage fix: `passt --socket /run/user/1000/libvirt/qemu/run/passt/1-Claude-OS-net0.socket --tcp-ports 127.0.0.1/2222:22`.
- Sol (gpt-6.1-sol, medium) available for bulk fixes: dispatch-wt.sh <phase> par/<brief>.md (WT=<worktree>);
  review each with a planted mutant; delete <phase>/cache,tmp at merge.

## Honest status
Ring 3 code is merged and live-checked on the screens changed today; the other ~180 built rows have not been
individually re-proved live this ring — that is exactly the landing work. Nothing running in the background.
