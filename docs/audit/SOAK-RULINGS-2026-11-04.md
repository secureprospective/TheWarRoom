# Soak review — Christopher's rulings (2026-11-04)

Evidence: `SOAK-COVERAGE-2026-11-04.md` (run-0952, 46 loops, 33.4% of statements ran) and `SOAK-FINDINGS-2026-11-04.md` (F1–F10).
Direction: mocks are law; slim, snappy, well engineered; one path per thing; when in doubt, stop and reassess.

## Product frame (ruled)
- **#1 goal: capture MFL leagues.** The app's main purpose is migrating an MFL league into a native TWR league.
- **Secondary: solo owner on an unmigrated MFL league.** Allowed to lag and be a bit clunky. Kept thin.

## Batch 1 — Lab and Research
| Package | Lines | Soak | Ruling |
|---|---:|---|---|
| labcompile | 4,659 | not linked | **Move out** to its own repo on the Beelink (e.g. `~/work/LeagueLab`). Not deleted. |
| lab | 3,282 | 11% ran | **Out of TheWarRoom for good**, into the same Lab repo. |
| facts | 3,640 | 0% ran | Goes with Lab. First list what archive, report, sources and comms read from it. |
| app/research | 2,765 | 55% ran | Find the calling screen. Keep only what a mock needs (wired once); the rest leaves, data kept. |

Lab was added mid-session weeks ago without his approval, and several agents failed to scrub it. Its purpose: understanding owner behaviour and league history.
It will take in other leagues' data, so it's built **multi-league from the start**.

## Batch 2 — MFL reference
- **Migration path: keep it and make it excellent.** `app/mflref/migration` (73% ran), `app/mflref/seed` (0%, needs its own import test), `mflref/lineup` (73%).
- **Solo-owner path: keep it, thin.** planwatch, store/mflref/moves, bids, contracts (3,343), trades (1,398), envelope (1,928), bid, whatif, tradeplans (~8.7k).
  Rule: no second copy of native rules. Feed MFL data into native engines. Test: list each re-implemented rule against its native function; that list is the cut list.
  It never writes to MFL; confirm that with evidence first.

## Batch 3 — native core
- **Kind A (event code: draft, trades, votes, imports, previews) — keep.** commissioner, transactions, governance, store/history import and transfer.
  It didn't fire in week 9. It needs event tests, not cuts.
- **Kind B (readers that sat unused while the same data was on screen) — trace, then keep one reader each.**
  app/season points/playoffs readers (0%), app/now reading/recap/nfl/live (0–16%), app/offers bid/pool APIs (~1%).
  Method: for every element on Home, Pulse and War Room Market, record which call fed it in the soak and which reader sat beside it unused. Pick one; cut the other.

## Correctness findings carried forward (not yet ruled on order)
F3 Home vs HQ record disagree (2-0/week 8 vs 7-1/week 9) · F8 startup fails on schema migration (integrity check ×2 under 2-min deadline) ·
F4/F9 click queueing at start and mid-run · F10 memory 2.1→3.5 GB · F6 per-launch nflverse downloads · F1 HQ second shell · F2 Trade Floor stub.
