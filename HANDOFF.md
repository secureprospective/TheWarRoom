# Handoff

- Baton: Bee, 2026-11-17; War Room recent auction wins.
- State: source confirms `MarketBook.Bids.Auctions` retains decided winner ID, salary/years, and `DecidedAt`; replay folds these from the native ledger. Added server-only league-view `market.wins.read`, an allowlisted completed-win projection, and hosted rendering. No schema change; local desktop UI path unchanged.
- Privacy: output includes only player/name, winning team, salary cents, years, decision time. Open/sealed bids, counts, points, and losing amounts are excluded.
- Tests: market, leagueservice, serverhttp and commandcatalog race tests passed (commandcatalog rerun after catalog adjustment); frontend 122 files / 906 tests and `tsc` pass; `make lint` pass.
- State update: rebased onto #336, preserving its hosted target reads/writes and adding wins as a separate league-public fact.
- Next: rerun impacted checks on the rebased tree, push/open PR and confirm CI green.
- Blocked: none.
- Tried/rejected: extending `bids.read` would carry a mixed owner/open-auction payload; dedicated allowlisted read keeps bid details out.
