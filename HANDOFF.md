# Handoff

## Baton
Bee — 2026-10-28, `session/terra-franchise-scope`. Do not push without direction.

## Where it stands
- Scoped reserve exceptions, player decision exceptions and contract quotes by franchise for per-side pools.
- Legacy player-only exceptions remain valid in single-pool leagues, but do not cross per-side copies.
- Contract quote schema changed; frozen command metadata updated. Fixture rewrites contained unrelated
  timestamps and were discarded.

## Next move
Review commits and run the full race suite on a quiet machine. Verify the reserve preview and contract
quote flows in the UI; the new franchise argument is available at the API boundary.

## Blocked on
No upstream configured, so `git pull --ff-only` could not run. No push attempted.

## Tried and rejected
Fixture regeneration changed volatile timestamps and revisions without an intended wire change.

## Debt left
Full race suite timed out under concurrent workload; targeted race tests passed. UI affordances for selecting
franchise in quotes and exception forms have not been implemented or verified.
Bee — 2026-10-28, `session/terra-side-ui`. Do not push.

## Where it stands
- Commissioner bid choices identify player and side; the tray sends both. Single-pool choices are unchanged.
- Startup draft readings identify the viewed side; live commissioner tray actions send it, except open.
- Build, touched Go race tests and composition retry, frontend suite and typecheck passed. Initial concurrent
  tests timed out under load. Hosted fixture regeneration made volatile changes; those were discarded.

## Next move
Review the branch and test AFC and NFC auction decisions and startup draft actions in the GUI.

## Blocked on
Nothing known; the GUI click-through remains outstanding.

## Tried and rejected
No stored side map: sides are counted from conferences. Fixture timestamp and identity churn was not retained.

## Debt left
Rookie draft per side; per-side O-Line lots; reserve and rule-exception scoping by franchise; per-side
contract quote. Older F3a O-Line transition question belongs to Christopher/ClaudeBox.
Tom (Beelink) — 2026-10-28, `session/migration`. Do not push.

## Where it stands
- "Migrate my MFL league" engine, M1–M5 (brief `~/fleet/briefs/tom-migration.md`):
  `internal/leaguemigration` (read, prepare, review, invites, assist), `mflsource` (read-only MFL
  reader), `commish.league.cutover`, the "Your MFL league" settings entry, the all-or-nothing ledger batch,
  and six `league.migrate.*` registry commands on the assist door, wired in composition.
- Proof: round trip on the held Legacy NFL (14432) and 10 synthetic edge leagues, checked against the raw
  MFL export field by field; 7 injected failure points leave nothing; scripted AI assist run end to end
  through the registry. Click script: `docs/build-handoffs/MFL_Migration_Click_Test.md`.

## Next move
Christopher reviews the report. Then the UI debt below, then the click walk (part B) on Claude-OS.

## Blocked on
Nothing.

## Tried and rejected
Storing MFL owner emails at import: they would sit in the league-visible ledger. Invites re-read MFL's owners
(read only) instead. Adding the move's commands to the Wails/native doors: that changes the frozen door
inventory and generated bindings; they sit on their own assist door until the UI needs them.

## Debt left
- UI: start-form "Import my MFL league", review screen, cutover confirm, invite screen; and
  `scoring.rules` shown on the scoring screen. None is built; part B of the click script waits on them.
- CLI: the move is not on `twr`'s hosted dispatch yet (assist door only).
- Invites are issued in the app; nothing emails the link. Invitee principal is the owner's email.
- MFL's owner emails come only with the commissioner's key: offered for 14432 only; other leagues list
  every owner for a hand invite unless a key is added.
- `mflsource` has no HTTP test (the MFL client pins MFL's hosts); it composes the existing fetchers.
- Settings MFL holds but the app has no field for (h2h, rostersPerPlayer, contract-year use) are not read;
  team-defense or DL/DB positions refuse the import with a plain list.
- A raw `commish.league.create` with the MFL preset accepts caller-supplied values (same power as setting
  rules by hand); the move's own path always builds them from the read.
