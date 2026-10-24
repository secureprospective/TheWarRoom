# Desktop device sessions

Flow: native-only webview action → serialized `serverclient.Desktop.Act` →
OS keychain (`TheWarRoom.Native`, SHA-256 of HTTPS origin + league) or Go memory →
HTTPS bearer → `Hosted.revoke` → existing transactional account-session revocation.
Only the session bearer is saved; passwords and recent-auth proofs are not.
There is no file fallback. Closing clears memory but retains the keychain session;
forget removes it locally, while revoke invalidates it at the authority.
A refused session clears memory and attempts keychain deletion. Provider failure
never exposes its diagnostic to the webview. Keychain calls are synchronous under
the desktop lock: no late asynchronous write can resurrect a forgotten secret.
An OS keychain prompt can block the action until answered.

Automated evidence:
- `TestDeviceKeychainRoundtripAndIsolation`: real adapter against the provider's
  in-process mock; roundtrip, deletion, server/league isolation (not OS proof).
- `TestCredentialRestoreCloseAndForget`: restore, close retention, local removal.
- `TestUnavailableKeychainLoginIsMemoryOnly`: HTTPS login with unavailable store;
  no persisted secret, clear on unauthorized read.
- `TestDesktopDeviceRevocation`: real HTTPS and SQLite, two device logins;
  current-device revoke preserves the other, all-device revoke denies both;
  neither appends league events.
- Frontend account test: saved-session controls and credential-free revoke action.

## Still required on the packaged candidate

No packaged three-OS keychain test was performed in this Linux worktree. On each
Windows/macOS/Linux owner target, record candidate SHA, OS/package, tester and result:
1. Use a disposable hosted account. Sign in; message must say OS keychain saved.
   Close/reopen the package; open the assigned franchise without a password.
2. Verify only `TheWarRoom.Native` holds the device secret; no bearer/password in
   webview responses, logs or app configuration. Authenticator verification must
   be required afresh for privileged access after restart.
3. Revoke this device; opening its franchise and reopening the package must fail.
   A second device remains usable. Repeat with all devices: both must fail.
4. Lock/deny the OS keychain and sign in: memory-only warning, working access in
   that window, no newly persisted plaintext credential; after close, sign-in is
   required. Existing inaccessible entries cannot be promised deleted: reconnect
   and revoke all devices before disposing of a machine.
5. Forget saved sign-in and restart: no restored session. If keychain deletion is
   denied, the UI must report failure rather than promise removal.

Plan correction: revocation existed in the accounts service, but no HTTPS revoke
route existed. This slice adds that narrow route; it grants no league authority.
