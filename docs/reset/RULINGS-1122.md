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

## Part 6 (quality and silent breakage) — Christopher 11-22 evening
Why: the code base is too big for one person and his AI helpers to hand-check. Go production grew from
142k lines (11-09) to 168k (11-22). Breakage is found only by chance. TWR must stay AI-maintained AND
human-maintained: minimal, fast, robust.
28. **Randomized canary checks** (T708). About 20 league journeys (sign in, lineup, live score moves, trade, waivers, final,
    recap) run on the Lab by a robot through the twr CLI/MCP. Each run picks a random subset at random times, with the seed
    logged so a red run can be replayed. The aim is human-like coverage at low token cost. Results show green/red on the tracker.
    The T688 report card is the seed of this. **Gate:** no new infrastructure (tiny models, AI harness) lands before the canaries exist.
29. **Line budget** (T709). Extend the root-file ratchet to the whole repo. A feature PR is net ~0 lines or carries a written
    reason. Baselines only go down.
30. **Desktop app out of the main build** (T710). It was a loss leader and learning, not the answer. Before removal, list
    anything worth keeping; Claudebox reviews that list.
31. **Season-in-a-box Go test on every PR** (T711):
    - real twr-server binary, temp dirs, fake clock;
    - one recorded real NFL week replayed from the archive;
    - driven through the twr CLI (testscript style): draft, lineup lock, live scoring, final, stat correction, standings, recap.
    It extends release_matrix_test and the league-refresh end-to-end test.
32. **Inspection after the season**, one area a week. The season-in-a-box test and the canaries prove nothing broke.
    Piece-tests they make redundant are removed carefully.
33. **AI-harness research re-run on 2027-01-02** (T707). The tiny-model and harness plan (infra-map §7) is directional bias,
    not a build.
