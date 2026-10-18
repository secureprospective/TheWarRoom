# UPDATE 2026-10-18 16:05 CT: compact-safe point (READ THIS FIRST; supersedes the 15:45 update and the 14:45 block's in-flight/next/ledger)

## State
- integ2 (`~/work/TheWarRoom/.worktrees/ring-4-integ2`, branch session/ring-4-integ2) tip **d792e394** (c8b merged). On top of it a **c9c merge is STAGED, UNCOMMITTED**: conflicts resolved (command.go union of G3 cases + Correction/Trade/TradeOutcome args; native_seed NativePick has Compensatory + Correction; draft_reading kept HEAD's "draft was run before the league began" guard; SYSTEM_MAP union; models.ts regenerated), runtime modes restored to HEAD, wire fixtures refreshed. `make verify` running → `integ2/verify27.log` (started 15:30, nohup; at 16:01 in the root race run). Waiter task bybobh9ee.
- Phase branches committed: c8b 070e17a8 (merged), c9c 7fa65ca9, **c10b 963e4f77** (Tom done 15:58; reviewed; my mutant restore-set-aside red→green). c10b WT base is f709bc39 (old): expect conflicts with c9b/c-perf/c8b/c9c. Known one: native_reading.go — integ2 has `seedNativeAssets` (my c8b refactor); c10b adds `seed.Pool = slices.Clone(seed.Pool)` → put it inside seedNativeAssets.
- **Sol c9b2-bids running** since 15:37 in WT ring-4-c9b2 (from d792e394). Liveness: mtime of `c9b2-bids/sessions/*.jsonl`. Done: non-empty `c9b2-bids/sentinel` (waiter b5pqwi1c3). Brief par/c9b2-bids.full.md (note at top).
- go-build clear job bmrej1pzx: still waiting for an idle box (no go processes). If lost after compaction: run `/usr/local/go/bin/go clean -cache` by hand when no builder/verify runs, then log in MOVED/MANIFEST/CT105.
- Caches of c8b, c9c, c10b deleted and logged (15:40, 16:01). /home 185 GB free.

## Next actions, in order
1. When verify27 shows VERIFY_EXIT=0: commit the c9c merge in integ2 with -F ("Integrate chunk 9c native trades…", note the review fixes: stalled DOT review → commissioner override; PendingTradeCapCheck real). If it fails: read the FAIL, fix, rerun. If verify27 has no exit and no make process (session restarted): rerun verify.
2. Merge session/ring-4-c10b into integ2 (`--no-ff --no-commit`), resolve, `wails generate module`, `git checkout HEAD -- frontend/wailsjs/runtime`, `TWR_WRITE_WIRE=1 go test -count=1 -timeout 45m -run 'Wire|TestScheduleEditOwnerRevertRestart' . ./internal/commissioner/`, also rerun Tom's draft wires (`TestTheDraftRoomWireIsWhatGoSends|TestDraftReadingSurvivesReopeningAndFixesTheWire|TestNativeDraftEngineWire`), make verify, commit.
3. When Sol c9b2 lands: review REPORT + diff, plant one mutant, fix, commit on its branch, merge onto integ2, verify, commit, delete caches + log.
4. Fix the jobs list (`internal/commissioner/jobs.go`): rookie draft, stat queue, playoffs (and anything else built in ring 4) still say "Not built yet"; c9b2 handles CM-035/036. Show built ones with their real state; test + mutant.
5. Final: verify, `make build VERSION=dev COMMIT=ring4 BUILD_DATE=ring4`, deploy to Claude-OS (scp to `claudeos:warroom-ring1/thewarroom.new`, `~/warroom-ring1/deploy.sh <old-sha>`), walk the commissioner screens (draft room, trades, corrections, bids) live, PR body from integ2/PR-notes.md (kept current through c10b), ask Christopher one plain question, squash-merge on his go, update Hermes T434.

## Refuted (do not retest)
- Root tests too slow for 45m: it was CPU contention. Fixed by the catalogue cache + c-perf.
- A planted `false && a || b` mutant "surviving" was my mutant's precedence, not a weak test; plant mutants as separate assignments.

## Honest status
c8b merged; c9c merged pending verify27; c10b committed, not merged; c9b2 building. Then jobs-list fix, final verify, live walk, PR. Nothing pushed. Nothing walked live yet.

# UPDATE 2026-10-18 15:45 CT (supersedes the in-flight table, next action 2, ledger and status below)
- **c8b corrections: merged** as d792e394 (verify26 failed only on the stale safeguards wire fixture, refreshed; wire + commissioner vitest pass). Review fix: the roster-add contract lookup is now one pass (it was quadratic).
- **c9c trades: committed** 7fa65ca9 on session/ring-4-c9c. Review fixes: a DOT review that can't reach the vote count goes to the commissioner's override; PendingTradeCapCheck is real (corrections wait for a pending trade). **Merge into integ2 is staged, uncommitted** (5 union conflicts resolved, bindings regenerated, runtime modes restored, wires refreshed); `make verify` → integ2/verify27.log. On VERIFY_EXIT=0: commit the merge with -F. If the session restarted: check `git -C ~/work/TheWarRoom/.worktrees/ring-4-integ2 status`, rerun verify if verify27 has no exit.
- **c9b2 bids: dispatched to Sol 15:37** in NEW WT ring-4-c9b2 (from d792e394, so it needs a merge onto the c9c merge). Transcript c9b2-bids/sessions/*.jsonl. Brief has a continuation note.
- **c10b draft room: Tom still running** (WT from f709bc39).
- Caches of c8b, c9c deleted and logged (15:40). go-build clear: still pending an idle moment (no go processes); run `go clean -cache` by hand when builders are quiet.
- Remaining: c10b, c9b2 → final verify, build, Claude-OS walk, PR.

# RING 4 BUILD: CURRENT STATE 2026-10-18 14:45 CT (read this block first; history below)

**Goal:** "finish ring 4", the commissioner side, native and app-owned (spec §16). Repo `~/work/TheWarRoom`. The main checkout is Christopher's Lab branch: NEVER touch it. The integration branch `session/ring-4-integ2` is in WT `.worktrees/ring-4-integ2`, tip **c150325a** (verify25 exit 0, 26 min, root race 1461s). Nothing is pushed.

**Christopher 10-18:**
- Sol has fresh tokens; two Sols at once is fine. Tom is still allowed, one at a time; he shares my Claude account.
- Disk hygiene matters: clear each phase's caches at merge.
- Clear `~/.cache/go-build` (31 GB) after verify. A job is armed for it (see below).

**Standing rules:**
- no email or outgoing messages;
- CT105 observe-only;
- log every move or deletion in MOVED.md, MANIFEST.md and CT105 context.md;
- never open MFL league 14432 or submit to MFL;
- no `--no-verify`; squash-merge only on Christopher's go;
- scp only, never rsync;
- per-deploy approval for live sites (Claude-OS is the test VM);
- no GLM;
- commit with `-F` plus the Co-Authored-By trailer.

## In flight (check: transcript mtime; done = non-empty `<phase>/sentinel`)

| Phase | Builder | WT | Started | Transcript | Brief |
|---|---|---|---|---|---|
| c10b-draftroom | Tom (Sonnet 5.5 high) | ring-4-c10b (from f709bc39, so it needs a merge onto c150325a) | 13:44 | c10b-draftroom/transcript.jsonl | par/c10b-draftroom-c.full.md (third run; earlier ones died at limits → .limit-2034, .limit-2310) |
| c9c-trades | Sol | ring-4-c9c (from c150325a) | 14:02 | c9c-trades/sessions/*.jsonl | par/c9c-trades-sol.full.md (first run died on ENOSPC → .enospc-1355; WT reset) |
| c8b-corrections | Sol | ring-4-c8b (from c150325a) | 13:55 | c8b-corrections/sessions/*.jsonl | par/c8b-corrections-sol.full.md |

- Tom's five-hour window was 69% used at 14:37 and resets at 18:40. If he dies on a limit, rename the dir `.limit-HHMM` and relaunch with a continuation note that keeps his edits.
- The go-build clear job is a background bash: it waits for VERIFY_EXIT in integ2/verify25.log (done), then for no `go`/`.test` process, then runs `go clean -cache`. If the session restarts it is gone: re-run `go clean -cache` at an idle moment, then log it in MOVED/MANIFEST/CT105.

## Next actions
1. For each finished builder:
   - read REPORT.md;
   - review the diff;
   - plant one mutant and check it goes red, then green;
   - fix what is wrong;
   - commit on its branch with -F;
   - merge into integ2 (`--no-ff --no-commit`, resolve, `wails generate module` + revert `frontend/wailsjs/runtime/*` if bindings changed, rerun Wire tests with TWR_WRITE_WIRE=1 `-timeout 45m -run 'Wire|TestScheduleEditOwnerRevertRestart'`);
   - make verify;
   - commit;
   - delete the phase's caches (go-cache, cache, lint-cache, tmp) and log it.
2. After c10b merges: create WT ring-4-c9b2 from the integ2 tip, `cp -r frontend/dist`, and dispatch c9b2-bids (par/c9b2-bids.full.md) on the first free builder (Sol preferred: `WT=… nohup ./dispatch-wt.sh <phase> <brief> &`).
3. Final steps:
   - verify, then `make build VERSION=dev COMMIT=ring4 BUILD_DATE=ring4`;
   - deploy to Claude-OS (scp to `claudeos:warroom-ring1/thewarroom.new`, then `~/warroom-ring1/deploy.sh <old-sha>`);
   - walk the commissioner screens live;
   - write the PR body from integ2/PR-notes.md;
   - ask Christopher one plain question about the PR and squash-merge;
   - update Hermes T434.

## Refuted (do not retest)
- "The root tests are too slow for 45m": the 10-15 FAIL (2400s) was CPU contention with Tom. Alone: ContractActionCommandsReplay takes 89s (it showed 900s), BidPhaseRules 5.9s (it showed 35s). Fixed for real: the catalogue cache (db0d78be), then c-perf.

## Ledger (integ2 first-parent, newest first)
- c150325a c-perf merge
- db0d78be catalogue built once
- 42b426a7 c9b bids merge
- f709bc39 c7h
- 1e1c1e70 c7g
- 0b707a57 c5g
- 61be6beb c9a
- 6eced62d c10a
- f180951a c8a
- eca63404 c7f
- …

Nothing is uncommitted on integ2. PR notes: integ2/PR-notes.md.

## Honest status
Four phases remain: c10b, c9c, c8b and c9b2. After those come the final verify, the Claude-OS walk and the PR. None of the three running phases has reported yet.

---
## In flight at 22:50
1. **Tom c8a-rollover-3**
   - Started 22:33 in WT `ring-4-c8`, branch `session/ring-4-c8`, base 7045814 plus uncommitted work.
   - Dir: `c8a-rollover-3/` (the old Sol attempt is in `c8a-rollover-3.sol-dead`).
   - Brief: `par/c8a-rollover-3.full.md`. It includes the binding season-numbering fix (remove `ledgerSeason=1`; creation records the NFL year).
   - On finish: read REPORT.md, review the diff, verify in the WT (needs frontend/dist), plant a mutant, commit in the WT, merge into integ2.
2. **verify5 on integ2**
   - Log: `integ2/verify5.log`, ending with `EXIT=`.
   - It covers the staged, UNCOMMITTED merge of `session/ring-4-c7f` (e0d80e5e) into integ2 (tip 214790ee).
   - Merge resolutions:
     - history/commissioner.go: `checkCommissionerCommand(..., lineup LineupFacts)`; preview also calls `lineupFactsFor` before BeginTx;
     - command.go validateDetails: strays include `a.Lineup != nil` and the Note check;
     - SYSTEM_MAP: the c7f results paragraph plus the Native lineups section, with c4 sections kept.
   - Bindings regenerated (runtime reverted); wires regenerated (commands, replaced, rules.legacy, native-results, native-lineup).
   - If EXIT=0: commit "Integrate chunk 7f native lineups into the integration branch" (-F, trailer). If red: fix it (likely the frontend registry needs the `commish.lineup.correct` row, or a controls test).

## Committed state
- integ2:
  - fe47853 c5;
  - 33a035d c6d;
  - 5effec0f c6 schedule + c7 results/standings;
  - **214790ee c4 merge** (verify4 0; fix: withWaitingInForce stamps the counted waiting change at the newest recorded instant).
- c7f: **e0d80e5e**. Tom's lineups (22 mutants red), plus my Legacy `lineups` window (3/1, no close, opens `lineup`, cited "Ring 4 lineup ruling: open all year, each player locks at kickoff"), plus test `TestLegacyLineupsStayOpenAllLeagueYear` (mutant red/green).
- WT `ring-4-c7g` (branch session/ring-4-c7g) created from 214790ee with dist. **Recreate from the integ2 tip after the c7f merge** (it is untouched): `git worktree remove` it and add again, or merge integ2 into it.

## Tom queue after c8a (serial)
- Order: c10a-draft-2 (WT c10) → c9a-waivers-2 (WT c9) → c5g-pricing-2 (WT c5g) → c7g-playoffs (CM-016, 046-049) → c7h-statqueue (CM-031) → c9b-bids → c10b-draftroom → c9c-trades → c8b-corrections.
- Briefs are all in `par/` with `.full.md`.
- c4–c10a WTs are based on older integ tips; merge integ2 into them, or merge them into integ2 and resolve.
- WTs for c7h/c8b/c9b/c9c/c10b do not exist yet: create each from the integ2 tip after its dependencies merge, and copy frontend/dist.

## Final steps (after the queue)
1. Merge all into integ2; full verify + build.
2. Deploy to Claude-OS (`scp build/bin/... claudeos:warroom-ring1/thewarroom.new`, then `deploy.sh <tag>`; no "Read data again" click, no MFL).
3. Walk it live.
4. Write the PR body (notes: Read data again is not role-gated by design; conference/power-rank criteria pass with no data; no production complete scores until play-by-play).
5. Ask Christopher one plain question about the PR and squash-merge.
6. Update Hermes T434.

## Rules that bite
- Never commit in a WT while Tom works there.
- Never two golangci-lint at once (global lock: if it fails with "parallel golangci-lint is running", rerun).
- Root race test about 520 s; Makefile timeout 25m.
- After `wails generate module`, `git checkout -- frontend/wailsjs/runtime/`.
- Never commit `.cache/`.
- Mutant: confirm the tests actually ran.
- The commit hook runs golangci-lint-full (slow).
- No email or messages without permission. CT105 is observe-only. No MFL. No work on main. No --no-verify.

---
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

## 10-14 11:50: Tom phases died at session limit (08:36); recovered
- c5c-legality, c4b-landing, c6c-gate all ended "session limit · resets 11:40am", no report, work left uncommitted.
- c6c-gate: implementation + 8 mutants were done. Claude reviewed, gated (verify 0, build 0, bundle 312,895), own mutant (reverted window still counted) caught → committed **a2fda4f** on session/ring-4-c6. Report RUN/c6c-gate/REPORT.md.
- Sol (reset) dispatched 11:40 on continuations: **c5c2-legality** (WT ring-4-c5) and **c4b2-landing** (WT ring-4-c4), briefs par/*-resume.full.md, prior diff in RUN/<old-phase>/prior.diff.
- New worktree `.worktrees/ring-4-c7`, branch session/ring-4-c7 from 121e588; **c7b-stats** dispatched 11:49 (CM-030/031/032 + TargetNativePlayerScore, Go only).
- c6 remaining: CM-015 owner wiring (Now/Race) — waits for c4 frontend.

## 10-14 12:13: c5c2 committed 3f4071d (season resolver RulesInForce, internal/nativecheck roster/lineup/IR/taxi, house rules; 10 Sol mutants + mine: penalty tie-break → red). c5d-cap dispatched to Sol 12:12 (CM-008 cap reading, CM-034, CM-038). Next c5 phase: c5e CM-009 contract pricing. c8a-rollover brief drafted in par/ (dispatch after c5+c6 integration).

## 10-14 12:18: c4b2 committed c83a9b4 (League landing, jobs, search). Mine: inclusive-close mutant → red. Deployed to Claude-OS tag r4d (sha d8b737da549a81a5), walked shots r4-60..62: jobs strip, search "trade deadline" → opens Calendar group + focus OK. Defects → c4c brief: "Needs a rulingChange" run-together, non-live jobs not visibly dimmed. Endpoint index (ring 0) shows M-codes — pre-existing, left. "Window exception: Not built yet" on c4 branch flips once c6 integrates (update controls.go availability then).
- c4c-tray dispatched to Sol 12:17 (tray + hold-to-fire everywhere + 3 fixes). Running: c5d-cap, c7b-stats, c4c-tray.

## 10-14 12:40: c7b-stats stopped after item 1 (stat port). Its verify failure was MINE: 2aa8f35 left frontend/src/app/commissioner/replaced.wire.testdata.json stale → regenerated, committed 3adc6fd on scoring-mfl and carried into c7. My mutant (POST row read as REG) survived → added held_test.go, now red. Verify 0, build 0 → committed 932e1e2 on session/ring-4-c7. c7c-score dispatched 12:40 (identity crosswalk, CM-030, CM-032, TargetNativePlayerScore). Next c7: c7d CM-031 queue, then results/standings (CM-007/016/033).
- The Needs-a-ruling citation drop is in Go: internal/commissioner/rules_reading.go startingSource (no colon → bare "Needs a ruling"). c4c-tray was told to fix it (fix 3).

## 10-14 12:45: c5d-cap partial → committed 6ced4cf (TargetNativeCapCheck + CapAllows + CM-008 proofs; mine: exact-cap boundary >= → red). c5e-overrides dispatched (CM-034, CM-038, plain-words deadline). Then c5f: CM-009 contract pricing.

## 10-14 12:58: c4c-tray partial → committed 09bdf4f (3 live-walk fixes incl. Go citation sentence; mine: Needs-prefix dedupe removal → red). Tray split: c4d-preview dispatched (Go dry-run preview + tray + RuleChangeForm); next c4e: other 3 forms + no-G2-by-click test; then league ops, glyph.

## 10-14 13:45: integration + gate policy change
- **session/ring-4-integ2** (WT .worktrees/ring-4-integ2) = scoring-mfl 3adc6fd + c5 (88691d1) + c6 (83d0dff). Verify 0 (bundle 313,647). Resolutions: ruleHistory = c5 rollover-aware + clock-free `replacedWaiting` (ledger keeps "Replaced" after rollover); window commands fold into c5's `seasonChanges` (`seasonFold`); RawArgs union + one `validateDetails`; complexity splits (publicNowRule, retriedReceipt/checkAppend, waitingChange/markReplaced). Mutant (windows dropped from fold) → 7 red. c5 commits AFTER 6ced4cf (1796e9f, c5f…) still need merging into integ2.
- c7 merged integ2 → 3d9cd42; score commands now typed `Args.Score`; governance `commandControls()` table. c7d-reading dispatched 13:30 (TargetNativePlayerScore + proofs).
- c4d-preview committed 4a56ecf (Go TargetCommissionerPreview sharing checkCommissionerCommand). NOTE at integration: c4's checkCommissionerCommand vs integ2's retriedReceipt/checkAppend → make preview call checkAppend.
- c5e committed 1796e9f (viewer-local cap deadline; CheckCap/CapAllows require zone).
- **common.md changed**: Sol runs targeted tests + lint only; Claude runs make verify/build before every commit (old copy par/common.prev-1340.md). Reason: verify ~8 min = half of each Sol run.
- Running (Sol): c7d-reading (c7), c5f-overrides (c5: projector, CM-034, CM-038), c4e-tray (c4: tray + RuleChangeForm).
- Queue: c8a-rollover on a new worktree from integ2 (after c5f merges in); c4f (other 3 forms + no-G2-click); c4 league ops + glyph; c5 CM-009 pricing; c7 CM-031 queue, results/standings; chunks 9, 10.

## 10-14 13:56: Christopher: Sol 20% of 5h left, resets in 2h50 (~16:45). Run at most 2 Sols; when Sol errors out, route next phases to Tom (tom-dispatch-wt.sh), back to Sol after ~16:45. c4e-tray done in 11 min under targeted-check policy; Claude gating it.

## ===== 10-14 14:25 COMPACT POINT — current state (read this first) =====
**Branch tips (none pushed; main = 766b30a):**
- session/ring-4-scoring-mfl 3adc6fd (rulings + Legacy template fix + wire regen) — WT ring-4-scoring-mfl
- session/ring-4-integ2 83d0dff = scoring-mfl + c5@6ced4cf + c6@a2fda4f (verify 0) — WT ring-4-integ2
- session/ring-4-c5: 3f4071d (resolver/legality), 6ced4cf (cap), 1796e9f (viewer-local deadline); **c5f2-overrides UNCOMMITTED in WT** (CM-034 overrides + projector, CM-038 reserve exceptions; Sol 12 mutants red; targeted checks only)
- session/ring-4-c6: a2fda4f (gate + window cmds). Remaining c6: CM-015 Now/Race owner wiring.
- session/ring-4-c4: 09bdf4f (fixes), 4a56ecf (Go preview), **d849273** (tray + RuleChangeForm; my fixes: Close via Act surface 'close', SideTray keyboard allowlist pinned, portal to .twr-app, refused preview parse). Deployed Claude-OS tag r4f sha 4559b8a936d9edca, walked shots r4-75..78: preview shows question/target/sources/before→after/affects/no blockers/hold. Seen: no-op change (9 → 9) not blocked → add Go blocker "already the rule" next c4 phase; couldn't switch select to Now via xdotool.
- session/ring-4-c7: 932e1e2 (stat port), 3d083de (identity, CM-030/032), 3d9cd42 (merged integ2; typed Args.Score; commandControls table); **c7d-reading UNCOMMITTED in WT** (TargetNativePlayerScore + proofs; Sol 15 mutants red, Sol ran full gates 0).

**In flight at compact:** background task btome98fk = my `make verify` on ring-4-c5 and ring-4-c7 (logs RUN/c5f2-overrides/verify-claude.log, RUN/c7d-reading/verify-claude.log). No Sol/Tom builders running.

**NEXT ACTIONS, in order:**
1. Read btome98fk result. If c5 verify=0: plant one mutant (e.g. reserve exception for other bucket), restore, commit c5f2 on c5 (msg: "Native overrides: contract/cap overrides and reserve exceptions (chunk 5, part 4b2)"). Same for c7d on c7 (mutant: past season with today's rules). If a verify fails, fix (lint cache must be per-WT: set GOLANGCI_LINT_CACHE, else ../ring-4-registry pollution).
2. Merge c5 (tip after commit) into integ2 (conflicts likely command.go/governance.go — use commandControls table pattern from c7; c5's new commands → table rows). Verify integ2.
3. **Sol at ~10% of 5h, resets ~16:45** → dispatch next phases to **Tom** (`RUN/tom-dispatch-wt.sh <phase> <brief>` with WT=), max 2 at once, until 16:45:
   - c8a-rollover: new WT `ring-4-c8` from integ2 (after step 2); brief par/c8a-rollover.md (+common.md → .full.md).
   - c4f: migrate start-league, role, undo forms into SideTray + test "no G2 by plain click" + Go preview blocker for a no-op change.
4. Later queue: c4 league ops (CM-051..054, M-093/095/137) + governed glyph; c5 CM-009 contract pricing; c6 CM-015 Now/Race; c7 CM-031 queue then results/standings (CM-007/016/033); chunks 9 (waivers/bids/DOT), 10 (draft).
5. Final: merge c4, c7, c8… into integ2 → verify/build → deploy Claude-OS → walk → ask Christopher (PR + squash-merge, one plain question).
**Policies now:** common.md = Sol/Tom run targeted tests + lint only; Claude runs make verify/build before every commit. Big scp to claudeos: use `scp -o ServerAliveInterval=5 -l 200000`. Integration note: c4's `checkCommissionerCommand` (preview) vs integ2's `retriedReceipt/checkAppend` → preview must call checkAppend.

## 14:40 update: NEXT ACTION 1 is DONE
- c5 verify 0 → mutant (bucket ignored) red → committed **1691249** on session/ring-4-c5.
- c7 verify 0 (first run hit "parallel golangci-lint is running": never run two lints at once) → mutant (incomplete shown as final) 7 red → committed **09f51a8** on session/ring-4-c7.
- Resume at NEXT ACTIONS step 2 (merge c5 1691249 into integ2), then step 3 (Tom: c8a-rollover, c4f).

## ===== 10-14 ~14:55 COMPACT POINT 2 — read this first =====
**Nothing is running** (no Sol/Tom builders, no verify). Christopher interrupted my verify run and called /compact-safe.

**In-progress, uncommitted: merge of session/ring-4-c5 @1691249 into session/ring-4-integ2** (WT `.worktrees/ring-4-integ2`, MERGE_HEAD=1691249, git status shows UU on command.go/ledger.go = conflicts RESOLVED in file but not yet `git add`ed). Resolutions done:
- command.go: RawArgs gains typed `Override *RawNativeOverride` (json "override"); Validate early-returns `decodeOverride(command, a)` for `overrideCommand(command)`; validateDetails refuses a.Override; validatePlacement adds `if overrideCommand → validateOverridePlacement()` after publicNowRule.
- native_override.go: `decodeOverride(command, a RawArgs)` (DeepEqual other args empty), `overrideCommand()` helper; old validateOverride + Args.Value JSON decode removed.
- native_projector.go: decodeOverride(…, record.Command.Args).
- ledger.go recordLabel: override case added to the switch.
- store/history/commissioner.go: kept integ2's checkAppend; added row {"native correction", CheckOverrideAppend} first.
- tests: native_override_test.go helper `overrideFromJSON` (strict `{"override":…}`); native_override_app_test.go builds Args.Override (strict decode) and uses domain.Money for the reused-key case.
- governance.go: Allow → `commandControls()` table (+ CM-034/038 rows) + `ledgerReader` (same refactor as c7 3d9cd42) — APPLIED in file, gofmt status unknown.
- Before the governance edit: internal tests all passed, root tests (Override|Cap|Reserve|Window|Gate|Native|Commissioner) passed, make build 0; verify4 failed only on gocyclo Allow (now refactored).

**NEXT ACTIONS, in order:**
1. In ring-4-integ2: `gofmt -l internal/`; run full verify with `GOLANGCI_LINT_CACHE=RUN/integ2/lint-cache` (never two lints at once). If 0: plant a mutant (e.g. drop CM-038 row from commandControls → reserve exception permission test red), restore, `git add -A`, commit the merge ("Integrate chunk 5 overrides and cap into the integration branch"; trailer).
2. Create WT `ring-4-c8` (branch session/ring-4-c8) from integ2 tip; dispatch **c8a-rollover** (par/c8a-rollover.md + common.md → .full.md). Sol resets ~16:45 (was ~10% at 14:00); before that use Tom: `WT=… RUN/tom-dispatch-wt.sh c8a-rollover par/c8a-rollover.full.md`.
3. Dispatch **c4f** on ring-4-c4 (tip d849273): migrate start-league/role/undo forms into SideTray, "no G2 by plain click" test, Go preview blocker for a no-op change (9 → 9 seen live). Write brief par/c4f-forms.md.
4. Then the queue from COMPACT POINT 1 step 4 (league ops+glyph; CM-009 pricing; CM-015 Now/Race; CM-031 queue; standings; chunks 9, 10), then final integration (merge c4 + c7 into integ2; c7 and integ2 both carry the commandControls refactor → trivial conflict), deploy, walk, ask Christopher.
**Branch tips now:** scoring-mfl 268f55f · integ2 83d0dff (+merge in progress) · c4 d849273 · c5 1691249 · c6 a2fda4f · c7 09f51a8.

## ===== 10-14 ~15:10 after compact 2 =====
- integ2 merge of c5 COMMITTED fe47853 (verify5 exit 0, mutant drop CM-038 row → TestNativeOverridePermissions red, restored).
- New worktree `.worktrees/ring-4-c8` branch session/ring-4-c8 @fe47853 (pnpm installed; run `make build` there before committing).
- Running on Tom (Sonnet high), check `stat -c %s <phase>/transcript.jsonl` + `<phase>/sentinel`:
  - c4f-forms (WT ring-4-c4): 3 forms into SideTray, no-G2-plain-click test, Go no-op blocker.
  - c7e-standings (WT ring-4-c7): native results, standings CM-007, override CM-033, real ResultFlipCheck.
  - c8a-rollover (WT ring-4-c8).
- Briefs ready, not dispatched (need a WT from integ2 tip; Sol after 16:45 reset): par/c5g-pricing.md (CM-009), par/c6d-week.md (CM-015 owner + command shape). Then c7f (CM-031 queue), c4g (league ops + glyph), chunk 9, chunk 10.
- worktrees ring-4-{c9,c10,c5g,c6d} created @fe47853, pnpm installed; briefs par/{c5g-pricing,c6d-week,c9a-waivers,c10a-draft}.full.md ready

## ===== 10-14 ~17:05 =====
- Tom runs c4f/c7e/c8a all died 15:10: "You've hit your session limit · resets 4:40pm" (Tom shares the Claude account; avoid >1 Tom).
- c6d-week (Sol) partial item 1 → reviewed, verify+build 0, my mutant (replay drops a frozen week) red → committed f135823 on session/ring-4-c6d.
- Sol running (16:41+): c4f-forms-2 (continuation, WT c4, prior.diff in c4f-forms-2/), c7e-standings-2 (continuation, WT c7), c8a-rollover (fresh; Tom's empty attempt moved to c8a-rollover-tom/), c5g-pricing (WT c5g), c6d-week-2 (WT c6d @f135823).
- Waiting: c9a-waivers (WT c9), c10a-draft (WT c10).
- 17:2x c8a part1 committed 7045814 (verify 0, mutant taxi-stays red). c8a-rollover-2 dispatched (Sol) with review fixes: cross-year redate bug, plain names, RFA/UFA route. c9a, c10a dispatched on Sol.
- 17:1x c4f DONE: committed 25f2045 on session/ring-4-c4 (verify+build 0, bundle 313,646; my mutant role no-op red). Integration note: CommandSafeguards() is a hand list → switch to integ2's commandControls() table at merge.
- common.md gained "Finish the phase" (Sol was self-stopping at ~10 min/100k tokens). Old copy par/common.prev-1715.md.
- c9a part 1 (pure policy) left uncommitted; c9a-waivers-2 continuation running on Sol in WT c9.
- Christopher 17:1x: "sol has 20% left and resets in 4hrs and 30min" (~21:40). NO new Sol dispatches. c4g-ops (league ops + glyph) on ONE Tom (Tom shares Claude account; never >1).
- Sol in flight: c7e-standings-2, c5g-pricing, c6d-week-2, c10a-draft, c8a-rollover-2, c9a-waivers-2.
- Season identity fix MOVED into par/c8a-rollover-3.md (rollover numbers seasons). Unseeded native league: resolver firstSeason()=ledgerSeason=1 (ordinal), but c6d's ScheduleSeasonReading anchors to creation year (needs NFL year for kickoffs). Unify: native seasons numbered by NFL year everywhere (creation.go ledgerSeason, rules_season.go firstSeason), then drop the fallback in schedule.go ScheduleSeasonReading. Test: rules reading and schedule agree on season for unseeded league.
- c6d ACCEPTED complete (item-4 kickoff-change proof impossible: native game has no kickoff; kickoff = NFL week). c7e committed 12027ad (verify 0; mutant tie-as-win → 3 standings tests red).
- 17:30 Sol: c8a-rollover-3 (WT c8, continues uncommitted c8a-2 work), c10a-draft-2 (WT c10). Waiting brief: c5g-pricing-2 (WT c5g, uncommitted part 1), c7f-lineups (WT c7, tip 12027ad).

## ===== 10-14 ~18:25 =====
- Sol OUT: all three 17:30 runs died "Codex error: The usage limit has been reached" (Christopher 17:15: 20% left, resets ~21:40).
- c6d-2 committed 3825b75→ amended 33a035d (removed .cache; .cache/ now in common info/exclude).
- integ2: merged c6d (ff to 33a035d) + c7 (12027ad) UNCOMMITTED merge in progress, all resolved+staged. Includes my changes:
  * schedule.edit now takes Games []ScheduleGame (atomic multi-game patch; single-game patch could never re-pair opponents). Tests updated + re-pair test.
  * ownPlacement() split out of validatePlacement (gocyclo).
  * Makefile GO_TEST_TIMEOUT := 25m (root race package was 377→489s, >600s with c5g/c9 → timeouts).
  * regenerated native-results-standings-wire.json (pairings changed by c6d home balance).
  Verify running → integ2/claude-verify2.log. On exit 0: mutant + commit "Integrate chunk 6 schedule and chunk 7 results/standings".
- Partial uncommitted trees (NOT committable now): c8 & c10 half-edited by killed runs (gofmt dirty); c9 & c5g: verify failed only by root timeout → rerun after Makefile timeout merges (or cherry-pick it).
- Briefs ready for Sol at reset: c8a-rollover-3 (+season-numbering fix), c10a-draft-2, c9a-waivers-2, c5g-pricing-2, c7f-lineups (+flip-warning wording fix; WT c7 tip 12027ad).
- Tom c4g-ops running (WT c4) since 17:17.
- Review notes for later: c7e flip warning wording; CommandSafeguards hand list → commandControls.

## 21:45 CT update
- **Christopher (after compact): "please do not use any more SOL as the rest of those tokens are need for automations."** Sol is OFF for the rest of ring 4. The Sol queue (c8a-rollover-3, c10a-draft-2, c9a-waivers-2, c5g-pricing-2) runs serially on Tom (ONE at a time), or I build it directly.
- Tom c7f-lineups had died 18:41 on the Claude five-hour limit having written nothing (old dir c7f-lineups.limit-1841). Relaunched 21:42 in WT ring-4-c7f.
- c4 merge into integ2: the nochange failure was a merge interaction. c5's resolver orders by RecordedAt, so withWaitingInForce's counted record (old instant) lost to an intervening immediate change. Fix: stamp the counted record at the newest later instant (rules.go). Commissioner pkg green; full verify4 running → integ2/verify4.log.
- 22:05 briefs written (par/*.md + .full.md): c7g-playoffs (CM-016, 046-049), c7h-statqueue (CM-031), c8b-corrections (CM-028/039, after 8a/9a/10a merge), c9b-bids (CM-013/036, after 9a), c9c-trades (CM-014/037, after 9a/9b/10a), c10b-draftroom (CM-043-045, after 10a).
- **Serial Tom queue (one at a time):** c7f (running) → c8a-rollover-3 (WT c8) → c10a-draft-2 (WT c10) → c9a-waivers-2 (WT c9) → c5g-pricing-2 (WT c5g) → c7g → c7h → c9b → c10b → c9c → c8b. Merge each into integ2 after review/verify/mutant; new WTs branch from integ2 tip and need frontend/dist copied.
- 22:20 c4 merge committed on integ2: 214790ee (verify4 0, mutant on waiting-instant fix red→green). WT ring-4-c7g created from 214790ee with dist.
- 22:45 Christopher: on the road early tomorrow; on 5h limits WAIT for cooldown, then continue. Goal unchanged.
- 23:05 c7f committed e0d80e5e on session/ring-4-c7f (Tom work + my Legacy 'lineups' window open all year, cited 'Ring 4 lineup ruling', test TestLegacyLineupsStayOpenAllLeagueYear, mutant red/green). Merge into integ2 staged (conflicts: history/commissioner.go threads LineupFacts into checkCommissionerCommand, preview gathers facts too; command.go strays union; SYSTEM_MAP). Bindings+wires regenerated. verify5 running → integ2/verify5.log. Tom c8a-rollover-3 running since 22:33 in WT c8.
- 23:10 verify5 EXIT=2: only TestNativeLineupSetThroughTheGateReadingAndRestart (assumed Legacy shuts lineups). Fixed: test removes the 'lineups' window first. verify6 running → integ2/verify6.log; if 0, commit the c7f merge.
- 02:45 10-15: limit reset. c7f merge committed eca63404 (verify6 0, root 527s). Tom c8a relaunched (prev cut 23:11 → c8a-rollover-3.limit-2311) with brief par/c8a-rollover-3b.full.md.
- 03:30 10-15: c8a committed cf9d9f50 on session/ring-4-c8 (Tom: 22 mutants; my mutant: pre-1920 season accepted → red). Tom c10a-draft-2 launched ~03:00 in WT c10.
  c8 merge into integ2 staged, UNCOMMITTED. Resolutions:
  - command.go: RawArgs gets Rollover and Season; strays union;
  - governance: SeasonRollover CM-027;
  - ledger Record: GeneratedSchedule plus Before;
  - schedule replay handles SeasonRollover;
  - history: appendRequest gains `records` (for FreezeSchedule), rollover prep after checkCommissionerCommand;
  - removed the ledgerSeason special cases in player_score/standings_reading (now `!seeded(records)`, new helper in rules_season.go), schedule.go ScheduleSeasonReading and native_results resultSchedule (firstSeason error);
  - tests given Season: 2026;
  - **StartLeagueForm gained a "First NFL season" input** (default this year; sends `season`; new test; mutant red).
  Wires and bindings regenerated. verify7 → integ2/verify7.log.
- 03:45 c8 merge committed on integ2 (verify9 0, root 603s).
- 04:20 10-15: c10a committed 694a3ba7 on session/ring-4-c10 (Tom; my mutant: commissioner door accepts draft.pick → red). Tom c9a-waivers-2 launched ~03:50 in WT c9.
  c10 merge into integ2 staged, UNCOMMITTED. Resolutions:
  - native gate unified on the lineup shape: Run(ctx, app, permit, input) (Receipt, error); runDraftPick decodes DraftPickInput from input (context smuggling removed), and draftPickAction() is in the registry next to lineupSetAction();
  - RecordDraftPick uses appendRequest;
  - draft helper proposed→withProposed (name clash with c4);
  - native_schedule leagueRecords(ctx);
  - validateDecoded handles draft;
  - SYSTEM_MAP keeps both sections.
  The draft's FinalStandings seam is still Missing: wiring it to the native standings is added to par/c10b-draftroom.md item 3b. verify10 → integ2/verify10.log.
- 04:30 c10 merge committed 6eced62d (verify11 0, root 806s). c9a committed 1cf595b2 (my mutant: dead money counts cut season → red). TODO in c9 merge: Legacy 'waivers' window all year opening cut + new action waiver_claim (OwnerWaiverAction claim was ufa_bid = May only); PR note: plain cuts all year, buyouts stay in cut_buyout window.
- 08:20 10-15: Tom c5g-pricing-2 relaunched 07:45 (prev cut 04:17 → .limit-0417, brief par/c5g-pricing-2b.full.md).
  c9 merge into integ2 staged, UNCOMMITTED. Resolutions:
  - one ExecuteCommissionerWithFacts (PrepareWaiverAppend after the shared checks, before the rollover prep);
  - waiver cut/claim registered as waiverAction(cmd) in the shared action shape (the registry patching is gone);
  - familyLabel helper; seed fields merged; SYSTEM_MAP gate paragraph rewritten;
  - **Legacy 'waivers' window 3/1 all year opens cut + waiver_claim (rulebook §8)**; OwnerWaiverAction claim → waiver_claim; leagueclock test extended (mutant red);
  - waiver tests: season; no-op rule sets removed (positionBounds {}, cap.enforcement hard, method reverse_standings).
  verify12 → integ2/verify12.log.
- 09:00 c9 merge committed on integ2 (verify13 0, root 1220s with Tom running).
- 08:55 10-15: c5g committed 16953244 on session/ring-4-c5g (my mutant: re-sign premium ignored → red). Tom c7g-playoffs launched 08:39 in WT c7g (base 61be6beb).
  c5g merge into integ2 staged, UNCOMMITTED. Resolutions:
  - the registry stays a package function holding lineup, draft, waiver and contract-start actions in the shared shape;
  - leagueRecordsIn dropped (leagueRecords(ctx));
  - ownPlacement uses ruleCommand (RulesSet + ContractActionSet);
  - ContractAction strays/validateDecoded handled.
  verify14 (regen + wire-write tests + verify) → integ2/verify14.log.
  **PR notes from c5g (need Christopher's ruling):**
  - tag only in a contract's last year;
  - a buyout charge lands wholly in the buyout season;
  - restructure money is spread evenly, with the remainder in the last year;
  - "years remaining" counts seasons after this one;
  - ExcludePlayoffContracts can't apply (playoff signings aren't marked).
  Also: usage limits are counted only for actions recorded in the app, and the contract owner action is a start gate only (execution not built).
- 14:10 10-15: c7g committed 9b9ad89a on session/ring-4-c7g (21 mutants; mine: winner judged without the home bonus → red).
  - Tom c7h-statqueue launched 14:00 in the new WT ring-4-c7h (from 61be6beb).
  - c5g merge on integ2 still UNCOMMITTED: verify17 timed out at 25m (no hang), so the Makefile GO_TEST_TIMEOUT is now 45m (PR note); verify18 running.
  - New brief par/c-perf.md: memoise RulesInForce per reading (c7g report: about 9 ms per player score, history race about 25 min). c8b gained 2b: rollover archives the decided champion.
  - PR notes: old snapshots have no conference map; bracket kickoff truth needs the held NFL schedule.
  - Queue after c7h: c9b → c10b → c9c → c8b → c-perf.
- c5g merge committed on integ2 (verify18 0).
- 14:50 c7g merge into integ2 staged (conflicts: command.go strays/validateDecoded union; SYSTEM_MAP rows). verify19 running. Queue reordered: c7h (running) → c-perf (root race now 32 min) → c9b → c10b → c9c → c8b.
- 15:10 c7h committed on session/ring-4-c7h (17 mutants; mine: correction only fills missing stats → red in history). Tom c9b-bids launched 14:53 in WT ring-4-c9b (from 0b707a57).
  PR notes from c7h, policy calls for Christopher:
  - (a) a week the newest file drops makes no item;
  - (b) the queue covers every directory-matched player, not only rostered ones;
  - (c) to reconsider a rejected item, undo the rejection first;
  - (d) the baseline is the first version the archive held.
- 17:50 c7g merge committed 1e1c1e70 (verify19 0; root 2039s). Tom c9b relaunched 17:42 (prev died 14:54 → .limit-1454). c7h merge staged on integ2 (conflicts: strays + Stat; checks table: facts.lineup for playoffs + statistic; app methods both). verify20 running.
- 19:20 c7h merge committed f709bc39 (verify20 0; root 2258s). WT ring-4-cperf created from f709bc39. Next Tom: c-perf after c9b.
- 19:50 c9b committed 09e18b38 on session/ring-4-c9b (33 mutants; mine: active limit off by one → red). Tom c-perf launched 19:36 in WT ring-4-cperf.
  - New brief par/c9b2-bids.md: bids.standFor rule + DueBids worker + playoff field + offseason rule + jobs list.
  - Queue: c-perf (running) → c9b2 → c10b → c9c → c8b.
  - PR notes from c9b (for Christopher):
    - withdraw allowed until decided;
    - a tied top is the commissioner's to settle;
    - no conditional drops;
    - the late mark is measured from the standing bid;
    - playoffs lift both the one-year limit and the tier years when playoffCapLifted;
    - RFA bids wait for tender recording.
- 20:50 verify22 (bids merge, staged UNCOMMITTED on integ2): the fixture-refresh step (full non-race root run) timed out at 45m under CPU contention with Tom c-perf (TestNativeContractActionSettingsGovernTheQuote 6m48s). Plan: wait for c-perf, merge it, then re-verify bids. Wire step: refresh only Wire-named tests + TestScheduleEditOwnerRevertRestart, then make verify.
- 21:00 c-perf committed b17f255c (standings 5x, bracket 7x faster; my mutant 'memo key ignores season' SURVIVED → added TestRulesMemoKeepsEachSeasonsOwnRulesAtOneInstant, now red/green). Tom c10b-draftroom launched 20:31 in WT ring-4-c10b (from f709bc39). verify23 (bids merge; wire step limited to Wire tests) running.
- 13:50 10-18 (back after the weekly limit; Tom c10b died 23:10 10-15 on "weekly limit, resets Oct 18 4am" → c10b-draftroom.limit-2310).
  - Root timing on 10-15 (2400s FAIL) was contention with Tom: alone, TestNativeContractActionCommandsReplayAndCompensate is 89s (900s in the contended run), TestNativeBidPhaseRules 5.9s (35s).
  - Profile: leaguerules.Catalogue rebuilt per record validation = 30% CPU → cached once (db0d78be; bid test 5.9→2.1s; mutant shared-list → red).
  - Bids merge committed 42b426a7; c-perf merged c150325a. verify25 (integ2/verify25.log) running on c150325a.
  - Tom c10b relaunched 13:44 (par/c10b-draftroom-c.full.md, continuation note) in WT ring-4-c10b.
  - Christopher: Sol has fresh tokens again. Sol c9c-trades dispatched 13:50 in NEW WT ring-4-c9c (from c150325a), brief par/c9c-trades-sol.full.md.
  - Queue: Tom c10b → Tom c9b2 (WT from integ2 tip after c10b merge) ; Sol c9c → c8b.
- 13:55 10-18: Christopher: two Sols at once is fine. Sol c8b-corrections dispatched 13:55 in NEW WT ring-4-c8b (from c150325a), brief par/c8b-corrections-sol.full.md. Running now: Tom c10b, Sol c9c, Sol c8b, verify25. Left after: c9b2 (next free builder, WT from integ2 tip after c10b merge).
- 14:05 10-18: /home hit 100% (564 MB free). Purged replayable caches (ring-4 finished phases + rings 0-3) and 14 merged clean worktrees → /home 60%, 177 GB free; logged in MANIFEST/MOVED/CT105 context. Sol c9c died 13:55 on ENOSPC (half-written temp files) → c9c-trades.enospc-1355; WT ring-4-c9c reset to c150325a; Sol c9c relaunched 14:02. Tom c10b, Sol c8b, verify25 survived.
- 14:16 10-18: verify25 on c150325a (bids + catalogue cache + c-perf) VERIFY_EXIT=0, 26 min; root race 1461s (was 2258s at f709bc39, before bids). go-build cache clear armed (waits for verify + no go process).
