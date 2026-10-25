# Native desktop account access

Launch a hosted-only window with the origin and league ID supplied by the operator:

```sh
thewarroom -native <https-origin> <native-league-id>
```

The origin is HTTPS with no path, query, fragment or embedded credentials. Normal
system certificate trust applies; redirects and insecure TLS are refused. Do not
put a password or enrollment code on the command line. This mode never constructs
the local App, opens its databases, asks for an MFL key or starts MFL workers.
Launching without arguments retains the existing local/reference app.

## Owner steps

1. Choose **Join with a code**. Use the intended username and the enrollment code
   personally handed to you by the commissioner. Choose your password. Codes last
   48 hours and can be used once. If expired, used or for another username, ask for
   a new code; if already enrolled, sign in instead.
2. Sign in, then choose **Open my franchise**. Enrollment creates an account, not
   a seat. Only the current server-authorized assignment determines your franchise.
   Ask the commissioner to check the assignment if access is refused; do not make
   another account. Commissioners without a franchise can still enter league access.
3. Commissioners and deputies must first choose **Authenticator**. Enter username
   and password, get the private setup key, and add it to a time-based authenticator.
   Enter your password again and confirm a six-digit code. Wait for the next code,
   enter password again, and choose **Verify fresh code**. Open the franchise.
   Verification lasts five minutes; a reused code cannot verify access.

**Account recovery** is always reachable, even without a session. It explains
expired codes, missing seats and lost passwords/phones. This build has no remote
recovery-code redemption or printing route: do not paste recovery codes into chat.
Contact the operator and offline key holder for the existing audited console
recovery (`twr-server recover`). Reassigning a seat cannot reset an account.
Recovery revokes old credentials; sign in and enroll the authenticator again.
Eight one-time printed codes remain the account recovery mechanism, but their
self-service desktop transport is not supplied by the current hosted API.

## AI helpers (commissioners)

After verifying a fresh authenticator code and opening league access, a
commissioner sees **AI helpers**:

1. **Connect a helper.** Choose "look and suggest" (read, preview) or "also
   record" (adds execute), and the franchise it acts for. The scope is every
   catalog command `accounts.DelegableCommand` allows; roles, seats, credentials
   and generic corrections are never delegable. The desktop shows the `twr connect`
   command and a `twr1.` connection line once; the helper ID stays filled in.
2. **Confirm a helper's change.** Paste the helper's `twrc1.` confirm request. The
   screen shows the command, reason and arguments before anything is sent; Go
   decodes and checks it again and posts it to `/v1/native/confirm`. The code
   covers that exact change for five minutes.
3. **Disconnect a helper** revokes its token at once.

The connection line is the only agent secret the webview ever shows, and only
right after it is made. Device bearer and recent-auth proof stay in Go. CLI and
MCP use: `cmd/twr/README.md`. Evidence:
`TestDesktopConnectsConfirmsAndRevokesAnAIHelper` (real HTTPS: refused without
fresh verification or league access, preview, unconfirmed G2 refused, confirmed
G2 recorded, revoked helper refused) and `nativeHelpers.test.tsx`.

## Boundary and state ownership

| Flow | Owner / failure |
|---|---|
| Native launch → Wails assets + local `/desktop/native` handler | `internal/composition/run.go`; no bound local mutation API or local App startup. |
| Screen → validated `RawDesktop` action | `internal/serverclient`; bounded JSON, closed action set, webview-origin check. No execute action. |
| Go → fixed HTTPS origin → account/TOTP routes | Go holds bearer and recent-auth proof in memory only. No bearer in webview, disk, storage or URL. Password fields clear after every attempt; setup keys clear on navigation. |
| Open franchise → authenticated `/v1/native/people` → existing shared service | Current session, league membership and privileged TOTP gates apply. The desktop returns only league name, role and assigned franchise; no revisions or ledger bodies. Failed rereads clear the old view. |
| Close / window sign-out | Clears Go credentials; does not claim server-side revocation. Keychain and device-management UI belong to slice 22. |

The people endpoint was needed because the earlier hosted API exposed only ledger
reads, not the current account's seat. The first owner integration test also found
that hosted admission checked `HasRole(Owner)`, but owners are seats, not explicit
role grants. Admission now checks current seat ownership, including verified alias
resolution already applied to the principal. No service/commissioner token code
or generated bindings changed. The local asset handler avoids new Wails bindings.

## Evidence and limits

- `TestDesktopNativeEnrollmentAssignedFranchise`: real trusted HTTPS enrollment,
  used-code refusal, login, no implicit seat, server seat assignment, owner entry,
  presentation-only response, window sign-out; reads/denials append no events.
- `TestDesktopExpiredInviteAndPrivilegedAuthenticator`: exact 48-hour refusal,
  privileged gate, TOTP setup/confirm/replay/fresh-code/expiry, missing session.
- `TestDesktopBoundaryAndTLS` and malformed-response tests: TLS trust, origin,
  JSON/unknown actions, bounded replies, no upstream error/secret forwarding.
- `nativeAccount.test.tsx`: rendered join/login/franchise flow, clearing stale
  private views, expired-invite and lost-factor guidance, setup-key clearing,
  strict webview response parsing and local transport only.
- Native startup is source-traced to bypass local App creation. Headless flag
  rejection is tested; a packaged Wails launch and novice walkthrough on each OS
  remain unproven. No public exposure, MFL calls or roster moves were exercised.
