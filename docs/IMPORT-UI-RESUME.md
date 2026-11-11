# TheWarRoom — UI-to-mocks RESUME (compact-safe #11, 2026-11-11 ~03:35 CT) — TRADE DESK: PUSH BATCH IN PROGRESS

## 0. FIRST THING AFTER COMPACT
Finish the push batch (section 9). One hand-run check job is in flight on Claude-OS (see IN-FLIGHT); read its log
first. Christopher graded the reads: 1 "A+", 2 "A+", 3 "A-, within error", 4 "I would move on, we can figure the
gap out later. We are hitting on what matters." Then: "work though this the best way possible, update the Go repo
if you think that fixes future issues as well" (= do the Go 1.26.11 bump).

## PUSH BATCH STATE (11-11)
Branch tip afc79cc0 (Claude-OS only, NOT pushed). Commits since f29c09b4: 329602e6 draft scarcity; e58ae54a tune
null-list fix; 3075f459 resume; c10fa999 Fit_Report keep table + parameter_words trade.keep; 14028654 keep.go
folded into fit.go (bloat tiny_files ratchet); 05d8151d placement.tsv row internal/tradelens -> trades; afc79cc0
boundary-adapters.tsv fingerprints for command_bindings.go and commissioner_host.go (straight wiring from 7055d9c5,
reviewed, say so in the PR).
`make verify` order: lint (ifaceguard, filelen, bloat, placement, frontend-transport, boundary, golangci) ->
check-wails-version -> govulncheck -> bindings-fresh -> tsc -> go test -race -> pnpm test/build/budget/csp.
All of lint passes at afc79cc0. KNOWN FAILURES still to fix:
1. govulncheck: GO-2026-6859 (net/http, stdlib), found in go1.26.10, fixed in go1.26.11. Not from this branch;
   CI (verify.yml uses go-version-file: go.mod, govulncheck v1.9.0) will fail on any PR. FIX = bump to 1.26.11 as
   its own commit on this branch: go.mod line 3, deploy/server/bundle.sh:13 (GOTOOLCHAIN), CLAUDE.md lines 56-57.
   Leave old resume/benchmark docs. Claude-OS has only /usr/local/go 1.26.10: use GOTOOLCHAIN=go1.26.11 (it
   downloads). RISK: golangci-lint panicked on other Go versions before (lesson: precommit needs the pinned
   toolchain); check the hook passes under 1.26.11, and update the env line in section 1 if it does. Beelink
   needs the same toolchain for its pre-push hook.
2. go test -race: internal/app/commissioner TestReleaseMatrixCatalogRolesAndDesktopDoors:
   "release_roles_matrix_test.go:75: catalog mutation commish.franchise.direction.set has no release matrix
   fixture". FIX = add a fixture row for the direction command in that matrix (owner of the franchise may; other
   owners may not; co-commish and up may), following the neighbouring rows. Rest of the race run: see the log.
Frontend tsc / vitest / build checks: result in the log (not seen yet when this was written).

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
ONE job on Claude-OS, started ~03:21 CT: bindings-fresh + tsc + `go test -race -timeout 60m ./...` + frontend
tests/build/budget/csp, by hand (the part of verify after govulncheck). Log ~/scratch/desk-verify2.log, last line
"exit N". Alive? `ssh claudeos 'ps -eo etime,args | grep -E "[g]o test -race|[v]itest"' < /dev/null`. It dies with
the CT105 shell if the session ends; just rerun the failed part. Result use: fix every FAIL, then one final verify.
Earlier full-verify log: ~/scratch/desk-verify.log. Test app stopped. Nothing on his :0. Xvfb :5 up.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Merged: rail, League Pulse (PR #223). Now: War
Room trade desk, "the biggest selling point"; current work is making its math match Christopher's human reads.
Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
`session/war-room-trade-desk`, tip afc79cc0. No git remote there: bundle -> scp via CT105
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
1. Read ~/scratch/desk-verify2.log on Claude-OS: list every FAIL (go and frontend).
2. Add the release-matrix fixture for commish.franchise.direction.set; run that package's tests.
3. Bump Go to 1.26.11 (three live pins, own commit); govulncheck clean; hook passes.
4. Fix anything else the log shows (wire snapshots, vitest), then ONE full `make verify` to exit 0.
5. Update this resume, bundle -> scp via CT105 -> Beelink worktree ~/work/TheWarRoom/.worktrees/ui (fresh worktree
   needs frontend/dist: pnpm install + build before push; use git -C) -> push -> open PR. PR body: what the desk
   math does, the four reads table, read 4 known fail (his ruling: move on), the two boundary fingerprints and why,
   the Go bump and why, filelen notes if asked. He merges after his look on :0 (squash).
6. Put the desk on his :0 for his look only when he asks or the PR is open (never claim what he sees).
7. Later, with him: veteran / backup-QB value (read 4), then power rankings rework.

## 10. Honest status
Reads 1-3 pass on the real league, in the app, and he graded them A+/A+/A-. Read 4 fails; he ruled move on.
Lint passes. The vulnerability scan and one Go test fail for the two known reasons above; frontend checks were
still running. Nothing is pushed. ETA to an open PR once resumed: about 40 minutes if the Go bump is clean.
