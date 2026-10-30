# Data-stub catalog: Claude vs Terra comparison (2026-10-30)

Inputs: the five approved mocks (HQ v10, War Room v6, Pulse v14, Home v5, Control Room v1).
- `claude-catalog-2026-10-30.md`: 330 rows, plus a source map of 78 data domains checked against origin/main 364379ba.
- `terra-{HQ,WR,LP,HM,CR}.md`: 1,890 rows (HQ 214 · WR 365 · LP 369 · HM 382 · CR 560). These are atomic: one row per chip, button and column.
- `terra-catalog-2026-10-30.md`: the first partial run. It is superseded by `terra-HQ.md`.

## Result
- **Coverage agrees.** I matched every Terra leaf name against the Claude catalog by word. Every leaf I checked falls inside a Claude row: stat columns, situation names, filters, button labels and tooltips. Claude has no domain that Terra lacks.
- **Use Terra's NEW/PARTIAL counts with care.** Terra opened only the files it cites, and it says so itself. For "is it built", use the Claude source map, which checked exported funcs per package on origin/main.
- **Use Terra for granularity.** Its rows are the checklist for stub fixtures, one stub per row.

## Findings both passes raised
1. **Bid visibility is inconsistent and needs a ruling.**
   - War Room shows who outbid you and the amount.
   - Home says the runner-up is sealed until the window closes.
   - Control Room says bids are private, even to the commissioner.
   - R2 says no early peeking.
2. **Role variants are unspecified on every owner screen.** It is not defined what commish, co-commish or a DOT sees differently.
3. **The mock inspectors only half-bind the selection.** Clicking a player changes the name, but the other drawers stay as fixture data. The build must make the inspector fully driven by the selected entity.
4. **The mocks show simulation outputs with no algorithm behind them yet:**
   - playoff odds
   - magic number
   - win odds
   - estimated winning bid
   - rivals count
   - power components
   - age curves
   - team needs
5. **Engine items carried forward:**
   - R3 open bids must not exceed cap room. Today the aggregation is left to the caller (`nativecheck/bids.go:186`).
   - `withdraw()` bug at `market/bid_execute.go:136`.
   - Seven jobs have no producer yet.

## Data-path scale (Claude source map)
- 27 domains exist.
- 31 are partial.
- 27 are new.
- Biggest new clusters:
  - private owner stores: notes, scouting, plans, pocket, versions, targets, watchlist, layout, snooze, saved views
  - models: Best 21, age curve, needs, posture lens, sim, bid market
  - NFL content: depth charts, play-by-play, news relevance, usage stats
  - archive narrative: eras, lineage, stories, trophies
  - permission sets
