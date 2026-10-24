# Native client commands

From the repo root:

```sh
go run ./frontend/scripts/nativegen
go run ./frontend/scripts/nativegen -check
go test -race ./frontend/scripts/nativegen
```

The generator validates the entire frozen `commissioner.NativeCommands()` publication before
writing `frontend/src/app/commands/native.gen.ts`. Identical schema subtrees are emitted once as
shared literal constants; the command table references them. This retains every schema, version,
permission, mode and safeguard without repeating nested mutation envelopes. TypeScript derives
request types from those literal schemas, rather than keeping a second argument list. The Go freshness test compares the complete
output byte-for-byte and rejects an intentionally incomplete declaration. No Wails bindings change.

Flow: Go catalog → generated schema and plain-word phrases → `commands/native.ts` typed requests,
capability display rows and named-argument grammar → injected native transport. Invalid shapes,
missing/denied/duplicate capabilities, unsupported modes and desktop-only imports fail before the
transport runs. The grammar only describes choices; it never executes a string or skips a form.

`nativeGrammar` supplies labels, named required/optional arguments, enabled flags and plain refusal
text to client command pickers. `createNativeClient` needs a fresh, argument-scoped capability
answer for each invocation. Catalog permission strings and local GM/admin display roles never
become grants. These client checks are not authorization: the server must recheck credentials,
current grants, arguments, deadlines and revisions, including inside append.

This is the client seam, not hosted route publication. No capability endpoint or hosted workflow is
invented in this slice. A future desktop native adapter supplies the transport and current scoped
answers; do not feed it credentials from the webview. Keep this module lazy with native workflows,
not in the shell entry bundle. Existing commissioner forms, `ChangeBar`, Go previews and
`HoldToFire` still own reasons, confirmation and retry keys, without changes here. The native client
never supplies a reason, chooses a stronger mode or infers confirmation from static gravity.

Evidence: `TestNativeClientCatalogParity`, `TestStaleGenerationRefusedWithoutWriting`,
`commands/native.test.ts` (including tsc negative type cases), and the existing safeguard/hold/proxy
tests. Actual hosted capability admission and HTTPS mutation execution belong to later slices.
