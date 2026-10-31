# TheWarRoom UI/UX build — RESUME #11 (2026-10-31 ~15:15 CT, compact-safe)

## 1. WHAT WE ARE DOING (Christopher re-scoped 10-31 — READ FIRST)
Make the five LOCKED mocks work in the REAL app from the EXISTING backend. Christopher 15:00: "We spent a full day and millions of tokens on the mock ups. Get the real app on target. That means everything, and functional." Mocks docs/ui/mocks/: franchise-hq-mock-v10, warroom-mock-v6, league-pulse-mock-v14, home-mock-v5, control-room-mock-v1 — no deviation. Engines/models/stores WORK: never rebuild/re-fit; add only thin reads that CALL existing code. Order: HQ → War Room → Home → Pulse → Control Room.
Repo: Beelink `chris@192.168.1.191:~/work/TheWarRoom` (key /root/.ssh/beelink). main = origin/main a16653cd (another session owns the main checkout — don't touch). Each slice = worktree `.worktrees/<name>`, local branch, no push. Fresh worktree: `cd frontend && pnpm install --frozen-lockfile && pnpm build` before commits. Commit with GOTOOLCHAIN=go1.26.10 (+ GOLANGCI_LINT_CACHE per wt), never --no-verify, trailers Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com> + Claude-Session: https://claude.ai/code/session_01QmkZHtxiunWcQ5mSqpHmtv. go is not on ssh PATH: `export PATH=$PATH:/usr/local/go/bin`.

## 2. AGENTS
- GPT: `~/fleet/bin/gpt-build.sh MODEL(sol|terra) THINK WT_NAME BRIEF RUN` → ~/fleet/runs/<RUN>/ (completion.txt, stdout.log). Launch `ssh -f … 'nohup … >/dev/null 2>&1 &'`. **5h window had 10% at 14:35, resets ~18:16 CT; weekly 70% left (reset 6d17h — the weekly reset already happened).** Sol high for everything WarRoom now. Terra FAILED trade-vis 4× (0 commits; stops on design questions) — do not give Terra multi-file/permission work.
- DeepSeek free: `~/fleet/bin/run-glm.sh <wt-path> <brief> <run> opencode/deepseek-v4.1-flash-free` → ~/fleet/runs/glm-<run>/. Works well SINGLE-lane (D20-fix, D45, D15, D13, recon all good). **Stalled 3× when 2 ran in parallel** (D26 earlier, DIV+FORM today) → one lane only; stalled = stderr mtime stops.
- Tom (Opus) weekly resets Nov 1 03:00 CT = fallback for hard work.
- New-binding checklist (builders missed it twice): desktop.go + doors-before.json + testdata/bindings/manifest.txt + parity count in internal/composition/command_parity_test.go + boundary tsv (`go run ./tools/boundarycheck -inventory`) + run go test ./internal/composition/ ./internal/serverhttp/ ./internal/commandcatalog/.
- Killing a run: `pgrep -f` pattern over ssh kills my own ssh shell — read pid.txt / match /proc cmdline, kill exact PID.

## 3. STATUS
| Slice | Branch | State |
|---|---|---|
| HQ backend D20 runway, D21 native moves, D26 age arc, D45 notes, D15 return week, D13 usage+matchup | all combined on **build/HQ-wire** (.worktrees/hq-wire) | ACCEPTED, reviewed, tests run by me. Golden reconcile 7124317f |
| HQ wiring (Sol) | build/HQ-wire 35cfd303, a421341e, 48ffff78, a8f32ad6; report docs/build-handoffs/HQ-WIRE-report.md | layout matches mock; my fixture walk found gaps → HQ-FIX |
| HQ-FIX (Sol) | build/HQ-wire: A inline taxi/IR b8334d46, B division name cebe0f69, C 5-week form 207f44f3 DONE | **D–I NOT DONE** (stopped to save quota). Brief ~/fleet/briefs/build/HQ-FIX.md (resume note appended: continue at D). Relaunch at 18:17 CT |
| Trade offer visibility | build/trade-visibility 2469f208 (Sol) | ACCEPTED. setting trades.offersVisible private/open, default private (Christopher reconfirmed). Debt: co-owners (identity model has none); speed: tradeEntryVisible replays all records per entry (fix only if ledger screen measures slow) |
| B3 live drives | build/B3-live | ACCEPTED; HQ live badges (HQ-FIX F) may need it merged |
| B2 tiebreak | build/B2-tiebreak | partial (reuse standings_rank.go, h2h subgroup bug, coin flip at commish finalize) |
| D11 projection ranges | — | only HQ item needing a NEW model → written method + replay proof before building |
| A1 split, A4 goldens | parked | |
| C1/E3 | stopped (rebuilds) | |

## 4. ARTIFACTS
- Screens: scratchpad hq-1920x1080.png, hq-2550x1352.png (Sol fixture), mock-2550.png. Sol's fixture harness: hq-wire/frontend/.tmp/hq/visual.html + native-fixture.json. Serve: `npx vite --host 0.0.0.0 --port 5199` in frontend on Beelink; LAN port is blocked → tunnel `ssh -f -N -L 5199:127.0.0.1:5199`; screenshot with CT105 `env DBUS_SESSION_BUS_ADDRESS=disabled: google-chrome --headless=new --no-sandbox --window-size=W,H --screenshot=… http://127.0.0.1:5199/.tmp/hq/visual.html`. Kill vite + tunnel after.
- Sol logs from D21: ~/logs/TheWarRoom/hq-d21-sol/.
- League preview HTML: Beelink ~/Downloads/TheWarRoom-preview.html.

## 5. FINDINGS FROM MY HQ WALK (fixture, vs mock at 2550)
Missing then: inline taxi/IR (now A), division id (now B), form bars (now C). Fixture too thin to judge numbers. "OPEN MFL" and "lineup.swap out=… in=…" ARE in the mock (not bugs). Cap reading counting expired contracts = intended pending-decision state (verified rollover.go:122-135), not a bug.

## 6. REFUTED / DON'T REDO
- Don't re-fit age arc (internal/model/params.go ArcStep/Survives). Grep internal/model, modelrun, m2service before any model work.
- Don't give DeepSeek work overlapping a Sol run's brief (DIV/FORM duplicated Sol's A–C; I misread Sol's fresh commits as old).
- Don't edit the c1faa2aa savvy-progress copy; live = /root/.claude/dev-mods/savvy-progress.

## 7. DECISIONS (Christopher 10-31)
Scope above. Correctness first (option 1: wire with honest "Not available yet" only where data truly absent). Tiebreak: walk all real tiebreakers then coin flip recorded in ledger. Trade offers checkbox, DEFAULT PRIVATE (reconfirmed); open = "on the block" engagement driver — open offers must surface where owners look (Home/Pulse) when wired. Peak window 90%. Settings congruence everywhere.

## 8. LEDGER
All local branches; nothing pushed; main untouched. Resume committed on build/HQ-wire (docs/ui/RESUME-uiux.md).

## 9. NEXT ACTIONS
1. At 18:17 CT (timer may have died in compaction — check time): launch `gpt-build.sh sol high hq-wire ~/fleet/briefs/build/HQ-FIX.md sol-HQ-FIX2` + done-watcher. Check git status in hq-wire is clean first.
2. While waiting: nothing on DeepSeek that overlaps HQ-FIX. Candidate single-lane DeepSeek job: War Room backend gaps (D38 stars/lanes, D33 lineup delta, D27 needs — check catalog + existing code first; D27 may be a new model → plan to Christopher).
3. When HQ-FIX lands: review commits; re-run fixture walk at both sizes vs mock; then run HQ against Christopher's REAL MFL league (find real-data path: browser fixture mode snapshot.json / hosted server / Claude-OS) and walk it; list wrong numbers.
4. Merge plan: combine trade-visibility + B3 into the HQ line when wiring needs them.
5. Then Christopher click-test HQ; then War Room.

## 10. HONEST STATUS
HQ backend done; HQ screen layout done; interactions (drag/cascade/save, matchup win %, live, drawer actions, tabs) NOT done; zero real-league verification yet. ETA HQ functional ~Nov 1–2 if Sol's evening window completes D–I.

## 12. UPDATE 16:10 CT (after compaction #11) — SUPERSEDES §3/§9 where they differ
- OpenCode free (DeepSeek AND GLM) went SILENT 15:10–15:35 (0-byte output; `opencode run "pong"` returns nothing). Ping before any dispatch. WR-D38 DeepSeek runs killed; logs ~/fleet/runs/glm-WR-D38-stall1, glm-WR-D38.
- build/HQ-wire b519cf43: research wire golden regenerated (Sol's 207f44f3 added "Form · Wk N" rows to the SHARED research player reading — additive, visible on research screen too).
- build/WR-D38 (.worktrees/wr-d38) 52f7822d, rebased on b519cf43: personal store action "swap" + RawChange.With for kind watch (rank targets within a position lane). Stars = existing watch add/remove; bid state = existing TargetNativeBids (yourBid/youLead; blind = own bid only). Need tag (D27) + est. win (D35) have no source → show "not available". No new binding. 20 tests PASS -race. FRONTEND NOT WIRED.
- build/D11-replay (.worktrees/d11-replay), rebased on b519cf43: 4b2013db/031927dd harness+proof (docs/ui/data-catalog/D11-REPLAY.md), ae1653bc+83f9e7d6 internal/projection (Params/DefaultParams/NewShape/Project) + store/history NativeProjections + binding TargetNativeProjections (projections.read, league-view G0, players = comma string ≤120, native.* leagues only) + all checklist items (doors 283, manifest, release_matrix inventory, bindings fresh), 32fe5fc8 memo keyed league|season|sha256(json rules.Scoring)|file shas. Contract batch all ok (composition, serverhttp, commandcatalog, app/commissioner, leagueservice, store/history, projection).
- Replay results: coverage 84% PASS; Spearman QB 0.429 FAIL (bar .45), RB .638/WR .535/TE .533 PASS; shrink beats raw 9/10 (TE fails); matchup step adds nothing (cap .10 → .430; .20 hurts) → not shipped; current-season-only (prior 0) worse → prior seasons required. ROS-vs-final-PF bar NOT run. Debt: first-call latency on a real league unmeasured; positions from nflverse labels, not league's.
- Harness: `go run ./tools/d11replay -dir ~/fleet/runs/config-study-p5-2026-10-26/cache [-prior -half -min -cap]` ~5 s.

## NEXT ACTIONS (now)
1. 18:17 CT: `gpt-build.sh sol high hq-wire ~/fleet/briefs/build/HQ-FIX.md sol-HQ-FIX2` + done-watcher (hq-wire clean at b519cf43; Sol must not regress the golden).
2. Me meanwhile: D33 lineup delta (method MODEL_METHODS_DRAFT §6; reuse powerrankings.Lineup + m2service.LineupRules; candidate value = D11 median via store NativeProjections) — on a new worktree off build/D11-replay; replay proof per §6 bars.
3. After HQ-FIX: merge D11 + WR-D38 into HQ line; wire D11 ranges into HQ drawer; fixture walk both sizes; real-league walk.

## 13. UPDATE 16:36 CT (after compaction #12)
- build/D33-delta (.worktrees/d33) off build/D11-replay: 36751f06 internal/lineupdelta (Rules adapter leaguerules→powerrankings, Compute before/after both parties: enters/leaves/benched/promoted, perWeek + thisWeek w/o bye/Out, unprojected named) + store/history NativeLineupDelta(TradeSides) + leagueservice RawLineupDeltaScope + binding TargetNativeLineupDelta (lineupdelta.read, franchise-view, G0; doors 285, schema 530fde5d…, manifest sorted, release matrix, boundary tsv refreshed). Contract batch all ok. c69dce27 tools/d33replay + docs/ui/data-catalog/D33-REPLAY.md.
- D33 replay on REAL league 14432 MFL exports (~/fleet/runs/d33-replay, 2023–25, 164 sides): (i) PASS 73% p=.002, (ii) PASS r=.548 slope 1.06 (trade's own effect; raw MFL optimal change r=.22 noisy), (iii) FAIL 64% (owners keep starting 4/10) → tile wording "your best lineup would bench X", (iv) PASS 100%.
- League 14432 MFL exports work read-only without login (transactions, weeklyResults, players, league).
- D28 RULING 10-31: owner selects posture (contend/retool/rebuild); no classifier; lens λ follows the pick. Open: public or private label.

## 14. UPDATE 16:50 CT — SUPERSEDES earlier NEXT ACTIONS
- build/D33-delta = build/WR-base (649fefbf): D11 + D33 (36751f06, replay c69dce27) + D38 swap (b730ed4e) + R3 bid cap fix 649fefbf (internal/commissioner/market/bid_cap.go openBidsFit: other standing bids count against cap, cap only, matched by Side+Player; test composition TestNativeOpenBidsTogetherMustFitTheCap, proven failing without fix). Regression batch all ok (composition, leagueservice, commissioner/..., store/history, nativecheck).
- build/WR-wire (.worktrees/wr-wire) = HQ-wire 08dd9975 + all WR-base commits cherry-picked (top a3ea876b), frontend dist built, bindings fresh. Brief ~/fleet/briefs/build/WR-WIRE.md (5.2 KB).
- Brief ~/fleet/briefs/build/A12-withdraw.md for Tom (owners can't withdraw native bids; commish removal via recorded correction or STOP). Tom resets Nov 1 03:00 CT. Works in .worktrees/d33.
- D25 Best 21 EXISTS (Race power mode topn). D27/D28 replay needs as-of model runs (modelrun.Spec{Season,AsOf} supports it) but no DB with held features found on Beelink (archived DBs lack model tables) → locate on staging VM before D27.
- RULING 10-31: D28 posture = OWNER SELECTS contend/retool/rebuild; lens λ .65/.50/.30 follows it; no classifier. OPEN QUESTION asked: is the chosen posture public or private? Await answer.
- Gate lesson: internal/commissioner TestPlayerLookupsNameASide forbids player lookup by id alone (mirrored leagues) — match side too.

## NEXT ACTIONS (now)
1. At 18:17 CT (timer buytayx5b; if dead check time): launch BOTH with done-watchers:
   `ssh -f … 'nohup ~/fleet/bin/gpt-build.sh sol high hq-wire ~/fleet/briefs/build/HQ-FIX.md sol-HQ-FIX2 >/dev/null 2>&1 &'`
   `ssh -f … 'nohup ~/fleet/bin/gpt-build.sh sol high wr-wire ~/fleet/briefs/build/WR-WIRE.md sol-WR-WIRE >/dev/null 2>&1 &'`
   Watch ~/fleet/runs/<run>/completion.txt.
2. Get Christopher's answer on posture public/private, then build D28 store (owner setting) + lens.
3. Nov 1 03:00: dispatch Tom on A12-withdraw.md.
4. Review Sol outputs; fixture walk both sizes; real-league walk; Christopher click test.
5. D27: find held-feature DB (staging VM), then method + as-of replay.

## 15. UPDATE 17:30 CT
- D27 committed 72f3ee4f (internal/teamneeds + tools/d27replay + D27-REPLAY.md). Results:
  - The method FAILS as a predictor: the surplus check got 65% (bar 70%), the needs check 12% (bar 50%).
  - It ships descriptive only, with no grade words.
  - The pick-equivalence check was not run. The DB copy is kept at ~/fleet/runs/d27-replay/history-copy.db.
- RULING: the posture label is PUBLIC to the whole league (Win now/Retool/Rebuild). Brief ~/fleet/briefs/build/D28-posture.md is for Tom, after A12, in .worktrees/d33.
- NEXT: 18:17 launch both Sol lanes; 03:00 Tom runs A12 then D28.
- 17:35 CT:
  - Sol lanes are armed on the Beelink: ~/fleet/bin/sol-at-1817.sh (pid in ~/fleet/runs/sol-at-1817.pid).
  - It launches both lanes at 18:20 CT and retries each lane up to 3 times, 15 min apart, if it hits the usage limit.
  - DO NOT launch by hand. My watcher is b0qh8u29g; it waits on that PID.
  - Baseline at both heads: go build ok, tsc ok, worktrees clean.

## NEXT ACTIONS (17:40 CT, supersedes)
1. When watcher b0qh8u29g fires (or check: `kill -0 $(cat ~/fleet/runs/sol-at-1817.pid)`):
   - Read ~/fleet/runs/sol-HQ-FIX2 and ~/fleet/runs/sol-WR-WIRE (completion.txt, stdout.log).
   - If stderr shows "usage limit" 3 times, relaunch after the next reset.
2. Review Sol commits (start heads 08dd9975 HQ, a3ea876b WR). Then:
   - Fixture walk at 1920 and 2550 against the mocks.
   - Real-league walk.
   - List wrong numbers.
3. Nov 1 03:00 CT: dispatch Tom on ~/fleet/briefs/build/A12-withdraw.md, then D28-posture.md (.worktrees/d33).
4. D27 pick-equivalence bar: fit pickvalue as of each date. DB copy at ~/fleet/runs/d27-replay/history-copy.db (412 MB; delete once done).
5. Christopher click test HQ, then War Room.
