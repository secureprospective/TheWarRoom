# Legacy NFL Behavior Explorer

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Independent HTML/CSS/JavaScript PWA and Python standard-library archive compiler. No runtime dependencies, external fonts or live MFL calls. Existing WarRoom application, databases and raw archive remain untouched.

## Users
Christopher explores the observed choices of managers/GMs in his 32-team dynasty league and wants an extensible evidence-first tool.

## Product Purpose
Compare observed franchise choices under shared calendar, entity, asset and context filters; inspect peer baselines, directed receipts, counterpart relationships, trajectories and guarded hindsight/usage observations. Preserve the fourteen-year January–December calendar as the principal work surface. No personality dashboard, causal winner score or NFL-player behavioral model.

## Operating Context
Read-only archive at `league-archive/raw/`, latest-per-file manifest at `league-archive/manifest.jsonl`. Local PWA server on loopback port 8765. Installation requires localhost or private HTTPS. Remote hosting and reboot autostart are not configured.

## Capabilities and Constraints
Seven linked views: Calendar, Compare, Franchise, Weekly, League, Players and Evidence. Default unflagged two-sided completed-trade policy; raw/ambiguous sensitivity modes. Contextual shares precede asset facets; counts, date rates and shares remain distinct. Optional explicit owner-tenure map; otherwise every subject is a franchise slot. Proposals/manual adjustments never become completed-trade denominators. Unknown data is not zero. Hindsight lineup gaps and observed starts are not ex-ante skill or eligibility-adjusted acquisition quality. 2026 is partial.

## Evidence on Hand
1,705 files across 2013–2026: 1,559 usable, 97 expected errors, 49 empty. 2,634 completed trade records; 1,725 structured-pick deals; 3,780 encoded transfers. Source-season encoding absent in 2013–2016. 4,215 of 5,590 scored team-weeks pass score/constraint reconciliation. Source capture timezone and true result-finalization clocks are not archived. No verified owner map currently supplied.

## Product Principles
- Every result states its scope, unit, eligibility, denominator and interpretation limit.
- Preserve subject-relative received/sent directions and source coordinates.
- Separate observations, proxies, missingness and modeled inference.
- Never infer owner continuity from franchise aliases or snapshots.
- Do not backfill current values or final standings as decision-time beliefs.
- No credentials, owner contacts, messages or raw archive exposed; generated private data stays out of git.
- Keep WarRoom's cold-navy, flat, scan-oriented data-interface identity.
