# Handoff

- Baton: Bee, 2026-11-17.
- State: PR #322 is open and unmerged. Restored the shallow outer-slice clone and the unchanged caller-isolation regression from `main`; added a concurrent reader/writer `-race` regression. Replaced `clonePlayerWeek` in canonical Sleeper assembly with shared local read-only weeks; outward reads still deep-copy. `go test -race ./internal/store/history/...` and `make lint` pass. PR body includes all callers, mutation status, profiles, and four local runs: baseline VmHWM 620,556 / 1,064,196 kB; candidate 676,272 / 686,916 kB. `maps.clone` fell from 31.10 MB to 0.51 MB in the endpoint profiles; RSS mean fell 19.1%, with substantial run variance.
- Next: Push this change if it is not yet published; then wait for PR review. Keep PR #322 unmerged.
- Blocked on: Nothing. Private benchmark artifacts remain under `.scratch/pulse-smem-1117/bench/`.
- Tried and rejected: Streamed workflow replies did not remove the profiled `encoding/json.Marshal` output copy; reverted. The first baseline run had suppressed VmHWM output and was excluded.
