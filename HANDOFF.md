# Handoff

- Baton: Bee, 2026-11-17; War Room recent auction wins.
- State: source confirms `MarketBook.Bids.Auctions` retains decided winner ID, salary/years, and `DecidedAt`; replay folds these from the native ledger. Added server-only league-view `market.wins.read` and hosted rendering. No schema change; local desktop UI unchanged.
- Privacy: projection includes player/name, winning team, salary cents, years, and decision time only. Open/sealed bids, counts, points, and losing amounts are excluded.
- Tests: `go test -race ./internal/commissioner/market ./internal/leagueservice ./internal/serverhttp ./internal/commandcatalog`; frontend Vitest (122 files, 907 tests); `pnpm tsc --noEmit`; `make lint` all pass.
- PR #340 is open; CI checks passed. The pre-push `verify` hook passed.
- Rebase preserved #336 hosted target reads/writes; wins are a separate league-public fact.
- Next: review/merge PR #340.
- Blocked: none.
- Tried/rejected: extending `bids.read` would mix owner-only and open-auction data; dedicated allowlisted read keeps bid details out.
