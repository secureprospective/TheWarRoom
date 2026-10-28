# Moving a league from MFL: click-test script

**Branch:** `session/migration` (2026-10-28). MFL is only read: no step submits anything to MFL.

What exists: the move's engine and its six registry commands on the assist door
(`league.migrate.prepare`, `.import`, `.review`, `.cutover`, `.invites`, `.invites.send`), and the AI
assist that drives them (`internal/leaguemigration/assistant.go`). The start-form button, review screen
and invite screen are **not built yet** (see Debt in `HANDOFF.md`), so part A is the walk today and part B
is the walk once those screens land.

## A. Headless walk (today, on the Beelink, dev database only)

| Step | Run | Passing looks like |
|---|---|---|
| 1 | `go test ./internal/leaguemigration/... -v -run 'Prepare\|Failure\|Read'` | 11 fixture leagues prepare whole; 11 failure paths each print a plain-words problem; 5 read failures each say "Nothing was imported" |
| 2 | `go test ./internal/store/history -v -run TestMigration` | Round trip PASS for the Legacy NFL (14432) and 10 edge leagues; 7 injected failures leave 0 ledger rows and the import runs again |
| 3 | `go test ./internal/commandcatalog -v -run 'Assist\|RawCutover'` | One ask, three confirms; 0 acts before each confirm; all 10 ledger entries read "via AI/CLI, acting for commish" |

## B. Click walk (Claude-OS, production build on a snapshot, once the screens exist)

1. **Start form → Import my MFL league.** Type `14432`. *Pass:* "I read MFL league 14432 (Legacy NFL,
   season 2026)", 32 franchises, the rostered-player and pick counts, and a note listing the settings MFL
   does not hold (they wait for a ruling). No MFL tab is opened or changed.
2. **Confirm the import** with a reason. *Pass:* the ledger shows two entries: "Created league …" with
   the MFL league's own settings, and "Recorded the starting rosters …". The league says it follows MFL.
3. **Rules screen → Scoring.** *Pass:* every MFL scoring rule shows, cited "Your MFL league".
4. **Review.** *Pass:* "Everything matches MFL…", 32 franchise rows each "Roster matches · Contracts match ·
   Picks match · Salary adjustments match · Name and division match". Run it twice: same words.
5. **Switch over** with a reason. *Pass:* ledger "Moved the league over from MFL: the app runs it now, and
   MFL no longer changes it", recorded-reason gravity, no undo offered. A second switch is refused.
6. **Invites.** *Pass:* one row per franchise; owners without an MFL email say "invite them by hand".
   Send: one act, each row turns to "Invitation sent to …". (Nothing is emailed: the invitation is
   issued in the app; delivering the link is still the commissioner's.)
7. **Assist:** ask "move my league 14432 over from MFL". *Pass:* it explains each step, waits at import,
   switch and send for a confirm with a typed reason, and every act it records says "via AI".
8. Close any MFL tab opened during the walk.
