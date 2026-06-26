HANDOFF — Session 10: B3b — League Rulebook (first Layer-2 config store)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- The ENTIRE B2b-Fetch arc is COMPLETE + merged to main (2026-06-26): offense (520eb0a),
  defense (677102a), kicker + module-close (this arc's squash). Every scouting/production
  Layer-1 fetcher exists, RAW + gsis-keyed, live-verified. crosswalk.Map now exposes
  Lookup (MFL->gsis), GSISForESPN, PFRMap (pfr->gsis). Shared seams: extcsv (CSV + gz +
  stream), cfbd.go (client + long-format helpers + EmitDropAmbiguous), the injected-resolver
  pattern. ingestion (Layer 1) is DONE.
- B1/B2/B3 done: mfl transport client, ingestion boundary helpers + fetchers, and
  internal/domain + internal/normalize (Layer-1 typed records, type system LOCKED).
- THIS SESSION opens a NEW layer: internal/store (Layer 2 CONFIG stores). B3b is the first.
  Branch fresh off main: git checkout main && git pull && git checkout -b session/b3b-league-rulebook.
  Confirm scope with Christopher first.

== RECON PHASE (standing build pattern — do this FIRST) ==
Haiku Explore fan-out over the READ-FIRST docs, then Claude verifies load-bearing claims
against source before any code:
  - docs/data-layer/MFL_Scoring_Rules_Decode.md  (the decoded scoring config — the heart of
    the rulebook's content)
  - docs/data-layer/MFL_API_Specification.md     (the `league` export = league settings /
    roster rules / scoring config; also `salaries`; host/JSON rules; league ID 14432)
  - docs/data-layer/MFL_API_Reference.md
  - docs/backend/Backend_Architecture.md         (the store layer + App-struct config boundary)
  - /root/.claude/plans/very-good-now-i-replicated-feigenbaum.md → "Wireframe 2 — Layer 2
    Config Stores" (the EXACT store file shape — clone it)
  - /root/.claude/plans/session-3-audit-build-sequencing.md → AD-05 and AD-21 (below)
  - docs/roadmap/Roadmap_and_Open_Questions.md   (the rulebook OQs that gate values — below)

== WHAT THIS SESSION IS (Build_Tracker row 9) ==
B3b — League Rulebook: an MFL-sourced, SQLite-backed Layer-2 CONFIG STORE for the league's
RULES (scoring config, roster rules, salary cap AMOUNT, tag/RFA rules, etc.) with a
delta-override mechanism. **Pure data access — NO rule LOGIC.** It STORES and SERVES rule
values; it does not compute dead cap, tag prices, or scores (that is B7/engine work that
READS from here). It is the first of three Layer-2 stores (B3b rulebook / B3c state / B4
calibration) and SETS THE STORE TEMPLATE every later store clones.

== THE STORE TEMPLATE (Wireframe 2 — clone this shape exactly) ==
Files:  internal/store/rulebook/rulebook.go · internal/store/rulebook/types.go ·
        internal/store/rulebook/rulebook_test.go
Public surface:  Get<X>(key) (Type, error) · Initialize(source) error · Reload() error
Write surface:   Set<X>(key, value) error  (admin-only — see AD-05)
types.go:        ALL data types for THIS store only; types do not bleed into rulebook.go.
File order:      struct → constructor → read methods → write methods → initialize/reload.
HARD RULES:      all SQL parameterized (NEVER concatenated — Codex/coding-standards);
                 NO store imports another store; the store validates its own Set;
                 schema-validated at the App-struct config boundary; file < 400 lines (filelen).
B3b specifics:   MFL-sourced DEFAULTS; **delta overrides stored as SEPARATE records** (a
                 commissioner override is a distinct row layered over the MFL default, not an
                 in-place mutation of it — so Reload() can re-pull MFL defaults without
                 clobbering overrides); Reload() re-fetches from the MFL `league` endpoint.

== LOCKED DECISIONS (do not relitigate) ==
- AD-05 (write path): B3b config writes are an ADMIN-ONLY path, schema-validated at the
  App-struct boundary, each store validating its own Set — **NEVER routed through B7.** B7 is
  the sole writer of LEAGUE STATE (B3c) only, not of config. (M9b Commissioner Rules UI is the
  eventual governance surface over B3b — not this session.)
- AD-21 (cap label split): the salary cap **AMOUNT** is league config → lives in B3b,
  MFL-sourced (DECISION-009 governs the amount). The cap-tier **PERCENTAGES**
  (Cold/Neutral/Hot thresholds, Layer 5) are engine CALIBRATION → B4, NOT here. Do not put
  tier thresholds in the rulebook. (OQ-006 cap-tier calibration is a B4 concern.)
- Layer law: B3b is Layer 2 — no Layer-1 logic enters here; it consumes the MFL `league`
  export (and the existing mfl transport client), it does not re-implement transport.

== OPEN QUESTIONS THIS SESSION TOUCHES (the MFL config is the AUTHORITY on values) ==
- OQ-002 / missed-FG value: the rulebook prose says "-3 / -1"; the MFL scoring config is the
  authority on which applies and when. Read it FROM the league export, store it; do not hardcode.
- Long-play discrete bonuses (20+/40+ yard events): confirm whether MFL exports them as
  separate scoring rules or embeds them — store what MFL actually returns.
- Franchise-tag price rule (avg of top-5 salaries at position) + RFA window — these are RULE
  VALUES to STORE; the CALCULATION timing is OQ-008/OQ-009 (B7, not here). Store the rule
  parameters; don't compute.
Confirm with Christopher which rule fields are in-scope for B3b v1.0 vs deferred.

== SHARED SEAMS AVAILABLE (reuse, don't re-invent) ==
- internal/mfl — the transport client (Do/DiscoverHost, rate-limit/backoff). The `league`
  export is a league-specific call → DiscoverHost first (B2 pattern).
- internal/ingestion — MFLList decoder, CheckAPIError (MFL returns 200 with an error
  envelope — guard it), ValidatePlayerID. The `league` payload's scoring config decode is new.
- internal/domain + internal/normalize — typed records (if a rule references positions/ids).
- internal/db — the SQLite/WAL handle pattern (parameterized queries).

== CONSTRAINTS ACTIVE ==
- No work on main; branch session/b3b-league-rulebook. Never git --no-verify.
- CT105 build: GOMEMLIMIT=1500MiB GOGC=20 make lint (warm cache: go build ./... first); then
  go test -race ./... . Go 1.26.4 at /usr/local/go/bin. Live MFL is env-gated (TWR_LIVE_MFL=1),
  opt-in, never in the default suite. Beelink is the Wails/GUI machine.
- Every custom gate proven by a planted failure (M3). Shared logic extracted not copy-pasted
  (M17). Review gate: Gemini 3.1 (agy out; re-auth via pct enter 104 if it returns) — TRIAGE
  every finding against source (its blind-review false-positive rate is high; the B2b-Fetch
  arc-close review produced 6 findings, all false positives).

== CLOSE GATE ==
- Build: make lint 0 + go test -race ./... green + (if it hits MFL) env-gated live PASS
  showing the real league rules decode + a delta override layering correctly over a default.
- A planted bad override / bad rule value is seen to fail the store's own validation.
- The store template is clean enough that B3c/B4 can clone it (it sets the pattern).
- Squash-merge to main after Christopher confirms; write the next handoff (B3c or B4 per the
  tracker dependency order) before clearing.
