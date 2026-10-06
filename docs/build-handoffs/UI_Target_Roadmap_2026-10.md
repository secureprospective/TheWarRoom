# TheWarRoom: the UI target and the roadmap to done (October 2026)

**Status:** this is a plan; nothing has been built from it.
- Written: 2026-10-06, by Claude with Christopher.
- Target: `docs/ui/Target_UI_Spec_2026-10.md`.
- Registry: `docs/ui/endpoint-registry.csv`.

## Why this exists

We are designing TheWarRoom's interface well before the infrastructure under it. The target
interface is what we build toward, and it defines done. Today's screens are a **testing harness**:
each harness screen retires once its target replacement is live.

The target covers every endpoint that matters:
- every MFL page and action (mapped 2026-10-05, 50 menu entries, 195 pages);
- every Legacy NFL Lab panel;
- every screen the app already has.

The target was researched on 2026-10-06 in two independent passes, then reconciled:

| Pass | Model | File | Census |
|---|---|---|---|
| Sol | gpt-6.1-sol | `~/fleet/briefs/warroom-endpoint-ia-bee-2026-10-06.md` | 442 occurrence rows |
| GLM | GLM 5.3 | `~/fleet/briefs/warroom-endpoint-ia-bee-pass2-2026-10-06.md` | 259 canonical rows |

The brief both passes answered is `~/fleet/briefs/warroom-endpoint-ia-2026-10-06.md`.

## What it looks like (rough draft r0)

`docs/ui/target-draft/TheWarRoom_Target_Draft_2026-10.html` is a static design board with eight
tabs: the visual system, Home, Franchise HQ, Trade Floor, War Room, League Pulse, the map of
everything and this roadmap. Ring 1 builds screens 1, 2, 3 and 5; ring 2 builds screen 4.
- Rosters, salaries and contract years are real (MFL, week 5 2026). Points, values, clocks and
  market figures are illustrative.
- Renders checked: `~/fleet/runs/warroom-endpoint-ia-2026-10-06/draft-renders/`.

## How we build: like an onion

Christopher's rule (2026-10-06):
- Start where people operate most often.
- Build the core systems a little at a time, until the core is solid.
- Then add layers.
- Expect to find better ways as we go. The target is not perfect, and the approach must leave
  room for error.

So the build is a set of **rings**, not a list of finished workspaces. Each ring:
- **goes through every layer**, from screen to data contract to data source;
- **deepens the most frequent paths first**;
- **ends with a review that revises the target spec itself** (spec revisions 1, 2, 3…).

A registry row that is not wired yet shows an honest "not wired yet" state, never invented data.
Screens are built against fixtures cut from **real league data** (the checked facts database
`tools/league-history/data/mfl.db`, `lab.json`, the MFL archive). When the infrastructure arrives,
each row's provider switches from fixture to live.

## Ring 0: the walking skeleton (core systems, thin)

| Core system | What ships in ring 0 |
|---|---|
| Tokens | `signal.*` (the five locked colour meanings, which no theme may override), `surface.*` (cold hsl 220 ramp, 4–13%), `brand.*` (crest and wordmark only). Gravity chrome steps. The glyph set: position, tier and posture glyphs, all monochrome. |
| Cards | The card base with four colour slots (status, verdict, countdown, provenance). The player card at three densities. |
| Shell | The four locked columns. Six nav nodes, at most three workspaces each. The inspector at 320px with `inspector.expand` to ~480px as an overlay. Calendar summon, the comms strip, five presets, and the workspace-header status strip. |
| Command registry | Every control is a verb with arguments, roles, aliases and an undo class. **A test fails on any clickable control without a verb.** |
| Endpoint registry | Every row resolves to a view or to "not wired". **A route test proves every row is reachable by at least two routes.** |
| Data contracts | A typed view model per row. Each value has a provider (fixture or live), plus provenance and freshness (per R11). |
| League-year clock | Phase, windows, countdowns and the T−7d / T−48h promotion rules. |
| Move envelope | The state machine on fixtures: Draft → Blocked / Ready → Handed off → Not yet done → Landed / Failed / Stale. Includes the audit log and correlation IDs. |

**Gate:**
- the route test and the verb test pass;
- one envelope runs end to end on fixtures;
- Christopher reviews it live on Claude-OS.

## Ring 1: the frequent loops, end to end

- **Home:** seasonal card and alert tray (lineup alert, IR violations, pending trades, waiver
  claims and order).
- **Franchise HQ:** lineup and roster.
- **Inspector:** player and franchise.
- **League Pulse › Now:** the Classic matchup view and live scoring from MFL `liveScoring`.
- **Trade Floor › Trade desk:** pending offers, accept / reject / revoke, the builder.
- **First live Acts:** `lineup.set`, `trade.propose` / `accept` / `reject` / `revoke`,
  `roster.ir`, `roster.taxi`. Each confirms against MFL exports (rosters, transactions,
  pendingTrades).

**Gate:**
- the walkthroughs for these loops pass the click budget;
- an envelope reaches Landed on the live gate;
- spec revision 1.

## Ring 2: the decision layer

- **The Lab, ported from JS to the Go engine** (`app/js/engine.js`, `draftlab.js`, `teams.js`,
  reading the facts database):
  - War Room › Market: going rate, buy/sell lists, asset profile, pick clock;
  - Trade Floor › Draft: timing, plays-like-pick, draft-or-buy;
  - Counterparties;
  - Offers.
- **Franchise HQ › Contracts and cap:** what-if, dead cap, and the window Acts (option, tender,
  re-sign, buyout, cut).
- **War Room:** the free-agent pool and bid tracking. Bid and claim plans live in HQ.

**Gate:**
- the numbers match the Lab's own tests;
- "How this is counted" appears on every analytic surface;
- spec revision 2.

## Ring 3: breadth and depth

- **League Pulse:** Race, Pools, Archive, and the report drawer for every MFL read report.
- **Comms:** channels and hand-offs.
- **Control Room.**
- **The live game view:** a followed-game bar, a field graphic, the last play, the box score, and
  the fantasy play spine. It is fed by ESPN first, Yahoo as hot failover and Fox as third, behind
  one adapter (R11).
- **Headshots:** ESPN, with Sleeper as fallback. Personal use only, and every card works without a
  photo.
- **The remaining Acts,** once their MFL targets are mapped.

**Gate:** every registry row is live or explicitly deferred; spec revision 3.

## Ring 4: the horizon (schemas decided now, built later)

- **Phase 2 writes:** the envelope's hand-off step becomes the write. Nothing else changes.
- **A portfolio of 15–25 leagues:** every verb and ID carries `league=` from ring 0.
- **The chat command engine:** the command registry is its grammar.
- **The commissioner role:** roles are already in the registry.
- **League branding:** only the `brand.*` namespace is themable.
- **The Default formation matchup view:** waits for Christopher's briefing.
- **The startup-draft engine:** see the brainstorm record below.

## Running beside the rings

**Gap closure (research, read-only):**
- Capture the MFL owner forms for the unmapped Acts: add/drop, propose, survivor, option,
  re-sign, buyout, tender and UFA bid. Use `mflmap` on Claude-OS, with no submits.
- Name the 14 unnamed hub link kinds.
- Collect the league rules facts from `league-archive/raw/allRules.json`, O=09 and the By-Laws.
- Ask Christopher: waiver mechanics, where DOT review shows, window time zones and edges,
  taxi/IR eligibility, whether Victory Points are used.
- Check whether MFL's exports cover pools and contract fields.

**Harness retirement:** a harness screen, and any IPC method only it uses, is deleted when its
replacement is live.

**The engine track:** the Core Build Plan stages continue. The queued item is removing NFL
contract data. The registry decides which infrastructure is needed next.

**Registry upkeep:**
- Each row's `status` moves through: spec → designed → built (fixture) → live → verified.
- Cross-checking Sol's 442 occurrence rows against GLM's 259 canonical rows is still open. Most
  differences are aliases recorded in GLM's duplicate ledger; the rest are reconciled in ring 0.

## Definition of done

- Every registry row is placed, reachable two ways, live, with a baseline (plus "How this is
  counted" where it is analytic), and has a registered verb.
- Every Act intent has been verified Landed on the Claude-OS live gate.
- The harness is retired.
- Phase 2 items are explicitly deferred, with their schemas already decided.

## Brainstorm record (concepts only, not in any ring yet)

- **The Gameday matchup comes in two views, for all leagues:**
  - **Default:** a formation laid "across the line". My offense faces their defense and their
    offense faces my defense. A spine of play events runs down the middle, with a bar of real
    games and a live feed on top.
  - **Classic:** a list of players. Legacy NFL uses Classic.
  - The default ruleset idea is 11 defenders against 6 offensive players plus an OL visual.
    Scoring is for another day.
- **Startup draft for leagues TheWarRoom hosts:**
  - Round 1 is an auction; rounds 2–25 are a snake. This is Christopher's candidate default.
  - The auction pool is the top 8 QB, 5 RB, 5 WR and 2 TE.
  - Every price is a share of the cap.
  - Max bid = cap left − money held back for open starting spots − bench minimums, with a check
    for each future year.
  - Snake contracts come from a book that is fixed when each player is drafted.
  - Proposed fixes for problems the scenarios exposed:
    - the minimum bid is the book price;
    - players come up in random order;
    - the pool is ranked by points per game with a games minimum;
    - owners who win no auction player take part in a round-2 lottery;
    - bids are sealed second-price.
  - The scenarios ran on Legacy's 2025 scoring.
