# RESUME — overnight-gaps-1124 (Tom head brain), written ~7:50 PM CT 11-24
First action after compaction: `~/fleet/bin/run-watch.sh status ~/fleet/runs/overnight-gaps-1124`, then re-arm
`run-watch.sh wait ~/fleet/runs/overnight-gaps-1124 50` (background) if `pgrep -f "run-watch.sh wait"` is empty.
Event log: REPORT.md (append `- <time> CT <job>: <result>`). Plan/done lines: PLAN.md.

## Authority (Christopher 11-24 ~4:20 PM): merge ANY PR that is CI green AND has a caught planted break (squash: gh pr merge N --squash).
## Routing (Christopher ~7 PM): builds -> Sol 6.2 (`launch-pulse-lane.sh sol62 <lane> <brief> medium`); Luna high/xhigh = research/specs; no Gemini unless GPT low. AGY Claude quota out ~68 h.

## Open now
- #515 MERGED ~7:55 PM.
- #516 PR-H Wails removal (Sol 6.2 + Tom gate fixes, 2 planted breaks). Merge when green (watch ci516).
- Lane `hostbench` (Sol 6.2): bench to hosted storage shape + guard test + re-profile. On done: review diff (test files only), revert any root HANDOFF.md edit, plant own break, commit, push, PR, watch, merge.
## Recurring fixes
- Line-budget baseline conflicts on every PR: merge origin/main into branch (force-push is BLOCKED), `git checkout --theirs docs/architecture/line-budget-baseline.txt && git add` it, then `scripts/check-line-budget.sh --write`, add, commit. Growth needs docs/architecture/line-budget-reasons/<branch-with-dashes>.md.
- Fresh worktree push fails until `cd frontend && pnpm install --frozen-lockfile && pnpm build`.
- Lanes write into tracked root HANDOFF.md: `git checkout -- HANDOFF.md` before commit; move RESULT.md out.
- Main checkout is on main: create worktrees with `git -C ~/work/TheWarRoom worktree add ...` from ~ (hook blocks cwd on main).
- Env: PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.11 GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1
## Next candidates (in order)
1. PR-I (desktop step 9): Makefile Wails targets, check-bindings-fresh.sh, check-wails-version.sh, wails.json refs in release/version scripts; Makefile LDFLAGS -> internal/serverversion; `make build` stays the server build. Sol 6.2 brief, after #516 merges.
2. Act on hostbench's verdict (real hosted p95 cause).
3. Clean 32-team load rerun with merges held (load/REPORT.md has run 1: 0x502, p95 273 ms, RSS 523 MB).
4. Worktree cleanup: wt-kept.txt (49 kept; most likely dirty only from untracked dist/node_modules) — remove merged ones whose `git status --porcelain --untracked-files=no` is empty.
5. Morning: finish REPORT.md (Needs Christopher on top), tidy-check.sh, tracker line, stop the watcher.
## Needs Christopher (so far)
- Canary (T708) token: mint a connection token for native.canary on the Lab with his authenticator; paste into Hermes ~/canary/canary.token (mode 600); `systemctl --user start twr-canary.service`. Agent guessed server twr-lab-260 (override TWR_CANARY_SERVER).
- PR-F ruling (solo-owner desktop leaves; internal/app/mflref kept tonight).
- Approve drafts on main: docs/specs/{draft-room-undo,history-in-out,football-evidence-split}-1124.md (questions inside each).
- Owner chat: retired 11-04, doesn't exist; revive? (docs/research/owner-chat-check-1124.md)
- MFL import has no screen since PR-H (hosted import = server console command). Want one?
- Clock-expiry auto-pick not built (Legacy rule = skip, correct for Dec 1).
- Idea: make the line-budget baseline regenerate on merge so parallel PRs stop colliding.
- Open: auto-draft skips 4-5 picks per round with only franchise.01 seated (expected 1) — check.

## Merged tonight
#508, #510 (was #509), #511, #512 (PR-E), #513 (PR-G), #514 (results reads ledger once), #515 (flight test).
## In-flight detail (alive checks)
- ci516: `~/fleet/runs/overnight-gaps-1124/ci-done.sh 516` (0 done, 1 pending). Then merge if green (PR body names 2 planted breaks).
- hostbench lane: alive `pgrep -f pulse-hostbench`; done `~/fleet/runs/pulse-hostbench/completion.txt`; report `~/fleet/runs/pulse-hostbench/RESULT.md`; transcript `~/fleet/runs/pulse-hostbench/sessions/*.jsonl`; worktree ~/work/TheWarRoom/.worktrees/pulse-hostbench.
## Refuted — do not retest
- "Auto-draft is broken for all-computer leagues": no — 1,133 picks ran on native.load1124b (~3 s/pick); driver wait was killed.
- "Clock expiry auto-picks for seated owners": not built; Legacy rule = SkipPick (Census CM-040); setup refuses AutoPick on purpose.
- "Rescoring the week causes the 360 ms burst": no — carrying unaffected scores saved only 4% (lane scorecarry, not shipped; worktree pulse-scorecarry kept for reference).
- "The 360 ms burst is the hosted p95 cause": not shown — it is a bench artifact (single-file storage rebuilds the 55k directory); hosted uses OpenWithReference. hostbench measures the real shape.
- "Evidence split needed for memory": no — RSS 523 MB flat vs v59 535 MB with #496 shared decoded copies.
- "Owner chat exists": no — retired 11-04 (endpoint-registry M-121).
- "#509 force-push": BLOCKED by guard; use new branch or merge main in.
