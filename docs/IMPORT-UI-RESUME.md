# TheWarRoom — UI-to-mocks RESUME (compact-safe #7, 2026-11-10 afternoon CT) — WAR ROOM TRADE DESK

## 0. FIRST THING AFTER COMPACT
Trade desk engine + screen are WRITTEN and COMPILE (go build, tsc clean, make build OK) but have NEVER BEEN
SEEN running. Next action: launch on :5 with the real league, open War Room, screenshot, compare to the mock
(docs/ui/mocks/warroom-mock-v6.html), fix what is wrong, THEN show Christopher. He said: "We need to get the
TradeDesk working in the spirit of TheWarRoom ... execute with precision and purpose, excellence is the standard,
not the goal." and earlier "cover the bases before presenting anything to me". Nothing is running.

## IN-FLIGHT
Nothing. Test app stopped. Xvfb :5 left up on Claude-OS. Nothing on his :0.

## 1. What we are doing
Build the five screens to the locked mocks on the NATIVE league. Done + merged: rail, League Pulse (PR #223, main
57c86968). Now: War Room trade desk, "the biggest selling point": beat KeepTradeCut by using the league's REAL
rules. Work on Claude-OS (`ssh claudeos`, always `< /dev/null`): worktree ~/work/TheWarRoom/.worktrees/ui, branch
`session/war-room-trade-desk`. Claude-OS has NO git remote: bundle -> scp via CT105 -> Beelink worktree
~/work/TheWarRoom/.worktrees/ui (MINE) -> push (PATH + GOTOOLCHAIN=go1.26.10, frontend/dist built). gh is
~/.local/bin/gh on the Beelink. NEVER touch .worktrees/website (Tom's) or the main checkout. He merges (my
`gh pr merge` is refused by the auto-mode check).
Env: export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" GOLANGCI_LINT_CACHE=$PWD/.golangci-cache VITEST_MAX_THREADS=2
VITEST_MIN_THREADS=1. Commit: -c user.name=secureprospective -c user.email=secureprospective@users.noreply.github.com,
-F message file, trailers Co-Authored-By: Claude Fable 5.2 <noreply@anthropic.com> + Claude-Session line.
Pre-commit runs golangci-lint-full (3-5 min on Go changes). Never --no-verify.

## 2. Harness
Edits: python replace scripts in the scratchpad, scp to claudeos:/tmp/twr-*.py, run, rm by name.
- `make build` (~30 s). `UIP=ui-profile2 sh ~/scratch/ui-desk.sh` launches on :5 (real league). Wait ~150 s.
  W=$(xdotool search --onlyvisible --name "The War Room"|head -1); windowmove 0 0; windowsize 1920 1080; scrot -o.
  Stop: kill $(pgrep -x thewarroom) (never pkill -f).
- Click points on :5: Home (86,168) War Room (86,201) Franchise HQ (86,235) League Pulse (86,268) Control Room
  (86,302). War Room tabs: TRADE DESK (280,22) MARKET (378,22) DRAFT ROOM (476,22) PLANS (570,22). Partner select
  (1418,303). POCKET (278,1052) NEGOTIATIONS (375,1052). On his :0 everything sits ~63 px lower.
- His screen: same launch with DISPLAY=:0 XAUTHORITY=$HOME/.Xauthority, then wmctrl maximize. Leave Brave alone.
- Show him: live on :0, or scp -i /root/.ssh/beelink to chris@192.168.1.191:~/Downloads + file:// path.
- Regenerate after Go API/catalog changes: `for s in doors browser mfl native; do go run
  ./frontend/scripts/nativegen -surface $s; done` then `make bindings` (noisy: pipe through tail).
- Real value numbers without the app: ~/scratch/ui-profile2/**/prepared-values/native.mfl-14432-2026-2026.json.

## 3. Gates
| Gate | State |
|---|---|
| go build ./..., tsc --noEmit, make build | pass (this window) |
| go test tradelens, governance, commissioner, commandcatalog | pass (before the last small edits; rerun) |
| Seen running on the real league | NOT YET |
| Direction command test (owner own team yes / other team no / commish yes) | NOT WRITTEN: write it |
| Frontend tests for warroomLens parser | NOT WRITTEN |
| make verify, wire snapshots, 110-char check, push, PR | not started |
Push-gate lessons: run `make verify` on Claude-OS first; TWR_WRITE_WIRE=1 go test ./internal/store/history when a
read gains a field; grep -rnE "^.{111,}$" frontend/src/app; `as` must not start a wrapped line; catalog/bindings
snapshot tests will need the regenerated files committed (already regenerated).

## 4. What was built this window (all on the branch; see section 8 for commit state)
DESIGN (mine, under his "free hand" + rulings): an asset is worth what it adds to (or holds in) a side's BEST N
(league starter count, 21 in his league) under that side's direction. Lens value = w*nowPPG + (1-w)*dynastyPPG;
w = contend 0.6 / retool 0.3 / rebuild 0. Lens net = change in that side's best N, so two depth pieces never add
up to a star (our answer to KTC's crowd curve, from the league's own starter count). Picks are valued by the
existing Pick Value model (dynasty units, now = 0) and compete for the best N like players. Values shown in the
engine's own unit (points per game, one decimal), NOT the mock's 0-100.
Real-league facts: best-21 cutoff is ~7.5-9.5 PPG on strong teams, ~2.6-4 on weak ones.
- internal/tradelens/tradelens.go (+_test.go, 6 tests pass): Direction, NowWeight, Read (our read: record
  >= .600 contend, <= .350 rebuild, middle: older-than-median roster at >= .500 contend else retool; no games =
  retool), Evaluate -> Reading{you, them {direction,set,nowWeight,before,after,net,out,in,needs,spare}, assets
  [{token,now,dynasty,you,them,valued,words,age,ageKnown}], verdict empty|both|balk|bad, tune[trim|ask|sweeten],
  notes}. Needs: starter-grade = top (teams x position min) by dynasty, signed next season; have < min = hole;
  == min at a position that can start more = thin. Spare = more starter-grade than the position max.
- internal/app/market/trade_lens.go: TradeLens(ctx, TradeLensSource, scope) gathers values (prepared pass),
  ledger base, lineup rules, picks (NewNative + NativePickHistory), directions, standings.
- internal/leagueservice/trade_lens_api.go: RawTradeLensScope + TargetNativeTradeLens (admitRead "tradelens.read").
  Pick token = pick:season:round:originalFranchise. Catalog row tradelens.read (franchise-view), desktop.go row,
  command_bindings.go, app API method, composition commissionerHost.NativeTradeLens. Hosted server does NOT
  implement it (same as value.read): desktop only for now.
- DIRECTION LEDGER ENTRY (approved): command `commish.franchise.direction.set`, people family:
  RawPeople.Direction, governance.DirectionSet event, Franchise.Direction, State.Directions(),
  commissioner.Directions(records), DirectionWords, authorizeDirection (seat holder = owner; else falls to the
  CM-018 control = co-commissioner and up). G1, visibility league, effective now. Catalog row added.
- Frontend: shell/warroomLens.ts (parser, readWarRoomLens, setWarRoomDirection, words), localWorkflow.ts direct
  line, WarRoom.tsx (lens read, held last reading, direction choose, calendar strip shows only upcoming max 8,
  inspector gets lens), WarRoomHeader.tsx (direction buttons live, value-by-age bars, needs chips, "open now:"),
  WarRoomTrade.tsx rewritten (ALL/OFF/DEF/PICKS chips, shelves ranked by value, YOU/THEM columns, tray totals,
  two lens bars, verdict, BLOCKED rows from the trade preview, YOUR BEST N + PROJ/WEEK tiles, lineup panels,
  TUNE IT with APPLY, their read: direction badge, needs, spare, lean), WarRoomInspector.tsx (to you / to them,
  why line, fills need), warroom.css additions at the end.

## 5. Current bug
None known; nothing has been seen running yet. Watch on first run: lens read speed (it replays the ledger base,
reads all player records and fits the pick curve on every deal change; cache per league revision if slow);
whether Pick Value fits on the imported league (needs native rookie-draft history; else picks show "—" and a
note says why); zeros for bench players on shelves (correct by design: explain, don't hide).

## 6. Established: do not redo
- Engines exist: value.read (appvalue: nowPPG, dynastyPPG raw points per game), lineupdelta.read, trade.offer
  preview, pickvalue, powerrankings.Count("topn"). No posture/window engine existed.
- New commands go in: commandcatalog/native_reads.go or native_mutations.go, desktop.go, composition/
  command_bindings.go, app/commissioner API, then regenerate (section 2).
- useContractRead blanks its reading when the key changes: hold the last reading yourself if needed.
- Earlier: War Room parser fix (hostedReadings.ts), `pwr` is a grid-row class, re-import = re-sync.

## 7. Decisions (Christopher, 11-10)
- APPROVED: ledger entry "owner set their direction"; label public (win now / retool / rebuild) (ruled 10-31).
- AGREED: partner who has not picked shows OUR read, labelled as ours.
- GOOD: lens = blend of the two existing values, nothing fitted. I OWE HIM the exact weights (0.6 / 0.3 / 0) with
  real examples from his league before it ships, plus the best-N counting choice explained plainly.
- Free hand to tweak/add/rework; cover the bases before presenting; limit his turns.
- Crowd sentiment left out, not faked. Pulse leftovers parked (Hermes task list, SUPPORT).
- Standing: Christopher; outcome first, short plain replies; Central time; ETAs; ping when needed; mock is the
  gate; never invent states; don't hunt for issues; stay out of Tom's worktree.

## 8. Ledger state
Branch session/war-room-trade-desk: 7a562134 (parser fix + old resume) then the WIP commit made at this
compact-safe (see `git log --oneline -3`; if the pre-commit lint refused it, the work is uncommitted in the
worktree: fix lint, commit). Not pushed. Untracked: .golangci-cache/.

## 9. NEXT ACTIONS, in order
1. Rerun go tests (tradelens, governance, commissioner, commandcatalog, leagueservice, app/market); launch on :5;
   screenshot the desk empty, with a partner, with a 2-for-1 deal; fix against the mock.
2. Click a direction on the scratch league; confirm header, lens words and ledger line.
3. Write the direction authorization test and a warroomLens parser test.
4. Pick real examples from his league for the weights (a vet to a contender vs a rebuilder; a 2-for-1; a pick).
5. Show him on :0 with the weights and examples; one full presentation.
6. Then: inspector polish, pocket / versions / send / negotiations (plan steps 6-8 from the earlier resume).
7. Before push: make verify, wire snapshots, 110-char check. He merges.
Stop rule: any OTHER new ledger event, schema change or permission change -> tell him first.

## 10. Honest status
Code compiles and unit tests pass; no one has seen the desk run. The weights are my proposal, not yet shown.
Hosted (phone/PWA) leagues do not get the lens yet. Live NFL field in Pulse still unseen during a real game.

## 11. COMMIT STATE AT COMPACT-SAFE #7 (read with section 8)
The WIP commit was REFUSED by the pre-commit lint, so all code from section 4 is UNCOMMITTED in the Claude-OS
worktree (on disk, intact). Only this resume is committed. Fix these six, then commit everything:
- internal/commandcatalog/desktop.go:19 commissionerDesktop is 61 lines (limit 60): move the methods table out.
- gocyclo > 15: app/market/trade_lens.go TradeLens (16): split the gather step; commissioner/governance.go
  AuthorizeRecords (16): fold the direction check into a helper with the lineup one; tradelens.go Evaluate (17)
  and needs (18): split out roster building and the per-position count.
- internal/tradelens/tradelens_test.go:22 gosec G101 false positive on the field name Token with a "pick:..."
  string: rename the test tokens or build the string, do not suppress blindly.
