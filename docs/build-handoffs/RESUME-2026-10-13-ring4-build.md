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
