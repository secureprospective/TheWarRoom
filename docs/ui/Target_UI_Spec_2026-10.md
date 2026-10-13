# TheWarRoom target UI: specification, revision 3 (2026-10-12)

Revisions 0–2 are in git history. Revision 1 closed ring 1 (2026-10-10), revision 2 closed ring 2
(2026-10-11), and revision 3 closes ring 3.

## What changed in rev 3

- §4: Control Room's third workspace is **League**, no longer "League (role-gated)". Its settings
  (M-091/M-097), by-laws (M-185) and accounting (M-098/M-197) are reads every GM may make. The two
  commissioner pages, the email log (M-093) and commissioner news (M-095), moved to ring 4 because the app
  runs as a GM (R1).
- §14 (new): what ring 3 built, by node, with the file for each claim.
- §15 (new): the ring 3 review and what is carried to ring 4.
- §15a (new): what the registry status `landed` means, distinct from an Act's Landed state (§6).
- Every kept ring 3 registry row is `built` (118 of 118). The 19 ring 3 rows still at `spec` are merged
  into a built place (17) or retired (2: M-075 Victory Points, M-192 option election).

## What changed in rev 2

- §6 gains the ProBoards hand-off lifecycle that ring 2 built for plans the league runs off MFL.
- §8: Q5 (IR/taxi eligibility) is built; new open questions for the ProBoards post format and the
  May tender / re-sign dates.
- §10 (new): what ring 2 actually built, screen by screen, with the file or registry row for each claim.
- §11 (new): Christopher's ring 2 decisions (D1–D11) and the panel rulings (PW1–PW6), in plain words,
  where they shape the screens.
- §12 (new): data ownership, from the facts-schema panel. **Not built**; it is the stand-alone app's plan.
- §13 (new): the ring 2 review and the gaps carried to ring 3.
- Every ring 2 registry row is now either `built` or deferred to ring 3 with a reason (§13). Deferred
  rows keep the status `spec` ("not wired" on screen): a new status word would grow the app's
  start-up bundle past its ceiling, because the registry ships inside it.

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

## 1a. Ring 1 findings against the rulings

Measured on Claude-OS, 2026-10-10, from Home. Command-bar typing costs 0 clicks.
MFL form steps are separate. Ruling 1 is unchanged: at most two in-app clicks, with MFL opened.

| Path | In-app clicks | Against ruling 1 |
|---|---:|---|
| Pulse Now | 1 | Within two; read path, not a ready move |
| My moves via nav | 2 | Within two; receipt list, not an MFL hand-off |
| My moves from Inspector | 1 | Within two; receipt list, not an MFL hand-off |
| My moves via command bar | 0 | Within two; receipt list, not an MFL hand-off |
| IR/taxi via nav | 4 | Does not meet ruling 1 |
| IR/taxi via command bar | 2 | Meets ruling 1 |
| One-swap lineup | 6 | Does not meet ruling 1 |
| Trade accept | Not walked | No pending offer; compliance not proved |

IR/taxi nav is HQ → player → Draft → Open MFL.
The command path is type → Enter → Draft → Open MFL;
Enter selects the player but is not a click. Lineup is HQ → Edit lineup → bench → start →
Check and save plan → Open MFL.

Ring 1 working budgets, approved 2026-10-09: Pulse 1, My moves 1, accept 3, IR 3,
lineup swap 5 (+2 per extra swap). Pulse and Inspector/command My moves meet those budgets.
Nav My moves, nav IR and lineup exceed them by one click. Command IR is within budget.
Accept is unmeasured. These budgets do not replace ruling 1; the gap needs Christopher's answer (§8).
Christopher said on 2026-10-10 that the lineup UI will be reworked later. No replacement is designed here.

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

## 4. Organization (GLM §4.1–4.2; three workspaces per node except Trade Floor)

| Node | Workspaces | Notes |
|---|---|---|
| Home | Seasonal card (owns the node) · alert tray · digests | phase router |
| War Room | Market · Valuations and pool · Research | Lab market, asset board, free-agent pool, bid tracker |
| Franchise HQ | Lineup and roster · Contracts and cap · My moves | Act plans, what-if, dead cap, app-local lists |
| Trade Floor | Trade desk · Offers history · Counterparties · Draft room | accept, reject, revoke, DOT state; draft timing inside the draft room |
| League Pulse | Now · Race · Pools / Archive | live game view and Classic/Default matchup in Now; report drawer for every MFL read report |
| Control Room | App · Data and sources · League | engine params gated to Admin; League is a GM read (commissioner pages are ring 4) |

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

- **States:** Draft → Blocked / Ready → Handed off → Not yet done / Not verified →
  Landed / Failed / Stale.
- **"Ready" means the app checked it, not that MFL accepted it.**
- **Landed** requires an observed MFL change that matches the row's full predicate. A partial
  observation reads "Not verified".
- A pending DOT review, a bid award or a waiver resolution are their own named lifecycle stages.
- **Unmapped MFL targets** stay drafts, with "MFL target not verified": add/drop form, propose
  form, survivor pick, option. They are closed in the roadmap's gap-closure work.
  Re-sign, buyout, tender and UFA bids are not MFL forms at all: the league runs them on
  ProBoards (ring 2, below).

### The ProBoards hand-off (ring 2)

Bids, buyouts and the contract Acts (tender, re-sign, franchise tag, extension, restructure) are
things MFL cannot do; the league runs them on ProBoards (decisions D3, D5, D11 in §11). The app
prepares the post; it never posts. Each plan moves through these states:

- **Draft → Ready or Blocked.** Go checks the rules when the plan is kept. A Blocked plan says why.
  A rule the app cannot check (for example, another owner's competing offer) is a "Not checked"
  note, not a block (`internal/envelope/contract.go:66`).
- **Handed off.** "Copy request and hand off" (contract Acts) or "Copy ProBoards post" (bids) puts the
  post text on the clipboard (`frontend/src/app/contracts/ContractActs.tsx:46`,
  `bids_plans_app.go:154`).
- **Pending.** For bids and contract Acts, the owner presses "I posted it" and the plan waits for
  MFL's own data. The label reads "Bid pending" for a bid and "Pending" for a contract Act
  (`frontend/src/app/shell/moveLabels.ts:25`).
  Cut and buyout plans have no such step: after the hand-off they read "Not yet done" until the
  rosters export shows the player gone.
- **Landed, Not verified, Stale or Withdrawn.** Landed needs MFL's data to show the result: the player
  gone from the roster for a cut or buyout, every planned contract cell for a contract Act
  (`internal/envelope/release.go:107`, `internal/envelope/contract.go:139`). Not verified means MFL's data cannot prove it (future cells
  are inferred). Stale means something changed underneath the plan (§11, PW2); it can be repriced.
  Withdrawn is "I didn't post it" (or "I didn't drop him" for a cut) after a hand-off, or
  "Remove plan" before one (`release_app.go:114`).
- A bid that someone else wins ends as **Awarded off-app**, the end of a plan rather than an error
  (`internal/envelope/bid.go:102`).

### Ring 1 Acts built and proved

- **lineup.set:** built and Landed live on 2026-10-09. The +Kiner −Perine plan Landed at 21:44:23Z.
  The revert Landed at 21:47:25Z. MFL was restored to the original lineup.
  Landing compares the full saved starter set for the explicit week against a changed baseline.
  MFL saves only on the bottom **Submit Partial Lineup** button. Ticks alone save nothing.
- **trade.accept:** built, not yet Landed live. The hand-off opens the safe trade desk (O=05).
  Offer disappearance is ambiguous; it does not prove acceptance or execution.
  The DOT window is seven days from first entry into DOT review, using MFL
  `defaultTradeExpirationDays=7`. After the window, unchanged evidence stays Not verified;
  inside it, stale-read Not verified can re-enter review. A later matching execution can still land.
- **roster.ir / roster.taxi:** built, not yet Landed live. MFL pages O=18 / O=98 were verified
  on 2026-10-10. Taxi has both directions; direction comes from held roster status.
  Eligibility is not held. Ready does not mean MFL will allow the move.
- **Deferred to ring 3:** trade.propose / trade.reject / trade.revoke. Their MFL forms were never
  captured; revoke is an irreversible GET. Never use that action link as an automatic hand-off.

Evidence: `docs/build-handoffs/RESUME-2026-10-09-ring1-p7.md` §§4–7 and the ring 1 run `PLAN.md`.
IR/taxi implementation is recorded in phase reports p7c1b and p7c2; 10-10 page checks are review findings.

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

Answered items retain their answers and dates. Remaining questions are owed before their ring.

1. **MFL owner-form targets:** partly answered, 2026-10-09. lineup.set and trade.accept are built;
   propose/reject/revoke are deferred to ring 3. IR/taxi pages were verified 2026-10-10, not Landed.
   Other unmapped targets remain open. Which missing forms and outcome captures should ring 3 obtain?
2. **Waivers:** direction answered, 2026-10-07. They run on ProBoards, not MFL; TheWarRoom builds
   its own waiver code. Exact mechanics remain open: what native claim/order flow is required?
3. **DOT review:** answered for ring 1, 2026-10-07. Trades only; three DOT approvals, then MFL
   commissioner approval. Review shows in the trade plan and My moves rail. Landed needs the
   public TRADE row, not disappearance. The seven-day review window was approved 2026-10-09.
4. **Window time zones:** answered, 2026-10-07. Display in the computer's local time zone.
   Window edges and their mapping to the eight stretches remain open.
5. **IR/taxi rules:** source answered, 2026-10-07. MFL settings govern for now. The settings page
   says IR classification and taxi experience under three years (`Ring1_Gap_Closure.md` §8).
   Eligibility is not held in the app. On 2026-10-10 MFL showed Perine Not-Eligible for IR and
   "Cannot be demoted" to taxi while the app offered those moves. **Answered 2026-10-10:**
   held eligibility gates these Acts in ring 2. **Built in ring 2:** IR needs an IR designation
   in MFL's injuries feed; taxi needs under three years of NFL experience from the players
   feed's draft year (`internal/engine/eligibility/eligibility.go:10`). An unknown verdict says
   "Not checked; MFL will decide" and leaves the Act available.
6. **Victory Points:** answered, 2026-10-07. Not used; the 2026 league export has no such setting.
7. The 14 unnamed hub link kinds.
8. Whether MFL's exports cover pools and contract fields.
9. Refresh cadence during live clocks.
10. Minimum desktop width.
11. A data-scope ruling before multiple leagues are supported.
12. **Browser hand-off:** each hand-off opens a new tab. The OS browser decides; there is no clean
    fix recorded. **Answered 2026-10-10:** no app change. Whoever tests closes the MFL tabs
    they opened when the work is done.
13. **Click-budget gap:** nav IR/taxi is 4 and lineup is 6, versus ruling 1's 2; nav My moves is
    2 versus its working budget of 1. Accept has not been walked. **Answered 2026-10-10:** the
    ring 1 working budgets stand as an exception. Ruling 1 is unchanged.
14. **Lineup rework:** **answered 2026-10-10:** it is part of the eventual move away from MFL,
    when TheWarRoom becomes its own stand-alone app. Not scheduled for ring 2.
15. **ProBoards post format (ring 2):** open. The post text the app writes for bids, buyouts and
    contract Acts has not been checked against what the commissioner expects. Christopher to confirm.
16. **Tender and re-sign dates (ring 2):** open. The registry says May 1–3 (M-193) and May 10–15
    (M-195); the rulebook has no such dates. The Acts say "Offseason only" without dates.
    A tender, re-sign or tag always starts the new contract at the season after the held contract
    expires (expires + 1), so only eligibility (`expires == season`) depends on when the app rolls over.

## 9. Ring 1 review

**What worked**
- The lineup envelope Landed twice live on 2026-10-09: the change and its revert.
- Persistent receipts and the watcher retain Not verified plans and can land later matching evidence.
- Home seasonal/alert surfaces, HQ lineup/roster, Classic Pulse Now and the trade desk were built.
- Inspector IR/taxi drafts open the correct MFL pages. Neither intent has Landed live.

**What revision 0 got wrong**
- The two-click target was not met by nav IR/taxi or the one-swap lineup path. Working budgets differ too.
- The state line omitted Not verified. A failed verification is not proof the watcher stopped.
- The roadmap's seven first Acts were too broad for captured evidence. Propose/reject/revoke moved to ring 3.
- Ready checks did not establish IR/taxi eligibility. MFL refused the offered Perine moves.
- "Submit Lineup" was not MFL's button label. Only bottom "Submit Partial Lineup" saves.
- Registry status `spec` labelled built Acts "not wired". Built and live-Landed now differ explicitly.

**What changes for ring 2**
- Keep ruling 1 unchanged. Ring 1's working budgets are an approved exception (§8 Q13).
- Gate IR/taxi on held eligibility (§8 Q5). The lineup rework waits for the stand-alone app (§8 Q14).
- Keep trade.accept and IR/taxi marked built, not Landed live, until their live evidence exists.
- Carry the missing trade forms and revoke safety constraint to ring 3.
- Ring 2 remains the decision layer: Lab, contracts/cap, free-agent pool and bid tracking.
  Its gate still requires Lab test parity, "How this is counted" and spec revision 2.
- Ring 1 is not approved by this draft. Christopher's go is still required.

Evidence: `Ring1_Gap_Closure.md` §§1, 8; `RESUME-2026-10-09-ring1-p7.md` §§4–7;
the ring 1 run `PLAN.md` and phase reports p2a, p2b, p3b, p4b, p5b, p6b, p7b, p7c2, p7e;
`UI_Target_Roadmap_2026-10.md` Rings 1–2. The 10-10 walkthrough/page findings are recorded in §1a and §8.

## 10. What ring 2 built, screen by screen

Ring 2 is the decision layer. Every claim here points at a file or a registry row; the audit with
receipts is the p2l report in the ring 2 run folder. Nothing in ring 2 posts to MFL or ProBoards.

**Shared by every analytic screen**
- A fold-out "How this is counted" on every Lab reading, with the count and the threshold it was
  judged against, the baseline, the archive cutoff, the method, the source, the input fingerprint
  and the build (`frontend/src/app/market/ReadingView.tsx:21`, registry L-33). The wording is
  written in Go; the screen only shows it.
- Lab figures are frozen archive history through the cutoff, not today's market, and the screen says so.
- A spread is drawn as a range bar with the 100% baseline marked, never as a grade
  (`ReadingView.tsx:9`; D4 in §11). Only a result that clears chance is marked best or worst.
- A reading with too few records says "Too few to judge", with the count and what it needs.

**War Room › Market** (L-01–L-14, L-33, L-35)
- Two views: Trade Market and Draft lab (`frontend/src/app/market/Market.tsx:44`).
- Trade Market: what the league pays for a win, the buy and sell lists (paid and delivered), the pick
  clock and trades per season by stretch (`Market.tsx:154`). Filters for time of year, asset kind and
  age split (L-06–L-08).
- Any asset opens its profile in the Inspector: why it is cheap or dear, its cheapest and dearest
  stretches, how it compares with other assets, what came back in its trades, and who buys it (L-09–L-13).
- Draft lab: return as a share of the slot, by season after the draft, by NFL round and by age;
  "plays like pick #N", hit rate, kept by year two, and draft it or buy it (L-15–L-22,
  `frontend/src/app/market/Draft.tsx`). Draft lab has its primary home in Trade Floor › Draft room; the Market button is a second route to the same lazy component and commands.

**Trade Floor › Draft room** (L-15–L-22)
- The same lazy Draft lab as the Market route: timing, plays-like-pick, return and draft-or-buy. MFL draft reports and live drafting remain deferred.

**Trade Floor › Counterparties** (L-23–L-29, L-36)
- Filters for standing on trade day, time of year and what the trade did (bought for now, built for
  later, like for like) (`frontend/src/app/counterparties/Counterparties.tsx:19`).
- Tier cards, each tier's trend over time, tier against tier, and a sortable 32-team table; a team
  opens a panel with its season cells, what it sent and took, and its partners
  (`frontend/src/app/counterparties/TeamPanel.tsx`). Standings are as of the archive cutoff.

**Trade Floor › Offers history** (L-30–L-32)
- How offers ended across the eight stretches, with sent / received / all. The header always says
  that MFL shows only the GM's own offers (`frontend/src/app/offers/Offers.tsx:21`).

**War Room › Valuations and pool** (M-039, M-056, M-194, C-06.7)
- Every free agent MFL lists, with position, team, age, experience and the rulebook §6 minimum
  salary, filtered and sorted (`frontend/src/app/pool/Pool.tsx:145`). Source and as-of are shown.
- A bid plan per player: salary, years and bid points, checked against the bid rules in Go
  (at most 30 open bids, `internal/engine/bid/rules.go:13`). It is the GM's own plan only; nothing
  about other teams' bids is stored. The plan follows the ProBoards hand-off in §6.

**Franchise HQ › Contracts and cap** (M-078, C-06.5, M-031, C-06.1, C-06.6, C-06.3, M-036, M-196, M-193, M-195)
- Every contract in the league, my team first, in exact cents with Go's own money text; a player's
  contract years; my franchise cap; MFL salary adjustments, labelled as MFL adjustments and not
  proven dead cap. MFL's can't-cut/can't-add lists are retired and not used by the War Room.
- **What-if:** each row of my team has one "What if…" menu: cut, buy out, or Contract Acts. The
  what-if shows the charge, my cap before and after, Undo and Reset. It keeps its starting numbers until
  Reset; if MFL changes underneath it, a banner says so (`whatif_app.go:273`).
- **Cut and buyout plans** come from the what-if ("Make this a plan"). A cut asks for confirmation
  ("It cannot be undone"), then opens MFL's drop page (M-036). A buyout is a post to the commissioner
  on ProBoards (M-196) (`frontend/src/app/contracts/ReleasePlans.tsx`).
- **Contract Acts:** RFA tender, re-sign, franchise tag, extension and restructure, offered only
  where they apply. The "Contract move" panel shows the price, the team cap before and after, and
  any "Not checked" notes before the plan is kept; one open plan per Act and player
  (`frontend/src/app/contracts/ContractActs.tsx`). The option election (M-192) is not built.

**Franchise HQ › My moves**
- Bid plans, cut and buyout plans and contract Act plans list here beside the ring 1 receipts
  (`frontend/src/app/shell/MyMoves.tsx:46`).

**Inspector: IR and taxi eligibility**
- The IR and taxi Acts now carry a verdict: eligible, not eligible with the reason, or not checked.
  Not eligible turns the Act off; not checked leaves it on (§8 Q5).

## 11. Ring 2 decisions that shape the screens

Christopher's decisions (D) and the panel rulings Claude adopted (PW), in plain words.

- **D1 The Lab:** the app rewrites the Lab's math in Go and reads a frozen, fingerprinted copy of the
  Lab's data. The Lab's code is only the reference the tests check. Its own data pipeline comes
  before the stand-alone app (§12).
- **D2 Taxi age:** MFL's rule, under three years of experience.
- **D3 Bids:** the app holds its own bid and claim plans, built to become the league's system later;
  for now it holds plans and honest status, nothing invented about other teams.
- **D4 Ranges, not grades:** show the range bar; the solid / rough / thin grade stays inside the engine.
- **D5 ProBoards hand-off** for option, tender, re-sign, extension, restructure, tag and free-agent
  bids: MFL has no owner pages for them.
- **D6 What-if:** keeps its starting numbers until Reset; unknown eligibility leaves an Act available.
- **D7 Playoff bids:** the $12M / $24M year limits stay in the playoffs; only playoff teams may bid then.
- **D8 Offers test data** may stay in git until a backend server holds it.
- **D9 Buyout what-if in season** is allowed, labelled "Offseason only".
- **D10 Rules are league settings, not fixed in the screens.** No rules-profile work in ring 2.
  Rules stay in Go, never in React.
- **D11 Buyouts** go to the commissioner on ProBoards; the app prepares the text. The app never makes
  a real roster move on MFL.
- **PW1** The what-if is a separate simulation; plans are drafted from it and it never writes MFL's copy.
- **PW2** A Ready plan goes Stale when its player's contract changes, a new dead-cap row appears,
  its window closes or the player leaves the roster.
- **PW3** Bids and claims are plans with one side table of bid facts. Being outbid is an ending,
  not an error. Nothing is stored about other franchises.
- **PW4** Landed: cuts and buyouts when the player leaves the roster; contract Acts when MFL's
  contract cells match. A dead cap that differs from the plan makes it Stale, not Failed.
- **PW5** Dead cap is charged per contract year: 35% (50% if restructured) of each paid year after
  the cut season, rounded once, snapped once to $10,000; nothing if claimed or in the final year
  (`internal/transactions/deadcap/deadcap.go:23`).
- **PW6** Future contract years are filled flat from MFL's salary and expiry. That is an inference,
  and "How this is counted" says so.
- **Go writes the words.** Rules, figures and the sentences that explain them come from Go; React
  lays them out (panel ruling W1). This includes the contract Act form: the Act names, the tender
  levels with their prices and the term lengths come from Go (`contract_acts_app.go:157`), read from
  the table that prices the tender (`internal/engine/contractacts/terms.go:40`).

## 12. Data ownership (from the facts-schema panel) — NOT BUILT

This is the stand-alone app's plan. None of it is in ring 2. The full triage is
`docs/build-handoffs/Facts_Schema_Panel_2026-10.md`.

- **FS1 No second store of facts.** The app compiles MFL's archived responses straight into the
  form the engines already read. The compiled copy is rebuilt whenever the chosen inputs, the
  compiler or the rules change. A failed rebuild keeps the last good copy, marked stale.
- **FS2 Two proofs.** For the frozen cutoff, the compiled data must match the Lab's frozen data row by
  row, and the existing parity tests must pass on the compiled data. The frozen pair stays as a
  permanent test.
- **FS3 MFL facts and the app's own events stay apart.** Facts from MFL can always be rebuilt. The
  app's own plans, bids and ledger stay in their own stores, never mixed into MFL's history.
- **FS4 An input manifest first.** Before compiling, the app pins exactly which archived files it
  uses. It never picks "the newest copy". A missing frozen file blocks a parity claim.
- **A prerequisite, s0:** bring the 2013–2026 league archive into the app's own archive as a
  registered source.
- **The league_event direction:** one table of league events that is only ever added to, never edited
  or deleted.
  MFL stays the authority until a cutover date; after it, the app's own events are. The rule is
  never "newest timestamp wins". This is the spine for the commissioner's office later.

## 13. Ring 2 review

**What worked**
- The Lab numbers on these screens pass the parity tests against the Lab's own JavaScript:
  market, draft, counterparties and offers, with the planted wrong values caught. The one figure the
  app adds itself, the trades-a-season count on the Counterparties standing filter
  (`counterparties_app.go:268`), has its own parity test (`counterparties_app_test.go:357`).
- Rules live in Go. The money is exact cents end to end, with one rounding and one $10,000 snap.
- Plans for things MFL cannot do have one honest lifecycle (§6) instead of pretending MFL accepted them.

**Carried to ring 3**
- The Draft room’s MFL draft reports (M-063–M-183); its analytic Draft lab is built.
- The option election (M-192), the acquisitions screen (C-06.4), and the projected stats, ranks, ADP
  and AAV feeds (M-046, M-047, M-067, M-069).
- Contract cells for future years are inferred, not read from MFL year by year (PW6).
- Act windows are phases (offseason / any time), not dated windows.
- The ProBoards post format (§8 Q15) and the tender / re-sign dates (§8 Q16).
- The stand-alone data pipeline s0–s5 (§12) and the checked-facts source view (L-34).

**Deferred registry rows** (ring 2, status `spec`; merged rows follow their targets)

| Rows | Why it is not built |
|---|---|
| M-046 | No projected-scores feed is held; ring 2 built only the free-agent pool and bid plans here. |
| M-047 | No player-ranks feed is held; ring 2 built only the free-agent pool and bid plans here. |
| M-063, M-064, M-065, M-066, M-087, M-102, M-176, M-179, M-183 | The MFL draft reports are not wired; the Lab's draft numbers (L-15–L-22) are in Trade Floor › Draft room and reachable from Market. |
| M-067 | No ADP feed is held; ring 2 built only the free-agent pool and bid plans here. |
| M-068 | The MFL draft reports are not wired; the Lab's draft numbers (L-15–L-22) are in Trade Floor › Draft room and reachable from Market; no future-pick ownership feed is held. |
| M-069 | No AAV feed is held; ring 2 built only the free-agent pool and bid plans here. |
| M-103 | The MFL draft reports are not wired; the Lab's draft numbers (L-15–L-22) are in Trade Floor › Draft room and reachable from Market; the app-local draft list is not built. |
| M-192 | No option Act is built; contract Acts cover tender, re-sign, tag, extension and restructure (ContractActs.tsx:13). |
| L-34 | The stand-alone data pipeline (s0–s5, Facts_Schema_Panel_2026-10.md) is not built; Control Room › Data and sources is not wired. |
| C-06.4 | No acquisitions screen is built (the Go roster handlers in internal/transactions/acquisitions have no Franchise HQ surface); free-agent adds are checked only by the bid plan rules (C-06.7). |
| M-163, M-175, M-177, M-178 (merged) | Their targets (M-037, M-176, M-183) are not built. M-162 and M-187 are merged into built places (Valuations and pool, M-039). |

## 14. What ring 3 built

Ring 3 was breadth and depth: every MFL read a GM uses gets a home, and the league's history becomes the
app's own checked facts.

- **League Pulse.** Now: my matchup, every matchup, results, recaps, the NFL schedule and box scores, and
  the live game: MFL live scoring (M-001), the game view (X-01) and the fantasy play spine (X-02), with
  finished games read once and held across restarts (`livegame_app.go`, `app/now/`). Race: standings, power,
  seeding, tiebreaks, brackets, all-play, schedule and lineups. Pools / Archive: every season from 2013
  read from the checked facts, one Pools report for the pool endpoints (`archive_app.go`, `Archive.tsx`).
- **War Room.** Research: search, the Inspector's facts, depth, injuries and batched MFL news
  (`research*.go`, `app/research/`). Valuations and pool: MFL projections, ranks, ADP and AAV
  (M-046/047/067/069) and the **asset board** (C-01, `asset_board.go`, `app/assets/`): every rostered
  player's points per game with his rank at his position, and his dynasty value and market price from the
  Lab's value model, also on every player card, the roster table and the Inspector.
- **Trade Floor.** The trade builder with propose, reject and withdraw hand-offs and the acquisition cost
  (p3t); the Draft room's MFL draft reports (M-063–M-183) beside the Lab's Draft lab, the NFL draft
  (M-087) and the app-local draft list (M-103).
- **Franchise HQ.** Incoming contract and bid-plan cost by season (C-06.4); lineup and roster read as
  loading, never failing, while MFL answers at launch.
- **Home.** IR alerts in the alert tray (M-149), digests, and the shell rows: presets (S-05), market
  filters kept per workspace (S-10), private watch, bait, note and draft lists (`app/personal/`).
- **Comms.** The league feed, the message board and polls read with the MFL key, and the ProBoards
  hand-off for the league's own board.
- **Control Room.** Data and sources: archive import, the frozen input manifest, each source's health
  and the checked facts with their nine checks (L-34, `data_sources_reading.go`). League: settings,
  by-laws and accounting. App: MFL's site news as articles (M-141) and the presets.
- **The facts and the Lab in Go.** `internal/facts` compiles the league archive into checked facts
  (37 relations, nine checks, exceptions pinned by identity); every franchise-season matches MFL's own
  standings pages (160 of 160, four known differences named). `internal/labcompile` compiles the Lab
  from those facts, including the full value analysis: 508,219 values match the Python reference with no
  difference. The app fits the value model after launch (`fitAssetModel`) and C-01 reads it.
- **Player photos** from ESPN, then Sleeper, cached for personal use, with an on/off switch.

## 15. Ring 3 review

**What worked**
- Sol built and Claude reviewed each phase with a planted mutant: in the Lab value analysis an age-band
  edge moved by one year failed both the goldens and the full-archive comparison (1,291 trade
  differences), and the clean code has none.
- Proof against MFL itself, not only against our own reference: the fact exam reads MFL's standings pages.
- The live look on Claude-OS before Christopher's click-through caught what tests did not: a player with
  negative points per game failed the whole asset board, and card qualifiers overflowed their tiles.

### 15a. Registry status: built and landed

The registry's `status` column (`docs/ui/endpoint-registry.csv`) says how far each row has come:

- **spec:** designed, not wired.
- **built:** the code is merged and its tests pass, but nobody has yet watched it work against the league.
- **landed:** seen working on Claude-OS against a snapshot of the live database: its data, its empty state
  where the league has one, and its Act, if it has one. The row's `ring_note` says when it was seen
  and names the screenshot in the ring's run directory.

A registry row that is `landed` is not the same as an Act that is Landed (§6). An Act is Landed only
when MFL shows the change the app planned. A row is `landed` when the screen does its job. For an Act
row, that means its hand-off opens correctly; the app itself never submits to MFL. M-027, Submit Lineup,
is both: its envelope Landed on MFL on 2026-10-09.

The landing walk of 2026-10-12 marked 70 rows `landed`. Its findings are the fix queue for the rows that
are still `built`.

**Carried to ring 4**
- The commissioner pages (M-093 email log, M-095 commissioner news): the app runs as a GM (R1).
- Waivers (M-024, M-156, M-157): they run on ProBoards; native claim and order mechanics are open (§8).
- Owner-only MFL communication pages (M-125–M-127): their addresses were never captured.
- Acts the league does not run: pick'em and survivor selection (M-172, M-173, M-198); awards (M-081)
  have no archived MFL report.
- NFL team stats (M-050) need nflverse team box-score ingestion; draft preferences (M-108) and cache
  tools (M-137) have no captured MFL address; the league switcher (S-14) waits for portfolio support.
- The Lab screens still read the frozen Lab document; switching them to the runtime compile is ring 4.
- Dead cap and form on the Inspector stay "not wired" until the engine has them.
