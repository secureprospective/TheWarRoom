# Ring 3 landing RESUME — 2026-10-12 18:22 CT (land-c03 committed; walk ~half done)

## What we are doing
Landing rings 0–3 live: walk every built endpoint row on Claude-OS, fix what is wrong, mark `landed`.
Worktree `~/work/TheWarRoom/.worktrees/ring-3-land`, branch `session/ring-3-land` (from main 5a1c329), NOT pushed.
Commits: c702d43 (resume), 93d3775 (leagueclock window rules, mine), 1dbe69c (land-c03, Sol + my review fixes).

## Christopher's rulings this window (do not relitigate)
- League dates: "use the base dates in the rules, but those must be able to be edited by the commissioner at will" → built.
- "dont let the game coming up slow us down … Lets keep moving as a normal build process. When something is fixed and
  tested we merge it." → deploy/merge after gates + live check without waiting on the game; per-unit PRs are fine.
- "engine points are important. Lab points still needs a face lift, its not as accurrat as engine points right now.
  Do not cut engine points, only improve the accurracy if you touch them." (memory saved:
  warroom-engine-points-outrank-lab-points). Never replace engine values with Lab values.
- Demo tonight still exists (MNF BUF @ LAR 7:15 PM CT on Claude-OS): live-game rows (X-01..X-03, M-001, M-016, M-072)
  land by watching them during the game.

## land-c03 (committed 1dbe69c) — what it is
C-03 seasonal card: rulebook windows dated from held NFL schedule (trade deadline = Week 9 first kickoff, §14;
playoff_bids_only = first kickoff of week lastRegular (13); multiyear_bids_close = week lastRegular+1), deadlines are
Acts (deadlineRoute: lineup→HQ lineup, trade→trade desk, bids→valuations-and-pool, else calendar.summon), Calendar link.
S-13 phase links mark current + name destinations. Calendar (M-096) WindowRow: date input, Set date, Back to the
rulebook/Clear date (window.set → TargetSetWindow). Trade deadline override = existing SET_TRADE_DEADLINE directive
(season-scoped in snapshot.splitWindowDates by deadline year == season); other windows = calendar WINDOW_DATE events
`window:<season>:<kind>`. trade_projection uses effective instant. S-01 placement → Command bar.
Sol gates: verify 0, build 0, bundle 305,818 B. AFTER my review edits only targeted tests ran (snapshot, fixtures,
transactions, root -run Trade|Window|Clock, vitest clock+shell 206) + lint → full verify NOT yet rerun.

## NEXT ACTIONS, in order
1. Gate: `timeout 600 nice -n 10 make verify …` and `make build …` (env below). Then deploy:
   gzip build/bin/thewarroom → scp claudeos:warroom-ring1/thewarroom.new.gz → gunzip, chmod → `bash deploy.sh land-c03`.
2. Live-check on Claude-OS: Home seasonal card (windows listed? Trade deadline Week 9 date ~Thu Oct 29 2026 kickoff;
   deadline Acts route), S-13 current phase marked, Calendar (bottom-left "Calendar"): nine windows with sources,
   set cut_day date → shows "set by the commissioner" → Back/Clear; trade deadline set & back to rulebook; trade desk
   reason. Screenshot RUN/shots/land-c03-*.png. Also note: held schedules for weeks 9/13/14 arrive via the hourly
   refreshWeeksInBackground loop (first pass at startup).
3. Apply registry updates: set status `landed` + ring_note from RUN/land-landed.tsv (63 rows so far, ids + proof text)
   plus C-03, S-13, S-01 after step 2; apply the 21 "with target" merge reasons + M-027 note from RUN/land-notes.tsv
   (verify M-027 note claim on HQ lineup first). Regenerate `(cd frontend && node scripts/gen-endpoints.mjs)`.
   Spec: define `landed` in docs/ui/Target_UI_Spec_2026-10.md ("seen working on Claude-OS against the live snapshot,
   with its data, its empty state and its Act (if any); screenshot in the run dir").
4. Push, PR, squash-merge (`~/.local/bin/gh pr merge <n> --squash`) — Christopher pre-authorised merge once fixed+tested;
   tell him the PR and what landed.
5. Brief Sol phase land-home (findings 1,2,3,4,7,8,11,13,14,15,17,19,20,22,25 from RUN/land-findings.md), then
   land-pool (5,21,23,24,26,27,28,29,30,32,33,34,12,16,18). Review each with a planted mutant; live check; merge.
6. Continue the walk for rows not yet seen: Pulse Now reports (M-003..M-088 tabs), Pools rows unviewed (M-023, M-083,
   M-084, M-085, M-105, M-114, M-111, M-112, M-174, M-200, M-201), Counterparties L-23..L-29/L-36 tables, Offers L-30..32,
   Draft lab L-15..L-18, L-21, L-22, Contracts rows (after fix 17/19), Control Room (League 5, Data 5, App 4),
   Inspector, Comms 3, Calendar 3, Command bar 2, Status strip 1, Home Alert tray 3, Digests 3.
   Helper: RUN/open-row.sh <ID> [wait] opens a row via the command bar and writes shots/row-<ID>.png (tab clicks
   shift after the first click — always open by id).
7. After the ring-3-land merge: `git worktree remove` ring-3 and ring-3-d; log in ~/MOVED.md, ~/archive/MANIFEST.md,
   CT105 context.md; delete phase caches (chmod -R u+w first).

## Files that carry the walk
- RUN/land-findings.md — 34 numbered findings (bugs + polish) with screenshot refs. The fix queue.
- RUN/land-landed.tsv — id<TAB>ring_note for rows proved live (63).
- RUN/land-notes.tsv — the 21 merge reasons + M-027.
- RUN/land-walk.txt — built rows grouped by placement.
- RUN/par/land-c03.md — the brief; RUN/land-c03/REPORT.md — Sol's report.

## Environment
Gate env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1; make … VERSION=dev COMMIT=ring3
BUILD_DATE=ring3. Frontend installed with pnpm in the worktree. Dispatch Sol: cd RUN && WT=<worktree> nohup setsid
./dispatch-wt.sh <phase> par/<brief>.md > <phase>.launch.log 2>&1 < /dev/null & ; done = <phase>/sentinel.
Claude-OS: RUN/shot.sh, click.sh, open-row.sh; app log ~/warroom-ring1/run.log; DB ~/.config/TheWarRoom/history-dev.db
(sqlite3 -readonly). Density keys N/T/M are global shortcuts — never type outside a focused input.

## Honest status
Nothing running. land-c03 not yet live-checked or fully re-gated after my review edits. ~70 of 183 rows walked.
