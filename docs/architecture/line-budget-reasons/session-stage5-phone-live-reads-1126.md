# Line budget reason: `session/stage5-phone-live-reads-1126`

- `frontend-src` (+21 lines): Extract shared `startBrief` / `startLiveReads` helpers and narrow `readLiveBrief` in `frontend/src/app/shell/homeReads.ts`, and wire `frontend/src/app/phone/LiveTab.tsx` default reader to `readLiveBrief` so phone Live skips the 7 Home-only reads while preserving Thin Home fanout and identical data/error semantics.
- `frontend-tests` (+227 lines): Add transport-level regression tests in `frontend/src/app/shell/homeReads.test.ts` and `frontend/src/app/phone/LiveTab.test.tsx` proving the default Live path skips all Home-only commands on first load and invalidation, renders results/lineups, surfaces affected errors, and keeps full Thin Home fanout intact.
