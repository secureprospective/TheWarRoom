League Pulse tells a GM what is happening in the league, where everyone stands, and what happened before.

## Tabs

- **Matchups:** understand the selected fantasy week.
- **Live:** follow the NFL action affecting fantasy scores.
- **Race:** understand the league's competitive position and playoff path.
- **Archive:** explore checked league history by season.

Four tabs separate watching games from understanding the week without making schedule or statistics into rail nodes. Archive belongs: league history explains the present, even if it is not live.

## Placement

| ID | Home | Why |
|---|---|---|
| X-03, M-005 | Matchups: central matchup | Classic/Default is the primary read; select any matchup. |
| M-072 | Matchups: matchup drawer | Player-level fantasy box behind the score. |
| M-003, M-004 | Matchups: left list | One week selector and one list for upcoming, live and final games. |
| M-007 | Matchups: weekly summary drawer | Week-wide context, not another primary board. |
| M-018, M-021, M-022 | Matchups: preview/recap drawer | Selected-game preview or recap; position leaders in the same drawer. |
| M-001 | Live: fantasy score panel | League scoring stays visible alongside NFL play. |
| X-01 | Live: game board | Field, last play and followed-game bar; game box in drawer. |
| X-02 | Live: play spine | Fantasy-relevant plays next to the selected game. |
| M-015 | Live: game selector | NFL schedule provides the entry to a game. |
| M-016 | Live: game box drawer | NFL box is detail for the selected game. |
| M-088 | Matchups: corrections drawer | Explain score changes and result flips after the week. |
| M-002 | Race: standings board | Official position is the anchor. |
| C-02, M-008, M-071 | Race: power drawer | One ranking surface with season/franchise and chart options. |
| M-090, M-170 | Race: playoff path panel | Seeds and bubble are one question, with distinct details. |
| M-074 | Race: tiebreaker drawer | Explain the order without crowding standings. |
| M-161 | Race: all-play drawer | Alternative competitive measure, not official standings. |
| M-014 | Race: bracket drawer | Playoff structure on demand. |
| M-011, M-012, M-013 | Matchups: schedule drawer | League, franchise and compact views answer one scheduling question. |
| M-009, M-010 | Matchups: lineup drawer | Week and franchise selection reveal starters and details. |
| M-076, M-077 | Race: starter points drawer | Compare production by position or player. |
| Archive, no ID supplied | Archive: season board | Browse checked seasons from 2013; details in drawer. |

## Moves out

**None.** NFL schedule and box scores are supporting evidence for live fantasy play, not a general NFL news destination. League-wide lineups and schedules belong with matchup comparison; editing my lineup belongs in **Franchise HQ**, reached by a contextual link rather than duplicated here. A new rail node would split related decisions and add navigation without a distinct job. This is a proposed reorganization, not a claim that the current registry already uses these placements.

## One-screen sketches

All tabs reuse the War Room shell: left rail, workspace tabs, bounded work area, right inspector and explicit drawers. At 1920×1080, no page scroll; long lists scroll inside their own labelled panels. Player, pick and franchise objects retain drag and keyboard-accessible alternatives; reading a table never commits an action.

- **Matchups:** left—week and all games; center—selected matchup, scores and starters; right—selected entity inspector; drawers—fantasy box, schedule, lineups, weekly summary, previews/recaps and corrections.
- **Live:** left—NFL games and follow selector; center—field, last play and fantasy play spine; right—fantasy scores and selected entity inspector; drawer—NFL game box. Keep the followed-game bar visible. When no game is live, show the schedule and its next kickoff instead of a blank field.
- **Race:** left—standings; center—seeds and bubble; right—selected franchise inspector; drawers—tiebreakers, bracket, power, all-play and starter points.
- **Archive:** left—season selector; center—selected season's checked facts; right—selected franchise/player inspector; drawer—available season detail. Do not invent unavailable historical facts.

Keep text legible and stable, with hard edges, restrained labelled accents and no glow. Every data panel identifies source, as-of time and loading, empty, stale or error state; never present a stale score as live. Drawers preserve selection and return focus when closed.

## Cuts and merges

- Merge schedule presentations into one drawer with week, franchise and compact lenses; preserve all three information sets.
- Merge the two lineup reads into one week/franchise selector with expandable detail; keep the data, not two report destinations.
- Merge power presentations into one ranking surface; retain season versus franchise, roster-count choices and the bar chart, and label the calculation. Do not equate power with standings.
- Merge matchup/result lists by game state and preview/recap by selected game. Keep fantasy and NFL boxes distinct.
- M-088 compares first held results with latest results; label that baseline and do not imply it is an exhaustive official correction feed.

## Keep out

Trades, market pricing, draft management, roster/lineup commits, commissioner settings, chat, news feeds, generic NFL research and decorative dashboards. Link to their existing homes rather than duplicating their workflows.

## Evidence and limits

I inspected the listed Now/Race frontend and backend through their entry points and sampled other files, the archive reader, registry placements, spec §§4 and 14, and both approved mock files' layout/drawer markup; I did not read every file or render the mocks. The spec says an `Archive.tsx` screen existed, while the brief says there is no screen; I did not inspect that frontend file, so current screen status is unverified. Archive detail availability and the proposed panel fit need verification against real data and a 1920×1080 mock.
