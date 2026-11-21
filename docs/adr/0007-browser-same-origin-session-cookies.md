# Browser access uses same-origin session cookies and single-league admission

Status: Accepted
Date: 2026-11-20

Browser access at `/app/` uses strict same-origin `__Host-twr_session` cookies (`HttpOnly`, `Secure`, `SameSite=Strict`) and single-league admission, while forbidding `Authorization` bearer headers and rejecting agent tokens when an `Origin` header is present (`internal/serverhttp/browser.go:79-106`, `internal/serverhttp/browser_session.go:159-174,225-245`, `docs/architecture/desktop-modes.md:4-5,18`). This keeps long-lived owner bearer credentials and AI helper tokens completely out of browser JavaScript/WebViews and blocks cross-site request forgery.

## Considered Options

- **Store owner bearer tokens in browser `localStorage` or JS memory**: Rejected because any script execution or WebView leak would expose long-lived account credentials.
- **Allow multi-league switching inside a single browser `/app/` session**: Rejected to keep browser admission single-league and CSRF-isolated; multi-league requests return HTTP 409 and defer to the desktop thin client.

## Consequences

- Browser `/app/` sessions are restricted to one admitted league per session and require HTTPS on the configured `PublicHost`.
- Multi-league owners and CLI/MCP agents authenticate through the desktop thin client or bearer/agent tokens without a browser `Origin`.
