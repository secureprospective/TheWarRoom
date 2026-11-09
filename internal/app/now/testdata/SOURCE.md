# Live feed captures

Unmodified public HTTP bodies fetched with curl during MNF, 2026-11-02 ~19:50 CT.
Stored as deterministic gzip (mtime 0); hashes below are of decompressed bodies.

| File | URL | HTTP | Bytes | SHA256 |
|---|---|---|---|---|
| sleeper-week8-live.json.gz | https://api.sleeper.com/v1/stats/nfl/regular/2026/8 | 200 | 485272 | e81d8e270d48f05425b2613760442772bbf6c23e453faf9f8b1aad67f05c2ff2 |
| espn-week8-live.json.gz | https://site.api.espn.com/apis/site/v2/sports/football/nfl/scoreboard | 200 | 234792 | 8b9a00a0aab6b849b8ac7d1bb2d901b7ea366595ddeb0ed067fafb9ffaee67b5 |

ESPN: regular season 2026, week 8, 14 events; event 401873036 in progress,
Q2 15:00. Sleeper: 2,080 keyed rows; includes negative rushing averages.
The old Sleeper URL without /v1 returned HTTP 404 (206-byte HTML).
