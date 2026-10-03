# TheWarRoom — Project CLAUDE.md
**Version:** 3.0 — 2026-10-03
**Project path:** `~/work/TheWarRoom` on the Beelink (the only clone that builds and runs the app)
**Pillars:** Business, Technical

## What this project is

A ranking engine and desktop app for the Legacy NFL, Christopher's 32-team dynasty IDP salary-cap
league on MFL (league 14432). Go scoring engine, Wails v2 shell, React + Tailwind + Zustand
frontend, SQLite (WAL), MFL API. The first user is Christopher as a GM (R1).

## Where the work stands

- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`: stages 0–8, each with a gate, and
  the binding rulings R1–R11. The reasoning behind it: `Core_Build_Reasoning_2026-10.md`.
- **Latest state:** the newest `docs/build-handoffs/RESUME-*.md`.
- **What exists:** `SYSTEM_MAP.md`.

## Session start

1. `git branch --show-current`. Never work on main; branches are `session/<short-description>`.
2. Read the plan's current stage and the latest RESUME.
3. Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB` (Go 1.26.4
   is not on the default PATH).
4. Gates: `make lint` (ifaceguard, filelen report, bloat ratchet, golangci-lint), `make test`
   (`go test -race ./...`), `make verify` (both plus the frontend build; the pre-push hook runs
   it). `make setup` once per clone wires the hooks.
5. Live tests are opt-in: `TWR_LIVE_MFL=1`, `TWR_LIVE_NFLVERSE=1`, `TWR_LIVE_CFBD=1`,
   `TWR_LIVE_EA=1`. MFL's players endpoint allows one call a day; leave its live test alone
   unless that call is the point.
6. Headless startup check: `go build -o /tmp/twr . && /tmp/twr -probe`. A dev build uses
   `thewarroom-dev.db`; never point a dev build at the real `thewarroom.db`.

## Workflow (R4)

- **Claude builds** and is the engineering authority: architecture, storage shape, refactors
  and standards are Claude's calls, explained so Christopher can veto. Christopher decides
  product intent, priorities and anything outward-facing.
- **Bee reviews** finished stages when GPT budget allows. Briefs go in a run directory under
  `~/fleet/runs/`; Bee writes drafts there and never runs git or opens a GUI.
- **Christopher runs live gates** on his desktop: the real app against the real league. Do not
  launch the app on his live desktop yourself.
- Merge to main only after Christopher confirms the live result.

## Hard Constraints (never route around)

- **No work on main.** Never `git --no-verify`.
- **MFL player IDs are strings.** IDs under 1000 keep their leading zeros. `playerid.New` is
  the only constructor; all SQLite id columns are TEXT.
- **MFL wins (R2).** A refresh overwrites the app's league and ledger state. Moves made in the
  app are what-if plans until made on MFL.
- **Confidence scores are internal engine flags.** Never surface them in the UI.
- **No scoring leak into the scouting prior.** The prior (draft capital, combine, college,
  film, RAS, breakout) must not reference fantasy points, projected volume, MFL scoring config
  or format-dependent volume; that independence is what stops it double-counting production.
  The production side of the blend uses usage and fantasy points by design (R6).
- **Scouting caps and weights are adjustable settings (R7),** stored as params and editable in
  the Admin Console. This supersedes the old rule that Layer 4 mechanics stay out of the Admin
  UI.
- **The coverage anchor applies at CB and S only.** Nil at every other position.
- **SL-019 is not applied at DT;** the Cushion Guard replaces it. Running both double-protects.
- **A lost source never stops the board (R11).** It runs on the measures still flowing,
  labelled as reduced; a rebalance applies only when Christopher approves it.
- **Do not reopen locked decisions** or add features the documents don't call for. If a locked
  decision creates a constraint that feels wrong, flag it to Christopher.

## Key documents

| Need | Document |
|---|---|
| Build doctrine: load before any build or review | `docs/agent-codex.md` |
| The plan, rulings and gates | `docs/build-handoffs/Core_Build_Plan_2026-10.md` |
| Why the plan is shaped this way | `docs/build-handoffs/Core_Build_Reasoning_2026-10.md` |
| What exists, and the import rules | `SYSTEM_MAP.md` |
| Engine architecture | `docs/scoring-engine/Engine_Specification.md` |
| A position's rubric | `docs/scoring-engine/<POSITION>_Rubric.md` |
| MFL API and scoring | `docs/data-layer/MFL_API_Specification.md`, `MFL_Scoring_Rules_Decode.md` |
| Approved data sources | `docs/sources/Approved_Sources.md` |
| UI direction | `docs/ui/UI_Direction_Document.md` |
| Open questions and decisions | `docs/roadmap/Roadmap_and_Open_Questions.md` |
| The 2026-10 engineering audit | `docs/reviews/engineering-audit-2026-10/` |
| Deferred: the Commissioner Suite | `docs/build-handoffs/Commissioner_Suite_Plan_DEFERRED.md` |
| History before the core plan | `docs/build-handoffs/Build_State_Archive_Through_Alpha.md` |

*Built by: Christopher Campbell + Claude (Anthropic)*
