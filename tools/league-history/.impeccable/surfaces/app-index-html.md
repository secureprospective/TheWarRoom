---
version: 1
slug: "app-index-html"
primary_target: "app/index.html"
related_targets: ["app/js/views/market.js"]
---

# Surface: the Lab's market screen (app/index.html, #/market)

Mode: operate. Desktop, wide screen. League-wide only; no team lens (PRODUCT.md).
Audience and job: Christopher, a GM, planning what to buy and sell and when, from how the
other 31 GMs price assets. Proof: the league's own valued trades since 2017, checked MFL facts.

## Direction contract

THESIS: Two lists, a buy list and a sell list: what the league sells too cheap and what it pays
too much for, each line saying when in the year it is cheapest. Refuses the full price table of
every asset kind with six numeric columns, where the reader has to find the finding.

OWN-WORLD: The Lab's established cold-navy data interface (canvas #0c1119, surface #101722, line
#2e3d50, text #e5edf5). Green #7fd1b0 marks what the league sells cheap, rose #e9a8b9 what it
overpays for, amber #edc689 the going rate. Flat panels, 1px lines, tabular numerals, one
log-scale range bar per line centred on the going rate.

STORY: He picks the stretch of the league year and the assets he cares about in the filter band
(the league year doubles as the chart of when the league trades), reads one sentence, scans the
buy list and the sell list repriced for that choice, sees which findings hold in both halves of
the history and how each compares with all year, then the pick clock for the same trades. He
clicks a line to see the trades behind it. (Round 7, 2026-10-06: Christopher asked for more
control over time of year and assets, with the calendar blended into the filters.)

FIRST VIEWPORT: Seasons control in the top bar. The filter band: the league year (eight stretches,
trades a season in each, grouped under the three windows) and the asset chips. Headline sentence at 22px. Below,
two equal columns: left "The league sells these too cheap", right "The league pays too much for
these"; each line a kind of asset, its cost as a percent of the going rate with its range bar,
an evidence mark and a three-cell time-of-year strip. A thin "priced about right" line under the
columns, then the pick clock.

FORM: Buy list, sell list; position 1 on the ordered list of six; seed key e4f51cb1.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

## Unresolved
- Draft timing, trade log and method screens keep their structure this round, with the team lens removed.
