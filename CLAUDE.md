# TheWarRoom — Project CLAUDE.md
**Version:** 3.3 — 2026-11-09
**Project path:** `~/work/TheWarRoom` on the Beelink (the only clone that builds and runs the app)
**Pillars:** Business, Technical

## What this project is

TheWarRoom houses a dynasty fantasy football league: its rules, its ledger and everything people do
in it. Every rule is a setting, any league can be built, and any MFL league can be imported. The
strategic reference is `North_Star.md`; read it before deciding what earns build time.

- **AI first, CLI first.** Every action is a named command in one catalogue (`internal/commandcatalog`).
  An AI agent working for a named person does everything through `cmd/twr` (CLI and MCP). Screens,
  minigames and reports are outputs laid over the same commands.
- **PWA-centric.** The React app served by `cmd/twr-server` to a browser or phone is the main
  interface. The desktop app (Wails) is first-class and supported; it runs the same screens. Where
  a desktop-only path and a hosted path do the same job, protect the hosted one.
- **The native ledger is the record.** MFL is import and reference only. The MFL→Native import is
  the front door and must be flawless for any league size or type.
- **Solo-owner tools.** `mflref`, `transactions` and the what-if desk let one owner use TheWarRoom on
  a league that still lives on MFL. Keep this framework simple: trim it, never grow it, never delete it.
- **The engines are the value.** The fitted model (`internal/model`, `model/fit`, `modelrun`,
  `appvalue`, `powerrankings`, `m2service`, `measures`, `projection`, `pickvalue`, `store/params`),
  the signal pipelines that feed it and the league mechanics under `internal/engine/*`. Do not break them.
- **Small on purpose.** One build serves every league type and rule set. Never add infrastructure
  per league type. Less code is a feature.

Stack: Go 1.26, SQLite (WAL), React + Tailwind + Zustand, Wails v2. The first league is the Legacy
NFL, Christopher's 32-team dynasty IDP salary-cap league (MFL league 14432).

## Where the work stands

- **Governing plan:** `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` (one always-on server, the same
  React app as PWA and desktop, CLI/MCP) with its rulings in `docs/audit/`.
- **Direction after that:** `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md` and the seam
  registry `docs/architecture/future-seams.md`.
- **Latest state:** the newest `docs/RESUME-*.md`. The 2026-11-09 debloat (what was cut, what was
  kept and why) is `docs/RESUME-debloat-1109.md`; words that still disagree across documents are
  listed in `docs/DEBLOAT-semantic-pot-1109.md`.
- **Engine plan and rulings R1–R12:** `docs/build-handoffs/Core_Build_Plan_2026-10.md`; the reasoning
  is `Core_Build_Reasoning_2026-10.md`. Where a later ruling disagrees (R2 is superseded: the native
  ledger wins), the later ruling governs.
- **Earlier plans, reference only:** `docs/build-handoffs/Hardening_Plan_2026-10.md` (its Method
  section still describes how work is briefed and reviewed) and `UI_Target_Roadmap_2026-10.md`.
- **What exists:** `SYSTEM_MAP.md`, a feature directory; the long narrative is
  `docs/architecture/system-narrative.md`. CI's `scripts/check-root-ratchet.sh` caps root Go
  files at **1** (`docs/architecture/root-go-baseline.txt`); feature code goes in `internal/app/`.
- **UI:** the target spec is `docs/ui/Target_UI_Spec_2026-10.md`. Every endpoint is a row in
  `docs/ui/endpoint-registry.csv`; `frontend/scripts/gen-endpoints.mjs` generates
  `frontend/src/app/registry/endpoints.gen.ts` from it (never hand-edit the generated file).

## Session start

1. `git branch --show-current`. Never work on main; branches are `session/<short-description>`.
2. Read `North_Star.md`, the governing plan and the newest `docs/RESUME-*.md`.
3. Toolchain (Go 1.26.10; the git hooks need this too):
   `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10 GOMEMLIMIT=1500MiB
   GOMAXPROCS=8 GOFLAGS="-mod=readonly -buildvcs=false" VITEST_MAX_THREADS=2 VITEST_MIN_THREADS=1`.
   Always pass `VERSION=dev COMMIT=<ring> BUILD_DATE=<ring>` to make. The commit gates are
   `timeout 600 nice -n 10 make verify …` and `make build …`, both exit 0.
   The entry bundle cap is **321,818 B** (ring 2 ended at 317,727): keep new screens lazy.
4. Gates: `make lint` (ifaceguard, filelen report, bloat ratchet, golangci-lint), `make test`
   (`go test -race ./...`), `make verify` (lint, race tests, vulnerability/bindings checks and
   frontend checks/build). `make setup` once per clone wires the hooks. Pre-push runs
   `scripts/prepush.sh`: docs-only skips; Go changes run lint and shuffled tests on changed
   packages and importers (race runs in CI); frontend changes run typecheck, tests, build and
   bundle budget. Build/config changes or no usable diff base fall back to `make verify`.
   Pushing needs Go on PATH too; CI runs the full gate on every push.
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
  - **Tom** (Claude Code on the Beelink, Opus at medium, never Sonnet) is head brain for ring 3
    and builds only when Sol is down, through `tom-dispatch-wt.sh`.
  - Briefs are files in `~/fleet/briefs/`. Write them in the scratchpad and `scp` them over,
    then check the size: `ssh -n` with a heredoc writes an empty file.
  - Dispatch: `dispatch-wt.sh <phase> <brief>` in the ring's run directory under `~/fleet/runs/`
    (copy the previous ring's scripts and change `RUN=`). Launch detached; done is
    `<phase>/sentinel`, liveness is the mtime of `<phase>/sessions/*.jsonl` (Sol) or
    `transcript.jsonl` (Tom). Sol's report is `<phase>/REPORT.md`. Commit with `git commit -F`.
  - Every builder report gets one mutant planted by the reviewer and a live click-through on
    Claude-OS before Christopher is asked to click.
  - Sol never runs git or opens a GUI.
- **The expert panel** gates decisions that set a standard or are hard to undo:
  `~/fleet/bin/panel-dispatch.sh <brief>`. Four seats from four labs, no Claude (Sol, Kimi, GLM,
  Nemotron); quotes are checked against their files; `ROUNDS=2` adds cross-examination. Its
  output is a recommendation to Christopher, not a decision.
- **Christopher follows progress** in the savvy-progress `/fleet` panel, where each pi session
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
  linear history and rebase fails on a branch holding a merge commit: **squash-merge**
  (`~/.local/bin/gh pr merge <n> --squash`).

## Hard Constraints (never route around)

- **No work on main.** Never `git --no-verify`.
- **Never submit anything to MFL** and never make a real roster move there. MFL is reference and
  links only; click nothing on owner pages, and keep owner-only captures out of git.
- **MFL player IDs are strings.** IDs under 1000 keep their leading zeros. `playerid.New` is
  the only constructor; all SQLite id columns are TEXT.
- **Native ledger wins (2026-11-05 client/server ruling 5a; supersedes core-plan R2).** MFL is
  import/reference only; solo owners and commissioners get a review banner for MFL changes.
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
- **The app houses the settings (10-27, Christopher).** A league is the values of its settings; the
  engine honours every setting for any league. A preset is a quick named bundle of suggested values
  applied at creation, nothing more: it never limits a value, never unlocks or forbids another setting,
  and no code asks which preset a league came from. Each setting stands alone; changing one never forces
  another. Every setting changes through the one rule-change flow. The only refusals are what the
  engine truly cannot run, said in plain words. Be accommodating, not restrictive.
  **An MFL import is just settings in the same app (10-31):** where a league came from is not a kind of
  league. No code branches on a league's source or on one league's ID, and nothing is locked because it
  came from MFL. Frontend = how a human works the backend; backend = engines (the math) + stores. Test:
  could a human reach this league by changing settings by hand, and change it back? If not, it is slop.
- **Do not reopen locked decisions** or add features the documents don't call for. If a locked
  decision creates a constraint that feels wrong, flag it to Christopher.

## Key documents

| Need | Document |
|---|---|
| How code is written and judged: load before any build or review | `docs/standards/quality-contract.md`, `docs/standards/architecture-review.md` |
| Older build doctrine (reference only) | `docs/agent-codex.md` |
| Purpose, scope and what earns build time | `North_Star.md` |
| The governing plan (client/server, PWA) and its rulings | `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md`, `docs/audit/` |
| Earlier plan; its Method section is still how work is briefed | `docs/build-handoffs/Hardening_Plan_2026-10.md` |
| The engine plan and rulings | `docs/build-handoffs/Core_Build_Plan_2026-10.md` |
| Why the plan is shaped this way | `docs/build-handoffs/Core_Build_Reasoning_2026-10.md` |
| Where each feature lives (two hops), and where new code goes | `SYSTEM_MAP.md` |
| Future direction: AI-first seams, social moments, tiny models (11-08) | `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md` |
| **Before any new server feature** (moments, shares, invites, chat, new AI powers): where it plugs in | `docs/architecture/future-seams.md` (`grep -rn "FUTURE-SEAM(" internal`) |
| What exists at length, and the import rules | `docs/architecture/system-narrative.md` |
| Which feature owns a root file or package | `docs/architecture/placement.tsv` |
| Engine architecture | `docs/scoring-engine/Engine_Specification.md` |
| A position's rubric | `docs/scoring-engine/<POSITION>_Rubric.md` |
| MFL API and scoring | `docs/data-layer/MFL_API_Specification.md`, `MFL_Scoring_Rules_Decode.md` |
| Approved data sources | `docs/sources/Approved_Sources.md` |
| UI direction | `docs/ui/UI_Direction_Document.md` |
| UI build order (rings, reference) | `docs/build-handoffs/UI_Target_Roadmap_2026-10.md` |
| UI target spec and its rulings | `docs/ui/Target_UI_Spec_2026-10.md` |
| Every endpoint and its status | `docs/ui/endpoint-registry.csv` |
| Commissioner surface: MFL map, cut, K1–K7 process, rulings | `docs/commissioner/README.md` |
| Open questions and decisions | `docs/roadmap/Roadmap_and_Open_Questions.md` |
| The 2026-10 engineering audit | `docs/build-handoffs/Deep_Audit_2026-10-19.md` (the Oct 3 audit was removed; see `docs/architecture/removed-code.md`) |
| Deferred: the Commissioner Suite | `docs/build-handoffs/Commissioner_Suite_Plan_DEFERRED.md` |
| History before the core plan | `docs/build-handoffs/Build_State_Archive_Through_Alpha.md` |

*Built by: Christopher Campbell + Claude (Anthropic)*
