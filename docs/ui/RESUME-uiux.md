# TheWarRoom UI/UX groundwork: RESUME (2026-10-30 night, compact-safe #7)

## 1. What we are doing
HTML mocks, one screen at a time; Christopher approves from screenshots/clicks.
- DONE: Franchise HQ v10, War Room v6, League Pulse v14, Home v5 (slate).
- **Control Room mock v1: in a good place** (Christopher 10-30: "really good start", "we are in a good place"). Last pass = de-slop + hover/focus states, commit 3c66f76c.
- DONE 10-30 eve: data-stub catalog of all 5 approved mocks, Claude + Terra independently, compared (commit 84bee24f, `docs/ui/data-catalog/`).
- **NEXT (Christopher, compact-safe #7 args): the DATA FLOW MAP.** "What we agreed to on the mockups is non-negotiable going into the build. Know the data flows: what's shared, what feeds into the next, what needs isolation." Plus address the two "open in both catalogs" items.
- He asked: "laser focused, maintain this level of excellence" next session.

Where: Beelink `chris@192.168.1.191:/home/chris/work/TheWarRoom`, worktree `.worktrees/uiux`, branch `session/ui-ux-groundwork` (local only, NOT pushed). Key `/root/.ssh/beelink`. Main checkout = other session (`session/league-history-pwa`), don't touch. Commit with `GOTOOLCHAIN=go1.26.10`, never `--no-verify`. Code truth lives on `origin/main` 364379ba — read with `git show origin/main:<path>` (the worktree base is old).

## 2. Harness (scratchpad `/tmp/claude-0/-root/ceeceeaf-c38e-4b3f-bddd-137d1de3f99a/scratchpad/`)
- Control Room source: `base.css cr.css cr.body.html` + JS `cr-rules.js cr1.js cr2.js cr4.js cr3.js` (order matters; cr3 = wiring + `go("jobs")` boot, must be last). Build: `bash build.sh` → `control-room-mock-v1.html`.
  - cr-rules.js = 130 real rule rows generated from `frontend/src/app/commissioner/rules.legacy.wire.testdata.json` + 5 added (lineup lock, format+, schedule+, lottery = PLANNED; bids.binding = FIXED R3).
  - cr1 = data (TEAMS, JOBS, SITS+SYN, PICKS, LEDGER), shell, card, editor, `record()` (refuses unbuilt via UNBUILT regex and non-permitted via mayRun, requires reasons, "MOCK · would record"), search (ranked word match + synonyms).
  - cr2 = views jobs/sit/rules(fitRules via ResizeObserver)/season/fix/picks(fitPicks)/people/league/server, desk() helper with extra mini-lists.
  - cr4 = permissions: CMDLIST (87 cmds), LOCKED (5 never), CONLY (11 delegable C-only), DOTONLY (trade.review/recuse), PLAIN names, PSETS (6 sets), `mayRun(c)` = the ONE permission check (C all; CC all but LOCKED, CONLY only if delegated via set to SELF="Ironclads"; dry_run always ok), staged edits PDRAFT + stageBar, whoIs/whoCan.
- Lost scratchpad → take `docs/ui/mocks/control-room-mock-v1.html` from the worktree (single file).
- Shots: `$SCRATCH/shot.sh <abs html> <abs png> 1920,1080 "<js>"` — JS must not contain `#` or `|` (sed delimiter); title shows `ERR …` on exceptions; set `document.title='R '+…` to read test results. Also check 2550,1352. `rm -rf hcr _t.html` after.
- **Class-collision trap (4 times now: .r .mine .wire .ghost):** base.css owns `.r .mine .wire .mv .tl .pin .grab .sel .ghost` etc. Prefix every new class `cr-`.
- Publish: scp to Beelink `~/Downloads/` + worktree `docs/ui/mocks/`, commit with trailers, give `file:///home/chris/Downloads/control-room-mock-v1.html`.
- Sol 6.1 runs: `ssh -f … 'SOL_WT=uiux SOL_BRIEF=<f>.md nohup ~/fleet/bin/launch-sol-reorg.sh >/dev/null 2>&1 &'`, brief in `~/fleet/briefs/purge/`, done = `~/fleet/runs/sol-uiux/completion.txt`, report `stdout.log`. Terra: `TERRA_WT=uiux TERRA_BRIEF=… TERRA_THINKING=high … launch-terra-reorg.sh`, run dir `~/fleet/runs/terra-uiux/`. Both finished in ~2 min each. Always arm a done-watcher (background until-loop) in the same step.
- Never grep the session JSONL with `.\{400\}` regex (hung). Python substring search is fine.

## 3. Status
| Item | State |
|---|---|
| HQ v10, War Room v6, Pulse v14, Home v5 | APPROVED |
| Control Room tree `docs/ui/control-room/Control_Room_Tree_v1.md` | v1 + Sol triage (77899aac) |
| Control Room mock v1 | good place; de-slopped (3c66f76c) |
| Permission hierarchy | ruled P1/P2, in mock |
| Build plan | NOT started — waits for Christopher |

## 4. Artifacts (worktree)
- `docs/ui/mocks/control-room-mock-v1.html` (current) + HQ/WarRoom/Pulse/Home mocks.
- `docs/ui/control-room/Control_Room_Tree_v1.md` — sources, motivations M1–M9, branches, every leaf, owner-overlap table §4, gaps §5, Sol triage §6.
- `docs/ui/research/commish-permission-hierarchy-terra-2026-10-30.md` (Terra), `docs/ui/research/control-room-critique-sol-2026-10-30.md` (Sol devil's advocate, grade D, triaged).
- `docs/ui/UI_Direction_v2_2026-10-30.md` — all rulings (read before any design).
- Commits tonight: 894e30c7 R3 · b7aa98cf tree · 77899aac triage · 4cf7cd23 rulings · 1a5e2490 mock v1 · 27876edd permissions · 7aca01a1 critique fixes · bd0a23da P1/P2 · 3c66f76c de-slop.

## 4b. Data catalog (84bee24f, worktree docs/ui/data-catalog/)
- `claude-catalog-2026-10-30.md` — 330 rows (SH/HQ/WR/LP/HM/CR) + SOURCE MAP D01–D78 (domain → EXISTS 27 / PARTIAL 31 / NEW 27, evidence = exported funcs on origin/main per package, rows).
- `terra-{HQ,WR,LP,HM,CR}.md` — 1,890 atomic rows (stub checklist). Terra NEW counts inflated (opened few files) — trust Claude source map for built/not.
- `COMPARE.md` — coverage agrees; findings.
- Key existing packages: engine/race (seeding, H2H, bubble), powerrankings (WinProbability, ExpectedWins, Blend), app/season (AllPlay, brackets, schedule grid), ingestion/livegame (scoreboard, possession, scoring plays), app/now (live spines/box), app/control (alerts: lineup/bye/IR, CompareAlerts), mflref/envelope (move envelopes), app/offers (TargetBid* draft/plan/posted, pool), store/personal (private offers, LastSeen), app/market (PickValue, DynastyCells), playerrecord (career), governance (roles, votes, elections, LastActivity, appeal), leagueclock (Calendar), transactions/deadcap, store/accounts TOTP, serverstatus, app/platform backup.

## 5. Current bug
None known.

## 6. Refuted / don't redo
- Earlier list (lightened fills, text on hatching, sparklines in tables, jet black on Home, one-name ticker, stacking alert band) still stands.
- Control Room: "trade reviewer" permission set (removed — P2); appeal ruling in a DOT set (appeals route to commish/deputy in code); 900 ms hold (ruled ~600 ms); raw command names in the permission list (use PLAIN words, command on hover); fake "Recorded" toasts; generic text rule editor as the build target (build must reuse built `RuleChangeForm` + backend preview).
- Rejected Sol points: drop tabs or rail (all approved mocks have both); "honesty F" graded a mock as an app (substance fixed).

## 7. Decisions (10-30)
- R1 Race→Power offseason YES · R2 window labels league-wide YES · R4 center cards approved exception.
- **R3 NO one-of bids: a bid is a bid**; open bids ≤ available cap; ends only when legally outbid. Engine bug logged: `internal/commissioner/market/bid_execute.go:136` withdraw() lets owners pull standing bids. Open-bids-vs-cap aggregation = lead to trace (`nativecheck/bids.go:186` "caller judges cap").
- Control Room = jobs + setup; co-commish near-parity.
- **Picks fully fluid** for commish (grant/forfeit/move/swap/condition/void); mini-game rewards can be picks.
- **Breadth: more is better** — open lineup lock, all-play/total-points, schedule patterns, rookie lottery. Function first; look secondary for Control Room.
- **Permission sets** (Terra model): commish-built named sets of commands, granted to people (scope league/teams, expiry), additive, never a promotion, private grants, staged+versioned edits.
- **P1 YES:** C-only actions delegable via sets (typed reason). Never delegable (MY call, told him, no objection yet): role grant/revoke, deputy order, succession settings.
- **P2 NO:** trade votes only from DOT members; sets can't carry trade.review/recuse.
- TOTP reset by commish ruled 10-30 (accounts README predates it; not a conflict).

## 8. Ledger
All committed on local branch through 84bee24f (catalogs) + this resume; nothing pushed; pushing not discussed.

## 8b. Open items to address (from catalogs)
1. **Bid visibility conflict** → needs Christopher ruling: WR shows outbidder name+amount; Home says runner-up sealed until window; CR says bids private even to commish; R2 "no early peeking".
2. **Role variants on owner screens unspecified** (what commish/co-commish/DOT see differently on HQ/WR/LP/HM).
3. Mock inspectors half-bind (build must drive every drawer from the selected entity); WR partner selector must swap shelf/direction; plan checks per plan.
4. Simulation outputs with no algorithm yet: playoff odds/magic/bye%, bid win odds/est. win price/rivals, power components, age curves, team needs, posture lens, Best 21.

## 9. Next actions
1. On "we are back": read this file + `docs/ui/data-catalog/COMPARE.md` + the SOURCE MAP section of `claude-catalog-2026-10-30.md`.
2. Address open items: put the bid-visibility question to Christopher (one AskUserQuestion, options: sealed until window for all / outbid owner sees top amount only, not name / full: name+amount) and the role-variant question (propose: owner screens identical for all roles; commish extras live only in Control Room + deep links; confirm).
3. Build the DATA FLOW MAP doc `docs/ui/data-catalog/DATA_FLOW_MAP.md`: for each D01–D78 domain → owner (package/store), producers → consumers (screens/rows), shared vs per-user-private vs commish-only vs public-ledger ISOLATION class, what feeds what (e.g. contracts+deadcap → cap runway → HQ/WR/Home/plans; valuation → Best21 → power → race/odds), refresh/trigger, and build order (dependency layers). Verify every "EXISTS" claim by opening the package (git show origin/main), not by name. Consider Terra cross-check pass after draft (slice per layer).
4. Then (later): Control Room build plan: fold tree §5 gaps + Sol critique carry-overs (reuse RuleChangeForm/previews/safeguards; shared owner cards + deep links with return route; guided flows for mistaken-cut repair, RFA match dispute, expansion) + rulings (P1/P2, R3 bug, picks, breadth) into a plan doc; propose slices; Christopher confirms before anything is dispatched.

## 10. Environment
Nothing running. Terra chain finished (CHAIN-DONE); Sol idle. Filing check exit 0. Scratchpad 755K.

## 11. Honest status
Mocks only; data illustrative except the 130 rule rows and command list (real, from origin/main). Not built: permission-sets engine (Terra est. 12–21 days), pick commands beyond move, TOTP reset, support/operator roles, 7 job producers, breadth options, lineup lock.
