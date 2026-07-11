HANDOFF — Session 34: §6 min-salary floor merged + the Transactions-panel lockup bug
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-11 (min-salary-floor session close)

== WHERE WE ARE ==
- B7c §6 MIN-SALARY FLOOR wired to real MFL draft-year experience — MERGED to main (squash `09f4575`).
  experience = season − draftYear (rookie=0), computed in-tx; Coordinator.ExecuteSign resolves the
  draft year via the players-DB Directory pre-tx (mirrors ExecuteTag/ExecuteExtension). Missing/
  sentinel/implausible draft data (MFL "0"/"1970", future, >30yr) → 0 exp → rookie $330k floor
  (Christopher's lenient ruling). The experienceKnown seam was removed; the floor is always enforced.
  GLM-5.2 blind review = 0 real defects (one coverage lead applied: plausibility-window boundary test).
- Automated gates all green (lint 0 / race / tsc+vite / GLM). Merged on gates+review per the §6
  deferred-gate precedent. THE LIVE FUNCTIONAL GATE WAS NOT RUN — see the blocker below.

== 🔴 THE BLOCKER (do this FIRST) — B7a Transactions panel locks the webview on tab click ==
Clicking the "B7a: Transactions" tab locks the whole UI (unclickable); a right-click → reload
recovers. Christopher confirmed it is REPRODUCIBLE on click and NOT a stale/compositor state — this
is almost certainly the real cause of the "APU compositor locks after first click" that got the §6
live gate deferred in the first place. It is a PRE-EXISTING §6 panel bug, independent of the
min-salary change (that change didn't touch the panel or its mount IPC).

Already ruled out this session:
  - Missing/mismatched Wails binding — GetFreeAgents / GetCurrentPhase bindings both exist.
  - Heavy list render — the live free-agent pool is EMPTY (0 player_status_events rows).
  - Blocking/deadlocking IPC — GetFreeAgents & GetCurrentPhase both use the read pool, no mutex, fast.
  - Concurrent mount IPC — the panel's useEffect fires refreshPhase()+refreshFreeAgents() together;
    sequencing them into one awaited chain did NOT fix it (tested live), so this is NOT the cause.
    That change was REVERTED — the panel (frontend/src/components/TransactionsPanel.tsx) is exactly
    as §6 shipped it, for a clean investigation.

NEXT DIAGNOSTIC STEPS (in order):
  1. On the Beelink, launch `wails dev -tags webkit2_41`, right-click → Inspect Element → Console,
     click the B7a Transactions tab, and READ the console error/stack. This is the missing signal —
     stop guessing the mechanism, get it from devtools. (The reload menu works, so Inspect should too.)
  2. If the console is silent, BISECT the §6 panel additions one at a time (relaunch between each):
     comment out the `void refreshFreeAgents()` mount call → then the SIGN control block → then the
     free-agent pool <ul>. Whichever removal stops the lockup localizes it.
  3. Check the wails dev TERMINAL for a Go-side panic/log when the tab is clicked.
Until this is fixed, NO §6 or min-salary functional gate can run — the Transactions dev panel is the
only surface that exercises them.

== THE DEFERRED FUNCTIONAL GATE (run once the lockup is fixed) ==
Combined gate staged in /root/paste.md (branch was session/b7c-min-salary-floor, now merged — re-stage
against main or a fresh branch):
  - Gate G (THIS build): pool a free agent (cut him), Sign at 0.2 ($200k) → REJECTED "below the §6
    minimum $330,000"; Sign at 0.7 ($700k) → OK (clears any floor). Proves the floor is live.
  - Gates A–F (deferred §6 click-through): cut→pool→sign, buyout lockout, retired-barred, phase
    window, rollover promotion, phantom-franchise reject.
Beelink clone /home/chris/opencode/TheWarRoom — reset to clean main before+after ([[reference_beelink_functional_gate]]).

== WHAT ELSE IS NEXT (after the lockup + gate) ==
- §6 follow-up: the first-class COMMISSIONER UFA CALENDAR — replace signingWindow()'s hardcoded
  phase set with a window that closes at Super Bowl kickoff / reopens on commissioner command
  (season_phases.meta). The SIGN handler + gate don't change (that's the seam).
- M-series UI (Build_Tracker row 30 M4 Transaction UI) — the real front-end beyond the dev panel.
  NOTE: whatever locks the dev panel may bite the real UI too — fixing the lockup first de-risks M4.

== LOCKED DECISIONS THAT CARRY ==
- Experience = season − draft_year; missing/sentinel/implausible → rookie floor (lenient). Plausibility
  window = [season−30, season]. draft_year is on all MFL players but "0"/"1970" are sentinels.
- Money = int64 cents, $10k grid. Ledger is KING. Single-writer law (AD-02), one WriteTx per op.
- ExecuteSign resolves the players-DB fact pre-tx; the season is authoritative only inside the tx.

== OPEN THE SESSION BY ==
Branch fresh off main: `git checkout main && git pull && git checkout -b session/<name>`.
Start on the panel lockup (devtools console) — it blocks everything downstream.
