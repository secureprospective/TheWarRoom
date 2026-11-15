# TheWarRoom — North Star
Version: 2.1 — 2026-11-09
Status: Strategic reference. Authoritative on purpose, scope, and what earns build time. CLAUDE.md is the
session entry point; technical specs are authoritative on implementation detail.

Changelog: 2.1 (2026-11-09) folds in the rulings made since 2.0: the client/server plan of 11-05, the
platform direction of 11-08 and the debloat rulings of 11-09. It adds the shape of the product (AI and
CLI first), makes the import the front door, names the solo-owner tools, and corrects the desktop's
standing. 2.0 (2026-10-28) replaced the June 1.2 framing (an MFL companion, "not a generic platform").
Older versions are in git history.

## Mission
TheWarRoom is a sandbox where people play a recreational game together: dynasty fantasy football. Every
rule is a setting, any league can be built, and AI can reach every corner. MFL is where people come from,
not the ceiling.

The market is weak. Build the juggernaut, and assume a big player copies us. We win on design,
engineering, speed, and a game people want to come back to.

The goal is to bring in as many leagues as possible with something so good it would be a mistake not
to use it.

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
- **Owners** who quit leagues from time suck, mind suck, and no cool factor. Every flagship choice must
  cut time or load, or add cool.
- **Commissioners** with full control. Manual overrides stay; when in doubt, the human decides.
- **AI actors** working for a named person through the same commands people use.
- **Solo owners** whose league still lives on MFL. They get TheWarRoom's tools (values, cap and contract
  maths, what-if plans) on their own team. This is one simple framework, not a second product.

## The job test
Before any work gets tokens, it must do at least one of these:
1. Widen the sandbox (more leagues can be built, more settings honoured).
2. Make the game more addictive (return loops that respect owners' time).
3. Make the league more trustworthy (correct scores, honest records, safe data).

None of the three: don't build it.

## Exceptional vs plain on purpose
Be exceptional where it counts:
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
   are what the engine truly cannot run (32 franchises is the cap).
   An MFL import is just settings in the same app: no code branches on where a league came from or on
   one league's ID, and nothing is locked because it came from MFL.
2. **Settings change through one flow.** Every setting changes through the same rule-change path; the
   engine does the work behind it. Lock only when play makes a change unsafe, and say why in plain words.
3. **Rules are league data.** Year-over-year changes need no code release.
4. **Correctness first, speed second.** A speed fix must prove the same decisions as a full replay.
5. **Lightning fast is law.** Measured on the server/web path:
   - a screen opens in under 200 ms;
   - a click responds in under 100 ms;
   - no spinners in normal use;
   - a timed 32-team load runs in CI on every change.
6. **The player ID is the player.** One record per real player (draft capital, production, reputation,
   age). Valuation is a function of that record; native values come first.
7. **A league is its rosters and contracts.** Recovery restores those first; history is secondary.
8. **MFL is import, never the boss.** The import is the front door, so it must be flawless for any
   league size or type: rosters, contracts, picks, rules and history land as-is, and a league that
   cannot land says exactly why. The native ledger is the record; MFL stays a live stat fallback and a
   reference only, and TheWarRoom never writes to MFL.
9. **CLI and AI come first.** Every action is a named command in one registry that the CLI, AI and the
   screens all use. A feature is not done until an agent can drive it from the CLI. AI acts for a named
   person, a person confirms what changes the league, and the ledger records it.
10. **Security is proportionate.** It's a recreational game; never the weak point, never heavier than
    it needs to be.

## Platform
- We host every league for at least the first year. Self-hosting comes second.
- The web app (PWA) is the main way people use TheWarRoom: the same React screens and Go engine behind
  our server, in a browser or on a phone. Where a job exists on both paths, the hosted one is the one
  we protect.
- The desktop app is first-class and supported. It runs the same screens against the same server, and
  it is how a solo owner runs TheWarRoom on their own machine. No league feature is desktop-only.
- The CLI (`twr`) and its MCP mode are the third door, and the one AI uses.
- Home hosting: a dedicated Linux laptop on the UPS, Cloudflare Tunnel (no open ports), and encrypted
  backups to Cloudflare R2. On a power outage: an in-app countdown, a final backup, and a clean shutdown.

## Growth
Fantasy grew on people talking to each other. Growth comes from the game's own community, not marketing:
1. an output engine with league branding;
2. moments: rare, brag-worthy, league-specific share cards, each carrying a "start your own league" link;
3. a Tuesday wrap-up and Thursday look-ahead, plus spreadsheet export;
4. clean printable rosters and cheat sheets;
5. first-class chat.

Never use real-player photos or NFL logos, and never ship AI slop.

## What this is not
- Not a betting or real-money platform.
- Not an MFL skin, and not an MFL mirror. The solo-owner tools read MFL; they never become a second
  kind of league.
- Not a research lab. Research and archive tooling live outside the app.
- Not one app per league type. League shape is settings.
- Not a feature pile. Breadth lives in settings; screens stay lean.
