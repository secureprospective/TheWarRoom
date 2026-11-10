# RESUME: TheWarRoom landing page (warroom.secureprospective.com), 2026-11-10

Written by Tom (Claude Code on the Beelink) before a context compaction. The session continues; nothing is merged.
On "we are back": read this, then start at NEXT ACTIONS item 1. Do not recap or re-plan.

## 1. What we are doing
Building the public landing page for TheWarRoom: a Super Bowl-ad-grade scroll story whose message is "you are playing
fantasy football the dumbest way possible; open your own league in TheWarRoom". Landing page only; sign-up wiring, the
live join page and serving are a later job for a cheaper model.

- Repo: `~/work/TheWarRoom` on the Beelink. Worktree `~/work/TheWarRoom/.worktrees/website`, branch `session/website` (from `origin/main` 8899a8bff).
- Never touch the main checkout (`session/league-history-pwa`, the Legacy NFL Lab) or branch `session/ui-to-mocks`.
- Starting handoff: `~/Downloads/TheWarRoom-WEBSITE-HANDOFF.md` (hard rules in its section 8 still bind).
- Christopher views the page at `file:///home/chris/Downloads/twr-landing/index.html` (a plain copy of `website/`; re-copy after every change).

## 2. Agents and harnesses
- Rendering and checking: Claude-OS (`ssh claudeos`), headless Chromium 153 through `~/work/site-study/twr-shots.mjs`
  (`CHROME=/usr/bin/chromium node twr-shots.mjs $HOME/work/twr-site <desktop|mobile> [stops e.g. 0,5.5,11]`; a stop is a scene number,
  halves are mid-flight). Copy `website/*` to `claudeos:work/twr-site/` first; shots land in `work/twr-site/out/`. A local copy of the
  script is `/tmp/twr-site/shots.mjs` (recreate from this description if /tmp was cleared). Contact sheets: ImageMagick `montage`.
- Run `~/fleet/bin/heavy-ok.sh` before heavy jobs. Never open a GUI on Christopher's desktop.
- Design hook: the `impeccable` skill scans every write to `website/`; ignores live in `.impeccable/config.json` (committed).
- Christopher's marked-up screenshots arrive in `~/content/feedback/YYYY-MM-DD_HH-MM*.png`.

## 3. Status
| Item | State |
|---|---|
| 12-scene landing page, desktop 1920x1080 | Built; every scene rendered and looked at as still frames; no console errors |
| Phone 390x844 | Rendered and looked at after the first two passes; NOT re-checked after the outline-letter and name changes |
| Motion (camera travel, slams, lights-on sweep, confetti) | NOT watched in motion by me; Christopher has scrolled it and sent marks |
| Switches, share-card builder, copy button | NOT clicked in a browser by me |
| "Open your league" buttons | Not wired (by design, later job) |
| Deploy | None. Needs Christopher's approval per deploy |

## 4. Artifacts
- `website/index.html` 21,044 B sha256 3d2307d3de8011c4…
- `website/site.css` 16,851 B sha256 5f4646c4f7e7bffb…
- `website/site.js` 18,461 B sha256 c41db330cf6cda99…
- `website/fonts/` Inter 400/600/700 and JetBrains Mono 400/700 woff2 plus OFL.txt, copied from `frontend/src/assets/fonts/`.
- Market research used for messaging: `~/Downloads/warroom-ux-landscape-report-2026-10-29.md` (Parts 1 and 2; it flags some of its own claims as unverified).

How the page works: scroll drives one camera 110 yards down a field. `site.js` holds one projection shared by the DOM scenes
(CSS 3D, `.lens` perspective 1000) and a 2D canvas (floor, walls with screens, portals, painted numbers, motes, confetti, goalposts).
Scenes are 2600 units (10 yards) apart; each `.ly` layer carries `--x/--y/--z` (and `--px/--py` for portrait). The camera holds at
each scene and eases between; `HY` is camera height per scene. Lights switch on from midfield (`litZ`). Outline type uses a solid dark
fill with `paint-order: stroke fill` and a doubled stroke; a transparent fill shows the font's inner seams (Christopher's red marks) — do not go back.

Scenes: 0 hero · 1 Exhibit A free app · 2 Exhibit B deep site · 3 Exhibit C duct tape · 4 why people quit · 5 midfield "Enough / One room" ·
6 game day · 7 trade desk · 8 any league · 9 red zone promise · 10 touchdown CTA · 11 extra point share card.

## 5. Current bug
None open.

## 6. Already tried, do not repeat
- Thin line-only field with no walls: read as flat and empty. Replaced by the solid environment.
- Next scene's layers visible during a hold (far fade out to 4200): ghost words behind headlines. Far fade now ends at 1500.
- Portal labels on distant portals: collided with centred headlines. Labels draw only when nearer than 2000.
- DOM yard signs at the sidelines: removed; yard numbers are painted on the floor and on portal beams.
- Hazard underline under "dumbest": struck through the next line. It is now a solid bright block with black text.
- Small tracked labels above headings (kickers): removed for the design hook; the scorebug names each chapter.
- `ignore-value wide-tracking "0.16em"`: the hook has no value handle for that rule; it is scoped to `website/index.html` with `*`.

## 7. Christopher's decisions
- The site is the only place people meet TheWarRoom for now: first class, built to move to an official domain later. Code stays in the TheWarRoom repo.
- Audience is every fantasy player. CTA "Open your league"; second door "Bring your league over".
- "We will never do ads in the app space, no unwanted betting odds. This is fantasy football, the best thing you ever played with your family, friends and work mates."
- Scrolling segments that get cooler as they go; depth, layers, 3D environment; Super Bowl ad grade.
- Punchy wording stays ("I like punchy wording, stick with it"). Hook ignore recorded.
- Real Hall of Fame names, THEN amended (see item 8 below): roll the names slightly.

Rulings on the eight things I said I held back (his numbering follows my list):
1. **Sound:** a subtle crowd noise only; when a button is clicked, cheers ring out and die off.
2. **Rendered objects:** only rendered steel and the goal post. Have an **Opus 5.5, low effort subagent** build them.
3. **The roast:** do all three: smash through the old tools, crack the spreadsheet, collapse the 1998 page as the camera hits it.
4. **Name the competitors.** "They suck and everyone knows it."
5. **Camera:** keep it at field level and subtle, like a running back slashing through the defence. No rafters drop, no dive.
6. **Real app screens:** do better if I can, but do not overwhelm the current state; he likes that it stayed focused.
7. **Share card:** give both options (text and image).
8. **Names:** roll them ever so slightly, like "Berry Rice" or "Brett Favor" (blended or bent Hall of Fame names, not exact real names).

Also ruled: after compaction, **use impeccable on the Beelink with sol-6.1 on high settings for a cross-pollination second opinion** on the page.

## 8. Ledger state
- Committed on `session/website`: `website/`, `.impeccable/config.json`, this file. Not pushed.
- Memory files added this session (outside the repo): `warroom-site-copy-stays-punchy`, `marked-up-screenshots-land-in-content-feedback`; `~/fleet/tools/build-memory-indexes.py` map updated.

## 9. Next actions, in order
1. Get the second opinion: run the impeccable skill's critique of `website/` through Sol 6.1 at high thinking on the Beelink (find the launcher under `~/fleet/bin/`, e.g. `launch-sol61-*.sh`, and the `zen-panel` skill for the one-shot route). Treat its answer as leads; verify before acting.
2. Names (ruling 8): replace Barry Sanders, Reggie White, Jerry Rice, Joe Montana and the play-by-play names in `index.html` and the `plays` array in `site.js` with slightly rolled names.
3. Name the competitors (ruling 4) in Exhibits A to C and the share-card options. Check each claim against the research before it goes on the page; the research flags several as unverified.
4. Roast (ruling 3): camera smashes through the junk windows; the spreadsheet cracks; the 1998 page collapses on contact.
5. Camera (ruling 5): add a subtle field-level lateral weave between scenes; keep height changes small.
6. Sound (ruling 1): a "sound on" control (browsers block autoplay), a quiet crowd loop, and a cheer that swells and dies off on button clicks. Free or generated audio only; no paid services.
7. Rendered steel and goal post (ruling 2): dispatch an Opus 5.5 low-effort subagent to produce them; place them without slowing first paint.
8. Real screens (ruling 6): raise the density of the game day and trade desk panels toward the locked mocks, without crowding the scene.
9. Share card (ruling 7): keep the text copy and add an image version.
10. Re-render every scene at desktop and phone size on Claude-OS, look at each, refresh `~/Downloads/twr-landing/`, then report to Christopher.

## 10. Environment notes
- Address him as Christopher; short plain replies, outcome first, Central time.
- Toolchain for git hooks: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10`.
- Never real league data, NFL logos or real-player photos on the site. Made-up league names come from the mocks (Gridiron Syndicate, Burnt Orange Militia and the rest).
- Claims on the page that he has not yet confirmed are true today: live play-by-play, cap and roster after a trade, contracts and salary cap, up to 32 teams, bringing a whole league over with history.

## 11. Honest status
A complete first cut exists and Christopher has scrolled it and responded with direction, not rejection. Nothing in items 1 to 9 is started.
Unproven: motion quality as I have only seen stills; phone layout after the latest changes; every interactive control. No ETA given.
