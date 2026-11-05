# Parked reports, 2026-11-05

The Research screen (War Room > Research) and Home > Digests were cut on 11-05; neither is in a locked mock.
Christopher: "we will want those reports for filtered stat inquiries by owners ... Flag it for now so we don't loose them."

Each row below is `retired` in docs/ui/endpoint-registry.csv with a disposition_note starting "PARKED 11-05".
The backend still builds these reports (internal/app/research). To bring one back: place owner stat inquiries in a mock,
set the row to `kept` with that placement, and give it a screen. List them: `grep "PARKED 11-05" docs/ui/endpoint-registry.csv`.

M-060 my players' news remains in Inspector.

| id | report | owner question |
|---|---|---|
| M-020 | Game Injury Report (previews child) | Who's hurt for this game? |
| M-032 | Transactions O=03 | What moves happened league-wide? |
| M-037 | Player Stats O=08 | Who's producing, at what level? |
| M-038 | My Top Performers | Who's carrying my team? |
| M-040 | Top Passers | QB leaders |
| M-041 | Top Rushers | RB leaders |
| M-042 | Top Receivers | WR/TE leaders |
| M-043 | Top Kickers | K leaders |
| M-044 | Top Defenders (IDP) | DT/DE/LB/CB/S leaders |
| M-045 | Top Defenses (team) | Team DST leaders |
| M-050 | NFL Team Stats | Team context |
| M-051 | NFL Team Injuries | Team injury context |
| M-052 | Top Adds/Drops/Starters O=117 | What is the league doing? |
| M-053 | Points Allowed by Position O=81 | Streaming matchup edge? |
| M-054 | Points Allowed per NFL team O=243 | Matchup detail |
| M-057 | NFL Injury Report | League-wide injury state |
| M-062 | Player Search | Find the player |
| M-163 | Top lists by position ◆ (11 splits × 2 lists) | Best at each position |
| M-164 | Top Adds/Drops/Started/Owned ◆ | Market pulse |
| M-165 | LATEST NEWS tab (shell) | News view |
| M-186 | Sortable Stats | Stat table |

## Seasonal card (sometimes tab) — UI home TBD

The Home tab was removed; retain its content for a future, undecided UI home. Existing Inspector news remains.

| id | content |
|---|---|
| M-048 | Fantasy Depth Charts |
| M-049 | Depth Chart Grid (merged into M-048) |
| M-058 | Player News |
| M-059 | News Article Page |
| M-147 | HOME tab (merged into seasonal card) |
| M-166 | News digests (merged into M-058 and C-05) |
| M-189 | Depth Charts (merged into M-048) |
| C-03 | Season, phase, week and deadlines |
| S-13 | Seasonal phase links |

## Alert tray (sometimes tab) — UI home TBD

The Home tab was removed; retain its content and deadline pinning for a future, undecided UI home.

| id | content |
|---|---|
| M-148 | Lineup Alert |
| M-149 | Possible IR Violations |
| M-153 | Pending Trades |
| M-156 | Waiver Claims |
| M-157 | Waiver Order |
