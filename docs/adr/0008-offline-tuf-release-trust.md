# Offline TUF release trust with 2-of-3 Ed25519 root threshold

Status: Accepted
Date: 2026-11-20

**In plain words:** Updates are checked against our own signing keys, so nobody can push a fake or older update, and we don't pay Apple or Microsoft for signing.

Server and desktop releases are verified using offline TUF (The Update Framework) metadata with a 2-of-3 Ed25519 root threshold, distinct targets/snapshot/timestamp keys, and pinned bootstrap metadata (`docs/delivery/Release_Trust.md`, `internal/releasetrust`) instead of relying on OS code-signing or TLS certificate pinning. This guarantees cryptographic rollback, freeze, and key-compromise protection for thin client and hosted server upgrades without paying for Apple/Microsoft notarization or breaking when public HTTPS certificates renew.

## Considered Options

- **Rely on HTTPS/TLS and OS code-signing (Apple notarization / Authenticode) alone**: Rejected because TLS does not protect against repository compromise or rollback attacks, and OS notarization adds recurring vendor cost without TUF's threshold rotation.
- **Pin public HTTPS TLS certificates in the desktop client**: Rejected because routine certificate renewals would brick installed clients.

## Consequences

- Publishing a production release requires an offline multi-custodian signing ceremony (`release.py`) and timely renewal of short-lived timestamp (1 day) and snapshot (7 days) metadata.
- Clients require a reasonably accurate UTC clock and must persist verified TUF metadata across restarts without overwriting it with bootstrap metadata.
