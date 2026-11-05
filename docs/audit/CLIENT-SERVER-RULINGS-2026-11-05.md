# Client/server rulings — Christopher, 2026-11-05

Answers to the 7 rulings in docs/audit/CLIENT-SERVER-RECOMMENDATIONS-2026-11-05.md §7.

1. **Thin apps, with solo in the first-class app.** All clients are thin windows onto a server. Only the first-class (desktop) app may run solo: no connection to a shared server, but the owner also installs the server locally to store the data, exactly as if connecting to an existing server. Do not change the app for solo: same app, same code path, pointed at a local server.
2. **Browser login:** secure cookies + CSRF (desktop/CLI keep their keys); security review before real owners use it.
3. **Speed law conditions:** home broadband, desktop or modern phone, one 32-team league hosted; report every breach.
4. **Backups:** raise the size limit to fit real data; pause league actions during an R2 outage long enough to break the 15-min RPO.
5a. **Our ledger wins** (MFL = import/reference). For solo use, owners need an alert prompting a refresh when updated data is available. Update the old "MFL wins" lines in CLAUDE.md.
5b. **TOTP optional for everyone**, commissioners included.
6. **Cutover moves leagues + accounts + plans + full history** — pending a second opinion (Tom) on this and the whole architecture.
7. **Human steps:** only first-time key setup and destructive restores need his click; everything routine is automatic.
