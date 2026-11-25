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
- Only Christopher and Claudebox may access Proxmox host `192.168.1.200` or its guests, including CT 257. Tom may recover the host only when Christopher declares it down.
- Lab = Proxmox CT 260. Christopher named Codex the operator for 1A, 1B and the league-ID check using the app and Beelink scripts. Before each window post the task, exact commands, scope, expected time and rollback. Wait for Christopher's go, which covers that window only. Use disposable test leagues and sign-ins. Verify league ID and role before every write. Record evidence in the run folder, add a RESUME line and push after each window. CT 257 remains restricted and its host location needs Christopher's confirmation.
- Approved Lab kit scripts are in `scripts/lab/`. `TWR_LAB_KIT=~/scratch/commish-1123` holds secrets. Keep them out of evidence.
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
| #523 Astra brief + 11-25 rulings (chat, draft expiry, lost picks) | **MERGED** | PR #523 |
| #524 10/12/16-team shape leagues (`scripts/lab/shapes.sh`) | **MERGED** | |
| #525 `league.chatLink` + "League chat" control (phone League, League Pulse) | **MERGED** | planted breaks: http://, dropped noopener |
| #526 Season rollback restores the data backup | **MERGED** | 8 offline tests; planted breaks caught |
| #527 Note privacy + edit test | Squash-merged at 09:54 Central, `ca9930d1e` | planted break: any-seat match |
| #528 Lost computer picks during write pauses | Open, conflicts with main | `TestAutoPickRetainsTurnAcrossDurabilityPause`; planted break caught |
| GATE 2: native.rehtest-10/12/16 drafted by computer, 43 rounds | Done, **with findings** | `~/fleet/runs/pulse-gate2verify/RESULT.md` |
| 1A static 56-row phone checklist | Done (47 expected-pass, 7 blocked, 2 excluded) | `~/fleet/runs/pulse-astra1a/RESULT.md` |
| 1C Tuesday report readiness | Done: **wrong leagues** (see §7) | `pulse-astra1c/`, `pulse-astra1c2/RESULT.md` |
| 2A Saturday swap work order | Done; its STOP (no data restore) closed by #526 | `~/fleet/runs/pulse-astra2a/RESULT.md` |
| W3 desktop inert controls | Done: 146 controls; 18 inert; 0 with a ready command | `~/fleet/runs/pulse-astraw3b/RESULT.md` |
| Lost-pick diagnosis | Done | `~/fleet/runs/pulse-lostpicks/RESULT.md` |

## 5. Merge queue — Christopher merges

Work through this list in order. Christopher performs each main merge after CI and review. Confirm the resulting Lab build from the release state and health evidence. The old chain scripts and logs are historical.

| Order | PR | What it unblocks | Dependency and current gate |
|---|---|---|---|
| 1 | [#527](https://github.com/secureprospective/TheWarRoom/pull/527) | Regression coverage for editing a private note and keeping it private across owners. | Squash-merged at 09:54 Central as `ca9930d1e`. Lab v96 verified healthy at 10:03 Central. |
| 2 | [#528](https://github.com/secureprospective/TheWarRoom/pull/528) | Reliable computer picks through write pauses, needed on the Lab for Sunday practice. | No code dependency on #527. #527 has landed. Conflicted. Carry `7410ed0ae` onto fetched `origin/main` in a replacement PR, preserve main's HANDOFF, regenerate the baseline, and rerun CI. |
| 3 | [#530](https://github.com/secureprospective/TheWarRoom/pull/530) | Server-side draft expiry and owed picks. Unblocks the shared missed-pick draft-room UI. | Depends on #528's lost-pick fix being on main. Conflicted. Carry only `7410ed0ae..3e4210a09` onto fetched `origin/main` in a replacement PR, regenerate the baseline, and rerun CI. |
| 4 | [#529](https://github.com/secureprospective/TheWarRoom/pull/529) | Publishes this handoff and the rehearsal run order on main. | No product dependency. Mergeable as of 08:07 Central. Docs follow the draft fixes so the handoff records their final PR numbers and status. |

Finished 11-25 afternoon:
- **8A (`astra8b`):** owner correction explanation is **BLOCKED**. After a correction the totals and the player's stats
  change, but no owner-readable read carries the reason or the previous official figure
  (`~/fleet/runs/pulse-astra8b/RESULT.md`, `capture.patch`). Per the plan this is a missing capability, not a
  display PR: write a short spec ("what changed, from what to what, why") for Christopher **after Dec 6**.
- **Draft-expiry server half: PR #530** (planted break caught). Its design: `~/fleet/runs/pulse-draftexpire/RESULT.md`.
  **Operator note:** under the default `skip`, a draft with an unmade owed pick **stays open** until the commissioner
  closes it, and closing finalizes the owed picks as not made. Tell Christopher before Tue 12-1.
- The frontend half (the "You missed round R pick P. Pick any time." line) is **not started**. See §8 step 4.

## 6. Christopher's decisions — do not reopen

1. Christopher merges to main. Follow §5 for the order and gates. The Nov 30–Dec 1 freeze still applies.
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

1. **Tuesday report readiness.** Hermes checks the two `native.tomtest-*` leagues, skips Dec 8, and can post green for
   AMBER or UNKNOWN checks. The offline proposal fixes all three and passed 11 fixtures. Before Tue 12-8, the authorized
   operator confirms the four real/practice IDs, uses `python3 labcommish.py reader "$LEAGUE"` for each, installs the scoped
   reads as `token-<league>` on Hermes, and applies and verifies the reviewed proposal. Exact procedure and patch are in
   `~/fleet/runs/astra-exec-1125/1C-REPORTCARD-1125.md`.
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

Christopher's 11-25 workflow update applies to this workstream. Finish existing work first. Start no branch or worktree while more than three of your own branches are open and unmerged. Use one worktree per task, branch from `origin/main`, and remove it after merge when clean. Do not merge main into task branches. Name the BUILD-PLAN step or RULINGS-1122 ruling in each brief. Record unrelated ideas without building them. Record blockers under Needs Christopher, then continue with the next plan item. Add one progress line per hour with PRs landed, open work, and blockers.

1. Prepare one PR at a time in the order in §5. Christopher merges each ready PR.
2. Confirm the automatic Lab release in `~/fleet/runs/twr-lab-autorelease/state` and verify health.
3. Confirm the draft-expiry server change is deployed before Sun 11-29.
4. Launch the frontend half: brief `sol-draftexpire-b` (base main after merge). The draft room shows "You missed round R
   pick P. Pick any time." on desktop and phone (shared component), and the late pick uses the existing `draft.pick`.
   Include vitest; the line rules (`controls.test.ts`: no raw `<a>`/`<button>` outside Act/HQAct, lines ≤110 chars); and
   a full `pnpm vitest run`, not only the touched files (an 11-25 lesson). Merging it needs Christopher's OK (§6.1).
5. Practice shape cleanup is in #531. The local verification passed and GitHub CI is running.
6. 1B seat proof: plan item 1B on a disposable copy. **`seat-link.sh` restarts the shared Lab**: do it in a quiet window,
   never with a draft open.
7. 1A on real phones: use `pulse-astra1a/RESULT.md` as the checklist. Phones: `~/fleet/bin/phone-chrome.py <s8|iphone> …`.
   Disposable leagues: `native.rehtest-10/12/16` (computer-drafted, 43 rounds).
8. §7.1 Tuesday report leagues and tokens.
9. Saturday: hand Christopher the 2A work order (`pulse-astra2a/RESULT.md`) with the exact `season-swap.sh` lines, and
   note the new `--rollback` data restore from #526.
10. Sunday and Tuesday: follow plan items 3A/3B/4A/4B exactly. Put `hold` back first.

## Needs Christopher

- **1B — seat proof.** Codex requests a separate quiet window. Before running any restart, show proof that every Lab draft is closed and `~/fleet/runs/twr-lab-autorelease/hold` exists. Verify the isolated `native.rehtest-*` league and acting role before each write. Follow rehearsal-plan 1B for claims, denial, sign-in read-back, rotation and revocation. Keep links and credentials out of evidence.
- **1A — phone checklist.** Codex requests a separate window with disposable owner and commissioner sign-ins ready on the S8 and iPhone. Confirm role and league, then follow the 56 rows in `~/fleet/runs/twr-v3-1117/MOBILE-V1-PARITY.md`. A missing fixture keeps the affected row blocked. Device evidence is in `~/fleet/runs/astra-exec-1125/1A-1B-PREFLIGHT-1125.md`.
- **BUILD-PLAN Stage 3 step 1 (T631).** Christopher accepted the 62-device v93 run for the Stage 5–6 gate. A 64-device run and integrity result remain separate proof and are outside this gate decision. Exact work order: `~/fleet/runs/twr-t631-1125/RESULT.md`.
- **T640 — Week 10 comparison.** Christopher or Claudebox needs read-only CT 257 access to the official Week 10 statistics pin and native results. Capture all 32 final `points` values using `~/fleet/runs/pulse-wk10/RESULT.md`. Compare to a fresh public MFL export. The older native capture was provisional.
- **T621 — CT 257 rotation.** Christopher or Claudebox needs privileged CT 257 console, service and key-file access to rotate the TLS origin and authenticator keys, update tunnel trust and verify sign-in and backups. Validate authenticator recovery before rotation.
- **T632 — Cloudflare.** Christopher decides access. Scope to the zone serving the approved TWR hostnames: Dynamic URL Redirects Read/Write for Single Redirects and SSL and Certificates Read/Write for per-host TLS settings. Changes are limited to those exact hostnames. Per-host TLS requires Advanced Certificate Manager. See `~/fleet/runs/astra-exec-1125/OPERATORS-1125.md`.
- **1C — Tuesday report and canary.** Codex's approved league-ID window stopped at preflight. The S8 was absent from ADB and its disposable session and role could not be verified. The next window needs the S8 connected with the verified throwaway Lab tab open. Read-token provisioning and T708 are separate actions. The report work order is in §7.1.
- **8A — owner correction explanation.** After Dec 6, Christopher approves a spec for showing the previous and corrected official figures and the reason. The headbrain owns the spec. The scoring/read owner then needs to expose the prior official version and an owner-readable correction event. Until that contract exists, leave the display blocked. To reproduce the evidence, use a disposable checkout at `e85334123`, run `git apply ~/fleet/runs/pulse-astra8b/capture.patch`, then `go test -race ./cmd/twr -run '^TestPulse8ACapture$' -count=1 -v`. The capture confirms changed totals and the missing explanation.

## Hourly progress (Central)

- 11-25 07:00 — #523 and #526 confirmed merged, Lab v95 verified; #527–#530 open; 8A blocked on the post-Dec 6 spec, canary blocked on T708. Stage 1 cleanup removed 41 clean worktrees and freed 3.05 GB; see `docs/reset/WORKTREES-1125.md` on `session/worktree-audit-1125`. Cleanup docs are open as #532 and practice-seat cleanup is open as #531. The merge queue is in §5. The 36 pre-existing dirty worktrees await Christopher’s keep/drop decisions.
- 11-25 08:00 — No new merges. #527–#532 open. Recorded the separate 1A and 1B gates. Started BUILD-PLAN Stage 3 step 1: v93 passed at 62 devices, with the 64-device rerun and integrity proof pending a quiet Lab window. 8A still awaits the post-Dec 6 spec, and the canary awaits T708.
- 11-25 08:27 — T631 coverage guard opened as #533. Stage 3 step 2 (T662) verified on main: #475 fixed the driver's week selection; all 4,110 v93 lineup reads returned 200. Evidence: `~/fleet/runs/twr-t631-1125/RESULT.md`.
- 11-25 08:53 — Stage 3 step 3 (T663) verified: Hermes timer active, Lab v95 matches main, ten offline hold checks passed. The operator must set the manual hold and verify no release is in flight before Sunday and Tuesday drafts. Evidence: `~/fleet/runs/astra-exec-1125/T663-RELEASE-1125.md`.
- 11-25 08:54 — Stage 3 step 4: T621 and T632 remain Christopher's security actions. T649 weekly audit passed with zero advisories when the withdrawn picomatch exception was omitted in scratch. Evidence: `~/fleet/runs/astra-exec-1125/T649-AUDIT-1125.md`.
- 11-25 08:54 — Stage 4: #478 and the recorded Nov 23 rulings are on main. Stages 5–6 await the full 32-team proof or Christopher's acceptance of the existing 62-device evidence.
- 11-25 08:57 — T640: saved native Week 10 results are provisional. A fresh public MFL read returned all 32 team scores, with five changes since the saved export. Await the authorized operator's official native read.
- 11-25 09:00 — No new merges. #527–#533 open; #533 is green. Repeated the T649 audit successfully after resuming. Stages 5–6 still await the T631 load gate; T621 and T632 await Christopher.
- 11-25 09:43 — Stage 2 evidence reconciled. #468 shipped T668/T677/T679; phone update and refresh captures and first-open timing still need an authorized observation. See `~/fleet/runs/astra-exec-1125/STAGE2-EVIDENCE-1125.md`.
- 11-25 09:45 — Rehearsal 1C: Tuesday report patch and 11 offline checks filed. Live rollout awaits the four verified leagues, scoped read tokens, and the Lab and Hermes operators.
- 11-25 09:58 — Christopher accepted the 62-device stage gate. #527 is on main at `ca9930d1e`; #528 repair is next. Stage 5 ranking and the operator handoff are filed in `~/fleet/runs/astra-exec-1125/`.
- 11-25 10:00 — #527 confirmed on healthy Lab v96 at 10:03. #528 feature/test transplant applies cleanly to current main in scratch. The replacement branch awaits one of the four tracked work PRs landing or closing. See the current access assignments under Needs Christopher.
- 11-25 10:22 — #527 recorded as squash-merged at 09:54. T621/T640 require Christopher or Claudebox on CT 257. T632 access scope documented for Christopher's decision. Tom remains a Lab app operator proposal for 1A, 1B and league-ID verification.
- 11-25 10:29 — Approved 1C league-ID window stopped at preflight: no S8 ADB device or verified disposable session. All four IDs remain unverified. Evidence: `~/fleet/runs/astra-exec-1125/1C-LEAGUE-ID-CHECK.jsonl`.

## 9. Lessons from 11-25 — do not repeat

- **Never plant a break with `git checkout -- <file>` on a lane's uncommitted tree.** It wiped Sol's change, and it had
  to be rebuilt. Use `cp file /tmp/x.bak`, then sed, then `cp` it back.
- **`python3 -I -m unittest` can't find modules in the current folder.** Use the repo's `make test-deploy` target.
- Regenerate the line-budget baseline on branches cut from current main. Record line growth in the reason file.
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
