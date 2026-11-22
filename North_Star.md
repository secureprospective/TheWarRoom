# TheWarRoom — North Star
Version: 3.0 — 2026-11-22
Status: Strategic reference. Authoritative on purpose, direction and what earns build time. README is the
front door; technical specs and `docs/adr/` are authoritative on implementation detail.
Owner: Christopher rules on this file. Agents propose changes; they never edit it without his ruling.

Changelog: 3.0 (2026-11-22) folds in the 11-20 and 11-22 direction sessions. It adds where we are in time
and the near-term target, splits leagues into two roles, turns the job test into a guideline with
"protects the league" added, makes the commissioner experience the off-season focus, and marks the
ideas under review in the reset. 2.1 (2026-11-09) and older are in git history.

## Mission
TheWarRoom is a sandbox where people play a recreational game together: dynasty fantasy football. Every
rule is a setting, any league can be built, and AI can reach every corner.

The market is weak and has stopped trying. Build the juggernaut, and assume a big player copies us. We win
on design, engineering, speed, and a game people want to come back to. In the long run: bring in as many
leagues as possible with something so good it would be a mistake not to use it.

**The near-term target.** Through 2027, success means one real 32-team league runs a full season on
TheWarRoom, and the leagues that trust us (Legacy first among them) are never put at risk.

## Where we are
- **2026 season:** live alpha. We learn mid-season, with real games on real phones.
- **This off-season:** prove the off-season jobs (draft, contracts, rollover) and the commissioner experience.
- **2027:** a real 32-team league starts with a rookie draft (target July 2027) and proves the in-season jobs.
  Only after that do we call it a first real release.
- **Testers:** Christopher is the main tester until about March. Interest is expected to grow when the
  draft engine is tested with real people.

**Prepared, not predicted.** Real life can't be planned. Which league moves to TheWarRoom first, and when,
is not decided. Our job is to be ready for whatever happens: options built, tested and waiting.

## Two kinds of league
- **MFL leagues.** The league lives on MFL and MFL is its record. TheWarRoom is a companion: it reads the
  league, adds the tools and the experience, and never writes to MFL. Code that serves this role is
  current, not leftover.
- **Native leagues.** TheWarRoom is the record. MFL is at most an import door and a live stat fallback.

A league moves from the first kind to the second only through the import (cutover), and only when its
people choose to.

## The shape of the product
TheWarRoom is AI first and CLI first, with screens laid over the top so anyone can use it.

- **TheWarRoom houses the league.** It holds the league, its rules and everything people do in it. The
  rules are settings, so the app frames whatever way people imagine playing fantasy football.
- **Everything is a command.** An AI agent, working for a named person, can do anything in TheWarRoom
  through the CLI. A person can do the same from a screen. Both go through one command list and one
  set of checks.
- **Screens are outputs.** So are minigames, share cards, shows and reports. They read the league and
  send commands; none of them owns league logic.
- **One build for every league.** A league type or rule set is never its own code path, screen set or
  storage. If supporting a new kind of league needs new infrastructure, the design is wrong.
- **Small on purpose.** The code is kept to what is good and working. Less code is a feature: it is
  what keeps the app fast and lets an AI change it safely.

## Who this serves
- **Owners** who quit leagues from time suck, mind suck, and no cool factor.
- **Commissioners** with full control. Manual overrides stay; when in doubt, the human decides.
- **AI actors** working for a named person through the same commands people use. A **Stand-in** (an AI
  holding an empty seat and playing it like an owner) is used in test leagues only, for now.
- **Solo owners** whose league still lives on MFL. They get TheWarRoom's tools on their own team. This is
  one simple framework, not a second product.

## What earns build time (a guideline, not a gate)
Good work does at least one of these:
1. **Saves time:** takes work off an owner's or commissioner's hands.
2. **Makes someone ask "what is that?":** the cool factor a bystander notices and tells a friend about.
3. **Protects the league:** correct scores, honest records, safe data, no surprises.

We don't know everything, and the market is weak because it stopped being inventive. Ideas that don't fit
neatly are welcome; pitch them as "would this work?" with a recommendation.

## Where we focus now
- **In season (now):** live scoring on the phone, built to mobile mock v1. It is the hook that gets owners
  talking. Anything a bystander sees there gets full polish.
- **Off-season:** the commissioner experience. The engines already save time on every commissioner job;
  how each job looks and feels is the hard part, and it is where other apps gave up. Each job is studied
  against Sleeper, ESPN, Yahoo and MFL first, then redesigned mock first until it is faster and clearer
  than anything on the market. Keep an open mind on what that turns out to be.
- **Until about March:** build for Christopher's solo use first: fast testing tools, importing many league
  shapes, clear error codes.
- **Test leagues:** copies of real league shapes (10, 12, 16, 32 teams, twin-pool), played by Stand-ins.
  They prove the app and carry the 32-team load.

## Exceptional vs plain on purpose
Be exceptional where it counts:
- live scoring on the phone;
- the commissioner experience;
- the any-setting engine;
- the import: any MFL league, any size, any type, lands whole;
- the valuation engines: two numbers per player, fitted on the league's own scoring;
- built-in intelligence: trade fairness, dynasty economics, draft help;
- AI access to every action;
- return loops and shareable moments.

Be plain on purpose where familiar UX helps: lineups, the trade screen, navigation, settings forms.
Owners should never relearn what already works.

## Principles
1. **The app houses the settings.** A league format is a set of setting values. Presets suggest values;
   they never limit them. No engine, seed, legality, or UI code branches on a preset name. The only limits
   are what the engine truly cannot run (32 franchises is the cap). No code branches on where a league
   came from or on one league's ID.
2. **Settings change through one flow.** Every setting changes through the same rule-change path. Lock
   only when play makes a change unsafe, and say why in plain words.
3. **Rules are league data.** Year-over-year changes need no code release.
4. **Correctness first, speed second.** A speed fix must prove the same decisions as a full replay.
5. **Lightning fast is law.** Measured on the server/web path:
   - a screen opens in under 200 ms;
   - a click responds in under 100 ms;
   - no spinners in normal use;
   - a timed 32-team load runs in CI on every change.
6. **The player ID is the player.** One record per real player. Valuation is a function of that record.
7. **A league is its rosters and contracts.** Recovery restores those first; history is secondary.
8. **The import is the front door.** It must be flawless for any league size or type: rosters, contracts,
   picks, rules and history land as-is, and a league that cannot land says exactly why. TheWarRoom never
   writes to MFL.
9. **CLI and AI come first.** Every action is a named command in one registry that the CLI, AI and the
   screens all use. AI acts for a named person, a person confirms what changes the league, and the ledger
   records it.
10. **Fixes reach people on their own.** Installed apps update themselves; nobody reinstalls. A problem an
    owner hits shows a short code they can copy or send to the commissioner.
11. **Security is proportionate.** It's a recreational game; never the weak point, never heavier than it
    needs to be.

## Platform
- We host every league for at least the first year. Self-hosting comes second.
- The web app (PWA) is the main way people use TheWarRoom: on the phone (mobile mock v1) and on the
  desktop browser (desktop mock v3), with the same Go engine behind our server.
- The CLI (`twr`) and its MCP mode are the door AI uses.
- **Under review in the reset:** the standalone desktop app and the old local mode. Until Christopher rules
  on them, nothing new is built on them.
- Home hosting: a dedicated Linux machine on the UPS, Cloudflare Tunnel (no open ports), and encrypted
  backups to Cloudflare R2.

## Growth
Fantasy grew on people talking to each other. Growth comes from the game's own community, not marketing:
1. live scoring good enough that owners show it to each other, and a live wall for streams and clips;
2. moments: rare, brag-worthy, league-specific share cards, each carrying a "start your own league" link;
3. a Tuesday wrap-up and Thursday look-ahead, plus spreadsheet export;
4. clean printable rosters and cheat sheets;
5. first-class chat.

Never use NFL logos, and never ship AI slop.

## What this is not
- Not a betting or real-money platform. No betting odds on screen.
- Not an MFL skin. Companion tools for MFL leagues read MFL; they never become a second kind of league.
- Not a research lab. Research and archive tooling live outside the app.
- Not one app per league type. League shape is settings.
- Not a feature pile. Breadth lives in settings; screens stay lean.

## How we work
Christopher is the strategic partner: the real world, what people need, priority, money and anything
outward. The AI is the tactical partner and logic engine: code, structure, models and tests. The AI
pitches ideas he wouldn't know to ask for, says what a ruling will cost before it locks, and flags
assumption risk ("This assumes X. If X is wrong, it costs Y. Cheapest check: Z."). Detail lives in
`~/fleet/AGENTS.md` ("How we build together"). Live progress lives on the tracker.
