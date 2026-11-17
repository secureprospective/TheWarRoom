# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #312 remains draft. Merged current `origin/main` (#316 Retry-After behavior) into this branch; the 3+3 timing runs are pending.
- Next: Run three sequential 180s `origin/main` benchmarks and three sequential 180s branch benchmarks with the `.scratch` harness; report reads/s, latency percentiles, CPU, peak RSS, 503s and errors. Update PR #312; mark ready only if errors are no worse and CPU improves. Do not merge.
- Blocked on: None currently; honor `heavy-ok.sh` before runs.
- Tried/rejected: Prior single-run measurements are insufficient for mean/spread; profiling logs are not substitutes for the required six timing runs.
