# Season in a box (T711 spike + binary box)

**Status:** spike, 2026-11-22 (`TestSeasonInABoxSpike` in `cmd/twr/season_in_a_box_test.go`); real-binary box added 2026-11-23 (`TestSeasonInABoxBinary` in `cmd/twr/season_box_binary_test.go`, covering STEP 1–9 including owner-door draft picks and the Tuesday report card `twr-check`).
**Goal:** a whole league week, played in seconds, offline, from a recorded real NFL game, so a regression in the
create -> draft -> lineup -> lock -> live scores -> final -> correction -> standings -> report card -> recap chain fails a test instead of a Sunday.

## What the spike does today

Run: `go test ./cmd/twr -run TestSeasonInABoxSpike -count=1 -v`.

Run time (`time go test ./cmd/twr -run TestSeasonInABoxSpike -count=1`, warm cache, 2026-11-22): 7.0 s wall, the test itself 5.9 s (`go test -race`: 15.7 s test, 51 s wall). The 5-minute budget is not close. Test status on the final tree: `cmd/twr`, `cmd/twr-server`, `internal/app/commissioner`, `internal/commissioner`, `internal/serverhttp` pass; `tools/twr-practice` passes 6 of 7 tests: `TestPrebuiltSandboxHTTPSLifecycleWithoutGo` fails on the base commit too (`frontend/dist` is not built, `tools/twr-practice/main_test.go:186`). `golangci-lint` on the changed packages: 0 issues.

| # | Journey step | How it runs | Status |
|---|---|---|---|
| 1 | Create native league | `hostedfixture.CreateWorkflowLeague` (32 franchises, rules, schedule, pinned football), then `twr ledger` over HTTPS | reached |
| 2 | Quick draft | Second owner seated on franchise.04 (`hostedfixture.AddWorkflowOwner`). `draft.order.set` moves franchise.04 to pick 2 while the clock runs, with a reason. Then franchise.01 takes rookie 5001 and franchise.04 takes rookie 5002, each through the store's `RecordDraftPick` under the owner's principal, with the calendar permit and the pinned facts. The owner door (`PickPath`) refuses both (B7). | reached through the store (`RecordDraftPick`); owner door NOT REACHED (B7) |
| 3 | Set lineups | `twr preview` of `commish.lineup.set` through the real token and dispatch path; recorded through the commissioner's own session (G2 needs a human confirmation). franchise.04 sets its own (starter 5002, a QB) under its principal, the same store route | reached (franchise.01 and franchise.04) |
| 4 | Pass lineup lock | fake clock advances 25 h past kickoff; a lineup that adds a player is refused ("locked at kickoff") | reached |
| 5 | Feed live stat lines in pieces, scores move | recorded ESPN summaries (`cmd/twr-server/testdata/live-week9.tar.gz`, frames 01 then 02) published to `history.LiveOverlay`; `results.read` read twice over HTTPS; franchise.01 goes 30.000 -> 60.500 provisional | reached |
| 6 | Final | `PinOfficialStats` on the synthetic 2026 week-1 files in `cmd/twr/testdata/official/` (no network); `results.read` shows franchise.01 final, no provisional (23.5 points) | reached |
| 7 | One stat correction | second pin of `stats_player_week_2026_corrected.csv.gz` (QB passing yards 250 -> 262); franchise.01 moves 23.5 -> 24.1. The starter's revision (the move off the last live line) is NOT REACHED, see B6 | reached (points); NOT REACHED (starter revision) |
| 8 | Standings | `standings.read` answers with status "Ranked" (not "Needs a ruling"). 1 of 16 games decided: franchise.04 at franchise.01, status "Final" (franchise.01 241000 = 24.1; franchise.04 0, its rookie's zero stat line). 15 pending. | reached (1 decided game) |
| 9 | Report / recap read | `results.read` answers week 1 (16 games, 32 franchise scores). The recap text has no hosted read: NOT REACHED (B2 recap) | report answered; NOT REACHED (recap) |

Values in the test log are the engine's fixed-point numbers (10,000 per point): `235000` is 23.5 points.

## Playoffs

`TestSeasonInABoxPlayoff` (`cmd/twr/season_box_playoff_test.go`) runs a complete 4-team playoff bracket against the real `twr-server` binary, CLI, fake clock and synthetic official files (`cmd/twr/testdata/official/*_playoff_w{1,2,3}.csv.gz`). In the playoff test only, normal `commish.rules.set` commands configure a single conference (`league.conferences=1`), a 1-week regular season (`schedule.regularWeeks=1`), and a 4-team, 2-round bracket (`playoffs.perConference=4`, `divisionWinners=0`, `wildCards=4`, `byeSeed=0`, `reseed=false`, `startWeek=2`, `finalWeek=3`, `playoffs.graph` with `semi_1`, `semi_2`, and `championship`), with a 2-game week-1 fixture schedule so all regular-season games go final.

| # | Playoff step | How it runs | Status |
|---|---|---|---|
| 1 | Regular season final | Week 1 lineups via owner door (`LineupPath`); synthetic week-1 official files pinned by the server's official job; `standings.read` reports 2 decided, 0 pending, through week 1 | reached |
| 2 | Build and seed bracket from standings | `bracket.read` reports `"Ready to build"` with seeds 1–4 (`franchise.01`..`franchise.04`); `bracket.build` (`league_rules`) and `bracket.seed` previewed via `twr preview` + workflow and executed through `/v1/native/workflow` | reached |
| 3 | Score week-2 semi-finals final | Synthetic week-2 official files pinned; `bracket.read` computes `League/semi_1` (`franchise.01` over `franchise.04`, 29–5) and `League/semi_2` (`franchise.03` over `franchise.02`, 22–11) | reached |
| 4 | Advance semi-final winners | `bracket.advance` previewed and executed through `/v1/native/workflow` for both semi-finals; `bracket.read` pairs `franchise.01` v `franchise.03` in `League/championship` | reached |
| 5 | Score week-3 championship final and name champion | Synthetic week-3 official files pinned; `bracket.read` decides `League/championship` (`franchise.03` over `franchise.01`, 26–12); `bracket.advance` confirmed through `/v1/native/workflow`; champion `franchise.03` (`Practice team 3`, `"Won the final"`) | reached |

All bracket writes go through the hosted `/v1/native/workflow` command path and all bracket reads through `twr read bracket.read`. No rehearsal playoffs are substituted.

## Real versus faked

Real: the hosted HTTP handler (`serverhttp.Hosted`, TLS, token scope, workflow door), the draft rules and calendar gate (`NativeActionGate`) and the pick checks inside `RecordDraftPick` (seat, turn, rookie pool, roster, cap), the `twr` cobra commands and
`serverclient.Agent`, the commissioner ledger and every rule check, native week scoring
(`commissioner.NativeResultsReading`, `history.nativeSources`), the live overlay, SQLite.

Faked: owner picks skip the owner door (B7) and are recorded through the store with the owner's principal and the pinned facts; the process boundary (the server runs in the test process on `httptest`, not as the `twr-server` binary); the
clock (`Store.SetClock`; account and due-worker clocks stay real); the NFL feed (recorded frames, no network); the
commissioner's G2 confirmation (the test calls `ExecuteCommissioner` as the seated principal); NFL player identity
(the box football evidence names `00-0000099` as the QB's NFL id, and the live generation carries the same id,
because the recorded game is week 9 replayed as league week 1). The official week-1 files are synthetic: one QB row
and one END GAME play, with the header copied from the real 2025 weekly file. Because they are synthetic, they show
the pin and correction mechanics; they are not real 2026 statistics. The second row (`00-0005002`, QB, every stat 0) is the rookie franchise.04 starts. Both week-1 rookies (5001, 5002) are seeded practice players, not a real draft class.

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
- **B3. Quick draft: the box drives two owners, not 32 slots (partly closed).** The seed starts a one-round 2027 rookie
  draft on franchise.01. Only franchise.01 and franchise.02 are seated by the seed; the box needs two picks (01 and 04). The box
  moves franchise.04 to pick two with `draft.order.set` (`internal/commissioner/draft_commands.go:69`, a change while the clock
  runs needs `Running` and a reason). No auto-pick path exists in Go: `autopick` is only a rule value
  (`internal/leaguerules/menus.go:106`) and nothing reads it, so the picks are the owners' own. Skips and assigns were not used:
  a skip forfeits the pick (`internal/commissioner/draft_preview.go:108`).
- **B5. The week-1 opponent has no roster (closed for the box).** franchise.04 now has a second owner (`AddWorkflowOwner`,
  `tools/hostedfixture/owner.go:36`), and the shared seed (`tools/hostedfixture/workflows.go`) adds one 2027 rookie QB, `5002`,
  with a GSIS id `00-0005002`. The seed change is the one shared-seed change in this step: Christopher to confirm it (option A
  in the old note). The box also gives `5002` a zero stat row in both week-1 files (`cmd/twr/testdata/official/`), so
  franchise.04 has a complete final score. A lineup can be set only for the setter's own franchise
  (`internal/commissioner/governance.go:330`), so franchise.04's lineup is recorded under its own principal.
- **B7. The owner door refuses every rookie pick (closed 2026-11-23).** Fixed on `main` (PR #468): `workflowHost` in
  `internal/serverhttp/hosted.go` now pins football for `raw.Pick != nil` as well as workflows, and `requestContext` sets the
  league's pinned football digest for pick requests. Covered by `TestHostedOwnerRookiePickAdmitted`
  (`internal/serverhttp/hosted_pick_test.go`) and STEP 2 of `TestSeasonInABoxBinary` (`cmd/twr/season_box_binary_test.go`),
  where `franchise.01` and `franchise.04` each pick a rookie (`5001`, `5002`) through `PickPath` with their own owner session
  tokens and `twr read native.assets.read` confirms both rookies on their rosters.
- **Harness note (not a product bug).** The box clock stands still between calls and keeps nanoseconds. A pick stamped at the
  instant of the one before would not yet count in the next pick's judgement (`s.now()` truncates to microseconds). The box
  advances the clock a minute between picks. Real owners pick seconds apart.
- **B4 (clock, live replay, binary draft and report card: closed 2026-11-23).** `TestSeasonInABoxBinary`
  (`cmd/twr/season_box_binary_test.go`) builds `twr-server -tags practice` and `twr-check` once, seeds a temp database as
  `tools/twr-practice` does (workflow league, second owner on `franchise.04`, pick-2 draft order, `franchise.01` lineup,
  replay QB IDs, agent tokens for `twr` and read-only `twr-check`), runs `upgrade`, starts `hosted -official-stats` on
  `127.0.0.1:0`, and drives the full week:
  - STEP 1: `twr ledger` over HTTPS.
  - STEP 2: `franchise.01` and `franchise.04` each draft a rookie (`5001`, `5002`) through the owner door (`PickPath`, owner
    session tokens), verified on their rosters via `twr read native.assets.read`.
  - STEP 3–4: `franchise.04` sets its week-1 lineup (`5002`) via `LineupPath`; `twr preview` admits `franchise.01`'s lineup
    before kickoff and refuses it with `"locked at kickoff"` after the fake clock moves 25 h.
  - STEP 5–8: recorded live frames replayed from `testdata/live-week9.tar.gz` move provisional points (3.00 -> 6.05) and mark
    the NFL game final; the server's official job pins `testdata/official` (final 23.5) and the corrected file (24.1);
    `standings.read` reports `"Ranked"` with 1 decided game (`franchise.01` over `franchise.04`) and 15 pending.
  - STEP 9: the Tuesday report card (`twr-check`, read mode) runs against the binary and passes (exit 0, no RED; `Official stats
    final` and `Lineups locked at kickoff` GREEN; `Scores add up` verifies both finished teams, 30 unfinished teams AMBER;
    `Standings match games` AMBER because only 1 of 16 week-1 games is decided).
  Skipped under `-short`. Seams (practice build only, `cmd/twr-server/clock_practice.go`):
  `TWR_PRACTICE_CLOCK_FILE` (one RFC 3339 instant; the clock is that instant plus wall time since the write; feeds the
  store, due worker, live replay job and official job), `TWR_PRACTICE_LIVE_DIR` (the live job replays recorded ESPN
  summary JSON frames from a directory in sorted filename order instead of the network, with an optional `week` file),
  and `TWR_PRACTICE_OFFICIAL_DIR` (the official job reads the season files from a directory, first pass after 1 s,
  then every 1 s). Release builds (`backup_hosted.go`) never read any of them; `TestReleaseServerIgnoresPracticeSeams`
  proves it. Still open in the binary box: the written recap text (sentences; no hosted recap read exists yet, B2 recap).
  Other wall-clock sites listed below are not on the fake clock. Original note:
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

1. Written recap text (B2 recap): add a hosted recap read (written sentences); `TestSeasonInABoxBinary` already runs the Tuesday report card (`twr-check`, STEP 9) and logs recap text as the one open STEP 9 sub-step.
2. Starter revision after a correction (B6).
3. Closed: `twr read` (B1), owner door for picks (B7), official stats job (B2a), real binary + fake clock + live replay + draft + report card (B4).
