# Settings slop audit (10-31)

**The model, from Christopher:** the app is the housing for the settings.
- Frontend = how a human works the backend.
- Backend = engines (the math) + stores (the database).
- A league is the values of its settings. An MFL import is just settings inside the app that already exists.
- Presets only save typing.

**The test for any code:** could a human reach this league by changing settings by hand, and change it back? If not, it is slop.

## Found

1. **Source-gated refusals.** Settings locked because the league came from MFL:
   - `commissioner/league_sides.go:118`: player pools ("an imported league keeps MFL's one player pool").
   - `commissioner/league_teams.go:180`: teams and divisions ("follows the league's MFL franchises").
   - **Fix:** delete the gate. Limits stay only where the engine truly cannot run.
2. **Source branching elsewhere** (review each; keep only true data provenance, e.g. "record the held assets" vs "start empty"
   at creation):
   - `native_start.go:16,41,160`
   - `draft_startup.go:109`
   - `jobs_league.go:25`
   - `migration.go:63`
   - `native_seed.go:340,345`
   - `creation.go:134,156`
   - `presets.go:115`
   - `principals.go:91`
   - `roster_restore.go:29`
   - `store/history/native_assets.go:99`
   - `app/commissioner/native_seed_api.go:91`
3. **MFL as an identity.**
   - `ingestion.LeagueID` (one hard-coded league) is used in 30 Go files.
   - The frontend forks MFL screens from native screens about 27 times (`FranchiseHQ` vs `NativeFranchiseHQ`, `native ?`
     in the War Room).
   - **Fix:** his real league is imported (existing migration) into a normal app league, and the screens serve app
     leagues only. The MFL-only screens retire once his league runs as an app league.
4. **Settings screen is heavy.** 130 settings. One change = pick from a dropdown, type, timing, preview, reason, confirm,
   one ledger entry.
   - **Fix:** edit many values, one preview, one save.
   - Reason is optional.
   - Timing is asked only where a mid-season change is unsafe.
5. **Presets as Go code.** `leaguerules/scoring_default32.go`, `leaguerules/mirror16.go`.
   - **Fix:** plain data bundles.
6. **Rule text gap.** CLAUDE.md:121 and North_Star.md principle 1 talk only about presets.
   - **Fix:** add one line: where a league came from is not a kind of league.

## Undone

- The 10-31 "mirror ledger" (hard-coded `native.mfl.14432`, writes refused) and the per-read MFL overrides, reverted in
  one commit.

## Order

1. Revert (done).
2. Gates (item 1) + rule text (item 6).
3. Import his league + one set of screens (item 3).
4. Settings sandbox screen (item 4).
5. Presets as data (item 5).
