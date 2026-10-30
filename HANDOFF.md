# Handoff

## Baton
Review — 2026-10-30, `session/upgrade-verify`. No push authorized.

## Where it stands
- Slice 2 wires installed `verify-release` ahead of candidate copying, extraction or execution.
- Install pins the bundle root; upgrade retains private TUF state, serializes runs, freezes and rechecks authenticated bytes, then uses the existing service-user sandbox.
- Snapshot, backup, migration and readiness ownership remain unchanged. Signing adds the server contract's monotonic release custom field.

## Evidence
- Go race: 38 top-level PASS / 81 including subtests across releasetrust, twr-server and serverversion.
- Python publisher: 11 PASS. Alpine actually ran: signed good upgrade, seven trust refusals, signed sandbox probe and newer-file refusal; unchanged binary/DB hashes and no service calls on refusals.
- Lint, boundary (159 frozen sources/functions), placement and shellcheck PASS.
- Logs and detailed limits: `_x/REPORT-upv-s2.md`, `_x/alpine.log`, `_x/go-tests.log`, `_x/python-tests.log`, `_x/lint.log`.

## Next move / blocked on
Review the commit. No production trust ceremony, real Proxmox drill or independent Claude review claimed. Alpine used a static Go-built server and checksum-verified cloudflared, not a new GoReleaser bundle build.

## Tried and rejected
- Bare binaries and candidate scripts before authentication remain rejected.
- An old shell fixture rewound only the schema number over newer tables; replaced with an older release stamp at the current schema. Go tests separately exercise actual migration and put-back.
- Expiry fixture must advance timestamp version: an equal-version expired delivery legitimately reuses a fresh cached timestamp.
