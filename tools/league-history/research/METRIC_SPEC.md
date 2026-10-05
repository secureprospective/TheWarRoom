# Proposed behavioral measure contracts

Research specification, not implemented functionality. Gate sequence and literature
triage: `BEHAVIOR_RESEARCH.md` / `SOURCES.md`.

## Common analytical frame

Each result must carry:

- **Subject:** franchise slot or verified GM-tenure ID; aliases are presentation only.
  Tenure maps include franchise, start/end, evidence, shared-manager/gap status.
- **Context:** execution-date interval, football season/phase, rules era, asset facets,
  verified ownership window and optional pre-event competitive state.
- **Population:** primary subject/cohort and a separately defined peer baseline.
  Default peers are other eligible subjects in the same era/phase/window. Do not
  collapse the peer population to one GM because a subject filter was applied.
- **Evidence policy:** MFL event type, two-sided/ambiguous/manual status, score/identity
  gates, handling of missing data, exclusions and exact source row references.
- **Quantity:** numerator, denominator/exposure, unit, distribution rather than only
  average where appropriate, metric version and archive revision.
- **Reliability:** observed coverage, censored boundaries, number of seasons/episodes,
  stability across periods. “Not enough evidence” is distinct from observed zero.
- **Inference target:** observed archive census or modeled repeatable/future tendency.
  Do not attach binomial confidence to the former as if incomplete records were sampled
  randomly. Modeled uncertainty must acknowledge clustered/repeated decisions.

A numeric tendency alone does not identify the reason for a decision.

## Subject-relative event ledger

A completed exchange involving A and B produces one deal and two analytical views:

- A sent: `franchise1_gave_up`; A received: `franchise2_gave_up`.
- B sent: `franchise2_gave_up`; B received: `franchise1_gave_up`.
- A–B relationship weight counts that deal once. League deals are not the sum of
  participant deals; that sum is twice the number of eligible two-participant deals.
- Each pick transfer has sender, receiver, raw token, round, reported draft year,
  original franchise when encoded, slot when known, execution timestamp, coverage
  class and identity-resolution status. FP and DP identities are not equated by guess.
- An empty encoded side is unknown economic consideration, not zero price. A manual
  record can inform observed custody without being a reliable negotiated exchange.
- Proposed, accepted, rejected, revoked and expired notices remain separate from
  completed TRADE records. Without validated offer linkage, do not construct an
  offer-to-completion funnel.

## First deep version: descriptive measures

| ID | Quantity / definition | Required context and denominator | Filter behavior / interpretation |
|---|---|---|---|
| B01 | Completed exchange participation: count of eligible distinct deal IDs involving subject | Same observed period and agreed exchange-completeness policy | Window, phase, partner, asset; records, not initiative |
| B02 | Participation rate: eligible deals / covered eligible days or finalized league weeks | Explicit exposure unit; no assumed closed-trading days until verified | Phase/week; separate counts from rate; season length/partial coverage cannot disappear |
| B03 | Relative activity: B01 divided by matched peer mean B01 | Same eligible context; peer mean based on all eligible peer subjects including true observed zeros | Reports activity lift, not trading skill; unavailable if peer mean is zero |
| B04 | Seasonal rhythm: subject share of deals in each phase/month, versus matched peers' distribution | Subject total and peer totals shown; draft-relative anchors separate from calendar months | Month heading selects comparable months; phase comparisons use real season anchors |
| B05 | Burst share: largest rolling seven-day eligible-deal count / subject's eligible deals in window | Covered dates and administrative/batch flags; fixed window definition | Indicates execution clustering, not how rapidly a GM thinks or negotiates |
| B06 | Received and sent picks: counts split by round, draft year/horizon and original franchise | Token-backed transfer records; unencoded/comment-only deals counted as exclusions | Direction, round, own-origin/other-origin; never equate count to value |
| B07 | Net pick flow: received minus sent, per round and horizon | Same B06 inclusion and paired sides | Signed heat scale centered at zero; positive means observed net receipts, not hoarding |
| B08 | Pick-in/out deal share: eligible deals with ≥1 received/sent pick / subject's eligible deals | Exchange policy and encoding era visible; mixed directions may overlap | Sums of incoming/outgoing shares need not be 100%; numerator/denominator clickable |
| B09 | First-round direction: received/sent R1 counts and deal shares | B06/B08 restricted to R1; slot uncertainty retained | Compare same era/phase; no imported “a first equals X” conversion |
| B10 | Horizon mix: distribution of received/sent draft years relative to verified current draft cycle | Actual draft cycle/phase, not source year blindly; unresolved years excluded | Show same-cycle, +1, +2, etc. as separate categories; report resolution coverage |
| B11 | Player-position acquisition/release mix | Received/sent player transfers with dated listed position; classify offense/IDP then position | Show per-asset shares and per-deal incidence separately; latest position must not overwrite history |
| B12 | Package shape: player count received versus sent; pick presence on each side | Known encoded player counts on both sides; consideration uncertainty separate | Examples: more players sent than received, player-for-pick; not “elite consolidation” without valuation |
| B13 | Partner breadth: number of distinct eligible counterparties | Window/phase/tenure and subject's deal count shown | Breadth measures coverage of observed relationships, not sociability |
| B14 | Top-partner share: max_j w_ij / sum_j w_ij | Eligible completed-deal weights; no offer or asset-count weights mixed in | Partner matrix cells open their events; no relationship motive inferred |
| B15 | Partner concentration: sum_j (w_ij / sum_j w_ij)^2 | At least one eligible deal; show observed partner count and number of deals | 1 means all observed deals with one partner; unavailable, not 0, with no deals |
| B16 | Effective partners: reciprocal of B15 | Same window, deal-count weights and policy | More interpretable than a personality index; compare with actual breadth |
| B17 | Draft selection mix: positions by round / verified selections in that class/window | Recorded drafts only; position at draft time and rules/class context | No early missing draft results interpreted as abstention; draft class can be selected |

### Example arithmetic (synthetic unit-test fixture)

Three verified A–B deals and one verified A–C deal give A four participant deals,
two distinct partners, top-partner share 3/4, concentration (3/4)^2+(1/4)^2=0.625,
and effective partners 1/0.625=1.6. These are not measured league findings.

If A sends two picks to B and receives one pick, A's flow is -1 and B's is +1.
The league's net flow is zero, with three individual transfers. Both A and B
participated in one deal; the league has one deal, not two.

## Second stage: weekly behavior and decision context

| ID | Quantity / definition | Eligibility and critical exclusions | Why it is not a psychological diagnosis |
|---|---|---|---|
| B18 | Acquire-to-use lag: eligible lineup weeks from verified receipt to first starter appearance | Verified possession, trade relative to lineup/NFL lock, bye/injury eligibility; censor if follow-up ends | Usage is observable; “they valued him for depth” is not automatically established |
| B19 | Acquisition utilization: acquired players started within specified eligible weeks / eligible acquired players | State the follow-up window, eligible player-week rule and censoring; link cases | Starting shows use, not whether acquiring was good or predictable |
| B20 | Release/retention/reacquisition episodes | Known entry/exit from directed moves, including audited manual events; distinguish roster from IR/taxi status | Incomplete episodes are left/right censored; no automatic sunk-cost inference |
| B21 | After-result activity: execution rate over a fixed post-finalization window following loss versus win | Finalized team results, comparable league weeks, matched exposure; disjoint/explicitly handled overlaps | “After loss” describes timing, not panic; commish entry can delay action |
| B22 | Pre-event competitive state: to-date record and all-play/points percentile | Only fully finalized earlier weeks; regular season/playoffs distinct; offseason state explicit | Never use eventual final standing/championship as information known at trade time |
| B23 | Hindsight lineup gap: feasible reported-optimal player points minus actual starter player points | Same point basis, legal lineup/source validation, adjustments separate; unverified home/optimal residuals quarantine | Knowing the week's best lineup afterward does not prove an avoidable decision error |
| B24 | Hindsight utilization ratio: actual feasible player-lineup points / feasible optimal player points | Only positive usable optimum; handle negative scoring and zero denominators explicitly | Ratios outside ordinary range demand inspection, not clipping into a flattering score |
| B25 | Week-to-week starter persistence/change | Comparable finalized eligible lineups; roster turnover/bye/injury overlays | Does not measure within-week edits or “over-tinkering,” which are not archived |

## Later inference or enrichment, not initial labels

### B26 — Draft-run response

Compare selecting a position after k recent selections at that position with comparable
pick opportunities. Control for remaining players, draft slot/round, rookie class,
roster needs and rules; one draft is a clustered sequence. Evidence of following a
run can reflect depletion/need, not irrational herding. S1 motivates the question,
not a fitted coefficient for this league.

### B27 — Stable versus changing tendency

Compare season-specific B06–B25 definitions under identical rules and coverage. Recent
versus earlier periods are explicit; show distributions and number of independent
periods. Shrink/withhold modeled estimates for sparse contexts, choose parameters on
historical training periods and examine later holdouts. A detected change could be
rules, a new owner, a rebuild, constraints or missing capture; it is not an owner detector.

### B28 — Partner preference beyond activity

Compare observed pair concentrations with an eligible, temporal network model preserving
participant activity within season/phase, excluding self deals and respecting tenure
eligibility. Repeated pair edges are allowed because this is a deal-count multigraph.
Do not compare everyone with 1/31 as if all partners were equally active/available.
Report sensitivity to grouping/null choice. Zero-variance or tiny ensembles yield
“not estimable,” not infinite abnormality. No collusion inference.

### B29 — Age/experience preference

Requires verified player birthdate/experience and position at transaction time. Position,
contract, production, rookie class and available market affect age distributions.
Show the age distribution actually acquired/released before claiming an adjusted
preference. Never infer age from a player ID or backfill current age into old trades.

### B30 — Revealed current/future pick exchange terms

Use sufficiently many verified bilateral pick-only trades with known round/year and
slot/slot-uncertainty, and no omitted cap/cash consideration. Local terms can be described
before a price curve is fitted. Fixed NFL or modern generic dynasty charts are not
historical ground truth. S3/S4 disagree about the correct utility objective; preserve that
choice rather than label all premiums mistakes.

### B31 — Retention after underperformance

Requires B20 custody, verified self-acquisition/manager tenure, production known at the
observation date, forecasts where available, injury/role/contract constraints and appropriate
comparison opportunities. “Holding after weak performance” is a descriptive conditional
pattern. Endowment effect, loss aversion and sunk cost are distinct hypotheses, not three
synonyms for it.

### B32 — Realized outcome after a decision

Report subsequent points actually started, later pick realization and tenure-compatible
results over explicit horizons. Censor unobserved futures and later resale, separate
outcomes from expectations and account for schedule luck. Equal-number observations
or adjustment variables alone do not establish causation. No automatic winner/loser
trade score with retrospectively selected values.

## Filters: first-class facets, not hidden calculations

### Eligibility facets

- Subject/verified tenure and comparison group.
- Date range, source season, calendar year, league phase and optional relative-to-draft days.
- Trade/waiver/manual/status-move channel; raw observed records versus vetted exchanges.
- Encoding availability, two-sided/ambiguous status and verified/unknown custody.
- Observation-only versus inference, complete versus censored follow-up.

### Behavioral facets

- Received / sent / net; pick / player / mixed package.
- Round, target draft cycle, horizon, own-origin versus other-origin, known slot range.
- Position/role and asset-count package shape.
- Counterparty and repeated-pair selection.
- Finalized pre-event state, post-win/post-loss window and actual usage.
- Later only: dated age/experience, cap/contract constraints and dated valuation source.

### Display/relationship controls

- Counts versus rates versus shares versus departure from baseline.
- Shared absolute scales or explicitly labelled independently normalized scales.
- Subject selection highlights peers rather than destroying the comparator.
- “Compare with league” keeps all non-subject context filters and excludes the subject
  from default peers. Filters defining a subject cohort do not silently redefine the
  baseline cohort; its policy is visible and editable.
- Filter chips, affected-view labels, undo, saved question/revision, evidence export.

Missing facet data is its own visible category. Selecting a known horizon must disclose
how many unresolved records left the analysis.

## Acceptance tests before implementation is called complete

1. Two participant views never double-count league deals; received/sent/net pick
   conservation reconciles for each fully resolved context.
2. Empty-side and unencoded-era fixtures never enter price/preference denominators
   as known zero consideration or known no-pick deals.
3. Proposals and acceptance notices never inflate completed-exchange counts; offer
   linkage remains unavailable unless explicitly validated.
4. Manual add/drop records update custody only under their validated interpretation;
   IR/taxi moves do not pretend to be sales or releases.
5. A future T placeholder with absent score is not a tie, completed week or pre-event result.
6. A trade before week finalization cannot use that week's eventual win, player scores
   or season-ending championship. Unknown finalization precision must be disclosed.
7. Early home/scoring residuals cannot inflate hindsight lineup-gap totals.
8. A current player-position change does not rewrite historical draft/trade position.
9. A tenure boundary/gap does not merge two owners into one behavioral biography.
10. Zero denominator, no observed events, missing coverage and censored outcome produce
    different states. No clipping/annualizing hides the distinctions.
11. A clicked calendar period, partner, position or flow facet produces matching subject
    counts, labelled peer baseline, receipts and CSV; each view's scope is testable.
12. Saved/reloaded questions include metric definition, evidence policy, full filter state,
    cohort/baseline and data revision—not just the route.
13. All conditional pattern statements show n, coverage, context and supporting events;
    unstable or filter-selected findings are not promoted to causal/personality claims.
14. Native keyboard controls, mobile evidence navigation and offline operation retain
    the existing PWA guarantees after the query model changes.

## Architecture consequences (bounded)

- Archive boundary: versioned raw parsing/validation with preserved evidence and metadata.
- Domain: directed transfers, weekly player/team observations, contextual anchors,
  tenure map and eligibility/uncertainty classes; don't thread untyped raw JSON into metrics.
- Analytical core: pure measure/query functions and invariant tests, reusable across views.
- UI: one explicit analysis frame, coordinated views and readable measure definitions.
- Browser delivery: indexed/season-partitioned data if necessary, measured worker-based
  aggregation only if profiling justifies it. Do not choose a framework or backend
  rewrite before the analytical contracts require one.
- Offline/security: keep generated private data out of git and maintain the serving
  allowlist; any new partitions/cache format get explicit revision/migration tests.

No new framework, machine-learning stack, GM personality classifier or external API
is needed to ship the first meaningful descriptive behavioral explorer.
