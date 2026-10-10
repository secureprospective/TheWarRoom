# TheWarRoom — Project CLAUDE.md
**Version:** 3.1 — 2026-10-10
**Project path:** `~/work/TheWarRoom` on the Beelink (the only clone that builds and runs the app)
**Pillars:** Business, Technical

## What this project is

A ranking engine and desktop app for the Legacy NFL, Christopher's 32-team dynasty IDP salary-cap
league on MFL (league 14432). Go scoring engine, Wails v2 shell, React + Tailwind + Zustand
frontend, SQLite (WAL), MFL API. The first user is Christopher as a GM (R1).

## Where the work stands

- **Plan:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`: stages 0–8, each with a gate, and
  the binding rulings R1–R12. The reasoning behind it: `Core_Build_Reasoning_2026-10.md`.
- **Latest state:** the newest `docs/build-handoffs/RESUME-*.md`.
- **What exists:** `SYSTEM_MAP.md`.
- **UI target track:** built in rings (`docs/build-handoffs/UI_Target_Roadmap_2026-10.md`), each
  ending with a spec revision (`docs/ui/Target_UI_Spec_2026-10.md`, revision 1 on 2026-10-10).
  Every endpoint is a row in `docs/ui/endpoint-registry.csv`; `frontend/scripts/gen-endpoints.mjs`
  generates `registry/endpoints.gen.ts` from it (never hand-edit the generated file).
  Ring 1 merged 2026-10-10 (PR #15). Ring 2, the decision layer, is next, with IR/taxi eligibility.

## Session start

1. `git branch --show-current`. Never work on main; branches are `session/<short-description>`.
2. Read the plan's current stage and the latest RESUME.
3. Toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB` (Go 1.26.4
   is not on the default PATH).
4. Gates: `make lint` (ifaceguard, filelen report, bloat ratchet, golangci-lint), `make test`
   (`go test -race ./...`), `make verify` (both plus the frontend build; the pre-push hook runs
   it). `make setup` once per clone wires the hooks. Pushing needs Go on PATH too, because the
   pre-push hook runs `make verify`.
5. Live tests are opt-in: `TWR_LIVE_MFL=1`, `TWR_LIVE_NFLVERSE=1`, `TWR_LIVE_CFBD=1`,
   `TWR_LIVE_EA=1`. MFL's players endpoint allows one call a day; leave its live test alone
   unless that call is the point.
6. Headless startup check: `go build -o /tmp/twr . && /tmp/twr -probe`. A dev build uses
   `thewarroom-dev.db`; never point a dev build at the real `thewarroom.db`.
   - Never run a production binary (`build/bin/thewarroom`) on the Beelink: it opens the real
     `thewarroom.db` and `history.db`. Check its stamp with `strings` or on Claude-OS.

## Workflow (R4)

- **Claude is the head brain** and the engineering authority: architecture, storage shape,
  refactors and standards are Claude's calls, explained so Christopher can veto. Christopher
  decides product intent, priorities and anything outward-facing. Claude plans, writes briefs,
  reviews every line Sol writes, fixes what is wrong, proves each fix with a test that fails
  without it, runs `make verify`, commits, and runs the live gates. Claude protects its context.
- **Sol writes the code:** `gpt-6.1-sol` through pi on Bee, at **medium** thinking. Do not raise
  it (Christopher, 2026-10-09: "it's been doing well on medium").
  - Briefs are files in `~/fleet/briefs/`. Write them in the scratchpad and `scp` them over,
    then check the size: `ssh -n` with a heredoc writes an empty file.
  - Dispatch: `dispatch.sh <phase> <brief>` in the ring's run directory under `~/fleet/runs/`,
    then poll `<phase>/sentinel`. Sol's report is `<phase>/REPORT.md`.
  - Sol never runs git or opens a GUI.
- **The expert panel** gates decisions that set a standard or are hard to undo:
  `~/fleet/bin/panel-dispatch.sh <brief>`. Four seats from four labs, no Claude (Sol, Kimi, GLM,
  Nemotron); quotes are checked against their files; `ROUNDS=2` adds cross-examination. Its
  output is a recommendation to Christopher, not a decision.
- **Christopher follows progress** in the savvy-progress `/agents` panel, where each pi session
  (Sol's phases, each panel seat) shows as its own card.
- **Claude runs live gates on Claude-OS (R12):** the production build (`make build`) against a
  snapshot of the live database, taken with SQLite's backup API, never a plain file copy.
  - Claude-OS is the libvirt VM `Claude-OS` under `qemu:///session` on this box. Start it with
    `virsh -c qemu:///session start Claude-OS` if it is off, and leave it running afterwards:
    it is light and in nobody's way (Christopher, 2026-10-03).
  - Reach it with `ssh claudeos`. Drive it with xdotool and scrot on `DISPLAY=:0`, and launch
    the app with `setsid -f` so the ssh session returns.
  - If Bee reboots, the VM comes back at the login screen: Christopher logs into the desktop
    before the app can start (otherwise GTK fails to init).
  - Deploy: `scp` the build to `~/warroom-ring1/thewarroom.new`, then
    `~/warroom-ring1/deploy.sh <old-sha>`, which keeps the previous binary.
  - Close the MFL tabs you opened when the work is done (spec revision 1, §8 Q12).
  - Screenshots and logs go to the run directory.
  - Never launch the app on Christopher's live desktop.
- Merge to main only on Christopher's go-ahead, after a passing live gate. `main` requires
  linear history: merge PRs with rebase (`~/.local/bin/gh pr merge <n> --rebase`).

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
| UI build order (rings) | `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` |
| UI target spec and its rulings | `docs/ui/Target_UI_Spec_2026-10.md` |
| Every endpoint and its status | `docs/ui/endpoint-registry.csv` |
| Open questions and decisions | `docs/roadmap/Roadmap_and_Open_Questions.md` |
| The 2026-10 engineering audit | `docs/reviews/engineering-audit-2026-10/` |
| Deferred: the Commissioner Suite | `docs/build-handoffs/Commissioner_Suite_Plan_DEFERRED.md` |
| History before the core plan | `docs/build-handoffs/Build_State_Archive_Through_Alpha.md` |

*Built by: Christopher Campbell + Claude (Anthropic)*
