# APP-SCHED-4

## Acquisition finding (source traced; copied profile queried)

1. Signals already acquires nflverse weekly CSVs and the DynastyProcess crosswalk; livegame holds ESPN live/detail feeds. The complete play-by-play/Sleeper/advanced-rushing adapters also exist, but the complete suite is reachable through pinned football imports, not ordinary held statistics.
2. The copied profile has successful 2026 weekly-stat fetches (latest 2026-11-01 07:11 UTC), but no nflverse play-by-play fetch. Part 3 found 630 absent starter player-weeks, 462 ambiguous positions and no complete scores.
3. This is not a wrong-season or failed weekly fetch: ordinary `heldSeason` reads only weekly files, always marks play-by-play missing, and Signals never downloads play-by-play. Refreshing the existing weekly feed alone cannot complete those scores.

## Intended change

Launch/manual/watched sync acquires season-scoped weekly statistics and play-by-play through the existing archived transport and freshness windows, independent of league origin. Ordinary history readers attach validated held plays without changing scoring math, preserving unavailable evidence and revision provenance. Failed downloads leave the earlier successful archive intact. No manual player exceptions, reported-score substitution or scoring-setting changes.

Proof uses SQLite backup copies inside `.tmp/sched4`, never the installed profile. Initial copied history database: 461,660,160 bytes. No remote git contact.
