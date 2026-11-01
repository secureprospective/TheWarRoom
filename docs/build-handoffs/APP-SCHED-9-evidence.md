# APP-SCHED-9

## Intended slice / findings

1. Missing-row admission currently cannot use complete snap files and dated depth membership.
   Decode/validate full snap rows, require internally reconciled offense/defense/special-team
   totals for both game teams, join dated held depth snapshots by GSIS, then admit only through
   existing final-PBP/event-conflict gates. Missing/partial files and unknown identities stay
   pending. Archive store owns reading/fingerprints; source owns pure coverage checks; existing
   held revision, feed and queue retain provenance. No scoring math changes.
2. Schedule byes need exact-week affiliation. Use held schedule bye declarations and dated depth
   membership; never current player-directory teams. Same recorded-zero/queue path.
3. 0835 has only its MFL external ID. Inspect held crosswalk/source IDs; do not join by name alone.
4. Imported lineup penalty inherits `zero_top_starter` from the base despite explicit partial
   lineups allowed. Preserve the setting-driven scoring path; translate the held setting, not
   league identity. Tests must show explicit permission overrides the base and unsupported values
   do not invent a penalty.

Verification planned: test rejection first, targeted race batches, plain `make build`, copied
SQLite profile's ordinary feed/results/standings, same 148-row cohort. No production writes,
independent reviewer or GUI evidence claimed. Existing untracked harness/cache remain untouched.

Starting ref: 48bacfac. `git pull --ff-only` refused: branch has no upstream.

## Item 3 — identity blocked, evidence-backed no change

Held DynastyProcess crosswalk has **zero rows for MFL 0835/835**. The canonical directory
has only provider=mfl/value=0835 for that player; no GSIS, ESPN, Sleeper or PFR bridge.
The same-name row is **MFL 17372**, GSIS `00-0040044`, ESPN `4685841`, PFR `ChilZi00`,
Sleeper missing. Those three external IDs already belong to the distinct canonical player
for MFL 17372. Historical depth also carries that GSIS/ESPN pair, not MFL 0835.

Exact missing link: authoritative evidence that league-local **0835 aliases 17372** (or one
of its three external IDs), followed by the normal identity merge/link flow. Neither a name
match nor DAL affiliation authorizes merging two canonical players. All seven 0835 weeks
remain pending; no zero or identity mutation made. Existing identity-gap zero-rejection tests
pass in the targeted source batch. Source-traced crosswalk guard rejects reused/misattributed
MFL IDs; no guard was weakened.
