# TheWarRoom target UI: specification, revision 0 (2026-10-06)

**What this is:** the merged target from two independent research passes:
- **Sol:** `~/fleet/briefs/warroom-endpoint-ia-bee-2026-10-06.md`
- **GLM:** `~/fleet/briefs/warroom-endpoint-ia-bee-pass2-2026-10-06.md`

**How it fits with the rest:**
- Christopher's rulings below are binding.
- The full detail of each section lives in the source answer cited.
- `docs/ui/endpoint-registry.csv` lists every endpoint row.
- The build order is in `docs/build-handoffs/UI_Target_Roadmap_2026-10.md`.
- This spec is revised at the end of every ring.

## 1. Rulings (Christopher, 2026-10-06)

1. **Two clicks:**
   - At most two in-app clicks to a ready move, with MFL opened.
   - The ≤600ms hold is a gesture, not a click; typing in the command bar costs 0 clicks.
   - Steps on MFL's own form are counted separately.
   - In Phase 2 the same count means the move is truly done.
2. **Inspector width:** rests at ~320px. `inspector.expand` widens it to ~480px as an overlay, with
   the workspace still visible and no page change.
3. **Neutral badges:** position, tier (contender / in the hunt / rebuilding) and posture (bought
   for now / built for later / like for like) are monochrome glyphs. This amends UI Direction §7.3.
4. **Onion build** (see the roadmap).
5. **Concepts, not yet briefed:**
   - live game view;
   - Gameday matchup in two views, Default (formation) and Classic (list);
   - headshots, personal use only;
   - startup-draft format.

## 2. What both passes agree on (the skeleton)

- **One command registry.** Every control is `domain.verb key=value`. The registry is also the
  future chat engine's grammar.
- **The inspector is the only entity page.** MFL's player hub (11 kinds of onward link) and
  franchise hub (25 kinds) become zones in the inspector.
- **Every Act is a move envelope:**
  - prepared, with pre-flight checks;
  - handed off to the exact MFL page;
  - verified on the next refresh from MFL's own data exports (transactions, rosters,
    pendingTrades, pendingWaivers, tradeBait, draftResults).
  - Phase 2 replaces only the hand-off step.
- **Lab analytics stay league-wide,** with no tint for the user's own team. Every number shows a
  baseline, a cell is marked best or worst only beyond chance, and "How this is counted" is always
  present. The offers chart is the only exception, labelled as the user's own offers.
- **MFL-hosted personal lists are app-local in Phase 1,** each with a hand-off to MFL: watch list,
  trade bait, scratchpad, draft list.

## 3. Channels: four jobs, four channels (GLM step 3; Sol step 3)

| Channel | Carries | Never carries |
|---|---|---|
| Colour (five locked meanings) | status and valence: green elite/positive · blue info · amber watch/clock running · red danger/injury/<1h · grey history | position, tier, gravity, identity |
| Chrome: size, border 1/1/2/3px, elevation, type weight | **consequence** G0 ambient · G1 watch · G2 stakes · G3 irreversible | good or bad |
| Countdown | **urgency** U0 none · U1 dated · U2 running (amber) · U3 under 1h or breached (red) | consequence |
| Monochrome glyph | classification: position, tier, posture; private markers (watch, bait) | good or bad, urgency |

**Ordering rule.** Time changes *placement*, not chrome:
- At T−7d an item moves up its lane.
- At T−48h it is pinned to Home's alert tray and the seasonal card.
- Within a lane, items sort by urgency (U), then consequence (G), then stable ID.

**Colour slots.** Each card has exactly four: status, verdict (only beyond chance), countdown and
provenance. Colour is always paired with a text label or icon.

## 4. Organization (GLM §4.1–4.2; Sol's cap of three workspaces per node)

| Node | Workspaces | Notes |
|---|---|---|
| Home | Seasonal card (owns the node) · alert tray · digests | phase router |
| War Room | Market · Valuations and pool · Research | Lab market, asset board, free-agent pool, bid tracker |
| Franchise HQ | Lineup and roster · Contracts and cap · My moves | Act plans, what-if, dead cap, app-local lists |
| Trade Floor | Trade desk and offers · Counterparties · Draft room | accept, reject, revoke, DOT state; draft timing inside the draft room |
| League Pulse | Now · Race · Pools / Archive | live game view and Classic/Default matchup in Now; report drawer for every MFL read report |
| Control Room | App · Data and sources · League (role-gated) | engine params gated to Admin |

- **Outside the nodes:** the inspector, comms (feed, chat, board, articles, polls, hand-offs), the
  summoned calendar, the command bar, and the status strip in the workspace header.
- **Admission rule for anything new** (Sol §4): it must name an existing job, a primary home,
  entity or time doors, a typed command, source / role / freshness, a gravity and a baseline.
  Never a new nav node and never a "More" bucket.

## 5. Card families (GLM step 5; Sol step 5)

There are seven: player, franchise, draft pick, trade/offer, deadline/clock, market segment and
report.
- Each has a fixed field order, three altitudes (glance, operate, interrogate) and three densities
  (Narrative, Tactical, Matrix). GLM §5 has the anatomy and sketches.
- Action trays hold every commit control. A read table never contains a commit control.
- The player card shows the two locked numbers: on-field-now and dynasty value.
- Photos are optional. The card falls back to initials.

## 6. Acts and the envelope (Sol §6 for rigour; GLM §6 for the verb table)

- **States:** Draft → Blocked / Ready → Handed off → Not yet done → Landed / Failed / Stale.
- **"Ready" means the app checked it, not that MFL accepted it.**
- **Landed** requires an observed MFL change that matches the row's full predicate. A partial
  observation reads "Not verified".
- A pending DOT review, a bid award or a waiver resolution are their own named lifecycle stages.
- **Unmapped MFL targets** stay drafts, with "MFL target not verified": add/drop form, propose
  form, survivor pick, option, re-sign, buyout, tender, UFA bid. They are closed in the roadmap's
  gap-closure work.

## 7. New concepts recorded this session

- **Live game view:**
  - Sources: ESPN `site.api.espn.com/apis/site/v2/sports/football/nfl/{scoreboard,summary?event=}`,
    Yahoo `api-secure.sports.yahoo.com/v1/editorial/s/scoreboard`, Fox `api.foxsports.com/bifrost/v1/nfl/event/<id>/data`.
    All three were probed and work without a key, but none is licensed.
  - Personal use behind one adapter, with failover. Facts are free to use (NBA v. Motorola).
    Logos are not, so use team abbreviations.
- **Headshots:** ESPN `a.espncdn.com/i/headshots/nfl/players/full/<espnId>.png` and Sleeper
  `sleepercdn.com/content/nfl/players/<sleeperId>.jpg`, keyed through the crosswalk. The photos
  are copyrighted: personal use only.
- **Gameday matchup:** Default (formation, laid across the line) or Classic (list). Legacy NFL uses
  Classic.

## 8. Open questions

Taken from both passes. These are owed before the ring where they matter.
1. MFL owner-form targets for the unmapped Acts.
2. Waiver mechanics.
3. Where DOT review shows.
4. Window time zones and edges, and how the windows map to the eight stretches.
5. Taxi and IR eligibility rules.
6. Whether Victory Points are used.
7. The 14 unnamed hub link kinds.
8. Whether MFL's exports cover pools and contract fields.
9. Refresh cadence during live clocks.
10. Minimum desktop width.
11. A data-scope ruling before multiple leagues are supported.
