# RESUME: TheWarRoom landing page (warroom.secureprospective.com), 2026-11-10 (second compaction)

Written by Tom (Claude Code on the Beelink). The session continues; nothing is merged, pushed or deployed.
On "we are back": read this, then start at NEXT ACTIONS item 1. Do not recap or re-plan.

## 1. What we are doing
Public landing page for TheWarRoom: a Super Bowl-ad-grade scroll story. Message: "you are playing fantasy football the
dumbest way possible; open your own league in TheWarRoom". Landing page only; sign-up wiring and serving are a later job.

- Repo `~/work/TheWarRoom`, worktree `~/work/TheWarRoom/.worktrees/website`, branch `session/website` (from origin/main 8899a8bff).
- Never touch the main checkout (`session/league-history-pwa`) or branch `session/ui-to-mocks`.
- Starting handoff: `~/Downloads/TheWarRoom-WEBSITE-HANDOFF.md` (its section 8 hard rules bind).
- Christopher views `file:///home/chris/Downloads/twr-landing/index.html` (plain copy of `website/`; re-copy after every change, including `steel.js`).

## 2. Agents and harnesses
- Rendering: Claude-OS (`ssh claudeos`), headless Chromium via Playwright. Copy `website/*` to `claudeos:work/twr-site/`, then in `~/work/site-study`:
  - `CHROME=/usr/bin/chromium node twr-shots.mjs $HOME/work/twr-site <desktop|mobile> [stops e.g. 0,6,8]` (stills, one per scene; uses `window.twrTop(scene)`).
  - `[CLICKS=1] CHROME=/usr/bin/chromium node motion.mjs $HOME/work/twr-site <desktop|mobile> 0-1,2-3` (12 frames per flight; CLICKS=1 also clicks sound, team switch, card, image save, text copy).
  - `node steelshots.mjs $HOME/work/twr-site` (needs `steel-test.html`, kept at `/tmp/twr-site/steel-test.html`, not in the repo).
  - Local copies of the scripts: `/tmp/twr-site/{shots,motion,steelshots}.mjs`. Shots land in `work/twr-site/out/`; pull to `/tmp/twr-site/out2/`, contact sheets with ImageMagick `montage`.
- Second opinion: `~/fleet/bin/launch-sol61-super.sh <brief.md>` (Sol 6.1, high). Brief `~/fleet/briefs/twr-landing-second-opinion.md`; answer `~/fleet/runs/sol61-twr-landing-second-opinion/stdout.log`.
- 3D models: built by an Opus 5.5 low-effort subagent to my spec, corrected by me. Christopher: "have it build to your spec, you just make the corrections needed as it has a tendency to half ass the edges and corners".
- Design hook: `impeccable` scans writes to `website/`; ignores in `.impeccable/config.json`.
- Christopher's marked-up screenshots: `~/content/feedback/YYYY-MM-DD_HH-MM*.png`.

## 3. Status
| Item | State |
|---|---|
| 12 scenes, desktop 1920x1080 | Rendered and looked at after all changes; no console errors |
| Phone 390x844 | Rendered and looked at; three collisions fixed. Christopher still sees spacing and readability problems in his own preview (see next actions) |
| Roast (shatter, spreadsheet crack, 1998 page collapse), camera weave | Seen in filmed flight frames 0-1 to 3-4, 4-5, 9-10 |
| Sound switch, team switch, card builder, image save, text copy | Clicked in a real browser, all work. The sound itself has NOT been heard by anyone |
| Rendered steel portal + goal post (`steel.js`) | In the page, looked at from several angles |
| Sol's verified bugs | Fixed (list in section 6b) |
| "Open your league" buttons | Not wired, by design (later job). Sol rates it the top problem |
| Deploy | None. Needs Christopher's approval per deploy |

## 4. Artifacts
`website/`: `index.html`, `site.css`, `site.js`, `steel.js` (new), `fonts/`. Sizes and hashes: see the commit that carries this file.
How it works: scroll drives one camera 110 yards down a field. `site.js` holds one projection shared by DOM scenes (CSS 3D, `.lens`
perspective 1000) and a 2D canvas (floor, walls, steel, motes, confetti). Scenes are 2600 units apart; `.ly` layers carry `--x/--y/--z`
(`--px/--py` for portrait). Per-flight scroll length is `SPAN[]` (viewport heights), scene tops `TOP[]`, `progress()` maps scroll to scene.
`steel.js` exposes `TWRSteel.portal/goalpost/draw` (flat-shaded polygon meshes, painter sort, near-plane clip, fog by colour blend).
A second canvas `#fx` (above the DOM) carries the shards. Scenes not in front of the camera are `inert`. The field canvas redraws only when the camera moves.
Sound is generated with Web Audio (pink noise through band-passes): `setSound`, `cheer`, `crack`, `thud`. No audio files.
Share card: `card()` text and `cardImage()` 1080x1350 PNG.

## 5. Current bug
None open.

## 6. Already tried, do not repeat
- Thin line-only field; far fade to 4200 (ghost words); labels on distant portals; DOM yard signs; hazard underline under "dumbest"; kickers above headings; transparent-fill outline type (inner seams). All replaced; do not go back.
- Drawing near portals on the front canvas to occlude the DOM: worked out on paper, not built. At the distance where it would be correct the steel is already off-screen at the edges, so it buys nothing.
- Sleeper under "ads": false. Sleeper is ad-free; the page says it "skips the ads and still can't run a contract or a salary cap".

### 6b. Sol 6.1 review (verdict: "does not meet Super Bowl ad grade")
Fixed: trade cap sums (Reggie Wight now $10.2M), one clock for feed and field panel, share label says Sent vs Copied, no confetti under reduced motion,
inert off-camera scenes, resize keeps story position, redraw only on movement, shorter early flights, larger phone type.
Open, raised with Christopher: dead-end CTA buttons; panels float in front of the hall instead of sitting in it; second half does not escalate
(suggest trade desk toggling two offers, switches changing the preview); weak copy "No cool factor / Time suck" (kept; his voice).
Sol's ideas not built: one play travelling through the room; settings reconfiguring the hall; an "inspect this desk" close-up.

## 7. Christopher's decisions
- Site is first class, the only place people meet TheWarRoom for now; code stays in this repo. CTA "Open your league"; second door "Bring your league over".
- Never ads, never unwanted betting odds. Punchy wording stays. Depth, layers, 3D, gets cooler as you scroll.
- Eight rulings, all now built: quiet crowd + cheer on button press; rendered steel and goal post only; all three roast effects; name the competitors;
  field-level subtle camera cut; better screens without crowding; share card as text and image; names rolled slightly ("Berry Rice", "Brett Favor").
- NEW 2026-11-10 (this compaction):
  - **Hero:** he agrees with Sol. Put real animated action in the hero, "not just 1 static look": players being dragged and dropped into lineups,
    live Sunday scoring, trade desk activity. "Clean animations, not sloppy halfassed placeholders."
  - **Competitor claims confirmed TRUE by him:** sportsbook odds next to the lineup on ESPN and Yahoo; Sleeper has no display ads (earns from paid
    pick'em games and avatar sales) and no native contracts or salary cap (commissioners track them in Google Sheets); MFL is paid (about $70 to $90+
    per league per year) with no official app (third-party clients such as MFL Mobile, MFL Platinum). These claims are settled; do not re-ask.
  - **Phone:** he saw spacing and readability problems in his browser preview (he is not sure the preview is accurate). Removing things on phone only is fine if it reads better.

## 8. Ledger state
- Commit 93e6de9fa: first cut. The commit carrying this file: everything in sections 3 and 4. Not pushed.
- Memory `warroom-site-copy-stays-punchy` updated with the rolled-names ruling.

## 9. Next actions, in order
1. Hero action (new ruling): replace the static hero dressing with a looping, authored product sequence in the War Room look, on a timeline, three beats:
   a player block dragged from the bench into a lineup slot (slot lights, projected points tick up); a live play landing and the matchup score changing;
   a trade block sliding across the desk with cap room updating. Keep the headline and "Snap the ball". Must pause when off-screen and stand still under
   reduced motion. Decide what happens to the four junk windows in the hero (they currently shatter on departure; the exhibits keep that effect either way).
2. Phone pass: render every scene at 390x844 and at a second size (e.g. 412x915 and 360x740), fix spacing and readability, and remove secondary panels
   on phone where that reads better (candidates: the "around the league" strip, the franchise wall beyond 12 names, the second paragraph in scene 8,
   junk windows that crowd the hero).
3. Optional tightening now the claims are confirmed: Exhibit B could name the price ("$70 to $90 a league, every year"); Exhibit A could say Sleeper commissioners keep contracts in a Google Sheet.
4. Re-render desktop and phone, look at each, refresh `~/Downloads/twr-landing/`, report.
5. Still for Christopher: has he heard the sound; are these page claims true today: live play-by-play, cap and roster after a trade, up to 32 teams, bringing a whole league over with history; does he want the open Sol items done.

## 10. Environment notes
- Address him as Christopher; short plain replies, outcome first, Central time.
- Git hooks toolchain: `export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/.local/bin:$PATH GOTOOLCHAIN=go1.26.10`.
- Never real league data, NFL logos or real-player photos. Made-up league names come from the mocks.
- ssh aliases: `claudeos`, `claudebox` (CT105, root), `hermes`.

## 11. Honest status
All eight rulings are built and seen rendered. Unproven: how the sound actually sounds; frame rate on a real phone (steel adds roughly 150 to 240
polygons per visible frame, drawn only while the camera moves); phone layout on sizes other than 390x844. The hero animation and the phone pass are not started.
