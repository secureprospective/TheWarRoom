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

## Current implementation

Browser metadata now comes from `catalog.BrowserCommands()` and generates
`browser.gen.ts`; typed browser handlers must exhaust its generated IDs. The frozen
62-command browser fixture exercises all three display roles and both local admission
and direct command-bar exposure (372 assertions). Removing a handler or naming an
undeclared action fails TypeScript checking. Nine MFL transport IDs use `mfl.*`;
saved bid receipts still use their original intent. Hosted workflow exposure remains 42.

Native (136), local MFL (9), desktop (136), hosted workflow (42), and browser (62)
metadata now feed `catalog.Registry` in `internal/commandcatalog`.
Relocation: `internal/app/catalog` → `internal/commandcatalog`; transport-neutral
metadata and execution must sit below both the app and hosted transports. Native and MFL definitions and desktop exposure
live beside the browser declarations in `internal/commandcatalog`; feature `Commands`,
`NativeCommands` and `MFLCommands` functions are compatibility views, not separate lists.
All JSON shapes use `catalog.Schema`. Raw-derived schemas stay tied to their existing
request types; shape metadata never replaces their validators.

`internal/composition/command_bindings.go` injects typed method values for the 136
existing desktop signatures. Generic binders decode positional arguments and call those
values directly, without reflective invocation. Native handlers use the existing shared
service for preview/execute, preserving the separate lineup/pick gates and held-seed
adapter. Browser handlers remain typed TypeScript functions, not server authority.

| Boundary | Current flow / failure |
| --- | --- |
| Desktop startup | Build APIs → bind registry → retain the unchanged Wails API objects. Missing/extra methods or absent handlers refuse startup. |
| Native/MFL registry calls | Look up ID, declared door and mode → checked argument adapter → existing Raw validation and service/API admission → checked JSON result. |
| Hosted workflow | The existing HTTP envelope aliases catalog-owned Raw types; its compatibility table and dispatch derive from catalog-owned executable workflows. The 42-command boundary is unchanged. |
| Evidence | `doors-before.json` freezes 281 native/MFL/Wails records, including schema hashes. Composition tests compare both compatibility views and executable entries; existing workflow/browser fixtures retain their own coverage. |

`frontend/scripts/nativegen -surface doors` inspects the same bound registry without
starting the app. It generates the Wails object surface and 136-method manifest,
99 named CLI mutation verbs, the 12 direct command-bar candidates, and unified
TypeScript argument maps for native, MFL, browser, Wails and workflow doors.
Wails still generates its JavaScript adapters and model declarations from the original
API signatures; the registry-derived manifest rejects missing or extra reflected methods.
Existing JSON CLI verbs and credential/confirmation admission remain unchanged. Named
verbs only wrap the existing generic RawCommand transport; reads, held seed and the
special draft-pick transport do not acquire new CLI routes.

Browser argument schemas name existing TypeScript domain types (or inline local shapes)
through `Schema.ClientType`. These browser-only references are not remote input schemas;
Go Raw validators still own every remote boundary. The generated browser argument map
now types handlers and dispatch, replacing their handwritten command argument annotations.
Search uses the generated direct-candidate list, while places, players and other dynamic
arguments retain their current sources. The four hidden bar-plumbing actions remain hidden.

The generator tests compare every output byte-for-byte and intentionally stale each new
file, checking refusal without a write. Existing native/MFL/browser freshness tests and
`make bindings-fresh` cover the remaining generated TypeScript and Wails JavaScript/models.
The frozen parity matrix in `internal/composition/command_matrix_test.go` covers all
299 executable entries × seven doors × three display roles (6,279 cells), including
absent exposure. It combines the original native/MFL/Wails, browser and workflow
fixtures, preserves mode and permission metadata, and intentionally rejects an
unregistered door and a missing declared door. CLI exposure retains the original
generic preview/execute boundary; its 99 named verbs are new aliases, not new authority.
Domain-role cells preserve the domain policy, not a grant to any display role;
existing credential/league/franchise admission tests remain authoritative.

Mounted handlers return their work through dispatch. Ledger filtering shares the
command's read with its rendering effect; account, helper, update and draft-pick
operations wait for admission/results. Market, offer-history and pool adapters turn
stored failures into failed receipts; opening a bid form awaits its first preview.
Only settled operations enter the bounded history, and failures retain a readable
reason, notice and log. Navigation and ordinary local view changes remain synchronous;
unrequested refresh/polling work does not manufacture command receipts.

The test-only hosted workflow compatibility inventory and its duplicate parity helper
were removed (no callers); production workflow validation/dispatch already uses the
catalog directly. The former native grammar/client and read-preview publication were
already removed before this branch; source searches remain empty. The unused Draft
command adapter was also removed after market dispatch moved to its state owner.

Compatibility views live beside their feature APIs; local MFL and Wails declarations
share `internal/commandcatalog/desktop.go`. The Wails generator emits both the typed
API holder and the binding surface, so their membership cannot drift. Generator
registry construction stays with composition's command bindings; named CLI dispatch
stays with its root command. These consolidations preserve the tiny-file ratchet
without raising its baseline.
