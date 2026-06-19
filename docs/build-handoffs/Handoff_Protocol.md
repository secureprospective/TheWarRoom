# TheWarRoom — Handoff Protocol
**Version:** 1.0 — June 2026
**Status:** Mandatory. The last act of every build session.

The build is 38 sessions. Each starts in a cold 200k context window. The handoff prompt is the bridge across that boundary: it lets the next session start fully oriented without re-deriving everything. Uniform and routine is the whole point — same template, every session, so the build never depends on remembering how to hand off.

---

## The Routine

**Writing a handoff is the third close gate of every session** (see `Build_Tracker.md` legend). A session is not `[x]` complete until its handoff exists.

At session close, in order:

1. **Build green** — lint + tests pass.
2. **Functional verification** — Christopher uses the actual behavior.
3. **Write the handoff** for the *next* session using the template below.
4. **Save it** to `docs/build-handoffs/handoffs/NN-<block>.md` (e.g., `handoffs/02-B1.md`). One file per upcoming session, named by its tracker number.
5. **Update `Build_Tracker.md`** — mark the completed session `[x]`; mark the next `[~]` if starting immediately.

When Christopher clears the session and opens a fresh one, he pastes the saved handoff file as the opening prompt. That is the entire start-of-session ritual.

**Who writes it:** the AI, at close, from what actually happened that session — not from the plan's prediction of it. Christopher reviews before it's saved. Real learnings and real mistakes go in, not a restatement of the tracker.

---

## Recon Phase (Haiku fan-out) — default opening move, every remaining session

The 38-module build runs on a standing pattern: **before design or build, spin up a Haiku Explore subagent to fan out over this session's `READ FIRST` docs and return a consolidated inventory** — not file dumps, the conclusions. Haiku is the cheap, disposable **recon/fetch tier** (per the Technical pillar's Multi-Agent Orchestration; agy remains the whole-deliverable tier when back). It reads breadth so the cold context window doesn't have to.

Mandatory guardrails (learned B3 / B2b-Schema):
- **Recon gathers; judgment stays with Claude + Christopher.** A subagent never owns a design decision, a human-review gate (AD-16), or a locked-decision reversal.
- **Verify the load-bearing claims against source before acting.** A subagent — like any review agent — can't see what the driver can. Spot-check the items a decision rests on (file:line) before they move a build.
- Good recon jobs: input inventories, endpoint/shape probes, cross-doc constraint sweeps, "which positions reference X" boundary checks. Bad jobs: anything that writes code, scores, or locks a decision.

Reflected in the `== RECON ==` field of the template below — fill it every session.

---

## The Template

Copy verbatim. Fill every field. Mark unknowns explicitly — never guess.

```
HANDOFF — Session NN: <Block> — <one-line scope>
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Just completed: Session <NN-1> (<block>). Build green: <yes/no>. Branch merged: <yes/no>.
- Working tree: <clean / what's outstanding>.
- This session's branch: session/<short-description>

== READ FIRST ==
- docs/build-handoffs/Build_Tracker.md  (row for Session NN)
- Wireframe <#> in the Session 2 plan — the structure this session's code must match
- <position rubric / module spec / store spec for this block>
- <any SL decision or rubric section that governs this session>

== RECON (Haiku fan-out — run before design/build) ==
- Spin a Haiku Explore subagent over the READ FIRST docs; ask for: <the specific
  consolidated inventory this session needs — e.g., "every input field per position",
  "exact endpoint shape", "which SL decisions bite here">.
- Verify its load-bearing claims against source (file:line) before acting.
- Recon gathers only; design/gates/locked-decision reversals stay with Claude + Christopher.

== GATE CHECK (confirm before writing code) ==
- Upstream complete: <list — e.g., B5a, B2b-Off>. Verified: <yes/no>
- Open questions that block this session: <OQ-### or "none">. Resolved: <yes/no/N/A>
- If any gate is open: STOP. Resolve with Christopher before starting.

== WHAT THIS SESSION BUILDS ==
- Files: <exact paths, per the wireframe>
- Public surface: <the one/few exported functions/methods, per the wireframe>
- Layer: <1 / 2 / 3>. This session touches ONLY its layer.

== CONSTRAINTS ACTIVE THIS SESSION ==
- Standards: <250 line target / 400 cap; schema-validate input; parameterized SQL; no hardcoded secrets>
- Architectural: <the specific SL decisions, hard constraints, or layer rules that bite here —
  e.g., "RAS active for WR, do NOT force scoreRAS to 1.000 (AD-09)";
  "NGS anchor CB/S only"; "modules never write B3c except through B7">
- Anti-spaghetti: <wireframe rule this session must not break —
  e.g., "no position file imports another"; "no store imports another store">

== CARRIED FROM LAST SESSION ==
- Decisions made: <what got decided mid-build that affects what's next>
- Mistakes / learnings: <what went wrong, what to avoid — the reason this field exists>
- Open items carried: <CAL-### / SL-OQ-### / OQ-### relevant downstream>

== CLOSE GATE FOR THIS SESSION ==
- Build green: lint + tests.
- Functional check: <the specific behavior Christopher should exercise and what passing looks like>
- Handoff: write Session <NN+1>'s handoff before clearing.
```

---

## Worked Example (B0 → B1)

The shape, filled, so the first one isn't written cold:

```
HANDOFF — Session 2: B1 — MFL API Client (HTTP transport only)
Project: TheWarRoom · Stack: Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL

== WHERE WE ARE ==
- Just completed: Session 1 (B0). Build green: yes. Branch merged: yes.
- Working tree: clean. App struct + DI pattern + SQLite WAL lifecycle hooks + IPC ping-pong verified.
- This session's branch: session/b1-mfl-client

== READ FIRST ==
- Build_Tracker.md (row 2, B1)
- Wireframe 1A — Transport Client
- docs/data-layer/MFL_API_Specification.md and MFL_API_Reference.md (League ID 14432, host www47)

== GATE CHECK ==
- Upstream complete: B0. Verified: yes
- Open questions that block this session: none
- (RISK-001 rate limiting, RISK-002 host routing both live here)

== WHAT THIS SESSION BUILDS ==
- Files: internal/mfl/client.go · ratelimiter.go · router.go · client_test.go
- Public surface: ONE exported method — Do(req Request) (Response, error). Everything else unexported.
- Layer: 1. Transport only.

== CONSTRAINTS ACTIVE THIS SESSION ==
- Standards: <400 lines/file; no hardcoded host/IP (host fetched + stored, re-fetched per RISK-002).
- Architectural: NO application domain types. NO ingestion logic. Transport only.
- Anti-spaghetti: nothing here knows what an MFL roster IS — it moves bytes.

== CARRIED FROM LAST SESSION ==
- Decisions: DB connection pool is on the App struct, injected (B0). B1 does not open its own.
- Mistakes/learnings: <fill from B0>
- Open items carried: none

== CLOSE GATE FOR THIS SESSION ==
- Build green: golangci-lint + go test ./internal/mfl/...
- Functional check: Do() against the live MFL league endpoint returns a 200 with backoff on a forced 429.
- Handoff: write Session 3's handoff (B2 — MFL Data Ingestion) before clearing.
```

---

## Why This Field Set

- **Read-first + gate check** make a cold session safe — it can't start on an unmet dependency or an open OQ.
- **Constraints active** front-loads only the rules that bite *this* session, so the architecture is obeyed by default, not by memory.
- **Mistakes/learnings** is the one field that compounds: it's how 38 sessions get smarter instead of repeating the same error. It is required — "none" is a valid answer only when it's true.
- **Close gate** names what "done" means before work starts, so done isn't negotiated at the end.

---

*Built by: Christopher Campbell + Claude Opus 4.8 (Anthropic).*
