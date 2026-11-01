# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/INTEGRATE-1101. APP-VALUE-6 implemented; local commits only, no remote operations.

## Where it stands
- Profiled cold Source.Read: 19,143 ms; history Book/Features dominate, no per-player refits. Removed redundant latest-row SQL window before preparation: 18,078 ms, identical Bills values.
- Desktop lifetime owns a single background producer, discovering seeded league/seasons at startup and every second. Separate per-scope prepared passes; screen reads never run valuation. Refresh/roster/contract/settings/params/revision/as-of changes refuse stale reuse. In-flight commits prevent publication.
- Preparing reads return non-error state; only value regions show “Values are being prepared”. Existing season signal fills them when ready. Actual preparation failures remain errors. Shutdown cancels and joins before closing stores.
- Copied-profile actual API/workflow harness: cold-after-startup 49.167 ms, first workflow 40.269 ms (preparing); ready at 18,499.677 ms; first prepared/repeat workflows 33.489/31.025 ms. Bills 35/35, Best 21 = 243.26284592650194, rank 3/32 unchanged. DB opening excluded.
- PASS: exact imported 25-player parity, targeted Go race 56 top-level / 68 named outcomes across four packages, tsc, whole Vitest 159 files / 1,732 tests, plain make build; screen-computation mutant rejected. Placement/transport and targeted lint pass. No desktop launch/live profile/network use.
- Flow/contract: docs/architecture/value-preparation.md. Profile top ten and ignored harness/log evidence: .tmp/app-value6/. Handwritten production 1,960 → 2,151 lines; tests 467 → 753 versus f954d1f6. No reduction claimed.

## Next move / blocked on
- Background preparation still takes ~18.5 s; in-memory only, broad history revision invalidates on unrelated commits, multiple leagues serialize. Reads return preparing quickly, not completed values in 200 ms. Real desktop visuals and hosted-server valuation plumbing remain unverified/inherited debt.
- Full make lint blocked by inherited tiny-file mismatch: 26 both at f954d1f6 and now, baseline 25. Standalone boundary blocked by untouched migrationInputs fingerprint (other inherited migration/platform/binding mismatches also remain). Only task-owned resource adapters/source fingerprints reviewed and updated; no unrelated allowlist refresh or baseline increase.

## Tried and rejected
- Repeated per-player fitting/N+1 hypothesis: profile/source show one valuation and one history query per season, not per player.
- Query-only fix: ~1.1 s saved, nowhere near 200 ms. Kept the safe reduction, then moved production off screen reads.
- Cache-hit-only proof: harness also measures immediate post-startup/preparing reads and actual workflow reads after publication; no hidden warm-up counted as cold values.
