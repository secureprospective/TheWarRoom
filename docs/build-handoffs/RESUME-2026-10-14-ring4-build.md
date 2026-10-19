# TheWarRoom ring 4 build: RESUME (current state 2026-10-14 08:35 CT; older addenda below)

## 1. What we are doing
Goal (Christopher): "finish ring 4", the commissioner side, native and app-owned (spec §16).
- Repo: Beelink `~/work/TheWarRoom`. The main checkout is Christopher's Lab (`session/league-history-pwa`): NEVER touch it.
- Ring 4 core was MERGED to main as 766b30a (#37, squash) on Christopher's go, 10-14 00:30.
- Remaining chunks (CHUNK-PLAN-draft.md): 4 shell, 5 legality, 6 calendar, 7 stats/standings, 8 rollover, 9 waivers/bids/DOT, 10 draft.
- Christopher asked for 5 and 6 side by side ("run 5 and 6 side by side now").

## 2. Agents
- **Sol first, always:** pi gpt-6.1-sol medium, `dispatch-wt.sh <phase> <brief>` with `WT=`.
- **Sol tokens:** Christopher at ~08:13: "sol has 12% 5hr tokens left, resets in 1hr 55min", so the reset is ~10:05.
  - Sol is OUT now. All three of its last runs stopped at budget.
  - After the reset, new phases go to Sol first.
- **Tom (stand-in):** Sonnet 5.5 high via `tom-dispatch-wt.sh` (WT=). Liveness: `<phase>/transcript.jsonl` mtime. Done: `<phase>/sentinel`.
- **Lesson:** a Sol run delivers about a quarter to half of a chunk before stopping, so briefs must be half-chunks or smaller.
- Every phase: I review, plant one mutant, commit with -F and the Co-Authored-By trailer. Never commit while a builder runs in that worktree.

## 3. In flight (as of 08:35), all Tom, started 08:20–08:24
| Phase | Worktree / branch | Brief | Covers |
|---|---|---|---|
| c5c-legality | .worktrees/ring-4-c5, session/ring-4-c5 (tip c5311b6) | par/c5c-legality.md | season resolver, roster 18-80, lineup, IR/taxi, house rules, governs |
| c4b-landing | .worktrees/ring-4-c4, session/ring-4-c4 (tip 99240e2) | par/c4b-landing.md | jobs reading, landing groups, control search |
| c6c-gate | .worktrees/ring-4-c6, session/ring-4-c6 (tip 7875625) | par/c6c-gate.md | native action gate, window commands, CM-023/025/026 governs |
- Check: `R=~/fleet/runs/warroom-ring4-2026-10-13; for p in c5c-legality c4b-landing c6c-gate; do ls -l --time-style=+%T $R/$p/transcript.jsonl; cat $R/$p/sentinel 2>/dev/null; done`
- On each finish:
  1. Read REPORT.md and the diff.
  2. Plant one mutant and restore it.
  3. Commit.
  4. Dispatch the next half-chunk (Sol if reset, else Tom).
- If one is killed, its reasoning is in its transcript.jsonl.

## 4. Branches and commits (none pushed except main)
- **session/ring-4-scoring-mfl** (from 766b30a):
  - 0cb7db0: the 10-14 rulings;
  - 2aa8f35: my fix, divisions of about four; no invented playoff field below 32.
  - This build is DEPLOYED on Claude-OS: sha256 e1abcb27…, backups r4b and r4c. Walked live, shots r4-40..57.
- **session/ring-4-c5** (from 766b30a):
  - 202364a: native seed from held data;
  - c5311b6: native franchise directory.
- **session/ring-4-c6** (from 766b30a):
  - c4e31dd: action calendar and exceptions. My reviewer change: the MFL-league TargetClock keeps the Rulebook §14 week 9 deadline;
  - 7875625: schedule generator and publish.
- **session/ring-4-c4** (from 0cb7db0): 99240e2, density and ledger filters.
- **Integration plan:** a new session/ring-4-integ2 from main, merging scoring-mfl, c5, c6, c4 (in that order). Then full verify and build, deploy to Claude-OS, walk, and ask Christopher for PR/merge.
  - Expect conflicts in `internal/commissioner/command.go`, `governance.go`, `ledger.go`, `store/history/commissioner.go` and `wailsjs` (regenerate).

## 5. Christopher's rulings this session (do not relitigate)
- **Scoring:** the Legacy preset matches MFL live. Now 1,775/1,783. The 8 residuals are unresolvable without MFL's credited stats; 7 are IDP tackle credits, plus London +1.
- **Time zone:** a change is confirm-only (G1).
- **Team count:** the Legacy template works for 2..32 teams.
- **Inactivity period:** the commissioner sets it (kept).
- **Two waiting changes:** the newest wins; the earlier one shows "Replaced by a later change on <local time>"; undo restores it.
- **My engineering calls (not asked):** takeover timing and the gravity table were kept.
- Questions to Christopher: one per question, short and plain. He rejected a multi-item question as "a wall of text".

## 6. Known open items
- Leagues created before 0cb7db0 keep their old seed. "Round each stat" shows Needs a ruling, so the scorer refuses them. These are test leagues only.
- The rules "Where it comes from" cell drops the Needs-a-ruling sentence when the citation has no "x: detail" colon. Fix in the c4 tray work.
- Chunk 8 (rollover) must apply only the newest waiting change per rule.
- **Remaining c4 work:** shared tray with preview/affects/blockers, hold-to-fire only in the tray, league ops (CM-051..054, M-093/095/137; NEVER send anything outward), governed glyph.
- **Remaining c5 work:** cap, contracts and pricing (CM-008/009), CM-034/038 commands.
- **Remaining c6 work:** CM-015 owner wiring (Now/Race).
- **Chunks 7–10:** not started.

## 7. Next actions
1. Review and commit each Tom phase as it finishes (section 3).
2. After ~10:05: dispatch the next half-chunks to Sol, two or three at a time, in disjoint worktrees.
3. When chunks 4–6 are done: integrate (section 4), gate, deploy, walk, then ask Christopher.
4. Then chunks 7 → 8 → 9 ∥ 10.

## 8. Environment
- Claude-OS: `ssh claudeos`. Deploy with `scp build/bin/thewarroom claudeos:warroom-ring1/thewarroom.new && ssh claudeos 'bash ~/warroom-ring1/deploy.sh <tag>'`.
  - One scp was cut off with "Connection reset by peer"; retry it and check the sha256.
- Walk helpers: `RUN/shot.sh`, `RUN/click.sh`. Control Room is at (87,384), League at (528,86). Scroll over the left margin (x=200) to move the page; over a table, the table scrolls.
- Gate env is in older sections below. Pre-push verify takes ~8 min.

## 9. Honest status / ETA
- Chunks 4–6 are each about half done.
- Chunks 7–10 are not started.
- At the current pace, with the Sol limits, ring 4 will most likely finish late on 10-15. Integration and the live walk add about 1–2 h per batch.

---
# Ring 4 build RESUME — 2026-10-13 18:55 CT (ring-4-integ tip 6235ee9)

Earlier history: RESUME.prev-1855.md, RESUME.prev-1708.md (same dir). Goal set by Christopher: **"finish ring 4"**
(a Stop hook holds the session to it).

## 1. What we are doing
Ring 4 = TheWarRoom commissioner core (spec rev 4 §16). Repo Beelink ~/work/TheWarRoom (192.168.1.191, user
chris). Main checkout = session/league-history-pwa = Christopher's Lab: NEVER touch/commit there.
RUN = ~/fleet/runs/warroom-ring4-2026-10-13. Integration branch session/ring-4-integ in
.worktrees/ring-4-integ. "Finished" = c-int2 forms reviewed + committed, full verify + build, deploy to
Claude-OS, my live walk with screenshots, then show Christopher and ask for a PR/merge go (squash).

## 2. Builders (Christopher's rules, 2026-10-13)
- **Sol first, always** ("always burn those tokens 1st"): pi gpt-6.1-sol medium. Codex usage limit hit 17:30;
  **resets ~20:40**. Probe: `pi -p --provider openai-codex --model gpt-6.1-sol --thinking low --no-session "Reply OK"`.
  Dispatch: `cd RUN; cat par/common.md par/<x>.md > par/<x>.full.md; WT=<wt> nohup setsid ./dispatch-wt.sh <phase>
  par/<x>.full.md > <phase>.launch.log 2>&1 < /dev/null &`. Done = `<phase>/sentinel`; alive = `<phase>/sessions/*.jsonl` mtime.
- **Tom only while Sol is out, and Tom = Sonnet high** ("if you use Tom builders they need to be Sonnet high
  thinking"): RUN/tom-dispatch-wt.sh (claude-sonnet-5-5 --effort high). Alive = `<phase>/transcript.jsonl`.
  Memory: tom-builders-run-sonnet-high.
- "pickup where they left off and take over" → Claude (head brain) may build directly; did c-int-c itself.
- Claude reviews every report: read core code, plant ≥1 own mutant, commit with -F (Co-Authored-By: Claude
  Opus 5.5 <noreply@anthropic.com>), never --no-verify. NEVER commit in a worktree while a builder works there.

## 3. In flight (check first)
| Phase | Started | Where | Brief | On result |
|---|---|---|---|---|
| c-int2 (Tom, Sonnet high) | 18:27 | .worktrees/ring-4-integ | par/c-int2.full.md | review, mutant, commit on ring-4-integ |
Check: `cat RUN/c-int2/sentinel`; `stat -c %y RUN/c-int2/transcript.jsonl`; `grep -c 'session limit' RUN/c-int2/transcript.jsonl`.
Recovery: the transcript. c-int2 scope: lazy League rules + People screens (shared picker; People shows votes and
succession), forms Start league / Change a rule / Give or take a role / Undo, HoldToFire (G2), typed reason (G3),
request keys + expected version, stale-revision reload. Files so far in frontend/src/app/commissioner/
(LeagueDesk, LeagueRules, LeaguePeople, LeaguePicker, StartLeagueForm, RuleChangeForm, RuleInput, RoleForm,
UndoEntry, ChangeBar, HoldToFire, hold.ts, forms.ts, ruleDraft.ts …).
Pre-reviewed already: forms.ts request-key logic correct; HoldToFire uses a submit-type Act but no <form>
exists, so a plain click cannot fire (keep it that way, or re-check if a <form> appears).

## 4. Committed
| Branch | Tip | Content | Review |
|---|---|---|---|
| main | 1f17d24 (#36) | spec rev 4 §16, census, CM registry | signed off |
| session/ring-4-integ | **6235ee9** | c1+c2+c3 (+rulings) → bf435e8 c-int people/governance authoriser → fe36292 c-int-b rules.set + preset/franchiseCount creation + Rules IPC + leaguerules catalogue.go → cf9c027 merge c-int-c → 6235ee9 merge c7a | each: own mutant caught |
| session/ring-4-c-int-c | 6124f4d | CM-022 succession/votes/elections (Claude took over Sol's draft; fixed: CC sets own inactivity period, caller-chosen in-season, target closes own vote-out, ballots G3, tie ruling private; added implicit activity) | 6 mutants caught |
| session/ring-4-c7a-scorer | aae59f4 | internal/fantasyscore pure scorer + nflverse adapter + golden vs MFL | mutant (solo-only tackles) caught |
Repo RESUME copy: committed on session/ring-4-c-int-c (integ is busy with c-int2); merge that commit into integ
after c-int2 is committed.

## 5. Open items / for Christopher (ask at hand-off, one at a time, as a matrix)
1. **Scoring preset vs MFL live config.** Golden 1,237/1,783 = 69.38%. 538/546 mismatches because MFL's live
   rules differ from Rulebook §4 (preset = rulebook exactly): CB/S TK +0.5, PD +0.5, IC +1; PAT ±1; ICY/FCY 0.1
   (preset 0.025); FU counts all fumbles; flat OFC; fumble-recovery TD 6; FG 60-99 band; per-stat cent rounding.
   8 unexplained (stat-credit differences). Question: Legacy preset follows the rulebook, MFL's live config,
   or rulebook + selected items. Preset unchanged until he rules.
2. Judgement calls to mention (veto-able): calendar.timezone counts as competitive (apply-now = G3); legacy_nfl
   only for 32 franchises; inactivity/voting periods commissioner-only; takeover waits the longer period while
   season dates need a ruling; rule gravity levels live in commissioner/rules.go (governance census has no gravity).

## 6. Refuted / do not redo
- Shared golangci-lint cache holds stale `../ring-4-registry/...` findings (worktree removed 16:20): not real.
  Use `GOLANGCI_LINT_CACHE=/tmp/<x>-lintcache` for gates/commits.
- A COnly=false mutant in rule/people auth "survives" only because governance.Allow's census consistency check
  also refuses it (defence in depth); use a wrong-row mutant instead.
- "17 unexplained" in the c7a brief was wrong; it is 8.
- Tom failures at 17:32 were the Claude session limit (reset 18:00), not the work.
- test_normalize.py, k6 survey, missed-FG boundary (0-29 −3 / 30+ −1), Inspector dead cap: settled earlier.

## 7. Decisions (Christopher, recorded in spec §16)
Authority: no MFL wiring now; native league, app ledger. One co-commissioner. Elections 17/32 else runoff top two
(ties in); runoff tie → CC (if not candidate) → DOT majority → Needs a ruling; 7-day periods. Legacy bids UFA+RFA
open, UDFA follows UFA. K4/K5 signed off (#36). Builder rules in §2.

## 8. Ledger / housekeeping
- Caches deleted + logged (MOVED, MANIFEST, CT105 backbone): ring 3 (15:50), K4 worktrees (16:20), ring 4 phases
  (17:07, 18:50). KEEP RUN/c-int-b/go-cache (30 GB, written after 18:00 by an unidentified live process).
- New worktree .worktrees/ring-4-c-int-c (git worktree add, 17:26). Filing gate PASS. Hermes T434 "BUILDING".

## 9. Next actions, in order
1. When c-int2's sentinel appears: read RUN/c-int2/REPORT.md; review LeagueDesk/forms/HoldToFire/RuleInput; plant
   a mutant (e.g. G3 confirm enabled with blank reason, or HoldToFire fires on early release); commit on
   ring-4-integ with a private lint cache.
2. Merge session/ring-4-c-int-c's RESUME commit into ring-4-integ.
3. Run `timeout 900 nice -n 10 make verify VERSION=dev COMMIT=ring4 BUILD_DATE=ring4` and `make build …` on ring-4-integ
   (gate env in CLAUDE.md, plus GOLANGCI_LINT_CACHE). Check entry bundle ≤ 321,818 B.
4. Deploy: `gzip -k build/bin/thewarroom`; scp to claudeos:~/warroom-ring1/thewarroom.new.gz; ssh: gunzip, chmod
   +x, `bash ~/warroom-ring1/deploy.sh <prev-tag>`; wait for "value model fitted" in ~/warroom-ring1/run.log.
5. Live walk on Claude-OS (1920×1080; Control Room nav (87,384), League tab (528,86); RUN/shot.sh, click.sh):
   start "Walk test 10-13" legacy_nfl; rules screen shows $125M / 18 to 80; set roster limits 20..70 next
   rollover (pending shown, 18 to 80 still in force); then now with reason (in force); undo (back to 18 to 80);
   grant co-commissioner; People + ledger. Screenshots RUN/shots/r4-*.png. Look at every screenshot.
6. Show Christopher (screens + §5 items), ask for the PR/merge go; push (pre-push verify ~8 min), open the PR.
7. From ~20:40 route any further phase (chunk 4: shell, jobs strip, trays, hold-to-fire promotion) to Sol.

## 10. Environment
CT105 `ssh root@192.168.1.105` (backbone log append-only); Hermes `ssh hermes` (`cc task show T434`); Claude-OS
`ssh claudeos` (screenshots via scp, never rsync). Never: MFL league 14432 / commissioner pages; ssh -n with
heredoc; pkill -f over ssh; production binary on the Beelink; agents on the Beelink desktop.

## 11. Honest status
Go side of ring 4 integrated and gated per phase; forms building on Tom. Nothing in ring 4 seen live yet.
Full verify on the merged integ tip not yet run. ETA: c-int2 ~19:15, gate + deploy + walk → first look ~20:15.

## 18:53 addendum (in-flight risk)
c-int2 Tom (pids 2599002/2599003, dispatcher 2598996) ended a turn with "The gate (`make verify`) is running. I'm
waiting for it to finish before planting break-tests and writing the report." Process still alive (expected to
resume on its background task). If the sentinel appears with report=NO: resume that session with
`cd ~/work/TheWarRoom/.worktrees/ring-4-integ; claude -p --resume <session_id> --model claude-sonnet-5-5 --effort high
--permission-mode bypassPermissions "Finish: break-tests, gates, REPORT to RUN/c-int2/REPORT.md"` (session id below),
or finish the review/gates myself.
"session_id":"6392ab13-5299-423f-bb2d-162c9a7f12a1"

## 23:15 addendum
c-int2 Tom finished 19:10 (EXIT=0, report=yes, verify 0, build 0, 44 mutants red). Claude's session stalled
19:10-23:00 on its own usage limit. Review done: my mutant (ChangeBar reason not trimmed) caught by 2 tests.
Fix added by Claude (uncommitted, in integ worktree): commissioner.Refusal / Refused() marks rule-decided
refusals; IPC returns Reason() = innermost sentence ("This league already has a co-commissioner: revoke them
first") instead of the wrapped chain; stale version is a typed staleVersion (Is ErrRevisionConflict). Mutant
(unmark people refusal) caught. Full verify+build on integ running: RUN/integ-gate/{verify,build}.{log,exit}.
Then: commit c-int2 + refusal fix on ring-4-integ, merge ring-4-c-int-c (RESUME doc), deploy, walk.
Claude-OS binary was relaunched 22:24 (ring 3 binary from 15:24); not by this session (observed only).

## 23:25 addendum
- Committed on session/ring-4-integ: 9a83dca c-int2 (forms + Refusal fix; full verify 0, build 0, 969 FE tests,
  entry 312,717 B) and 014eeb0 merge of ring-4-c-int-c (RESUME doc).
- Deployed 9a83dca build to Claude-OS 23:13 (sha256 eb7f71b7…; backup tag "ring3"); value model fitted 23:14.
- Live walk 23:14-23:19, shots RUN/shots/r4-01..21: start league legacy_nfl OK; next-rollover change shows
  Waiting with 18 to 80 in force OK; Now + reason applies OK; undo OK (version 3→4, original spent); ledger
  local times OK; People OK; cap $125M (wire). Role grant cannot be walked live (no seated owners; invites need
  a second app) — covered by Go IPC test.
- 9 polish defects found → Sol phase c-int2p (dispatched 23:21, brief par/c-int2p.md, WT ring-4-integ):
  hold button layout, lingering status, census/fact-R jargon, UTC date, start form always open, empty role list
  unexplained, undo label layout, duplicated scoring source, "until ... is built" phrasing.
  Check: cat RUN/c-int2p/sentinel; alive = RUN/c-int2p/sessions/*.jsonl mtime. Do NOT commit in integ while it runs.
- Then: review + mutant, commit, gate, redeploy, re-walk the changed screens, show Christopher, ask PR go.

## 00:00 addendum (ready for Christopher)
- c-int2p (Sol) done 23:48; reviewed (mutant: empty preset labelled Legacy → caught); committed fdda111 on
  session/ring-4-integ. Full verify 0 / build 0, 978 FE tests, entry 312,717 B.
- Redeployed fdda111 to Claude-OS 23:56 (sha256 618136fb…, backup tag r4a). Re-walk shots r4-30..37: plain sources,
  $125M, start folded, role explanation, hold fill inside full-width button + status below + clears. All 9 fixed.
- New open item: two pending changes to the same rule both show "Waits for the next rollover" — which applies at
  rollover is undefined on screen (rollover not built). Raise with Christopher.
- NEXT: ask Christopher (matrix, one at a time): PR/merge go; scoring preset (rulebook vs MFL live); veto calls.
  On go: push session/ring-4-integ (pre-push verify ~8 min), gh pr create, squash merge only on his go.

## Addendum 10-14: PR open
Christopher chose "Push + open PR". The push passed its pre-push verify (push=0, integ-gate/push.log). PR #37 is open and MERGEABLE, 18 commits, and has no CI checks. T434 updated (Hermes 1823473). Next: merge only on his explicit go (`gh pr merge 37 --squash`), then ask questions (b) preset, (c) veto calls, (d) two waiting changes.

## Addendum 10-14: merged
On Christopher's go, #37 was squash-merged into main as 766b30a. Still to ask: (b) scoring preset, (c) veto calls, (d) two waiting changes.

## Ruling 10-14 (b): scoring preset = MATCH MFL LIVE
The Legacy preset copies MFL's live config: CB/S TK +0.5, PD +0.5, IC +1, PAT ±1, ICY/FCY 0.1, FU all fumbles, flat OFC, fumble-recovery TD 6, FG 60-99, per-stat cent rounding. Rulebook §4 gets corrected later. Target: golden 1,783/1,783 except the 8 unexplained, which need investigating. This needs a new phase (Sol first).

## Rulings 10-14 (Christopher), phase r4-rulings dispatched to Sol 06:59
- Time zone change: confirm only (G1), no reason needed.
- Legacy template: any team count from 2 to 32.
- Inactivity period: commissioner sets it (kept as built).
- Two waiting changes to one rule: the newest wins, the earlier shows "Replaced by a later change".
- Scoring: match MFL live (see above).
- Takeover timing and the seriousness table are my engineering calls; kept as built.
Brief: par/r4-rulings.md. Worktree .worktrees/ring-4-scoring-mfl (branch session/ring-4-scoring-mfl from main 766b30a). Sentinel r4-rulings/sentinel; transcript r4-rulings/sessions/*.jsonl. Next: review with mutants, run the gates, redeploy to Claude-OS, walk it, then PR on his go.

## 10-14: chunk 4 brief drafted
par/c4-shell.md (Christopher chose "plan the next chunk"). Dispatch it to Sol AFTER r4-rulings is reviewed and committed, on a worktree branched from session/ring-4-scoring-mfl (shared commissioner frontend files). Pre-check notes: r4-rulings-claude-precheck.md.

## 10-14 07:17: chunks 5 and 6 dispatched side by side (Christopher: "run 5 and 6 side by side now")
- c5-legality: .worktrees/ring-4-c5, branch session/ring-4-c5 from 766b30a.
- c6-calendar: .worktrees/ring-4-c6, branch session/ring-4-c6 from 766b30a.
- Three Sols are running with r4-rulings. File ownership is split in each brief. Both are Go-only; their screens come with chunk 4.
- Integration order: r4-rulings, then c5 and c6 merged into one integ branch, then chunk 4 on top.

## 10-14 07:34: r4-rulings reviewed and committed
Committed 0cb7db0 on session/ring-4-scoring-mfl. MFL scoring match is 1,775/1,783; the 8 residuals were rechecked and remain unresolved (tackle credits). My mutant (negative half-cent rounding) was caught. Note for chunk 8 (rollover): it must apply only the newest waiting change per rule, as the reading does.
- 07:35: c4-shell dispatched. Worktree .worktrees/ring-4-c4, branch session/ring-4-c4 from 0cb7db0. Three Sols are running (c4, c5, c6).

## 10-14: chunk 5 part 1 committed
- c5-legality stopped at budget after the seed. Committed 202364a; my mutant (contract year +1) was caught.
- c5b-resolver dispatched: the resolver, shape, roster, lineup, IR/taxi, house rules and governs proofs.
- Next, c5c: cap, contracts, CM-034/038 and their governs proofs.

## 10-14: chunk 6 part 1 committed
- c6-calendar stopped at budget. Committed c4e31dd.
- Reviewer change: the MFL-league TargetClock keeps the Rulebook §14 week 9 kickoff deadline. My brief had wrongly asked for it to be replaced; spec §16.2 keeps owner MFL readings.
- My mutant (deadline not localised) was caught.
- c6b-schedule dispatched: schedule, native action gate, window commands, governs proofs.
Lesson: Sol stops at about 30 minutes of budget, so phases must be half a chunk.

## 10-14 08:12: rulings build deployed to Claude-OS and walked (Christopher: "deploy the rulings build")
Build 2aa8f35, sha256 e1abcb27, backup tags r4b and r4c.
- The walk found 12-team Legacy with 8 divisions and an invented 6-per-conference playoff field. Fixed in 2aa8f35 (divisions of about four; playoff field Needs a ruling below 32); mutants red.
- Verified on screen (shots r4-40..57):
  - replaced waiting change ("20 to 70: Replaced by a later change on 10/13/2026, 11:58:28 PM"; 22 to 75 waiting);
  - new leagues show MFL scoring, labelled "Legacy NFL rules (the league's MFL scoring)", with the CB/S rows and 0.1 return yards;
  - 12 teams gives 2 divisions, with the playoff rows marked Needs a ruling.
- Known and not fixed:
  - leagues created before the rulings keep the old seed; their "Round each stat" shows Needs a ruling, so the scorer would refuse them. These are test leagues only.
  - the "Where it comes from" cell drops a Needs-a-ruling sentence that has no "code: detail" colon. Polish for c4.

## 10-14 08:20: chunk 5 part 2a committed; part 3 to Tom
- c5b-resolver delivered only the native directory. Committed c5311b6; my mutant (undone renames still applied) was caught.
- Sol has 12% of its 5-hour tokens left and resets at about 10:05, so no new Sol dispatch.
- c5c-legality went to Tom (Sonnet high) in .worktrees/ring-4-c5: resolver, roster, lineup, IR/taxi, house rules, governs.

## 10-14 08:24: c4 part 1 and c6 part 2a committed; Sol out until ~10:05
- c4-shell delivered density and ledger filters: 99240e2, my mutant (inclusive end date) caught.
- c6b-schedule delivered the generator and publish: 7875625, my mutant (byes not counted) caught.
- Tom (Sonnet high) now on c4b-landing (jobs reading, landing, search), c6c-gate (native gate, window commands, governs) and c5c-legality.
