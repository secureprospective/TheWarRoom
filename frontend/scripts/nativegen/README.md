# Native client commands

From the repo root:

```sh
go run ./frontend/scripts/nativegen
go run ./frontend/scripts/nativegen -check
go test -race ./frontend/scripts/nativegen
```

The generator binds the executable registry through `composition.CommandRegistry()` and validates its native publication before
writing `frontend/src/app/commands/native.gen.ts`. Identical schema subtrees are emitted once as
shared literal constants; the command table references them. This retains every schema, version,
permission, mode and safeguard without repeating nested mutation envelopes. TypeScript derives
request types from those literal schemas, rather than keeping a second argument list. The Go freshness test compares the complete
output byte-for-byte and rejects an intentionally incomplete declaration. Existing Wails method signatures are preserved.

Flow: Go catalog → generated schema and plain-word phrases → `commands/native.ts`
typed requests → `nativeEnvelope` → hosted workflow transport. The envelope checks
shape, declared modes and hosted availability. It does not infer capability grants;
the server rechecks credentials, current grants, arguments, deadlines and revisions.
Existing commissioner forms, `ChangeBar`, Go previews and `HoldToFire` retain ownership
of reasons, confirmation and retry keys. The envelope never supplies these itself.

The unused grammar, capability-display client and injected transport were removed.
The live hosted envelope and its negative TypeScript cases remain tested alongside
the generator freshness/parity tests and hosted admission tests. The executable
cross-door migration is described in `docs/architecture/command-spine.md`.

## Local MFL reference plans

`go run ./frontend/scripts/nativegen -surface mfl` generates `commands/mfl.gen.ts`
from the executable registry's local MFL entries; add `-check` to refuse stale output. The same schema
interner preserves modes, roles, desktop-only availability and safeguards.
`commands/mfl.ts` uses the shared shape checker without importing the native
catalog. `offers/api.ts` supplies only the existing local Wails adapters. No
remote route, key handling or hosted capability grant is generated.

Browser command labels, aliases, role sets, effects, argument names and keyboard bindings
come from `catalog.BrowserCommands()`. Run `go run ./frontend/scripts/nativegen -surface browser`.
The generator test refuses stale output in CI. `browser-before.json` freezes the old 62 browser
entries; the role/door table checks local admission and direct command-bar exposure. Typed
browser handlers remain in `commands/registry.ts`; its exhaustive generated-ID map refuses
missing handlers and undeclared command IDs during TypeScript checking. Browser argument types now come from `arguments.gen.ts`, generated from registry-owned
`Schema.ClientType` references to the existing browser domain types and inline local shapes.
These local types do not become remote JSON authority.

## All remaining doors

```sh
go run ./frontend/scripts/nativegen -surface doors
go run ./frontend/scripts/nativegen -surface doors -check
```

This generates the Wails object surface (`internal/composition/wails_surface.gen.go`),
its method manifest (`testdata/bindings/manifest.txt`), named CLI mutation verbs
(`cmd/twr/verbs.gen.go`), direct command-bar candidates (`commandbar.gen.ts`) and
unified argument maps (`arguments.gen.ts`). Wails arguments retain positional tuples;
native, MFL and workflow arguments retain their original JSON schemas. The existing
Wails generator owns `frontend/wailsjs`; run `make bindings` if API signatures change.

Every output has a byte-for-byte freshness test in the normal Go test suite/CI. The new
doors test also stales each file separately and proves check mode refuses it without writing.
The command-bar list excludes actions needing arguments and the four hidden plumbing actions;
dynamic endpoint/player/place search results are unchanged. CLI verbs preserve the JSON
preview/execute path and never grant capabilities or bypass confirmation.
