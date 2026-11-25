# RESUME — carrying out Astra's rehearsal plan (handoff written 2026-11-25 by Tom)

**Read this whole file before doing anything.** Then open the two plan files in §2. This file tells you what is done,
what is running, what to do next, and the rules Christopher set. It replaces `RESUME-edges-1125.md` (11-25 morning)
for current state; that file is still worth reading for the GATE 2 and chat history.

## 0. The assignment, in one paragraph

Christopher handed Tom two plans written by Astra 6.2 and said: "begin these plans and because your tokens are running out
you MUST lean heavily into gpt, protect your context and keep gpt busy with well scoped work." **Your job is to keep
carrying out those two plans, session by session, against the calendar in §3**, as head brain:
- write tight briefs (under 40 lines, one outcome, done lines with checks);
- launch GPT lanes (Sol medium for multi-file work, Luna high only for one-file jobs with a known outcome);
- review every diff, plant one deliberate break per PR and watch its test fail;
- open PRs, and merge when green (see §6 for merge authority).

Do not do the typing yourself; your own tool calls are for review, briefs, Lab operations that need secrets, and
decisions. His standing words, 11-25: "lean heavily into gpt for the heavy lifting from here on out to protect your
context."

## 1. Machines and access

- Repo `~/work/TheWarRoom`. **Never work in the main checkout; it sits on `main`, and a hook blocks git writes there.** Use
  `.worktrees/<name>` on `session/<name>` branches. Toolchain line: project `CLAUDE.md`, Session start step 3.
- Lab = Proxmox CT 260. `ssh -F /home/chris/work/TheWarRoom/.worktrees/pulse-a010s/.scratch/ssh-lab.conf lab`, then
  `pct exec 260 -- <cmd>`. Log: `/var/lib/twr/server.log`. Lab kit (holds **secrets**, never print it):
  `TWR_LAB_KIT=~/scratch/commish-1123` (it has KEEP.md). The kit scripts are `scripts/lab/*`.
- Lab automatic releases: `~/fleet/runs/twr-lab-autorelease/`. A file named `hold` there pauses them. **The pause was
  lifted 11-25 ~12:10 UTC** (logged in `hold-log.txt`), so every merge to main now releases to the Lab and restarts it.
  **Put `hold` back before any Lab draft opens** (Sun 11-29 practice, Tue 12-1 real) and remove it after.
- Season server = Acer, 192.168.1.162. **Only Saturday night, only with Christopher's OK, with the exact command handed to
  him.** Never a test build.
- Lanes: `BASE=<branch> ~/fleet/bin/launch-pulse-lane.sh <sol|luna> <lane> <brief.md> <medium|high> [nowt]`, launched
  detached (`nohup setsid … &`). The run dir is `~/fleet/runs/pulse-<lane>/`: `completion.txt` = done, `stdout.log` tail =
  report, `RESULT.md` = evidence, `sessions/*.jsonl` mtime = alive. The worktree is `.worktrees/pulse-<lane>`.
- Watcher: `~/fleet/bin/run-watch.sh add|wait|status ~/fleet/runs/astra-exec-1125`. Register every lane at launch. Use
  `test -n "$(find ~/fleet/runs/pulse-<lane>/sessions -name '*.jsonl' -mmin -25)"` as the alive check (11-25 lesson: a
  `pgrep -f` alive check matched nothing real, and a lane that never started looked "running" for an hour). Keep a
  background `wait … 50` armed.
- Codex on the Beelink is now set up: `~/fleet/docs/codex-access.md`; Claude-OS helper `~/fleet/bin/claudeos`.

## 2. The two plans (the source of truth for what to build)

1. **Rehearsal plan (8 sessions, Nov 25 → Dec 8):** `~/fleet/runs/astra-rehearsal-plan-1125/rehearsal-plan.md`.
   Items 1A–8B. Its "Builder contract" and "Calendar protection" sections are binding.
2. **Whole-PWA integration brief (W0–W5):** `~/fleet/briefs/twr-whole-pwa-integration-1125.md`. It covers the 287 desktop
   v3 entries and the 56 mobile jobs. Observation first; one existing connection per PR. General phone changes wait for
   the post-Dec 6 study.
3. Background: the source brief `~/fleet/briefs/astra-ui-edges-1125.md` (repo copy `docs/reset/UI-EDGES-1125.md`), and
   the rulings `docs/reset/RULINGS-1124.md` and `docs/reset/RULINGS-1125.md`.

## 3. Calendar (Central time)

| When | What | Rule |
|---|---|---|
| Thu 11-26 – Sat 11-28 | Finish the draft-expiry setting; get every fix onto the Lab; seat proof (1B) | Merges OK when green |
| **Sat 11-28 night** | Season swap (2A), with Christopher's OK and the exact command. The new rollback (#526) restores data | Only with his OK |
| **Sun 11-29** | Practice drafts on the Lab, default + twin copies (3A/3B). Christopher drafts by hand | `hold` on; no restarts |
| **Mon 11-30 – Tue 12-1** | **Real drafts on the Lab.** No risky merges, no releases, no capacity changes | `hold` on; 4A is the only exception |
| Wed 12-2 – Thu 12-3 | Forecast accuracy (5A), live surface check (5B). Week 13 starts Thu 12-3 | |
| Sun 12-6 | Watch the real Sunday on phones (7A); replay overnight (7B) | No Season changes |
| Tue 12-8 | First Tuesday report card (8B) | |

## 4. Done so far (11-25), with evidence

| Item | State | Evidence |
|---|---|---|
| #523 Astra brief + 11-25 rulings (chat, draft expiry, lost picks) | In merge chain | PR #523 |
| #524 10/12/16-team shape leagues (`scripts/lab/shapes.sh`) | **MERGED** | |
| #525 `league.chatLink` + "League chat" control (phone League, League Pulse) | **MERGED** | planted breaks: http://, dropped noopener |
| #526 Season rollback restores the data backup | In merge chain | 8 offline tests; planted breaks caught |
| #527 Note privacy + edit test | In merge chain | planted break: any-seat match |
| #528 Lost computer picks during write pauses | In merge chain | `TestAutoPickRetainsTurnAcrossDurabilityPause`; planted break caught |
| GATE 2: native.rehtest-10/12/16 drafted by computer, 43 rounds | Done, **with findings** | `~/fleet/runs/pulse-gate2verify/RESULT.md` |
| 1A static 56-row phone checklist | Done (47 expected-pass, 7 blocked, 2 excluded) | `~/fleet/runs/pulse-astra1a/RESULT.md` |
| 1C Tuesday report readiness | Done: **wrong leagues** (see §7) | `pulse-astra1c/`, `pulse-astra1c2/RESULT.md` |
| 2A Saturday swap work order | Done; its STOP (no data restore) closed by #526 | `~/fleet/runs/pulse-astra2a/RESULT.md` |
| W3 desktop inert controls | Done: 146 controls; 18 inert; 0 with a ready command | `~/fleet/runs/pulse-astraw3b/RESULT.md` |
| Lost-pick diagnosis | Done | `~/fleet/runs/pulse-lostpicks/RESULT.md` |

**Merge chain** (background): `~/fleet/runs/astra-exec-1125/merge-chain.sh`, log `merge-chain.log`. It works through
526 → 523 → 527 → 528 in order. For each PR it re-merges main (`remerge.sh`, which resolves only line-budget-baseline
conflicts), waits for CI and squash-merges if all checks pass. **First thing to do: `cat ~/fleet/runs/astra-exec-1125/merge-chain.log`.**
Any line saying "NOT GREEN" or "FAILED" is yours to fix. Then re-run the chain with the remaining `pr:worktree` pairs.

## 5. Running when this was written

| Lane | What | On done |
|---|---|---|
| `astra8b` (Sol) | 8A: can an owner tell what changed after a stat correction? Isolated local run; no code in tree (`capture.patch`) | Read RESULT.md. If data exists but no screen shows it, brief the follow-on display PR **for after Dec 6** |
| `draftexpirea` (Sol, base `session/pulse-draftpause`) | **Server half** of Christopher's draft-expiry ruling (§6.2): `autopick`, `skip` keeping an owed pick, `wait`. Told not to run git; cap 600 lines | Review the diff and plant a break (e.g. make the owed pick lose its original slot). Commit, PR, merge when green **before Sun 11-29**. Then launch the **frontend half** (§8 step 4) |

The design for the expiry setting is in `~/fleet/runs/pulse-draftexpire/RESULT.md`. A key point from it: `DueDrafts`
must not auto-close a draft while owed picks remain.

## 6. Christopher's decisions — do not reopen

1. Merge authority (11-25): "After these lanes finish, review them, build the PR's and merge them when green." That
   covers #523–#528 and the `draftexpirea` PR. **Anything new after that needs his merge OK again**, and nothing merges
   Mon 11-30 – Tue 12-1.
2. Draft clock expiry for a human seat (11-25), verbatim: "Lets build a commissioner setting that does the above, default is
   that pick is skipped but can be picked at any time." The setting `draft.onExpire` gets three values: `autopick` (the
   best-ranked available player), `skip` (**the default**: the turn moves on, the seat keeps the pick and can make it
   any time while the draft is open, in its original slot), and a new `wait` (no expiry). It changes through the one
   rule-change flow; presets only suggest. Recorded in `docs/reset/RULINGS-1125.md`.
3. Lost-pick fix: merge before Sun 11-29 so the practice drafts test it.
4. Chat: link out for the rehearsal (`league.chatLink`, done). Native chat later is one store with rooms. Alerts are
   mentions-only. The Ask button stays canned; a real AI helper comes later.
5. Settled in Astra's plan: rehearsal first; default and twin leagues are judged separately; reuse current software;
   no public launch; Legacy is never a test target; no standing deploy permission.

## 7. Open findings that need action (ordered)

1. **Tuesday report covers the wrong leagues.** Hermes `~/fleet/twr-reportcard/run.sh` checks only
   `native.tomtest-default/twin`. The rehearsal leagues are `native.rehearsal-default/twin` (real) and
   `native.rehtest-default/twin` (practice copies); confirm their exact ids. None has a read token. To fix: mint a
   read-only helper token per league (Lab kit, `labcommish.py helper`, scopes `native.directory.read results.read
   standings.read nfl.games.read lineup.read`), put each in Hermes as `token-<league>`, and change `run.sh`'s league list
   (exact line in `pulse-astra1c2/RESULT.md`). Due before Tue 12-8.
2. **Writes pause after every Lab restart** until the off-site backup is verified ("changes are paused because the
   off-site backup is not current"; `internal/offsite/manager.go` CheckWrite; "Restart starts closed"). This also
   blocks **sign-ins**. #528 stops computer picks being lost. Human turns still lose clock time, so during drafts: no
   restarts, `hold` on. If a restart does hit, the commissioner can use `draft.timer` (pause / resume / set remaining).
3. **The shape leagues' franchise.01 is commissioner-held** (`-take … -no-login`) and cannot be released: "The last
   commissioner cannot be removed; choose a successor first". `shapes.sh release` therefore fails at that step. Fix the
   script (keep the seat, or seat a computer successor). Small Luna job.
4. Canary (1C): BLOCKED until Christopher issues the T708 write token.
5. A lane committed and pushed by itself (draftpause). Lanes must not run git: say so in every brief.

## 8. Next actions, in order

1. Read `merge-chain.log`. Fix anything that isn't merged, and re-run `merge-chain.sh` for the rest. Confirm the Lab
   picked up the new main: `~/fleet/runs/twr-lab-autorelease/state`, and the Lab version on the health page.
2. `run-watch.sh status ~/fleet/runs/astra-exec-1125`; re-arm `wait`.
3. On `draftexpirea` done: review it, plant a break, commit with `git commit -F`, PR and merge when green.
4. Launch the frontend half: brief `sol-draftexpire-b` (base main after merge). The draft room shows "You missed round R
   pick P. Pick any time." on desktop and phone (shared component), and the late pick uses the existing `draft.pick`.
   Include vitest; the line rules (`controls.test.ts`: no raw `<a>`/`<button>` outside Act/HQAct, lines ≤110 chars); and
   a full `pnpm vitest run`, not only the touched files (an 11-25 lesson). Merging it needs Christopher's OK (§6.1).
5. Fix the `shapes.sh release` franchise.01 step (§7.3).
6. 1B seat proof: plan item 1B on a disposable copy. **`seat-link.sh` restarts the shared Lab**: do it in a quiet window,
   never with a draft open.
7. 1A on real phones: use `pulse-astra1a/RESULT.md` as the checklist. Phones: `~/fleet/bin/phone-chrome.py <s8|iphone> …`.
   Disposable leagues: `native.rehtest-10/12/16` (computer-drafted, 43 rounds).
8. §7.1 Tuesday report leagues and tokens.
9. Saturday: hand Christopher the 2A work order (`pulse-astra2a/RESULT.md`) with the exact `season-swap.sh` lines, and
   note the new `--rollback` data restore from #526.
10. Sunday and Tuesday: follow plan items 3A/3B/4A/4B exactly. Put `hold` back first.

## 9. Lessons from 11-25 — do not repeat

- **Never plant a break with `git checkout -- <file>` on a lane's uncommitted tree.** It wiped Sol's change, and it had
  to be rebuilt. Use `cp file /tmp/x.bak`, then sed, then `cp` it back.
- **`python3 -I -m unittest` can't find modules in the current folder.** Use the repo's `make test-deploy` target.
- Every PR edits `docs/architecture/line-budget-baseline.txt`, so each merge makes the next PR conflict. Use
  `~/fleet/runs/astra-exec-1125/remerge.sh <worktree>`. Docs growth also needs a reason file.
- A fresh worktree has no `frontend/node_modules`, and pre-push then fails with TS5108. Fix: `cd frontend && pnpm
  install --frozen-lockfile --prefer-offline`.
- Only one lint can run at a time ("parallel golangci-lint is running"). Retry the push after about 90 s.
- A hook blocks any command whose starting folder is the main checkout when it runs `git merge`/commit. `cd` into the
  worktree first, as a separate step.
- `pgrep -f <pattern>` matches your own shell when the pattern appears in your command line. Never kill with it.
- `twr-server create-league -teams` defaults to 32 and ignores the preset. Always pass `-teams N`.
- Lanes often stop short or test themselves hollow (Luna's first W3 pass; the hollow note-edit test). Always check the
  report against the done lines, and give a "Part 2" addendum rather than accepting a "NOT MET".
- A lane's `HANDOFF.md` edits must not be committed. Restore it from main.
