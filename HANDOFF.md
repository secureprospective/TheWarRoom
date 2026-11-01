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
