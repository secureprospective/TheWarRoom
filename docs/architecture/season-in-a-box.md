# Season in a box (T711 spike)

**Status:** spike, 2026-11-22. Test: `TestSeasonInABoxSpike` in `cmd/twr/season_in_a_box_test.go`.
**Goal:** a whole league week, played in seconds, offline, from a recorded real NFL game, so a regression in the
create -> lineup -> lock -> live scores -> final -> correction -> standings -> recap chain fails a test instead of a Sunday.

## What the spike does today

Run: `go test ./cmd/twr -run TestSeasonInABoxSpike -count=1 -v`.

Run time (`time go test ./cmd/twr -run TestSeasonInABoxSpike -count=1`): 3.9 s wall with a warm build cache; the test
itself 2.8 s; 6.5 s on the first run after an edit. The 5-minute budget is not close.

| # | Journey step | How it runs | Status |
|---|---|---|---|
| 1 | Create native league | `hostedfixture.CreateWorkflowLeague` (32 franchises, rules, schedule, pinned football), then `twr ledger` over HTTPS | reached |
| 2 | Quick draft | none | not reached (B3) |
| 3 | Set lineups | `twr preview` of `commish.lineup.set` through the real token and dispatch path; recorded through the commissioner's own session (G2 needs a human confirmation). Only franchise.01 sets one: see B5 | reached (franchise.01 only) |
| 4 | Pass lineup lock | fake clock advances 25 h past kickoff; a lineup that adds a player is refused ("locked at kickoff") | reached |
| 5 | Feed live stat lines in pieces, scores move | recorded ESPN summaries (`cmd/twr-server/testdata/live-week9.tar.gz`, frames 01 then 02) published to `history.LiveOverlay`; `results.read` read twice over HTTPS; franchise.01 goes 30.000 -> 60.500 provisional | reached |
| 6 | Final | `PinOfficialStats` on the synthetic 2026 week-1 files in `cmd/twr/testdata/official/` (no network); `results.read` shows franchise.01 final, no provisional (23.5 points) | reached |
| 7 | One stat correction | second pin of `stats_player_week_2026_corrected.csv.gz` (QB passing yards 250 -> 262); franchise.01 moves 23.5 -> 24.1. The starter's revision (the move off the last live line) is NOT REACHED, see B6 | reached (points); NOT REACHED (starter revision) |
| 8 | Standings | `standings.read` answers with status "Ranked" (not "Needs a ruling"). 0 of 16 games decided: the one pinned game waits for franchise.04's lineup. See B5 | answered; NOT REACHED (decided games, B5) |
| 9 | Report / recap read | `results.read` answers week 1 (16 games, 32 franchise scores). The recap text has no hosted read: NOT REACHED (B2 recap) | report answered; NOT REACHED (recap) |

Values in the test log are the engine's fixed-point numbers (10,000 per point): `235000` is 23.5 points.

## Real versus faked

Real: the hosted HTTP handler (`serverhttp.Hosted`, TLS, token scope, workflow door), the `twr` cobra commands and
`serverclient.Agent`, the commissioner ledger and every rule check, native week scoring
(`commissioner.NativeResultsReading`, `history.nativeSources`), the live overlay, SQLite.

Faked: the process boundary (the server runs in the test process on `httptest`, not as the `twr-server` binary); the
clock (`Store.SetClock`; account and due-worker clocks stay real); the NFL feed (recorded frames, no network); the
commissioner's G2 confirmation (the test calls `ExecuteCommissioner` as the seated principal); NFL player identity
(the box football evidence names `00-0000099` as the QB's NFL id, and the live generation carries the same id,
because the recorded game is week 9 replayed as league week 1). The official week-1 files are synthetic: one QB row
and one END GAME play, with the header copied from the real 2025 weekly file. Because they are synthetic, they show
the pin and correction mechanics; they are not real 2026 statistics.

Reuse: `newReleaseCLIWith` in `release_matrix_test.go` is `newReleaseCLI` with an optional seed that returns the
football digest; the matrix test is unchanged in behaviour. No new dependency.

## testscript decision

`github.com/rogpeppe/go-internal` is not in `go.mod`. Not added: the journey needs Go-level fixtures (a fake store
clock, a live overlay publisher, a seeded ledger) that testscript cannot reach without custom commands, and a new
module means `go.mod` and `go.sum` changes under `-mod=readonly`. Extending the existing Go end-to-end style kept the
spike to one file. Revisit when the box runs the real `twr-server` binary: then the steps become shell-like
(`exec twr ...`) and testscript earns its place.

## Blockers (what stops the full journey)

- **B1. `twr` has no read verb.** Reads (`results.read`, `standings.read`, `lineup.read`) are only reachable over
  `/v1/native/workflow` in mode `read`, and `serverclient.Agent.postWorkflow` is unexported
  (`internal/serverclient/agent.go:110`); the CLI offers only `ledger`, `preview`, `execute`
  (`cmd/twr/main.go:113,134`). The spike posts the workflow request itself (`boxRead`). Proposal: exported
  `Agent.Read(ctx, command, scope)` and `twr read <command> [scope.json]`; small, read-only, token scope already gates it.
- **B2 (in-process part closed; recap still open).** The box now pins a final and a correction from files on disk.
  What is still open: (a) the server's own official job (`officialKeeper`, `cmd/twr-server/official_stats.go`, package
  `main`) is not run by the box, so the server's scheduling, retry and guard are untested here (its guard is tested
  at `cmd/twr-server/official_stats_test.go:10`); (b) the recap (step 9) has no hosted read. The original problem was: a final exists only when
  official nflverse weekly and play-by-play are pinned (`PinOfficialStats`,
  `internal/store/history/football_finals.go:45`); a live frame is always provisional
  (`internal/store/history/live_overlay.go:206`, `officialWeek` short-circuits live once official). The server's only
  driver of that is `officialKeeper`, started at `cmd/twr-server/hosted.go:140`, whose file source is hard-wired to the
  network (`cmd/twr-server/official_stats.go:72`, `nflverseFiles` at `:42`; URLs at
  `internal/fantasyscore/source/plays.go:16,21`). The `officialFiles` type (`official_stats.go:36`) is the right seam
  but nothing selects it. Needed: (a) a trimmed fixture of weekly + play-by-play + roster CSV for one real week whose
  game list matches the box schedule (existing `internal/fantasyscore/source/testdata/e5_*` are another
  season/teams), (b) in a test: build `history.OfficialSeason` (as `officialSeason`, `official_stats.go:150`) and call
  `PinOfficialStats` then re-pin the league; for a correction, offer a second, changed weekly file (the pin adds a
  revision and `StatRevision` shows the move). For the real binary: a `-official-dir` flag (practice build tag) that
  makes `startOfficialStats` read `stats_player_week_<season>.csv.gz` etc. from a directory.
- **B5. The week-1 opponent has no roster.** The schedule pairs franchise.01 with franchise.04. The workflow seed
  (`tools/hostedfixture/workflows.go:44-48`) rosters only franchise.01 (0099, 0100, 0101) and franchise.02 (0102). A lineup
  can be set only for the setter's own franchise (`internal/commissioner/governance.go:330`), and one person holds one
  seat (`internal/governance/seats.go:85`), so the commissioner cannot set franchise.04's lineup. The fix is a second
  owner (`hostedfixture.AddWorkflowOwner`, `tools/hostedfixture/owner.go:36`) plus a rostered franchise.04 player.
  That is a change to the shared seed, which this spike did not make. Until it lands, the game stays "No result yet" and
  standings have 0 decided games. Choice for Christopher: (A) add one franchise.04 player and its owner to the seed
  (small; touches every seed importer); (B) leave it as a logged NOT REACHED.
- **B6. Starter revision after a correction.** `Revision` on a starter is the move off the last live line
  (`internal/commissioner/score_revision.go:30-38`). After the official pin, live overlay reads for that week are
  ignored (`internal/store/history/live_overlay.go:204-206`), so no live line is in the week's evidence and the
  revision is nil. Read from the code, not yet traced at runtime. Fix options: keep live lines in the evidence
  after the official pin, or accept that the revision is shown only on the live path.
- **B3. Quick draft.** The workflow league seeds rosters directly and starts a 2027 rookie draft; each pick is an
  owner action (`serverhttp.PickPath`, `draft.pick`, `internal/commissioner/draft_wire.go:21`). No commissioner
  autopick/quick-draft command exists to drive 32 slots. Proposal: a test loop that seats a principal per franchise
  (as `hostedfixture.AddWorkflowOwner`) and picks in order, or a G3 `draft.autopick` command if the product wants one.
- **B4. Real binary and a clock it obeys.** `twr-server` reads wall time in many places: `cmd/twr-server/hosted.go:364-365`
  (`NewHostClock`, the store and due worker), `football_pins.go:52`, `official_stats.go:72`, `league_load.go:370`,
  `league_refresh.go:101`, `live_collector.go:160,256,284`, `live_packets.go:124-185`. No flag or env var replaces them.
  Proposal: one `-clock-file` flag (practice build tag, like `backup_practice.go`) read by a single `clock func()
  time.Time` passed to those sites; the test rewrites the file to advance time. The live collector also has no
  replay-from-directory mode outside tests (`live_collector.go:301` replay is for restart restore); add `-live-replay-dir`
  under the same tag. Launching: `tools/twr-practice up` already provisions a sandbox server with a disposable account
  and authenticator; a box harness would build `-tags practice`, start it with `-clock-file`, and pass the printed token
  scope to `twr connect`.

## Next steps, in order

1. `twr read` (B1): makes every read in the journey a CLI call.
2. Official stats (B2): done in process for final and correction. Next: run the server's own official job
   (`-official-dir`, B2a) and add a recap read (B2 recap).
3. Draft loop (B3), and the franchise.04 opponent (B5), so week 1 has a decided game.
4. Clock and feed flags, then swap the in-process server for the real binary (B4); consider testscript then.
5. Starter revision after a correction (B6).
