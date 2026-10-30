# TheWarRoom UI/UX groundwork: RESUME (2026-10-30 night, compact-safe #6)

## 1. What we are doing
HTML mocks, one screen at a time; Christopher approves from screenshots/clicks.
- DONE: Franchise HQ v10, War Room v6, League Pulse v14, Home v5 (slate).
- **Control Room mock v1: in a good place** (Christopher 10-30: "really good start", "we are in a good place"). Last pass = de-slop + hover/focus states, commit 3c66f76c.
- NEXT: Christopher picks: (a) Control Room build plan (tree + Sol critique + rulings → plan), or (b) next screen. ASK; don't infer.
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
All committed on local branch; nothing pushed; pushing not discussed.

## 9. Next actions
1. On "we are back": read this file + `UI_Direction_v2`. Ask Christopher: Control Room build plan, or next screen?
2. If build plan: fold tree §5 gaps + Sol critique carry-overs (reuse RuleChangeForm/previews/safeguards; shared owner cards + deep links with return route; guided flows for mistaken-cut repair, RFA match dispute, expansion) + rulings (P1/P2, R3 bug, picks, breadth) into a plan doc; propose slices; Christopher confirms before anything is dispatched.

## 10. Environment
Nothing running. Sol + Terra idle (completion files present). Filing check exit 0. Scratchpad 670K.

## 11. Honest status
Mocks only; data illustrative except the 130 rule rows and command list (real, from origin/main). Not built: permission-sets engine (Terra est. 12–21 days), pick commands beyond move, TOTP reset, support/operator roles, 7 job producers, breadth options, lineup lock.
