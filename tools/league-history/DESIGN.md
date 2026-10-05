---
name: Legacy NFL History
description: Read-only league history in WarRoom's existing data-interface world
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
ramp in `app.js` is logarithmic; numeric labels remain exact. Levels four and five use
dark ink. Partial periods use amber; blue denotes focus and links, not activity volume.

## Typography

System workhorse sans with tabular numerals throughout. Display hierarchy is compact
and secondary to the data. Main heading scales on mobile; table/heatmap labels are
12–13px. Source paths and indexed JSON alone use monospace.

## Layout

1500px maximum content width, fluid 3.5% side padding; 18px mobile margins at 760px and
below. Desktop controls form grouped horizontal bands. Mobile navigation is a full
second row; filters stack. Calendar remains a two-dimensional horizontal scroller,
with sticky year labels. Evidence tables scroll rather than collapsing columns.

## Elevation & Depth

No shadows or simulated physical materials. Borders and surface tone define groups.

## Shapes

Nearly square controls, dense rectangular heat cells, one-pixel edges. No card grid,
large pills or decorative containers.

## Components

Month/day cells are buttons with numeric labels, descriptive accessible names and
pressed state. Sources use inline details, not modals. Tables have real headers.
Global franchise/player filters are separated from calendar-only date/activity filters.
Render loading, retryable error and no-match states explicitly. Focus is a two-pixel
blue outline. Feedback motion is short brightness change; reduced motion disables it.

## Do's and Don'ts

- Preserve calendar-year vs. source-season vocabulary and source evidence.
- Never imply a missing endpoint, missing champion or future month is a observed zero.
- Keep completed deals, pick quantities and proposals distinct.
- Do not infer owners or continuous ownership from a franchise ID or snapshot.
- Do not add remote font/icon dependencies or imagery to this data surface.
