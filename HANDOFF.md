# Handoff

## Baton
Bee — 2026-10-27, `session/revamp-E-edge-scoring`. E2b complete; do not push without authorization.

## Where it stands
E2a is committed at 77b2c7ea and a08c543a. E2b attaches validated Sleeper-owned quantities to a per-revision canonical MFL/week lookup after nflverse play attachment. Direct Sleeper→MFL works without GSIS; known Sleeper-only weeks are retained, and ordinary GSIS reads are unchanged. Unknown numeric nonzero rows are counted per season per revision; team and zero rows are excluded. No roster or identity inference.

The four `TestPinnedSleeper*` tests cover import → database reopen → canonical read → scoring, overlap preservation and revision absence. Mausi totals +1.407 unrounded; DJ Johnson +0.167. Legacy evidence tests remain. Targeted race tests and `go build ./...` passed; named-test PASS count 11 (the opt-in unknown-ID study test is skipped without its cache environment). Targeted golangci-lint passed. No full suite, make verify, or push.

## Next move
Review E2b, then E5–E6 under the slice plan. To run the unknown-ID study report, supply `TWR_STUDY_CACHE` and `TWR_SLEEPER_STUDY_CACHE` as documented in its test.

## Blocked on
Unknown-ID fantasy-roster share needs independent roster evidence; no new identity resolution authorized. E3 RTL-rate discrepancy (0.09368635 versus study 0.09386) remains unresolved. Branch has no upstream; `git pull --ff-only` was previously refused.

## Tried and rejected
Do not infer fantasy-roster membership or bye status from weekly statistics. Do not require GSIS for a direct Sleeper→MFL bridge, create fake GSIS identities, or inner-join away Sleeper-only weeks. Unknown IDs are not automatically irrelevant.

Debt left: unknown Sleeper identity/roster coverage, E5–E6, E3 rate reconciliation.
