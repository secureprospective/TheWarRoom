# APP-ROSTER-12

## Step 1 — held kickoffs

Used the existing provider, MFL's **public, league-independent NFL schedule**:
`nflschedule.Fetch(ctx, client, "2026", week)` → `mfl.Client.Do` (api host,
no L/APIKEY) → `archive.Transport` → `history.Store.Record` on a SQLite
backup-API copy of the authorized scratch profile. This is the same path used by
`fetchLineupWeek` / `fillScheduleDates`; ScheduleCache only reads the held bodies.
No new provider, owner page, remote git, live configuration or source DB write.
All six requests succeeded November 2, 2026. Each parsed and validated, returned
the requested week, and graded through `nflschedule.ToWeek`.

Public response bytes are retained in `APP-ROSTER-12-schedules/`, with requested
URL, fetch time, SHA256 and size in `index.tsv`; no league/private response is
included. These are evidence, not hardcoded application dates.

| Week | Games | First kickoff UTC |
| --- | ---: | --- |
| 1 | 16 | 2026-09-10T00:20:00Z |
| 2 | 16 | 2026-09-18T00:15:00Z |
| 3 | 16 | 2026-09-25T00:15:00Z |
| 4 | 16 | 2026-10-02T00:15:00Z |
| 5 | 15 | 2026-10-09T00:15:00Z |
| 6 | 14 | 2026-10-16T00:15:00Z |

Verification: test discovery followed by targeted `-race` schedule fetch/cache
checks: **11 test/subtest PASS**, zero FAIL/SKIP; plain `make build` passed.
Launch gate passed before heavy work. Logs in ignored `.tmp/roster12/`.
Historical classification and performance work follow separately.
