# APP-ROSTER-3 — admission and effective projection

## Intended slice

Ordinary roster command → typed placement validation → transactional authority/time
check → existing decision preparation → event append. History owns real receipt time;
the command and event envelope retain effective time. Retry identity includes the
requested effective time. Omitted timing normalizes to `now`. Refusals append nothing.

Native asset reads → select effective entries at T → effective/revision ordering →
existing projectors over cloned seed. No lineup, importer or contract-policy changes.
Pre-seed reconstruction and unknown contracts remain outside this run.

## Step 1

- Explicit timestamps enabled for cuts, reserve moves, roster corrections, bid/waiver
  awards and trade outcome commands. Other command placement remains unchanged.
- Past commands require commissioner authority (including co-commissioner authority
  where the existing correction permission permits it); a trusted migration caller
  uses the same commissioner authority, never a source-specific exemption.
- Receipt time remains the store's real receipt clock. Both times remain durable;
  public ledger `What` includes explicit effective time and `When` remains receipt
  time, without changing the public wire shape or generated bindings.
- Unit tables cover every enabled command, omitted/now/equal-now/past/future timing;
  store integration observes durable times, public ledger and unchanged database
  fingerprints on refusal. Targeted admission race tests passed.

## Debt

Pre-observation reconstruction, unknown-acquisition contracts, lineup-at-lock checks,
transaction translation/idempotency, 32-team replay equality and the outstanding 76
lineup refusals are not addressed. No GUI or independent review performed.

## Step 2

- Both explicit-time native asset reads and `NativeAssetsAsOf` select inclusive
  effective time, then replay in effective/revision order. Untimed supporting
  records retain receipt timing; rules' own rollover scheduling is unchanged.
  Implicit current reads replay the known ledger; store current reads pass now.
- The source slice stays revision-ordered for compensation links and contract
  evidence. Only projection indices are sorted. The cloned seed and ledger bodies
  remain unchanged. Historical reads retain the full source revision.
- Market decision preparation now reads at the requested effective instant; the
  append still stamps real receipt time. A cut's waiver window starts at its
  effective instant. Final recovery projection validates the complete repaired
  timeline inside the append transaction, so dependent conflicts roll back.
- Existing bid/claim/offer dependencies must precede their outcome in effective
  history; no synthetic offers or claims are invented to repair invalid ordering.
- Historical compensation is computed from selected history, not today's active
  map. Backdated correction explanations retain their real receipt time. Closed
  native checkpoints are not reused across independently timed roster moves.
- Authority keeps each command's existing restrictions (notably commissioner-only
  trade overrides). Only owner roster moves gain commissioner correction authority;
  unchanged owner retries are looked up before a new backdating refusal.

New tests exercise a cut received after an award but effective before it, a trade
between weeks, both sides of each instant, inclusive boundaries, reserve revision
 ties, correction/compensation, source immutability, actual seated-owner authority
and preservation of commissioner-only trade override permissions.

Focused race batch: **119 test/subtest PASS, 0 FAIL**, three passing packages.
Order, cutoff and backdating-authority mutants each failed the intended behavioral
assertions and were restored. The initial cutoff mutant failed compilation; it was
corrected to a compiling mutant and then failed membership assertions.

### Existing regression debt verified on the starting production files

`TestSinglePoolStartupDraftIsUnchanged` (`internal/commissioner/draft_startup_test.go:315`)
expects `90ca21fd230f0aa6`, obtains `8a6936145d37309e`. The same race-enabled test with
all modified pre-existing production files restored to `0fd3b6c4` obtains exactly
the same digest. Its frozen expectation is unchanged. Do not describe the full
commissioner suite as green. The final batch and build results follow below.
