# TWR direction grilling — 2026-11-22 (planning only, no build)
Source: Christopher's answers, round by round. Feeds North Star v3, product ADRs, reset plan for Tom.

## Round 1 rulings
1. Mission keeps the juggernaut long-term aim + near-term target: through 2027, one real 32-team league
   runs a full season on TWR; Legacy never put at risk. Legacy will likely be the FIRST real migration
   (that league trusts him most). Until July 2027 launch: dummy test leagues of different sizes and shapes.
2. Two roles: MFL leagues (TWR companion, MFL is the record) and native leagues (TWR is the record).
3. Build for Christopher's solo use first until ~March; exception: bystander-visible phone live scoring gets full polish now.
4. Feature tests (a general guideline, not a gate): saves time / makes someone ask "what is that?" / protects the league.
   Must stay inventive — we are bucking a weak FFB app market and don't know everything.
5. Rule split: he rules league behaviour, what owners see, priority, money, outward. AI rules code, structure,
   models, tests; pitches "would this work?" with a recommendation; states the cost of a ruling before it locks;
   ALSO flags assumption risk (where a plan rests on something unproven).
6. Tom builds, Claudebox plans and reviews against direction docs; Tom never edits direction docs without his ruling. For this week's build.
7. Decision records split: product (plain words, for him) and technical (for agents); drop file:line refs; retire 0002.

## Round 2 rulings
8. NO plan for which league moves first (Legacy vs startup). He doesn't know how it plays out. Don't predict.
9. Test leagues from real shapes, agent-played owners — ALSO tests AI placeholder-owner behaviour.
10. Engines already save time in every commish area. The hard part (and the market's failure) is the UI/UX
    shaping of each commissioner job. Off-season focus = shape the UX of those jobs.
11. No weekly-wow schedule. He threads that into builds himself.
STANDING (from 8/11): real life is impossible to predict — AI's job is to keep us PREPARED (options ready),
   not to make plans about real-world outcomes.
12. Assumption-risk flag format: "This assumes X. If X is wrong, it costs Y. Cheapest check: Z."

## Round 3 rulings
13. New glossary term **Stand-in**: an AI holding an empty seat, playing the team like an owner (lineups, bids,
    trades) until a real person claims it. Caretaker stays commissioner-only. (Apply on the docs branch.)
14. Stand-ins: test leagues only for now; real-league use is an open question (assumption: AI trades feel fair).
15. Off-season direction: every commissioner job redesigned until faster and clearer than the market; mock-first.
    Keep minds open to possibilities — the line is a direction, not a fence.
16. Market study per commissioner job before any mock. He has signed up to Sleeper, ESPN, Yahoo (more coming)
    for the AI to soak later. (Read-only: never make real moves in those accounts.)
17. North Star v3.0 folds rounds 1-3; Tom can't edit it without his ruling.

## Round 4 rulings (part 1 closed)
18. Tracker on Hermes is the one place for mid-build steering. Sessions MUST keep it current, plain English,
    organised, without him nagging (last week updates were often confusing and late).
19. (Not answered; assumed yes) Tom briefs carry done lines, quoted rulings, assumption flags, "what the owner sees".
20. Stop Tom only for: hard limit, clash with a product ruling, an assumption proven wrong. Else log and keep moving.
21. Part 1 done. He feels it is maturing; this session fine-tunes rough edges.

## Part 2 (workspace clean) — Q22-27 all accepted 11-22; grilling stopped there by Christopher.
22 archive-branch the 596 dirty files then main checkout to main; 23 worktrees: remove merged+clean, back up >7d idle to CT105, never touch <7d;
24 delete merged branches, bundle unmerged >30d to CT105 first; 25 README only front door, North Star direction, docs/resume/ one per workstream, HANDOFF.md retired;
26 superseded docs -> docs/archive/ + index; 27 proof = cold-start agent finds current work/direction/rules in 5 min, transcript on tracker.

## North Star v3.0 (11-22)
- Committed f0d78d432 on session/direction-docs-1122 (not pushed). Rulings: real-player photos allowed (no NFL logos); standalone desktop app + old local mode under review in the reset, nothing new built on them.

## Part 3 (Stages 4-6) — all yes 11-22
Cut = clashes with a ruling; unproven = keep + flag; slop = unused / duplicate / owner-confusing / fake tests (not style);
order = live phone, league safety, agent speed, tidying; Tom deletes unused freely, owner-visible changes need Christopher;
weekly habit check on tracker; start after v51 + 32-team load test; game days NOT a hard stop for this work (only users).

## Part 4 (rehearsal season) — 11-22, all accepted; full text TWR-REHEARSAL-SEASON-1122.md
Two native 32-team leagues (default + twin-pool), Christopher 1 team each, others auto best-projected; Stand-ins no date;
draft in existing PWA draft room, no clock for him, instant auto-picks, test drafts first, stream if clean; start Nov 29 lab;
laptop = season server (Sat updates w/ OK, frozen game days); lab open for live fixes on build Sundays; Monday report card;
wall = real live screens by Dec 5; Dec 6 recording + overnight phone/tablet size sweep + real S8/iPhone live session on lab.
Standing: reuse what exists.

## Part 5 (users, ICP, how we serve them) — 11-22 evening; research notes in /root/twr-icp-research-1122/
Q1. "No app does everything well" = for this work, EVERYTHING a dynasty / big-league commissioner and owners patch together today
    (host + trade values + contracts + history + chat + dues). Big leagues are "FFB of the future". Smaller dynasty and redraft
    are still supported and worked with, but are not the focus.
Q2 (partly answered). Five pieces owned (host, values, contracts, history, chat) not objected to. DUES: investigate linking LeagueSafe
    (or similar) so owners stay in TWR's interface and LeagueSafe does the money part; TWR never holds money. Research first.
DOCKET (Christopher 11-22): plan a "super hardcore" marketing campaign after Jan 1 2027 targeting leagues displaced by NFL.com
    fantasy moving to ESPN (announced 2026-07-16). Christopher sets up domains/website ahead. GATE: in-season network/setup must be
    proven (rehearsal season) so leagues we soak up survive on TWR. Move fast.
Q3 (11-22). Pitch: "you already moved once; move again to something better" (NFL.com->ESPN leagues). Add simulated REDRAFT leagues
    10/12/16 teams alongside the two 32-team leagues (laptop if headroom after Nov 29, else Lab). Check native redraft mode exists.
Q4 (11-22, Christopher's technical directive). One server = one copy of every NFL fact (live box score, schedule, players, injuries,
    news...), every league reads it locally; nothing league-neutral fetched or stored per league. Plan it cleanly. Tom: audit first.
Q5 (11-22). Order accepted: v50/v51 + rehearsal setup -> Tom's shared-facts audit -> redraft leagues land on clean design.
Q6 (11-22, Christopher). PROJECTIONS: pull real NFL STAT-LINE projections (yards, catches, TDs...), never a site's fantasy-point
    number; compute each league's projection from its own scoring rules. Applies regardless of what exists now. Research sources.
    Research back (08-stat-projections.md): Sleeper free stat lines incl. IDP (unlicensed, undocumented); FantasyPros personal-only +
    no competing products; SportsDataIO commercial; nflverse free for build-our-own. Pending Christopher: source plan.
Q7 (11-22). Three-step projection plan ACCEPTED: rehearsal on Sleeper stat lines (+snapshots, blend with history range) -> build OUR
    OWN model (winning prescription) -> before paid leagues pick own model vs licensed feed. Pull all past data possible (Sleeper past
    weeks, MFL archive, nflverse; ask about PFF). Model must handle depth: backups/3rd-stringers inheriting a starter's role
    (32-team leagues roster them). FUTURE: public TWR research site (e.g. which team's backup LBs are worth stashing) as a funnel
    into the app. Public use needs our own model or a licensed feed (not Sleeper data).
Q8 GATE (11-22, accepted) before any outside league joins: (1) rehearsal season clean through fantasy playoffs on season server —
    no data loss, live scoring holds every game day, clean Saturday swaps, one deliberate rollback proven; (2) all shapes run: two
    32-team + 10/12/16 redraft; (3) shared NFL-facts design in + 32-team load test passes on it; (4) history in AND out: import past
    seasons from ESPN and MFL (champions, records, H2H, old rosters) + full league export. ESPN import on Tom's list in December.
Q9 (11-22, accepted). Must-match before campaign = requirements: Sleeper-speed live scores + alerts; full lineup on one phone screen;
    automatic stat corrections showing what changed; MFL-level settings + history. Chat + draft board solid, not out-Sleeper Sleeper.
    NUMBER ONE (Christopher): the UI/UX decides stickiness. Desktop is close to best in business; PHONE IS NOT CLOSE, a long way off.
    Phone must economise tiny screen space, be intuitive and inventive, may change by league type (redraft != dynasty).
    "Good enough is not good enough." Christopher finds phone direction hard -> AI must bring strong phone direction options.
Q10 (11-22, accepted). Phone designed around the owner's MOMENTS of the week; league type decides what each moment shows.
    Phone market study (Sleeper/ESPN/Yahoo, read-only) + 2-3 mock directions on a real phone, starting after Dec 6.
    PRINCIPLE: the app brings information forward by moment. The TODAY tab is the home of "what matters right now":
    e.g. redraft off-season -> commissioner nudges owners to re-sign up and pay dues; 5 min to kickoff -> app opens on live games.
Q11 (11-22, accepted). ONE Today tab across all of a user's leagues, ranked by urgency/time, tap -> that league's own look.
    Today tab = "our winner", the stickiness nexus and the default place the app opens (except moments like kickoff -> live).
    Own future SESSION + build path: very thorough multi-pass research (fantasy player psychology, complaints, psychology of app
    use — why people love/hate/keep apps, habit/notification/attention research), following leads pass to pass, BEFORE any design.
Q12 (11-22). Gaps section leads with the FUN LAYER + LORE (recaps, power rankings, awards, trophies, Hall of Fame).
Q13 (11-22, accepted). The seven social-moments rules (docs/research/social-moments-2026-11-08.md §2) are the fun-layer foundation;
    rule 5 updated: real-player photos allowed per North Star v3, NFL logos still not.
Q14 (11-22, accepted). Awards = built-in automatic set (champion, runner-up, high week, best trade, title-winning pickup, bust, last
    place...) + commissioner-made named trophies (by rule or by hand). All perpetual; each year's winner added like an engraved plate;
    imported history fills old plates. CUSTOMIZATION across the app: owner profiles, fun customizable lore; league reskins later
    ("another ball of wax"), standard look stays default. Reuse proven patterns. History/archive/trophy case live deeper in the app,
    not front and center (moments surface on Today).
Q15 (11-22, accepted). Weekly recap: Tuesday morning (after MNF + first stat corrections), voice picked by commissioner
    (clean -> full roast), facts only, offered in app, one tap to share.
Q16 (11-22, accepted). Two power rankings for dynasty (this season + dynasty outlook), one for redraft; in Tuesday recap with arrows.
    ENGINE EXISTS (internal/powerrankings, M2: roster model values as best lineup/top N/whole roster, on-field-now or dynasty,
    z-blended with results at 4/(4+weeks played)). Gap: native leagues have no power-rank input yet. Improve it NOT by hand:
    lessons from the own-projection research/model (step 2) + candidate inputs (potential points, consistency, schedule strength),
    proven by replay against past seasons like D11.
STANDING (11-22): TWR houses leagues as a SANDBOX, not a rigid box (governance/trade fairness will be quick: tools, league decides).
Q17 (11-22, open). MINI GAMES — keep every owner opening the app even when their team is bad; rewards in league currency
    (e.g. compensation round-2 pick for best season-long prediction %). Brainstorm list given; Christopher to pick/add.
Q17 (11-22, accepted). Mini-games list accepted as the starting catalogue (prediction, off-season prediction, elimination/cup/toilet
    bowl, skill, social, one-week mini draft); commissioner toggles each + sets reward; rewards in league currency only, never real
    money. Christopher will add tidbits later. DYNASTY STOCK MARKET -> public WEBSITE attraction (Kalshi/Polymarket-style read of
    what people think players are worth), not in the app. Pretend money only.
Q18 (11-22). Governance set accepted with change: rules NOT printed next to things; instead a quick click from the moment to the exact
    rule (e.g. UFA frenzy -> veteran minimum rule one tap away). WIZARDS: every league-type preset gets a setup wizard (scoring,
    mini-games, everything); wizards wherever commissioners do work they dislike. Audience skews older, may not use the AI
    plain-language path -> wizards are the main road, AI optional. 32-team first-timer must not be overwhelmed. Goal: so cool that
    owners start their own league if the commissioner quits.
Q19 (11-22, accepted). Wizard order: league import (ESPN/MFL) + new-league setup -> off-season rollover -> owner replacement -> rest
    (draft, playoffs, rule vote, new-owner welcome tour); "start your own league" path everywhere. Every wizard step = same commands as
    settings/AI. NO TIMELINE: wizards need a solid, well-tested foundation first (gate part 4 import engine comes before its wizard).
Q20 (11-22, accepted). Trade fairness: value scale on every trade (now + future picks); commissioner picks review style; private
    suspicious-trade heads-up to commissioner only, never auto-block. DEFAULTS: redraft = no review; dynasty = review ON, which
    makes the commissioner choose how many reviewers (a real-time decision gate). Christopher's estimate (unproven): ~half of leagues
    are mostly strangers -> setup wizard asks "do most owners know each other?" to suggest the default.
Q21 (11-22, accepted). Draft assistant in the draft room, reads league settings, commissioner can switch it off; plus a hook for an
    AI assistant into the draft room (Christopher's idea for the next draft-room level).
    WIZARD = choose-your-own-adventure, not linear. "Do your owners know each other?" is an EARLY question that shapes the rest:
    strangers -> trade review on, stronger activity nudges + signup/verification flow, mini-games pushed harder (keep last place
    setting lineups).
