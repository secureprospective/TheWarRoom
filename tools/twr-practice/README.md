# Throwaway hosted league for HTTPS integration tests

Development only. Neither this command nor its fixture package is linked into release binaries.
Linux only.

## Build on the machine you are testing

Run from the repository root. Requires Go, OpenSSL, and built frontend assets. On a fresh
worktree: `cd frontend && pnpm install && pnpm build`, then return to the root.

```sh
go run ./tools/twr-practice up .scratch/practice
# .scratch must already exist; any existing practice directory must be empty.
go run ./tools/twr-practice code .scratch/practice
go run ./tools/twr-practice down .scratch/practice
```

## Build once, copy to a test machine without Go

On a Linux build machine with Go and pnpm, run this one-liner from the repository root:

```sh
(cd frontend && pnpm install && pnpm build) && mkdir -p .scratch/practice-bundle && go build -o .scratch/practice-bundle/twr-practice ./tools/twr-practice && go build -tags practice -o .scratch/practice-bundle/twr-server ./cmd/twr-server && cp docs/ui/endpoint-registry.csv .scratch/practice-bundle/registry.csv
```

Use the same Linux architecture on both machines. Copy the **whole directory**, including
`registry.csv`, preserving executable bits (replace `tester@test-machine` with your SSH destination):

```sh
scp -r .scratch/practice-bundle tester@test-machine:~/practice-bundle
```

On the test machine, with OpenSSL already installed:

```sh
mkdir -p "$HOME/practice-sandboxes"
"$HOME/practice-bundle/twr-practice" up -bin "$HOME/practice-bundle" "$HOME/practice-sandboxes/https-test"
# Use the printed loopback HTTPS address and disposable credentials.
"$HOME/practice-bundle/twr-practice" code "$HOME/practice-sandboxes/https-test"
"$HOME/practice-bundle/twr-practice" down "$HOME/practice-sandboxes/https-test"
```

No checkout, Go, pnpm or frontend files are needed on the test machine.
`-bin` copies `twr-server` into the sandbox, refusing missing,
non-regular or non-executable files, and copies the bundle's preview registry. OpenSSL still
creates fresh disposable TLS keys. Without `-bin`, `up` keeps its repository-based build
behavior. `code` and `down` never build anything. Keep the launcher in place until cleanup;
the printed code/stop commands use its absolute path when `-bin` is supplied.

`up` prints the random loopback HTTPS address, commissioner username, random password,
authenticator secret, current-code command, and cleanup command. `owner` and `partner`
use the same disposable password and need no authenticator step-up. The league has four
roster players, a free agent, a one-week schedule and a 2027 draft with one available rookie.
No MFL data is imported. If a code has already been used, wait for the next 30-second
step before requesting another.

TLS and authenticator keys are disposable. TLS expires after one day: stop and recreate the
sandbox then. Keep printed credentials and the sandbox directory private; they are not real
accounts. `down` checks the recorded PID's executable before signaling, waits for shutdown, and
then deletes the directory. Missing/invalid ownership records or a PID belonging to a different
executable fail closed and leave the directory untouched. An already stopped server can still
be cleaned up. A startup failure stops its child and deletes only the claimed empty directory.
Never point these commands at a directory you want to keep.

## Implementation and proof

Flow: claim an empty owner-only directory → copy preview registry/build binaries/generate
loopback TLS (or copy bundle binaries/registry and generate TLS) → initialize one database through trusted account/history APIs → create canonical
league/owner seat/enroll TOTP → launch the real `twr-server hosted` → record PID and credentials.
No network bootstrap or authorization override is added. The shared league creation helper in
`tools/hostedfixture` is also used by the hosted transport tests. The hosted server retains
its normal admission and clock-health checks; this tool does not disable them.

`go test -race ./tools/twr-practice ./internal/serverhttp` exercises the real up → HTTPS password
sign-in → TOTP step-up → enter league → down lifecycle, verifies nonempty-directory refusal
preserves the running league, and rejects unowned directories, symlinks and unrelated PIDs.
An RFC 6238 vector independently checks the disposable code generator.

`TestPrebuiltSandboxHTTPSLifecycleWithoutGo` builds the bundle first, then changes to an
unrelated empty working directory and restricts `PATH` to OpenSSL alone. It exercises real
`up -bin` → HTTPS sign-in → TOTP step-up → enter league → `code` → `down`, including the
nonempty-directory refusal and server shutdown. Go cannot be found during any of these
actions. `TestPrebuiltSandboxRefusesInvalidBinaries` checks both binary names for missing,
non-executable and directory inputs, and verifies cleanup leaves the input bundle untouched.

## Off-site backup boundary

The launcher builds its server with `-tags practice`, an explicit disposable-test
build, not the deployable binary. This variant refuses R2 configuration, validates
the private sandbox folders and colocated runtime files, and logs that it has no
off-site protection. The release builder never uses this tag. A normal server
binary remains paused without verified R2 coverage; there is no runtime switch
to disable that rule. Prebuilt practice bundles must use the tagged build shown
above. Never use the practice binary for real league state.
