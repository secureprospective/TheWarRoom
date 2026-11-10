# TheWarRoom — UI-to-mocks RESUME (compact-safe #8, 2026-11-10 ~13:15 CT) — WAR ROOM TRADE DESK

## 0. FIRST THING AFTER COMPACT
Desk runs on the real league and its lens math is PROVEN (hand recompute matched to the decimal). Waiting on
two things from Christopher: (a) GPT's answer to the pick-value brief, (b) his look on :0. Meanwhile continue
section 9. Nothing is running.

## IN-FLIGHT
Nothing. Test app stopped. Xvfb :5 up on Claude-OS. Nothing on his :0.
OUT WITH CHRISTOPHER: `/home/chris/Downloads/TheWarRoom-PICK-VALUE-BRIEF.md` on the Beelink (53,523 bytes; copy
in the scratchpad). He gives it to GPT; the answer comes back to me as a pick value table (20 numbers: rounds
1-5 x early/mid/late/unknown, in long-term PPG, plus a per-year-out multiplier). It holds all 123 Legacy 2026
trades that moved a pick + what each 2026 draft slot became. REAL LEAGUE DATA: never commit it.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Merged: rail, League Pulse (PR #223). Now: War
Room trade desk, "the biggest selling point". Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree
~/work/TheWarRoom/.worktrees/ui, branch `session/war-room-trade-desk`. No git remote there: bundle -> scp via
CT105 -> Beelink worktree ~/work/TheWarRoom/.worktrees/ui (MINE) -> push. NEVER touch .worktrees/website
(Tom's) or the main checkout. He merges.
Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache VITEST_MAX_THREADS=2
VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective -c user.email=secureprospective@users.noreply.github.com
-F msgfile, trailers Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line. Never --no-verify.
Lint limits that bit: funlen 60, gocyclo 15, gochecknoglobals, exhaustive (switch on Direction needs all cases),
gosec G101 on a literal in a field named Token.

## 2. Harness
Edits: python replace scripts in the scratchpad, scp to claudeos:/tmp/twr-*.py, run, rm by name.
- `make build` (~20 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on :5. Wait for the window + ~95 s.
  One-liner used all session: until W=$(xdotool search --onlyvisible --name "The War Room"|head -1) && [ -n "$W" ];
  do sleep 3; done; sleep 95; windowmove 0 0; windowsize 1920 1080; click War Room (86,201); sleep 16.
  (Bare `sleep 150 &&` is blocked by the harness; the until-loop form works.) Stop: kill $(pgrep -x thewarroom).
- Desk click points on :5: partner select (1432,285) then keys Down Down Return = New England; shelf row n
  position tag x=261, y=355+48*(n-1); their shelf first row (1298,618) once a partner is chosen; filter chips
  ALL (275,310) OFF (355,310) DEF (435,310) PICKS (513,310); direction buttons (958|1026|1095, 89).
- Independent check data: ~/scratch/ui-profile2/config/TheWarRoom/prepared-values/native.mfl-14432-2026-2026.json
  (reading.players[]: player, franchise, position, nowPPG, dynastyPPG, age, available). Vikings = franchise.24,
  New England = franchise.02. Ledger: history.db table league_event (kind, payload); MFL exports: fetch_log join
  raw_archive on sha256 (TYPE=transactions has all 131 season trades, FP_orig_year_round / DP_round_slot 0-based).
- Show him: scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads + file:// path, or live on :0.

## 3. Gates
| Gate | State |
|---|---|
| go build, tsc, make build | pass |
| go test tradelens, commissioner (direction test), app/market; vitest warroomLens | pass |
| Lens math vs hand recompute on real league (shelf, 2-for-1, totals, tune) | MATCHED at 60/30/0; NOT re-walked at 65/50/30 |
| Direction clicked live | NOT DONE: buttons are owner-only, nobody seated on the scratch copy |
| PROJ/WEEK, cap runway, 7-1 record independently recomputed | NOT DONE |
| make verify, wire snapshots, push, PR | not started |

## 4. What exists (all committed on the branch, NOT pushed)
- internal/tradelens: Direction.NowWeight = contend 0.65 / retool 0.50 / rebuild 0.30 (THE one place; frontend
  reads nowWeight from the reading). Evaluate, rosters, needs + starterBar, tune, Read (our read).
- internal/app/market/trade_lens.go (TradeLens, lensLeague, lensPlayers, lensPicks, ourReads; true median).
- commish.franchise.direction.set (owner of the seat, else co-commissioner and up); tradelens.read.
- Frontend: warroomLens.ts (+test), WarRoom.tsx, WarRoomHeader.tsx, WarRoomTrade.tsx, WarRoomInspector.tsx, css.
- Fixed this window: shelf values/ages hidden by old css; standings reader lacked `conference` (regression from
  PR #223, broke partner record/cap; fix is on this branch only); used picks listed on shelves (HostedRoster pick
  now has `used`); open windows read "passed"; desk column layout (.build is a flex column, TUNE IT max 132px,
  effect scrolls); Holes / Thin split; plain pick note; "the owner picks" when the viewer is not the owner.

## 5. Open problems
- PICKS HAVE NO VALUE on the imported league: Pick Value fits only on a native league's own rookie drafts
  (NewNative drops the built-in sample for "native." leagues; NativePickHistory reads draft commands) and the
  import brings none. The seed snapshot DOES hold the 2026 draft as made picks (season, round, slot, player; 160
  rows, original ''). Plan: wait for GPT's table, then design the pick value with Christopher. Importing past
  drafts = import change = his ruling first.
- Lineup panels in the effect area are squeezed (they scroll inside ~85 px). Polish later.
- trades.read logs "you may act only for your own franchise" repeatedly for a commissioner viewing a team:
  pre-existing, not chased.

## 6. Established: do not redo
- Long-term value (dynastyPPG) = 3 seasons, this one first, each later x0.75, with age curve and contract tilt.
  So long-term already carries ~43% this season. Christopher knows; his weights stand.
- Best 21 (any position) is the lens count by his "value is value" ruling; PROJ/WEEK (legal lineup, existing
  lineupdelta) sits beside it. Told him a 4th QB reads +4.9 that way; no change ordered.
- ExpirationYear = last season under contract. Engines exist: value.read, lineupdelta, pickvalue, powerrankings.
- useContractRead blanks on key change (desk holds last lens in a ref). parseHostedStandings is strict on keys.

## 7. Decisions (Christopher, 11-10)
- WEIGHTS: ".65 / .50 / .30 was with testing and human opinion from my league mates, go with it, however make
  sure its uniform where ever that weight matters." DONE in code (commit 1ec399fc).
- PICKS: "picks are essentual to dynasty football, we need to get this right"; a rough draft from a foreign model
  (GPT) comes first, then I build it.
- Earlier, still binding: direction ledger entry approved and public; partner without a pick shows OUR read,
  labelled; free hand on the desk; cover the bases before presenting; mock is the gate; he merges; stop rule: any
  other new ledger event, schema, import or permission change -> tell him first.

## 8. Ledger state
Branch session/war-room-trade-desk on Claude-OS, tip = the resume commit after 1ec399fc (weights), fcc8cb9d /
8c527bc6 (checked against the real league), 7055d9c5 (desk + direction). Tree clean. Not pushed. Untracked:
.golangci-cache/.

## 9. NEXT ACTIONS, in order
1. Rebuild, launch on :5, re-walk Vikings x New England (Jeanty + London for Brissett) at 65/50/30 and recompute
   by hand from the values file (w = .65 both sides); screenshot to his Downloads.
2. Recompute PROJ/WEEK, cap runway and the record independently, or say plainly they are unchecked.
3. When GPT's table arrives: check it against the 123 trades and the 2026 draft outcomes, propose the pick value
   design to Christopher (where the numbers live, how they move by year and slot), build after his yes.
4. Inspector polish, pocket / versions / send / negotiations.
5. Before push: make verify on Claude-OS, wire snapshots (TWR_WRITE_WIRE=1 go test ./internal/store/history if a
   read gained a field), 110-char check, bundle -> Beelink -> push -> PR. He merges after his look on :0.

## 10. Honest status
Engine math proven at the old weights; the new weights are unit-tested but not yet seen on screen. Picks are
unvalued on his league until the pick work lands. Direction never clicked live. Hosted (phone) leagues do not
get the lens. Full suite not run this window.

## 11. PICK FINDINGS (11-10 ~16:00 CT, after GPT's reports) — proposal is WITH CHRISTOPHER, nothing built
GPT reports: Beelink ~/work/TheWarRoom/GUIDES/pick-value-gpt-independent/ (v1 + COMPARISON-v2; gitignored, private).
GPT table (LT PPG, unknown slot) R1-5: 11.5/6.0/4.6/4.0/2.8, all confidence LOW, wait 0.85 assumed; no table fits the trades.
Measured by me (prototype, reverted, tree clean at bd4f6c6e):
- Existing Pick Value v2 fed the seeded 2026 draft (snapshot.Picks with player; 159 rows) reads too flat:
  1.01 = 6.55, 5.32 = 3.38 (round means ~6.3/5.3/4.9/4.0/3.6) vs what the picks became 9.7/6.1/5.7/5.4/4.8.
- On the desk a pick is lens (1-w)*LT against Best 21; median 21st-best is 7.1, so a 6.0 second adds 0 on all 32
  teams and an 11.5 first cracks 2 of 32 contenders. Any table reads ~0. This is the real problem.
- Proposed: pick worth to a side = mean over the players really taken around that slot (now + dynasty as they
  stand today, in the side's lens) of what each adds to the side's Best 21, x 0.75 per year out (existing
  pickvalue.discount). Measured medians across teams: R1 3.4 (early 4.0, late 2.2; 1.7-7.5 by team), R2-R5 0.6-0.9.
- Needs his yes on: the approach; 2027 = one year out at 0.75. No new import/ledger/schema/permission change.
- 11-10 ~16:20 Christopher: "Human behavour is seasonal, and will hit the edges, use the human behavior as an averge
  multiplier, not a scope." => trades give ONE average multiplier on the model's pick value; never fit/bound to trades.
  Measured on 43 pick-for-player trades (8 reversals out; player add vs model pick add, typical team):
  ratio of totals 1.8-1.9, per-trade median 1.15-1.2, per-trade range 0 to 11, months 0.5 to 5.5.
  Wait counted from the NEXT draft keeps it steady by period (1.9 pre-draft / 1.7 in season); counted from the season
  it drifts (1.9 / 2.3). So: next draft = no wait. Awaiting his pick of the multiplier (I recommend 1.2) + go.
- 11-10 ~16:50 BUILT for his test (he: "I lean into your recommendation for this test"): tradelens Asset.Comps
  [][]Comp + Weight, worth() = Weight x best group's mean add over the roster's N-th player; roster.count = best N
  players + picks' worth. market/trade_lens.go: pickMarket const 1.2, takenPicks, pickComps (own round, then later
  rounds so an earlier pick is never worth less), wait counted from the next open draft at pickvalue.discount.
  Hand recompute MATCHED on screen: Vikings 300.8 with picks, holds 2.0/1.5/0.4/0.3/0.2/0.1, NE adds 4.3.
  Shots with him: Beelink ~/Downloads/warroom-picks-{1,2,3,4}.png. AWAITING his human read of them.
  If he approves: make 1.2 a league setting (pickvalue.market), drop unused TradeLensSource.Players, relabel
  "YOUR BEST 21" (it now includes picks' worth; header 281.8 is the long-term best 21, a different read), reconcile
  with pickvalue v2 used by power rankings (two pick valuers now).
