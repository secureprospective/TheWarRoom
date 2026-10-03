# TheWarRoom — Claude's digest of purpose, vision and state (2026-10-03)

Owner of this file: Claude (driving session). Bee does not edit it.
Status: DRAFT — hypotheses marked (H) await Bee's evidence.

## Purpose (from North_Star.md v1.2, June 2026)

Christopher's own tool for the Legacy NFL (MFL league 14432, 32-team dynasty, IDP, salary cap,
contracts). Phase 1 = personal tool: "competitive edge for one GM processing all 32 teams."
Four pillars: (1) six-layer Scoring Engine — "the value proposition; every module surfaces some
view of its output"; (2) Transaction System — rule enforcement, Phase 1 read-only vs MFL,
outputs executed by hand on MFL; (3) eight Modules (M1 assets, M2 power, M3 matchups,
M4 transactions, M5 free agency, M6 rookie draft, M7 trade analyzer, M8 commissioner);
(4) Admin Console — calibration through UI, not code.

Roadmap Phase 1 completion list: MFL pipeline (32 rosters, contracts, players, scoring),
engine with rulebook-verified values, 32-team asset rankings, per-team roster view, basic
transaction validation (cap/roster/deadline), dead-cap calculator. Nice-to-have: power
rankings, free-agent pool with scores. **None of the checkboxes were ever ticked.**

## Timeline (git: 104 commits Jun, 206 Jul, 6 Aug, 1 Sep)

| When | What |
|---|---|
| Jun 13–28 | Specs, scaffold, MFL client, ingestion, normalize, 3 stores, engine spine, all 10 L4 rubrics (Jun 27), output store |
| Jul 2 | M1 asset rankings live — first visible engine output |
| Jul 3–18 | B7 transaction engine + contract §-series (tag, extension, restructure, buyout, rollover, FA, UFA calendar), M4 transaction UI |
| Jul 6 | Vision_2026: chat "Capologist", portfolio desk, January planner, MFL cutover |
| Jul 19–21 | M2 power rankings; scouting wiring sprint; review harvest |
| Jul 24–25 | B-1→B-5 console UI redesign; "Alpha Gate passed" |
| Jul 27 | Workflow switched to GLM-implements / DeepSeek-reviews; new 10-session "Commissioner Suite" plan (lives only on CT105: `/root/.claude/plans/commissioner-suite-build-sequence.md`) |
| Jul 28 | Sessions 0–2 merged, live-gated PASS (recorded only on origin/main) |
| Aug | pnpm guardrail PR; multi-agent channel era (Nexus, Warden, panels); Pi baton Aug 19 |
| Sep–now | Path-repoint docs only. Data last written Jul 27. |

## Repo state (verified)

- Authoritative clone `~/work/TheWarRoom`, checkout `session/m1b-bash` = local main + 4 commits
  (deps bump wails 2.13, regenerated bindings, path repoints, docs). 1 uncommitted doc.
- `origin/main` = local main + 3 commits (live-gate PASS record, CLAUDE.md session close, pnpm
  guardrail). Checkout and origin/main have diverged both ways; code differs only in deps.
- Unmerged/stale branches: b7c-buyout (141 behind; buyout re-landed as 73727e5 on main),
  session-0/1/2 originals, warden-pr2 + migration/pnpm (= the pnpm PR), ~20 remote heads.
- Two league DBs: `thewarroom.db` (stamped builds, Jul 24, 16 tables, 831 roster rows) and
  `thewarroom-dev.db` (un-stamped builds, Jul 27, 20 tables, 1375 roster rows).
- CLAUDE.md workflow section prescribes retired tooling (OpenCode/GLM/DeepSeek) and CT105 paths.
- SYSTEM_MAP.md still tags built packages "[planned: Bn]".
- `.project.yaml` says 11 buzz-acp services run from here; no buzz units exist now.

## Drift hypotheses (H) — Bee to confirm or refute

1. (H) **The engine — "the value proposition" — runs mostly on neutral inputs.** L2 base is a
   labelled proxy; ~10 of 11 scouting signals fetched but unwired. If true, the M1 number is
   driven mostly by age decay and cap tier, and every downstream module inherits that.
2. (H) **Effort migrated from GM intelligence to league-office machinery.** July went to the
   contract/transaction suite and a Commissioner Suite — a league-instance product (Phase 2/3) —
   while the GM-facing modules a Phase-1 user needs in season (M3 matchups, M5 free agency,
   M6 rookie draft, M7 trade analyzer) were never started.
3. (H) **The in-app ledger and MFL can drift apart.** Phase 1 is read-only against MFL, yet the
   per-year salary ledger is "the sole cap truth" and is mutated by in-app transactions. Whether
   a refresh from MFL reconciles it is the question that decides if the app is usable mid-season.
4. (H) **Vision documents outran the build.** Vision_2026 horizons, MFL cutover, AI owners —
   the README markets features the engine does not yet feed.
5. (H) **Process overhead accreted:** many agents, panels, multi-model reviews; docs that tell a
   new session to use retired tools; two competing build sequences (Build_Tracker vs the CT105
   Commissioner Suite plan) and the authoritative one is not in the repo.

## Scope decision — Christopher, 2026-10-03 (LOCKED)

His words: "If we are unable to put a measurable on the real NFL players, then the rest is only
assumptions on the application output, which every single other fantasy application does. We
are different: the weights of the real NFL players dictate how the application responds to the
application's user inputs." "If we do not have a solid core the rest is window dressing."

- **First plan = cleanup + the core.** Cleanup puts code, branches, data and docs on one
  timeline. Core = base scouting (real-NFL-player measurable) → power rankings (M2).
- **Measurable = two numbers per player, kept separate:** on-field-now (production, snaps,
  film) and dynasty asset value (age, athleticism, breakout, contract). M2 can use either.
- **Deferred:** Layer 2 fantasy-points scoring (stays on MFL's points), the Commissioner Suite,
  contract/transaction machinery beyond keeping it working, all Vision_2026 horizons.
- Passed to Bee as FEEDBACK-2 Part C before its Sector 2 closed.

## Standing bars — Christopher, 2026-10-03 (memory: seek-the-better-architecture-and-cut-ai-slop)

1. Where a better architecture exists, explore the options before moving on. Present them side
   by side with a recommendation; he chooses.
2. Always identify and remove AI slop: volume from repetition ("600 if tags vs 4 short lines").
   The project's own slop catalog (agent-codex §4) lacks this smell; adding it is a cleanup item.
   Leads measured: 10 parallel L4 rubric files (~1,400 lines); branch-heavy files
   (transactions_app.go 44, contracts.go 44, rulebook.go 38); Go files at the 400-line cap
   (state.go = 400); frontend over the cap (TransactionWorkspace.tsx 817, CalendarBoard.tsx 747).
   Passed to Bee as FEEDBACK-2 Part D (new Sector 4f, slop hunt in 4c).

## Headline finding — H1 confirmed (Sector 2 + my check, 2026-10-03)

`ScoutingAdjusted = BasePoints × AgePull × L4.Combined` (`internal/engine/pipeline.go`).
BasePoints = prior-season MFL fantasy points (`playerScores W=YTD`); missing → 0.
Dev DB `season_scores`: 1299 rows, 184 (14%) base = 0 → final score 0 regardless of scouting.
L4 combined spans 0.933–1.146; base spans 0–477. Rough log-variance: base ≈ all, L4 ≈ 0.1%.
**The app ranks players by last season's fantasy points; the real-player measurable is a ±7%
nudge.** Inverts the thesis. Architecture question for the plan: the measurable must stand on its
own (two numbers), with fantasy points as one input to on-field-now, not the base.
Other Sector-2 facts: two season sources (hard-coded `SeasonYear="2026"` vs phase log);
`ScoreLeague` never rescores an existing (season, rulebook) board, so param edits never reach M1;
K rubric gets no inputs; DT cushion params stored but ignored (literals 8.00/0.90);
L6 scarcity = 0 for all positions; M2 = 60% sum of those scores + 40% MFL all-play.

## Design direction — the blend (Christopher, 2026-10-03)

"Last year's points are important, but we need to build in a blending effect... rookies have a
clearer scouting pattern as our application is new and not looking back... the blending effect
needs to be a fluid part of the engine, because in 5 years we will have clear rookie scouting
and current years played."

Leading candidate (credibility weighting, the actuarial form): measurable = Z·production +
(1−Z)·scouting prior; Z = evidence/(evidence+k); evidence = recency-discounted snaps/games;
k per position, admin-tunable now, learned from the app's own history later.
Consequence: **start the clock now** — persist each season's raw scouting inputs and the
production that followed, or the calibration data is lost for good. Passed to Bee as
FEEDBACK-2 Part F for evaluation against alternatives in Sector 4f.

## Calendar fact that shapes "usable"

Today is 2026-10-03 — the NFL regular season is around Week 5. The league's trade deadline is
Week 9 per the docs. In-season, the useful surfaces are current rosters/cap, power rankings,
matchups, free agency and trades. The app's data is from late July.

## Open questions for Christopher (to become a matrix after Bee lands)

- Which user does "functional and usable" serve first: Christopher as a GM, or as commissioner?
- Is the app's ledger meant to mirror MFL (refresh-and-reconcile) or be the system of record?
- Which plan is authoritative: Build_Tracker or the CT105 Commissioner Suite plan?
- Who builds now that OpenCode/GLM/DeepSeek are retired — Claude, Bee, or a split?
- What must work this season, before the Week-9 deadline?
- Branch reconciliation: fold m1b-bash and origin/main into one line before cleanup starts.
