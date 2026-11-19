# Browser door migration blocker — superseded

**Superseded 2026-11-18 by Part 3 of the Bundle 7 task:** do not add last-used league state or a schema migration. `POST /app/session` refuses accounts admitted to more than one league with HTTP 409 and the specified desktop-app message. Implement the browser door without recency state.

The original concern and rejected recency approaches remain in the commit that introduced this note. The active lane state and next steps are in `HANDOFF.md`.
