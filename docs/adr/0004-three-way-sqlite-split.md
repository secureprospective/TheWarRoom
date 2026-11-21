# Three-way SQLite physical split on the hosted server

Status: Accepted
Date: 2026-11-20

The hosted server splits persistence across three SQLite database roles (`cmd/twr-server/league_files.go:17-27,107-130`, `docs/architecture/storage-recovery-identity.md:22-67`): `accounts.db` for sign-ins, join links, security audit, and the league directory; `reference.db` for shared football archives, model runs, and canonical player identities; and one isolated `leagues/<league>/league.db` per native league. Every league connection attaches `reference.db` read-only as schema `reference` (`db.OpenWithReference`), giving each league its own writer, cryptographic audit chain, and off-site R2 backup set with zero cross-league blast radius.

## Considered Options

- **Single shared `server.db` for all leagues and accounts**: Rejected because one league's write lock, corruption, or restore would affect every other league on the hosted server and prevent per-league backup isolation.
- **Duplicate football reference tables inside each `league.db`**: Rejected because multi-gigabyte NFL archives, player records, and model runs would be replicated per league and drift out of sync across leagues.

## Consequences

- Cross-file writes between `accounts.db`, `reference.db`, and `league.db` are ordered rather than atomic across SQLite files (`docs/architecture/storage-recovery-identity.md:52-72`), requiring pre-minted canonical player IDs and writer-reservation ordering when withdrawing credentials.
- Each `league.db` keeps its own `league_player` rows so a restore refuses any `reference.db` missing a player ID owned by that league.
