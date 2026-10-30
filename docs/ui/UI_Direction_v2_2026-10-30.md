# UI direction v2: command center (2026-10-30)

Rulings from Christopher's UI/UX groundwork session. Amends `UI_Direction_Document.md` and `Target_UI_Spec_2026-10.md`; the four-channel visual system (colour = meaning, frame = consequence, countdown = urgency, glyph = classification) still governs.

## Look
- Contrast is a textured rubber grip / safety edge on heavy equipment: knurl texture, hard borders, monochrome hazard stripes. Never glow, bubbles, soft gradients or blur.
- Hover depth is subtle: 1–2px transform plus a hard offset shadow, under 80ms, transform/opacity only. Hover never lifts past an item's G-level frame.
- Alert colour is an accent (bar, label, dot), not a wash. Text stays high-contrast.

## Space
- A cinematic terminal: the page never scrolls. A fixed-height table may scroll inside its own frame.
- Extra detail opens in drawers: side drawer for a task, bottom drawer for "right now" data, nested drawer for confirms only (max stack 2), peek strip for previews. Player and team detail always goes to the inspector.
- Rows stay compact (30px). Spare screen space gets real function (matchup, form, inline taxi/IR, ledgers), never stretched rows. Primary target 1920x1080; checked at 2550x1352.

## Rail
- Opens on click. Six fixed nodes; leaves never move.
- One pinned NOW line per node, set by season phase.
- "My shortcuts" group at the top, filled by the owner in Customize mode (pin ☆, drag grip).

## Interaction
- Position tags are drag handles. Bench → lineup, lineup → bench, lineup ↔ lineup; the lineup auto-adjusts (bumped starters cascade into weaker eligible slots, holes auto-fill from the best eligible bench player not on bye).
- The app never second-guesses a deliberate move. If the user commands badly, they lose.
- Commit controls live only in action trays and drawers, never in read tables. G3 is a ~600ms hold.
- Owners can add their own inspector drawers (marked MINE).

## Approved additions
Function with form is welcome; slop is not. Approved on HQ: this-week matchup panel with win %, last-5 form bars, inline taxi/IR on tall screens, decisions-due list, cap runway, dead-cap ledger, season move ledger. Win % is new engine math; nothing computes it yet.

## Mocks (docs/ui/mocks/)
- `rail-mock-v1.html`: rail options (option 3 ruled).
- `franchise-hq-mock-v8.html`: Franchise HQ, all three workspaces, RFA/UFA contract caps, content-sized release drawer, release drawer with G3 hold, Customize mode.
- `glm-rail-funnel-2026-10-30.md`: GLM 5.3 rail tree. Leads only; triaged in session.

## Contract end (10-30)
- Final-year cap shows RFA (lime) or UFA (signal yellow) as a textured rubber block with a legend. These hues are classification only and sit outside the five meaning colours; always paired with a letter or legend.
- Drawers are sized to their content; a confirm drops in below the actions instead of covering them.
