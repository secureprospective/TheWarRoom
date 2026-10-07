# Saved lineup fixtures

- `league.json`: unchanged MFL league export from the ring-1 data run (2026-10-07).
- `roster-0025.json`: franchise 0025 selected unchanged from that run's `rosters.json`.
- `directory-0025.json`: id, position and name selected for that roster from the league-history
  run's archived `raw/players_2026.json` (2026-10-04 run). All 48 roster ids resolve.
- `players-0025.json`: the same directory fields for the 21 real saved week-5 starters, ordered
  as `internal/ingestion/livescoring/testdata/liveScoring-w5.json` lists them.

The directory is an earlier archive, not a live refresh. Saved starters and current roster are
from the ring-1 exports. No hand-built player positions are used.
