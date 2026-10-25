# League service

Transport-independent native league reads, previews and changes. Wails and hosted
HTTP share this service; the history store owns atomic authorization and append.

Release-blocker evidence and the remaining matrix gaps are recorded in
`docs/build-handoffs/Release_Blocker_Matrix_2026-10.md`. The `release_*_test.go`
tables use disposable SQLite leagues and injected server clocks, never live data.
