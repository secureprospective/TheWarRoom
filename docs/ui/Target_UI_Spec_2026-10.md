# TheWarRoom target UI: specification, revision 1 (2026-10-10)

Revision 0 is in git history. Christopher answered the §8 ring 1 questions on 2026-10-10.

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

- **States:** Draft → Blocked / Ready → Handed off → Not yet done / Not verified →
  Landed / Failed / Stale.
- **"Ready" means the app checked it, not that MFL accepted it.**
- **Landed** requires an observed MFL change that matches the row's full predicate. A partial
  observation reads "Not verified".
- A pending DOT review, a bid award or a waiver resolution are their own named lifecycle stages.
- **Unmapped MFL targets** stay drafts, with "MFL target not verified": add/drop form, propose
  form, survivor pick, option, re-sign, buyout, tender, UFA bid. They are closed in the roadmap's
  gap-closure work.

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
   held eligibility gates these Acts in ring 2. The evidence it uses is still open.
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
