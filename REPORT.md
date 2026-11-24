Done: all requested lines passed.
- Pointer/prune test: `TestLeaguePinsShareDecodedFootballByDigest`; removing cache made it fail: a=0x25e27c0dc000, b=0x25e27c0dc240 (distinct pointers).
- Missing retry: `TestMissingPinnedReferenceStopsHostedSelection` passed (missing remains missing before 1 minute; retry picks it up).
- N=8 TotalAlloc deltas: 8 independent `LoadFootball` decodes 367,840 B; cached `selectPins` build 58,496 B (cached side includes per-league setup).
- Race: `go test -race -timeout=30m ./cmd/twr-server`, `./internal/store/history`, `./internal/serverhttp` all passed.
- Lint: `make lint VERSION=dev COMMIT=pincache BUILD_DATE=pincache` passed; line-budget reason/baseline updated.
- Immutability: `LoadFootball` constructs digest/raw/stats; production readers only read them. `SetPinnedFootball` installs the same pointer in lineup/bid environments without mutation. `PinOfficialStats` creates/imports a new evidence body; it does not modify current evidence.
- Construction-time slices/maps are populated/sorted before returning the pin; no post-load in-place raw/stats mutation found. `Kickoffs` lazily builds/sorts its cache once; it exposed that slice, so it now returns a copy.
- Tidy found only other fleet agents' active work; left their processes untouched. No task-owned processes remain.
- Changed: `cmd/twr-server/football_pins.go`, `cmd/twr-server/reference_backup_test.go`, `internal/store/history/football_evidence.go`, `internal/store/history/football_schedule_test.go`, line-budget baseline/reason, `HANDOFF.md`.
