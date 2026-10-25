# Install The War Room

These are unsigned **candidate installers**, not approved owner releases. Only open a
copy Christopher has personally authenticated. A checksum beside a download catches
copy errors; it does not establish who published it. Stop if the publisher or file
is unexpected. No paid Apple signing or Windows Authenticode is used.

## Windows

Download `warroom-windows-amd64.exe` and double-click it. If Windows says “Windows
protected your PC”, choose **More info → Run anyway**, but only for the copy you
have authenticated with Christopher. Do not turn SmartScreen off. If your computer's
policy does not offer that option, stop and ask Christopher.

Follow the installer's Next/Install prompts. It installs for your account, adds
**The War Room** to Start, and installs Microsoft's free WebView2 browser component.
Keep the internet connected for that component's first installation. If installation
fails, check the connection and retry; ask Christopher if it still fails. Open
**The War Room** from Start. Remove it later through Windows Settings → Apps.
Your league data is not removed by uninstalling.

## Mac

Download `warroom-darwin-arm64.dmg` on an Apple silicon Mac. Double-click it, then
drag **thewarroom** onto **Applications** in the disk window. Eject the disk and
open the app from Applications.

macOS may block this unsigned, unnotarized app. Only after authenticating it with
Christopher, go to **System Settings → Privacy & Security**, find the blocked app,
choose **Open Anyway**, then confirm **Open**. On older systems Control-click →
Open may also offer an exception. If macOS says the app is damaged or contains
malware, stop and ask Christopher; do not override that warning. Never disable
Gatekeeper or remove quarantine with a command. WebKit comes with macOS; there is
no separate browser component to install. To uninstall, move the app to Trash;
that does not delete your league data.

## Linux (Ubuntu 24.04, Intel/AMD 64-bit)

Download `warroom-linux-amd64.deb`. In Files, open it with **App Center** or
**Software Install**, choose **Install**, and approve the system's password prompt.
The package manager installs GTK 3 and WebKitGTK 4.1 along with the app. Keep the
internet connected while it installs dependencies. Open **The War Room** from the
applications menu. Remove it later through your software manager.

If the download opens as an archive, or your software manager cannot install it,
stop and ask Christopher to arrange installation. Do not extract the binary or
paste terminal commands. Other distributions and CPUs are not yet supported by
these candidates. League data is not removed by uninstalling.

## Target inventory and release gate

**Owner OS/CPU inventory has not been supplied.** The inherited provisional build
matrix is Windows x64 (runner Windows Server 2022), Apple silicon macOS (runner
macOS 15 arm64), and Ubuntu 24.04 x64. These runners are not proof of Windows 10/11,
Intel Macs, other Linux distributions, or any particular owner's machine. Christopher
must record each owner's OS version and CPU before approving the delivery matrix.
Do not treat these provisional targets as an inventoried owner list.

The exact installer files above are the `--artifact` inputs to slice 43's TUF
publisher; see `Release_Trust.md`. CI builds no private signing keys, production
trust root or Tunnel secrets. First-install public-root embedding and updater
integration are slice 44's boundary, not a claim made by this packaging change.

## Acceptance on each clean machine

Use a disposable clean OS account/VM without live War Room files. Record OS build,
CPU, candidate commit, package SHA-256, tester, date, screenshots and logs.

1. Browser-download the exact authenticated installer. Windows Mark of the Web and
   macOS quarantine must be present. Follow the warning steps above and record the
   actual controls. A blocked installation is a failure, not permission to weaken
   system security.
2. On Windows test with WebView2 absent, then already installed. On Linux test with
   GTK/WebKit runtime dependencies absent. The installer must supply prerequisites
   or provide a useful failure. Retry with internet connected and confirm a rendered,
   responsive app. macOS supplies WebKit itself.
3. Open a second copy against the same disposable data directory. It must refuse
   with “another instance” while the first stays usable. Close the first normally;
   relaunch must succeed.
4. Force-kill the app with Task Manager / Force Quit / System Monitor, then relaunch.
   It must succeed despite the old `.lock` file. Never delete the lock as a fix.
5. Remove the app through the normal OS mechanism. Confirm unrelated files and
   retained league data are not deleted.

**No manual warning-path or novice install evidence is recorded yet.** Owner
inventory, warning dialogs and the full responsive-shell drill remain release
blockers for slices 51–52. Automated readiness is not a substitute for these steps.

## Automation contract and source trace

`.github/workflows/desktop-packages.yml` already owns packaging (there is no package
job in the current `verify.yml`). It is extended, not duplicated. Its native
`package` matrix builds the frontend and Wails executable, then creates a per-user
NSIS installer, drag-to-Applications DMG, or dependency-declaring Debian package.
The small NSIS recipe follows Wails' bootstrapper approach but avoids its generated
machine-wide/admin installer and broad recursive uninstall. Only named installed
files are deleted. CI authenticates the bundled bootstrapper's Microsoft signature;
Evergreen runtime installation still needs Microsoft's network service.

Flow: reviewed native binary → `scripts/desktop/package.py` → exact TUF target
installer + checksum → artifact transfer → separate fresh `install` runner → native
installation → installed executable `-package-smoke <disposable-directory>` → Wails
WebView DOM ready marker → second-copy refusal → forced kill → successor readiness.
The smoke flag never constructs the local App or opens league data, credentials or
refresh workers. It holds the existing `flock`/`LockFileEx` guard beside a disposable
`smoke.db` path; the database itself must not exist. Errors/timeouts fail the job.
Linux uses Xvfb but no WebKit sandbox-disabling switch. The install runner does not
install Go, Node, Wails or development headers. Hosted images still contain other
software; a fresh runner is not a pristine consumer OS image.

Python tests inspect Debian dependencies, desktop launcher, executable permissions
and checksum, and exercise normal-close/kill lock release on all native builders.
`install (windows-amd64)`, `install (darwin-arm64)` and `install (linux-amd64)` exercise
the **installed** executable's real WebView and cross-process lock. DOM readiness
proves browser initialization, not the complete signed-in UI, OS security dialogs,
keychain, enrollment, updater, or an absent-dependency Windows machine. Capture those
separately in the clean-machine drill above. CI does not disable OS warnings or
simulate a successful human exception.

### Upstream research

Checked for this slice (2026-10-25):
- [Wails v2.13 NSIS guide](https://github.com/wailsapp/wails/blob/v2.13.0/website/docs/guides/windows-installer.mdx)
  and its generated `build/windows/installer/wails_tools.nsh`: NSIS, Evergreen
  bootstrapper `/silent /install`, native package outputs.
- [Wails installation requirements](https://github.com/wailsapp/wails/blob/v2.13.0/website/docs/gettingstarted/installation.mdx):
  Linux GTK/WebKit development dependencies, WebKitGTK 4.1 build tag.
- [Microsoft WebView2 distribution](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/distribution):
  Evergreen bootstrapper, per-user installation, online prerequisite installation.
- [GitHub-hosted runners](https://docs.github.com/en/actions/reference/runners/github-hosted-runners):
  fresh hosted VMs, OS/CPU runner labels and preinstalled-software limits.
- [Apple unsigned-app guidance](https://support.apple.com/en-us/102445):
  Privacy & Security exception after a blocked launch; never bypass malware warnings.

Developer reproduction: frozen frontend install/build, exact-version Wails CLI,
Go from `go.mod`, Node 24, pnpm from `frontend/package.json`, native C compiler and
platform packaging tools. Run `python scripts/desktop/package.py`. Windows additionally
requires NSIS and the Microsoft-signature-checked bootstrapper at
`.tmp/desktop/MicrosoftEdgeWebview2Setup.exe`; output is `.tmp/desktop/artifacts`.
