# Hosted private notes: storage decision required

## Question

May this slice add the existing personal-notes table to the hosted league's migration list,
with the corresponding server release-contract update? If not, which already-versioned,
league-local storage location is intended for it?

The task forbids release-number changes. Do not work around that constraint with unversioned
DDL, editing a historical migration, calling the desktop schema initializer on a league file,
or putting private notes in the shared evidence file.

## Observed blocker

- `internal/store/personal/store.go`: `Store.Initialize` calls `db.Pools.InitializeMain`.
- `internal/db/main_schema.go`: the desktop main schema is version 2 and owns
  `personal_entries`; the existing notes store queries this table.
- `internal/store/history/file_migrations.go`: hosted league schema is version 15;
  its migration list creates no `personal_entries` table.
- `cmd/twr-server/league_files.go`: each hosted league opens its own versioned file.
- `internal/serverversion/version_test.go:TestCurrentContract`: the current release row must
  match `history.LeagueSchemaVersion()`; a new league migration requires a release row.

A disposable league file was created with `history.PrepareNewLeague`, then checked with
`db.HasTable` and passed to the existing `personal.New(pools).Initialize`. Both attempts
reported that `personal_entries` was absent and returned this same error:

```text
personal: initialize schema: db: this database was last written by a NEWER version of TheWarRoom (schema v15; this build understands up to v2) — update the app to open it
```

No release metadata or schema migrations were changed. Partial implementation was removed
from the reviewable tree rather than committing a route that cannot work on hosted files.

## Intended access rule

1. Catalog marks PRIV entries `franchise-owner`; dispatch admits the current authenticated
   franchise seat before invoking notes storage. Commissioner roles and caretakers confer no access.
2. Scope every operation to its league and franchise; reuse `store/personal`, never a league reply
   cache, public read, commissioner view, search, export or payload/error log.
3. Other owners, commissioners, non-members and signed-out callers receive an identical generic
   refusal body and status. Desktop note behavior remains unchanged.

## Resume after the decision

Add the approved migration using the existing table definition, then wire the catalog read and
execute-only private-note CRUD through `StoredHost` to `store/personal`. The non-app → app
import rule prevents `leagueservice` from importing `app/offers`; reuse its store rather than
constructing the local app or duplicating notes storage. Use one hosted adapter in `offers/api.ts`
for HQ and War Room, retaining local Wails branches.

Verify real hosted owner add/read/remove, every generic refusal, private-cache/log/public-read
noninterference, touched-package race tests, frontend tests and lint. The proposed implementation
is not accepted or verified; no functional change is included in this draft.

## Slop seen

- `frontend/src/app/shell/HQInspector.tsx:102-109`: hosted mode still initiates the local age-arc
  binding while explicitly rejecting notes. Mode-aware routing should be handled by the owning slices.
- `frontend/src/app/shell/warroomInspectorReads.ts:34`: the hosted branch invokes the local
  `readAgeArc` adapter beside hosted workflow reads. Separate hosted-read work owns that mismatch.
