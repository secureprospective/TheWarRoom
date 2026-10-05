# Legacy NFL History

<!-- impeccable:product-schema 1 -->

## Platform
web

## Stack
Confirmed: independent HTML/CSS/JavaScript PWA, Python standard-library archive compiler. No dependencies at runtime. Existing WarRoom application and databases remain untouched.

## Users
Christopher explores his 32-team dynasty league's history and wants to keep extending this tool.

## Product Purpose
Make fourteen archived MFL seasons understandable through a clickable twelve-month calendar, pick-trade heatmaps, league summaries, team histories and player histories.

## Operating Context
Read-only local archive in `league-archive/raw/`; latest-per-file manifest in `league-archive/manifest.jsonl`. PWA requires localhost or HTTPS. Deployment beyond this machine is undecided.

## Capabilities and Constraints
Calendar filters must connect to underlying events. Completed trades differ from proposals. Actual transaction date differs from MFL archive season. Snapshots are not complete ownership intervals. 2026 is partial. No credentials, owner contact details, message boards or raw archive served. Generated league data stays out of git.

## Evidence on Hand
1,705 JSON files, 2013–2026: 1,559 usable, 97 expected errors, 49 empty. Existing WarRoom UI supplies the navy, restrained data-interface visual identity; this tool does not redesign it.

## Product Principles
- Every result traces back to an archive file and record.
- Unknown and unavailable are not zero.
- Keep transactions, draft selections and roster snapshots distinct.
- Interpret league IDs as strings, preserving leading zeros.
