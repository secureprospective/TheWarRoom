# Schedule reference refresh

MFL's `schedule` read (no week filter) → validated `RawScheduleWeek` → importer-only
franchise mapping → `schedule.generate` with explicit fixtures → `schedule.publish`.
Both commands use ordinary commissioner checks and the history-store append path;
there is no source-dependent schedule or standings reader. Commissioners may also
choose these fixtures directly instead of asking the generator for pairings.

The import batch holds creation, starting assets, answers and schedule publication.
A failed step rolls the entire batch back. Re-import of an existing league updates
only its schedule; it does not replace its roster. Changed source identities are
refused rather than silently assigning games to new teams.

## Automatic acquisition

The background launch worker refreshes reference schedules even when the ordinary
roster refresh is not due. Manual refresh and watched roster refresh also acquire
schedules under the existing refresh lock. The importer resolves its recorded
league identity and original importing principal; normal ledger permissions still
apply. Cut-over leagues no longer follow MFL. A failed read/publication logs the
failure, preserves the held schedule and retries at the next refresh.

`syncLeagueSchedules` → `mflsource.FollowingLeague` → validated source fetch →
`BackfillSchedule` → atomic ordinary schedule commands. Unchanged schedules write
nothing. No human runs a command. `twr schedule-backfill` remains available for
operators using an explicit destination, principal and reason.

All published leagues, regardless of origin, also acquire missing NFL week dates
through the existing `nflSchedule` source. The archive transport saves these bodies;
the ordinary local `ScheduleDates` reader selects them. Dates already held are not
fetched again. No kickoff or current week is hard-coded.

## Shape and remaining limit

- Weeks are consecutive from 1, bounded to 52; past weeks are not filtered out.
- Home/away flags and leading-zero franchise identities are preserved.
- Each game and participating franchise occurs once per week. Odd game counts and
  multiple resting franchises are valid.
- Empty playoff weeks remain empty, not fabricated franchise byes.
- Historical results are NOT imported. `NativeResults`/`NativeStandings` compute
  from `ResultSources`: recorded lineups and NFL-stat-derived player scores. There
  is no finalized franchise-score import port in that path. `ScoreFranchise` and
  `ScorePlayer` are deltas, not replacement totals; standings overrides are not
  game results. A reported-score evidence contract needs review before extending
  durable storage; do not fabricate NFL statistics or use standings overrides.

## Evidence (APP-SCHED-2)

`TestLaunchAutomaticallyBackfillsSchedule` exercises the launch worker with roster
refresh not due, repeated no-op publication and failed-read preservation.
`TestBackgroundReferenceSyncDoesNotRequireLaunchRefreshDue` tests ordering;
`TestFollowingLeagueUsesRecordedImportIdentity` tests identity selection.
`TestScheduleRefreshFillsOnlyMissingNFLDates` proves the same acquisition for a
non-imported league, no repeated date fetches and current week 8 on November 1.

SQLite-backup copy proof, with the launch reference worker and no CLI:
- Original NFL weeks: 7, 8, 9, 13, 14. Missing: 1–6, 10–12, 15–17.
- After launch: all 17 weeks held; commissioner entries 2 → 4; repeat stays 4.
- Bills week 8: away at New England Patriots; current week 8.
- App standings still have 0 decided / 208 pending, all teams 0–0 with zero PF.
- Read-only MFL `weeklyResults`: 16 completed games each in weeks 1–7, none later.
  MFL Bills: 6–1, 1912.35 PF; app Bills: 0–0, 0 PF. This comparison FAILS.

Proof output: `.tmp/sched2/profile-proof.log` (ignored local artifact). No GUI or
real profile writes. Full task acceptance remains blocked on historical results.
