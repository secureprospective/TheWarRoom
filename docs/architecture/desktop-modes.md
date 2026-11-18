# The desktop app has two modes. Read this before planning desktop, server or phone work.

Checked against the code at `main` 3f8149a27 on 2026-11-14; thin Home updated against the b3home
candidate on 2026-11-18. Christopher's ruling that day: **the usable product is the desktop app in thin
mode.** The phone/browser is a placeholder (registration, sign-in, and Live Pulse for the signed-in team
only) until he reopens mobile.

A session on 2026-11-14 treated "the desktop app" as one all-in-one engine and nearly planned work on the
wrong side of the client/server split. Both modes exist in one binary today. Never assume which one a
sentence means.

## The two modes

| | **Thin (hosted). This is the product.** | **Local (all-in-one, legacy)** |
|---|---|---|
| Start | `thewarroom -native <server-url> <league>` | `thewarroom` (no flag) |
| Go side | `composition.runNative` builds only `serverclient.Desktop` and the window. It "constructs only the hosted client and WebView, never the local App." | `composition.runDesktop` builds the full local App: local SQLite, engines, MFL sync, live feeds, clocks, backups. |
| Data | The league server over HTTPS (Cloudflare Tunnel). No league database on the computer. | `thewarroom.db`, `history.db`, `whatif.db` in the user's config directory. |
| Screens | `NativeStartup.tsx` mounts `NativeAccount` (sign-in, authenticator, AI helpers) and, after admission, the fixed app-level `ThinHome` (Needs-you, the admitted league card and shortcuts). Opening a card/shortcut mounts `HostedLeague` (League Pulse Now/Race, Franchise HQ, War Room, Set lineup, Roster and reserves, Draft room, League rules) and `HostedWorkflows` (bids, waivers, trades, results, rollover). Alpha Home uses only `NativeView.league`; it does not enumerate the server-wide league list. | `TargetMount` → the full shell `TargetApp.tsx`: Home, War Room, Franchise HQ, League Pulse, Control Room, command bar. |
| Live football | The server runs a shared ESPN/Sleeper collector for the current week; scores are labelled provisional. Each new generation publishes a scores topic to open change streams. Pulse rereads the public NFL scoreboard and selected game detail (field position and plays), following the live game with the most own starters by default; owners may switch games. Manual Refresh remains available. | ESPN live, Sleeper fill-in, nflverse finals and MFL live, polled inside the app. |
| Direction | Grows until it carries everything. | Kept as-is until thin mode carries everything (governing plan step 3). Do not grow it or add features to it. |

`ThinHome` exports `thinHomeRoute` for the planned thin rail; it does not build the rail. Its league card opens existing `HostedLeague` at HQ; shortcuts and actionable Needs-you items pass their matching hosted entry, with Control Room opening the hosted rules surface.

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

Thin windows now watch negotiated topic invalidations after sign-in/enter and expose them through
`/desktop/changes`; `nativeApi.subscribeChanged` is the webview transport. The thin-only
`data/hostedReads.ts` owns its one subscriber and identity fence: sign-out and known identity
changes clear private reads/rendered state. Admission hints recheck status without hiding the
screen; a changed role or franchise resets private state before the new view renders. Views and changes carry a decimal
auth generation; no credential crosses into the webview. See
[hosted change notifications](hosted-changes.md). Local mode is unchanged.

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

- **Official live finals:** the hosted collector shares `internal/app/now.LiveStatsPoller` without constructing
  the local App. ESPN/Sleeper generations remain provisional; nflverse acquisition is deferred. Raw current-week
  packets and identities are checkpointed under `live-scores/` beside the server file; restart validates digests
  and rebuilds through the same parsers. The L5 `PublishScores` interface is not yet connected.
- **Fresh football facts:** each league is pinned to the football evidence it was loaded with
  (`cmd/twr-server/football_pins.go`). A running server can swap a pin through `pin-football` on its
  private status socket, but nothing produces new pins. Live scoring is a separate immutable current-week overlay;
  ticks leave action-relevant pin digests unchanged. Historical weeks still use the selected pin.
- **MFL calls:** `twr-server league-load` is a one-time console load from a desktop MFL copy. There is no
  ongoing MFL sync.
- **Browser access to the league:** `internal/serverhttp/browser.go` (`browserRoute`) lets a browser reach
  only the public pages (promo, join, set password) and their JSON doors. Every other route refuses a
  request with an `Origin` header.

## So "finish the desktop" means

1. Bring each full screen into the thin window, reading the server's catalog reads instead of local Wails
   bindings (`frontend/src/app/data/provider.ts` picks Wails whenever `window.go` exists).
2. Give the server a live collector and fresh football facts, so thin screens and the phone have live
   data.
3. Leave local mode alone.

**Never:**
- start the local App inside thin mode;
- give thin mode a local league database;
- re-create engine logic in the frontend or the desktop client;
- delete local mode before thin mode carries everything.

## Rules for every session

1. In a plan, "desktop" means **thin mode** unless Christopher says local. If a sentence could mean
   either, ask him. Do not pick.
2. Before planning desktop, server or phone work, run the checks below. Do not reason from memory, from
   older plans or from a resume note.
3. Documents that say "the desktop runs the same screens" or "the PWA is the main interface" describe the
   **target**, not today's code. This file describes today's code.
4. A PR that changes any fact in this file updates this file in the same PR.

## Check it yourself (about two minutes)

```sh
sed -n '/^func runNative/,/^}/p' internal/composition/run.go        # thin mode: no local App
cat frontend/src/app/control/NativeStartup.tsx                      # which screens a hosted window mounts
go list -deps ./cmd/twr-server | grep -E 'internal/(app/now|ingestion/livegame)$'  # now includes the shared poller and parser
grep -n 'pin-football' cmd/twr-server/football_pins.go              # frozen football pin
grep -n 'func browserRoute' -A12 internal/serverhttp/browser.go     # what a browser may reach
```

If any of these disagree with this file, the code wins. Fix this file before planning.

Hosted NFL flow: the collector validates acquired summary packets with `livegame.ParseSummary`, then atomically replaces the shared current-slate detail map. `nfl.game.read {league, game}` uses directory-read admission and returns only public NFL facts, with an acquisition timestamp/stale flag or explicit no-detail words. Hosted Pulse fences superseded requests, rereads on its topic refresh, and uses the existing field/plays components. Local polling and private lineup admission are unchanged.

Registry wiring review: `commissionerWeekBindings` adds only `Bind2(api.TargetNFLGame, "league", "game")`. It forwards to the admitted shared reading; no composition policy or decision-bearing function changed. The boundary source fingerprint records this inspected additive binding.

Hosted player inspector reads `player.usage.read` and `player.agearc.read` are catalogued and
member-admitted; HQ and War Room use shared mode-aware API adapters rather than local Wails calls.
Both currently refuse with “hosted player reading is unavailable”: the league-load path supplies
player directory facts and NFL weeks, not the weekly usage observations/points-allowed research
feed or the birth-dated player book, model scores and fitted run parameters. Reference-file table
existence is not evidence those inputs were acquired. No new ingest or schema is included here.

Hosted HQ My moves now adds `franchise.moves.history.read`: a season page over the server-native
append-only ledger, using the existing audience-filtered descriptions. Members can read public
franchise history; sealed/private rows keep their existing Audience policy. Corrections remain
new rows. `moves.read` stays owner-only lifecycle data, and local desktop envelope audits are unchanged.
Source and evidence: [HQ native history](../build-handoffs/HQ-MOVES-SOURCE-2026-11-17.md).
