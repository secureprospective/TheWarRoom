# Imported schedules use the league ledger

MFL's `schedule` read (no week filter) → validated `RawScheduleWeek` → importer-only
franchise mapping → `schedule.generate` with explicit fixtures → `schedule.publish`.
Both commands use the ordinary commissioner checks and history-store append path;
there is no source-dependent schedule or standings reader. A commissioner may also
choose these fixtures directly, instead of asking the generator for pairings.

The import batch holds creation, starting assets, answers and schedule publication.
A failed step rolls the entire batch back. Re-import of an existing league updates
only its schedule; it does not re-seed or replace its roster. A changed source
franchise directory is refused rather than silently assigning games to new teams.

## Existing leagues

There was no existing migration sync command. With the app closed, first make a
SQLite backup of the profile. Run the explicit one-off migration against that copy:

```sh
go run ./cmd/twr schedule-backfill \
  --db /path/to/copied/profile/history.db \
  --league native.example --mfl-league LEAGUE_NUMBER --season YEAR \
  --principal RECORDED_COMMISSIONER_ID --reason 'Carry the published schedule'
```

The destination must exist. The principal must pass normal commissioner
permissions. Backfill maps through the destination's recorded starting franchises,
checks the season, then atomically drafts and publishes only a changed schedule.
The first command fences the revision; concurrent league changes require retry.
An unchanged schedule writes nothing. No command writes to MFL.

## Shape and limits

- Weeks are consecutive from 1, bounded to 52; past weeks are not filtered out.
- Home/away flags and leading-zero franchise identities are preserved.
- Each game and each participating franchise occurs once per week. An odd number
  of games is valid. Nonparticipating franchises rest; more than one may rest.
- Empty weeks are retained in storage for unseeded playoffs, but are not shown as
  fabricated franchise byes in owner schedule readings.
- Weekly scores in the schedule are not imported as lineup/stat evidence. Existing
  imports contain neither historical weekly lineups nor decided game results.
  Standings must not treat absent scores as zero-point games.
- `week.read` additionally needs held NFL week dates. The supplied profile has only
  five distinct NFL weeks held, so its current-week selector still reports missing
  dates, although its schedule contains the real week-8 opponent.

## Evidence

`TestMFLShapedSchedule` exercises MFL JSON, odd matchup count, resting teams,
empty playoffs, malformed/duplicate/missing weeks and unknown franchise rejection.
History-store migration tests exercise new import, existing-league backfill,
re-import/re-sync no-ops, changed publications, authorization, season rejection,
rollback after drafting, and preserved starting assets.

Copied-profile proof: 17 weeks read, two ledger entries; repeat writes zero.
Bills week 8: away at New England Patriots. Matchup read returns that game with
no result because neither team has a recorded lineup. Standings: zero decided,
208 pending; displayed first three are Bills, Patriots, Jets, all 0–0—not played
standings. No score, lineup, result or missing NFL date is invented.
