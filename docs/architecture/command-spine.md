# One command spine

Every action should have one named definition. A definition says, in ordinary words,
what the action does, what information it needs, who may request it, whether it changes
anything, and which handler runs it. Each entrance uses that definition; none grants
permission merely because an action appears in a list. Existing account, league and
franchise checks remain authoritative. MFL remains reference data and local planning.

## Registry and execution

Extend the existing Go command catalog rather than inventing another protocol. Keep
stable native command IDs, argument schemas, permission policies, modes and availability.
An executable entry pairs that metadata with a typed handler and explicit door exposure.
Read handlers return checked JSON; mutations use the existing preview/execute service.
The registry rejects duplicate IDs and absent handlers. Do not use reflection to invoke
unvalidated input or replace domain authorization with display roles.

The intended flow is:

`door → Raw input validation → registry lookup → existing admission and handler → result`

Unknown actions, wrong modes and unavailable doors fail closed. A UI receipt is written
only when the returned operation settles, with done or failed and a reason. UI-local
operations (navigation, filters, overlays) have browser handlers; they must not acquire
server authority or become remote mutations as a side effect of this migration.

## Entrances

| Entrance | Derivation and preserved boundary |
| --- | --- |
| Hosted workflow | Derive its allowlist and dispatch from executable entries. Preserve exactly 13 reads and 29 mutations, including execute-only draft picks. |
| Generic HTTP and CLI | Look up stable IDs and modes in the same registry before calling existing service admission. Generate discoverable CLI verbs from exposed entries; retain existing JSON verbs. |
| Desktop/Wails | Generate typed adapters and bindings from registry-owned method exposure. Retain existing signatures during migration; bindings manifest rejects unregistered methods. |
| TypeScript | Generate command metadata, IDs and argument types from Go; retain typed browser handlers and explicit UI-local exposure. Nine local MFL planning operations keep their `mfl.*` names and current import behavior. |
| Command bar/chat | Search generated labels, arguments and exposure, invoke the same dispatcher and preserve confirmation requirements. |

Desktop-only operations stay desktop-only. A new entry appears only in its declared
doors, not automatically in privileged or unsupported transports.

## Drift gate and proof

Use one table-driven parity test against a frozen pre-migration fixture: every old ID,
role/permission, mode and door must survive unchanged. The hosted set remains 42.
Generated-file freshness checks run in the existing verify pipeline. Also reject a door
with no entry, duplicate IDs, missing executable handlers and unsupported modes. Exercise
an intentionally unregistered action to establish that the gate fails for the right reason.
Schema validation does not replace Raw validation or current credential admission.

## Migration order

1. Commit this design before changing execution.
2. Move hosted read handlers and the 42-command workflow exposure into executable
   entries; test parity and refusal without changing service behavior.
3. Make asynchronous UI handlers return their work; record real outcomes and show/log
   missing mounted handlers. Preserve synchronous navigation and keyboard behavior.
4. Remove unused native grammar/client and read-preview publication after confirming
   no production callers; retain the live native envelope and its validation tests.
5. Consolidate the remaining native, desktop, MFL and UI inventories; generate all
   door metadata and replace redundant lists/generators only after parity is established.
6. Run the full gate once, push once, inspect CI and report any unfinished migration
   explicitly. Do not describe an intermediate hosted registry as the completed spine.

Delete obsolete catalogs and tests only when their unique behavior is covered by the
registry parity/dispatch tests. Endpoint documentation remains separate navigation
information until its command links can be generated without changing destinations.
