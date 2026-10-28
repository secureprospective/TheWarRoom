# Research surface

## Direction contract

THESIS: one read-only player-research desk, not a second valuation engine. Find a player, then
choose one report. Research never writes production or scouting observations.
OWN-WORLD: inherit the existing shell's neutral surfaces, typography, monochrome ownership
marker and shared Report chrome. Injury designations stay words from MFL, never invented grades.
STORY: the GM identifies his player, compares this league's scoring, checks role and injuries,
and hands off to the full MFL news article only when he asks.
FIRST VIEWPORT: search by name/position/NFL team above report buttons; my performers first,
then all stats and leaders. Choices sit immediately above the chosen report. Source and as-of
precede rows; missing feeds remain named lines. True weekly/depth grids scroll only internally.
FORM: the brief pins the existing Race composition, with the shared ReportView and lazy screen.
FINISH: offline wire/DOM/state tests, both required gates, and documented live-click tests for
Claude's review. No graphical program or network service is used in this build.

## Performance scoring

Player stats, My top performers and Position leaders sum exactly the weekly scores they
show, at their displayed two-decimal precision, including a held current-week score.
A missing week stays a dash and contributes nothing; partial/stale history remains labelled.
Players come from those weekly feeds, not from a separately refreshed YTD list. Model totals
and the inspector's finished-week-only calculations keep their existing meanings.

Archive matching checks both the league URL season and an explicit `playerScores` `YEAR`.
The model legitimately fetches `YEAR=<previous season>` through the current league URL;
that copy must never answer a current-season Research hydration or failed-refresh fallback.
