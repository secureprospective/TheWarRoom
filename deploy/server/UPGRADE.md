# Update the league server first

Use the CT console. These steps never run on or change the Proxmox host.
Keep desktop apps closed until the server is ready. Do not run the installer again.

1. Download and verify the new server bundle using the release instructions in README.md.
   Unpack it in a new folder inside the CT. Keep the previous bundle.
2. From that folder run `sh upgrade.sh /root/twr-new/twr-server`, replacing the path with
   the new binary's actual path. Do not replace the installed binary by hand.
3. Wait for **Server upgrade ready**. Update desktop apps next, then open the league.
   Confirm that its latest ledger entry is still present. Changes stay paused until
   the first fresh off-site backup uploads and verifies.

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

File flow: `serverversion.Preflight` → `db.Inspect` (read-only; no migration) → stopped-service
snapshot and verified off-site set → history/accounts migration → monotonic release stamp
→ schema, audit and checkpoint verification → binary replacement → TLS readiness.
Migration failure restores the snapshot, not an attempted reverse SQL migration. The
service's startup preflight refuses older files requiring this tool, newer schemas, and
newer release stamps before opening a database writer. Deployment owns service exclusion:
do not run a mutating `twr-server upgrade` while the server is running. `upgrade -check`
is read-only and may be used while serving. The upgrade does not change configuration.
