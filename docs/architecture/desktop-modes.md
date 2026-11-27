# Hosted app and league server

The React PWA is served at `/app/` by `twr-server`; `twr` provides CLI and MCP access.
The server owns the league databases, engines, feeds, clocks, and backups.

Hosted League rules expose Change only to commissioners and co-commissioners. The shared rule input
builds a typed value; the hosted room owns preview → explicit confirm → execute → reread, using the
rules revision and request key. The server authorizes and records the change; no local proxy is used.
A commissioner without a franchise opens league-wide Pulse and rule tools, not franchise tasks.
Their room reads only rules; Pulse independently reads the directory, schedule, results and standings
(and optional bracket/odds/football views), preserving the rest when an optional read is refused.
`week.read` accepts an empty franchise for admitted league members and returns only published week
numbers and kickoff timing, without a selected matchup or private team data. Their Pulse opens on
the server-selected current week with explicit week navigation. Hosted odds read once per season on opening,
then only on a league topic (the standings/results invalidation), not scores, team, values or clock topics,
manual Refresh or automatic retries of other reads. A refused odds read shows “Playoff odds: unavailable.”
When the server refuses odds, the screen keeps its other readings.

For change notifications, see [hosted change notifications](hosted-changes.md).

## What the server (`cmd/twr-server`) carries today

- **Engines:** `engine/*` (competition, rookie draft, bids), `fantasyscore` and `fantasyscore/source` (the
  ESPN → Sleeper → nflverse source order), `model`, `modelrun`, `appvalue`, `tradelens`, `projection`,
  `powerrankings`, and the `ingestion/*` parsers for league reads.
- **League:** one database file per league, a sign-in file and a football reference file; the command
  catalog (preview → confirm → execute); "something changed" pings over SSE
  (`/v1/native/invalidations`); shared ESPN/Sleeper live acquisition; accounts and seats (`/v1/auth/*`); R2 backups; signed upgrades; status.
- **Reads the full screens will need already exist** in `internal/commandcatalog/native_reads.go`:
  lineup, results, standings, player score, values, trade lens, odds, bids, moves, draft, schedule,
  rules and the commissioner reads.

## What the server does NOT carry today

- **Official live finals:** the hosted collector shares `internal/app/now.LiveStatsPoller`.
  ESPN/Sleeper generations remain provisional; nflverse acquisition is deferred. Raw current-week
  packets and identities are checkpointed under `live-scores/` beside the server file; restart validates digests
  and rebuilds through the same parsers. The L5 `PublishScores` interface is not yet connected.
- **Fresh football facts:** each league selects one football-evidence digest
  (`cmd/twr-server/football_pins.go`). A running server can swap a pin through `pin-football` on its
  private status socket. With the server stopped, `refresh-nfl-schedule` can append missing published
  NFL kickoff dates from MFL and pin a new immutable evidence version to one league; there is no
  automatic schedule refresh. Live scoring remains a separate immutable current-week overlay; ticks
  leave action-relevant pin digests unchanged. Historical weeks use the selected pin.
- **MFL calls:** `twr-server mirror-fetch` pulls fresh MFL league exports, players, and NFL schedule
  weeks into a standalone `history.db` mirror atomically.
  `twr-server league-load` is a one-time console load from an MFL mirror, and
  `twr-server league-refresh` refreshes an already-loaded league from a newer read-only mirror while
  the service is stopped (preserving seats, rules, and in-app commissioner edits, stopping on open
  trade offers, and supporting `-dry-run`).
  Hosted reads get one automatic MFL-wide fetch: the keyless NFL injuries export, refreshed every four
  hours and exposed as status facts to league members. `refresh-nfl-injuries` is its operator refresh
  while the server is stopped. `refresh-nfl-schedule` remains an explicit keyless `nflSchedule` request;
  there is no automatic league-specific MFL sync.
- **Browser access to the league:** `internal/serverhttp/browser.go` admits `/app/` and its hosted JSON command routes only when Origin is exactly `https://` plus the configured public host. `/app/session` issues the revocable owner session in the `__Host-twr_session` cookie and returns the admitted leagues. The browser never receives the owner bearer. Agent-token requests with Origin, mixed Authorization+cookie requests, enrollment and the changes feed remain refused. Static app responses use the required CSP, `nosniff`, same-origin referrer policy and no-cache service worker.

The server embeds the build at `frontend/dist-web` when available, otherwise `frontend/dist`, and serves it under `/app/`. The server-side session route scans current league state for the signed-in account; it does not call `commish.leagues.read` or return leagues the account cannot enter.

## Check it yourself (about two minutes)

```sh
go list -deps ./cmd/twr-server | grep -E 'internal/(app/now|ingestion/livegame)$'  # now includes the shared poller and parser
grep -n 'pin-football' cmd/twr-server/football_pins.go              # frozen football pin
grep -n 'func (h Hosted) originAdmissible' -A12 internal/serverhttp/browser.go # exact browser Origin allowlist
rg -n 'appSession|authenticateNative' internal/serverhttp/browser_session.go internal/serverhttp/hosted.go
```

If any of these disagree with this file, the code wins. Fix this file before planning.

Hosted NFL flow: the collector validates acquired summary packets with `livegame.ParseSummary`, then atomically replaces the shared current-slate detail map. `nfl.game.read {league, game}` uses directory-read admission and returns only public NFL facts, with an acquisition timestamp/stale flag or explicit no-detail words. Hosted Pulse fences superseded requests, rereads on its topic refresh, and uses the existing field/plays components. Pulse follows the live game with the most own starters by default; owners may switch games, and manual Refresh remains available. Private lineup admission is unchanged.

Hosted player inspector reads `player.usage.read` and `player.agearc.read` are catalogued and
member-admitted; HQ and War Room use shared hosted API adapters.
Both currently refuse with “hosted player reading is unavailable”: the league-load path supplies
player directory facts and NFL weeks, not the weekly usage observations/points-allowed research
feed or the birth-dated player book, model scores and fitted run parameters. Reference-file table
existence is not evidence those inputs were acquired. No new ingest or schema is included here.

Hosted HQ My moves now adds `franchise.moves.history.read`: a season page over the server-native
append-only ledger, using the existing audience-filtered descriptions. Members can read public
franchise history; sealed/private rows keep their existing Audience policy. Corrections remain
new rows. `moves.read` stays owner-only lifecycle data.
Source and evidence: [HQ native history](../build-handoffs/HQ-MOVES-SOURCE-2026-11-17.md).
