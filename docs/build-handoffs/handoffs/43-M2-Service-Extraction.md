HANDOFF — Session 43: M2 Service Extraction — thin the m2_app.go adapter (SLIM_MAP §6.1)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

Origin: full-codebase slim review (docs/reviews/warroom-fullmap-2026-07/SLIM_MAP.md §6.1).
Christopher's ruling 2026-07-20: EXTRACT — mirror M1's thin-adapter pattern. Follow-up
cleanup, NOT an original Build_Tracker row.

== WHERE WE ARE ==
- Just completed: SLIM_MAP tiers 1–3 low-risk items on session/slim-cleanup (pushed):
  dead-code/dedup, M4 stage() try/catch, TransactionsPanel deleted, scouting docs.
- Working tree: assume clean off main after slim-cleanup merges (confirm at start).
- This session's branch: session/m2-service-extraction (fresh off main).

== READ FIRST ==
- docs/reviews/warroom-fullmap-2026-07/SLIM_MAP.md §1 (map), §6.1 (this ruling)
- m2_app.go (the adapter to thin) and m1_app.go (the pattern to mirror)
- internal/rankings/ (M1's service package — the delegation target model)
- internal/powerrankings/ (M2's math-only package — where orchestration should land)
- docs/modules/M2_Power_Rankings_Design.md (the merge-time design + review trail)

== RECON (Haiku fan-out — run before design/build) ==
- Ask for: every function currently in m2_app.go with its exact signature + what it
  touches (fetcher call? store read? pure transform?), and the mirror-image list from
  m1_app.go + internal/rankings, so the extraction target is a proven 1:1 shape.
- Verify against source (file:line) which m2_app.go funcs are pure (move freely) vs which
  touch the IPC/fetcher boundary (STAY in the adapter — that's the legit adapter job).

== GATE CHECK (confirm before writing code) ==
- Upstream complete: M2 slice-1 (merged 686a574), M1 (internal/rankings). Verified: <y/n>
- Open questions that block: none known. The blend MATH is not changing — only WHERE the
  orchestration lives. If the extraction would change any number, STOP.

== WHAT THIS SESSION BUILDS ==
- Move orchestration (parseStanding, buildBlendInputs, aggregateScouting, buildPowerRows,
  resolveAggMode, clampWeight, + the rest of the ~12) OUT of m2_app.go into a service —
  either internal/powerrankings (extend it past pure math) or a new internal/m2service,
  matching however internal/rankings relates to m1_app.go. Pick the mirror, don't invent.
- m2_app.go becomes a thin adapter: validate → route → format, like m1_app.go.
- Public surface: one/few exported service entrypoints the adapter calls (mirror M1).
- Layer: 2/3 boundary (the app adapter + a service package). No engine/store law changes.

== CONSTRAINTS ACTIVE THIS SESSION ==
- Standards: 400-line file cap (filelen); parameterized SQL; no globals (gochecknoglobals);
  typed IPC boundary (ifaceguard — no any/interface{}); error-wrap %w.
- Architectural: this is a MECHANICAL move — the M2 numbers must be byte-identical before
  and after. DUP7 (parseStanding vs leaguestandings.Validate) folds into the service here.
- Anti-spaghetti: adapter holds NO business logic post-extraction; the service imports down
  only (depguard). Do not make powerrankings import the fetcher if it currently doesn't —
  pass data in.

== CARRIED FROM LAST SESSION ==
- Decisions: M2 math (robust-z median/MAD scouting blend + all-play z, weighted, min-max
  display) is LOCKED and untouched — see M2 design doc. This session moves code, not math.
- Mistakes/learnings: the SLIM_MAP botched-commit lesson — when splitting/staging, verify
  the committed tree (git show), a bad git-add pathspec silently stages nothing.
- Open items carried: M2 slice-2 (weeklyResults cols) still deferred, unrelated.

== CLOSE GATE FOR THIS SESSION ==
- Build green: make lint 0 + go test -race ./... green.
- GLM 5.2 blind review of the extraction diff (leads-not-findings, triage vs source).
- Functional check (Beelink live gate): M2 Power Rankings board loads live (32 franchises),
  weight slider re-orders on release, sum/top-N toggle changes ranks, sorting works with no
  NaN cells — IDENTICAL behavior to pre-extraction. Reset Wails clone before/after.
- Handoff: write the next session's handoff before clearing.
