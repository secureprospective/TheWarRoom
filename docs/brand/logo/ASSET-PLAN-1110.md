# The War Room logo: full asset plan

Written 2026-11-10. The mark is **The Roof** (Christopher: "THE ROOF it is"). This plan covers every asset to be made from it. Christopher's floor: no flat below 8K, nothing less than the best for animation, and the lead animated piece is **The Move-In** in four forms: flat still, flat animated, 3D still, 3D animated.

Claude (Tom, on the Beelink) directs and reviews. Sol 6.1 on Bee, at medium, writes the generator scripts and does the heavy lifting from briefs. Nothing goes public without Christopher's approval for that release.

## 1. The master, which everything is cut from

One geometry, on a grid 16 wide and 14 tall. Every edge is a whole number.

| Part | Position (x, y from top-left) | Size |
|---|---|---|
| Roof | x 0 to 14, y 0 to 4 | 14 x 4 |
| Wall | x 2 to 6, y 4 to 14 | 4 x 10 |
| Block | x 8 to 16, y 6 to 14 | 8 x 8 |

- Eave: the roof passes the wall by 2 on the left. Overhang: the block passes the roof by 2 on the right. Gap between the block and the roof, and the block and the wall: 2.
- Colour: black and white only. The mark owns none of the four signal colours.
- Master vectors: `docs/brand/logo/master/` (`twr-mark-black.svg`, `twr-mark-white.svg`, `twr-favicon.svg`).
- In 3D the same grid gains depth: roof and wall are 4 deep; the block is an 8 x 8 x 8 cube centred on them, so it stands 2 proud at the front and back as well as at the side. Seen dead front-on, the 3D object is exactly the flat mark.

## 2. The Move-In storyboard (shared by the flat and the 3D film)

Six seconds, 60 frames a second, no bounce: steel stops hard.

| Time | Beat | What happens | Still that must read on its own |
|---|---|---|---|
| 0.0 to 0.6 | Dark | The empty room. One thin field line on the floor. | |
| 0.6 to 1.6 | The room | The wall stands. The roof lowers and seats on it with a hard stop. | **The room**: roof and wall, no block |
| 1.6 to 3.2 | The move | The block slides in along the floor from the right, fast, then brakes. | **Arriving**: block half a mark-width out |
| 3.2 to 3.6 | Home | It stops 2 units proud of the roof. Light comes up under the roof. | **Home**: the finished mark |
| 3.6 to 4.6 | The match | 3D only: the camera eases to dead front and the object becomes the flat mark, frame for frame. | |
| 4.6 to 6.0 | The name | THE WAR ROOM sets in mono capitals, one letter at a time, like a stamp. Hold. | Mark beside the name |

Cut-downs from the same timeline: a 2-second sting (the move and the stop only), and a loop called **The Flock** (more blocks, each smaller, queue up behind the first and move in).

Rule taken from Sol's review: the mark is required to read at the three named stills, not at every frame.

## 3. Inventory

### A. Vector masters

| # | Asset | Format |
|---|---|---|
| A1 | Mark, black and white | SVG, PDF |
| A2 | Wordmark THE WAR ROOM, outlined (JetBrains Mono Bold, wide-spaced; the font's licence allows it) | SVG, PDF |
| A3 | Lock-up, mark beside name | SVG, PDF, black and white |
| A4 | Lock-up, mark above name | SVG, PDF, black and white |
| A5 | One-colour stamp version (same mark; proves the 2-unit gap holds when ink spreads) | SVG |

### B. Flat stills, 8K floor

The mark scales by exactly 512: **8192 x 7168 pixels**, every edge on a pixel.

| # | Asset | Size | Backgrounds |
|---|---|---|---|
| B1 | Mark alone | 8192 x 7168 | transparent (black, white), solid white, solid black |
| B2 | Mark, square with clear space | 8192 x 8192 | same four |
| B3 | Lock-up beside name | 8192 wide | same four |
| B4 | Lock-up above name | 8192 wide | same four |
| B5 | Widescreen plate, mark and name on the site's dark steel | 7680 x 4320 | steel |
| B6 | The three Move-In stills (room, arriving, home) | 7680 x 4320 | black, steel |
| B7 | The Flock still | 7680 x 4320 | black, steel |

PNG, 16-bit where there is texture, plus the vector original for each.

### C. Flat animation: The Move-In

| # | Asset | Spec |
|---|---|---|
| C1 | Master | 7680 x 4320, 60 fps, lossless frame sequence. It is drawn from vectors, so 8K costs little. |
| C2 | Master with transparency | Same, with an alpha channel, as ProRes 4444 |
| C3 | Delivery films | 8K and 4K, H.265 and AV1; a web version with transparency (WebM) |
| C4 | Live web version | SVG and CSS, resolution-free, under 5 KB, honours reduced-motion (shows the Home still) |
| C5 | 2-second sting | Same formats as C3 and C4 |
| C6 | The Flock loop | Same formats as C3 and C4, seamless |
| C7 | Square and vertical cuts | 4320 x 4320 and 4320 x 7680 |

### D. 3D still: The Move-In as a steel object

Built in Blender (free), run headless. Roof and wall: dark blasted steel with a welded seam where they meet. Block: lighter machined steel. Small chamfer on every edge to catch light. Set in the site's dark command room.

| # | Asset | Size |
|---|---|---|
| D1 | Dead front (matches the flat mark exactly) | 7680 x 4320 and 8192 x 8192 |
| D2 | Three-quarter hero | 7680 x 4320 and 8192 x 8192 |
| D3 | Low angle, looking up under the roof | 7680 x 4320 |
| D4 | The three Move-In stills in 3D (room, arriving, home) | 7680 x 4320 |
| D5 | Broken apart: roof, wall and block separated, in order | 7680 x 4320 |
| D6 | Each of the above with a transparent background | same sizes |

Path-traced (Blender Cycles), 16-bit PNG plus the raw EXR.

### E. 3D animation: The Move-In

| # | Asset | Spec |
|---|---|---|
| E1 | Master film, 6 seconds | 60 fps, motion blur, lossless frames, then ProRes and H.265/AV1 |
| E2 | 2-second sting | same |
| E3 | The Flock loop | same |
| E4 | Turntable, 360 degrees, 8 seconds | same |
| E5 | Build and break-apart, 5 seconds | same |

**Resolution is decided by one measured number, not a guess.** The Beelink has 16 processor threads, 27 GB of memory and a Radeon 680M, and no paid render farm is allowed. Path-tracing 360 frames at 8K on this machine may take weeks. So: render one finished frame at 4K and one at 8K, time them, and then choose:

1. Path-traced 8K if the whole film fits in about three nights.
2. Otherwise path-traced 4K, 60 fps.
3. Otherwise Blender's fast renderer (Eevee) at 8K, checked side by side against a path-traced frame.

The 8K floor for flats and 3D stills holds regardless.

### F. Platform sizes (cut from the masters; these sizes are fixed by the platforms)

| # | Asset | Size |
|---|---|---|
| F1 | Browser tab icon | SVG, plus ICO at 16, 32, 48 |
| F2 | Phone home screen (Apple) | 180 |
| F3 | Web app icons, plain and maskable | 192, 512 |
| F4 | Desktop app icon (replaces the placeholder `build/appicon.png` and `build/windows/icon.ico`) | 1024, ICO 16 to 256 |
| F5 | Share image for the site | 1200 x 630 |
| F6 | Social profile and banner | 800 x 800, 1500 x 500 |

### G. Rule sheet and proofs

| # | Asset |
|---|---|
| G1 | Rule sheet, one page: clear space, smallest size, one-colour version, the two lock-ups, what never to do |
| G2 | Proof of the seven tests in the brief, each shown rendered |
| G3 | Stamp and embroidery simulation (ink spread and thread pull on the 2-unit gap) |

## 4. Order of work and gates

| Phase | Work | Who | Gate before the next phase |
|---|---|---|---|
| 1 | Vector masters and outlined wordmark (A) | Sol writes the generator; Claude reviews | Edges land on whole pixels at 16 and at 8192; lock-ups rendered and looked at |
| 2 | Flat stills (B) and platform sizes (F) | Sol | Pixel check on B1; every file opened and looked at |
| 3 | Flat animation (C) | Sol | Christopher sees a low-resolution timing pass first; the three stills read |
| 4 | 3D still (D) | Sol writes the Blender script; Claude installs Blender and runs it | Christopher approves a 1080p look test before any 8K render |
| 5 | 3D animation (E) | Sol script; Claude runs | Christopher approves a low-resolution timing pass; the two timed test frames set the resolution |
| 6 | Rule sheet, proofs (G); icons wired into the app and site on the `session/twr-logo` branch | Claude | Christopher approves each public release |

## 5. Where things live

- Masters, generator scripts, this plan: the repo, branch `session/twr-logo`, worktree `~/work/TheWarRoom/.worktrees/logo`, under `docs/brand/logo/`.
- Renders (large files, kept out of git): `~/fleet/runs/twr-logo-1110/out/`.
- Briefs for Sol: `~/fleet/briefs/twr-logo-*.md`. Sol's runs: `~/fleet/runs/sol61-twr-logo-*/`.
- Proof pages from the design session: `~/fleet/runs/twr-logo-1110/` (`proofs-1.html`, `proofs-2.html`, `final/index.html`).

## 6. Known limits

- The look-alike search was visual, not trademark clearance. Storage and moving brands, app-store icons and trademark registers were not fully covered.
- Two other products use nearly the same name (Fantasy War Room, War Room Fantasy Football). The mark should appear with the full name wherever there is room.
