# TheWarRoom ring 4 — RESUME (ClaudeBox, 2026-10-19 08:00 CT)

## 1. What we are doing
Goal (/goal): finish ring 4 and close every loose end. Ping Christopher only at the PR question.
Beelink chris@192.168.1.191, worktree ~/work/TheWarRoom/.worktrees/ring-4-integ2, branch session/ring-4-integ2.
Handoff: CT105 /root/TheWarRoom-ring4-HANDOFF.md (constraints §6 and commands §7 still apply — read §6 and §7).
Env for every Beelink command: export the §7 env (PATH with /usr/local/go/bin etc.) or the commit hooks fail.

## 2. In flight
- PUSH: `git push -u origin session/ring-4-integ2` started 07:27 CT, detached; pre-push runs make verify (~45 min).
  Log ~/fleet/runs/warroom-ring4-2026-10-13/integ2/push1.log; done when `grep PUSH_EXIT` there. Race run 32 min in at 08:00.
  ClaudeBox waiter task bl37e7w0w (it dies at compaction; just re-check the log).
  **Do not edit the worktree until PUSH_EXIT is present** — verify reads the working files (vitest runs after go test).
- Claude-OS runs r4n (build of a6c09ed3), walk league "Ring 4 walk 10-18".

## 3. Status
| Item | State |
|---|---|
| verify24 | VERIFY_EXIT=0 |
| frontend half (SeedTray edit) | FE_EXIT=0, 1347/1347 (integ2/fe24b.log) |
| commits | bc3a8637 Starting assets preview…; a6c09ed3 docs: ring 4 handoff |
| push | running (above) |
| walk r4n | done except bracket acts (extra scope, skipped) |

Walk PASS: starting assets recorded (reason typing keeps preview), contract correction (Roberts J.J. $1.25M), rule
"Shared date boundary = Start of day" (now), rollover dry run "Season check passed. Nothing was recorded." (rollover NOT
recorded), draft room honest gate, schedule generate+publish+published edit (Bengals v Titans wk13), waiver/bid empty states.
Shots: ClaudeBox scratchpad r4n-*.s.png; Beelink run dir shots/r4n-*.png.

## 4. Findings to FIX (patch scripts ready)
Scripts: integ2/p1_tests.py (tests first) and integ2/p2_fix.py (fixes), copies in ClaudeBox scratchpad. Run from worktree root:
`python3 ~/fleet/runs/warroom-ring4-2026-10-13/integ2/p1_tests.py` → run tests, see them FAIL → `p2_fix.py` → pass.
- A act tray drops preview while typing → useChangePreview.ts keeps last preview, wait "Checking the change again…".
- B corrections/overrides/score/lineup-correct say "Reason (optional)" but Go refuses blank (validateOverridePlacement etc.)
  → reasonByPlacement() in command.go, used by safeguardOf (preview_command.go) and CommandSafeguards (safeguards.go).
  DO NOT change gravity: it is stored in the ledger and re-checked at replay (ledger.go:44).
  Needs wire regen: TWR_WRITE_WIRE=1 TestCommissionerCommandSafeguardsWire (commands.wire.testdata.json) + §7 wire run.
- I stat queue fails on every real 2026 file: nflverse team rows with empty player_id (team penalties, one team safety)
  → ReadWeeks skips empty player_id; malformed non-empty still refused. Put in PR as call 14.
- K new tray shows previous receipt (outcome is global, form "rule"/command name) → ChangeBar forgets outcome on mount.
- L preview rows sort as text (week 1,10,11,2) → labelOrder natural sort in ChangedPreviewFields (preview_operations.go).
- F "· League version N" in rulesReport.ts:24 and peopleReport.ts:16 → removed. (No new test written for F yet; check
  existing report tests still pass; consider adding a not.toContain assertion.)
After fixes: frontend half + `go test` for ./internal/commissioner ./internal/fantasyscore/source and the root acts tests,
then commit (git commit -F, trailer), build+deploy r4o BEFORE pushing, push (runs full verify), re-walk A/B/I/K/L on r4o.

## 5. Follow-ups for PR notes (not fixing)
D rollover preview lists ~2,299 rows one by one (like seed before summary); E raw window ids ("open fa","resign","ufa");
G division alignment raw "Franchise.01: 00"; H command bar doesn't index rules; J rule value box prefilled raw
"division_weighted"; tray header/Close scrolls away with content; schedule edit "Affects" generic 32 franchises.

## 6. Refuted / do not retest
- C "confirm still live after receipt" is NOT a bug: useRequestKey keeps one key → re-fire replays idempotently.
- First commit failure was env (no go on PATH), not code.
- Clicks not registering = spice tablet; run ./unstick-pointer.sh.

## 7. Next actions
1. `grep PUSH_EXIT integ2/push1.log`; if not 0 read FAIL and fix.
2. Apply p1_tests.py; run the targeted tests; confirm the new ones fail.
3. Apply p2_fix.py; regen wires; frontend half + targeted go tests; commit.
4. Build + deploy r4o; push detached (push2.log); re-walk the fixed items on r4o (stat queue opens!).
5. Add c11 section + walk findings + call 14 + follow-ups to integ2/PR-notes.md; open PR
   (`~/.local/bin/gh pr create --base main --head session/ring-4-integ2`), body ends with 🤖 line.
6. PushNotification + ask Christopher ONE question (accept calls as written / name changes).
7. On his go: squash-merge, `cc task check T434`, remove ring-4 worktrees via git worktree remove, go clean -cache,
   log in ~/MOVED.md, ~/archive/MANIFEST.md, CT105 backbone/context.md.

## 8. Honest status
Walk found 6 fixable bugs; none fixed yet. PR question ETA ~10:15 CT (fix ~30 min + push verify 45 min + re-walk).
