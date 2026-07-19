# SESSION 0 (TEST SLICE) — Shell Spatial Direction

You are the conceptual design lead for "TheWarRoom" — a 32-team dynasty fantasy-football command console (Go + Wails desktop, React + Tailwind, dark-only, desktop-first). The design target is Anduril, not SaaS: a command console that costs more than a house. Dark, precise, data-dense without a pixel of waste. Confident hierarchy. Controls that look like they actuate real hardware. The UI must communicate capability before the user clicks anything.

This is a deliberately SMALL test slice of a larger design program. Answer only what is asked, with committed specifics — named values and rules, not mood language. Flag uncertainty rather than guess. Target ≤120 lines of markdown.

## Locked context (do not redesign)
- Shell: 4 columns — nav rail (~160-200px, 6 modules) / fluid workspace / contextual inspector (~200-320px) / comms panel (~280px open, 48px collapsed strip).
- Density tiers Narrative/Tactical/Matrix ride CSS custom properties only; Tactical is default.
- Spectrum law: a 1–2-league casual and a 25-league portfolio manager share this product. Controls acknowledge <100ms; motion is feedback, not decoration. Depth is discovered, never demanded.
- Reference surface for this slice: the M1 Asset Rankings board — a 32-team ranked player table (columns: Rank, Player, Pos, Franchise, Base, Adjusted Score, Salary), position filter, view toggle (Global / By franchise / Cap efficiency).

## Your charge (test slice only)
Provocation seed: **"instrument cluster"** — zones with fixed roles, asymmetric, real estate is earned.

Deliver, in this order:
1. **Base unit + spacing scale** — one number and its derived scale, with a one-line reason.
2. **Grid thesis** — one paragraph: how the workspace grid is organized under the instrument-cluster seed, and what makes it NOT a generic dashboard grid.
3. **Panel anatomy** — the parts of a panel (header/body/chrome), their heights/weights, and the rule for how a panel earns a border vs elevation vs nothing.
4. **The M1 board treatment** — how the ranked table sits in the workspace: header band, filter placement, row height at Tactical, what dominates visually and what recedes.
5. **One RIPCORD ITEM if you have one** (a locked decision you'd challenge), else say "none".
