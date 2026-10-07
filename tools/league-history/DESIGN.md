---
name: Legacy NFL Lab
description: A cold-navy data interface that shows where the whole league misprices assets, and when.
colors:
  canvas: "#0c1119"
  surface: "#101722"
  raised: "#1b2635"
  line: "#2e3d50"
  grid: "#1f2b3a"
  text: "#e5edf5"
  muted: "#a5b4c6"
  faint: "#7f90a6"
  accent: "#86bbf1"
  accent-ink: "#0c1119"
  green: "#7fd1b0"
  rose: "#e9a8b9"
  amber: "#edc689"
  green-tint: "#13291f"
  rose-tint: "#2a1820"
  selected-fill: "#1d3550"
  bar-fill: "#34506e"
typography:
  headline:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "28px"
    fontWeight: 650
    lineHeight: 1.5
    letterSpacing: "-0.02em"
    fontFeature: "tnum"
  finding:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "22px"
    fontWeight: 560
    lineHeight: 1.35
    letterSpacing: "-0.01em"
    fontFeature: "tnum"
  title:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "17px"
    fontWeight: 650
    lineHeight: 1.5
    letterSpacing: "-0.01em"
    fontFeature: "tnum"
  figure:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "22px"
    fontWeight: 650
    lineHeight: 1.2
    letterSpacing: "-0.01em"
    fontFeature: "tnum"
  body:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "tnum"
  data:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "13px"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "tnum"
  label:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, \"Segoe UI\", sans-serif"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.4
    fontFeature: "tnum"
rounded:
  cell: "3px"
  control: "4px"
  panel: "6px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "22px"
  gutter: "28px"
  rail: "232px"
components:
  panel:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.panel}"
    padding: "14px 16px 12px"
  nav-item:
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "7px 10px"
  nav-item-active:
    backgroundColor: "{colors.raised}"
    textColor: "{colors.text}"
  button-ghost:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "6px 11px"
  button-ghost-hover:
    backgroundColor: "{colors.raised}"
  button-link:
    textColor: "{colors.accent}"
    typography: "{typography.data}"
  segmented-option:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.text}"
    padding: "4px 10px"
  segmented-option-on:
    backgroundColor: "{colors.selected-fill}"
    textColor: "{colors.text}"
  select:
    backgroundColor: "{colors.canvas}"
    textColor: "{colors.text}"
    rounded: "{rounded.control}"
    padding: "5px 8px"
  market-line:
    textColor: "{colors.text}"
    padding: "12px 8px"
  market-line-hover:
    backgroundColor: "{colors.raised}"
  strip-cell:
    textColor: "{colors.muted}"
    rounded: "{rounded.cell}"
    padding: "4px 8px"
  strip-cell-marked-cheap:
    backgroundColor: "{colors.green-tint}"
  strip-cell-marked-dear:
    backgroundColor: "{colors.rose-tint}"
  drawer:
    backgroundColor: "{colors.surface}"
    width: "min(780px, 100vw)"
  trade-card:
    backgroundColor: "{colors.canvas}"
    rounded: "{rounded.panel}"
    padding: "10px 12px"
---

# Design System: Legacy NFL Lab

## Overview

**Creative North Star: "The League Ledger at Night"**

The Lab is a desk instrument for one GM reading the whole league's market. Everything sits on a cold navy field: flat panels, 1px lines, tabular numerals, and three signal colors that each mean one thing. The interface stays quiet so the findings can be loud. A screen opens on a plain sentence stating the finding. The number and its likely range come after, and the records behind any number are one click away in a side drawer.

Density is high but ordered. Lists and tables carry the work, and the only charts are small SVG instruments: a log-scale cost bar, an effect bar, the league-year picker's bars. They are drawn from the same lines and dots as the rest of the interface. Nothing is decorative. Where a measure does not clear chance it gets no signal color and is set back in muted or faint text, or named as "within chance".

The Lab is league-wide. No team, including Christopher's own, gets a color, a marker or a highlight.

**Key Characteristics:**
- Dark navy field, with tonal steps for depth (canvas, surface, raised) instead of shadows.
- Green means the league sells it cheap, rose means the league overpays, amber is the going-rate reference. Nothing else uses these colors.
- One system-ui stack with tabular numerals everywhere.
- 1px line borders, small radii (3, 4, 6px), no gradients.
- Findings come first as sentences. Numbers, ranges and records follow.
- Every number can be opened to its records in the drawer.

## Colors

A cold, low-chroma navy neutral set with one pale-blue interaction accent and three pastel signal hues. All of them are tuned to read at 4.5:1 or better on every navy surface.

### Primary
- **Ice Blue** (accent): the interaction color. It covers links, "See the N trades" prompts, the focus ring (2px outline, 2px offset), the active nav edge, the selected segmented underline, the neutral range stroke on the draft and price bars, the bars of the chosen stretches in the league-year picker, and the frame of a chosen filter button. The skip link fills with it, with Night Ink (accent-ink) text on top.

### Secondary
- **Sea Glass Green** (green): what the league sells too cheap. It colors the buy-list heading, the cost figure and range stroke of a cheap line and an "up" lean in the draft table.
- **Dusk Rose** (rose): what the league pays too much for. It colors the sell-list heading, the cost figure and range stroke of a dear line, a "down" lean, and error text.

### Tertiary
- **Going-Rate Amber** (amber): the reference, never a verdict. It marks the dashed going-rate tick on the cost bar and its "going rate" label, the dashed 1.0 reference on the draft and price range bars, and the "delivered since" figure and left-edge mark on the asset behind a number in a trade card. It also marks evidence that holds in only one half of the history ("Weaker lately", "Clear only lately", "Fading").

### Neutral
- **Night Canvas** (canvas): the page field, the backgrounds of inputs and segmented controls, and the inside of trade cards.
- **Panel Navy** (surface): panels, list columns, the filter band, the pick clock, the rail and the drawer.
- **Raised Slate** (raised): hover and active fills for nav items, buttons, table rows and market lines.
- **Steel Line** (line): every 1px border on panels, controls, the rail, the top bar and the drawer, plus the minor ticks on the cost bar.
- **Grid Line** (grid): row rules inside tables and lists, the outline of time-of-year strip cells, and the cost-bar axis.
- **Frost Text** (text): primary text, figures and range-bar dots.
- **Mist** (muted): ledes, panel notes, table headers and the meta line under a market line.
- **Faint Slate** (faint): nav hints, scale tick labels, the rail footer, the top-bar field labels and rows too thin to judge. Its value was raised so it meets 4.5:1 on canvas, surface and raised. Do not darken it again.
- **Green Tint / Rose Tint** (green-tint, rose-tint): the dark wash behind a cell that clears chance. Used for marked time-of-year strip cells and for the up and down tones in the draft round grid.
- **Selected Navy** (selected-fill): the fill of the selected segmented option.
- **Histogram Navy** (#4a6f95): the league-year bars when the whole year is in view. Chosen stretches turn Ice Blue; stretches left out drop to Dimmed Bar (#263649) with faint labels. A chosen stretch sits on Chosen Navy (#142235) inside a #2c4a6b frame; chosen buttons and chips fill #1d3550 with an Ice Blue frame. Hovered filter buttons take a #45597a frame.

### Named Rules
**The Three Signals Rule.** Green, rose and amber carry meaning and nothing else. Green is cheap, rose is dear, amber is the going rate or a caveat on evidence. Do not use them for decoration, branding, categories or team identity.

**The Clears-Chance Rule.** Signal color appears only on a finding that clears chance. A strip cell is marked only when that time of year beats both others beyond chance. An asset kind joins a list only when its whole likely range sits on one side of the going rate. Anything weaker stays muted or faint and is described in words ("within chance", "Too few played out to judge").

**The League-Wide Rule.** The Lab shows the league, not a team. No franchise gets a color, a badge, a pinned row or a "your team" emphasis. The trade log may offer a plain team filter for lookups, but a filtered result is styled the same as an unfiltered one.

## Typography

**Body Font:** system-ui (with -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif)

**Character:** A single native interface face with tabular numerals set at the root, so figures line up in every column and every sentence. Hierarchy comes from size and weight steps (400, 560, 600, 650), not from a second family.

### Hierarchy
- **Headline** (650, 28px, -0.02em): the screen title, one per screen ("What the league misprices").
- **Finding** (560, 22px, 1.35, -0.01em, balanced wrap, max 34em): the plain-sentence headline finding under the title. It drops to 19px below 900px.
- **Title** (650, 15-19px, -0.01em): section headings at 19px, list headings at 17px, panel headings at 16px, filter-band and pick-clock headings at 15px.
- **Figure** (650, 22px, -0.01em): the cost figure on a market line, followed by a 12.5px muted unit ("of the going rate").
- **Body** (400, 15px, 1.5): the lede and "say" sentences (15px at 1.45, max 900px). Notes run at 13-13.5px in muted, kept to 62-110ch.
- **Data** (400, 13px): table cells and trade-card assets. Table headers are 560 weight at 12px in muted, and row-header names are 560.
- **Label** (400, 12.5px): nav hints, strip cells, meta lines, the "How this is counted" summary and scale ticks (11.5px in the SVG). Top-bar field labels are the one uppercase use: 12px, 0.06em tracking, faint. They label form controls only.

### Named Rules
**The Tabular Numerals Rule.** Every numeral is tabular. The root sets `font-variant-numeric: tabular-nums`, and numeric columns align right.

**The Sentence-Before-Number Rule.** A finding is set as a sentence in text color at body size or larger. Its figure, range and evidence follow in smaller muted type.

## Layout

A fixed left rail (232px) carries the brand, the screen nav (a label and a one-line hint) and a footer showing the archive date and offline status. The work area to its right has a sticky top bar (10px 28px padding, 96% canvas with a 6px backdrop blur, 1px bottom line) holding the season controls, with a right-aligned note ("The whole league, every team"). Content sits in a main column (22px 28px 60px padding, max 1480px). Heads and ledes cap at 920px.

The market screen is two equal columns: the buy list on the left and the sell list on the right (16px gap). A filter band sits above them (round 7): the league year on the left (1.6fr), the assets on the right (1fr). A "Priced about right" line and a "Too uncertain to judge" line wrap beneath the lists, then the full-width pick clock and the contenders aside, both computed on the same filters. Other screens use stacked panels (16px apart), pinned-conclusion cards in an auto-fit grid (min 280px, 12px gap) and wide data tables.

Spacing follows a 4px base: 4, 8, 12, 16, 22, 28. Desktop is the target. At 1100px and below, split panels stack. At 1300px and below, the filter band stacks. At 900px and below, the market columns stack, the league year shows four stretches to a row without the window headings, and a market line becomes a single column. At 760px and below, the rail turns into a horizontal nav bar, hints hide, and the gutters drop to 12px.

## Elevation & Depth

The system is flat. Depth comes from three tonal steps (canvas, surface, raised) and 1px lines, not shadows. The one exception is the records drawer, which slides over the screen from the right and carries a soft ambient shadow to its left so it reads as above the page.

### Shadow Vocabulary
- **Drawer cast** (`box-shadow: -16px 0 40px rgba(0, 0, 0, 0.45)`): only on the records drawer.
- **Selected underline** (`box-shadow: inset 0 -2px 0 var(--accent)`): a state rule on the selected segmented option. It is not elevation.

### Named Rules
**The Flat Lines Rule.** Surfaces are flat. Separate them with a tonal step and a 1px line. The only element that casts a shadow is a layer that covers the page.

## Shapes

Corners are small and functional. Panels, lists, the filter band, pins and trade cards use 6px. Filter buttons and chips use 3px. Controls, nav items and selects use 4px (`--radius`). Time-of-year strip cells use 3px. Market lines are square (0) because they are rows divided by grid rules. Borders are always 1px solid, Steel Line around containers and Grid Line inside them. Accent edges are 2px: the active nav item has a 2px Ice Blue left edge, and the asset behind a number in a trade card has a 2px amber left edge (other assets get a 2px Steel Line edge). SVG range strokes have round caps, and value dots are filled Frost Text circles.

## Components

### Buttons
- **Ghost:** a quiet outlined control. Panel Navy fill, 1px Steel Line border, 4px radius, 6px 11px padding, 13px text. Hover fills with Raised Slate. Used for Close, Download and "Show more".
- **Link:** text-only, Ice Blue, underlined with a 2px offset, 13px. Used for actions inside pins and notes.
- **Focus:** every interactive element shows a 2px Ice Blue outline at a 2px offset. Market lines inset it (-2px offset).

### Segmented control
- **Style:** inline options inside one 1px Steel Line frame (4px radius), Night Canvas fill, 4px 10px padding, 12.5px text, 1px dividers.
- **State:** hover fills Raised Slate. The selected option fills Selected Navy with a 2px inset Ice Blue underline.

### Cards / Containers
- **Panel:** Panel Navy, 1px Steel Line, 6px radius, 14px 16px 12px padding. The head holds a 16px title, a 13px muted note (max 760px) and right-aligned controls.
- **Pin:** a conclusion card with the same material, 11px 14px padding, a "say" sentence, an optional muted basis line and a link action.
- **"How this is counted":** a disclosure in each panel or screen foot. The summary is faint 12.5px and turns muted on hover. Body paragraphs cap at 820px.

### Inputs / Fields
- **Style:** Night Canvas fill, 1px Steel Line, 4px radius, 5px 8px padding, inheriting the type. The top-bar fields pair an uppercase faint label with the select. Search inputs are at least 260px wide.
- **Focus:** the global Ice Blue ring.

### Navigation
- **Rail:** each item is a 14px 560-weight label over a 12.5px faint hint, padded 7px 10px with a 4px radius and a 2px transparent left edge. Hover fills Raised Slate. Active fills Raised Slate, turns the left edge Ice Blue and lifts the hint to muted. On mobile the rail becomes a horizontal scrolling row with hints hidden.

### Data tables
- **Style:** 13px, collapsed, with a 1px Grid Line under each row and 6px 8px cells. Numbers align right. Headers are muted 12px at 560 weight. Clickable rows and cells fill Raised Slate on hover and open the drawer. Thin-evidence rows drop to faint.

### Market line (signature)
One kind of asset per line, laid out as a four-row grid: name (16px, 600) and cost figure (22px, 650, green when cheap and rose when dear) share the first row, then the cost bar, the time-of-year strip and the meta line. The whole line is a single button with a Grid Line top rule, 12px 8px padding and a Raised Slate hover. The meta line pairs an evidence statement on the left with an Ice Blue underlined "See the N trades" on the right. The statement reads Frost Text when the finding holds in both halves and amber when it holds in only one. On narrow widths every row stacks.

### Cost bar (signature)
A log-scale SVG range bar on a fixed domain (0.2x to 5x the going rate, max 460px wide). The 2px Grid Line axis has minor Steel Line ticks at 25%, 50%, 200% and 400%. A tall dashed amber tick marks the going rate (100%). The likely range is a 4px round-capped stroke in the line's signal color, and the estimate is a Frost Text dot (r 5).

**The One Scale Per List Rule.** All cost bars in a list share one scale. Its tick labels are printed once at the head of the list (faint, with "going rate" in amber), never repeated on each line, so bars compare directly by eye.

### Time-of-year strip (signature)
Three equal cells under each market line: Offseason, Draft to kickoff, In season. Each cell (1px Grid Line, 3px radius, 4px 8px padding) shows the window in muted 12.5px over that window's cost in Frost Text 14px. Only a window that beats both others beyond chance is marked: its border takes the line's signal color and its fill takes the matching tint. A window without data shows a dash. The strip appears only when the stretches in view reach two or more windows; when the view is narrowed, the meta line adds the kind's all-year figure ("· all year 39%").

### Records drawer
A right-hand sheet (min(780px, 100vw)) in Panel Navy with a 1px Steel Line left edge and the drawer cast shadow. The head holds a 17px title, a muted count and note, and Download and Close ghost buttons. The body scrolls and pages its records. Escape closes it and returns focus to the element that opened it. Inside, each trade card is a Night Canvas box (6px radius) with a muted date row and two sides listing assets, side by side, with the asset behind the number edged in amber.

### Draft timing (round 8)
The draft screen reuses the market's grammar: the filter band (seven draft stretches as bars of what
a pick returns, then Position / NFL round chips and the Judge-on segmented control), the per-position
panel (`.ap`, two even columns of `.ap-when-grid` cells; cells colour their figure green or rose only
when the range clears the slot, and a best or worst cell takes the green or rose frame), and two
league maps (`.dm-table`): a cell that clears the slot fills Sea Glass tint `#13291f` or rose tint
`#2a1820` with its figure in the signal colour at 600 weight; picks counts sit beside in faint 12px.

### Filter band (round 7)
The market's controls, in one Panel Navy frame above the lists. **Time of year:** three window headings (Offseason, Draft to kickoff, In season; 12.5px 600, a 1px Steel Line underline that turns Ice Blue when that window is chosen) over eight stretch buttons in calendar order. Each stretch is a column: completed trades a season (muted 12.5px) over a bar up to 52px tall, the stretch's name (12.5px, two lines reserved) and its months (faint 12px). **Assets:** rows of 3px-radius chips (Players by position; Next draft 1st/2nd/3rd+; Later drafts 1st/2nd/3rd+) under 84px muted row titles, with Everything / Players / Picks presets and a Together / Split by age segmented control. Selection rule for both: from everything, a click picks just that one; after that clicks add or remove; emptying returns to everything. Nothing chosen means every control is plain; once narrowed, the chosen are lit and the rest dimmed.

## Do's and Don'ts

### Do:
- **Do** lead every screen and panel with the finding as a sentence, then give the figure, the range and the records link.
- **Do** keep green for what the league sells cheap, rose for what it overpays for, and amber for the going rate and for one-half-only evidence.
- **Do** color a finding only when it clears chance. Leave everything else muted or faint and say "within chance" or "too few to judge" in words.
- **Do** draw every cost bar in a list on the one shared log scale, labelled once at the head, with the dashed amber going-rate mark at 100%.
- **Do** make every number open its records in the drawer.
- **Do** separate surfaces with tonal steps (canvas, surface, raised) and 1px Steel Line or Grid Line borders.
- **Do** keep numerals tabular and right-align numeric columns.
- **Do** keep text at 4.5:1 or better on canvas, surface and raised. Faint Slate is the floor.

### Don't:
- **Don't** highlight, color, badge or pin any team, including Christopher's. The Lab is league-wide.
- **Don't** show a pattern that fails to clear chance as a finding, with a signal color or a marked strip cell.
- **Don't** use green, rose or amber as decoration or category colors.
- **Don't** give each cost bar its own scale or its own tick labels.
- **Don't** add shadows to panels, cards or controls. Only the covering drawer casts one.
- **Don't** put uppercase tracked labels above headings. The uppercase style labels top-bar form fields only.
