# RESUME: TheWarRoom ring 1, phase 7 (gate), paused for Sol's usage limit (2026-10-09 ~17:30 CDT)

> Written by `/compact-safe` on ClaudeBox (CT105). Christopher: "we will continue when Sol comes
> back online." On "we are back": check Sol is usable (§1), then start at §9 item 1.
> Supersedes RESUME 5eb7035 (ring 1 session close 10-07); its history is in git.

## 1. In flight
- **Nothing running.** Sol p7c2 died at 17:23:53 CDT: `Codex error: The usage limit has been
  reached` (stderr `~/fleet/runs/warroom-ring1-2026-10-07/p7c2/stderr.log`). It read files and ran
  a baseline `pnpm test`/build only; **worktree clean, no code written.** Reset time unknown.
- Check Sol is back before re-dispatch (a 1-line prompt costs little):
  `ssh chris@192.168.1.191 '~/.local/bin/pi -p --provider openai-codex --model gpt-6.1-sol --thinking medium "reply OK"'`
- Claude-OS: TheWarRoom running (pid 23165, binary `~/warroom-ring1/thewarroom` = f23fb60 build,
  launched 16:42 CDT 10-09 via `launch.sh`, log `run.log`; 10-07 log kept as `run.log.20261007`).
  No plan awaiting MFL. It does NOT contain 7b/7c1 code yet (deploy is a 7e step).

## 2. What we are doing
Ring 1 of the UI target (`docs/build-handoffs/UI_Target_Roadmap_2026-10.md` › Ring 1). Gate:
walkthroughs within click budget · an envelope reaches Landed live · spec revision 1 ·
Christopher's go before push/merge.
- Beelink `chris@192.168.1.191`, worktree `~/work/TheWarRoom/.worktrees/ring-0`, branch
  **`session/ring-1`**, HEAD **df22231**, NOT pushed. `origin/main` 3d01f33. Never `mv` the worktree.
- Working model (Christopher, 10-09): Claude = head brain, protects context; **Sol (gpt-6.1-sol)
  on Bee at MEDIUM** (do not raise: "it's been doing well on medium") codes; Claude reviews every
  line, fixes, proves fixes with tests that fail without them, runs `make verify`, commits.
  Laws: excellence is the standard; snappy UI is law; slop is banned.

## 3. Agents and harnesses
- Dispatch: `cd ~/fleet/runs/warroom-ring1-2026-10-07 && (setsid nohup ./dispatch.sh <phase> <brief> > <phase>.dispatch.log 2>&1 < /dev/null &)`
  Then wait in background: poll `<run>/<phase>/sentinel` (alive = mtime of `<phase>/sessions/*.jsonl`).
  Output `<phase>/REPORT.md`. Re-dispatch after a stop = new phase dir name (e.g. p7c2b).
- Briefs `~/fleet/briefs/warroom-ring1-*.md`; common rules `warroom-ring1-common.md`.
  Run log: `~/fleet/runs/warroom-ring1-2026-10-07/PLAN.md` (10-09 entries appended).
- Wails bindings: Claude regenerates (`wails generate module` in worktree), then
  `git checkout -- frontend/wailsjs/runtime` (wails flips file modes there: noise), commit App.d.ts/App.js only.
- Claude-OS GUI: `ssh chris@192.168.1.191 'ssh claudeos ...'` with `DISPLAY=:0 xdotool` + `scrot`;
  helpers this session in scratchpad (shot.sh, click.sh) are trivial to recreate. MCP browser fails.
  Christopher authorised Claude to click MFL itself (10-09) for the live test.

## 4. Phase 7 status (plan approved by Christopher 10-09)
| Step | State |
|---|---|
| 7a diagnose 10-07 "Not verified" | **DONE.** Verdict was correct; watcher keeps observing not_verified (store.go Awaiting includes it). MFL held original lineup. No TWR fix. |
| 7b DOT-review window + entry headroom | **DONE 008f07d.** Entry 319,536/325,000. |
| 7c1 IR target O=18 + roster.taxi both directions (Go) | **DONE 6662d0f + bindings df22231.** |
| 7c2 IR/taxi Acts in Inspector (frontend) | **NOT STARTED** (Sol limit). Brief ready: `~/fleet/briefs/warroom-ring1-p7c2-ir-taxi-ui.md` |
| 7d live Landed | **PASS 10-09.** lineup-ce4eba65 (+Kiner −Perine) Landed 21:44:23Z; revert lineup-4ff824e7 Landed 21:47:25Z. MFL back to original (Stroud QB, Perine RB). |
| 7e walkthroughs/click budget on Claude-OS | not started |
| 7f spec revision 1 | not started |
| 7g push + PR + merge on Christopher's go | not started |

## 5. Click budget (proposed by Claude, approved 10-09) — from Home, command bar = 0
Pulse Now 1 · My moves 1 · accept an offer 3 (tray → Plan accept → Open MFL) · IR 3
(alert/player → Draft → Open MFL; IR alert is Unavailable until MFL injuries ingested, so via
Inspector) · lineup one swap 5 (+2 per extra swap). Taxi: Draft → Open MFL after player is open.

## 6. Refuted — do not retest
- "Watcher gives up after Not verified" — FALSE. not_verified is in Awaiting; Match still lands.
- "Week mismatch on 10-07" — FALSE. 2026 season began Sep 10; week 5 = Oct 8-12.
- "Stale MFL tab saved original" — Christopher saved once. Cause of 10-07 non-save unproven;
  MFL saves only on the bottom button **"Submit Partial Lineup"** (ticks alone save nothing).
- A FreshFail feed never reaches a predicate (`observedAt` rejects it); "Not verified" from feeds
  comes from FreshStale-with-note (held copy after failed refresh).
- `taxiDestination` `case domain.RosterIR` is NOT redundant: `exhaustive` linter requires it.

## 7. Decisions (10-09)
- Sol stays medium. Ring 1 Acts: lineup.set, trade.accept built; roster.ir + roster.taxi (both
  directions) in phase 7; **trade.propose / reject / revoke deferred to ring 3** (MFL forms never
  captured; revoke is an irreversible GET) — record in spec revision 1 with reasons.
- IR/taxi registry status = "built, not yet Landed live" until a real Landed.
- Trade DOT window: 7 days from FIRST entry into DOTReview (MFL defaultTradeExpirationDays=7);
  after it, Not verified and frozen on unchanged evidence; inside it, stale-read Not verified
  re-enters review.
- Approved "lineup Partial→NotYetDone before lock" fix was WITHDRAWN (cosmetic; told Christopher).
- Project CLAUDE.md to be rewritten at session close (Beelink repo): Sol-on-Bee-medium workflow,
  expert panel on non-Claude models via pi, Christopher monitors progress in the file-tree mod
  (assumed; he did not correct it).

## 8. Ledger
Committed on session/ring-1 (not pushed): 008f07d (7b), 6662d0f (7c1), df22231 (bindings),
plus this resume commit. Uncommitted: none. `make verify` green at df22231's parent (6662d0f)
and bindings-only commit passed hooks.

## 9. Next actions, in order
1. Confirm Sol responds (§1). Then dispatch p7c2 (same brief; phase dir `p7c2b` if `p7c2` reuse is
   awkward — dispatch.sh clears the sentinel, so `p7c2` is fine).
2. Review 7c2 diff line by line (visible-text tests, no IDs on screen, lazy, entry ≤325,000),
   `make verify`, commit.
3. Deploy to Claude-OS: `make build VERSION=ring1-dev COMMIT=<sha>`, scp, `~/warroom-ring1/deploy.sh <old-sha>`
   (keeps old binary). scp can be cut off: check size + sha256.
4. 7e walkthroughs on Claude-OS with screenshots, counting clicks vs §5; check no loading flash
   on Home/HQ/Pulse Now/Trade desk after 7b lazy CommandBar/AppSettings; check command bar opens
   early. Fix list (copy/UI, brief Sol): app says "press Submit Lineup" but MFL button is "Submit
   Partial Lineup"; each hand-off opens a new MFL tab (stale-tab risk); superseded card shows raw
   correlation ID; superseded card track omits its Not verified step; Home endpoint index lists
   seasonal card "not wired"; hover tooltip lingers over next card after bench click.
5. If Christopher has a real IR-eligible or taxi move: live Landed for roster.ir / roster.taxi.
6. 7f spec revision 1 (Sol drafts from PLAN.md + this doc; Claude reviews; Christopher approves).
7. 7g push branch, PR, merge only after Christopher confirms live; then session close
   (/session-close) incl. project CLAUDE.md rewrite (§7).

## 10. Environment
- Toolchain over ssh: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH
  GOMEMLIMIT=1500MiB GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false"`;
  `timeout 600 nice -n 10 make verify VERSION=dev COMMIT=ring1` (~5 min).
- Edits: python all-or-nothing anchor scripts scp'd to the run dir (p7b_fix.py, p7b_fix2.py).
- Claude-OS DB read-only: `sqlite3 -readonly ~/.config/TheWarRoom/thewarroom.db` (tables
  move_envelopes, move_audit). `~/warroom-ring1/plan_state.sh` prints lineup audit.

## 11. Honest status
Gate: Landed PASS; click budget, spec rev 1, Christopher's go outstanding. IR/taxi have never
landed live. 7b lazy-load has no live flash check yet. Christopher has not yet used 5b/6b/7b/7c
live. Remaining effort ≈ 7c2 (one Sol run) + review + deploy + walkthroughs + spec rev.
