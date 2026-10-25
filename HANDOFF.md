# Handoff

## Baton
Bee — 2026-10-25. Slice 36 on `session/cs36`; no merge authorized.

## Where it stands
- Encrypted R2 matched-set capture/read-back, retention, offline staged restore and hosted DB coverage admission implemented.
- Targeted race tests and disposable Alpine deployment/setup passed; push/CI pending.
- Setup and limits: `deploy/server/BACKUPS.md`. Source/health/restore seams: `internal/offsite/README.md`.

## Next move
Review PR/CI. Slice 37 owns spare-host activation/auth-epoch reset and timed recovery; slice 38 consumes private backup status.

## Blocked on
Real R2 credentials and physical host-loss/spare-host drill are unavailable; no live recoverability or four-hour RTO claim.

## Tried and rejected
- Upload success/ETag or persisted local receipt as coverage: full encrypted read-back required; restart begins closed.
- Private restore key on CT: only public X25519 recipient accepted.
