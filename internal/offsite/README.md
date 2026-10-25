# Hosted off-site recovery

`Capture` → DB writer barrier (`Pools.CaptureHosted`) → database snapshot and
immutable config → snapshot audit checkpoint → manifest/digests → gzip/tar + age
public recipient → one encrypted R2 object → full GET/hash verification → manager
coverage → post-verification retention. Auth, ledger, evidence and audit share
the single hosted database. No desktop three-file backup behavior is changed.

`Manager` owns synchronized live coverage and scheduling. `Status` is the private
health/dashboard seam; `CheckWrite` is injected into DB transaction admission and
the HTTP explanation gate. Restart cannot deserialize a trusted local receipt.
`Pools.BeginWrite` reserves the writer, then rechecks coverage; `ExecWrite` keeps
single-statement writes on the same path. All hosted history/account stores use
these paths, including background due work; raw `Write()` remains the existing
startup/desktop DB API, not an alternate hosted mutation route.

`Restore` belongs on the recovery side. It accepts an identity in memory,
validates bounded Raw manifest/member input before path use, decrypts/verifies the
entire archive, and leaves only a complete new private staging directory. A key
that matches no recipient is `ErrRestoreKey`, distinct from damage. It does not
activate a server or reset credentials: `R2.Sets` (newest first) and `R2.Open`
feed `twr-server restore`, which compares the checkpoint, resets sign-ins in the
staged copy and hands it to the CT script ([RESTORE.md](../../deploy/server/RESTORE.md)).
`internal/serverstatus` reads `Manager.Status` (coverage, newest failure) for the
console dashboard over an owner-only socket, never a public route.

Failure routes: any capture/audit/crypto/PUT/GET/hash error preserves previous
coverage; coverage expiry rejects new queued writes without mutation. Pruning
failure is reported but does not invalidate the already verified newest set.
Corrupt restore deletes staged output, never touches active files. Backup retries
have a deadline; archive sizes, file counts and retained sets are bounded.
Configuration must be immutable while the server runs; deployment binds it
read-only, and setup stops the service before replacement.

See [the commissioner setup and limits](../../deploy/server/BACKUPS.md).
