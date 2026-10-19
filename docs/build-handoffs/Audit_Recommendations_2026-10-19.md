# Deep-audit recommendations to fold into the Hardening Plan (2026-10-19)

Source: `docs/build-handoffs/Deep_Audit_2026-10-19.md` (findings N1–N4, P1–P4, S1–S3, R1–R3, M1–M3, E1–E2).
Plan: `docs/build-handoffs/Hardening_Plan_2026-10.md` (F1–F15, G1–G9, phases 0–12, rulings 1–7).

Each recommendation names: what changes, done-when (the test or measurement), where it goes in the plan, est. days.

| ID | Recommendation | Done when | Plan slot | Days |
|---|---|---|---|---|
| A1 | Bump Go toolchain 1.26.4 → 1.26.9; add `govulncheck ./...` to `make verify` (fail on reachable vulns) | govulncheck reports 0 reachable | new 1b | 0.1 |
| A2 | Webview CSP meta (`default-src 'self'; img-src 'self' data:`), pin MFL discovered host to `*.myfantasyleague.com` (refuse otherwise), cap MFL response bodies (e.g. 32 MB, LimitReader), create DB files 0600 | a test per item fails without it | new 1b | 0.4 |
| A3 | Native bid place/raise refused once `standingAt + stand` has passed (auction is closed, awaiting processing) | failing-first test: bid at stand+1s refused; snipe window 20–24h unchanged | phase 4 | 0.5 |
| A4 | Native identity: random installation id generated once, stored in thewarroom.db (and backed up); principal = hash(installation id), not data-dir path + uid. Migration maps the existing path-derived principal to the new id so current leagues keep their commissioner. Add a commissioner recovery command (G3) | restore of a backup into a different HOME keeps commissioner rights (test) | phase 3 (migration) + phase 6 gate | 0.5–1 |
| A5 | Native clock: replace 1-second polling with sleep-until-next-due + wake on ledger append; CPU profile before/after | idle CPU < 2% of a core on Claude-OS with the dev db; no due item processed later than 1 s while open | phase 7 | 1 |
| A6 | Hourly refresh skips crosswalk / archive read / model fit / engine run when inputs' hash unchanged | an unchanged hour logs "skipped" and takes < 3 s total | phase 7 | 0.5–1 |
| A7 | Screens read the last published snapshot; refresh builds off-lock and publishes atomically (merges with F1/F2 fix) | a test holding a slow fake MFL refresh: TargetSnapshot returns in < 200 ms | phase 4 | 1 |
| A8 | Shutdown waits on every background worker before closing DBs | goleak/shutdown test with an in-flight facts compile | phase 4 | 0.25 |
| A9 | Split `package main` (24.6k lines): move `*_reading.go` presentation logic into `internal/readings/<area>`; App becomes thin feature services. Done together with faster tests | root package < 8k lines; root -race test < 10 min; coverage unchanged | phase 2 | 2–3 |
| A10 | Frontend: ESLint with react-hooks rules only; upgrade Vite/Vitest/TypeScript/plugin-react together | `pnpm audit` 0 high/critical; bundle budget passes; eslint clean | phase 2 (tooling) | 0.5–1 |
| A11 | Native owner screens for the ~20 unwired owner IPC methods (lineup, bids, trades, waivers, standings, results) | each method reached from a screen; live click-through | new 7b (before any native league; ideally after phase 10 catalog, plain version acceptable) | 2–3 |
| A12 | Cross-platform: per-OS instance lock (flock on unix, LockFileEx on Windows); Windows + macOS builds; GitHub Actions CI (lint, test, 3 builds) | CI green on 3 OSes; Windows binary launches | new 7c | 1.5–2 |
| A13 | Install/update path for other owners (decide: signed installers + manual update vs self-update) | decision recorded; panel if durable | 7c decision | — |
| A14 | Kicker valuation: confirm/implement fallback to the simpler rule where the fit loses (K) | test: K value equals last-season rule when model is not kept | phase 5 side check | 0.25 |
| A15 | UX walk: full click-through of every screen on an unused VM, findings triaged into fixes | report with screenshots per screen | new UX phase after 7 | 0.5 + fixes |
| A16 | Decision for Christopher (N3): native leagues this season commissioner-run on one machine, or bring client/server forward | ruling recorded | ruling 8 | — |
