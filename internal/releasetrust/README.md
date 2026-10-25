# Release trust

Public install bootstrap and persistent verification of signed release artifacts/configuration,
using upstream `github.com/theupdateframework/go-tuf/v2` v2.4.2. No signing implementation,
league access, installer, launch logic or paid service lives here.

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
with this same consumer. Staged launch, interrupted installation recovery and three-OS proof
remain slices 44–45, not acceptance claims of this package.

Evidence: `go test -race ./internal/releasetrust/...` exercises generated throwaway keys,
metadata/artifact/config tampering, all four expiries, three release-role rollbacks across
restart, both sides of threshold rotation and public-only bootstrap. These are in-memory
repository tests; actual hosted TLS/cache behavior and offline custody need the delivery drill.
