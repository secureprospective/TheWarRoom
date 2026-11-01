# Handoff

## Baton
Bee / APP-GATES-2 — 2026-11-01, build/GATES-1. Local only; do not contact remote.

## Where it stands
- `0c23461c` merges the two rule helpers; `make bloat` passes 25/25 and `go test -race ./internal/leaguerules/...` passed.
- `make placement` still fails: schedule following reads league origin and frontend native forks are 20/19. `make boundary` still fails first at `startValuePreparation`; no fingerprints were refreshed.
- Full frontend run: 1747 passed, 4 failed in `leagueScreens.test.tsx`. First failure timed out rendering every rule form; subsequent tests got empty hosts. This suggests a timeout and cascading state leak, not yet proven or fixed. Previous run had 19 failures across 3 files.
- Preset guard archive absent in this worktree: not reproducible. `make lint` stopped at placement after ifaceguard, filelen and bloat.
- `heavy-ok.sh` stopped: disk 78%, available RAM 11670 MB, swap 94%, three matching race processes (one actual race command plus wrappers). No full race suite or build was run; build is also a heavy job. Cache `.golangci-cache/` was already untracked at start.

## Next move / blocked on
- Route schedule following through a real league setting/accessor, not source or an allowlist exception, and replace the new frontend fork with the shared path. Verify cutover behavior and native leagues.
- Review boundary differences against their introducing commits and fix unintended policy before refreshing any intended wiring fingerprints. Investigate frontend timeout/cascade; prove full run zero failures.
- Wait for the other race job and swap pressure to clear, rerun heavy-ok, then one full race suite and plain `make build`. Do not bypass hooks or gates.

## Tried and rejected
- Raising ratchets or refreshing fingerprints without review would hide violations.
- Assuming the frontend failures are fixed because isolated tests pass would not explain the full-run cascade.
