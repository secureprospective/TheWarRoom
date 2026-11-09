# APP-GATES-4 — evidence and boundary review

## Implemented slice

- Import preparation writes `schedule.follow` (source, external league number, season) into the ordinary creation seed. The same catalogue, natural-value decoder, schema, rule authorization and `commish.rules.set` path allow a commissioner to set or clear it for any league.
- `FollowingLeague` resolves that setting only; neither origin nor the app league ID chooses following. A cutover contributes an empty setting through the season rule fold at its recorded instant, within the existing atomic standalone cutover act. A later ordinary setting can enable following again.
- Launch calls `BackfillFollowing` for older imports. It alone derives the missing setting from the recorded import and historical app ID. It records one optimistic ordinary rules command; any explicit seed or rule change, including a clear, prevents replacement. Failed writes are logged and do not pretend the migration succeeded. Origin allowlist addition is limited to this authorized compatibility initializer.
- Both War Room hosts get dates and failure from the calendar reader. The reference reader refuses failed clock provenance with its reason; the component no longer chooses a status source by host. Native fork ratchet tightened 19 → 18.
- Boundary closure additionally exposed two native dependency leaks. Shared roster counting moved unchanged from MFL-backed `m2service` into pure `powerrankings`; both consumers reuse it. Resolved `PlayerFacts` moved to `domain`, with a compatibility alias in `normalize`; `playerrecord` no longer depends on an MFL normalizer. No model/engine calculation changed. No fence/debt exemption was added. Tiny-file ratchet tightened 25 → 24.

## Verification

- `make placement`: PASS, 140 paths / 16 features; root Go files 1; native forks 18.
- `make boundary`: PASS, 169 frozen functions/sources. All 15 reviewer-approved fingerprint changes plus one new launch-backfill function below; no unrelated hash refreshed.
- `make lint`: PASS, 0 issues. An earlier attempt was blocked by another agent's active golangci-lint; no process was stopped and a later normal attempt passed.
- Final targeted package `go test -race -json`: 752 named test/subtest passes, 11 packages PASS, one helper package with no tests. Packages: commissioner, leaguerules, leaguemigration, leaguemigration/mflsource, commandcatalog, appvalue, m2service, powerrankings, playerrecord, domain, normalize.
- Composition migration/schedule batch: 4 tests PASS with `-race`. Enhanced backfill/set/clear/cutover test subsequently passed with `-race`: two backfill calls append once; ordinary clear survives another backfill; ordinary set restores following; cutover clears it; subsequent backfill does not restore it.
- Whole Vitest, once: 161 files / 1,751 tests PASS. After adding the calendar regression, its whole region file passed: 35 tests (reference success/failure plus existing hosted calendar assertions). No claim of a second whole run.
- Plain `make build`: PASS, compiled frontend/application and packaged desktop binary. Wails emitted existing missing-model notices; this is build evidence, not a UI/live-feed acceptance test. Wails regenerated `frontend/wailsjs/go/models.ts`; it was not hand edited.
- `heavy-ok.sh`: STOP (swap 83%, another full race suite running). No full-repo race suite was launched.
- Local code commits: `dc3191373` (A1), `cd0c2070b` (A2), `fde9ecbc8` (B). Every code commit passed normal hooks; no bypass or push.

## Residual debt / unknowns

The first broad composition package race run failed, before reaching a clean package result. Observed failures: `TestFrozenDoorMetadata` (schedule edit/generate/publish schema drift), `TestNativeStatFeedRevisionNeverMovesTheScoreOnItsOwn`, `TestNativeScoreKeepsTheCountedVersionUntilTheLeagueApprovesTheNewest`, `TestNativeStatQueueCM031ApprovalGovernsScoreAndQueue`, and `TestNativeStatPreviewStatesPointsAndFlipBeforeItFires` (panic after empty preview). These are not fixed or waived here; baseline reproduction was not performed, so their origin is unproven. The affected migration/schedule batch passes; the whole composition package is **not proven green**. Preset archive guard remains **not reproducible**, carried from prior handoff. No live hosted/desktop click-through or real external-feed test was run.

An initial `git pull --ff-only` attempt failed for lack of branch tracking, before the predecessor brief's local-only rule was loaded. No merge resulted; all produced commits remain local, and no push was attempted.

## Final fingerprint list

All paths below are relative to `internal/composition/`. Full accepted identities and hashes are in `docs/architecture/boundary-adapters.tsv`. Causes cite the already-reviewed branch wiring; A1's additions are explicitly identified.

- `refresh_app.go:(*App).refreshInBackground`
  - `a1b6da95ec86a35f391fabe880c9367e7c7611c480115a27cf9afdb67d4bb9ce`
  - 98ce88c3 (launch/sync schedule worker); dc319137 (launch backfill).
- `command_bindings.go:#source`
  - `5dfbdda823d9e270eaf67b1cc8e4d6c3c30567576685ed211b292cbf4dc9a4a1`
  - f5d0d806 (native value command wiring).
- `commissioner_host.go:#source`
  - `95ffc2f2e64fc951db5332d9c72cc35cc3349c418156a0365da4840c9f4e967f`
  - 31c71c7e (saved value preparation resource ports).
- `platform_host.go:#source`
  - `3be90aec8845c727fa3468667f1b54803726c5a9ed898dd3aea3a9a994d0d9fa`
  - 3318ec6c / d0479ccb (startup readiness and held league picker).
- `refresh_app.go:#source`
  - `0e98350f25349ec0f003be80dcd0818eb7dfa801241de02113087fadbc939c79`
  - 98ce88c3 (launch/sync schedule worker); dc319137 (launch backfill).
- `week_app.go:#source`
  - `c9b133bd2173d0f53bd921b9564914a1fe9e0c6afc93408d36c3d8d4cdbd85d1`
  - 7759a803 (missing NFL dates via shared feed).
- `migration_app.go:#source`
  - `3ff64f6667fe3b379e70dc858c0719d27f7bdf17a21c2952b83563fbc75301e2`
  - 55c05ef2 / 98ce88c3 / 2ec535d8 / 6692531b (migration, schedule, lineups, scoring acquisition); dc319137 (setting backfill / credential conversion).
- `migration_app.go:(*App).migrationInputs`
  - `b9a72d1f25fb0c9139659cab39dc089cb10c3a6e8ffd55aee4e11b19ac9160f1`
  - 55c05ef2 / 98ce88c3 / 2ec535d8 / 6692531b (migration, schedule, lineups, scoring acquisition); dc319137 (setting backfill / credential conversion).
- `commissioner_host.go:(*App).startValuePreparation`
  - `88373ea0a82e99a9c89a2d16d468a3461ccabd0d433ae15c26d7bb354c916adf`
  - 31c71c7e (saved value preparation resource ports).
- `commissioner_host.go:(*App).valueSource`
  - `635b05010e9ad1b5851a796ad2cfb97142fedb3235f6cb93e33a42e11fa45d98`
  - 31c71c7e (saved value preparation resource ports).
- `migration_app.go:(*App).heldLineupPlayers`
  - `f1aae9ee10510e60963a6ea265bc57f982bee5b365e6856d3cc99a6623791a52`
  - 55c05ef2 / 98ce88c3 / 2ec535d8 / 6692531b (migration, schedule, lineups, scoring acquisition); dc319137 (setting backfill / credential conversion).
- `migration_app.go:(*App).syncLeagueSchedules`
  - `3de8085d2cfb20d81110d326e79d0d01b02834784d38f605548d4b8ae5ab37e0`
  - 55c05ef2 / 98ce88c3 / 2ec535d8 / 6692531b (migration, schedule, lineups, scoring acquisition); dc319137 (setting backfill / credential conversion).
- `migration_app.go:(controlHost).Migrate`
  - `e02c4780e7ef94da2f14e2b3d93d1776b18702ba2f7ea38ae79612ef70a2d253`
  - 55c05ef2 / 98ce88c3 / 2ec535d8 / 6692531b (migration, schedule, lineups, scoring acquisition); dc319137 (setting backfill / credential conversion).
- `platform_host.go:(platformHost).Leagues`
  - `d90b34eb9ccebe4ad2eee2b7b202c4221f6052d7066e56b909093f37ed8728b0`
  - 3318ec6c / d0479ccb (startup readiness and held league picker).
- `week_app.go:(*App).fillScheduleDates`
  - `0a497d64cc5a6abe382d4ddd1ff29833ced524ba9ab147bc65d373fb93179cc4`
  - 7759a803 (missing NFL dates via shared feed).
- `migration_app.go:(*App).backfillScheduleFollowing`
  - `7d4df11fbed67f042256291167aa191cfa02a09dc22db4ceb3990187d713bcbf`
  - dc319137 (A1 launch-only compatibility initializer; resource enumeration/logging; decisions in leaguemigration).
