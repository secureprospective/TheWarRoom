# Handoff

## Baton
Bee / APP-MERGE-4 — 2026-11-02, build/INTEGRATE-1102. Local only; no remote.

## Where it stands
- ROSTER-1 (`6dcf0224d`) merged with INTEGRATE-1102 (`5e38e4e2e`).
- Both follow settings retained; cutover still clears schedule.follow. Inline current
  week and delayed ten-minute played-week worker preserved; transactions run inside it.
- Live stats and Pulse files unchanged. Current assets use the existing logical
  reading instant; explicit historical reads keep their exact observation time.
- Final race PASS totals (tests + subtests): composition 847, commissioner 599,
  history 465 (4 opt-in skips), migration 33, mflsource 42, leaguerules 113.
  Discovery preceded batches. Migrationtest has no test files. No full suite.
- go build ./..., plain make build, shell Vitest (47 files / 332 tests) and tsc PASS.
- Resolutions, generated-file reasons and evidence: docs/build-handoffs/APP-MERGE-4.md.
  Prior work: APP-GATES-4.md, APP-SCHED-9-evidence.md, APP-ROSTER-12-evidence.md
  in docs/build-handoffs/.

## Next move / blocked on
Claude's full suite and independent review; disposable-profile worker/UI acceptance.
Christopher's 26 first-kickoff conflict rulings, older-import transaction-follow
activation and off-roster missing-contract jobs remain. Combined worker timing and
transaction activation independent of schedule following remain unproved.

## Tried and rejected
Broadly clamping historical projector time hid historical cuts/trades: reverted;
only current-reading callers advance to the logical reading instant. No rule
weakening, invented source dates, direct production ledger writes, worker timeout
increase or remote operation. Historical fixture observation dates are fixed;
the production pre-observation guard stays intact.
