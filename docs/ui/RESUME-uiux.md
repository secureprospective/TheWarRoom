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
