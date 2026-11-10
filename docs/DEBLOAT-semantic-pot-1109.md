# TheWarRoom semantic pot (words that drifted; fix after the cut)

Started 2026-11-09. One line per item: where, what it says, what is true now.

1. README + North Star: "MFL is the record / MFL wins (R2)". True now: the native ledger wins, MFL is import and reference only (ruling 2026-11-05, in CLAUDE.md).
2. README "6-layer engine, 10 position models". `internal/engine` root is 58 lines; the live valuation is the fitted model (`internal/model`, `modelrun`, `appvalue`). `internal/engine/*` today is league mechanics (rookie draft, competition, bids, eligibility, race).
3. README "Atomic Transaction Engine / single transaction coordinator" = `internal/transactions`, which only the desktop MFL-reference what-if path links. The native ledger writer is `internal/commissioner` + `internal/store/history`.
4. `internal/composition/lab_app.go` is production wire types, not Lab code (bloat ledger 2a).
5. Root CLAUDE.md on `main` checkout (v3.0) vs branch CLAUDE.md (v3.2) differ; Lab moved to its own repo 2026-11-04 but CLAUDE.md still describes "the Lab screens".
6. Handoff says Beelink is 192.168.1.191; standing instructions say 192.168.1.190.
7. "Research", "Lab", "Market" overlap: `internal/app/research`, `app/market`, `composition/lab_app.go`.
8. North Star Phase 1 "read-only against MFL, personal tool" vs today's hosted multi-owner server with join page.
9. Handoff size table (420k) omits test data, JSON fixtures and mock HTML; tracked total is 725k.
10. "MFL mirror" / "MFL reference" / "mflref" / "what-if league": true meaning now = the solo-owner framework (one owner using TWR tools on their own MFL league). Not a league mode. Ruling 2026-11-09.
11. "Import" vs "migration" vs "Move my MFL league" vs "seed": one thing, MFL→Native import. Pick one word.
12. README, North Star and CLAUDE.md all describe a "desktop app" product; none says the PWA is the main interface (only `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` does). The debloat handoff does not mention the PWA at all.
13. "hosted" / "browser" / "PWA" / "server client" name one surface in different files.
14. "Research" (found 11-09 evening): the ruling "research screens split OUT of the app" is already done in the
    frontend (the screen is gone; 307 lines remain). What is still named research (`internal/app/research`,
    `ingestion/researchfeeds`, `frontend/src/app/research`, the `research.read` browser command) is now the
    player-reading feed behind Home tiles, the Franchise HQ and War Room inspectors, roster facts and offers.
    Not a cut. Rename candidate: "player readings".
15. Project `CLAUDE.md` and `AGENTS.md` still name `docs/build-handoffs/Hardening_Plan_2026-10.md` as "the
    current plan" and `UI_Target_Roadmap_2026-10.md` as the UI build order; the live direction is
    `docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` (PWA-centric) plus the 11-09 rulings. Six handoff docs that
    those files and README link to were restored after batch 1 deleted them. Decide which are spent, then
    fix the instruction files and delete together.
16. `CLAUDE.md` names `registry/endpoints.gen.ts` (now `frontend/src/app/registry/endpoints.gen.ts`).
