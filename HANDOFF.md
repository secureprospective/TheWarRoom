# Handoff

- Baton: Bee, 2026-11-17.
- State: pulse ADP/tier job stopped at its source gate. Server-held football facts and `draft.read` expose player identity/position/team/draft year and draft state, not ADP or tiers. `marketfeeds` ADP is local-app feed state; hosted league load transfers native events plus selected pinned-football evidence, not the mirror's market-feed archive.
- Next: before implementation, provide an approved hosted ADP feed/import and a held tier source or approved tier derivation. Then extend the existing `draft.read` path and board; no schema change made here.
- Blocked: Step 1 found no hosted ADP/tier data; brief forbids adding a feed or calling outside services. Feature tests not run; PR: N/A (stopped before implementation).
- Tried/rejected: inferring tiers or routing local `marketfeeds` data into hosted UI; neither is server-held evidence.
