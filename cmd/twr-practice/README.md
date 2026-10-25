# Throwaway hosted league for desktop click tests

Development only. Neither this command nor its fixture package is linked into desktop/server
release binaries or included in installer bundles. Run from the repository root on Linux.
Requires the normal desktop build dependencies (Go, WebKitGTK 4.1), OpenSSL, and built frontend
assets. On a fresh worktree: `cd frontend && pnpm install && pnpm build`, then return to the root.

```sh
go run ./cmd/twr-practice up .scratch/practice
# .scratch must already exist; any existing practice directory must be empty.
go run ./cmd/twr-practice code .scratch/practice
go run ./cmd/twr-practice down .scratch/practice
```

`up` prints the random loopback HTTPS address, commissioner username, random password,
authenticator secret, current-code command, and an exact desktop launch command. Copy that
command as printed. It inherits `DISPLAY`, including an Xvfb display, and uses its own HOME,
XDG configuration/data/cache/runtime folders and CA. Its deliberately unavailable session bus
prevents reading/writing the real OS keychain; sign-in stays in memory for this window only.
No real app profile or league archive is opened. Do not remove the printed environment settings.
Close the desktop window before `down`.

Sign in as the printed commissioner, use **Authenticate** with the code from `code`, then
enter the league. The commissioner also owns franchise.01; franchise.02 is vacant. The
league has two franchises and no imported MFL data. The confirmation uses the preceding time
step, so the current code is immediately usable. If a code has already been used, wait for the
next 30-second authenticator step before using `code` again.

TLS and authenticator keys are disposable. TLS expires after one day: stop and recreate the
sandbox then. Keep printed credentials and the sandbox directory private; they are not real
accounts. `down` checks the recorded PID's executable before signaling, waits for shutdown, and
then deletes the directory. Missing/invalid ownership records or a PID belonging to a different
executable fail closed and leave the directory untouched. An already stopped server can still
be cleaned up. A startup failure stops its child and deletes only the claimed empty directory.
Never point these commands at a directory you want to keep.

## Implementation and proof

Flow: claim an empty owner-only directory → copy preview registry/build binaries/generate
loopback TLS → initialize one database through trusted account/history APIs → create canonical
league/owner seat/enroll TOTP → launch the real `twr-server hosted` → record PID and credentials.
No network bootstrap or authorization override is added. The shared league creation helper in
`internal/hostedfixture` is also used by the hosted transport tests. The hosted server retains
its normal admission and clock-health checks; this tool does not disable them.

`go test -race ./cmd/twr-practice ./internal/serverhttp` exercises the real up → HTTPS password
sign-in → TOTP step-up → enter league → down lifecycle, verifies nonempty-directory refusal
preserves the running league, and rejects unowned directories, symlinks and unrelated PIDs.
An RFC 6238 vector independently checks the disposable code generator. The lifecycle test builds
the desktop but does not launch a GUI; visual acceptance still requires the printed command.
