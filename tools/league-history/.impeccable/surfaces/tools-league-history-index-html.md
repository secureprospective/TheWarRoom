---
version: 1
slug: "tools-league-history-index-html"
primary_target: "tools/league-history/index.html"
related_targets: ["tools/league-history/style.css","tools/league-history/app.js"]
---

# League history PWA

Mode: Operate / Read. Christopher's first task is finding when picks move over fourteen years, then tracing the teams and players involved. User pinned a clickable year-by-month heatmap and an extensible PWA. Existing WarRoom navy identity is inherited, not replaced. Code-led; no image generation is available.

## Direction contract
THESIS: A fourteen-year calendar is the actual work surface, not a dashboard thumbnail. Refuses decorative cards ahead of the evidence.
OWN-WORLD: Inherit WarRoom's cold navy surfaces, workhorse sans, blue focus edges, subdued green sequential data ramp, square dense table language.
STORY: Filter by team, player, event type and dates; click a month to expose exact events and provenance; continue into histories.
FIRST VIEWPORT: Compact identity and navigation, strong calendar heading, labelled filters, a full-width fourteen-row/twelve-column heatmap with counts and partial-period treatment. Event ledger immediately beneath. Signature interaction: selecting a cell exposes its dated ledger without losing the calendar context.
FORM: User-pinned calendar matrix; no concept roll because the principal composition is specified. Mobile preserves the two-dimensional calendar in a labelled horizontal scroller, not unreadable tiny cells. Motion is feedback-only; reduced motion respected.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance.
