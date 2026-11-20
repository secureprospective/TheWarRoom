# Desktop v3 gap list (2026-11-20)

Target: `~/Downloads/TheWarRoom-mock-v3-FINAL.html` (Beelink) at 1920x1080. Reference extracts: `.scratch/mock/v3.css` (898 lines), `.scratch/mock/v3.js` (2180 lines).
Current: thin app at 1920x1080 on Claude-OS (screenshot `~/scratch/dv3/app-home.png` on claudeos).

## Key findings
- The mock is a script-built prototype: one 92 KB stylesheet (tokens, palettes, knurl/hazard/hatch/perf textures, keycaps) and a frame + board system.
- The legacy local shell (`TargetApp.tsx`, `homeLayout.ts`) was built to an older mock (`home-mock-v5`, 3x2 tiles). **Nothing visual in legacy is v3.** v3 is a new shell.
- Reusable: the data layer. `homeReads.readThinHomeBrief`, `homeNeedsYou.gatherNeedsYou`, Pulse reads, hosted reads/changes, the `HostedLeague` and `HostedWorkflows` rooms.
- Fonts (Inter, JetBrains Mono) are already bundled.
- Method: v3 CSS ported verbatim; React components emit the mock's DOM and class names; sample data (`W`, `PL`, `docket()`) is replaced by hosted reads. Nothing is approximated by eye.

## Regions (grade: missing / partial / match)
| Region | Mock source (v3.js) | Current thin app | Data | Grade |
|---|---|---|---|---|
| Top bar: mark, league chip, search/cmd, Needs you, Assistant, Feeds, avatar | shell.js `top()` | none (bare "Legacy NFL" title) | admitted leagues, needs-you | missing |
| Alert band (NOW, most urgent item) | `band()` | none | needs-you | missing |
| Rail: app zone (Home, boards, shortcuts) + league zone (cart, nodes, leaves, NOW, shaped-off) | `rail()` | none on Home; old rail inside league | leagues, roles, rules shape | missing |
| Board tab strip (Home, Today, Game day, All leagues live, Commissioner desk, Front office, Board, Add a piece, Edit) | grid.js + v3 shell patch (l.1395) | none | account prefs (boards) | missing |
| Home board grid 24x14, piece frame, move/resize/stack/full | grid.js (l.484-760) | 2-column list | account prefs | missing |
| Piece: Docket (needs you / waiting on / landed) | pieces `docket` | "Needs you" list | needs-you, moves, offers | partial (data) / missing (UI) |
| Piece: This week across my leagues | pieces | league card shows week only | results, schedule, week | missing |
| Piece: My players today | pieces | none | lineup, live scores | missing |
| Piece: NFL news (chips: all, my players, injuries, wire) | pieces + l.1449 | none | news feed? (verify read exists) | missing |
| Piece: Deadlines | pieces `DL` | none | calendar | missing |
| Piece: Shortcuts | pieces | 4 buttons on card | static + roles | missing |
| Inspector | `insp()` | none | selection | missing |
| Status line | `status()` | none | connection, live lag, clock, week | missing |
| Overlays: league switcher, command palette (Ctrl K), account menu, map (?) | shell.js acts | none | leagues, players | missing |
| Screens: HQ, War Room, Pulse, Control Room in v3 frame and palettes | screens.js, trade.js, v3 tool sections | old hosted rooms | existing hosted reads | partial |

## Batches (one PR each, only after the batch is finished and screenshot-checked)
1. **Frame + Home**: v3.css, top bar, band, rail, tab strip, board grid with the default Home board and its six pieces, inspector, status line, switcher, command palette. Real data.
2. **League screens in the frame**: HQ, War Room, Pulse, Control Room reskinned to v3 (palettes, tabs, panels), still using the existing hosted rooms.
3. **Board editing**: edit mode, add a piece, move/resize/stack, the other boards (Today, Game day, All leagues live, Commissioner desk, Front office).
4. **Phone**: mobile v1 side-by-side on the S8 and iPhone.

## Data blockers
- Lab week stuck on 9: hosted `week.read` reads only the pinned football evidence (`internal/leagueservice/football.go:41`); `refresh-nfl-schedule` re-fetches stale weeks but never re-pins them. Fix on branch `session/refresh-pins-final-weeks` (in progress).
- NFL news source: confirm a hosted read exists before building the piece; if not, it is a logic gap to build.
