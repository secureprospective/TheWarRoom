# Slice 10 — server football evidence

## Flow and ownership

Console/service caller supplies JSON bytes and their SHA-256 → `history.ImportFootball`
checks the version, provenance, canonical player IDs, unique reference identities, facts
and NFL weeks before any write → existing immutable `raw_archive` and `fetch_log`.
No new database, migration, package, dependency, network fetch or MFL key.

At startup, `leagueservice.NewFootballHost(ctx, host, store, digest)` loads that exact
body, verifies its hash and validates it again before installing the existing
`history.LineupEnvironment` port. The returned Host supplies copied player facts and
schedule dates to the existing service. Importing another valid body does not change
an already selected environment. Configure the digest explicitly again after restart.
Call this before serving concurrent requests, like the existing environment setters.
The due worker can use the selected Host's `NativeFacts` provider.

An executing caller carries the reviewed digest using `WithFootballDigest`. The Host
checks it before dispatch and composes the original credential check with another
check inside append, before retry lookup. Missing or different digests refuse without
appending. Previews check the same digest using the request context supplied by the
Host. Domain authorization, calendar gates and append remain the existing store's job.
The digest is reference evidence, never a credential or a privilege grant.

## Evidence

- `TestHostedCommandRequiresPinnedFootballDigestWithoutDesktopKey`: same roster-cut
  command refuses missing/changed pins, commits once with the selected pin, preserves
  retry identity, and refuses a retry with a changed pin. Desktop facts panic if used.
- `TestFootballAdmissionRechecksInsideAppendAndPreservesCredential`: a credential
  denied inside append still denies; no new event.
- `TestFootballImportPinsIdentityFactsAndScheduleAcrossReopen`: copied identity/fact/
  schedule ports, exact archive digest after reopening, missing week/season refusal.
- `TestInvalidFootballImportPreservesPriorState`: invalid IDs, identities, facts,
  schedules, JSON and hashes leave archive/fetch counts and prior evidence unchanged.

This is service/port integration, not a public evidence-upload endpoint or deployable
account-authorized import UI. The loopback fixture binary still exposes only its
existing fixed authority. Stat evidence and frozen scoring remain slice 11. The
selected digest lives in startup configuration/request context, not a new durable
command field; changing command schemas or generated bindings is outside this slice.
No claim of a live deployment, source ingestion, or packaged-client workflow.
