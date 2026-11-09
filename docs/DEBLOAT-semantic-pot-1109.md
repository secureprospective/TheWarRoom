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
