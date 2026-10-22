> **Adopted copy** of `christopher-coding-standards/docs/architecture-review.md` @ c5cc559. Change the master, then copy it back.

# Architecture, data-flow and preserving-debloat review

A task-scoped extension of the [quality contract](quality-contract.md), especially Q4/Q5/Q6. Review whether the design does the requested job clearly and economically, with verifiable contracts and effects. **Nodes are real code responsibilities/data-flow components—not agent workflow stages.** This method suggests changes; it neither rewrites code nor grants new authority.

## 1. Capture the initial relevant slice

For new modules or material cross-component/trust/state changes, record intended behavior, preserved guarantees and a small flow before implementation. Reuse an existing slice for a small fix; no new diagram for every function. Include only components/dependencies needed to understand this task.

A node can be a function/module, existing service/store, external system or meaningful boundary. Use natural project granularity; do not create a class, file, adapter or layer to satisfy a drawing.

| Record concisely | Why it matters |
|---|---|
| Responsibility and real symbol/path when known | Identifies the actual job/owner, not a made-up component. |
| Input → output contract | Meaning, important invariants and necessary transformations; not every field. |
| State/effects and ownership | Reads/writes/calls/resources; explicit none for pure logic. |
| Trust/failure boundary | Where shape/domain/permission checks and safe failure handling belong. |
| Evidence and open assumptions | Intent, source-traced, exercised or unknown; verification still needed. |

Edges describe what data/control crosses, where and under which material condition. Include relevant failure routes and HTTP/event/queue/store/external boundaries. Record ordering, transaction, lifetime, retry or cancellation assumptions when they affect correctness. Feedback loops and multiple coordinated actors may be legitimate; no forced DAG, purity law or universal layer scheme.

## 2. Trace actual implementation independently

Use the affected source, entrypoints, callers, state writes and meaningful tests—not just the author's map or a function name. Follow transformations/defaults/validation, mutation ownership and resource lifetime across boundaries. Dynamic callbacks, generated integrations and external contracts may need targeted evidence beyond static search; no direct-call grep match is not dead-code proof.

Keep these distinct:

| Level | What it establishes / does not establish |
|---|---|
| **Intent** | A proposed design/guarantee. Does not prove implementation or execution. |
| **Source-traced** | Inspected route/contracts/effects at named source locations. Does not prove every runtime condition or test result. |
| **Exercised** | Named test/runtime evidence, matching the candidate and relevant path/assertions. A test file, typecheck or green skipped job alone is insufficient. |
| **Unknown** | Reachability, contract, effect or behavior not established. State what smallest additional evidence would resolve it. |

Ask:
- Do required routes and failure/recovery guarantees exist? Which unexpected edges/effects are reachable?
- Does each value retain its intended meaning across parsing/conversion/defaulting? Are caller/callee assumptions compatible?
- Who owns writes and decisions? Is required atomicity/ordering/cancellation preserved under relevant concurrent changes?
- Is trusted identity actually trusted; does parsing validate shape while authorization checks allowed behavior? Are command/SQL/path/rendering and external-response boundaries safe?
- Does each node/transformation contribute a real job/guarantee, or only forward/copy the same thing?
- Do tests observe success, rejection, failure and preserved state where material, rather than mock away the behavior?

Compare actual and initial flows. A mismatch can be a defect **or a better design**. Preserve required behavior, explain a material change and update current intent/rationale; do not restore pointless scaffolding to match a stale map. Nor can redrawing a map authorize a new data disclosure or other unapproved risk.

## 3. Return preserving debloat suggestions

Look for repeated business rules with competing owners, purposeless forwarding layers/config knobs, redundant representations/data movement, mixed responsibilities or tiny-file fragmentation, obsolete scaffolding/history comments and weak/duplicate tests. Establish actual callers/compatibility before labelling something unused.

Necessary protocol adapters, domain types, error context, observability and useful tests are not bloat by default. Similar syntax may hide different guarantees; don't consolidate it blindly. Prioritize unsafe/incorrect routes, then high-confidence maintenance reduction, then optional polish.

Each meaningful suggestion can be a few lines:

```text
Finding/location: concrete node/edge/source and unnecessary cost or defect.
Proposal: smallest coherent simplification, no unrelated cleanup.
Preserve: behavior/API/state/security/error/compatibility guarantees.
Evidence: inspected callers/effects; exercised versus unknown routes.
Verify/risk: targeted preservation checks, dynamic/timing/ownership hazards.
Impact: measured or explicitly estimated/unmeasured counts and clarity/navigation change.
```

A valid no-change result needs no invented finding. Apply only authorized in-scope changes, one coherent hypothesis at a time; rerun affected verification/review. Material behavior/risk/scope changes require existing owner decisions, not an automatic rewrite approval hidden in this method.

## 4. Measure honestly, never optimize a quota

Compare the complete affected handwritten production file set, including added replacement helpers. Use the same formatting, encoding/newline and counting convention on both snapshots. A relocation is not a reduction unless the full set shrinks. Keep test/documentation counts separate; deleting tests or useful diagnostics is not evidence of improvement.

A simple convention is physical lines (`splitlines`), Unicode code-point totals including LF, and optionally UTF-8 bytes reported **separately**. State the convention; don't change normalization/semantics to manufacture savings. Exclude generated/vendor/lock output from handwritten source counts without ignoring compatibility, integrity or security obligations.

Actual reduction requires measured snapshots; estimates remain estimates. Report fewer responsibilities/representations/navigation hops alongside counts when established. Smaller but harder-to-read or behavior-changing code fails. Clearer safer code may grow. No size quota, slop score or custom measurement service.

## 5. Keep useful evidence small and current

Retain a current slice in project design docs when it helps future work, with significant prior rationale in history. Don't duplicate the whole system map in every prompt or pretend hand-maintained maps cannot drift. An existing source trace can be reused only while its relevant candidate/contracts remain applicable.

Return concrete findings/suggestions, material unknowns and the smallest required verification; distinguish source observation from executed behavior. A tiny fix may need one flow sentence, while a cross-boundary change may merit a small node/edge table. Use the project's adopted review/delegation/acceptance process—no new service, extra mandatory agent, repeated approval or whole-repo audit implied.

### Illustration, not a project skeleton

```text
request + trusted identity → parse command → permitted state transition → persist → safe reply
invalid/denied/invalid-state → safe rejection, no unauthorized mutation
persistence failure → meaningful failure, no false success
```

Review which real component owns permission/transition/persistence and whether the contract needs atomicity against concurrent updates. A diagram proves none of that. A pure quote/formatting task may need no store at all; don't copy this cancellation-like flow into unrelated work.
