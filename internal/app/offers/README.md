# Offers and local MFL plans

`MFLCommands()` is the completed, desktop-only catalog for the free-agent player
pool and bid plans. `Commands()` also retains the other desktop binding inventory;
unset declarations are not publishable. `bid.submit` retains its existing command
ID, but **saves a local plan**, never submits a bid.

Flow: pool form or command action → `offers/api.ts` → catalog-generated
`commands/mfl.ts` preflight → existing Wails adapter → Raw validation → serialized
local moves store. Reads/preview do not save. Hand-off rechecks current bid rules;
posted records the owner's statement, not a detected external submission.
Withdraw removes only the local plan, never a ProBoards post or MFL transaction.
The existing explicit plan buttons supply owner-action intent for the three G2
operations. Preview, a role and successful clipboard access are not confirmation.

The solo owner's desktop custody is the permission boundary; local role metadata
is not a hosted grant. There is no HTTP fallback, agent bearer or MFL-key argument.
Hosted CLI/MCP tokens do not authorize these commands: they remain desktop-only,
with the same catalog availability and safeguards regardless of caller. There is
no local CLI listener in this slice. Do not route local IDs to native dispatch.
The pool refresh uses the existing read-only MFL freeAgents export; plans only
use held reference data and local persistence. MFL credentials stay in Control's
existing desktop OS-keyring operations, never in the league server.

Generate with `go run ./frontend/scripts/nativegen -surface mfl`. The generator's
race tests check freshness against the Go catalog. Fixture tests in `catalog_test.go`
observe names, no-write previews and rejection with unchanged saved plans;
`commands/mfl.test.ts` observes fail-closed modes/roles/availability/owner intent.
Existing bid lifecycle tests cover legal transitions and current-rule rechecks.

This is a player pool, not pick'em or survivor side games. Those remain deferred.
Slice 49 owns the wider MFL-reference versus hosted-league labeling pass.
