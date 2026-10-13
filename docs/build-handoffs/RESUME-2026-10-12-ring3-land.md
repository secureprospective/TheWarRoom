# Ring 3 landing RESUME — 2026-10-12 23:55 CT (main ab17dfc after #25; land-polish on Sol)

## What we are doing
Landing rings 0–3 live: walk every built endpoint row on Claude-OS, fix what is wrong, mark `landed`, merge each
fix when tested (Christopher pre-authorised: "When something is fixed and tested we merge it").
Main = 106bca6 (PR #19 C-03 + 70 rows landed; PR #20 live-game parser faults + clock test pin).

## Christopher's rulings (do not relitigate)
- League dates: rulebook base dates, commissioner-editable at will (built, #19).
- Keep moving as a normal build; merge when fixed and tested.
- Engine points are the accurate value: never cut or replace with Lab; only improve accuracy (memory
  warroom-engine-points-outrank-lab-points).
- Christopher 10-12 ~19:00: "work through the Endpoint Index systematically; ask me if you need me".

## In flight (10-12 23:55) — read this first
Main = ab17dfc: #25 landing batch 2 (home, pulse, engine, shell + live-check fixes 1e9157b/6f5a992), on top of
#24 (ClaudeBox's MFL commissioner map, docs). Deployed on Claude-OS as "batch2b" (23:47) and checked live.
Registry PR (session/ring-3-registry3, in worktree ring-3-land-pool): S-10, S-02, M-149, C-05, M-032, M-058 landed.

Sol phase land-polish RUNNING since 23:09 (worktree ring-3-land-polish, branched from batch2 09751b3 = same code as
#25 minus the live-check fixes; merge main into it at review). Check: ls -l RUN/land-polish/sessions/ (mtime),
cat RUN/land-polish/sentinel. Resume after a quota stop: WT=~/work/TheWarRoom/.worktrees/ring-3-land-polish ./resume-wt.sh land-polish.
Next Sol brief written, NOT dispatched: par/land-tray-density.md (rulings 16 + 37, finding 58) — dispatch after
land-polish merges (both touch command-bar routing); worktree ring-3-land-tray from main.

Claude-OS: Christopher 23:12 "claudebox is done with claudeos". 23:34 passt (VM network) segfaulted during my 31 MB
scp; Christopher chose "I reattach the NIC": virsh detach/attach-device (saved XML RUN/claudeos-nic-2026-10-12.xml),
network back 23:45. Copy builds with rsync --bwlimit=8000 --partial, not scp.
Christopher 23:50: "im not doing anything in brave" — yet his week 6 lineup changed between 23:13 and 23:47 (Kiner,
Benton, Tillman in) and Brave's tab title changed to an MFL page. Reported; not ours to act on.

## Landed so far
Registry on main: 70 landed. RUN/land-landed.tsv has more since (Control Room League/Data/App, Now reports, archive,
counterparties, comms, command bar, status strip, M-001/X-01/M-016/M-072) — apply in the next registry commit
(same script as #19: replace placeholder ring_note else append).
Still to walk: X-02/X-03 (need a play by Coleman/Shakir/Kincaid; none by 47 plays), L-28/L-29/L-36 (team panel in
Inspector; open L-27 then scroll Inspector), L-30..32 offers, Draft lab L-15..L-22 and M-065/M-087/M-176 (after
land-pool), HQ rows (after land-hq), Home alert tray/digests (after land-home), S-10 filter persistence, S-06 (finding 37),
pools (product call), M-114 (finding 43).

## Findings
RUN/land-findings.md, 1–45. 22 → ring 4 (Offers history reads the frozen Lab doc). Finding 33 corrected: the league
never ran an MFL pool (every season's pool file empty; survivor "League Does Not Use Survivor Pool").

## Christopher's answers 10-12 23:12 (rulings, do not relitigate)
- 16: bye shortfalls go in the Alert tray AND stay on HQ.
- 33: retire the six pool rows (M-111/112/113/174/200/201). Past MFL pool data is worthless; War Room will get its own
  pools, mini games and league awards much later (memory warroom-pools-are-future-native-not-mfl).
- 37: density N/T/M must change tables too.
-> next Sol brief par/land-tray-density.md after land-polish.

## Process note
19:04 I ran `pkill -f` with a pattern in my own command line (killed only my own shell). Use TaskStop for my
background tasks; never pkill -f.

## Environment
Gate env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOMEMLIMIT=1500MiB GOMAXPROCS=8
GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1; make … VERSION=dev COMMIT=ring3
BUILD_DATE=ring3. Deploy: gzip build/bin/thewarroom → scp claudeos:warroom-ring1/thewarroom.new.gz (retry on
"Connection reset") → gunzip, chmod → bash deploy.sh <tag>; wait for "value model fitted" after the last "starting dev".
Claude-OS is 1920x1080 since 10-12 20:03 (Christopher; Calendar link now (38,979)). Helpers: RUN/shot.sh, click.sh, open-row.sh <ID> [wait]. Scroll the page with the mouse over x=905 (the
pane scrollbar), not over tables (nested scroll). Density buttons N/T/M at (839/861/883, 86) with the Inspector at
rest; the Inspector expand/collapse toggle at ~(1130–1141, 86).
