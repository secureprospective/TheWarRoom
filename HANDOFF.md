# Handoff

## Baton
Bee / gate debt — 2026-11-01, build/GATES-1. Local work only; no remote contact.

## Where it stands
- APP-GATES-1 is incomplete. Digest bisect identified 511bc901: the setup job excludes inapplicable rulings. Draft, order and assets hashes match its parent; only Jobs changed (8 rulings to 6). The test now pins the unchanged components and the revised aggregate.
- Focused commissioner race tests 3/3 passed. Preset guard `TestOnlyTheCatalogNamesAPreset` passes here; the reported baseline archive is not present in this worktree, so the failure remains unverified.
- `make bloat` fails 26 tiny files vs 25; `make boundary` fails first at value preparation, with additional migration and other composition inventory differences; `make placement` fails source-origin check and native-fork ratchet (20 vs 19).
- Frontend tests: full run 1732 passed, 19 failed in 3 files; isolated reruns of those 3 files passed all 28 tests. Possible load/concurrency flake, cause not established. Typecheck, golangci-lint, ifaceguard, vet, frontend transport, filelen, Wails version and existing bundle/CSP checks passed.

## Next move / blocked on
- Fix bloat by merging a tiny file into its natural neighbour, then re-run the ratchet. Resolve origin-based schedule following and the frontend fork without hiding policy.
- Move decision logic to the correct service boundaries, including migration, value preparation and the other changed composition functions; do not refresh fingerprints to mask policy.
- Reproduce preset guard with the original archive context; investigate frontend failures separately. Run remaining gates and one full race suite only after `heavy-ok.sh` passes.

## Tried and rejected
- Updating boundary fingerprints or raising ratchets: would hide unreviewed changes.
- Assuming the aggregate digest drift was a draft regression: component hashes and bisect show a deliberate Jobs change instead.
