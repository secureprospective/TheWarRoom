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
