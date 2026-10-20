# HANDOFF — G0: Author the Go Overlay (coding standards)

**Saved:** 2026-06-10, Session 3 close. **Type:** Pre-build gate (not a numbered build session).
**Paste this as the opening prompt of a fresh session when ready to start the Go overlay.**

This is a **PLANNING + AUTHORING** session. Talk through the open questions first; decide; then write. No guessing. B0 (and therefore the entire 38-session build) is blocked until this overlay merges.

---

```
TheWarRoom — Session G0: Author the Go Overlay (coding standards)
This is a PLANNING + AUTHORING session. Talk through the open questions first; decide; then write. No guessing.

== CONTEXT ==
- Repo to work in: christopher-coding-standards (https://github.com/secureprospective/christopher-coding-standards)
- TheWarRoom is Go · Wails v2 · React+Tailwind+Zustand · SQLite WAL. Its 38-session build CANNOT start
  (B0 is blocked) until the Go overlay exists.
- What exists today: language-agnostic standards + a TypeScript overlay ONLY (templates/typescript/).
  There is NO Go overlay. This is net-new authoring, not a label fix.
- The TS overlay is the PATTERN to mirror: Biome (lint+format), Zod (boundary schema), Stryker (mutation),
  pre-commit framework, free-tooling-first, CI actions pinned to commit SHAs.

== READ FIRST ==
- christopher-coding-standards: README.md, AGENTS.md, docs/adr/0001, and ALL of templates/typescript/
- TheWarRoom: docs/build-handoffs/Build_Tracker.md (the standing rules every session inherits)
- The Session 3 audit: /root/.claude/plans/session-3-audit-build-sequencing.md (AD-06, AD-08 — enforcement
  decisions the linter should back deterministically)

== WHAT THE GO OVERLAY MUST DELIVER (mirror the TS overlay file set) ==
- templates/go/.golangci.yml          — the linter ruleset
- templates/go/.pre-commit-config.yaml — golangci-lint + gofumpt + gitleaks, SHAs pinned
- templates/go/Makefile                — lint / format / test / (mutation?) targets
- templates/go/validation/example.go   — the boundary-validation pattern (the Zod equivalent)
- templates/go/naming-conventions.md   — Go naming + TheWarRoom wireframe conventions
                                         (Get[X], Fetch[DataType], Score[Component], one exported fn/file)
- templates/go/README.md               — adoption walkthrough
- AGENTS.md                            — fill the Go stack-specific commands
- Fix the phase label (Go is Phase 2 in this repo; TheWarRoom CLAUDE.md had said "Phase 1D")

== OPEN QUESTIONS TO DECIDE BEFORE WRITING ==
1. golangci-lint ruleset strictness — which linters on? Specifically:
   - File/function size caps: how do we enforce the 250-line target / 400-line hard cap deterministically?
     (funlen for functions; what for file length?)
   - depguard — can we turn architectural laws into LINT rules? e.g. forbid "store imports another store,"
     forbid an L4 position file importing pipeline.go, forbid handlers importing StateWriter (AD-05/AD-08).
     This is the big lever: the linter enforcing the wireframes, not just style.
2. Boundary validation — the Zod equivalent for "all external input schema-validated" (Wails IPC + MFL
   ingestion). Library (go-playground/validator? ozzo?) vs explicit hand-rolled validators? Pick the pattern.
3. Mutation testing — adopt for Go now (gremlins), defer, or skip? TS runs Stryker weekly, not per-PR.
4. Formatter — gofmt or gofumpt (stricter)?

== VERIFICATION (mirror the TS overlay's end check) ==
Drop templates/go/ into an empty Go module. Write a deliberately bad .go file: an unchecked error,
an over-400-line function, a hardcoded credential, and a cross-layer import. Confirm golangci-lint flags
each, the pre-commit hook blocks the commit, and gitleaks catches the secret. If any step fails, the overlay
is incomplete.

== CLOSE ==
Merge the Go overlay in the standards repo, correct the phase label, then write the B0 handoff
(Session 1) per docs/build-handoffs/Handoff_Protocol.md. B0 is unblocked the moment this merges.
```

---

**Highest-leverage decision to chew on (question 1, depguard):** if `golangci-lint` can forbid the imports that would violate the layer law and the wireframes, then "no store imports another store" and "handlers never touch `StateWriter`" stop being discipline and become build failures — the deterministic-enforcement principle the standards are built on. Decide this first; it shapes the whole `.golangci.yml`.
