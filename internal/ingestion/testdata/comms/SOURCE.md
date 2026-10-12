# Comms fixture provenance

Copied without network access from the existing league archive on 2026-10-11:
- `raw/2026/messageBoard.json` → `messageBoard-2026.json`: a genuinely empty object.
- `raw/2026/polls.json` → `polls-2026.json`: a genuinely empty object.
- `raw/2013/messageBoard.json` → `messageBoard-2013.json`: an authentication error, not archived posts.
- `raw/2026/players.json` → deterministic gzip `players-2026.json.gz`: player names, positions and teams.
- `raw/2026/rosters.json` → `rosters-2026.json`: current salaries and contract expiration years.

The feed tests reuse the existing real 2026 `transactions/testdata/transactions.json` and league
fixtures. Neither transactions export carries historical salary/contract fields. Current roster
terms are therefore explicitly labelled as held roster terms, never claimed as the transaction's
original contract. Constructed non-empty thread/poll shapes are test inputs, not captured exports;
unknown non-empty board shapes fail as unavailable instead of becoming an empty board.
