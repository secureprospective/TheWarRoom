# RESUME — TheWarRoom engineering audit + cleanup/core plan (2026-10-03)

Session driver: Claude (Opus 5.5) on the Beelink. Compact-safe written ~11:20 local.
Companion (keep reading after this): `~/fleet/runs/warroom-dataflow-2026-10-03/CLAUDE-DIGEST.md`
— purpose, timeline, locked decisions, the headline finding, in more detail.

## 1. WHAT WE ARE DOING

Christopher revived TheWarRoom (parked 09-28). **This session builds no code.** It produces a
plan: cleanup first, then build the core component by component so each feeds the next, until
the current system works as intended. The future roadmap is out of scope. Method: Bee maps the
code and data, Astra researches outside sources, Claude triages, then a discussion with
Christopher (questions as a pick-from matrix) fills the gaps, then the plan is written.

- Repo: `~/work/TheWarRoom` (authoritative clone), remote `secureprospective/TheWarRoom`.
  Checkout `session/m1b-bash` = local main + 4 commits; **`origin/main` = local main + 3 other
  commits** (live-gate PASS record for Sessions 1+2 on 07-28, CLAUDE.md session close, pnpm
  guardrail PR #2). Code differs only in deps (wails 2.13 vs 2.12, x/* versions) and generated
  bindings. One uncommitted doc not ours: `docs/reviews/warroom-review-week-2026-07/SESSION_PROMPT_review-harvest.md`.
- League data (live, do not write): `~/.config/TheWarRoom/thewarroom.db` (stamped builds,
  Jul 24) and `thewarroom-dev.db` (un-stamped builds, Jul 27). Open only with `?immutable=1`.
  A plain read-only open of a WAL DB creates `-wal`/`-shm`. I did that once; nothing held them
  open and I deleted both, so the DB file itself is unchanged.

## 2. AGENTS + HARNESSES (both IN FLIGHT at compaction)

| Agent | What | Brief | Run dir | Alive check | Recovery |
|---|---|---|---|---|---|
| **Bee** (Pi, gpt-6-luna, thinking max, 8h timeout from 09:50) | Read-only code + data audit, 4 sectors | `~/fleet/briefs/bee-warroom-dataflow-2026-10-03.md` | `~/fleet/runs/warroom-dataflow-2026-10-03/` | `pgrep -af bee-warroom-dataflow`; transcript mtime: `find ~/.pi/agent/sessions -name '*59481552-1d38-40a1-9ad8-6fa567d3483d*'` | Session id `59481552-1d38-40a1-9ad8-6fa567d3483d`; exit line lands in `warroom-dataflow.sentinel` |
| **Astra** (Pi, run by Christopher himself, started ~10:44) | Outside research: source currency, gap sources, stabilization evidence for `k`, starting values | `~/fleet/briefs/astra-warroom-core-research-2026-10-03.md` | `~/fleet/runs/warroom-core-research-2026-10-03/` | Its process is not visible to me; check `logs/steplog.md` mtime (`find ... -mmin -30`) | Ask Christopher for its session if it dies |

Runner: `~/fleet/bin/run-bee-warroom-dataflow.sh`. Monitors are session-scoped and die at
compaction. **Re-arm them on resume.** For Bee, watch the sentinel mtimes and `pgrep`. For
Astra, watch the sentinels and a stall check using `find -mmin -30`. `-newermt '-30 minutes'`
was broken: it gave a false stall alert.

**Sentinels to expect.** Bee: `SECTOR-3.DONE`, `SECTOR-4.DONE`, `FINDINGS-INDEX.md`,
`REVIEW-DONE`, `warroom-dataflow.sentinel`. Astra: `SECTOR-1..4.DONE`, `FINDINGS-INDEX.md`,
`RESEARCH-DONE`. **Review each sector as it lands.** Send a thin sector back by writing
`FEEDBACK-N.md` in that run dir (each agent checks for it after every sector).

## 3. GATES / STATUS

| Item | State |
|---|---|
| Bee S1 Inventory (`INVENTORY.md`) | PASSED after one revision (FEEDBACK-1 dev DB + origin/main docs; FEEDBACK-2 Part A: neither DB is a faithful league mirror) |
| Bee S2 Flows (`MAP-data-flows.md`, `REPORT-2-flows.md`) | PASSED after revision (FEEDBACK-2 Parts B–F). 13 findings: 7 HIGH, 6 MEDIUM |
| Bee S3 Evidence (tests, DB checks, migration test on copies) | IN PROGRESS at compaction |
| Bee S4 Intent vs reality, slop hunt, seams, 4f architecture options | NOT STARTED |
| Astra S1–S4 | S1 IN PROGRESS at compaction |
| Discussion matrix with Christopher | NOT STARTED, waits for both agents |
| Plan document | NOT STARTED |

Bee's IPC count was checked independently: 24 exported `*App` methods = 24 bindings.

## 4. KEY FINDINGS SO FAR (verified against source or DB)

1. **HEADLINE (H1 confirmed).** `ScoutingAdjusted = BasePoints × AgePull × L4.Combined`
   (`internal/engine/pipeline.go`). BasePoints = MFL `playerScores W=YTD` for season−1; missing
   → 0. Dev DB: 184 of 1299 scored players (14%) have base 0, so they score 0. L4 spans
   0.933–1.146 against base 0–477. **The app ranks by last season's fantasy points; scouting is
   a ±7% nudge.** This inverts the thesis. Bee rates it HIGH; I rate it the blocker.
2. **No MFL refresh after the first seed.** `state.Initialize` seeds only when empty.
3. **Neither DB is a faithful league mirror.** The release DB was test-cycled through phases
   and rolled over to 2027 (395 §14 expiry releases). The dev DB has in-app test transactions.
   The plan must start from a fresh MFL pull plus reconciliation.
4. **Two season sources.** Hard-coded `ingestion.SeasonYear = "2026"` versus the phase-log
   season.
5. `ScoreLeague` never rescores an existing (season, rulebook) board, so `SetParam` never
   reaches M1.
6. Madden is pinned to `m24-ratings` (2023 edition) with no freshness guard. In June, m25 was
   empty and m26 returned 500.
7. The K rubric gets no inputs. The DT cushion params are stored but ignored (literals
   8.00/0.90). L6 scarcity = 0 everywhere.
8. Fetchers built but unwired: `nflproduction`, `touchshare` (snaps), `kicking`, `pfrpassrush`,
   `salaryadjustments`, `nflSchedule`. The first two are exactly what the blend needs.
9. App logs: 7 files in the live dir, all 0 bytes. Cause is with Bee (FEEDBACK-2 B3).
10. M2 = 60% sum of M1 AdjustedScore (median/MAD z) + 40% MFL all-play win%. It inherits #1.
11. Docs drift: CLAUDE.md prescribes retired tooling (OpenCode/GLM/DeepSeek) and CT105 paths;
    SYSTEM_MAP tags built packages "[planned]"; `.project.yaml` claims 11 buzz-acp services (none
    exist); the Commissioner Suite plan lives only on CT105
    (`/root/.claude/plans/commissioner-suite-build-sequence.md`), not in the repo.
12. Task list (Hermes): T070 — the pre-push `ifaceguard` fails, which is why the branch is
    unpushed. T015 — `CFBD_API_KEY` is exported in `.bashrc`, and the CFBD scouting signals are
    gated on that key.

## 5. CURRENT OPEN QUESTIONS (not bugs)

- Are the raw scouting inputs persisted per season, or fetched live and lost? (FEEDBACK-2 F.3)
  If lost, "start the clock now" jumps to the front of the plan.
- Do Sector 3 results show test failures, and are they sandbox or code?

## 6. HYPOTHESES ALREADY SETTLED — DO NOT RETEST

- "`thewarroom.db` is the real league DB" — REFUTED. Phase log shows test cycling and a 2027
  rollover.
- "The roster gap 831 vs 1375 is real league change" — REFUTED. 395 of the 544 are the test
  rollover's expiries.
- "The scouting layer meaningfully moves scores" — REFUTED (finding #1).
- "There's a newer DB copy from the live gates" — CHECKED. Only the two in `~/.config/TheWarRoom/`.
- "The buzz-acp services run from this repo" — REFUTED. No buzz units exist.
- "Astra stalled" at ~10:47 — FALSE ALARM. My monitor's `-newermt` was broken.

## 7. DECISIONS (Christopher, 2026-10-03 — do not relitigate)

- **D-A Scope.** The first plan is cleanup plus the core. Core = base scouting (the
  real-NFL-player measurable) → M2 power rankings. "If we do not have a solid core the rest is
  window dressing."
- **D-B Deferred:** Layer 2 fantasy-points scoring (stays on MFL's points), the Commissioner
  Suite, contract/transaction machinery beyond keeping it working, all Vision_2026 horizons.
- **D-C Two numbers per player, kept separate:** on-field-now and dynasty value. M2 can use
  either.
- **D-D The blend.** Last year's points matter, but rookies and injured players need a fluid,
  evidence-weighted blend with scouting. Leading candidate, to be evaluated against
  alternatives (Bee 4f): credibility weighting. Z = evidence/(evidence+k), with k per position,
  tunable now and learned from the app's own history over ~5 years. **Start the clock now.**
- **D-E Standing bars (memory `seek-the-better-architecture-and-cut-ai-slop`).** Where a better
  architecture exists, explore the options before moving on; always hunt AI slop, meaning volume
  from repetition. Leads: 10 parallel L4 rubric files (~1,400 lines), branch-heavy files, Go
  files at the 400-line cap, frontend over the cap (817/747 lines). The codex slop catalog lacks
  the volume smell.

## 8. LEDGER STATE

- Nothing changed in `~/work/TheWarRoom` except this resume doc's commit (§3 of compact-safe),
  on `session/m1b-bash`, **not pushed**.
- New files: the two briefs above, the runner, both run dirs, the memory
  `seek-the-better-architecture-and-cut-ai-slop.md` (+ MEMORY.md line).
- Task list: one item added for the revived TheWarRoom planning work (see compact-safe report).

## 9. NEXT ACTIONS, IN ORDER

1. **Re-arm both monitors** (§2). Check that Bee is alive (`pgrep`, transcript mtime) and that
   Astra is still writing.
2. **Review Bee Sector 3** (`REPORT-3-evidence.md`) as it lands. Check: test pass/fail is
   separated into sandbox and code causes; the F.3 persistence question is answered; the
   rank-correlation decomposition of finding #1 is done properly. Send FEEDBACK-3 if thin.
3. **Review Astra sectors** as they land. Check: every source row has a URL fetched today;
   every number has a primary citation; the Madden 2026 endpoint is settled.
4. **Review Bee Sector 4.** Must contain: the 4a intent ledger; 4b doc drift; 4c slop hunt with
   measured short forms; 4d seams ranked for the core scope; 4e Bee's sequencing; 4f
   architecture options, including the blend versus alternatives.
5. **Triage** both runs into one findings set with my severities. Only the core scope plus the
   cleanup.
6. **Discussion.** Present the gaps to Christopher as an AskUserQuestion matrix (≤4 per screen,
   drafted options, recommended first). Expected topics: GM-first vs commissioner;
   ledger-mirrors-MFL vs system-of-record; who builds now that OpenCode/GLM/DeepSeek are
   retired; what must work before the Week-9 trade deadline; branch reconciliation; the
   architecture options from 4f; any new sources needing approval (Approved_Sources is locked).
7. **Write the plan** into the repo (`docs/`), plus a durable reasoning doc
   (memory `capture-session-reasoning-durably`). Order: cleanup → fresh MFL pull and
   reconciliation → crosswalk → each signal → the two measurables via the blend → M2. Commit
   only on a session branch; never main.

## 10. RELAY / ENVIRONMENT NOTES

- Go: `/usr/local/go/bin` (not on PATH). golangci-lint at `~/go/bin`.
- Hermes task list: `ssh hermes '~/work/command-center/bin/cc task ...'`. CT105: `ssh claudebox`.
- Never let subordinate agents touch git, the GUI, or `~/.config/TheWarRoom`. Prompts go in
  files, not chat. Reports to Christopher are brass tacks. Explain every internal label.
- Beelink IP: this repo's commits say .191 (eno1); `~/.claude/CLAUDE.md` says .190. Unresolved
  and not on the critical path.

## 11. HONEST STATUS

Mapping is about half done. The findings so far are verified, but Sector 3 (tests actually run)
and all of Astra's research are unproven. No plan exists yet. Expect several more hours of agent
time, then the discussion.
