# The desktop app has two modes. Read this before planning desktop, server or phone work.

Checked against the code at `main` 3f8149a27 on 2026-11-14. Christopher's ruling that day: **the usable
product is the desktop app in thin mode.** The phone/browser is a placeholder (registration, sign-in, and
Live Pulse for the signed-in team only) until he reopens mobile.

A session on 2026-11-14 treated "the desktop app" as one all-in-one engine and nearly planned work on the
wrong side of the client/server split. Both modes exist in one binary today. Never assume which one a
sentence means.

## The two modes

| | **Thin (hosted). This is the product.** | **Local (all-in-one, legacy)** |
|---|---|---|
| Start | `thewarroom -native <server-url> <league>` | `thewarroom` (no flag) |
| Go side | `composition.runNative` builds only `serverclient.Desktop` and the window. It "constructs only the hosted client and WebView, never the local App." | `composition.runDesktop` builds the full local App: local SQLite, engines, MFL sync, live feeds, clocks, backups. |
| Data | The league server over HTTPS (Cloudflare Tunnel). No league database on the computer. | `thewarroom.db`, `history.db`, `whatif.db` in the user's config directory. |
| Screens | `NativeStartup.tsx` mounts `NativeAccount` (sign-in, authenticator, AI helpers), then `HostedLeague` (League Pulse Now/Race, Franchise HQ, War Room, Set lineup, Roster and reserves, Draft room, League rules) and `HostedWorkflows` (bids, waivers, trades, results, rollover). | `TargetMount` → the full shell `TargetApp.tsx`: Home, War Room, Franchise HQ, League Pulse, Control Room, command bar. |
| Live football | The server runs a shared ESPN/Sleeper collector for the current week; scores are labelled provisional. Pulse reads them with manual Refresh; push notifications wait for the publisher connection, and live NFL games are coming next. | ESPN live, Sleeper fill-in, nflverse finals and MFL live, polled inside the app. |
| Direction | Grows until it carries everything. | Kept as-is until thin mode carries everything (governing plan step 3). Do not grow it or add features to it. |

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
