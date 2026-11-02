# Handoff

## Baton
ClaudeBox / review — 2026-11-01, build/SCHED-1. APP-SCHED-7 measurement only at 1207f2da.
Local docs/evidence commit only; no remote contact or push.

## Where it stands
- Fresh scratch-profile SQLite backup; ordinary archived lineup/schedule backfill, scoring acquisition,
  stat ledger, native results/standings: 148 recorded weeks, 24 exact / 0 within .01 / 9 differ /
  115 unavailable. Evidence: `docs/build-handoffs/APP-SCHED-7-evidence.md` and its two TSVs.
- Previous part 5 TSV is actually 4 exact / 29 differ / 115 unavailable; previous prose 4/33/111
  was an arithmetic error. Completeness did not regress.
- Bills MFL 6–1 / PF 1912.35; native 0/7 complete, W-L/PF unavailable (displayed zeros not outcomes).
- 472 absent rows: 36 bye candidates (historical affiliation unverified), 53 exact-week injury Out,
  141 positive snap/absent scoring-row, 242 unresolved. Seven separate 0835 identity gaps.
- Disposable measurement PASS 208.44 s; eight native player-item readings race PASS 33.82 s.
  Initial full measurement race run interrupted at external 600 s deadline during adoption;
  non-race retry resumed idempotently on copy. No production changes, GUI or independent review.

## Next move / blocked on
- Position queue integration remains DEFERRED; do not start without head-brain authorization.
- Resolve 242 unknown absences and historical affiliation for 36 bye candidates; existing MFL
  bye and nflverse exact-week Out/snap sources are documented, no did-not-play zero built.
- Nine complete score differences: largest Giants invalid-lineup penalty; remaining player credit
  residuals need MFL stat breakdown, including Schultz W4. No scoring-rule diagnosis established.
- Earlier debt: verified identity 0835; slow adoption/native reads; live/advanced inputs;
  source-health UI; 76 historical-roster refusals.

## Tried and rejected
- Missing weekly row or missing snaps as proof of inactivity/zero: unsupported.
- Current MFL injury status as historical nonparticipation: not week-specific.
- Treating weekly StatApprove as durable player-position approval remains rejected.
ClaudeBox / review — 2026-11-01, build/SCHED-1, APP-SCHED-9.
Local commits only; no upstream configured, no push.

## Where it stands
- Through db6b642bc: reconciled full-snap/no-event zeros + dated depth membership; schedule
  bye zeros; partialLineupAllowed YES maps to ordinary penalty none. No scoring math/forks.
- Same 148 rows: 61 exact / 1 within .01 / 15 differ / 71 unavailable (was 45/1/13/89).
  55 no-snap + 32 bye zeros newly admitted. 198 player-weeks remain unavailable (was 285).
- Childress 0835 pending: no source-ID bridge; same-name IDs belong to canonical MFL 17372.
- Bills MFL 6–1 / PF 1912.35; native still 0/7 complete, W-L/PF unavailable.
- Evidence + full ledgers: docs/build-handoffs/APP-SCHED-9-evidence.md and matching TSVs.
- Race: 319 top-level PASS / 9 opt-in SKIP across source/history/league/settings packages.
  Final copied replay PASS 169.18 s, byte-identical results/standings, ledger idempotent 10478.
  Plain make build PASS. Commit hooks PASS. No independent review or GUI evidence.
- make lint blocked by inherited tiny_files 26 > 25. Separate source-branch and boundary
  gates also fail on unchanged starting-ref files; details in evidence. No gate weakened.

## Next move / blocked on
- Review the new zero admission and partial-lineup translation; live-check imported partial
  lineups and queue evidence on a disposable profile. Existing ledgers need an explicit ordinary
  setting change; importer does not silently rewrite them.
- Need historical membership / stronger snap completeness evidence for remaining gaps; no
  identity guess or current-team zero. Bills Gonzalez W1–6 still lacks dated membership.
- 15 complete residual differences, secondary adapters, pinned absence evidence, rare stat
  fallback completeness, whole-season cost, 76 backfill refusals and inherited gates remain.
- Pre-existing untracked sched8 measurement test/cache untouched; own one-use tests removed.

## Tried and rejected
- Name-only merge of 0835 into 17372: external IDs already owned by another canonical player.
- Rounded snap shares alone: partial files can fit adjacent totals; now require unique totals.
- Zero for missing roster rows, current affiliations, absent partial-schedule entries or
  conflicting team games. Those stay pending.
