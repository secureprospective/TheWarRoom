# Handoff

## Baton
Bee → ClaudeBox / review — 2026-11-01, build/ROSTER-1.
APP-ROSTER-2 incomplete. Local commits only; no remote contact.

## Where it stands
- `f1249898`: new bounded `NativeAssetsAsOf` receipt-history API and tests.
  Existing admission/reading/lineup/import callers unchanged; API has no production caller.
  It rejects times before the seed observation; it is NOT effective-order projection.
- Fresh backup-API copy: 895 transactions validate; none replayed. Ordinary lineup
  sync records 148, refuses 76; repeat records zero, refuses 76. Bills weeks 1–7 recorded.
- Refused 76 → 76. Exact remaining reasons: docs/build-handoffs/APP-ROSTER-2-refusals.txt.
- Targeted race tests: 34 test/subtest PASS, zero FAIL; plain make build PASS;
  commit hooks PASS. No GUI/independent review. No frontend source changed.
- Evidence/limits: docs/build-handoffs/APP-ROSTER-2-evidence.md.
- Both disposable profile directories deleted; ignored text logs remain in .tmp/roster2/.

## Next move / blocked on
- No new ruling needed; the sanctioned generic extension remains unfinished.
- Implement effective ≤ now on ordinary roster commands, commissioner authority for
  backdating, real receipt time and both times visible in the ledger.
- Project effective order then revision, with backward-consistent reconstruction
  around the seed's observation. Unknown historical acquisition contracts use league
  defaults, flagged "contract from import, unverified" for commissioner review.
- Then lineup membership at kickoff, ordinary-command transaction translation,
  transaction-repeat-zero and 32-franchise end-of-replay equality proof.
- Current baseline differences remain documented in docs/build-handoffs/APP-ROSTER-1-baseline.tsv;
  replay equality is unproven. Earlier scoring debt remains in prior handoff/history.

## Tried and rejected
- Passing a historical time to existing NativeAssetsReading alone still does not
  filter roster history. The new receipt-history API is not a pre-seed substitute.
- No raw SQL, store-clock backdating, source-specific league branch, invented contract
  or lineup membership bypass was added.
