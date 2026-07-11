HANDOFF — Session 35: §6 commissioner UFA calendar (signing-window gate) merged
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-11 (UFA-calendar session close)

== WHERE WE ARE ==
- B7c §6 COMMISSIONER UFA CALENDAR — MERGED to main (squash `3e5066c`). The Free_Agency_Design
  Q4 forward seam is now built: a commissioner-discretion signing window that gates SIGN on top of
  its phase floor (OFFSEASON+REGULAR_SEASON). The window:
    * rides the append-only season_phases.meta slot as a from==to SUB-PHASE directive
      ({"ufa_window":"open"|"closed"}) — NOT a phase transition, so CurrentPhase / season derivation
      are untouched;
    * defaults OPEN when no directive exists → v1 behavior preserved EXACTLY (old rows have meta='',
      excluded by the LIKE filter);
    * PERSISTS across ordinary phase transitions and rollovers (which write meta='') until the
      commissioner toggles it again ("closes at Super Bowl, stays closed until reopened");
    * rejects a redundant toggle (no-silent-no-op).
- New op SET_SIGNING_WINDOW (KindSetSigningWindow, legal in EVERY phase). State primitives
  SigningWindowClosed (read, committed pool) + AppendSigningWindow (write) on SeasonScope; new file
  internal/store/state/signing_window.go. gatePhase layers the window override onto SIGN only.
  IPC: buildRequest SET_SIGNING_WINDOW case + windowOpen DTO field in transactions_app.go.
- GLM-5.2 blind review = 0 correctness defects. Applied all 3 cheap advisories (load-bearing
  one-request-per-tx comment on gatePhase; LIKE-scan forward-seam note; fake error uses the word).
- Gates: `GOMEMLIMIT=3000MiB GOGC=40 make lint` 0 issues · `go test -race` green · tsc+vite clean.
- README updated: UFA calendar moved from roadmap → Feature Showcase; min-salary floor added to the
  §6 row.

== 🔴 STILL THE BLOCKER (unchanged from handoff 34) — B7a Transactions panel locks the webview ==
Clicking the "B7a: Transactions" tab locks the whole UI (right-click→reload recovers). PRE-EXISTING
§6 panel bug, independent of both the min-salary floor AND this UFA-calendar change (neither touched
the panel — TransactionsPanel.tsx is still exactly as §6 shipped it). This blocks EVERY §6 /
min-salary / UFA-window functional gate, because the dev panel is the only surface that exercises them.
NEXT DIAGNOSTIC (do FIRST, per handoff 34): on the Beelink `wails dev -tags webkit2_41`, right-click →
Inspect → Console, click the tab, READ the error/stack. If silent, bisect the §6 panel additions
(comment out the mount refreshFreeAgents() → SIGN control block → free-agent <ul>).

== DEFERRED FUNCTIONAL GATES (run once the lockup is fixed) — now THREE stacked ==
1. Gate G — min-salary floor (handoff 34): sign at $200k → REJECTED "below §6 minimum $330,000";
   sign at $700k → OK.
2. Gates A–F — §6 free-agency click-through (handoff 33): cut→pool→sign, buyout lockout, retired-
   barred, phase window, rollover promotion, phantom-franchise reject.
3. Gate H (THIS build) — UFA calendar: SET_SIGNING_WINDOW close → a SIGN that clears the floor is
   REJECTED "signing window is closed by the commissioner"; reopen → same SIGN lands. (Needs a dev-
   panel control for SET_SIGNING_WINDOW — NOT yet built; add it WITH the lockup fix, same panel.)
Beelink clone /home/chris/opencode/TheWarRoom — reset to clean main before+after ([[reference_beelink_functional_gate]]).

== WHAT ELSE IS NEXT ==
- Frontend: a commissioner control for SET_SIGNING_WINDOW (open/close toggle) — deferred WITH the
  panel lockup so the lockup investigation keeps a pristine panel. The backend op is callable now.
- M-series UI (Build_Tracker row 30 M4 Transaction UI) — the real front-end beyond the dev panel.

== LOCKED DECISIONS THAT CARRY ==
- The signing window is a from==to season_phases.meta directive; latest ufa_window directive wins;
  default OPEN; persists across transitions/rollovers until toggled. One-request-per-WriteTx is
  load-bearing for the committed-read gate (documented on gatePhase).
- Experience = season − draft_year → §6 floor; missing/sentinel → rookie $330k (handoff 34).
- Money = int64 cents, $10k grid. Ledger is KING. Single-writer law (AD-02), one WriteTx per op.

== OPEN THE SESSION BY ==
Branch fresh off main: `git checkout main && git pull && git checkout -b session/<name>`.
Start on the panel lockup (devtools console) — it blocks all three deferred functional gates.
