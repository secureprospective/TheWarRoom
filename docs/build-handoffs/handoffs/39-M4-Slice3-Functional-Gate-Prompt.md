HANDOFF PROMPT — Session 39: M4 slice-3 LIVE FUNCTIONAL GATE (Beelink GUI)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-18 · Purpose: the ONE deferred gate that blocks the slice-3 squash-merge to main.

========================================================================
PASTE-INTO-NEW-SESSION PROMPT (copy everything between the ===== rails)
========================================================================
=====
We are running the deferred M4 slice-3 LIVE FUNCTIONAL GATE for TheWarRoom on the Beelink GUI.
This is a functional-verification session — NOT a build session. Do not write feature code. If a
sub-gate fails, capture the exact behavior and stop; the fix is a later code session, not a re-quote.

Branch under test: session/m4-slice3-trade  ·  HEAD efda820 (ed3b2de + this gate doc)
Beelink Wails clone: /home/chris/opencode/TheWarRoom

Read these three, in order, before touching anything:
  1. TheWarRoom project CLAUDE.md header (full state + every gotcha: lint GOMEMLIMIT, Wails nil→null
     `?? []`, single-writer, D1/D4/D5).
  2. docs/build-handoffs/handoffs/38-M4-Slice3-Trade-Builder.md  (what the 4 builds are + GLM review).
  3. docs/transactions/M4_Slice3_Beelink_Functional_Gate.md  (the DETAILED gate script — this is the
     checklist you execute; every pass/fail criterion is in there).

Hygiene (memory [[reference_beelink_functional_gate]]): reset the clone to clean main BEFORE and
AFTER — force checkout + reset --hard + clean -fdx + prune session branches — and record which commit
the gate ran on. Then:
  export PATH=/usr/local/go/bin:/root/go/bin:$PATH
  git fetch origin && git checkout session/m4-slice3-trade && git rev-parse HEAD   # expect efda820
  GOMEMLIMIT=3000MiB GOGC=40 wails dev -tags webkit2_41
Use a FRESH db import for the franchise-name gate (a stale db shows id fallback — that is EXPECTED).

Execute the four gates from the gate-script doc and report PASS/FAIL per sub-gate with what you saw:
  GATE 1 TRADE — 2-team then 3-team atomic swap; BOTH rosters + BOTH caps move; guard rails blocked
    (no-destination, same-franchise, double-add, atomic-cart-after-fail); "from" label never lies.
  GATE 2 NAMES — real team names in rail + trade dropdowns + roster header; stale-db id-fallback OK.
  GATE 3 D6 COMMISSIONER — phase-advance changes legal ops; rollover shows the ENGINE reason when not
    in PLAYOFFS; cap-relief drops that franchise's cap; picker refreshes after a commit.
  GATE 4 DOLLAR BREAKDOWN — "Cap impact (pre-commit)" section: red charge / green credit, franchise
    name resolved, signed + $10k-snapped. LOAD-BEARING INVARIANT: quote == ledger every time (off-grid
    cap relief $3.006M must quote AND commit $3.01M). Expected no-line cases: Death $0, trade-only,
    phase-advance.

If ALL four pass:
  - squash-merge session/m4-slice3-trade → main
  - run the session-close merge steps (update project CLAUDE.md build state + a handoff, append to
    /root/.claude/backbone/context.md, commit the backbone, update memory [[project_thewarroom]])
  - reset the Beelink clone back to clean main
Standing authorization applies: merge/push after the gate passes without asking per-action
([[feedback_claude_runs_git_ops]]).

If ANY sub-gate fails: do NOT merge. Screenshot the modal / roster / cap view, note which op and which
criterion failed, bring it back. Report which commit the gate ran on either way.
=====

========================================================================
NOTES FOR THE OPERATOR (Christopher) — not part of the paste
========================================================================
- This gate covers all 4 slice-3 builds in one GUI pass (trade / names / commissioner / dollar
  breakdown). Everything runnable off the Beelink is already green; GLM 5.2 blind review is applied on
  all 4. The only thing left is USING it.
- The league-calendar feature is a SEPARATE branch (session/league-calendar, backend WIP, no UI) and
  is NOT part of this gate — do not touch it here.
- Screenshot gate is available if the GUI shows something that can't be diagnosed from files.
