# Throwaway hosted league for desktop click tests

Development only. Neither this command nor its fixture package is linked into desktop/server
release binaries or included in installer bundles. Linux only.

## Build on the machine you are testing

Run from the repository root. Requires the normal desktop build dependencies (Go, WebKitGTK 4.1), OpenSSL, and built frontend
assets. On a fresh worktree: `cd frontend && pnpm install && pnpm build`, then return to the root.

```sh
go run ./cmd/twr-practice up .scratch/practice
# .scratch must already exist; any existing practice directory must be empty.
go run ./cmd/twr-practice code .scratch/practice
go run ./cmd/twr-practice down .scratch/practice
```

## Build once, copy to a test machine without Go

On a Linux build machine with Go, pnpm and the normal WebKitGTK 4.1 development
libraries, run this one-liner from the repository root:

```sh
(cd frontend && pnpm install && pnpm build) && mkdir -p .scratch/practice-bundle && go build -o .scratch/practice-bundle/twr-practice ./cmd/twr-practice && go build -tags practice -o .scratch/practice-bundle/twr-server ./cmd/twr-server && go build -tags desktop,production,webkit2_41 -o .scratch/practice-bundle/thewarroom . && cp docs/ui/endpoint-registry.csv .scratch/practice-bundle/registry.csv
```

The desktop embeds the freshly built frontend assets. Use the same Linux architecture
and compatible system libraries on both machines; this is not a cross-build or a static
bundle. Copy the **whole directory**, including `registry.csv`, preserving executable bits
(replace `tester@test-machine` with your SSH destination):

```sh
scp -r .scratch/practice-bundle tester@test-machine:~/practice-bundle
```

On the test machine, with OpenSSL and the desktop runtime libraries already installed:

```sh
mkdir -p "$HOME/practice-sandboxes"
"$HOME/practice-bundle/twr-practice" up -bin "$HOME/practice-bundle" "$HOME/practice-sandboxes/click-test"
# Launch the exact Desktop command printed by up, in the graphical session.
"$HOME/practice-bundle/twr-practice" code "$HOME/practice-sandboxes/click-test"
# Close the desktop window first.
"$HOME/practice-bundle/twr-practice" down "$HOME/practice-sandboxes/click-test"
```

No checkout, Go, pkg-config, development headers, pnpm or frontend files are needed on the
test machine. `-bin` copies `twr-server` and `thewarroom` into the sandbox, refusing missing,
non-regular or non-executable files, and copies the bundle's preview registry. OpenSSL still
creates fresh disposable TLS keys. Without `-bin`, `up` keeps its repository-based build
behavior. `code` and `down` never build anything. Keep the launcher in place until cleanup;
the printed code/stop commands use its absolute path when `-bin` is supplied.

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
loopback TLS (or copy bundle binaries/registry and generate TLS) → initialize one database through trusted account/history APIs → create canonical
league/owner seat/enroll TOTP → launch the real `twr-server hosted` → record PID and credentials.
No network bootstrap or authorization override is added. The shared league creation helper in
`internal/hostedfixture` is also used by the hosted transport tests. The hosted server retains
its normal admission and clock-health checks; this tool does not disable them.

`go test -race ./cmd/twr-practice ./internal/serverhttp` exercises the real up → HTTPS password
sign-in → TOTP step-up → enter league → down lifecycle, verifies nonempty-directory refusal
preserves the running league, and rejects unowned directories, symlinks and unrelated PIDs.
An RFC 6238 vector independently checks the disposable code generator. The lifecycle test builds
the desktop but does not launch a GUI; visual acceptance still requires the printed command.

`TestPrebuiltSandboxHTTPSLifecycleWithoutGo` builds the bundle first, then changes to an
unrelated empty working directory and restricts `PATH` to OpenSSL alone. It exercises real
`up -bin` → HTTPS sign-in → TOTP step-up → enter league → `code` → `down`, including the
nonempty-directory refusal and server shutdown. Go cannot be found during any of these
actions. `TestPrebuiltSandboxRefusesInvalidBinaries` checks both binary names for missing,
non-executable and directory inputs, and verifies cleanup leaves the input bundle untouched.
The desktop is copied but not launched: graphical click acceptance remains a separate test.

## Off-site backup boundary

The launcher builds its server with `-tags practice`, an explicit disposable-test
build, not the deployable binary. This variant refuses R2 configuration, validates
the private sandbox folders and colocated runtime files, and logs that it has no
off-site protection. The release builder never uses this tag. A normal server
binary remains paused without verified R2 coverage; there is no runtime switch
to disable that rule. Prebuilt practice bundles must use the tagged build shown
above. Never use the practice binary for real league state.

### Update-required screen

`up` also prints an **Update-required click test** command. With an unstamped development
binary, this sets `TWR_PRACTICE_UPDATE=app` in the isolated practice profile and opens the real
blocking desktop screen. It offers one button, **Update and reopen**. Without installer trust,
clicking it must explain that update setup is missing, preserve the screen, and permit retry;
no test signing keys or unsigned executable are installed. Release-stamped builds ignore the
practice override. For a valid verified-executable launch use the generated-key staging test
in `internal/releasetrust`; actual signed release/three-OS click drills require the install
handoff in `docs/delivery/Release_Trust.md`.
