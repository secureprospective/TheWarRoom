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
