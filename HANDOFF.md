# Handoff

- Baton: Bee, lane `joinonce`, 2026-11-19.
- State: Rebased on `origin/bundle/b9` (`78670acd8`). One successful owner claim now consumes its join link through the unique digest-derived request key on the seat event; no schema migration. Sequential, second-seat and concurrent regressions pass. Disabled-marker mutant is killed. `make test`, Vitest (153 files/1,026 tests), and TypeScript pass on this base.
- Next: push `lane/joinonce` (pre-push gate), open one PR into `bundle/b9`, then verify CI.
- Blocked on: none.
- Tried and rejected: physical join-link-table migration; the existing commissioner event/request index provides atomic single-use state without changing schema.
