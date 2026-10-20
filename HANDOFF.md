# HANDOFF

- **Baton:** Christopher/reviewer — 2026-10-20; Reorg 9b complete on `session/reorg-race-v2`; no push.
- **Where it stands:** `internal/app/season/power.go` owns the shared power rows, results and view constants formerly in scoring. Scoring, Race, fixtures and root adapters use season directly; no aliases remain in scoring. `PluralNoun` moved from Sources to `internal/app/report/prose.go`, with direct tests beside it.
- **Guards/docs:** shared set is report plus season; explicit Race and season boundaries added, all feature allowlists updated, and map/placement/extraction guidance updated.
- **Proof:** `make lint` exits 0. Deliberate Race imports of scoring and sources each fail `app-race-no-siblings` depguard; both removed. Root wire goldens pass unchanged; bindings freshness passes with no generated bindings diff. Final locked `make verify` exits 0, including the full race suite and frontend gates. Root Go file count remains 218.
- **Next move:** review and merge on Christopher's approval. Check Race's season/franchise reports retain their existing wording and rows; no UI or wire changes are intended.
- **Blocked on:** nothing.
- **Tried/rejected:** sibling-feature dependencies are forbidden; shared models and wording now have explicit shared homes instead of aliases or guard exceptions.
