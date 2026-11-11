# TheWarRoom — UI-to-mocks RESUME (updated 2026-11-11 03:20 CT) — TRADE DESK: DRAFT SCARCITY BUILT, READS 1-3 PASS

## 0. FIRST THING AFTER COMPACT
Draft scarcity is BUILT and committed (329602e6) and a null-list desk bug fixed (e58ae54a). Next: section 9 from
item 4 (Fit_Report keep table, parameter_words), then the push batch. Read 4 still fails: veteran/backup-QB session.

## DRAFT SCARCITY (what shipped, 11-11)
Input.Drafted = every player the league took in its own drafts (market: takenPicks comps). roster.drafted() = per
position, mean lead (over) of those players. roster.player(a) = max(0, over - drafted[pos]) is a PLAYER's trade
worth; count() uses it. Picks (worth) use plain over: they are the supply, nothing taken off. No constant, no fit,
all rounds. Real league draft level (retool): CB 1.5 DE 1.6 DT .6 LB 3.0 PK 1.0 QB 2.5 RB 2.1 S 3.1 TE 1.1 WR 2.0.
Refuted on the harness (d3/d5/d6.py): a supply multiplier on over (moves both sides, R3 stays level); taking the
draft level off comps too (picks collapse, R2 fails). Rounds 2+ only, median, x0.5 all also pass; all-rounds mean
chosen because it has no cut-off to defend. Players the model no longer values have no position, so they are not
in a position's mean. Desk bug fixed: tune() returned nil -> JSON null -> frontend refused the reading on a level
trade with no suggestions. Desk UI: click a shelf row, then ADD TO TRADE (1690,1009); CLEAR (1484,1055); PICKS
filter (514,310). Screenshot tool on Claude-OS is scrot (no ImageMagick). Claude-OS was rebooted 11-11 ~02:55 CT.
On screen (NE reads win-now by record): R1 -3.2/+2.6, R2 +0.0/+0.0 BOTH WIN, R3 -2.6/+0.9. Shots in Beelink
~/Downloads/TWR-desk-read{1,2,3}-draft-scarcity.png. Go test numbers (them = retool): R1 -3.2/+2.2, R2 0.0/+0.2,
R3 -2.6/+1.1, R4 +15.8/-16.0.

## IN-FLIGHT
Nothing. Test app stopped. Xvfb :5 up on Claude-OS. Nothing on his :0. Both Luna dispatches finished.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Merged: rail, League Pulse (PR #223). Now: War
Room trade desk, "the biggest selling point"; current work is making its math match Christopher's human reads.
Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
`session/war-room-trade-desk`, code tip 5b21f90b (+ resume commits). No git remote there: bundle -> scp via CT105
-> Beelink worktree ~/work/TheWarRoom/.worktrees/ui (MINE) -> push. NEVER touch .worktrees/website (Tom's) or the
main checkout. He merges after his look on :0.
Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache VITEST_MAX_THREADS=2
VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective -c user.email=secureprospective@users.noreply.github.com
-F msgfile, trailers Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line. Never --no-verify.
Lint limits that bit: funlen 60 lines / 40 statements, gocyclo 15, gochecknoglobals, exhaustive, gosec G101.
Write tool on this file: Read it first or the write is refused.

## 2. Harness (the fast way to test trade math — use it before touching Go)
CT105 scratchpad /tmp/claude-0/-root/84aad0c9-c764-46a4-bdd1-488ea2f3d933/scratchpad/scar/ :
- lens.py = Python copy of tradelens AS IT WAS BEFORE 5b21f90b (floor at teams x seats, bar on picks). x1.py `mk(floor=
  'roster', rel=KEEP, nobar=True)` builds the engine as it is NOW. `report(E)` prints R1..R4h with PASS/FAIL.
- lens.json = anonymous real lens input (f01 = Vikings/franchise.24, f02 = New England/.02, f03 = franchise.21, a
  QB-needy team with early picks). Tokens: Brissett a0122, Downs a2031, Herbert a0543. Each pick has Slot + Comps.
- reads.json = the four proposals. seasons.csv = 9,388 anonymous player-seasons 2021-26 (pid, pos, season, points,
  games, age, draft_round, rookie_season). rel.py / rel2.py / rel3.py = keep measurements. x7.py = shipped keep values.
- Shipped keep (in x7.py): CB .671 DE .661 DT .675 PK .185 LB .743 QB .496 RB .806 S .634 TE .811 WR .797.
To check real Go on the real league: temp-patch market/trade_lens.go to json-dump `in` before tradelens.Evaluate to
/tmp/twr-lens-in.json, make build, launch, open War Room, kill; a temp _test.go in internal/tradelens loads the dump
and runs Evaluate (real ids: Vikings franchise.24, NE franchise.02, Brissett 12616, Downs 17563, Herbert 14779,
R4 partner franchise.21). Remove patch, test file and dump after. Done once; numbers matched the harness.
App walk: `make build` (~50 s); `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` (display :5); wait for window "The War
Room" + 95 s; windowmove 0 0, windowsize 1920 1080; click War Room (86,201); wait 16-20 s. Stop: kill $(pgrep -x
thewarroom). Desk clicks: partner select (1432,285) then Down/Down/Return 0.5 s apart (= New England). Shelf rows
move with values: screenshot before clicking. Show him: scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads.
Bee: brief file in Beelink ~/fleet/briefs, run dir ~/fleet/runs/<name>, launch pi directly with ssh -f (model
gpt-6-luna, --thinking high), sentinel file, done-watcher in the same step. Real league data never leaves the lab:
anonymous ids only.

## 3. Gates
| Gate | State |
|---|---|
| go build, make build, go test tradelens + model/fit + store/params + app/market | pass |
| pre-commit hook on 40f40b6d and 5b21f90b | pass |
| Real Go on real league = Python harness | MATCHED |
| Read 1 (a 1st >> Brissett) | PASS: you -3.2, them +1.5 |
| Read 2 (2nd + two 3rds ~ a 1st) | PASS: 0.0 / +0.2 |
| Read 3 (NE snatches 2028 1st + 2027 2nd for Downs) | PASS: you -2.6, them +1.1 (329602e6) |
| Read 4 (Herbert ~ two 1sts + two 2nds, QB-needy partner) | FAIL: +15.8 / -16.0 |
| Screenshots of reads 1-3 on the new math | TAKEN, sent (read 4 not staged) |
| tsc, vitest, make verify, wire snapshots, push, PR | NOT RUN |

## 4. What exists (committed on the branch, NOT pushed)
7055d9c5 desk + direction; fcc8cb9d, 8c527bc6; 1ec399fc weights 65/50/30; df4b06a7 pick value; a734abe4 scarcity;
0e134734 slots by record; 40f40b6d kicker fix (model says K, lineup says PK: market.lensPosition / modelPosition);
5b21f90b:
- tradelens.replacement: floor = league[rostered] (best player left once the league has taken as many as it rosters
  at the position; free agents are in the pool). No seat math there any more.
- roster.over = keep[pos] x max(0, lens value - floor); Input.Keep map (missing position = 1).
- roster.worth(pick) = Weight x best group's mean of comps' over. NO roster bar. best() returns total only.
- internal/model/fit/keep.go fitKeep: slope of next-season PPG on this season's, pairs with 6+ games in both, clamp
  0..1. fit.Result.Keep. tools/fit writes key `trade.keep` per position (fittedValues helper). params.KeyTradeKeep;
  fitted.go describes it; ONLY the trade.keep rows were merged into fitted.json (rerunning the tool drifts 189 other
  values because the data differs from the 10-27 fit: do not regenerate the whole file).
- market.lensLeague fills in.Keep from src.Params().GetPosition(KeyTradeKeep, modelPosition(slot)).
- Fit data lives on the Beelink: ~/scratch/twr-research/fit.db (+ -wal) and players.json (has
  outcome.weekly_fantasy_points 2021-25; no Claude-OS database has that measure). Copy to Claude-OS, run
  `go run ./tools/fit -db .. -players .. -params /tmp/x.json -report /tmp/x.md`, delete the copy.

## 5. Open problems
- Read 3: level, not a snatch. Next fix = draft scarcity (his order).
- Read 4: player model, not scarcity. Backup and free-agent QBs carry points-per-game-when-playing values (best
  free-agent QB reads 14.8; the roster-line QB reads 8-9), so the QB floor is far too high; QB keep is only .50.
  Needs expected weekly points (ppg x chance of playing). Veteran work he said to do together. He now says draft
  scarcity is "the last need for parity": check read 4 again after it and tell him plainly if it still fails.
- Not done for 5b21f90b: Fit_Report.md keep table; parameter_words entry for trade.keep (settings-screen wording).
- Two pick valuers (pickvalue v2 for power rankings vs lens comps); TradeLensSource.Players unused; pickMarket 1.2
  should become a league setting; one >110-char line in pickComps; power rankings rework after trade value.

## 6. Established: do not redo
- Luna round 1 (per-position multiplier QB 1.32 .. PK 0.56 from surplus per seat) and round 2 (no passing formula):
  Beelink ~/fleet/runs/bee-twr-scarcity-1111 and -r2-1111. A 0.5-1.5 multiplier fails reads 2-4. Do not re-dispatch
  the same question; Luna quit in 2-3 minutes both times even with a runnable harness.
- Floor at teams x seats: every QB near 0. Floor at best free agent: QB floor 14.8 (worse). Roster line is the keeper.
- Keep measured in percentile units, or only on the top half: S comes out .67-.77 and read 3 fails. PPG slope over
  all regulars is what shipped. The safety figure swings .5-.65 with how games are counted; data alone will not
  make read 3 a snatch.
- The model's own trust (17 games vs k_dynasty: QB .86 S .61 CB .63): read 3 level, read 1 both lose. Not used.
- Bar on picks is what collapsed rounds 2-5; removing it fixed read 2.
- GPT/Gemini pick tables: none fits the 123 trades. Do not hardcode a pick table.
- pickMarket 1.2 measured on 43 pick-for-player trades (median 1.15-1.2). Next draft = no wait, 0.75 per later one.
- Slot-by-record (guessSlots) built; first 16 of round 1 vs second 16 is his ruling.

## 7. Decisions (Christopher)
- Weights .65/.50/.30. Human behaviour = one average multiplier, never the bounds.
- Scarcity belongs in the trade calculator and draft value, not power rankings; scaled by roster requirements; log
  league-size findings, build no scale.
- Do not mark QBs and safeties independently: one formula for every position.
- Keep source: he chose "A (recommended)" = measured from history (11-10).
- NEXT (11-10, verbatim): "it looks like we do need to do a draft scarcity, which should be easier with it being a
  finiet supply, and I am confident thats the last need for parity."
- His reason on read 3: "Starting safeties are plentiful in every draft into the late 2nd, sometimes late 3rd."
- Read 4: Herbert ~ two 1sts + two 2nds, confirmed.
- "We need to update how the power rankings work when we get this ironed out."
- Standing: mock is the gate; he merges; stop rule (new ledger event, schema, import or permission change -> ask).
- His four reads: memory project_warroom_trade_value_human_anchors.md.

## 8. Scarcity-scale findings log (do NOT build)
Roster-line replacement moves with league size and roster limits by itself. Keep shares are league-agnostic model
values. At 32 teams one-QB lineups still flatten QBs because backups are overvalued by the player model.

## 9. NEXT ACTIONS, in order
1. Design draft scarcity on the harness first (Python, seconds per try). Idea to test: a position's edge is worth
   less the more of its starters each draft supplies cheaply. Supply is finite and countable: per position, from
   seasons.csv (draft_round, rookie_season), the share of starter-level seasons produced by players drafted after
   round 1 (Luna's table, share from round 1: QB .69 DE .40 WR .27 DT .26 CB .21 RB .17 LB .16 TE .14 S .11 PK 0), or
   starters arriving per draft against starter seats (teams x lineup). One formula, all positions, constants fitted
   by tools/fit like trade.keep. Target: R3 them >= +0.5 with R1 and R2 still passing; watch RB/TE/LB, which also
   draft late, and the top-30 list (lens.top). Report R4 honestly.
2. Code it beside keep (fit measure -> fitted.json rows only -> Input -> over), tests, make build.
3. Check real Go on the real league (section 2 method), then stage the four reads on :5 and send screenshots with
   a plain pass/fail table.
4. Finish 5b21f90b's loose ends: Fit_Report keep table, parameter_words for trade.keep.
5. If read 4 still fails: veteran / backup-QB value session with him. Then power rankings rework.
6. Before push: tsc + vitest, make verify, wire snapshots, 110-char check, bundle -> Beelink -> push -> PR.

## 10. Honest status
Reads 1 and 2 pass on the real league in real code; read 3 is level; read 4 fails for a player-model reason that
draft scarcity may not reach. No screenshots exist for the new math. Frontend checks and the full suite have not
run since before the pick-value work. Nothing is pushed.
