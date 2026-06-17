# AGENTS.md

**Advisory guidance, not enforcement.** Pre-commit hooks, `make lint`, and the compiler enforce the rules. If guidance here conflicts with a gate, the gate wins — fix the violation, do not bypass. Never `git --no-verify`.

## Project profile

- **Project name:** TheWarRoom
- **Primary language(s):** Go 1.26 (backend, scoring engine), TypeScript + React (frontend)
- **Public exposure:** Internal (Phase 1 personal tool; League alpha / public beta are Phase 2–3)
- **Standards version:** christopher-coding-standards Go overlay (Phase 2)

## Build doctrine — load before writing or reviewing code

- **`docs/agent-codex.md`** — the 17 motifs, slop catalog, and canon→motif map. Cite motifs by ID (`§M3`) in review notes.
- **`CLAUDE.md` → Hard Constraints** — the never-route-around rules (string player IDs, zero scoring leak, NGS anchors, SL-019 exclusions). Locked decisions are not reopened silently; flag conflicts to Christopher.
- **`Fable_TheWarRoom_code_plan.md`** — copy the matching Go skeleton (Section 4) and read the per-session brief (Section 6) before writing. Never invent structure. Every domain number lives in Section 5 — read it, never recall.

## Code footprint

- Target file size under 250 lines. Hard cap 400 (`make filelen` enforces). Pre-split from the wireframe; do not write-then-hack-apart.
- No copy-paste. Check `SYSTEM_MAP.md` for existing utilities first.
- Add a dependency only when the alternative is >~15 lines of native code.
- No commented-out code. Delete; `git` remembers.

## Security mandates

- All external input (MFL API responses, CSV ingest, IPC payloads) validated through an explicit hand-written schema (`internal/schema`) before business logic. No ad-hoc parsing, no `interface{}`/`any` escapes (`ifaceguard` enforces).
- Parameterized SQL only. No string concatenation into queries (`gosec` enforces). Raw `database/sql` is confined to `internal/db` and `internal/store` (`depguard` enforces).
- No hardcoded secrets, tokens, or hosts. The MFL host is discovered at runtime, never hardcoded.
- Errors wrapped with context (`%w`), never silently dropped (`errcheck` + `wrapcheck`).

## Architecture as build errors (do not route around)

The three-layer law is enforced by `depguard`, not goodwill:
- Layer 1 (`internal/mfl`, `internal/ingestion`) never imports up into the engine, stores, transactions, or API.
- `internal/engine` is pure — imports no store, no DB, no I/O. All state arrives as parameters.
- Stores never import each other. Only the B7 coordinator holds a `StateWriter`.

If a locked decision creates a constraint that feels wrong, **flag it to Christopher** — do not work around it.

## Workflow

- Tests required for every functional change. `make test` runs with `-race` (non-negotiable).
- Run before committing: `make lint` then `make test`. Fix hook failures; never `--no-verify`.

## Stack-specific commands

- **Lint:** `make lint` (ifaceguard + filelen + golangci-lint)
- **Format:** `make fmt`
- **Test:** `make test` (`go test -race ./...`)
- **Type check (frontend):** `cd frontend && pnpm tsc --noEmit`
- **Build:** `wails build` (desktop binary) / `go build ./...` (engine-only)

## Commit and PR conventions

- Conventional prefixes: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `build:`, `ci:`.
- Append `[AI-assisted]` to commit messages an AI agent generated or substantially modified (audit traceability).

## What not to do

- Do not edit `.git/`, `node_modules/`, `frontend/dist/`, `build/bin/`, or `frontend/wailsjs/` (generated bindings — regenerate with `wails generate module`).
- Do not pin third-party CI actions or pre-commit hooks to mutable tags. Pin to commit SHAs (the trivy-action lesson).
- Do not introduce a new top-level directory or `internal/` package without updating `SYSTEM_MAP.md`.
- Do not add a package-level `var`. The only sanctioned globals: sentinel errors (`var ErrFoo = errors.New(...)` with `//nolint:gochecknoglobals` + explanation — `nolintlint` requires it) and `//go:embed` vars (auto-exempt, no nolint). Everything else is the failure mode the linter exists to stop. `nolintlint` rejects any unused or unexplained `//nolint`.
- Do not generate an architecture-overview section in this file (inflates tokens, does not change behavior).

## Where to look

- `SYSTEM_MAP.md` — what exists and where it goes. Check before writing.
- `docs/build-handoffs/Build_Tracker.md` — the 38-session sequence; current progress.
- `docs/` — engine spec, rubrics, data layer, UI, backend architecture.
- `.pre-commit-config.yaml` + `.golangci.yml` — the gates that run on your commit.
