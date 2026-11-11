# TheWarRoom — UI-to-mocks RESUME (compact-safe #9, 2026-11-11 early CT) — TRADE DESK: PICK VALUE + SCARCITY

## 0. FIRST THING AFTER COMPACT
Do section 9 item 1: dispatch a Bee (Luna, high) positional-tier study, then code its ratio into the lens.
Christopher ordered it in his /compact-safe message (verbatim in section 7). Nothing is running.

## IN-FLIGHT
Nothing. Test app stopped. Xvfb :5 up on Claude-OS. Nothing on his :0. No dispatch launched yet.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Merged: rail, League Pulse (PR #223). Now: War
Room trade desk, "the biggest selling point"; current work is making its math match human judgement.
Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
`session/war-room-trade-desk`. No git remote there: bundle -> scp via CT105 -> Beelink worktree
~/work/TheWarRoom/.worktrees/ui (MINE) -> push. NEVER touch .worktrees/website (Tom's) or the main checkout
(GUIDES/ there is gitignored; GPT's pick reports live in GUIDES/pick-value-gpt-independent/, read only). He merges.
Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache VITEST_MAX_THREADS=2
VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective -c user.email=secureprospective@users.noreply.github.com
-F msgfile, trailers Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line. Never --no-verify.
Lint limits that bit: funlen 60, gocyclo 15 (lensPicks hit 16), gochecknoglobals, exhaustive, gosec G101.

## 2. Harness
Edits: python replace scripts in the scratchpad, scp to claudeos:/tmp/twr-*.py, run, rm by name.
- `make build` (~20 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on :5. Then:
  until W=$(xdotool search --onlyvisible --name "The War Room"|head -1) && [ -n "$W" ]; do sleep 3; done; sleep 95;
  windowmove 0 0; windowsize 1920 1080; click War Room (86,201); sleep 16. Screenshot: scrot -o /tmp/twr-x.png.
  Stop: kill $(pgrep -x thewarroom). Bare `sleep N &&` is blocked by the harness.
- Desk clicks on :5: partner select (1432,285), then keys Down / Down / Return with 0.5 s between (= New England;
  one more Down from there = Jets). Filter chips ALL (275,310) OFF (355,310) DEF (435,310) PICKS (513,310).
  My shelf row n: x=261, y=355+48*(n-1). Their shelf first row y=617 (601 when their panel is shorter), 48 apart;
  scroll it with xdotool click 5 at (1430,800). CLEAR (1484,1055). ROW ORDER MOVES WITH VALUES: always screenshot
  and read the rows before clicking. Values can re-run mid-walk (desk shows "Not available yet"); just retake.
- Check data: ~/scratch/ui-profile2/config/TheWarRoom/prepared-values/native.mfl-14432-2026-2026.json
  (reading.players[]: player, franchise, position, nowPPG, dynastyPPG, age, available). history.db: league_event
  (kind, payload; seed = kind commish.league.seed_held, has players[] and picks[]; lineup rules in
  commish.league.create; week lineups in commish.lineup.correct), model_scores (run_id, mfl_id, now, now_ppg,
  dynasty, dynasty_ppg, prior, past_games, season_games, z_past, z_now, on_field). Vikings = franchise.24,
  New England = franchise.02, Jets = franchise.03.
- Show him: scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads + file:// path.
- Bee dispatch: /fleet-dispatch skill (brief is a file, sized to window, STDOUT is findings, ssh -f launch, arm a
  done-watcher in the same step). Real league data must not leave the lab: give Bee aggregates or run it where the
  data already is; never commit it.

## 3. Gates
| Gate | State |
|---|---|
| go build, make build, go test tradelens + app/market | pass |
| pre-commit hook (lint, secrets) on each commit | pass |
| Pick math vs hand recompute on the real league (before scarcity) | MATCHED (Vikings 300.8, holds, NE 4.3) |
| Scarcity + slot-by-record numbers hand recomputed | NOT DONE |
| His read 1 (a 1st >> Brissett) | PASS |
| His read 2 (2nd + two 3rds for a 1st ~ even) | "isnt bad"; desk says they balk -2.0 |
| His read 3 (NE snatches 2028 1st + 2027 2nd for Downs) | FAIL, they -3.3 |
| His read 4 (Herbert ~ two 1sts + two 2nds) | FAIL, Herbert reads 0.0 |
| vitest, tsc after the label change; make verify; wire snapshots; push; PR | NOT RUN |

## 4. What exists (all committed on the branch, NOT pushed)
Code tip 0e134734. Chain: 7055d9c5 desk + direction; fcc8cb9d, 8c527bc6 checked vs real league; 1ec399fc weights
65/50/30; bd4f6c6e resume 8; df4b06a7 pick value; a734abe4 scarcity; 0e134734 slots by record.
- internal/tradelens/tradelens.go: Comp{Position, Now, Dynasty}; Asset.Comps [][]Comp + Weight; Direction.blend;
  replacement(in, d) = per position, the first player past teams x seats, seats = Min + flex shared by room
  (Max-Min) across slots, free agents in the pool; roster.floor, roster.over (lens value over replacement, min 0),
  roster.worth (pick = Weight x best group's mean of what each comp adds over the roster's N-th player);
  roster.count = best N players' over-replacement values + picks' worth. Tests cover all of it.
- internal/app/market/trade_lens.go: pickMarket const 1.2 (league multiplier, typical 2026 pick-for-player trade),
  takenPicks (seed's made picks -> today's values), pickComps (own round within teams/4 slots, then each later
  round so an earlier pick is never worth less), guessSlots (no draft order -> slot = teams - standings rank + 1),
  lensPicks (wait counted from the next open draft at params pickvalue.discount 0.75), lensPlayers adds valued
  free agents as franchise-less assets. Standings now read before picks.
- frontend WarRoomTrade.tsx: result panel label "YOUR EDGE ... best N over replacement, plus picks".

## 5. Open problems
- Reads 3 and 4 fail. Evidence: Downs S age 21, model prior .99, z_now .96 on 9 games -> S1, 5.6 over
  replacement to NE. Herbert now pct .30 on 8 games at z_now .78 vs 45 past games -> QB30, dyn 15.95 < Cousins (38)
  19.17. One-QB lineup at 32 teams: QB33 scores 14.7, so every QB sits near 0 (Lamar 2.0).
- Kickers read high (NE's Patterson 4.5 vs Downs 2.4 on the Vikings' side).
- Rounds 2-5 read 0.3 on a strong roster while a mid 1st reads 2.5: later picks look too cheap.
- A 0-value starter can show -0.0 held: his leaving lowers the bar that picks must beat.
- Two pick valuers now: pickvalue v2 (power rankings) and the lens comps. Reconcile when power rankings are redone.
- TradeLensSource.Players is now unused by the lens. pickMarket should become a league setting once approved.
- Lineup panels squeezed; repeated trades.read permission log lines (pre-existing, not chased).

## 6. Established: do not redo
- GPT/Gemini tables: no table fits the 123 trades; all confidence low. Do not hardcode a pick table.
- Existing pickvalue v2 fed the 2026 draft reads too flat (1.01 = 6.55 vs outcomes 11.7).
- Any flat pick value against Best 21 reads ~0 (median 21st man 7.1): picks must be priced on their spread.
- League multiplier measured on 43 pick-for-player trades: median 1.15-1.2, pooled 1.8-1.9, single trades 0-11,
  months 0.5-5.5. Steady by period only when the next draft counts as no wait.
- Seat rule checked vs week-9 real starts: within 0.3 except RB (1.59 vs 2.16) and WR (2.88 vs 3.28). A greedy
  fill by raw points was tried and dropped (it hands all flex to S/CB).
- Free agents in the replacement pool barely moved kickers.
- Slot-by-record made read 3 worse for a true reason: the Vikings' picks are late ones.
- dynastyPPG = 3 seasons, this one first, x0.75 each later, age curve, contract tilt. Header 281.2 is that
  long-term best 21, a different read from the panel's EDGE.

## 7. Decisions (Christopher)
- Weights .65/.50/.30, one place. DONE.
- "Human behavour is seasonal, and will hit the edges, use the human behavior as an averge multiplier, not a scope."
- "I lean into your recommendation for this test" (1.2, next draft = no wait, 0.75 after).
- "scarcity has to be a factor in the actual trade calculator"; not power rankings, "other than, say, like draft
  value". "scarcity needs to be scaled by roster requirments. We will start with 32 teams but not build the scale,
  only log your findings for future sessions."
- Round 1 in a 32-teamer: first 16 picks are a different tier; the second 16 "are more like premium 2nd round
  picks"; humans guess early/late from team of origin. BUILT (guessSlots).
- Read 4 confirmed: "correct: Herbert two 1sts 2 2nds".
- NEXT ORDER (11-11, verbatim): "I dont think you need to really mark: QBs and safeties independantly, what you
  need to do is spin up a Bee luna to do a quick study on player stats, have it judge tier levels of each position
  based on stat totals, based on the number of relavent starter and backup tiers to each position create a ratio
  that will filter the positional scarsity. If there is something im missing feel free to add somthing to that
  equation, but i think luna is more than capiable to do that simple math for you, then you code it."
- "We need to update how the power rankings work when we get this ironed out."
- Standing: mock is the gate; he merges; stop rule (new ledger event, schema, import or permission change -> ask).
- His four reads are in memory: project_warroom_trade_value_human_anchors.md. Every value change is checked
  against them before he sees it.

## 8. Scarcity-scale findings log (do NOT build; his ruling)
Replacement level already moves with league size and lineup rules through teams x seats, but nothing scales how
much scarcity counts. At 32 teams: one-QB lineups flatten every QB; kickers inflate because few are rostered;
S and CB are flat (S64 = 9.3 long term, QB64 = 5.1, RB64 = 5.1). A scale probably needs roster depth (bench
size), not only starters.

## 9. NEXT ACTIONS, in order
1. Write the Bee (Luna) brief and dispatch: per position, from player stat totals, judge tier levels (elite /
   starter / flex / backup / replacement), count the relevant starter and backup tiers for a 32-team league with
   this lineup (QB 1-1, RB 1-3, WR 2-5, TE 1-3, DT/DE/LB/CB/S 2-4, PK 1-1, 21 starters), and return one ratio per
   position that filters positional scarcity, with its arithmetic. My additions to offer: year-to-year stickiness
   of a position's top tier (why humans discount safeties) and how many seasons a position's starter lasts (why a
   franchise QB is worth two 1sts and two 2nds). Arm a done-watcher.
2. Check Luna's ratios against the four reads in Python on the values file before coding anything.
3. Code the ratio into tradelens (one place, beside replacement), tests, rebuild, re-stage all four reads,
   hand-recompute one, screenshots to his Downloads with a plain pass/fail table.
4. Then: veteran/half-season trust (z_now) with him if reads still fail; power rankings rework after that.
5. Before push: tsc + vitest, make verify on Claude-OS, wire snapshots if a read gained a field, 110-char check
   (one long line in pickComps), bundle -> Beelink -> push -> PR. He merges after his look on :0.

## 10. Honest status
Pick value and scarcity are built and unit-tested; only the pre-scarcity pick math was hand-checked on the real
league. Two of his four reads still fail and one is only "not bad". Proposals 1 and 4 were not re-staged after
the slot change. Frontend checks and the full suite have not run on today's work. Nothing is pushed.

## 11. UPDATE 2026-11-10 evening CT (after compact #9) — Luna study done, three reads pass on the harness
- Luna round 1 (ratios QB 1.32 .. PK 0.56, surplus per seat) and round 2 (no passing formula) are in Beelink
  ~/fleet/runs/bee-twr-scarcity-1111 and -r2-1111. Neither passes the reads. Do not re-dispatch the same question.
- Harness: CT105 scratchpad scar/ (lens.py = exact Python copy of tradelens, lens.json = anonymous real lens input,
  reads.json, x1..x5.py experiments, rel.py history measures, seasons.csv). Same files in the r2 run folder.
- Kicker bug fixed + committed 40f40b6d (model says K, lineup says PK; kickers had no replacement level).
- PASSES R1, R2, R3 (robust to both directions and +-20%): (1) replacement level = first player past the number
  ROSTERED at the position (not teams x seats); (2) over = keep[pos] x max(0, value - floor), keep = share of a
  starter's lead over replacement that carries to next season, measured from 2021-25 history: QB .66 RB .82 WR .87
  TE .81 DT .83 DE .67 LB .66 CB .45 S .54 PK .08; (3) pick worth = Weight x best group mean of comps' over, with NO
  roster bar. Edges: R1 -2.1/+0.4, R2 0.0/+0.1, R3 -1.9/+0.8.
- Alternative keep = the model's own fitted trust at 17 games from k_dynasty (QB .86 RB .78 WR .75 TE .73 LB .73 CB .63
  DT .61 S .61 DE .60 PK .11): wire-only, saner top list (QBs in top 10), but R1 them -1.0 and R3 them 0.0.
- R4 / R4h still fail in every version (picks beat a QB10-valued Herbert by 7-10). Cause is the player model, not
  scarcity: backup and free-agent QBs carry points-per-game-when-playing values (best free-agent QB reads 14.8,
  QB89 reads 8-9), so the QB floor is far too high. Needs expected weekly points (ppg x chance of playing). Part 7
  work with Christopher.
- NOT CODED YET: waiting on his ruling on which keep source, then code 1-3 in tradelens (Input.Keep map), tests,
  rebuild, re-stage the four reads on :5, screenshots.

## 12. UPDATE 2026-11-10 late CT — option A coded (Christopher chose A), commit 5b21f90b, NOT pushed
- Coded: replacement at the roster line; over = keep x lead; picks = mean comp worth, no roster bar; trade.keep per
  position fitted by tools/fit (slope of next-season PPG on this season's, 6+ games both) and shipped in fitted.json
  (only those rows merged; other fitted values untouched). Fit data: Beelink ~/scratch/twr-research/fit.db + players.json.
- Fitted keep: QB .50 RB .81 WR .80 TE .81 LB .74 DT .68 CB .67 DE .66 S .63 PK .18. These differ from my CSV
  measure (S .51, CB .44) because the fit data counts games differently; with the shipped values R3 is LEVEL.
- Real league, real Go code: R1 -3.2/+1.5 pass; R2 0.0/+0.2 pass; R3 -1.6/-0.2 (was +1.8/-3.8) not a snatch;
  R4 +15.8/-15.0 fail (player model: backup QB values).
- Not done: screenshots of the four reads, Fit_Report.md keep table, parameter_words entry for trade.keep, tsc/vitest,
  make verify, repo copy of this resume, push/PR.
