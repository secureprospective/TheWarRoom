# TheWarRoom — why the October 2026 core plan is shaped this way
**Written:** 2026-10-03, at the close of the engineering-audit session that revived the project.
**Companion to:** `Core_Build_Plan_2026-10.md` (what to build). This file is the why: the
reasoning, the rejected options and the doctrine. Read it before reopening any ruling in the plan.

## 1. The thesis, in Christopher's words

> "If we are unable to put a measurable on the real NFL players, then the rest is only
> assumptions on the application output, which every single other fantasy application does.
> We are different: the weights of the real NFL players dictate how the application responds
> to the application's user inputs."

> "If we do not have a solid core the rest is window dressing."

> "Excellence is the standard, not the goal."

Everything in the plan serves that thesis. Commissioner tools, the fantasy-points engine and the
Vision horizons wait, because without a real-player measurable they are built on assumptions.

## 2. What the audit found: the app inverted its own thesis

The engine computed `BasePoints × AgePull × L4.Combined`.
- `BasePoints` was last season's MFL fantasy points; a missing score became 0.
- `L4` was the real-player scouting, an S-curve product **capped by constants** at film ±5%,
  RAS ±8% and breakout ±5%. So it could never leave about 0.83–1.19.

Measured on the July development database:
- Spearman ρ between the final score and last season's points = 0.992.
- Removing scouting moved ranks a mean of 9.6 places (max 79 of 1,299).
- 14% of players, every rookie and anyone without 2025 points, scored exactly 0 whatever their
  scouting said.

**The fantasy points dictated; the real-player weights nudged.** That is the opposite of the
thesis. It was a design property (the caps), not a data bug.

Other facts that shaped the order of work:
- The app never refreshed from MFL after its first load.
- It had two sources for the season.
- Its scores table could not be recomputed: the key is `(season, config, mfl_id)` and the
  triggers forbid change.
- It stored none of its raw scouting inputs.
- Its Madden feed was the 2023 edition.
- Neither database on disk was a faithful league mirror; both carried test cycling.

The code itself was healthy: build, vet, race tests (926 pass, 0 fail) and lint were all green.
Evidence: `docs/reviews/engineering-audit-2026-10/`.

## 3. The blend, and why this shape

Christopher's direction:

> "Last year's points are important, but we need to build in a blending effect... rookies or
> injured players have no points last year, but rookies have a clearer scouting pattern as our
> application is new and not looking back... the blending effect needs to be a fluid part of the
> engine, because in 5 years we will have clear rookie scouting and current years played."

**Chosen: a credibility blend on a within-position percentile scale (option B).**

```
measurable = Z·production + (1−Z)·prior
Z          = e/(e+k)
e          = exposure-weighted snaps
```

- A rookie's Z is about 0, so the prior carries him.
- An injured veteran keeps his earlier seasons at a discount.
- A starter's production dominates.

The parameters are stored, not coded, so a fitted model (option D) can later replace the
parameter source without a rewrite.

**Rejected:**
- **A, today's multiplicative shape.** It produces zeros and is capped by construction.
- **C, separate production and scouting scores combined only at M2.** It never gives a player
  his own measurable, and leaves rookies with half a picture.
- **D now, a full fitted model first.** It is the strongest end state, but slowest to first
  use, and the IDP data gaps would stall it. B grows into D.

**Two numbers, kept separate (Christopher's choice):**
- On-field-now: how good the player is today.
- Dynasty: the long-term asset.

Mixing them in one score was part of the original confusion: age, athleticism and current
production were all multiplied together.

## 4. What the research changed (core-research run, Sector 3)

1. **k is fitted, not looked up.**
   - No published NFL position k exists.
   - Under a random-effects reading, `k = σ²/τ²`: per-snap noise over between-player variance.
     That is estimable from free nflverse 2021–2025 by split-half reliability.
   - So "hand-set now, learned later" became **"fitted from five free seasons now, refined from
     our own clock later"**. Only the prior's weight must wait for history we keep ourselves.
   - Don't mistake a study's inclusion cutoff (for example "250 pass-rush snaps") for k.
2. **Use is stable; efficiency is noise.**
   - WR targets per route run, year to year: R² 0.41. Yards per target: 0.08 (Stuart 2014).
   - Pressure rate r 0.72 vs sack rate 0.51 (Eager/PFF 2020).
   - Player-level coverage EPA is about 0 year to year (Eager & Chahrouri/PFF 2018).
   - So on-field-now is built from usage and opportunity, and CB/S production is shrunk hard.
3. **Draft capital is the only well-evidenced prior.**
   - Player-level R² is about 0.2–0.3 (Hadley et al., *Frontiers* 2025).
   - The combine is weak and inconsistent (Kuzmits & Adams 2008; Teramoto et al. 2016: RB
     10-yard split about 9%).
   - Breakout age is a WR hypothesis.
   - Madden has no evidence. That, plus a 2023-pinned feed, is why Madden left the core (R8).
4. **No flat 3% age decay.** Published peaks (QB ANY/A about 27, RB volume 26, WR 25–29) are
   survivorship-biased. Curves are fitted with imputation; the peaks are sanity checks only.
5. **IDP has a hard limit.** No free source gives pressures, routes, run stops or coverage
   snaps. IDP on-field-now = defensive snaps + box-score rates per snap. Say so in the UI; don't
   pretend otherwise.

Every number above is cited with its quote in the research run's `REPORT-3-evidence.md`, copied
to the evidence folder.

## 4a. The career and age arcs: how the research folds into the blend

Christopher asked whether every finding had been carried into how the blend moves along a
career and an age curve. On first pass it had not. This section is the complete accounting.

### The model in one picture

```
prior          = f(draft capital, combine, college)            pre-NFL facts only, fitted jointly
production_s   = per-component rates for past season s, age-adjusted to today along the talent arc
e              = effective count of exposure across seasons     (Σw)²/Σw², recency-weighted
on_field_now   = Z_now · production + (1 − Z_now) · prior       Z_now from within-season reliability
dynasty_start  = Z_dyn · production + (1 − Z_dyn) · prior       Z_dyn from season-pair stability
dynasty        = Σ_t discount_t · P(on field at t) · talent_arc(age+t, exp+t | dynasty_start)
```

### Two design errors this pass caught

1. **k measured against the league mean would under-weight scouting.**
   - Split-half reliability measures signal against the population mean. The blend shrinks
     toward a *player-specific* prior.
   - The right k uses the talent spread left after the prior: `τ²_resid = τ²(1 − R²_prior)`.
   - With draft capital's player-level R² of about 0.2–0.3 (E9a), k is about 1.25–1.4× larger
     than the naive value. That figure is illustrative: E9a's R² is for career value, not
     per-snap production, and Stage 6.1 measures the real R² per position. So the prior holds weight longer for well-scouted players.
   - This is also the answer to the question the research could not settle: "how many seasons
     does scouting keep adding information?" (E9f). The blend answers it per player, from the
     fitted k and how many snaps he has. It is not a calendar rule.
2. **Recency and age would have counted decline twice.**
   - Discounting old seasons *and* applying an age curve both push older players down.
   - Fix: age-adjust each past season to today along the talent arc first. Then recency
     measures only lost information (role, team and scheme change; E0.2, E7).

### One defect in today's code this pass found

**Age is counted twice today.** L3 applies `(1 − 0.03)^(age − peak)`, and 9 of the 10 L4 rubrics
also carry an age-trajectory sub-signal inside breakout (for example `wr.go`, weight 0.15). The
new design has age in one place only: the arcs.

### Every Astra finding and where it lands

| Finding | What it says | Where it lands |
|---|---|---|
| E0 | `k = σ²/τ²`; an inclusion cutoff is not k; YoY ≠ split-half; reliability vs the mean ≠ weight vs a prior; same target, same scale; injured weeks add no trials | Plan Stage 6.2 (k against the prior), 6.3 (two k's), percentile scale, injury rule in Stage 7 |
| E1 | QB passer-rating "stabilizes" at 50–110 attempts, but the method conflates α and R² | Not used for k; QB k fitted (Stage 6.2) |
| E2 | Pressure rate repeats (0.72) more than sack rate (0.51); 250 snaps is a filter | IDP: no free pressures, so QB hits are the nearest proxy; sacks get a larger k |
| E3 | Earning targets repeats (R² 0.41); efficiency after the target barely does (0.08) | k per component: opportunity vs efficiency (Stage 6.3) |
| E4 | Player-level coverage EPA is about 0 year to year | CB/S production heavily shrunk; large k expected |
| E5 | Tackle totals, split-half: LB 0.64–0.73, DB 0.46–0.57, DL 0.51–0.57; totals mix exposure and rate | Separate snap share (role) from per-snap rate; k per component |
| E6 | Kicker shrinkage beats raw; `1 − exp(−n/a)` is a competing form; distance context first | The shape of Z is tested (Stage 6.4); K by distance band |
| E7 | Shrinkage beats the naive current average; the population choice matters; effective count `(Σw)²/Σw²` | Pool within position or role; effective-count evidence (Stage 6.5) |
| E8a–c | Marcel 5/4/3 baseline; Year 2 outweighs Year 1 for young players; blog weights are a lead only | Recency fitted after age adjustment (6.5); experience term for the year-1→2 jump (6.6) |
| E9a | Draft capital: player-level R² about 0.2–0.3; it also predicts games and starts, i.e. opportunity | Core of the prior (6.1); also an input to the survival arc (6.7) |
| E9b | Combine weak overall; RB 10-yard split about 9%; WR vertical 3.7%; athleticism shifts aging | Small prior add-ons via the joint fit; athleticism × age tested in the talent arc (6.6) |
| E9c | TE college receptions and YPR add information beyond draft order | College production in the joint prior |
| E9d | WR breakout age: defined, not validated | Tested as a WR prior feature in the joint fit; kept only if the holdout supports it |
| E9e | No Madden evidence; consensus boards overlap with draft capital | Madden out (R8); one joint prior stops double-counting |
| E9f | Scouting's useful lifetime: could not establish | Answered structurally by k against the prior (above) |
| E10a | Survivorship bias; WAR-share curves include exits; linemen decline later | Talent arc with imputation (6.6); **exits become the survival arc (6.7)**: for dynasty, attrition is signal, not bias |
| E10b | QB ANY/A peaks about 27, with accelerating decline (−0.04, −0.14, −0.24) | Talent arc is curved, not a flat 3%; a sanity check for the fit |
| E10c | RB volume peaks at 26, then steady decline, understated by survivorship | Sanity check; the survival arc carries the RB cliff |
| E10d | WR best seasons at ages 25–29 (selected sample) | Sanity check |
| E10e | TE about 27, sensitive to elite survivors | Sanity check |
| E10f | Kicker curves that condition on career length can't forecast | Survival arc never conditions on the future (6.7) |
| E10g | No numeric IDP age curves | IDP arcs fitted from our own data only; flagged as such in the UI |
| Sector 2, G2/G5 | Injury designation ≠ games missed; data sources change in 2022 and 2023 | Survival arc uses observed snaps, not designations; fits mark source changes |

Three things remain unestablished even after fitting, and must stay labelled in the app:
- No IDP opportunity denominators (Sector 2, G3).
- The prior's weight for our *own* scouting history needs the clock: years of stored priors.
- No free source validates film grades.

## 5. Architecture doctrine from this session

- **Start the clock first.** A raw fetch archive (content-addressed, append-only) comes before
  any engine change. Every week without it is calibration data that can never be rebuilt:
  "what our scouting said about him in October" cannot be backfilled from anyone's files.
- **Runs, not slots.** Key score boards by a scoring run that carries its params snapshot. This
  keeps the append-only rule (AD-04) and makes recomputing ordinary. The old key made recompute
  impossible and silently ignored every param edit.
- **Rubric as data.** The 10 L4 rubric files were one routine copied with different constants:
  1,540 lines, 44% comments, plus 1,929 test lines. Refactor to one routine plus a settings
  table, **proved identical by a golden test before any number changes**. Then change numbers
  as separate, reported steps.
- **MFL wins.** Phase 1 is read-only against MFL. A refresh overwrites; what the app does
  locally is a what-if. A two-way ledger mid-season would have been the hardest thing in the
  app to get right, for no GM benefit.
- **The prior stays leak-free.** The old zero-leak rule survives for the scouting prior, which
  must never see fantasy points or volume. Otherwise it double-counts production and stops
  being independent evidence.

## 5a. Historical data: the measure dictionary (Christopher, 2026-10-03)

Christopher's requirement:

> "The data we save historically [must be] a readable indexing formula that is low
> maintenance and creates clean outputs for the application... maintainable just in case we
> lose or gain a data source and we need to rebalance the blend."

**Chosen: a measure dictionary.** Every number is keyed `player · season · week · measure`. The
measure is named for what it means (`opportunity.targets`), never for its source's column. A
mapping table (`source_fields`) is the only place a source's vocabulary lives.

Rejected:
- **One table per source.** Readable one source at a time. But every source change is a schema
  change plus blend edits, and a lost source leaves a dead table. That is the coupling which
  makes a source change expensive.
- **One generic table with each source's own field names.** Cheap to write, but unreadable, and
  the blend must learn every source's vocabulary. Source churn would still reach the engine.

The rules that make it low-maintenance (plan Stage 1):
- **Store facts and counts, never derived values.** A formula change never rewrites history.
- **One measure, one meaning.** Sources with different definitions never silently mix.
- **Corrections append with `as_of`.** Any past day's knowledge can be rebuilt, and fits can't
  peek at the future.
- **No row is dropped for a missing ID match.** Unmatched rows wait for the crosswalk.
- **The engine and UI read only the features layer.** That is where the "clean outputs" come
  from.
- **Registries are checked-in files.** The Measure Dictionary doc is generated from them, so it
  can't drift.

When a source dies (Christopher's ruling R11), the board keeps running on what still flows and
says so. A rebalance is prepared from history and shown next to the current board. It is
applied only on his approval. Param sets record the measure set they were fitted on, so the app
always knows when its parameters no longer match its data.

Why this suits the blend in particular:
- The blend's parameters are per measure (k per component, prior coefficients per input).
- Losing a source removes measures; it does not break the formula.
- Rebalancing is refitting on the same history with fewer measures. That is a run of the
  fitting tool, not a rewrite.

## 6. Christopher's standing bars (apply to every stage)

> "If there is a better way architecturally to do something we need to explore those options,
> and not just keep moving... the 'smarter' architecture is the only thing that will make this a
> successful build."

> "ALWAYS identify AI slop and clean it up, no excuses to see something like this example: 600
> if tags, when it could be done with 4 short lines."

The worked example from this codebase is the 10 rubric files (§5). The smell to catch is
**parallel, near-identical units that differ only in constants**, plus comment walls that
restate spec docs inline.

## 7. Process lessons (what this session paid for)

- **A database on disk is not the league.** The audit agent first called the release DB "the
  real league DB". Its own phase log showed test cycling and a 2027 rollover. Check provenance
  before trusting state.
- **Status docs lag the remote.** The snapshot's docs were three commits behind `origin/main`.
  Compare against the remote before calling anything current.
- **Web search summaries can be stale.** A generated answer said 2025/2026 injury data didn't
  exist; the maintainer's release metadata proved it did. Primary metadata wins.
- **Budget the subordinate runs.** Both GPT runs hit usage limits. Bee finished Sectors 1–3
  (inventory, flows, executed evidence) and Astra Sectors 1–3 (sources, gaps, research). Claude
  wrote both Sector 4s. Put the irreplaceable work (executed checks, cited research) first in a
  brief, and synthesis last, which the driver can do.
- **Retired tooling lingers in docs.** CLAUDE.md still prescribed OpenCode, GLM and DeepSeek
  two months after they were retired. Docs are part of the timeline cleanup, not an afterthought.
