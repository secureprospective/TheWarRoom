# Pure scoring engine and quarantined Layer 4 scouting prior

Status: Accepted
Date: 2026-11-20

The scoring and valuation engine (`internal/engine`, `internal/model`) is enforced pure with zero I/O or store imports by `depguard` (`.golangci.yml`, `SYSTEM_MAP.md:28`), receiving all state as explicit parameters assembled by `internal/composition`. In addition, the Layer 4 scouting prior is strictly quarantined from fantasy production and scoring configuration (`CLAUDE.md`) so dynasty valuations never double-count production.

## Considered Options

- **Allow engine packages to query stores or external feeds directly**: Rejected because hidden I/O makes scoring runs non-deterministic, couples valuation logic to SQLite schemas, and prevents reproducible offline fitting and testing.
- **Blend scouting priors directly with fantasy production weights in one scoring config**: Rejected because mixing prospective scouting priors into production-based scoring parameters silently double-counts observed NFL output.

## Consequences

- Scoring and valuation runs are deterministic functions of an explicit parameter snapshot and pre-assembled history features.
- `internal/composition` must assemble all required player facts upfront before invoking the engine, and `depguard` fails the build if an engine package imports any store or I/O package.
