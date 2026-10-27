# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. Do not push without authorization.

## Where it stands
E1–E6 done. Edge scoring, sources, play definitions and RTL are documented in
`docs/architecture/default32-scoring.md`; code remains authoritative.

## Next move
Review the documentation and resolve the debt below.

## Debt left
- Unknown Sleeper IDs need identity and roster coverage; do not assume they are irrelevant.
- Held play bodies without a `sack` column can count red-zone sacks as attempts. Repin those
  bodies before requiring the column; requiring it now rejects pinned evidence.
- The red-zone boundary includes the 20-yard line; Christopher has not confirmed it.
