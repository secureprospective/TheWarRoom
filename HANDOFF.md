# Handoff

## Baton
Bee → ClaudeBox / review — 2026-11-01, build/ROSTER-1.
APP-ROSTER-1 incomplete: baseline/evidence only. Local commit; no remote contact.

## Where it stands
- No production code changed. Item 1 traced; history import and roster-at-lock not implemented.
- Ordinary copied-profile lineup backfill remains 148 accepted / 76 refused of 224;
  repeat records zero. Bills weeks 1–7 already record through the ordinary path.
- Archived transactions validate: 895 rows. Existing seed vs latest archived roster:
  28 franchises equal, 4 mismatched; not a history replay proof.
- Targeted race tests: 24 test/subtest PASS; plain make build PASS.
- Copy measurement race run timed out at 600 s; non-race resume completed.
- Evidence: docs/build-handoffs/APP-ROSTER-1-evidence.md and APP-ROSTER-1-baseline.tsv
  in the same directory. Private disposable copy/logs/harness: .tmp/roster1/.

## Next move / blocked on
- Review generic historical roster admission/projection: ordinary cuts/reserves/trades
  require effective=now; native asset reads ignore the historical instant for roster projection.
- Establish historical baseline and contract provenance for players dropped before seed;
  correct.roster refuses to invent contracts. Respect differing seed/feed observation times.
- Then implement generic as-of behavior, importer mapping, automatic sync and replay-equality proof.
- Earlier scoring debt and position queue deferral remain as described in prior handoff/history.

## Tried and rejected
- Passing historical time to NativeAssetsReading alone: still projects every active roster entry.
- Raw SQL, backdating receipt clock, fabricating contracts, bypassing lineup roster checks:
  would evade the ordinary ledger contract; none implemented.
