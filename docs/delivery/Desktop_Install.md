# Desktop candidate install and proving drill

Targets pending owner OS/CPU inventory: Windows x64 (Windows 10/11 with WebView2),
macOS Apple silicon (arm64), Linux x64 with GTK 3 and WebKitGTK 4.1 (Ubuntu 24.04
baseline). Other CPUs, older OS versions and other Linux distributions are unproved.
CI uses Windows Server 2022, macOS 15 arm64 and Ubuntu 24.04 native runners; these
are build environments, not substitutes for clean owner machines.

## Trust and install

These are unsigned **candidate packages**, not approved owner releases. No paid
signing, Authenticode, Apple Developer account or notarization is used. CI uploads
ZIPs (Windows/macOS) and a tar.gz (Linux), plus SHA-256 sidecars. Sidecars catch
copy errors, not malicious replacement. The operator must approve exact package
bytes and pass them through `scripts/release/release.py` and the offline TUF
ceremony in `Release_Trust.md` before owner delivery. No production trust root is
embedded by this packaging slice; enrollment and updater acceptance are separate.
Never share a tunnel credential, private release key, MFL key or real database.

Use a disposable clean OS account/VM, without live War Room files. Download the
exact candidate via the authenticated operator handoff, retain its SHA-256 and
commit, and disconnect networking for this install/launch drill. Do not enroll,
configure MFL or import real league state. Stop if the publisher/hash is unexpected.

- **Windows:** extract the ZIP to a user-writable folder (not inside the ZIP).
  Install Microsoft's free Evergreen WebView2 Runtime from Microsoft's official
  WebView2 download page if missing; the app intentionally does not silently
  download a runtime. Double-click `thewarroom.exe`. SmartScreen may say
  “Windows protected your PC”: only after authenticating this candidate, choose
  More info → Run anyway if offered. Never disable SmartScreen globally. Managed
  policy may forbid this; ask the operator rather than bypass it. Missing-runtime
  failure must give the WebView2 prerequisite rather than an empty app window.
- **macOS:** extract the ZIP and drag `thewarroom.app` to Applications (or your
  user's Applications folder). Launch from Finder. An unsigned/unnotarized app
  may be blocked: after authenticating the candidate, try Control-click → Open;
  if blocked, System Settings → Privacy & Security → Open Anyway, then confirm.
  On newer macOS the settings step may be required even after Control-click.
  Do not remove quarantine recursively or disable Gatekeeper. If policy forbids
  the exception, stop. WebKit is supplied by macOS; no separate WebView install.
- **Linux:** on Ubuntu 24.04 install runtime dependencies with
  `sudo apt-get install libgtk-3-0t64 libwebkit2gtk-4.1-0`.
  Extract with `tar -xzf thewarroom-linux-amd64.tar.gz`, then run `./thewarroom`
  from a desktop session (Wayland or X11). Do not run as root. The tarball retains
  its executable bit. Other distributions need equivalent GTK 3/WebKitGTK 4.1
  runtime packages; older WebKitGTK 4.0 alone is insufficient. A headless shell
  cannot prove a desktop launch.

## Acceptance on each clean machine

Record OS build, CPU, commit, package SHA-256, tester, date, screenshots and logs.
Use the *downloaded/extracted package*, never `go run` or a development build.

1. Exercise the warning path from a browser-downloaded package (macOS quarantine
   and Windows Mark of the Web must be present). Follow the instructions above.
   Record the actual warning and controls; inability to proceed is a failure.
2. Without the optional Windows/Linux WebView dependency, observe a useful failure;
   install the prerequisite, relaunch, and confirm a rendered, responsive shell.
   macOS uses its built-in WebKit. The UI must render, not just keep a process alive.
3. Open a second copy against the same disposable account/data directory. It must
   refuse with “another instance” while the first stays usable. Close the first
   normally; relaunch must succeed.
4. Force-kill the running app (Task Manager / Force Quit / `kill -9`), then relaunch.
   It must succeed despite a remaining `.lock` file. Never delete the lock as a fix.
5. Close the app. Retain evidence outside the app folder; remove only disposable
   package/account/VM state. Do not delete a live installation's databases.

## Automation and evidence limits

`desktop-packages.yml` checks/builds platform-neutral frontend assets once on Linux
and transfers that same commit-named bundle within the workflow to each native
packager. Existing `ContractActs.tsx`/`contractActs.ts` and
`TradeBuilder.tsx`/`tradeBuilder.ts` module names confuse TypeScript on the default
case-insensitive Windows/macOS filesystems. Native frontend development there
remains blocked pending a separate source rename; packaging does not rename or
patch source, skip the Linux TypeScript check, or use assets from a different run.
The workflow builds native packages without regenerating bindings, extracts them
and runs `-version` to verify the executable loader/build stamp.
`python -m unittest discover -s scripts/desktop -v` proves archive bytes/modes/
checksums and contention, normal close and forced-kill release using the existing
`flock`/`LockFileEx` implementation compiled into a tiny subprocess probe. It never
opens a database or makes an MFL call. No new lock implementation is introduced.

The CLI smoke does **not** initialize a WebView, prove GUI launch, exercise OS
warnings, or prove clean-machine installation. Hosted runners have SDKs and may
already have WebView dependencies. All three manual drills above remain release
blockers until their evidence is recorded on the exact candidate; no passes are
claimed by this document. Owner OS/CPU inventory is still outstanding.

Local reproduction after frozen frontend install/build on a case-sensitive
filesystem (or downloading the exact candidate's checked frontend asset bundle
to `frontend/dist`) and exact-version Wails CLI installation:
`python scripts/desktop/package.py`. Build tools additionally
need Go from `go.mod`, Node 24, pnpm from `frontend/package.json`, a native C compiler,
and Linux GTK/WebKit development headers. Output is under `.tmp/desktop/artifacts`.
The script refuses non-target hosts rather than silently cross-compiling.
