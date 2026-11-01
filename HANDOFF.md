# Handoff

## Baton
Review / ClaudeBox — 2026-11-01, build/SCHED-1. APP-SCHED-3 partially delivered;
local commits only, no git remote contact.

## Where it stands
- 4d824ce8: validated played starters map in importer, then ordinary commissioner LineupCorrect
  via single-command ledger append. No reported-score adapter, new event/storage or math changes.
  Empty slots omitted; unknown real mappings refused; unchanged sets idempotent; season fenced.
  Each lineup atomic; rejected franchise-weeks do not stop independent ones; all errors returned.
- 70235754: acquisition automatic on existing launch/manual/watched reference sync, using original
  recorded principal. Launch uses held player archive before a target snapshot exists; no players fetch.
- SQLite-backup copy `.tmp/sched3/profile`: launch reaches 152 entries (2 original + 2 schedule +
  148 lineups); repeat stays 152. 76 franchise-weeks refused by ordinary current-active-roster check.
- Existing ResultSources now reads recorded starters, but all totals remain incomplete. Bills app
  0–0 / PF 0 vs MFL 6–1 / 1912.35; 0 decided / 208 pending. Score parity FAILS.
- 4,667 unique real starting player-weeks diagnosed: 3,568 incomplete, 637 unavailable,
  462 unresolved scoring position. Missing play-by-play affects 4,660 (overlaps); 630 absent
  player-weeks; 0835 has no NFL identity for W1–7. No statistics fabricated or settings changed.
- Targeted race: 18 top-level / 68 named PASS, zero failures (deduplicated). Plain make build PASS;
  code commit hooks passed. No GUI/full-suite/independent review. Production 113→313 physical lines;
  tests 139→341 vs 396d10ca, whole affected Go files. No reduction claimed.
- Durable detail and exact missing IDs/weeks, comparison and first roster refusal per franchise-week:
  `docs/build-handoffs/APP-SCHED-3-evidence.md` and its three companion TSVs.
  Raw proof/results/race/build logs in `.tmp/sched3`; one-use harnesses removed.

## Next move / blocked on
- Decide how historical roster membership reaches the shared lineup append check. Current snapshot
  cannot prove past active membership (IR/taxi/trades/releases); do not bypass or fake it.
- Obtain missing actual NFL evidence/play-by-play and settle scoring positions through approved flows.
  Re-run comparison; currently exact=0 / within .01=0 / complete differences=0 / unavailable=224.
- Completely empty starting lineups are not representable by existing lineup command; no contract change.
- MFL top five reported by wins then PF, not verified MFL tie-break order. Numeric stat-by-stat parity
  cannot be established without complete app scores. Head-brain review and live click-through still due.

## Tried and rejected
- Batch lineup appends: store intentionally permits lineup commands only individually; use existing path.
- Skipping current-roster validation, rewriting seed to fit history, inventing missing stats or using MFL
  scores as final results: violates shared semantics. No scorer/result/standings override introduced.
