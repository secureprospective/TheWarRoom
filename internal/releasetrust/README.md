# Release trust

Public install bootstrap and persistent verification of signed release artifacts/configuration,
using upstream `github.com/theupdateframework/go-tuf/v2` v2.4.2. No signing implementation,
league access, installer or paid service lives here.

| Entry | Contract / effects |
|---|---|
| `DecodeConfig` / `RawConfig.Validate` | Bounded public routing JSON; rejects unknown fields, credentials, private addresses and certificate pins. No writes. |
| `Bootstrap.Validate` | Independently handed public root; checks two-of-three root policy and self-signatures with go-tuf. Self-signing does not establish first-install authenticity. |
| `New` / `Client.Download` | System HTTPS trust, bounded downloads; refreshes root/timestamp/snapshot/targets and verifies payload bytes. Writes trusted metadata and verified targets only to the supplied private cache. Returns no file on failure. |

The desktop integration (slice 44) must keep one Client/cache under its single-instance lock,
retain the cache across upgrades/restarts, and never delete or reset it following an error.
Supply an application-owned private directory, not an uploaded/shared folder. Local cache files
are trusted state; replacing/deleting the entire cache defeats remembered rollback protection.
An existing metadata directory without root.json fails closed. The bootstrap is used only when
no metadata directory exists; subsequent calls use the persisted, possibly rotated root, not
the old installer root. Verified intermediate metadata may advance even if later refresh fails;
never undo that progress. Native league data is never opened by this package.

`Download` accepts safe basenames and caps each target at 512 MiB. After downloading
`client-config.json`, call `DecodeConfig` before using it. Do not apply unsigned routing supplied
by the update host. First-install routing and root come from the authenticated installer;
a public hostname alone does not authenticate a root. Upstream HTTP calls have a 30-second
request timeout and HTTPS-only redirects; no custom CA or leaf pin is bundled. System UTC time
and an uncompromised local cache are required. The API is blocking; run off the UI thread.

Maintainer tooling and the step-by-step offline ceremony: `docs/delivery/Release_Trust.md`.
The publisher already uses Python's maintained reference TUF library; this Go package is its
native consumer, not a second publisher. `scripts/release/verify.go` checks hosted publications
with this same consumer. Staging and retry recovery are described below; three-OS packaging proof remains
a separate delivery drill.

Evidence: `go test -race ./internal/releasetrust/...` exercises generated throwaway keys,
metadata/artifact/config tampering, all four expiries, three release-role rollbacks across
restart, both sides of threshold rotation and public-only bootstrap. These are in-memory
repository tests; actual hosted TLS/cache behavior and offline custody need the delivery drill.

## Desktop staging

`Client.Stage(installedVersion)` downloads `desktop-<os>-<arch>.json` as a TUF target,
then its executable target. The strict manifest contains only `version` (canonical `vX.Y.Z`),
`target` (safe basename) and `sha256` (lowercase hex of the executable). Publish the native
executable directly, not a tar/zip archive. The extra digest binds the manifest and executable
even if metadata is refreshed between downloads. The existing publisher accepts both as
`--artifact` arguments; no new signing system or dependency is introduced.

Flow: hosted refusal → `serverclient.Desktop.Act` remembers the required side → webview
blocking screen → explicit Update and reopen → fresh TUF verification → version admission →
verified private executable → atomic staged file and remembered application version → start
child with the same fixed league origin → quit old window. Server-too-old instead asks the
commissioner to update the server and offers a return to sign-in. No update request opens
league files, extracts paths, replaces the installed executable, or runs migrations.

The authenticated installer must provision `bootstrap-config.json` and `bootstrap-root.json`
in `os.UserConfigDir()/TheWarRoom/updates` (private owner directory). These are public but
trusted install material, never downloaded on trust failure. `composition.runNative` holds
its instance lock for the full window lifetime; a staged child waits up to ten seconds for
that lock before opening. A missing setup produces an actionable failure on the same screen.
Development stamps cannot install owner updates. `serverclient.ConfigureUpdater` owns launch;
its webview handler shares the existing origin/JSON/input validation boundary.

Recovery: partial temporary files are never launch candidates. Every button attempt refreshes
TUF and reconstructs the staged executable from verified bytes; even an existing staged file
is not trusted as launch authority. `desktop-version` remembers the highest accepted application
version across process restart; lower releases are rejected even when newer TUF metadata signs
them. Equal accepted versions can be retried, but a candidate must be newer than the running
build. Never reset this file or metadata after a failure. Keep the original installed app as
recovery entrypoint. Launch success means the OS started the child, not that a new GUI proved
healthy; automatic installed-shortcut replacement and three-OS GUI health checks remain the
packaging drill. Staged executables are retained, not automatically garbage-collected.

Evidence: `TestValidStagedDesktopLaunchAndInterruptedRecovery` runs the verified staged POSIX
executable after restart/reconstruction; `TestStageRejectsUnsafeUpdatesWithoutNativeWrites`
observes corruption, all metadata expiries, installed/remembered downgrades and malformed
manifests leaving a native-data sentinel unchanged and no launchable staged file.
`TestUpdateScreenBlocksNativeActionsAndRetriesSafely` checks the native gate and retry.
The frontend `updateRequired.test.tsx` checks the one action, safe retry, server-first wording,
and typed 426 routing. No live production trust root or three-OS GUI launch is claimed.
