HANDOFF — Session 33: Free Agency live gate (deferred) + what's next
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL
Written: 2026-07-11 (B7c §6 Free Agency v1 session close)

== WHERE WE ARE ==
- B7c §6 FREE AGENCY v1 (pool + record-a-signing) MERGED to main (squash `4b31662`).
  The free-agent pool, the SIGN op, UFA promotion on rollover, and the enforced release
  status chokepoint are all in. NOT a live auction (deferred as multi-user). Panel-gated
  design + GLM blind code review done; see docs/transactions/Free_Agency_*.md.
- ALL of §8–§14 plus §6 free agency are now built. The transaction rulebook is substantially
  complete; §7 RFA and the real §6 auction are the remaining rulebook gaps (both multi-user).

== THE ONE UNFINISHED GATE (do this first) ==
The LIVE GUI functional gate was DEFERRED — the Beelink WebKitGTK/APU compositor locks up
after the first click. This is NOT a code problem:
  - `thewarroom -probe` against the live Beelink DB passed EVERY startup step (db.Open,
    params, rulebook, state.Initialize, transactions, output) — backend healthy, no
    corruption, no hang.
  - The `FreeAgents` SQL runs in 3ms. IPC is not hanging.
  - It's the known APU-compositor bad-state that a REBOOT cleared for the §11 and §14 gates.
TO RUN IT: reboot the Beelink, then PLAIN `wails dev -tags webkit2_41` (the config that passed
§14 — drop the WEBKIT_DISABLE_* env vars; those force software rendering that stalls on the
first repaint). Gate steps are staged in /root/paste.md (6 sub-gates A–F): cut→pool→sign,
buyout lockout, retired-barred, phase window, rollover promotion, phantom-franchise reject.
Reset the Beelink clone to clean main BEFORE and AFTER ([[reference_beelink_functional_gate]]).

== WHAT TO BUILD NEXT (pick per Build_Tracker.md + Task_list.md) ==
1. Free-agency FOLLOW-UPS (small, unblock now that the pool exists):
   - Wire the §6 min-salary floor: it's built + tested (`MinSalaryFloor`) but SKIPPED in v1
     because there's no years-of-experience source. Add experience to the player facts (MFL
     DETAILS or a derived table), then pass (years, true) from Sign's apply / a Coordinator
     pre-tx resolve — the seam is already there (experienceKnown param).
   - The first-class COMMISSIONER UFA CALENDAR (Christopher's forward seam): a window that
     CLOSES at Super Bowl kickoff and reopens on commissioner command (finer than the 3 phases;
     use season_phases.meta). Replace `signingWindow()`'s hardcoded phase set with the toggled
     window — the SIGN handler and gate do NOT change (that's the whole point of the seam).
2. M-series UI (per Build_Tracker.md) — the real front-end for the engine + transactions,
   beyond the dev TransactionsPanel.

== LOCKED DECISIONS THAT CARRY (reuse, don't re-litigate) ==
- Money = int64 cents, FLAT $10k grid on every figure. CapUsed = Σ(cells)+Σ(dead_cap)−Σ(cap_relief),
  floored at 0, roster-joined. Ledger is KING.
- Single-writer law (AD-02): one spanning WriteTx per op. TxWriter is at its 10-member
  interfacebloat cap; new surface groups into an embedded interface (StatusWriter is the pool's).
- player_status_events is the pool's source of truth (append-only, double-immutable). The pool =
  latest event FREE_AGENT AND off-roster. ReleasePlayer is the enforced status chokepoint —
  any new removal path MUST pass the right status.
- The §12 buyout lockout is DERIVED from the dead_cap row (deadcap.BuyoutReason, league_year ≥
  season), zero new writes. Don't add a lockout column unless dead_cap ever gets purged.
- SIGN clears ALL prior contract_years cells (logged) before laying a fresh term — required
  because ReleasePlayer leaves cells behind (the boundary-year collision).

== BUILD GATES (every commit) ==
- `GOMEMLIMIT=3000MiB GOGC=40 make lint` → 0 (ifaceguard + filelen 400 + golangci-lint; the
  linter OOMs at 1500MiB now — use 3000MiB); `go test -race ./...` green; tsc+vite clean;
  depguard proven (plant an import, see it deny).
- GLM-5.2 blind CODE review over SSH to the Beelink ([[reference_glm_review_over_ssh]]):
  `opencode run -m zai/glm-5.2 "$(cat prompt)"` detached with a sentinel; leads-not-findings,
  triage vs source. Panel design gate first for any durable/standard-setting decision.
- Functional gate live on the Beelink (reset the clone before+after). If it black-screens or
  locks: run `thewarroom -probe` (XDG_CONFIG_HOME=copy) FIRST to prove backend vs rendering —
  the compositor bad-state needs a reboot, not a code fix.

== OPEN THE SESSION BY ==
Branching fresh off main: `git checkout main && git pull && git checkout -b session/<name>`.
If closing the deferred live gate: no branch needed — just reboot the Beelink and run paste.md.
