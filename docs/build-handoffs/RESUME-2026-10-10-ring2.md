# RESUME: TheWarRoom ring 2, the decision layer (started 2026-10-10)

## ⏩ COMPACT-SAFE 2026-10-11 05:30 CDT — READ THIS FIRST (supersedes everything below)

**Goal (active /goal):** finish ring 2 (p2h2, p2i done; p2j–k, p2l, facts panel), Claude as head
brain, Sol builds, Claude-OS tests; merge to main only after Christopher's live click-through. RING 3
is Christopher + Tom's: do not start it. Christopher has a bigger job for Claude after ring 2.

**State:** committed on session/ring-2 (ring-0 worktree): through **abad777** (p2h2+p2h3 cut/buyout
plans, live-verified). **UNCOMMITTED in ring-0:** p2j + p2k (contract Acts) + p2k2 in flight.
Facts-schema panel DONE (4 seats; triage in scratchpad `panel-facts-run4-triage.md`, copy on Bee
`~/fleet/runs/warroom-ring2-2026-10-10/panel-facts-run4-triage.md`). Lab branch has resume commit
8674ae2 (merge into ring-2 before main).

**In flight:**
- p2k2 on Sol (pi), started 05:16, run dir `~/fleet/runs/warroom-ring2-2026-10-10/p2k2/`, done =
  `p2k2/sentinel`; alive = mtime of `p2k2/sessions/*.jsonl` (74 s old at 05:30). Watcher bzm24lydp
  (dies with compaction; restart the 60 s loop: sentinel → done, >900 s idle → stall). If stalled on
  Sol's limit: `p2k2/stderr.log`; finish on Tom (below). Recover reasoning: the session jsonl.
- Sol 5 h window ~20% left at 05:25, resets ~07:35 CDT.

**NEXT ACTIONS, IN ORDER**
1. When p2k2 lands: read REPORT; plant one mutant of my own; deploy (Bee: gzip → scp claudeos →
   deploy.sh); click Oluokun row What if… → Contract Acts…: menu anchored to row inside work column,
   "Contract move" panel at top with cap table before keeping, Keep plan button → Plan: Ready + link;
   post text "Adds 1 year: 2029 at $16,500,000"; Remove plan wording "Removed by you before it was
   handed off"; notes once. Remove any test plan after. Fix pass if needed (Tom if Sol is low).
2. Commit p2j+p2k+p2k2 in ring-0 (export PATH for hooks; commit message via FILE, scp'd — an
   apostrophe inside ssh '…' broke the last one).
3. Dispatch p2l on TOM: `ssh -f … 'cd ~/fleet/runs/warroom-ring2-2026-10-10 && nohup setsid
   ./tom-dispatch-wt.sh p2l ~/fleet/briefs/warroom-ring2-p2l-audit-spec-rev2.md > p2l.launch.log 2>&1
   < /dev/null &'`. Liveness = mtime of `p2l/transcript.jsonl`. Brief says HEAD holds p2a–p2k (true
   after step 2).
4. Review p2l (gate tables, spec rev 2, Facts_Schema_Panel doc); commit; merge session/ring-2-lab
   (8674ae2 resume) into session/ring-2.
5. PushNotification Christopher for the click-through: what to test + what passing looks like
   (pool/bids, what-if, cut/buyout plans, contract Acts, My moves). Merge to main only after he confirms.

**Open for Christopher:** ProBoards post format; registry May dates for tender (May 1–3) / re-sign
(May 10–15) are not in the rulebook (Acts say "Offseason only" without dates).

**Decisions this stretch:** D7 playoffs keep $12M/$24M tiers, only playoff teams bid (built in p2j);
D8 offers goldens stay in git until a backend host; D9 buyout what-if in-season labelled; D10 rules
are league subsets but NO rules-profile work now (commissioner's office later); D11 buyout → the
commissioner via ProBoards (built). Windows: Christopher's v1 ruling phase_gate.go:12 (all ops every
phase except BUYOUT/ROLLOVER/SIGN) → tag/extension/restructure any phase; tender/re-sign offseason.
Unknown history = "Not checked" note, not a block (Claude's ruling).

**Standing rules:** never make a real roster move on MFL (screenshot only, click nothing, close tab);
never --no-verify; never send to MFL; Tom = Opus medium, never Sonnet; panels non-Claude.

**Refuted / don't redo:** /fleet not showing Sol was the savvy-progress trigger (regex missed
dispatch-wt.sh; /fleet never polled) — FIXED both machines; Bee helper already scans run dirs.
Editing a running bash script in place corrupts it: swap via new file (mv).

## ⏩ UPDATE 2026-10-11 05:25 CDT — Sol near its 5 h limit (~20% left; resets ~07:35 CDT)
- p2k2 stays on Sol (already running). p2l goes to TOM: `~/fleet/runs/warroom-ring2-2026-10-10/
  tom-dispatch-wt.sh <phase> <brief>` (claude -p, claude-opus-5-5, --effort medium, bypass perms,
  stream-json transcript at <phase>/transcript.jsonl; liveness = its mtime; same sentinel). Smoke
  test passed 05:22 (Bash + Write headless; Tom's guardrail hook allows it).
- If p2k2 stalls on a Sol rate limit: check p2k2/stderr.log, then finish it on Tom with a brief
  "continue p2k2 from the current tree; read p2k2 brief + what's done".

## ⏩ UPDATE 2026-10-11 05:17 CDT
- p2k done 05:13 (verify/build 0, 597 FE, 12 mutants + Claude's tag-eligibility mutant killed,
  entry 321,741). Live 05:14: Oluokun offered Extend/Restructure only; Extend 1y $16.5M Ready with
  Not-checked notes; card team table 2029 +$16.5M; test plan removed. Defects → p2k2.
- p2k2 dispatched 05:16 (`~/fleet/briefs/warroom-ring2-p2k2-acts-polish.md`): menu anchored to row
  (was pinned bottom-right behind Inspector), "Contract move" panel at top with cap table before
  keeping, Keep plan button + feedback, per-Act post text, withdraw wording, notes once, spacing.
  Watcher bzm24lydp. p2j+p2k+p2k2 still UNCOMMITTED in ring-0.
- /fleet fix: savvy-progress register.tsx (PI_LAUNCH matches dispatch-wt.sh; /fleet polls Bee
  itself) on CT105 and Tom, byte-identical, tests 25/25 and file-tree 24/24 both machines.
  dispatch-wt.sh also symlinks sessions into ~/.pi/agent/sessions (harmless; helper scans runs).

## ⏩ UPDATE 2026-10-11 04:52 CDT
- p2h2+p2h3 commit is abad777 (amended message; d2ca6ca was pre-amend).
- p2j done 04:46 (verify/build 0, 596 FE, 10 mutants, entry 321,741) but every Act Blocked; NOT
  committed. Live 04:47: RFA tender offered on a 2028 UFA, menu off-screen, "Rule unclear" text.
- p2k dispatched 04:48 (`~/fleet/briefs/warroom-ring2-p2k-contract-acts-ready.md`): windows from
  Christopher's v1 ruling in phase_gate.go:12 (tag/ext/restructure every phase; tender/re-sign
  offseason), only applicable Acts, unknown history = "Not checked" note not a block, team cap
  before/after via what-if coordinator, menu in viewport, plain labels. Watcher bh8yl7djg.
- p2l brief staged: `~/fleet/briefs/warroom-ring2-p2l-audit-spec-rev2.md` (triage + resume copied to
  the run dir for Sol). After p2k: review, live check, commit p2j+p2k, dispatch p2l, then notify
  Christopher (PushNotification) for the click-through.

## ⏩ UPDATE 2026-10-11 04:25 CDT
- Facts-schema panel COMPLETE: engineer seat (Sol) 14/14 quotes, verdict (c). Triage folded in
  `panel-facts-run4-triage.md` (scratchpad; copy into repo docs at p2l): FS1 refined (manifest-keyed
  cache, stale-on-fail), new FS4 input manifest (archive.go:87 has no cutoff), FS2 gate table, D3 →
  one append-only league_event table with an MFL-until-cutover authority policy (spec rev 2 only).
- Tom: overflow / Sol-down builder, Opus medium (memory saved). RING 3 belongs to Christopher + Tom (2026-10-11): do not start any ring 3 work; Christopher has a bigger job for Claude after ring 2.

## ⏩ UPDATE 2026-10-11 04:10 CDT
- p2h2+p2h3 COMMITTED d2ca6ca on session/ring-2 (live-checked 04:06: buttons, Remove plan, I didn't
  drop him, no Rebase on Blocked, no ids). Claude-OS test plans withdrawn; what-if Reset.
- p2j dispatched 04:08 (brief has D7 playoff-bid fix, D11 buyout channel text, withdrawn-card fold);
  watcher bzxc1jor2. Engineer panel seat (Sol) running: `~/fleet/runs/panel-brief-20261011T090833Z-3776547`
  (watcher b4jc559vf); fold into panel-facts-run4-triage.
- Then: p2j review/live/commit → p2k scope (check resume build order) → p2l → notify Christopher.

## ⏩ UPDATE 2026-10-11 03:52 CDT
- p2h2 done (Sol, sentinel 03:46): verify/build 0, 583 FE tests, entry 321,652, 3 mutants killed +
  Claude's (buyout term floor) killed. Live on Claude-OS 03:47–03:50: plan Ready, card, confirmation,
  Cancel, Confirm opened O=257 (tab closed), Not yet done, buyout Blocked. UNCOMMITTED in ring-0.
- p2h3 (fix pass: buttons, cut withdraw, no Rebase on Blocked, wording, no ids in post, 4 px shift)
  dispatched 03:50, brief `~/fleet/briefs/warroom-ring2-p2h3-plan-polish.md`; watcher bc47issd8.
  Next: review, redeploy, click the six fixes, commit p2h2+p2h3 on session/ring-2, then p2j.
- Live Claude-OS app DB now holds a Banks cut plan (Not yet done) and a Benton buyout plan
  (Blocked): withdraw both with p2h3's new buttons during its check.

## ⏩ UPDATE 2026-10-11 03:25 CDT (supersedes everything below; read this first)
**Christopher 10-11 ~02:20: "no more using sonnet, that ruined our context windows and delayed us."**
Sol is the only builder. Sonnet built p2i/p2i2 (headless `claude -p`, CLAUDE_RC_REMOTE=1) — don't repeat.

**Committed on session/ring-2 (ring-0 worktree):** `8e5151e` p2h + p2h1 (per-cell dead cap PW5;
What-if panel in Contracts and cap, one "What if…" menu per row, Change column, local baseline);
`52142ac` merge of session/ring-2-lab `7aff3f5` (p2i + p2i2: War Room › Valuations and pool, bid
plans, bid_facts, rule checks, ProBoards post copy, "I didn't post it"). Merge: registry CSV kept
C-06.6 built + C-06.7 in pool; models.ts regenerated. verify 578/578, entry 321,552/321,818.
Live-verified on Claude-OS: pool 804–808 FAs, §6 minimums match rulebook, points 1.800 (2y $1.5M),
clipboard post text exact; what-if cut Banks $700,000, buyout Benton $1,630,000, cut Benton
$2,280,000, Undo, Reset all correct by hand. Claude's own mutant (restructured rate ignored) killed.
Not live-tested: the stale "MFL changed" banner (tests only). Claude-OS runs the merged build.

**IN FLIGHT: p2h2 on Sol** (cut/buyout plans from the what-if, PW2 invalidation, PW4 Landed, cut
confirmation, buyout channel; plus carry-over polish: "Expires / status" header wraps, ~40 px gap under
the empty What-if sentence, menu Acts in monospace). Brief `~/fleet/briefs/warroom-ring2-p2h2-cut-plans.md`.
Run dir `~/fleet/runs/warroom-ring2-2026-10-10/p2h2/`, worktree ring-0 (UNSTAGED Sol edits: do not
commit there until done). Started 02:52; stream hung 03:05; resumed 03:21 with
`~/fleet/runs/warroom-ring2-2026-10-10/resume-phase.sh p2h2` (kills wrapper then pi, reruns the SAME
session id; writes sentinel itself). Alive = mtime of `p2h2/sessions/*.jsonl` (idle >900 s = hung →
run resume-phase.sh again via `ssh -f`). Done = `p2h2/sentinel`. CT105 watcher bjsl6ht0u.
On done: read REPORT.md, review checks (offseason, 2 buyouts, Landed predicate), plant own mutant,
deploy, click Make-a-plan → confirmation → hand-off → My moves card; fix pass if needed; commit.

**NEXT ACTIONS, in order**
1. Finish p2h2 as above; commit on session/ring-2.
2. Dispatch p2j (brief staged `~/fleet/briefs/warroom-ring2-p2j-contract-acts.md`: RFA tender,
   re-sign, franchise tag, extension, restructure; ProBoards hand-off D5; Landed by contract cells
   PW4; "Rule unclear" disables instead of inventing). Live check, polish pass, commit.
3. Run the panel's engineer seat (Sol) on the facts schema: `cd ~/fleet/briefs && SEATS=engineer
   ssh -f … ~/fleet/bin/panel-dispatch.sh ~/fleet/briefs/panel-ring2-facts-schema.md`, then fold it into
   the run-4 triage below and commit.
4. p2l: full parity + UI audit, spec revision 2 (brief not written yet).
5. Christopher's click-through, then merge session/ring-2 → main (he confirms first).

**Facts-schema panel run 4 (02:56–03:02, `~/fleet/runs/panel-panel-ring2-facts-schema-20261011T075559Z-2950223/`)**
Seats Kimi K3 (6/10 quotes ok), GLM 5.3 FLASH standing in (13/13 ok), Nemotron Ultra (0/11 by the
checker: formatting only, spot-checked). `~/fleet/bin/panel-dispatch.sh` gained `MECHANIC_MODEL`
(default z-ai/glm-5.3; backup /tmp/panel-dispatch.bak on Bee). GLM 5.3 still 429 at 02:32.
- D1 split (Kimi a facts.db; GLM-flash c compile; Nemotron against c). **Triage: (c) stands.**
  Nemotron's evidence refutes it (checks.py:139 proves over in-memory dicts, not SQL). Compiled facts
  are a pure function of archived bodies: a facts.db can be materialized later at zero migration cost.
- **New prerequisite s0 (verified):** archive is `TheWarRoom/league-archive/raw` (1,720 files, 110 MB,
  2013–2026); history.db raw_archive holds only app-fetched bodies. s0 ingests it through `Record`
  (archive.go:30) as a registered source.
- D2: all three say both proofs (row diff vs frozen lab.json + goldens on compiled input; frozen pair
  a permanent fixture). D3: imported rows carry provenance; app-authored events stay in their own
  append-only stores (PW3); an override names what it supersedes; the compiler merges.

**Christopher answered the open questions (2026-10-11 03:40 CDT), now D7–D10 below.** Still open:
ProBoards post format (he checks later). **Action from D7:** p2i lifted the $12M/$24M tiers in the
playoffs: wrong. Fix in a small p2i3 (or fold into p2j): tiers stay; in the playoffs only playoff
teams may bid (a Blocked reason for a non-playoff team). D10: no rules-profile work in ring 2 (stay on plan).

**Environment lessons:** launch Bee jobs with `ssh -f` (plain ssh + nohup & hangs the call);
commits in a worktree need `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH` for the
pre-commit hooks; background Claude subagents die on the confirm-action hook's "defer".

## ⏩ UPDATE 2026-10-10 20:00 CDT (supersedes the in-flight list below)
- DONE + live-verified + committed on session/ring-2 (ring-0 worktree): p2g+p2g2 Contracts and cap
  `878b87a`; resume branch merged `0cb6952`; p2f+p2f2 Offers history (lab `2b6cb6e`) merged
  `1f5281d` (verify 525/525, entry 321,212/321,818); frame border-box fix `7484f14`.
- Bottom "scrollbar strip" is NOT the app: it is Claude-OS's autohidden xfce4-panel dock
  (306x49 at 487,751). Not a defect.
- Sol budget: ~25% at 19:20, resets ~22:25. p2h dispatch is DURABLE ON BEE: pid of
  `~/fleet/runs/warroom-ring2-2026-10-10/p2h-at-2230.sh` (nohup, started 20:02) sleeps to 22:30,
  then runs `dispatch-wt.sh p2h ~/fleet/briefs/warroom-ring2-p2h-whatif.md` once (skips if
  p2h/started exists). Check: `pgrep -af p2h-at-2230` before 22:30; after, `cat p2h/started` and
  transcript mtime `stat -c %Y p2h/sessions/*.jsonl` (idle > 600 s = stalled). Done = p2h/sentinel.
  Local watcher bst19sn93 (CT105 session shell, started 20:25) polls p2h: prints "p2h done",
  "STALL p2h", or "p2h DID NOT START" (after 22:40); if it is gone after compaction, start one. On done: read p2h/REPORT.md,
  review Part 1 math myself + plant own mutant (snap once vs per cell), make build, deploy,
  click the what-if on Claude-OS (cut, buyout, Undo, Reset, stale banner), fix pass if needed,
  commit on session/ring-2, then write + dispatch p2h2 (plans from what-if, PW2/PW4), then p2i.
- Briefs staged: p2h (dead-cap PW5 fix + what-if in Contracts, no plans), p2i (pool + bid plans,
  PW3). Still to write: p2h2 (cut/buyout plans from the what-if, PW2/PW4 Landed), after p2h lands.
- Flagged to Christopher: offers goldens contain his raw offer list (same data already in the
  committed lab.json.gz); default buyout what-if allowed in-season labelled "Offseason only".

## ⏩ RESUME HERE (compact-safe 2026-10-10 19:20 CDT)
Goal (Christopher /goal): finish ring 2; Claude is head brain, Sol builds (GLM 5.3 the backup),
the panel answers durable questions, Claude tests on Claude-OS. Excellence is the standard.

**In flight (dispatch-wt.sh; check = transcript mtime; done = `sentinel` exists):**
| Run | Builder | Worktree / branch | Started | Brief |
|---|---|---|---|---|
| p2f Trade Floor › Offers (L-30–L-32) | Sol medium | `.worktrees/ring-2-lab` / session/ring-2-lab | 19:00 | `~/fleet/briefs/warroom-ring2-p2f-offers.md` |
| p2g HQ › Contracts and cap (M-078, C-06.5, M-031, M-025, cap summary, Inspector-collapse shell fix) | Sol medium | `.worktrees/ring-0` / session/ring-2 | 19:02 | `~/fleet/briefs/warroom-ring2-p2g-contracts.md` |
Check: `ssh -n -i /root/.ssh/beelink chris@192.168.1.191 'R=/home/chris/fleet/runs/warroom-ring2-2026-10-10; for p in p2f p2g; do echo $p $(( $(date +%s) - $(stat -c %Y $R/$p/sessions/*.jsonl) ))s; cat $R/$p/sentinel 2>/dev/null; done'`
Both active at 19:16 (17/18 files changed). Kept dirs: `p2g-glm429` (GLM 429 at start, no edits),
`p2a3` (DeepSeek timed out after 2h17m; its partial work is `p2a3/deepseek-partial.patch`).

**Committed (all verified live on Claude-OS unless said):**
- session/ring-2 (`.worktrees/ring-0`): `4028695` p2a3 IR activation (Sol p2a3b on DeepSeek's
  partial) ← `d754680` merge of ring-2-lab ← `1a41757` p2e4–p2e5 ← `603dc23` p2e3 ← `2a42b7a` p2e
  ← `020ee39` resume ← `91cdb94` merge (eligibility, Market, Draft). make verify 503 frontend tests;
  entry 321,091 / ceiling 321,818 (not raised).
- session/ring-2-lab fast-forwarded to `4028695` before p2f.
- make verify now regenerates Wails bindings and runs `tsc --noEmit` (catches embed-and-shadow
  duplicates that only `wails build` used to catch). Always ALSO run `make build` before deploy.

**Next actions, in order:**
1. When p2f lands: review (whose offers the data holds and the L-32 header says so; parity max
   error; plant one mutant of my own in the offers port; D4/W1/How counted), `make build`, deploy,
   click Trade Floor › Trade desk and offers › Offers at 1280×800 with the Inspector open (no
   sideways page scroll, no mid-word breaks, labels). Commit on ring-2-lab.
2. When p2g lands: review (integer cents end to end, inferred years marked, no computed dead
   cap, M-031 labelled "MFL adjustments", M-025 honest), mutant of my own, `make build`, deploy,
   click Franchise HQ › Contracts and cap + a player's contract years in the Inspector + close the
   Inspector (work column must widen). Commit on session/ring-2.
3. Merge ring-2-lab into ring-2 (stash nothing needed if ring-0 is clean; watch shared files:
   registry CSV, endpoints.gen.ts — regenerate with `cd frontend && node scripts/gen-endpoints.mjs`,
   models.ts, SYSTEM_MAP.md, TargetApp.tsx), make verify + make build, deploy, recheck both.
4. Next builds: p2h cut/buyout what-if + PW5 dead-cap fix (35% of EACH remaining year's own
   salary, snapped to $10k; PW1/PW2 what-if design); p2i pool/bids (PW3); p2j–k contract Acts with
   ProBoards hand-off (D5); p2l audit + spec revision 2; then the facts-schema panel (D1).
5. Christopher's own click-through before any merge to main.

**Decisions (binding):** D5 ProBoards hand-off for option/tender/re-sign/extension/restructure/
tag/FA bids (MFL has no owner pages for them). D6 what-if keeps its baseline until Reset; unknown
eligibility leaves the Act available. Captures (owner-only, never commit):
`~/fleet/runs/warroom-ring2-2026-10-10/mfl-captures/`.

**Lessons tonight:** builders cannot see the screen; every layout phase needs Claude's live
check at 1280×800 with the Inspector open (~700 px work column). Counterparties took 4 rounds:
fixed-width tier grid, 30-row trend table, 11-column team table (squeezing broke words: never
`break-all`/`anywhere`; use a narrow/wide container query, nowrap numbers, let prose wrap).
Goldens cannot reach rules that live in Lab view code (the season flag in tradelog.js:169):
add edge tests (`internal/lab/team_flag_test.go`).

**Builder availability (19:20):** Sol working (two parallel runs). GLM nvidia: still 429 at 19:00.
DeepSeek: timed out on long runs (2h17m, "Request timed out"). Kimi: not a builder. OpenRouter,
opencode, zen: 402.

**Refuted / do not retest:** Kimi as builder; DeepSeek for long builds; injuries via www47 (use
api.myfantasyleague.com without L); `pkill -f` over ssh (kills own session: use PIDs); CT105 clock
is UTC (use Bee's date). Remote python/heredoc edits inside ssh '…' quoting break: write the
script locally, scp it, run it.

**Claude-OS:** black screen + "Power Manager … AccessDenied" popup FIXED 17:45/18:00: xfce4-power-
manager idled into suspend (14 min) and suspend.target is masked; inactivity-on-ac/battery set
to 0 (Never, xfpm 4.20 slider 0–60); X blanking off now and at login
(`~/.config/autostart/no-blank.desktop`: `xset s off s noblank -dpms`). App relaunch after reboot:
`bash ~/warroom-ring1/launch.sh`. passt recovery (Bee): `LIBVIRT_DEFAULT_URI=qemu:///session virsh
detach-device Claude-OS /tmp/claudeos-if.xml --live` then `attach-device`. Deploy gzipped:
`gzip -1 -c build/bin/thewarroom > /tmp/twr.gz; scp … claudeos:warroom-ring1/thewarroom.new.gz;
ssh claudeos "cd ~/warroom-ring1 && gunzip -f thewarroom.new.gz && chmod +x thewarroom.new && bash
deploy.sh <old-label>"`. Current label: p2a3b build (deployed 18:58, ring-0 at 4028695).
Screenshots: scratchpad `shot.sh <name>` (virsh screenshot via Bee). Clicks: `ssh claudeos
'DISPLAY=:0 xdotool mousemove X Y click 1'`. Command bar (95,153), first result (367,193); Trade
Floor (80,312) › Counterparties tab (510,86); HQ My moves tab (616,86). Pearsall has a Ready
activation plan in My moves from the live test (not submitted). Close MFL tabs after any capture.
Known pre-ring-2 shell defect: a horizontal scrollbar strip at the bottom of every page.

## What we are doing
Ring 2 of `UI_Target_Roadmap_2026-10.md`: the Lab's analytics ported to Go (War Room › Market,
Trade Floor › Draft, Counterparties, Offers), Franchise HQ › Contracts and cap (what-if, dead cap,
window Acts), the free-agent pool and bid tracking, plus IR/taxi eligibility gating (spec rev 1
§8 Q5). Gate: numbers match the Lab's tests; "How this is counted" on every analytic surface;
spec revision 2.

- Branch `session/ring-2` off main fd6871d, worktree `~/work/TheWarRoom/.worktrees/ring-0` (Beelink).
- Sol (gpt-6.1-sol, medium) builds on Bee; Claude (head brain) briefs, reviews, commits, runs the
  Claude-OS live gates; the expert panel (`~/fleet/bin/panel-dispatch.sh`) rules on durable decisions.
- Run folder `~/fleet/runs/warroom-ring2-2026-10-10/` (`dispatch.sh <phase> <brief>`; transcripts in
  `<phase>/sessions/`, which `/fleet` now finds).

## Pass 1 (p0 survey), 2026-10-10
- Sol's survey: `~/fleet/runs/warroom-ring2-2026-10-10/p0/REPORT.md` (18 KB; full evidence in
  `report-draft.md`). Sol hit its usage limit just before writing it; Claude promoted the staged
  final (`report-final-staged.md`, complete, all 7 sections) to REPORT.md.
- Lab tests 14/14 pass (`p0/lab-tests.log`); `make verify` exit 0 on the branch baseline (`p0/verify.log`).
- What exists in Go today: roster terms, contracts, dead-cap ledger, salary adjustments, a cut and
  buyout preview, IR/taxi Acts (built, not Landed). No Lab port, no bid ledger, no eligibility data.
- What the Lab is: a Python compiler turns the archived facts database into `lab.json`; the JS
  engine (`engine.js`, `draftlab.js`, `teams.js`, about 40 functions) computes from it.
- Findings worth keeping:
  - MFL `injuries.json` gives status (IR, IR-R, IR-PUP, Out…), not an eligibility verdict;
    `players_DETAILS` has no canIR/canDemote. Eligibility must come from verified rules, not guesses.
  - Taxi age rule conflicts (verified by Claude): the captured MFL setting says under 3 years
    (`Ring1_Gap_Closure.md:600–605`); the written rulebook says 3 or fewer (`Official_Rulebook.md:18–22`).
  - DEFECT (not a question; the rule is Christopher-locked 2026-07-04, `Math_Rules_Reference.md:81–87`:
    35% of EACH remaining paid year's own cell, summed, $10k snap): `deadcap.Charge`
    (`internal/transactions/deadcap/deadcap.go:25–36`) charges the CURRENT cap salary × years, so
    any contract whose cells differ by year is wrong. Fix in p2h; verified 10-10 by Claude.
  - Waivers run on ProBoards per the spec; no MFL bid state was found in the archive.
  - `salaryAdjustments` are MFL adjustments, not proven itemised dead cap.

## Proposed build order (Sol's p2a–p2l, condensed; each one Sol run)
| Run | What | Depends on |
|---|---|---|
| p2a | IR/taxi eligibility: eligible / ineligible / unknown with reason, source and time; unknown blocks Ready | Q3 |
| p2b–p2c | Market: going rate, buy/sell lists, asset profile, pick clock | Q1, panel |
| p2d | Draft: timing, plays-like-pick, hit rate, draft-or-buy | p2b |
| p2e | Counterparties | p2b |
| p2f | Offers (owner-scoped history) | p2b |
| p2g | Contracts and cap in HQ, in cents | none |
| p2h | Cut / buyout what-if and plans; fix the dead-cap defect | p2g, Q5 |
| p2i | Free-agent pool and bid plans | p2g, Q2 |
| p2j–p2k | Option / tender / re-sign Acts and their lifecycle in My moves | p2g–p2i, Q4 |
| p2l | Full parity and UI audit, spec revision 2 | all |
If Christopher wants TheWarRoom to own its data (Q1 b), five compiler runs (s1–s5) come before p2b.

## Questions for Christopher (pass 1)
1. The Lab. On 10-06 (T399) you said the Lab is your own research tool and its work never mixes
   into TheWarRoom branches; the roadmap says ring 2 ports it. Which? (a) TheWarRoom rewrites the
   Lab's math in Go and reads a frozen, fingerprinted copy of the Lab's compiled data;
   (b) as (a), then TheWarRoom also gets its own Go pipeline from the MFL archive (5 more runs,
   fully stand-alone). Either way the Lab's code is only a reference for tests and nothing of it is
   merged. Recommendation: (a) now, (b) before the stand-alone app.
2. Bids: track only your own bid and claim plans locally until we have verified how waivers run
   (ProBoards); never show other teams' bids. Recommendation: yes.
3. Taxi age: under 3 years (MFL setting) or 3 or fewer (rulebook)? MFL is what enforces it.
4. Option, tender, re-sign: I capture the MFL owner pages on Claude-OS (no submits) instead of
   asking you to describe them. Recommendation: yes.
5. What-if: keep the numbers you started from until you press Reset, even if MFL data refreshes.
   Recommendation: yes.

## Decisions (Christopher, 2026-10-10)
- **D1 The Lab:** TheWarRoom rewrites the Lab's math in Go now, reading a frozen, fingerprinted copy
  of the Lab's compiled data; before the stand-alone app it gets its own pipeline from the MFL
  archive (compiler runs s1–s5). No Lab code is merged; the Lab is only the reference the tests check.
- **D2 Taxi age:** check MFL's rule, under 3 years of experience. The rulebook's "3 or fewer" is a
  rulebook mismatch for Christopher to fix there.
- **D3 Bids:** "We just need the framework, eventually ProBoards and MFL will be eliminated." Build
  the bid and claim framework as TheWarRoom's own system, designed to become the authority in the
  stand-alone app; for now it holds plans and honest status, nothing invented about other teams.
- **D4 Spread tiers:** show the range bar itself; the solid / rough / thin grade stays internal,
  never on screen (Christopher, 2026-10-10).
- **D7 Playoff bids (2026-10-11):** the playoffs do NOT lift the $12M/$24M year tiers; the rule stays
  and only narrows: in the playoffs only playoff teams have bid access.
- **D8 Offers goldens:** may stay in git until a formal backend server host holds the data.
- **D9 Buyout what-if in-season:** allowed for now, labelled "Offseason only".
- **D10 Rules framing (Christopher, 2026-10-11, narrowed same night):** "these rules are subsets, not
  hard coding into the ui. The next league could change the rules." Then: don't go down the rabbit
  hole now. Premise: what happens on ProBoards is what MFL can't do (bidding, the extra rule
  frameworks that make 32 teams possible). No rules-profile work in ring 2; the commissioner's office
  mapping (later) will settle how the front end handles rules. Keep W1 (rules live in Go, not React).
- **D11 Buyout channel (2026-10-11):** a buyout goes to the commissioner via ProBoards; the app is the
  UI that prepares it (copy text), like bids. **Standing rule:** never make a real roster move on MFL;
  MFL is reference and links only.
- Defaults taken (Q4, Q5, his to overturn): Claude captures the option/tender/re-sign MFL pages on
  Claude-OS, no submits; a what-if keeps its starting numbers until Reset.

## Expert panel
### Run 1 (10-10): parity method and the wire shape. Seats: Kimi (visionary, rerun at medium after a
22-min stall), GLM (mechanic), Nemotron (devil); Sol's seat skipped (out of tokens).
Runs: `~/fleet/runs/panel-panel-ring2-parity-contract-20261010T155210Z-304800/` (GLM, Nemotron) and
`...-20261010T161439Z-556587/` (Kimi). Quotes: GLM 16/17 ok, Kimi 5/5, Nemotron unparsed (format;
Claude spot-checked its `gauss` quote, real).

**Claude's triage (recommendation; durable, so Christopher may overrule):**
- P1 Parity: adopt JS golden fixtures (all three agree). Generated offline by a pinned Node command
  from the frozen `lab.json`; goldens stamped with BOTH the lab.json hash and the engine.js hash
  (GLM F3, Kimi F4); Node is never a runtime dependency.
- P2 Port the JS algorithms faithfully: same Gauss elimination with partial pivoting, same
  summation order. Rejected: Nemotron's gonum QR (would diverge from the reference by design).
- P3 Tolerance: exact for IDs, counts, enums, nulls, membership, cents; floats within
  1e-9 + 1e-9×|want| (GLM; enum margins measured: nearest non-identity 1.6e-3). Record the max
  observed error per function; Kimi's bit-exact target is the aim, not the gate. Rejected:
  Nemotron's looser 1e-7/1e-6 tiers.
- P4 The generator fails on any non-identity decision margin under 1e-6 and names the row (GLM F2).
- P5 Required mutants: a threshold flip (8→7 delivered records; 80→81 trades) must turn a golden
  cell value↔null (GLM F5); a seeded Go bug (one ridge term dropped) must fail parity (Kimi F3).
- P6 The Lab's "measured 2026-10-06" comments become coded assertions in TheWarRoom's own parity
  tests (Nemotron F5), never edits to the Lab (D1).
- P7 Goldens also carry the compiler's prices so Go re-checks the compiler↔JS anchor (GLM F4,
  Nemotron F2) at the JS suite's own 0.01.
- W1 Wire shape: one generic shell extending the existing `Sourced[T]`/Provenance seam, with a
  typed payload per surface (GLM F6; Kimi and Nemotron converge on shell + typed body). Shell
  carries status (ok / thin sample / failed / not wired), absence reason, count and the threshold
  it was judged against, baseline, cutoff (`archiveThrough`), method, provenance (input hash,
  build stamp), and "How this is counted" text written in Go. React does no math.
- Open for Christopher (Q6): the Lab's spread tiers (solid / rough / thin). Nemotron and Kimi:
  a confidence score, banned (CLAUDE.md:86). GLM: a plain grading of the shown range, allowed.
  Claude's recommendation: show the range itself, keep the tier internal.

### Run 2 (10-10 14:37): what-if vs plans, bid/claim framework (D3), Landed for contract Acts.
GLM `~/fleet/runs/panel-panel-ring2-plans-whatif-20261010T193726Z-2548439/mechanic.out` (13/13 quotes
ok); Nemotron same dir `devil.out` (0/24 quotes resolvable: leads only; agrees on the core); Kimi
(medium) returned nothing (rejected, thin). Sol's seat skipped (building p2c).

**Claude's triage (adopted, follows D3 and the locked rules):**
- PW1 The what-if stays a separate simulation in `whatif.db`; plans (envelopes) are drafted FROM it,
  never a new envelope kind; it never writes the mirror. Baseline marker = the mirror's own stamps
  (max `contract_years.last_updated`, newest `dead_cap_ledger` row, rosters revision) recorded at
  start; when live is newer, a banner says "MFL changed since this started — Reset"; nothing moves
  until Reset (Q5 default). No content hashing.
- PW2 A Ready plan is invalidated through the existing Invalidate → Stale → Rebase path
  (`internal/envelope/envelope.go` ~244) on: a newer contract change for its player, a new dead-cap
  row for the franchise, its window closing, or the player leaving the roster.
- PW3 Bids and claims are envelopes using the states already reserved (BidPending, WaiverPending,
  `envelope.go` ~247, ~299) plus one side table `bid_facts` keyed by the envelope's correlation id
  (kind, player, amount in cents, years, bid points, RFA tender, deadline, evidence URL/note). The
  rulebook limits (30 active bids, increments, snipe rule) are check-time code, not schema. Outbid
  = terminal "awarded off-app", not an error. Nothing structured is stored about other franchises.
  The stand-alone auction later adds franchise and auction ids to the same table.
- PW4 Landed: cut and buyout by the player leaving the roster in the rosters export; option,
  tender, re-sign by contract cells (a new Contracts observation source for the watcher). A dead
  cap that differs from the plan is an Invalidate ("$X planned vs $Y now"), not a failure.
- PW5 Dead-cap defect fix: charge per cell, summing 35% (50% restructured) of each paid year's
  own cell for the years AFTER the cut season, one half-up rounding, one $10k snap at the end; 0
  if claimed or in the final year. (GLM wrote "years ≥ season"; Claude corrected it: the locked
  rule charges nothing on a final-year cut, so the cut year's own cell is not counted.)
- PW6 Seeded future years are flat-filled (an inference); the what-if says so in its
  "How this is counted" note until the app holds real cells.


### Run 3 (10-10 20:03): facts schema before compiler runs s1–s5. PROVISIONAL (one valid seat).
`~/fleet/runs/panel-panel-ring2-facts-schema-20261011T010349Z-1898530/`. GLM 429 (no answer);
Kimi produced degenerate text (rejected); Sol skipped (budget for p2h). Nemotron devil.out: quote
checker failed on its format; Claude spot-checked: `internal/lab/decode.go:21` Data struct,
`history.go:63/77/88` raw_archive/loads/observations, `archive.go:64` Body — real (line numbers
drifted; Lab schema has 29 tables, not 23).
**Claude's triage (recommendation; rerun GLM + Sol seats before s1 starts):**
- FS1 Facts are not a second store: s1–s5 compile `history.db` raw_archive bodies straight into the
  columnar structs the engines already read (`internal/lab` Data), cached by input hash; the Lab's
  `checks.py` proofs (facts vs MFL's own figures) become compile-time assertions that fail the build.
  Rejected: porting the Lab's 29-table `mfl.db` (a derived cache with sync risk; the engines never
  query it relationally). Wrong if an engine needs keyed random access the slices can't serve.
- FS2 Proof: for the frozen cutoff (2026-10-06), diff the compiled columns against decoded
  lab.json row by row (exact ids/counts/enums/cents; floats per P3), then run the existing goldens
  on the compiled input. When the archive advances, keep the frozen pair as a regression fixture
  and regenerate goldens with the new SHA in SOURCE.md.
- FS3 Rejected Nemotron's "bids as observations rows": MFL-derived facts stay rebuildable (history),
  TheWarRoom-authored events stay in envelopes + side tables (PW3 bid_facts), never mixed; the
  stand-alone authority later adds its own event tables, not rows in observations.
- Facts schema: provisional triage above (run 3); rerun with GLM + Sol seats before s1.

## Next actions
1. (done: panel run 1 triaged above)
2. (done: D1–D3)
3. p2a (eligibility) brief written: `~/fleet/briefs/warroom-ring2-p2a-eligibility.md`. Answer key
   captured by Claude on Claude-OS 11:27 CDT: `~/fleet/runs/warroom-ring2-2026-10-10/mfl-O18.html`
   (IR page, 55 rows, 38 "Can not be deactivated") and `mfl-O98.html` (taxi, 69 rows, 28 "Cannot be
   demoted"); owner-only pages, never in git. Defaults taken: unknown eligibility leaves the Act
   available with "Not checked; MFL will decide"; ineligible disables it with the reason.
   Dispatch when Sol resets (about 12:20 CDT).
4. (done: D4)

## Progress log (newest last)
- 11:48 p2a (eligibility) dispatched to GLM 5.3 high (Sol out; Christopher: "use GLM as backup").
  `dispatch.sh` now takes PROVIDER/MODEL/THINK; `dispatch-wt.sh` also takes WT (a second worktree).
- 12:22 p2b (Lab Market engine) to Sol in worktree `.worktrees/ring-2-lab`, branch
  `session/ring-2-lab` (off session/ring-2) so two builders never share files.
- 14:17 p2b2 review fixes (goldens gzipped 6.5 MB→600 KB; decode once; app build stamp; stable
  display-order ties allowed under 1e-12). Claude moved the cache onto the App (no package
  global), fixed a wrapcheck, verified (own mutant Ridge 0.25→0.26 fails parity; regeneration
  byte-identical; make verify 0) and committed **fafe73a** on session/ring-2-lab.
- 14:23 p2a done: rule 86/86 vs MFL (the brief's "124" was Claude's miscount). Live gate on
  Claude-OS 14:25: taxi correct for Perine; **IR stale**: injuries is fetched from the league host
  (www47), which MFL refuses ("must go to api.myfantasyleague.com"); the app silently fell back to
  the 10-05 archive. Also: fixture +19,682 lines and not reproducible by cmd/fixtures.
- 14:27 p2a2 fixes to GLM; 14:28 p2c (Market screens) to Sol in ring-2-lab.
- 14:37 panel run 2 (what-if vs plans; bid/claim framework D3; Landed for contract Acts):
  GLM + Nemotron `~/fleet/runs/panel-panel-ring2-plans-whatif-20261010T193726Z-2548439/`,
  Kimi (medium) `...-20261010T193729Z-2548650/`. Sol's seat skipped (busy building).
- Waiters: always use full /home/chris paths in ssh tests; `~` expands on CT105 (bug found 10-10).

## Notes
- Sol's usage limit: the p0 run hit it at 10:49 CDT; Christopher said it resets in about 1h30.
- The Lab lives only on `session/league-history-pwa` (main checkout); read it, never change it.
