# Design: My MFL moves on the hosted app (spec docs/specs/mfl-helper-small.md)
Written 2026-11-23 by a Sonnet helper for Tom (read-only scouting); open choices 1-2 at the bottom were settled the same evening, see mfl-helper-small.md "Decided".

Done looks like: a hosted owner of an MFL-living league opens "My MFL moves", sees free agents with values, plans a bid and a lineup with plain-word checks, taps Open on MFL / I did it / Withdraw; nothing goes to MFL.

## 1. Data: no hosted path exists today
- Hosted is native-only: browser session admits one native league (desktop-modes.md:79); `league-load`/`league-refresh` (cmd/twr-server/league_load.go:319, league_refresh.go:41) turn a desktop `history.db` mirror into a NATIVE league, service stopped. `mirror-fetch` (main.go:236, mirror_fetch.go:64) writes that mirror by the desktop Prepare path. desktop-modes.md:78: "no automatic league-specific MFL sync". Only the NFL injuries export is fetched live.
- Smallest path: a server-side `mflmirror` refresh that calls the same public fetchers the desktop uses (internal/mfl/client.go, ingestion/freeagents/fetcher.go:46 Fetch, league/allrules.go, players, leaguestandings, leagueschedule) into a per-MFL-league held store in the hosted DB. Public exports only, no MFL login or key.
- Throttle: one refresh per MFL league per 5 min, enforced in the store (last_fetched_at), and single-flight. Refresh is a command (below); a click inside the window returns the held copy with its age. Host discovery already caches 15 min (client.go:141).
- Reuse `mirror-fetch`'s fetch-and-write code (extract from cmd/twr-server/mirror_fetch.go into internal/leagueload or a new thin internal/mflmirror) so the CLI and the new command share it. Do not grow mflref.

## 2. Commands (internal/commandcatalog; Availability "server"; one new leagueservice file `mfl_moves.go`)
Pattern: ownerPrivateCommand (native_mutations.go:157, native_reads.go:194), facts/safeguards like native_reads.go:29, franchise from the session only (player_personal.go:279 franchiseOwnerAccess; no franchise arg).
| Command | Kind | Reuses |
|---|---|---|
| `mflmoves.refresh` {league} | owner-private Execute (reads MFL, writes only our mirror) | shared fetch (sec 1), RefreshOffersPool (offers/pool/refresh.go) |
| `mflmoves.pool.read` {league, position?, sort?} | owner-private read | offers TargetPool pool_api.go:108 + valuePoolRow :226 |
| `mflmoves.bid.check` {league, add, amount, drop[]} | owner-private read (Preview) | contracts TargetAcquisition acquisition_api.go:85 (limits :259, cut :275, roster :304), envelope/bid.go:62 |
| `mflmoves.lineup.check` {league, starters[]} | owner-private read | TargetCheckLineup lineup_api.go:117, lineup/rules.go:29, byes.go:24 |
| `mflmoves.plan.save` {league, kind bid or lineup, body} | owner-private Execute | moves.SaveBid bidfacts.go:17, TargetDraftLineup :182 (checks re-run server-side on save) |
| `mflmoves.plans.read` {league} | owner-private read | moves store list |
| `mflmoves.plan.link` {league, plan} | owner-private read | planwatch RosterMoveTarget moves.go:59 + ValidateMoveURL :12 (returns the URL; the browser opens it) |
| `mflmoves.plan.done` / `.withdraw` {league, plan} | owner-private Execute | envelope HandOffManual :314, WithdrawBid :353 |
Extraction needed: the contracts/offers APIs take a desktop `Host` (contracts/api.go:23, offers/api.go:25) implemented by composition/contracts_host.go and moves_app.go. Write one small hosted Host adapter (in leagueservice) over the held mirror, implementing only RosterSnapshot/AcquisitionSnapshot, ContractsLineups/Week, RulesFreshness, OffersFreeAgents, OffersEngineValues/AssetValues; no-op OpenMoveURL/WakeMoves/OpenBrowser. Do not import internal/composition. Add each command to docs/ui/endpoint-registry.csv, regen endpoints.gen.ts, and CLI/MCP gets them free.

## 3. Plan storage and seat scoping
- Reuse the moves store (internal/store/mflref/moves) in the hosted DB: it already holds envelopes by correlation id. Add `league` and `franchise` columns/keys so every read and write is `WHERE league=? AND franchise=<seated>`. Franchise comes only from the owner's seat (franchiseOwnerAccess); a request naming a franchise is rejected (same safeguard text as native_reads.go:31). Results are uncached.
- Alternative rejected: internal/store/personal (notes/watch entries only; no envelope states).
- Pool, rules, rosters of the whole league are public MFL data and stay shared; only plans and the "mine" roster slice are private.

## 4. Frontend
Hosted franchise screen is frontend/src/app/shell/NativeFranchiseHQ.tsx (lazy at v3/V3Shell.tsx:27); the existing moves view HQMovesV3.tsx + shell/HQMoves.tsx are lane-based and reusable for plan states. Add `shell/MflMovesPanel.tsx` loaded with `lazy(() => import(...))` from NativeFranchiseHQ behind a "My MFL moves" tab; own chunk, nothing in entry (cap 321,818 B). Commands via nativeCommands/native.gen. Check bundle budget in the PR.

## 5. Lanes
A. Data and commands read side (1 PR). Done: `mflmoves.refresh` + `.pool.read` return held MFL pool with values on the Lab for league 14432; second refresh within 5 min returns held copy. Tests: throttle test (fails if the window check is removed); franchise-from-seat test (other franchise's request refused).
B. Checks and plans (1 PR). Done: bid/lineup checks and plan save/done/withdraw/link work through the catalogue; same plan = same answer as desktop golden cases. One test per check, each fails when its check is removed: cap room after bid (:259 path), roster size after drop (:304), bid limit (:259), lineup slot counts, bye, IR; plus withdraw-only-from-planned, link only to validated MFL host (ValidateMoveURL), cross-owner read refused.
C. Frontend (1 PR). Done: lazy panel on the PWA shows pool, bid and lineup forms, plain-word problems, Open on MFL / I did it / Withdraw; bundle budget passes; Lab click-through (no click on MFL) with screenshot.

## 6. Risks
- Sends to MFL: none by design. The mirror client is GET-only on public exports; add a test that the mirror HTTP client rejects non-GET and any import/submit type. `plan.link` only builds a URL; "I did it" is a local state change (HandOffManual) and never verifies by posting. Never use the owner's MFL key (internal/mflkey).
- Leaks: plans and "mine" reads keyed by seated franchise only; pool is public. Check logs and error text do not echo other franchises' plans. Mirror store must not be served by existing league-view reads to non-owners.
- League-source branching: the commands must key on the league's data, not on "is MFL". Hosted already has native leagues; do not add `if source==mfl`. The MFL-living league needs a registration row (MFL league number + which franchise is mine) stored as plain league settings, not a league kind.
- Hosted Host adapter must not call refreshMu/serializers from the desktop; use one store lock.
## Open product choices for Christopher
1. How does an owner say which MFL franchise is theirs: type the MFL league number and pick their team from the public list (anyone could pick another owner's team, but they only see public data and their own plans), or only the commissioner can attach a league? Recommend: owner picks, private plans, no MFL login.
2. Should the hosted server hold an MFL-living league at all as a league (seat, sign-in), or is this a separate "solo" login with just this panel? Recommend a normal seat so permissions and the one-session rule stay the same.
