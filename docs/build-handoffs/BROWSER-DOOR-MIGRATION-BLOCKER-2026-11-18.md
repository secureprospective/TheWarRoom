# Browser door blocked: persistent league selection

**Status:** Draft blocker for Bundle 7, 2026-11-18. No browser route or server code has been changed.

## Finding

The browser session contract must return exactly one league for an account admitted to more than one league: the most recently used one. The current server account schema has no per-account or per-session league recency. `account_sessions` stores `account_id`, `device`, issue/last-used times and revocation, but no league (`internal/store/accounts/sessions.go`). The server-wide `league_directory` stores league state and creation time, but no account relationship or use time (`internal/store/accounts/server_file.go`). The desktop's session `device` is the fixed label `The War Room desktop` (`internal/serverclient/credentials.go`).

A process-local map would lose the selection on server restart. Inferring it from league mutations misses read-only use. Neither meets the locked contract, and repurposing the device label would corrupt its existing meaning. Persisting the last successfully used league therefore needs a server-account database migration. The task's fallback says to stop and open a draft PR when a rule requires a migration.

## Decision needed

Approve or reject a migration that persists each account's most recently used admitted league and records recency only after an authorized league request succeeds. On approval, decide whether recency belongs to the account or to each device session; no schema or route work should proceed before that choice. The browser must never receive a list of other leagues or call `commish.leagues.read`.

## Preserved security contract for the resumed lane

- Browser Origin is exactly `https://` plus the configured public host; no wildcard, HTTP, `null`, alternate configured origins, or missing Origin on cookie requests.
- `__Host-twr_session` is HttpOnly, Secure, SameSite=Strict, Path=/, host-only; it carries the same revocable owner session as desktop.
- Agent tokens are refused with Origin and never work from cookies; Authorization plus the session cookie is refused.
- Cookie authentication changes credential transport only. Existing authorization and permission checks remain; changes feed and enrollment remain closed to browsers.
- `/app/` serves the built web client with the required CSP, `nosniff`, same-origin referrer policy, correct manifest/service-worker types and no-cache service worker.
