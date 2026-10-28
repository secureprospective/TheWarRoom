# TheWarRoom — North Star
Version: 2.0 — 2026-10-28
Status: Strategic reference. Authoritative on purpose, scope, and what earns build time. CLAUDE.md is the
session entry point; technical specs are authoritative on implementation detail.

Changelog: 2.0 replaces the June 1.2 framing (an MFL companion, "not a generic platform"). Source: the
2026-10-28 direction talk. 1.2 is in git history.

## Mission
TheWarRoom is a sandbox where people play a recreational game together: dynasty fantasy football. Every
rule is a setting, any league can be built, and AI can reach every corner. MFL is where people come from,
not the ceiling.

The market is weak. Build the juggernaut, and assume a big player copies us. We win on design,
engineering, speed, and a game people want to come back to.

## Who this serves
- **Owners** who quit leagues from time suck, mind suck, and no cool factor. Every flagship choice must
  cut time or load, or add cool.
- **Commissioners** with full control. Manual overrides stay; when in doubt, the human decides.
- **AI actors** working for a named person through the same commands people use.

## The job test
Before any work gets tokens, it must do at least one of these:
1. Widen the sandbox (more leagues can be built, more settings honoured).
2. Make the game more addictive (return loops that respect owners' time).
3. Make the league more trustworthy (correct scores, honest records, safe data).

None of the three: don't build it.

## Exceptional vs plain on purpose
Be exceptional where it counts:
- the any-setting engine;
- built-in intelligence: trade fairness, dynasty economics, draft help;
- AI access to every action;
- return loops and shareable moments.

Be plain on purpose where familiar UX helps: lineups, the trade screen, navigation, settings forms.
Owners should never relearn what already works.

## Principles
1. **The app houses the settings.** A league format is a set of setting values. Presets suggest values;
   they never limit them. No engine, seed, legality, or UI code branches on a preset name. The only limits
   are what the engine truly cannot run (32 franchises is the cap).
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
8. **MFL is import, never the boss.** Migrating a league from MFL must be a no-brainer, and imports land
   as-is. MFL stays a live stat fallback and a reference only.
9. **CLI and chat are first-class.** Every action is a named command in one registry that the UI, the CLI,
   and AI all use. AI acts for a named person and the ledger records it.
10. **Security is proportionate.** It's a recreational game; never the weak point, never heavier than
    it needs to be.

## Platform
- We host every league for at least the first year. Self-hosting comes second.
- The web app (PWA) is the main delivery: the same React screens and Go engine behind our server. The
  desktop app is a convenience shell; it keeps building in CI but never drives decisions. No desktop-only
  features. Phone-specific layout comes later.
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
- Not an MFL skin. Research and archive tooling live outside the app.
- Not a feature pile. Breadth lives in settings; screens stay lean.
