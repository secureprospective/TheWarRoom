# Opaque random canonical player identities with side-qualified ownership

Status: Accepted
Date: 2026-11-20

**In plain words:** Every player gets TheWarRoom's own permanent ID, with MFL, ESPN and the rest as labels on it, so no outside site can scramble a league's roster.

TheWarRoom assigns each player an opaque, immutable `player.<128-bit random hex>` canonical identity (`docs/architecture/storage-recovery-identity.md`, `internal/commissioner/native_seed.go`) mapped to external providers (`mfl`, `gsis`, `sleeper`, `espn`, `pfr`) via `player_external_id`, and keys roster ownership uniquely on `(league, side, canonical_player)` rather than MFL player IDs. This decouples the native league ledger and multi-provider football data from MFL's string IDs and supports twin-pool leagues (where each conference side owns an independent copy of every NFL player) without schema changes.

## Considered Options

- **Use MFL player IDs as the primary key across the native ledger**: Rejected because MFL IDs are owned by an external host, cannot represent players without an MFL record, and tie multi-provider football ingestion to one vendor's numbering.
- **Key ownership on `(league, canonical_player)` without a `side` dimension**: Rejected because twin-pool leagues require each conference side to roster the same NFL player independently.

## Consequences

- Legacy MFL-shaped APIs and what-if caches require an explicit compatibility adapter (`docs/architecture/storage-recovery-identity.md`).
- Roster restore points and transfer documents must bundle owned canonical IDs and provider links so imports can verify identity without guessing.
