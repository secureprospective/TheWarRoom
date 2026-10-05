# From archive browser to behavioral analysis

Research/triage completed 2026-10-05; implementation subsequently approved by
Christopher. The archive-supported behavior explorer is now implemented; see
`METRIC_SPEC.md` implementation status and `../README.md` for exact scope. Research
itself was read-only; subsequent implementation leaves WarRoom, databases and raw
archive untouched. Christopher clarified that the subjects are
league managers/GMs. Source references S1–S14 are in `SOURCES.md`; proposed metric
contracts are in `METRIC_SPEC.md`.

## Recommendation

Rewire the app around **a manager's decisions in context, compared with the league's
norm for that same context, with every claim linked to its evidence**. Do not bolt
personality scores onto the existing event-count dashboard.

A result should read like:

> In this selected period and phase, this subject received future first-round picks
> in x of n eligible exchanges. Peers did so in a of b exchanges. This is consistent
> across k observed seasons; these omitted/ambiguous records limit the comparison.
> Open the actual exchanges.

The symbols above are a result template, not fabricated local findings. Before an
owner-tenure map is supplied, subjects must be **franchise slots**, not named GMs.

## Where the first version stopped short

1. It asks “which events exist?” rather than “which repeatable decisions differ from
   the league?” Event counts and histories are ingredients, not behavioral analysis.
2. It counts picks without retaining a subject-relative analytical frame: received
   versus sent, round, horizon, original franchise, asset/package context and omissions.
3. It treats filters primarily as record retrieval; they do not expose opportunity,
   matched peers, rates, distribution or change over time.
4. It leaves useful archived weekly results, starters, bench players, optimal lineups
   and score adjustments out of the behavioral dataset.
5. It does not resolve historical managers, deal completeness or event timing precision.
6. It exposes early pick-count zeros without a prominent era-level warning that the
   encoding is absent. A sparse export is not a passive-manager finding.
7. Its calendar/ledger selection and other report scopes are intentionally separate.
   That makes sense for browsing, but breaks a coherent behavioral question.

## Research triage

| Subject | What the literature supports | Decision for this PWA |
|---|---|---|
| Draft behavior (S1) | Choice sequence and competitors' preceding choices can matter; draft conditions matter | Measure position/round distributions first; later test run-following conditional on available pool, class and rules |
| Planning and skill (S2) | Repeated fantasy performance/decisions contain signal despite noisy outcomes | Separate decision tendencies, persistence and outcomes; do not equate championships with causal skill |
| Pick horizon and pricing (S3) | Current/future capital and clean exchanges are analytically distinct | Directional round/horizon flows now; local price estimation only after completeness/sample gates |
| Strategic objective (S4, S6) | Value depends on the utility function, lineup constraints and needs | Contextual comparisons; no universal trade-winner score |
| Retention/attachment (S5, S14) | Acquisition history is a useful hypothesis, not direct evidence of motive | Observable reacquisition/retention first; defer sunk-cost interpretations |
| Partner networks (S7) | Degree, strength and concentration are different quantities | Partner matrix and concentration with denominators; no collusion/friendship diagnosis |
| Visual analysis (S8, S9) | Linked filters, comparable views, history and provenance support reasoning | One shared analysis context, persistent baseline, evidence drill-through and saved questions |
| Statistical reliability (S10) | Small-sample proportions need appropriate assumptions | Counts first; stability/coverage visible; model intervals only for stated inferential targets |
| Manager-analysis products (S11) | A dossier is a useful organizational precedent | Adopt dossier structure, not undocumented Manager DNA formulas |
| Trading and winning (S12, S13) | Vendor observational studies motivate timing/net-flow questions | Test locally; reject causal prescriptions and imported Sleeper averages |

**Important disagreement:** surplus-value research can disfavor trading up, while the
utility-function critique can justify elite-player-seeking under a different objective.
That is why this app should expose the trade-off rather than stamp “irrational” on a GM.

## Archive audit: consequences, not assumptions

A separate read-only raw-data audit was performed. Its detailed findings and actual
league coverage table stay outside the public repository. No individual manager
profiles or private owner information were published.

- **Identity:** league exports supply franchise IDs but no owner-tenure history.
  A franchise slot can have multiple owners; aliases, start/end dates and evidence
  must be supplied/verified separately. Never invent fourteen-year GM continuity.
- **Pick encoding:** the early trade exports have no FP/DP tokens. The useful fourteen-year
  event calendar needs a visible metric-specific coverage layer, while structured
  pick-flow comparisons start only in supported periods. Later encoding does not
  prove all economic consideration is present.
- **Bilateral completeness:** some TRADE rows contain an empty encoded side. These may
  have omitted financial consideration, administrative explanations or other missing
  context. They are not automatically free acquisitions or negotiated exchanges.
  Keep them discoverable and exclude/segregate them from explanatory exchange metrics
  until reviewed. The strongest early calendar spike is affected by this problem.
- **Actor/timing:** a commissioner-entry flag dominates completed trade records. It
  identifies entry/execution, not necessarily initiation or decision time. No requester
  identity or stable offer-link ID was found. Acceptance rate, response speed and
  negotiation style are not defensible from naive proposal/completion division.
- **Manual movements:** LOAD_ROSTERS contains add/drop-style payloads the first compiler
  did not interpret. They matter for custody validation, but must remain a separate
  actor/context class rather than automatically count as discretionary waiver decisions.
- **Unused rich source:** allWeeklyResults contains starters/nonstarters, individual
  scores, reported optimal lineups and team scores. Candidate rows must be validated
  before utilization metrics; end-of-season standings alone are not enough.
- **Future placeholders:** future weekly results can contain T result codes with no
  score. A T is not sufficient evidence of a completed tie or an available week.
- **Scoring reconciliation:** in two early years, some home rows differ from summed
  starter scores by three points after the explicit adjustment field. This is consistent
  with an additional home scoring component, but its rule/formula has not been verified.
  Do not turn those residuals into GM lineup mistakes. Optimal totals also need a common
  scoring basis.
- **Drafts:** recorded selections and draft-start anchors are absent early. “Did not
  draft” and “draft not archived” must be separate. Do not use January 1 as a fake
  draft-relative reference date.
- **Player attributes:** directory exports provide IDs/names/positions/NFL teams, not
  validated historical birthdates or contemporaneous trade-market values. Youth preference
  needs enrichment; latest listed position is not necessarily position at transaction time.
- **Contracts:** cap levels change by season. Salary/adjustment sources help, but a latest
  salary snapshot is not automatically the salary known at an earlier transaction.
  Constraint-aware trade conclusions need a separate time-consistency audit.

## What the filters should let Christopher ask

The whole application should support questions, not just categories. Initial examples:

1. **Who receives first-round picks?** Select eligible era, R1, incoming direction,
   future horizon and season phase; compare subjects against matched peers.
2. **When do they spend them?** Flip direction to outgoing and compare before/after
   the league's actual draft date, not a fixed “rookie season” month.
3. **Are they broad traders or concentrated partners?** Select the same dates and
   asset context; compare distinct partners, top-partner share and effective partners.
4. **Does a loss change their activity?** Compare their execution rate after completed
   losses versus wins in equivalent league weeks, with eligible exposure shown.
5. **Do acquired players actually get used?** Follow incoming players into subsequent
   eligible starter lists; separate injured/bye weeks and incomplete follow-up.
6. **What is their draft behavior?** Filter rookie class, round, position group and
   available-pool conditions; compare class-specific choices, not generic Zero-RB labels.
7. **Do they rotate or retain?** Inspect validated acquisitions, releases, reacquisitions
   and unresolved intervals; distinguish ordinary IR/taxi moves from leaving a roster.
8. **Has their pattern changed?** Compare rolling periods within verified tenure, with
   identical coverage/rules/context. A possible change point is not proof of owner change.
9. **Is this league-wide or subject-specific?** Toggle the matched league baseline;
   keep its denominator visible instead of recalculating it down to the selected subject.
10. **Can I trust this result?** Open excluded-record counts, boundary/censoring notes,
    data coverage and exact source rows before acting on a tendency.

These questions are proposed analytical contracts. No live manager findings have
been claimed by this research pass.

## Rewire the information architecture

### League laboratory

Landing surface: the original twelve-month calendar, now with a selectable behavior
measure and a clearly labelled coverage/comparison layer. Display **counts, rates,
share or departure from the matched norm** as distinct units. Raw counts remain one
option, not the sole lens. For flows, show received/sent/net separately; do not feed
signed net flows into the old positive-only logarithmic heat scale.

Calendar selection controls coordinated peer distributions, asset flow summaries,
partner matrix and evidence ledger. Highlight selection without silently removing
context from every view. On small screens, views stack with the active question and
baseline visible; no unlabelled filter scopes hidden in tabs.

### Subject dossier

At first a franchise dossier; later a GM-tenure dossier. Sections: measured tendencies,
calendar/phase rhythm, incoming/outgoing asset mix, partners, usage/roster cycles,
change through time, observed outcomes and supporting events. Statements are
proportional and qualified, not a radar chart of invented personality traits.

### Compare

Two or more subjects or verified tenures on the same scales. Matched peer distribution,
medians and selected subjects; context includes rules/era, period, phase and observed
competitive state. Show trade totals separately from partner concentration and
positions selected separately from positions available.

### Paths and evidence

A selected pick/player can lead through validated transfers, draft realization and
subsequent usage. Broken identity links remain visible as unresolved; do not draw
continuous ownership over missing events. Each result can expose exact inclusion/
exclusion policy, source rows and exportable supporting records.

### Saved questions

Examples: “future-first receivers during the draft window” or “activity after losses.”
Save all filters, measure definition version, peer baseline and data revision. A saved
question reloads the same analysis, not merely the current page name.

## Implementation sequence

### Gate 0 — Identity, evidence and temporal contract

Exact first step: produce a **metric eligibility/coverage table plus subject-relative
transfer ledger**. Preserve commissioner flag, raw side identity, ambiguous consideration,
manual moves, source season, execution time and capture-time precision. Define a
local manager-tenure mapping format; do not require that map to browse franchises.

Pass only when cash/empty-side records, early unencoded periods, future T placeholders,
manual moves and missing draft anchors are distinguished in fixtures and real records.
No behavioral labels ship before this gate.

### Gate 1 — Behavior explorer with defensible descriptive measures

Ship incoming/outgoing pick flows by round/horizon, phase-normalized participation,
partner concentration, player-count package shapes and era-aware positional choices.
One query state drives calendar, comparison and receipts. Immediate denominators,
peer baseline, exclusions and copy naming each quantity are mandatory.

Pass when conservation identities reconcile, peer baseline cannot accidentally become
the subject's own filtered sample, and changing a facet updates all linked views
and exported evidence consistently. Subject-specific private outputs stay local.

### Gate 2 — Weekly context, usage and response patterns

Bring in weekly results, score/lineup validation, draft-relative anchors and validated
player custody. Only finalized weeks feed pre-event context; execution during a week
cannot see its eventual outcome. Quarantine unresolved scoring bases rather than
manufacture hindsight “errors.”

Pass with hand-traced acquisition-to-use examples, winning/losing exposure controls,
future/censored-week fixtures and score-adjustment reconciliation. Keep hindsight
lineup utilization separate from ex-ante decision quality.

### Gate 3 — Stability and conditional patterns

Measure whether tendencies persist across seasons/tenures or change. Inspect repeated
partners against a time/activity-preserving comparison model. For any modeled tendency,
report the assumption, number of independent periods, sensitivity and held-out behavior.
Use within-year normalization so changing league rules/activity do not masquerade as
individual preference. Do not publish automatic significance after arbitrary filter fishing.

### Gate 4 — Valuation, constraints and outcome evaluation

Only after verified owner tenure, dated attributes/values and cap/contract rules can
the tool explore age preference, economic willingness to exchange current/future capital,
retention after underperformance and strategy-conditioned realized outcomes. Outcomes
are descriptive associations unless a credible causal design is separately established.
The ranking engine's current values must not be backfilled as historical market beliefs.

## Deliberately not shipping

- GM acceptance probabilities, time-to-response or “initiates trades” from this export.
- Collusion/friendship judgments from partner concentration.
- Stubbornness, panic, irrationality or personality typing from an action pattern.
- A generic calculator's hindsight winner/loser label as behavioral ground truth.
- “More trades wins championships,” or “selling picks is always good.”
- Three-point unverified scoring adjustments counted as lineup mistakes.
- A complete fourteen-year manager biography built from one franchise ID.

## Decisions still requiring Christopher

- A historical owner/GM-to-franchise tenure map, including gaps and co-managers, if
  individual-manager attribution is required. Otherwise the first deep version must
  explicitly analyze franchise slots.
- Verified trade-window/deadline history and treatment of exceptional/manual transfers.
  Imported rules/events are used when available; gaps remain gaps.
- Dated NFL age/experience and value sources before age/value-sensitive metrics.

The approved archive-supported implementation delivers Gate 0 eligibility/identity
plumbing, the Gate 1 coordinated explorer, and explicitly qualified Gate 2 observations.
Annual splits and associations are descriptive, not Gate 3 fitted stability or Gate 4
valuation/causality. The original research rationale remains above; the implementation
status in `METRIC_SPEC.md` distinguishes shipped subsets from the remaining roadmap.
