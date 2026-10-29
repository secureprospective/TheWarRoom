# Handoff

## Baton
Review — 2026-10-29, `session/p9-s2b`. Implementation committed; no push authorized.

## Where it stands
- `leaguemigration.Source.Read` returns native `Candidate`/`Plan`; raw read, Prepare
  translation and settings now belong to `leaguemigration/mflsource`.
- Native writer loses its boundary seam exemption. Candidate preserves held-ID owners
  and actionable problems, including invitation reads of currently malformed sources.
- Research reads held names/limits through `LeagueReference`; MFL config implements it.
- MFL debt 53 → 50: one reference port removes an import; two MFL fixture-builder
  pairs become a fenced fixture source (not erased imports). Only composition retains
  a runtime seam exemption. No composition/core fingerprints changed.
- No commands/features retired, shared NFL ingestion moved, network writes or pushes.

## Evidence
- Final scoped race batches: 86 top-level PASS; 11 preparation fixtures and 11 stored
  roundtrips PASS. Malformed source, read/record rollback, fresh fake-source
  Prepare → Import → Review and corrected-source retry PASS.
- Real source adapter exercised with guarded fake HTTP transport: zero write requests.
- `make lint`, `make boundary` PASS (159 fingerprints); pinned pnpm 10.34.3 frozen
  install/build PASS. Existing large-chunk warning remains. Hooks run on commit.
- Worktree-only detailed debt reasons, logs and moves: `_x/REPORT.md`, `_x/MOVED.md`.

## Next move / blocked on
- Independent review; no GUI acceptance or full-suite claim.
- Remaining 50 pairs: typed held ports/frozen root-core wiring, MFL-only shared-source
  transport readers, and slice-3 rule/what-if/state policy. Transitive closure is not
  enforced yet. Further frozen adapter changes require explicit review, not rehashing.
- `_p9/`, `_x/`, `.lintcache/` are untracked scratch/cache, not deliverables.

## Tried and rejected
- Leaving raw MFL config/draft types on the native Source port: still couples writer
  to fenced packages. Adapter now returns the prepared native plan and problems.
- Separate tiny Plan-only file: failed bloat ratchet; native contract stays together
  in `export.go`. No baseline increase or boundary fingerprint refresh.
