# Commissioner Suite — Build Sequence (DEFERRED)

**Status:** DEFERRED. Out of scope under `Core_Build_Plan_2026-10.md`: what is built is kept
working and nothing is extended until Christopher reopens the track.

**Provenance:** this is a reconstruction. The locked plan lived only on CT105 at
`/root/.claude/plans/commissioner-suite-build-sequence.md` (written 2026-07-27); that file and
the session that wrote it were removed in a later CT105 cleanup. Everything below comes from
records that survive:
- CT105 memory `project_commissioner_suite_build_sequence.md` (the sequence and how it was
  derived);
- the CT105 context log for 2026-07-27 and 2026-07-28;
- this repo: `PENDING_LIVE_GATE_CATALOG.md`, `Build_State_Archive_Through_Alpha.md` and the git
  history.

Nothing here was invented to fill a gap. Per-session detail beyond these records is lost.

## How the sequence was derived

Christopher supplied a screenshot of MFL's real commissioner panel for league 14432. Every
control on it was traced, link by link, to what this codebase actually has behind it (yes, no
or partial). The sequence orders the gaps. Each session was to start with a UI-mapping brief
before any backend code.

## The sequence

| # | Session | Status |
|---|---|---|
| 0 | Taxi/IR cap-math fix, plus a commissioner toggle and slot counts for IR and taxi (a mainstay in 32-team dynasty, not an edge case) | Merged `5800a05` |
| 1 | Activity / transaction feed, acquisition provenance, OQ-013 reconciliation | Merged `ae584c9`; live-gated PASS 2026-07-28 |
| 2 | Retroactive correction (append-only ledger) and roster, position, taxi and IR limit enforcement | Backend merged `4d80e1c`; enforcement live-gated PASS 2026-07-28. **The "Correct this entry" UI was never built**; the `CORRECT` backend is kept. |
| 3 | Trade-value / fairness signal: surface `AdjustedScore` and `PowerScore` in `TradeBuilder.tsx` | Not started |
| 4 | Cap timeline | Not started |
| 5 | DOT vote state | Not started |
| 6 | Trade block | Not started |
| 7 | League history viewer | Not started |
| 8 | MaxPF and pre-draft scouting | Not started |
| 9 | Year-end awards | Not started |

Sessions 3–6 share the trade-domain code. Sessions 5 and 6 were deliberately interleaved
between commissioner-facing and GM-facing work.

## When this reopens

- Session 3 depends on the core plan: its fairness signal should read the Stage 7 measurables,
  not today's `AdjustedScore`.
- The correction UI (Session 2's missing half) is the cheapest outstanding item: the backend
  and IPC exist.
- Each session gets a live gate by Christopher on his desktop before merging to main.
