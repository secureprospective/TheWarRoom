# Update the league server first

Use the CT console. These steps never run on or change the Proxmox host.
Keep desktop apps closed until the server is ready. Do not run the installer again.

1. Obtain the signed server bundle and its TUF metadata folder from your release provider.
   Keep the archive intact; do not unpack it or run any program from it. Keep the previous bundle.
2. Run the **installed** upgrade tool, giving the archive and metadata folder:
   `sh /usr/local/libexec/twr-upgrade /root/twr-new/server.tar.gz /root/twr-new/metadata`.
   Use the bundle's original target basename (not its hash-prefixed download name).
   Do not replace the installed binary or run the new bundle's scripts by hand.
3. Wait for **Server upgrade ready**. Update desktop apps next, then open the league.
   Confirm that its latest ledger entry is still present. Changes stay paused until
   the first fresh off-site backup uploads and verifies.

The installed server first authenticates the archive using the public root pinned at
`/etc/twr-release/root.json`, fresh signed metadata and the persistent private trust cache
at `/var/lib/twr-release`. It rejects damaged bytes, expired or rolled-back metadata and
releases older than the installed server. Never delete the cache to retry a refusal.
No pinned root means **reinstall from a verified bundle**; a downloaded root is not a repair.
Verification refusal leaves services and installed league data untouched. Ask your release
provider for a fresh signed bundle and metadata. Run only one console upgrade at a time.
If an interrupted upgrade leaves `upgrade.lock` in the cache, confirm no upgrade or verifier
is running before removing only that lock folder; retain all trusted metadata.

After verification the tool freezes and rechecks the archive's fingerprint, extracts only
the server executable, and runs its read-only league check as `twr` inside the same
bubblewrap sandbox as the service. Candidate code never runs as root.
The tool first reads the league file without migrating it. An unsupported newer file or
newer server stamp is refused without changing it. It then stops the Tunnel and server,
keeps the installed binary and private settings, and takes a SQLite snapshot of the
whole league, accounts, audit chains, and transfer fences. It verifies and saves their
checkpoint. If R2 is configured, a matched encrypted recovery set must upload and read
back successfully before migration. With no R2, the server remains practice/read-only
for native changes as before; local migration still needs its verified snapshot.
Migration and the new release stamp happen only after backup. The tool checks the
resulting schema, both audit chains and their saved checkpoint before replacing the
installed binary. TLS application readiness must pass before reopening the Tunnel.

## If it stops

Leave the Tunnel stopped. Read the single next action in the refusal. A failed migration
or data readiness check puts the pre-upgrade database back. The tool deliberately does
not restart on failure: if putting the copy back failed, starting would be unsafe.
Keep the saved folder printed by the tool; it holds private keys, so never share it.
After a reported successful put-back, start the previous server from the CT console
with `rc-service twr-server start`, then `rc-service twr-tunnel start` once it is ready.
For a damaged file, use RESTORE.md instead.

If startup fails after migration, keep the new binary and data together and repair the
reported startup problem. Do not put an older binary over the upgraded file: release
stamps deliberately refuse that even when the schema did not change.

A deliberate rollback requires a matched set: stop the Tunnel and server, save the
current full set first, then restore **both** the previous binary and its database from
`league/server.db` in the saved folder, together with that folder's settings. Remove only
the stopped database's `server.db-wal` and `server.db-shm` before replacing it; keep file
owners and private modes. Start and verify the old server before reopening the Tunnel.
This loses changes made after the saved snapshot; the commissioner must choose that
loss explicitly. Off-site disaster recovery uses RESTORE.md, not a binary downgrade.

## Compatibility contract and implementation flow

`internal/serverversion` owns the append-only release rows, protocol major, current and
previous minor window, security floor and supported durable command versions. A security
floor may retire the previous minor immediately. Headerless pre-version clients are
protocol 1.0 / command 1, not an unlimited bypass. Unsupported versions receive HTTP 426,
`X-TWR-Update: app|server`, and a validated plain-English response. Every hosted route is
gated before accounts, native data, backups or SSE. No SSE connection is needed to learn
that an update is required. Desktop/agent Go transports attach versions to every request;
`serverclient.ErrUpdateRequired` and `*serverclient.UpdateRequired` are slice 44's signal.

File flow: installed `verify-release` → frozen archive fingerprint check → sandboxed
`serverversion.Preflight` → `db.Inspect` (read-only; no migration) → stopped-service
snapshot and verified off-site set → history/accounts migration → monotonic release stamp
→ schema, audit and checkpoint verification → binary replacement → TLS readiness.
Migration failure restores the snapshot, not an attempted reverse SQL migration. The
service's startup preflight refuses older files requiring this tool, newer schemas, and
newer release stamps before opening a database writer. Deployment owns service exclusion:
do not run a mutating `twr-server upgrade` while the server is running. `upgrade -check`
is read-only and may be used while serving. The upgrade does not change configuration.
