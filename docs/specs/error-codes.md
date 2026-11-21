# Error Codes, Copy, and Send to Commissioner

When something breaks on the phone app (PWA) or desktop, any owner can hand the exact failure to the
commissioner in one tap.

## What the owner sees

Wherever the app displays an error, the owner sees:

1. **The plain message** describing what failed in ordinary words (no stack traces or internal prefixes).
2. **One short error code** right below it, formatted `TWR-XXXX-XXXX` (for example, `TWR-7K9M-2P4X`).
3. **Copy** — a tap-sized button (at least 44 px tall) that copies the code, the time in US Central
   (`America/Chicago`), the screen name, the app version, the league, and the plain message. The button
   label changes to **Copied** for 2 seconds.
4. **Send to commissioner** — a tap-sized button beside **Copy** that files a problem report through the
   existing problem-report path with the code and copied details attached, then confirms **Sent — thanks**.
   If the owner is not yet signed into a league or the server cannot be reached, the app tells them to
   tap **Copy** and share it directly.

## How a code is made

- **Format**: `TWR-XXXX-XXXX` using 32 unambiguous uppercase letters and digits (`2–9`, `A–Z` without
  `0`, `O`, `1`, or `I`), so codes can be read aloud or typed from a screenshot without mix-ups.
- **Server failures**: generated with `crypto/rand` on the server for every failed HTTP request or
  command (`status >= 400`). The first character after `TWR-` is never `C`, so server codes never collide
  with client codes. The server returns the code in the `X-TWR-Error-Code` response header, and the
  desktop proxy forwards that header to the app.
- **Client-only failures**: when the network is down, the server cannot be reached, a local binding
  fails, or a screen crashes in the app's error boundary, the client generates a code in the same format
  starting with `TWR-C` (`TWR-CXXX-XXXX`) using Web Crypto randomness.

## What the server records

For every server-generated code, the server writes one structured `slog` line to the existing server log:

- `time` (timestamp)
- `code` (`TWR-XXXX-XXXX`)
- `status` (HTTP status code)
- `route` (known server route, or `/other` for unknown paths)
- `command` (workflow or native command name, when present)
- `league` (validated `native.*` league ID, when present)
- `account` (authenticated account ID, when present)

The server never logs passwords, bearer tokens, session cookies, authenticator codes, setup keys,
recovery codes, or join codes. Searching the server log for the owner's code jumps straight to the
exact failure.

## What the commissioner sees

Tapping **Send to commissioner** submits the standard `feedback.report.create` command
(`internal/leagueservice/problem_reports.go`). The report appears newest-first in the commissioner's
**Feedback** tab alongside normal member reports, showing the owner's code, Central time, screen name,
app version, league, and message, plus the standard route, release, viewport width, and user agent.
Existing admission checks and the 10-reports-per-hour limit apply unchanged.

## Out of scope

- No second problem-report table, endpoint, or inbox.
- No automatic background crash telemetry or silent error uploads; the owner always chooses to copy or
  send.
- No server-side storage of client-only `TWR-C` codes unless the owner taps **Send to commissioner**.
- No user-facing stack traces, SQL errors, or raw server internals.
