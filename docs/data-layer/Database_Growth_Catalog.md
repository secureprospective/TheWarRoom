# Database Growth Catalog

What will make the SQLite files hold up as real leagues fill them and reads become the bottleneck.
A living list: add an entry when a pressure is found, update it when measured. Never guess a
threshold; measure it with the big-league fixture (next session) against a read time budget.

Settings today (`internal/db/pools.go:24`): WAL, one writer and up to 10 readers per file,
`foreign_keys(ON)`, `recursive_triggers(ON)`, `synchronous(FULL)`. 51 production tables, 6 named indexes.

Measured 2026-10-22 on a copy of the real `history.db` (375 MB): 179 archived bodies, 460 MB
unzipped in total, the largest 97.95 MB (nflverse play-by-play 2025 CSV).

| # | Pressure | Symptom | How to measure | Trigger | Smallest fix | Status |
|---|---|---|---|---|---|---|
| D1 | Read speed | Every native read and command re-parses the whole league ledger (one event is 772 KB) | time a native read on the big-league fixture | a read over its budget | keep a folded state per league and fold only new events | open (store F12, partial) |
| D2 | Read speed | Scoring a starter re-parses every held copy of the season stats CSV (2026: 24 fetches, 9 distinct, growing) | count parses and time one scoring pass | a week's scoring over budget | parse each distinct copy once and key it by sha256 | open (F13, partial) |
| D3 | Read speed | Archive lookup is a substring scan over `fetch_log.url` | `EXPLAIN QUERY PLAN` on `archiveContainsQuery` | fallback reads over budget | match on an indexed source or URL key | open (F14, partial) |
| D4 | Growth | Every what-if preview writes a full league copy to `whatif.db` and never removes it | `whatif.db` size and row counts over a season | file growth per week | delete sessions nothing references | open (F15) |
| D5 | Integrity | Native scores depend on which archived copy of a stats file was read | replay a past week after removing one copy | any change in a past result | pin the copy each scored week used | open (F9) |
| D6 | Growth | `history.db` holds every fetch body for good (224 MB of archive) | `raw_archive` count and sum of `size` per month | file size or backup time over budget | a retention rule for bodies nothing pins | open (new) |
| D7 | Integrity | An archived body unzipped with no limit | — | — | `archive.Gunzip(body, size)`: the body must match its recorded size, with a 256 MiB ceiling | fixed on `session/hardening-method` (83070a6a); all 179 real rows match |
| D8 | Many leagues | One file serves every league; one writer per file | write wait time with several active leagues | writes queue past budget | decide one file vs one per league at the client/server panel | open (panel) |

Fixed earlier (in source on main `d3dabe7a`): F1–F8, F10, F11. See `docs/build-handoffs/Hardening_Plan_2026-10.md`.
