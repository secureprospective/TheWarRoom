# Decision records

Two kinds. **Product decisions** (plain English, ruled by Christopher) are in [product/](product/README.md); read them first.
The **technical decisions** below are for agents; each starts with an "In plain words" line.

- [0001-mfl-league-refresh-authority.md](0001-mfl-league-refresh-authority.md) — MFL league refresh splits authority between native governance/rules and MFL mirror assets/results.
- ~~[0002-v3-shell-owns-control-room-tab.md](0002-v3-shell-owns-control-room-tab.md)~~ (retired 2026-11-22) — Desktop v3 shell owns Control Room tab state and syncs `useHostedControlRoom`.
- [0003-refuse-second-league-load-after-cutover.md](0003-refuse-second-league-load-after-cutover.md) — `league-load` is strictly one-time and refused after cutover; ongoing sync uses `league-refresh`.
- [0004-three-way-sqlite-split.md](0004-three-way-sqlite-split.md) — Hosted server splits SQLite into `accounts.db`, `reference.db` (attached read-only), and per-league `league.db`.
- [0005-pure-scoring-engine-and-prior-quarantine.md](0005-pure-scoring-engine-and-prior-quarantine.md) — Scoring engine is enforced pure by `depguard`, with Layer 4 scouting prior quarantined from production config.
- [0006-opaque-canonical-player-identity.md](0006-opaque-canonical-player-identity.md) — Opaque random canonical player IDs and side-qualified ownership decouple the ledger from MFL IDs.
- [0007-browser-same-origin-session-cookies.md](0007-browser-same-origin-session-cookies.md) — Browser `/app/` uses same-origin `__Host-twr_session` cookies and single-league admission, forbidding bearer tokens.
- [0008-offline-tuf-release-trust.md](0008-offline-tuf-release-trust.md) — Offline TUF release trust with a 2-of-3 Ed25519 root threshold replaces OS code-signing and TLS pinning.
