# T708 canary report

- Done: seeded deterministic subset selection; `go test -race ./cmd/twr-canary` passed.
- Done: safety gates reject non-Lab and non-allowlisted/native IDs before server access; refusal unit tests pass.
- Done: Lab-only flag, token, allowlist, report, exit-code interface and plain-words README.
- Done: `make lint VERSION=dev COMMIT=canary BUILD_DATE=canary` passed (0 issues).
- Not met: hostedfixture/serverhttp end-to-end green/replay test and planted wrong-standings/lineup red test.
- Not met: real preview/execute coverage. Unsafe/unavailable payloads are explicitly skipped; this implementation is read-only and does not satisfy the requested execute behavior.
- Not met: refusal tests only test target and missing allowlist setup; MFL-id and no-write server assertions absent.
- Selected journeys: always 1,2,3,4,8,13,20 plus five seeded draws from non-N journeys. Read paths run when available; unresolved writes/read doors are skipped with reasons. N journeys 15,17,18,22 excluded from random pool.
- Files changed: `HANDOFF.md`, `REPORT.md`, `cmd/twr-canary/main.go`, `cmd/twr-canary/main_test.go`, `cmd/twr-canary/README.md`.
- `tidy-check.sh` reports unrelated existing scratch `desktop-1124` and unrelated fleet processes; left untouched.
