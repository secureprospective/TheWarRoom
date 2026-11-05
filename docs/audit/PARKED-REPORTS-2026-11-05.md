# Parked reports, 2026-11-05

The Research screen (War Room > Research) and Home > Digests were cut on 11-05; neither is in a locked mock.
Christopher: "we will want those reports for filtered stat inquiries by owners ... Flag it for now so we don't loose them."

Each row below is `retired` in docs/ui/endpoint-registry.csv with a disposition_note starting "PARKED 11-05".
The backend still builds these reports (internal/app/research). To bring one back: place owner stat inquiries in a mock,
set the row to `kept` with that placement, and give it a screen. List them: `grep "PARKED 11-05" docs/ui/endpoint-registry.csv`.

Moved, not parked: M-048 depth charts, M-058 player news, M-059 article page -> Home > Seasonal card (mock tiles); M-060 my players' news -> Inspector.

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
