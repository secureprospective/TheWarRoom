# League service

Transport-independent native league reads, previews and changes. Wails and hosted
HTTP share this service; the history store owns atomic authorization and append.

Release-blocker evidence and the current-role matrix are recorded in
`docs/build-handoffs/Release_Blocker_Matrix_2026-10.md`. The `release_*_test.go`
tables use disposable SQLite leagues and injected server clocks, never live data.

The catalog-backed mutation fixture and adapter tests live in
`internal/app/commissioner/testdata/release-mutations.json` and adjacent
`release_*_test.go` files; CLI/MCP use that same fixture. Future support/operator
grants remain outside the implemented role model.
