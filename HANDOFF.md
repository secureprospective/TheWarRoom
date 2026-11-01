# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-2 partially delivered;
local commits only, no remote contact.

## Where it stands
- 52e48a62: schedules refresh automatically at launch (including when roster refresh is not due), manual refresh and watched sync. Ordinary commissioner checks/publication; original recorded principal; importer-only identity mapping. CLI retained but unnecessary.
- a82f05af: all published leagues acquire missing NFL dates from the existing archived NFL source. No source branches in engines/readers, no hard-coded dates.
- SQLite-backup copy under .tmp/sched2/profile: launch reference worker, no CLI. Bills week 8 away at NE; current week 8. Ledger 2 → 4 entries; repeat stays 4. Original dates 7,8,9,13,14; now all 1–17 held.
- Final complete race batches: 68 top-level / 141 named PASS (control, migration packages, selected composition). Broader composition batch timed out at 280s; 236 named PASS, one failure: FrozenDoorMetadata schedule schemas. Overlay of a08bc291 reproduces it. No gates weakened.
- Plain make build passed again after launch-fetch deduplication. No GUI or real-profile writes.
- Whole affected handwritten Go files vs a08bc291 (physical lines): production 538 → 656; tests 167 → 350. No reduction claimed.

## Next move / blocked on
- Item 2 NOT implemented. Read-only MFL shows 112 completed games in weeks 1–7; app still 0 decided / 208 pending. App top five all 0–0/PF 0. MFL Bills 6–1/PF 1912.35: comparison FAILS.
- Existing NativeResults/NativeStandings compute via ResultSources (ledger lineups + NFL-stat-derived player scores). No finalized franchise-total import port found; ScorePlayer/ScoreFranchise are deltas, standings overrides are not results.
- ClaudeBox must review the smallest generic reported-score evidence/storage adapter into ResultSources, including real lineups, authoritative totals, corrections, season isolation and retry semantics. Mapping stays in importer. Do not fabricate NFL stats or silently add an imported-engine branch.
- Relevant proof/logs: .tmp/sched2/profile-proof.log, targeted-race.jsonl, final-small-race.jsonl, baseline-metadata.log, final-build.log. Harness removed after use.

## Tried and rejected
- Standings overrides or zeroed/fabricated player statistics to force historical totals: not real results and bypass shared scoring semantics.
- Unreviewed durable reported-score event format: outside the brief's wire-only constraint; needs the head brain's storage-contract decision.
