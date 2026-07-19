# SESSION 0 — Shell Spatial Direction

## 1. Base unit + spacing scale
**Base unit: 4px.** Pixel-snappable at 1× and 1.5× DPI, divides cleanly into 8/12/16 typographic rhythm, and forces density decisions to declare themselves.

| Token | Value | Use |
|---|---|---|
| `--u-1` | 4 | inline icon gaps, hairline offsets |
| `--u-2` | 8 | control padding, row inset minimum |
| `--u-3` | 12 | default gutters, body inset |
| `--u-4` | 16 | panel padding, section gap |
| `--u-6` | 24 | panel-to-panel gap |
| `--u-8` | 32 | header band height unit |
| `--u-12` | 48 | module break |
| `--u-16` | 64 | viewport margin floor |

No half-units. If something needs 6px, it's wrong.

## 2. Grid thesis
The workspace is **three fixed-role zones, not a Bento field**: a dominant *read zone* (the ranked table, ~62% of workspace width), a *control band* (filter + view toggle, 40px tall, full-width, never reflows), and a *delta edge* (cap/salary spark, right-anchored, 120px, only renders when a row is hovered or pinned). Zones do not compete for real estate — optical weight is allocated by *decision frequency*, not by how much data a zone could display. A zone that isn't earning its pixels at the current task collapses to a labeled strip; it does not sit idle. This is not a dashboard because nothing tiles — the table is the instrument, everything else is bezel.

## 3. Panel anatomy
- **Header**: 32px tall, `--u-2` horizontal inset, label at 11px/600 uppercase tracked +0.08em, right-aligned count or state chip at 11px/400. No icon unless the module is one of the six nav-rail modules.
- **Body**: `--u-3` inset all sides, `--u-4` between sub-sections.
- **Chrome**: 1px top divider only (header↔body). No footer by default; footers appear only on panels holding actuated controls.

**Earning elevation:**
- **Nothing**: inline content, table rows, form fields.
- **Border (1px, `--line-1`)**: persistent structural panels. Default state.
- **Elevation (shadow `0 8px 24px rgba(0,0,0,0.5)`, no border)**: transient surfaces only — popovers, dragged-row ghost, modal sheets. Never applied to a panel that lives in the shell.
- Rule: *if it's there when you reload, it gets a border. If it's there because you did something, it gets elevation.*

## 4. M1 board treatment
- **Header band**: 40px, sits flush under the control band. Contains the board title `M1 // ASSET RANKINGS` (13px/600), live count chip right-aligned, and the position filter as a segmented control — not a dropdown. Segments: `ALL · QB · RB · WR · TE · DST`. Active segment gets a 2px bottom rule in `--accent`, not a fill.
- **View toggle** (Global / By franchise / Cap efficiency): right side of control band, three text buttons, active state = 1px inset border + slightly brighter text. No pill, no fill.
- **Row height at Tactical**: **28px.** Columns: Rank (32px, right-aligned, tabular nums) · Player (fluid, min 180px) · Pos (40px) · Franchise (72px, tricode) · Base (64px, tabular) · Adjusted (72px, tabular, bold-weight — this is the dominant value) · Salary (88px, right-aligned, tabular, muted). Header row identical height, uppercase 10px, bottom border 1px.
- **Dominates**: Adjusted Score column (heavier weight, slightly brighter foreground `--text-1`), the rank number on hover (background tint), and the hovered row's 1px left-edge accent rule.
- **Recedes**: Salary (muted `--text-3`), Franchise tricode (same), Pos label (small caps, dimmed). Base score sits between — present but not competing with Adjusted.
- **No row striping.** Hover is a 4% white overlay. Selected (pinned into inspector) gets the 2px left accent rule persistently.

## 5. Ripcord item
**The comms panel as a fixed fourth column.** A 280px column that's open by default eats ~18% of a 1440px viewport for what is fundamentally an interrupt surface. I'd challenge this to a *summoned overlay* that slides over the inspector edge on ⌘K or notification tap, same width, but zero footprint when idle. The collapsed 48px strip can stay as the trigger. If comms must persist for the 25-league PM persona, make it a density-tier-aware default: Narrative = collapsed, Tactical = collapsed, Matrix = open.
