---
name: Legacy NFL History
description: Evidence-first franchise behavior laboratory in WarRoom's inherited data-interface world
colors:
  canvas: "#0c1119"
  surface: "#101722"
  raised: "#1b2635"
  line: "#2e3d50"
  text: "#e5edf5"
  muted: "#a5b4c6"
  focus: "#86bbf1"
  green: "#a5dcc7"
  partial: "#edc689"
  flag: "#e9a8b9"
  link-hover: "#cce2ff"
  control-hover: "#223247"
  selected-row: "#1c2d3f"
  table-line: "#2d3b4d"
  heat-zero: "#182331"
  heat-one: "#233d3e"
  heat-two: "#2d5850"
  heat-three: "#326d5b"
  heat-four: "#68b698"
  negative-one: "#402c35"
  negative-two: "#65404b"
  negative-three: "#895162"
  negative-four: "#b46e82"
typography:
  body:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif"
    fontSize: "15px"
    lineHeight: 1.5
  heading:
    fontSize: "34px"
    fontWeight: 650
    lineHeight: 1.2
    letterSpacing: "-0.025em"
rounded:
  control: "4px"
  heat-cell: "3px"
spacing:
  tight: "4px"
  group: "12px"
  section: "32px"
---

## Overview

Inherits the existing WarRoom cold-navy, flat, scan-oriented identity. This is an
Operate/Read surface used beside the league archive, not a new marketing identity.
The real year/month calendar leads; details remain readable in evidence tables.

## Colors

Surface variables in `style.css` are authoritative. The heatmap's six-step sequential
ramp in `behavior_views.mjs` is logarithmic; numeric labels remain exact. Levels four and five use
dark ink. The rose negative-net ramp encodes outgoing net quantity, not bad performance;
its positive counterpart is green. Partial periods use amber; blue denotes focus and
links, not activity volume. Flags use rose text. Neither sign nor hue asserts value.

## Typography

System workhorse sans with tabular numerals throughout. Display hierarchy is compact
and secondary to the data. Main heading scales on mobile; table/heatmap labels are
12–13px, with smaller secondary metadata. Source paths and indexed JSON alone use
monospace. Body 15px desktop/14px mobile; compact h2 1.3rem and h3 1.07rem. Inherited
fractional-rem label scales are intentional density tiers, not a separate display
font system. Detector off-ramp size advisories are recorded, not silently suppressed.

## Layout

1500px maximum content width with 3.5% side padding. At 1050px the layout changes to
three-column primary/four-column advanced filters and single-column dossiers. At
600px the root is 14px with 1rem margins, full-row wrapping navigation and two-column
filters. Desktop primary controls form six horizontal columns. Questions/saved lenses
and detailed facets are progressive disclosures; they can remain open during work. Calendar remains a two-dimensional horizontal scroller,
with sticky year labels. Evidence tables scroll rather than collapsing columns.

## Elevation & Depth

No shadows or simulated physical materials. Borders and surface tone define groups.

## Shapes

Nearly square controls, dense rectangular heat cells, one-pixel edges. No card grid,
large pills or decorative containers.

## Components

Month/day cells are buttons with numeric labels, descriptive accessible names and
pressed state. Sources use inline details, not modals. Tables have real headers.
One shared franchise/player/calendar/source-season/policy/context lens carries across
analysis views. Trade-only facets are disabled for non-trade channels, prior-record
facets require a subject, and incompatible asset measures offer a clear-facets action.
Auxiliary draft source-season and weekly-observation scopes are labelled explicitly.
Selected period buttons use pressed state; filters preserve keyboard focus on repaint.
Saved lenses, deletion, URL sharing, CSV and findings JSON are explicit actions.
Render loading, retryable error and no-match states explicitly. Focus is a two-pixel
blue outline. Feedback motion is short brightness change; reduced motion disables it.

## Do's and Don'ts

- Preserve calendar-year vs. source-season vocabulary and source evidence.
- Never imply a missing endpoint, missing champion or future month is a observed zero.
- Keep completed deals, pick quantities and proposals distinct.
- Do not infer owners or continuous ownership from a franchise ID or snapshot.
- Do not add remote font/icon dependencies or imagery to this data surface.
- Keep shares/percentage-point differences, counts and calendar-date rates visibly distinct.
- Leave conditional denominator and exclusion explanations beside the relevant measures.
- Unknown pick encoding blocks complete-package interpretation, not just numeric pick counts.
- Mobile tables retain their real columns in named keyboard-focusable horizontal scrollers.
