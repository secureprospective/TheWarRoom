# Frozen MFL reference boundary

MFL import remains supported. What-if sessions, local bid/trade/roster/lineup plans
and their watchers remain visible and unchanged. No command submits a move to MFL.

## Owners and allowlists

- `boundary-mfl.tsv` lists exact reference packages, the composition seam, native
  closure roots and remaining direct import pairs. Each pair has its own reason.
  This is not a claim that mixed application adapters have native-only closure.
- `boundary-mfl-commands.tsv` freezes the 11 qualified local operations, their
  contracts/trades desktop facades, platform compatibility facade, and six migration
  operations. Generic platform readings are included to freeze that facade as a
  whole. Existing metadata such as `unset` and `both` is compatibility inventory,
  not permission to publish a remote command. Sending invitations is not exercised
  or authorized by this fence.
- Reference envelope/check/predicate and lineup policy belongs under
  `internal/mflref`; local SQL belongs under `internal/store/mflref`; bid-plan and
  held-seed adapters belong under `internal/app/mflref`. Go package names stay
  stable for generated model compatibility.

`make boundary` rejects new reference packages, unreviewed commands, new direct
imports and native production paths reaching reference packages transitively.
The existing synthetic command, package and forbidden-import tests prove rejection.
Tests can use reference fixtures; production native roots cannot launder a reference
import through composition or legacy debt. The import graph excludes test files.

## Persistence and compatibility

Only code ownership changed. Table names, JSON keys, IDs, migration versions and
ordering are unchanged. State still owns the shared whatif database migration,
including trade-plan DDL. Composition injects its initializer into the independent
trade-plan repository before every Save/List; no store imports another store.
Literal old session/plan JSON survives close/reopen. Old stored lineup/trade intent
names are normalized on reading without rewriting append-only rows; new plans must
use qualified names. These stored-data aliases have no expiry and are not command
aliases. No public command alias was added or removed.

## Making an MFL fix

1. Fix the reference owner, preserve import and local-plan behavior, add a targeted
   regression test (including old-data reopen for persistence changes).
2. Run targeted race tests and `make boundary`. A new command/package/import pair
   needs explicit ownership review and a reason before updating its exact allowlist;
   do not add wildcard seams. Never refresh a composition fingerprint to hide policy.
3. Regenerate bindings if needed, verify freshness and frontend build, then commit
   with hooks. Keep MFL transport read-only and never invoke live writes.

## Honest remaining scope

Direct exceptions are 47 → 44: seed translation and lineup ownership remove three;
reference fixture classification removes eight; explicit mixed-facade ports exposed
by the new envelope/store/bid fences add eight. Source transport/normalization,
rulebook mirror storage, standings services and mixed source-reading application
adapters remain named debt. Shared NFL sources stay shared. Shared bid arithmetic
stays in the engine because transaction pricing uses its minimum-salary function;
native policy remains in nativecheck, not reference plan checks. Shared state ledger,
raw session records and migrations were not duplicated or moved into competing stores.
