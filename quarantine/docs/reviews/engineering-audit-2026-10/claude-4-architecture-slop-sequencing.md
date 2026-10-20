# Sector 4 — intent vs reality, slop, architecture (Claude, 2026-10-03)

Bee was stopped after Sector 3 to save GPT budget (Christopher's call, 11:17). I wrote this
sector from the code on `session/m1b-bash` (`~/work/TheWarRoom`), Bee's Sectors 1–3 and
Astra's Sectors 1–2. Scope is the core only: the real-player measurable → M2.

## 4-F3 — "Start the clock": what is persisted today (answer to FEEDBACK-2 Part F.3)

**HIGH. The raw scouting inputs are fetched live, used once and discarded.**

- **Every table the app creates** (grep of `CREATE TABLE` in `internal/`):
  - calendar_events, cap_relief_ledger, contracts, contract_year_changes, contract_years
  - dead_cap_ledger, league_schedule_cache, param_defaults, param_overrides
  - player_status_events, rosters, rulebook_active, rulebook_overrides, rulebook_versions
  - schema_migrations, season_phases, season_scores, standings_cache, trade_notes
  - transaction_corrections, transaction_counts

  None of them holds a player directory, a scouting input or a production stat.
- **`season_scores`** (`internal/output/output.go:83`) stores the *effective* layer
  multipliers plus two raw values:
  - `film_raw`
  - `tb_ras` (the RAS value kept for tie-breaks)
- **Not stored anywhere:**
  - Breakout age, college share and school tier.
  - Coverage, the Madden ratings, combine measurements and draft capital.
  - Positions and the MFL player directory. These are held in an in-memory `normalize.Lookup`.
  - Snaps and any weekly production.
- **Consequence:** every week that passes without an input store is calibration data that cannot
  be rebuilt later. Historical nflverse files can backfill production, but not "what our scouting
  said about this player in October 2026".

## 4-A — Architecture: why the measurable is a nudge, and the options

### What the code does

`internal/engine/pipeline.go:32-41`:

```
ScoutingAdjusted = BasePoints × AgePull × L4.Combined
```

All 10 position rubrics (`internal/engine/l4/{offense,defense,kicker}`) end in the same line:

```
Combined = film × RAS × breakout
```

Each factor goes through an S-curve that is **capped by a constant**:
- Film: ±5% (±3% at DT and LB).
- RAS: ±8% (±4% at LB).
- Breakout: ±5%.

So L4 can never leave about **0.83–1.19, by construction**. The ±7% we observed is not a data
problem. The rubric caps are designed that way, and any player with BasePoints = 0 stays at 0.

The rubric comments call these caps "rubric constants — never admin-exposed (Hard Constraint)".
That **conflicts with North Star pillar 4** ("calibration through UI, not code") and with the
"learned later" requirement. **This needs Christopher's ruling.**

### Options

| | Shape | Rookie / injured | Two numbers | Grows into "learned" | Cost from today |
|---|---|---|---|---|---|
| **A. Today** | points × age × capped scouting multiplier | score 0 | no | no | — |
| **B. Credibility blend on one scale (recommended)** | Per position, put production and scouting on one scale: percentile, or z within position. `measurable = Z·production + (1−Z)·prior`, `Z = e/(e+k)` | prior carries them | yes: different recency, horizon and prior weight per number | yes: k, weights and curves become stored params, fitted from the clock data | new production layer + prior mapping + input store; L4 rubric logic becomes the prior |
| **C. Separate scores, combined only at M2** | Keep production and scouting apart; M2 weights them | rookie has only a scouting number; no per-player measurable | partly | weakly | small, but leaves the thesis unmet at player level |
| **D. Fitted model now** (empirical Bayes or a hierarchical model) | Fit priors and shrinkage from 2021–2025 nflverse history | prior carries them | yes | already learned | high: needs a modelling pipeline before anything ships; data gaps at IDP |

**Recommendation: B, built so D is a later swap of the parameter source, not a rewrite.**
- B is the shape Christopher described.
- It reuses the existing scouting assembly as the prior.
- On a percentile scale it is robust to positions with very different raw units.
- Its parameters (k, recency weights, prior weights, age curve) live in the param store, so
  "hand-set now, learned later" changes data, not code.

### The two numbers under B

- **On-field-now** = blend of current-season production rates and the scouting prior. Evidence is
  this season's snaps plus last season's, discounted. k is small, so production dominates quickly.
- **Dynasty** = on-field-now carried across a horizon by the position age curve, with a heavier
  prior weight and a slower recency discount. Contract and cap are applied after, as today's L5
  does, so asset value and cap value stay separable.
- **IDP constraint (Astra G3):** no free source gives routes, pass-rush snaps, run stops or
  coverage snaps. IDP production therefore means **defensive snaps + box-score rates per snap**:
  tackles, TFL, sacks, QB hits, PD, INT. No opportunity denominators. Write this into the design
  as a known limit.
- **Fantasy points** (MFL YTD) become one production input. They are no longer the base.

### Structural changes any option needs

1. **Score boards keyed by run, not by (season, config).**
   - Today: `season_scores` has PK `(season, scoring_config_id, mfl_id)` and triggers that forbid
     UPDATE and DELETE (AD-04). That is why `ScoreLeague` can only skip (S2-F7): a board can
     never be recomputed.
   - Fix: a `scoring_runs` row carries run_id, as_of, a params snapshot and a hash of the inputs.
     Scores are keyed by run_id. This keeps the append-only intent and makes recompute normal.
2. **Input store (the clock).**
   - Append-only rows of (source, season, week/as_of, player, field, value).
   - Plus a persisted player directory: MFL id, GSIS/PFR crosswalk ids, position, birthdate, draft.
   - Every fetch writes here first. The engine reads from here, never straight from the network.
   - This also fixes "the app forgets the league" (S2-F2), because MFL pulls land in it too.
3. **Rubric as data.** One engine function plus a per-position spec. See slop item S-1.

## 4-C — Slop hunt (core scope, measured)

| # | Where | Measure | Short form |
|---|---|---|---|
| S-1 | `internal/engine/l4/*` — 10 rubric structs | 1,540 lines (44% comment) + 1,929 test lines. Every `Apply` has the same shape: three S-curves, a weighted breakout composite, `Combined = film × RAS × breakout`. Only the constants and curves differ. | One `Rubric` func (~40 lines) + a `PositionSpec` table (10 rows of caps, steepness, weights, curves) held as params. One table-driven test replaces 10 parallel suites. Special cases (DT cushion, QB RAS-off, SL-021 blend) become spec fields. |
| S-2 | The same files | SL-0xx / AD-xx decision labels repeated in comments throughout. Each re-explains its spec doc inline (`wr.go`: 52 of 127 lines are comments). | Point to the spec once per position; keep only why-comments the code can't say. |
| S-3 | `internal/scouting/assembly` | 1,875 lines, 34% comment. `breakoutage.go` and `breakoutage_idp.go` repeat the same four functions (Build, fetch shares, derive, earliest); `offenseMaddenComposite` and `maddenComposite` are twins. | Collapse twins behind a position-family parameter. Measure in the build session before cutting. |
| S-4 | `internal/composition/defaults.go` | DT cushion literals 8.00 and 0.90 duplicate the stored params `cushion_guard.*`, which are therefore dead (S2-F8). | Read the param; delete the literal. |
| S-5 | Six unwired fetchers (S1-F2) | `nflproduction` and `kicking` point at 404 URLs (Astra S1). | Delete the dead two. Rewire `stats_player` through the input store. Keep `touchshare` (snaps) for production. |
| S-6 | `transactions_app.go` 44 branches, `contracts.go` 44, `rulebook.go` 38; 4 frontend files of 509–817 lines | Not inspected in depth: outside core scope. | Record as leads. Only touch them where the core path runs through. |

**Rule for the plan:** add "parallel near-identical units that differ only in constants" to
the project's slop catalogue (agent-codex §4), with S-1 as the worked example.

## 4-D — Seams on the core path, ranked

1. **MFL → state:** seed-once, no refresh (S2-F2). Everything downstream is stale until this works.
2. **Season source:** hard-coded `SeasonYear` vs the phase log (S2-F3, S3-F1).
3. **Crosswalk:** MFL id ↔ GSIS/PFR/ESPN through DynastyProcess. Every production and prior signal
   joins here; its match rate decides coverage. Not yet measured on live data.
4. **Sources → input store:** today each fetch → assembly → multiplier, in memory.
5. **Input store → measurable (blend).**
6. **Measurable → board:** recompute blocked by the PK and triggers (S2-F7).
7. **Board → M2:** M2 sums M1 AdjustedScore, then median/MAD z × 0.6 + MFL all-play × 0.4. M2 must
   choose which number it sums: on-field-now for "this season", dynasty for "the franchise".

## 4-E — Sequencing (proposal for the plan)

0. **One line:**
   - Merge origin/main and `session/m1b-bash`; fix the `ifaceguard` pre-push failure.
   - Bring CLAUDE.md, SYSTEM_MAP and Build_Tracker up to today. Retire the retired-tool workflow
     and record the Commissioner Suite plan as deferred.
   - Fix the empty app logs.
1. **Input store + player directory + scoring_runs.** Schema only, with tests. This starts the
   clock first.
2. **MFL refresh into the store.** Reconcile under Christopher's ledger ruling. One season source.
3. **Crosswalk**, with a measured match rate per position.
4. **Each signal into the store**, one at a time, each with a freshness check:
   - production: `stats_player`, snaps, injuries
   - priors: draft capital, combine, CFBD college, Madden per ruling
5. **Rubric-as-data refactor (S-1)**, done as a pure refactor with identical outputs, *before*
   the blend changes any numbers.
6. **The blend → two measurables**, with starting params from Astra S3. Evidence check: Spearman
   vs the old board, and a rookie and injured-veteran case set.
7. **M2 on the new numbers**, with recompute through `scoring_runs`.

## 4-B — Doc drift (core-relevant only)

- **CLAUDE.md:** workflow prescribes retired tools (OpenCode/GLM/DeepSeek) and CT105 paths.
- **SYSTEM_MAP:**
  - Built packages still tagged "[planned]".
  - IPC described as `Ping` only, when 24 methods exist (S1-F1, S1-F5).
- **Roadmap Phase 1 checklist:** never ticked.
- **Build_Tracker vs the CT105-only Commissioner Suite plan:** two sequences.
- **Rubric docs:** declare their caps a "Hard Constraint, never admin-exposed", against North
  Star's calibration-through-UI.
