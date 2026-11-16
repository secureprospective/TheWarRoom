# Player values on the hosted server (F51): built, blocked on two gaps
**Date:** 2026-11-16
**Branch:** session/server-values (worktree `.worktrees/values`)
**Status:** experimental, don't merge as-is. The code is sound and tested, but the first real values pass came out
empty for two causes found and documented below. Do not create `/etc/twr/signals` on any server until both are closed.

## What changed
- **`-signals` server switch** (`cmd/twr-server/server_values.go`; launcher adds it when `/etc/twr/signals` exists,
  `deploy/server/twr-sandbox`). Five minutes after start and then daily: `sources.Service.LoadSignals` loads the
  nflverse feeds into the reference file, then writes player records from those facts with an empty listing
  (`noListing`, no MFL call). A background `appvalue.Preparation` prepares each seeded league's values and saves
  passes under `prepared-values/`. Requests read only completed passes.
- **Values reach the server's services:** `StoredHost.Values` (`leagueservice.ValueReader`) plus `NativeValues`
  pass-throughs on `dueHost`/`footballHost`; `serverhttp.Hosted.Values` set at the four `StoredHost` builders.
- **Per-league value revision** (panel mechanic F2): `appvalue.Source` gains an optional `LeagueRevision(ctx,
  league)` that, when set, replaces `Revision`. The server hashes the reference file plus that league's own file,
  so another league's commit no longer sends every league back to "preparing". `""` means every league
  (discovery). The desktop is unchanged: its `valueSource` is frozen by `tools/boundarycheck`, and the first
  attempt, which edited it, was refused at pre-push.
- **Odds never go blank** (panel engineer F6): `Service.oddsValues` gives labelled results-only odds when the host
  has no values, a pass is preparing, or a pass failed ("player values could not be read (…), so the odds come
  from results alone").
- **Waits for records** (found by the dry run): until player records exist, nothing is prepared and reads say
  "preparing". Before this the producer retried every second and logged the desktop's "MFL's players listing has
  not been read" each time (about 86,000 lines a day). Repeated failures are logged once.
- **Docs:** README "Player values" section (with the not-yet-usable status); `docs/architecture/desktop-modes.md`
  no longer says nflverse acquisition is deferred.
- `runHosted` stayed under funlen by moving live scoring and values into `startLeagueJobs`.

## Why
Christopher's order (2026-11-15): "Playoff odds and player values on the server", with his ruling that the server
collects its own inputs. Odds run "Reduced" (results only) until values exist.

## Verification
- Targeted tests green: `go test ./internal/appvalue` (new subtest "other league"), `./internal/leagueservice -run
  TestOdds`, `./cmd/twr-server -run 'TestValueRevisionIsTheLeaguesOwn|TestValuesWaitForPlayerRecords'`.
- Every new test was proven by a planted mutant that made it fail: global revision, missing failed-pass fallback,
  producer not waiting, read not gated. All four were restored.
- Commit hooks (golangci-lint full, secrets, ifaceguard) passed on both commits. The pre-push hook runs on push.
  The full suite runs in CI on a PR.
- **Dry run.** Fresh snapshot of CT 257 taken 06:41 CT with SQLite `VACUUM INTO` over a read-only connection, run
  under a systemd unit capped at 1 GiB, no swap, 200% CPU, `GOMEMLIMIT=768MiB`, `GOMAXPROCS=2`. Results:
  - **Download:** 23 files, 0 failed, about 70 s, 974,223 observations. Records written for 9,758 players.
  - **Values pass:** prepared about 8 s after the records.
  - **Memory:** anonymous peak 660 MiB at the end of the download (RSS 687). Idle before the download about
    480 MiB anonymous. No OOM.
  - **Reference file:** grew 21 → 265 MiB, with a 66 MiB WAL.
- **The pass was empty.** All 1,462 rostered players have 0 points, and every one reads "not in the players
  database (aggregate or unknown id)". The pass notes say "No NFL weekly statistics held for 2022…2026".

## Open items / what Claude or Christopher should check
1. **Weekly files are not archived on the server** (fix is in Tom's lane, small). The values engine reads
   weekly points only from archived weekly files (`history.HeldWeeks` → `readWeeklyFiles` →
   `ExactArchivedBodies`). The desktop archives every fetch by wrapping its client:
   `internal/composition/app.go:232`, `&archive.Transport{Base: Network, Sink: hist, Judge: ingestion.MFLRefusal}`.
   The server passes `Transport: k.client.Transport` (nil), so after the load `raw_archive` still holds only its
   earlier 15 rows. Fix: give the keeper an `archive.Transport` with `Sink` = the reference store. Check what
   `Judge` should be for nflverse.
2. **Records are unlisted, and the model values only listed records** (a decision for Christopher, with a
   recommendation). `internal/modelrun/modelrun.go:349` rejects any player with `!rec.Listed`. Listed means
   present in MFL's players listing. With `noListing` all 9,758 records are `listed=0`, even though every one has
   an MFL link in `player_external_id`. Options:
   - **(a)** The server makes MFL's players call once a day, as the original F51 plan and panel brief said. That
     is one call, and the allowance is per day.
   - **(b)** Count records resolved from nflverse as listed. That changes what "listed" means in the engine.
     These engines are protected, and per CLAUDE.md the call is Christopher's.

   Tom recommends (a) now, with (b) as a later ruling.
3. **Memory headroom.** 660 MiB of 1 GiB came during the download alone, on a pass that valued nobody. A real pass
   has to be measured after 1 and 2 are fixed. Proposed refusal gates from the panel: peak above about 800 MiB
   (under 20% headroom), any OOM, or live-data age over two poll intervals.
4. **Archive growth.** One load added about 245 MiB to the reference file (observations plus indexes), against
   2.9 GB free on CT 257. The panel says keep archives (the tables are append-only by trigger) and add a free-space
   floor before optional collection. Not built.
5. **Not addressed from the panel:** engineer F3 (decoded weekly cache keyed by SHA grows with each revision;
   bound it before a long soak), F7/Bender (say "college measures unavailable" on value-based odds rather than
   calling them full).
6. **Branch tracking:** `session/server-values` tracks `origin/main`. Push with
   `git push -u origin session/server-values`, never a bare `git push`.
7. **Dry-run kit** in `/tmp/f51/` (not durable): `rehearsal257.tgz` (the snapshot; it includes the test server's
   `accounts.db`, so delete it when done), `twr-server`, `sample.sh`, `samples.txt`, `server.log`,
   `samples-run1.txt`, `server-run1.log`, throwaway `tls.*`/`totp.key`, and `snaptool/` (a static read-only
   `VACUUM INTO` tool). To rerun:
   - Wipe and re-extract `data/`.
   - Start it with `systemd-run --user --unit=f51-rehearsal -p MemoryMax=1G -p MemorySwapMax=0 -p CPUQuota=200%`.
     The full command is in the RESUME.
   - Run `sample.sh` alongside it.
8. **Test-server keys.** CT 257's TLS key and authenticator key were printed into Tom's session transcript on
   11-16. They never left the Beelink. Christopher should rotate them.
