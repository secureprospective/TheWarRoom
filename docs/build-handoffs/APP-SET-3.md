# APP-SET-3: presets on existing leagues (compare, apply)

## Baton
Tom → integrator, 2026-11-03, `build/SET-3`, base `build/INTEGRATE-1102` @ `8c15fe6c`. Local only.
Ruling (Christopher 11-02) implemented. It supersedes the 10-27 "preset only at creation" rule.

## What was added
- **Compare** (`internal/commissioner/preset_compare.go`, `PresetCompare`). It reads the league's rules in force
  and the preset's bundle at the league's team count, both through `leaguerules.Catalogue/ReadSettings/Current`,
  the same catalogue the rules screen uses. For each differing setting it returns the setting, the league
  value and the preset value. With no differences it says "This league already matches the … on every
  setting." It skips preset values a hand edit cannot send: null, plus counted fields (alignment, sides,
  startup list). Only presets that carry values can be compared: 32-Team Default, 16-Team Mirror and Legacy NFL.
- **Apply** (`PresetRuleChanges` → `leagueservice.ExecuteRules` → `Store.ExecuteRulesBatchWithFacts`). The
  chosen fields (or all of them) become ordinary `commish.rules.set` commands, one per setting, in catalogue
  order. They are recorded atomically through the existing commissioner batch loop: the same Prepared,
  Validate, credential admission, authorizer, `checkAppend` (rules, shape, sides, draft…) and gravity as a hand
  edit. A rules-only batch now also gathers each step's own `appendFacts`, as a single execute does. There is
  no per-preset code path. A dry run returns the changes, and the UI previews each one through the ordinary
  `TargetCommissionerPreview`. The confirm uses the strictest safeguard Go names among them
  (confirm/hold/reason).
- **Registry**: `TargetCommissionerPresetCompare(league, preset)` is a logical read
  `commish.rules.preset.read` (league-view, gm/commish), beside `commish.rules.read`.
  `TargetCommissionerPresetApply(body)` is a commish G3 generic door that dispatches `commish.rules.set`
  (listed with Execute/Preview as generic). doors-before.json gains 3 rows (294→297, parity test bumped).
  The release matrix inventory and the api.test method count (55→57) are updated. nativegen and wails
  bindings were regenerated and are fresh.
- **UI**: Control Room → RULES, a "COMPARE WITH A PRESET ›" button at the end of the filter chips. It opens
  `control/room/PresetCompare.tsx` in the inspector. You choose a preset, toggle any setting on or off (all are
  on by default), and pick when it takes effect: now-only settings force Now, as on the hand form. Type a
  reason, preview the changes, then record them. The stale "presets only at creation" text in the default
  inspector was corrected.

## Refusal reuse
The mid-season refusal is the hand edit's own check (`checkPoolsChange`/`playBegun` etc. via `checkAppend`).
Apply wraps it as "league change N of M: <same text>". A refusal leaves nothing recorded. The test proves the
preset error ends with the hand error, with an identical `Refusal`.

## Tests (`internal/store/history/preset_apply_test.go`, `frontend/.../presetCompare.test.tsx`)
- Native Default32 vs Default32: no differences. Imported Legacy-32 vs Default32: a pinned 18-field list
  in plain words.
- Apply subset (3 fields, next rollover): receipts and full ledger records are `reflect.DeepEqual` to the same
  edits made by hand on a twin store.
- Mirror16 with a generated schedule: Player pools is refused identically to the hand edit. Revision is
  unchanged. The rest applies.
- Another league's rules, ledger and comparison readings are byte-identical before and after an apply.
- An owner is refused identically. A non-rules command is refused in a rules batch.
- Vitest: the list, a subset apply after preview, the no-differences case, and a refusal shown word for word.

## Verify
- `-race -p 2`: store/history 25 PASS; commissioner, app/commissioner, commandcatalog and leagueservice preset,
  catalog and registry: 38 PASS; composition parity, registry and matrix: 6 PASS.
- golangci-lint on the touched packages: 0 issues.
- vitest control/room + commissioner: 49 files / 519 PASS. `tsc --noEmit` clean. nativegen -check fresh.
- `CI=true make build` OK.

## Debt (not caused here; all fail identically on base 8c15fe6c)
- `history.TestCommissionerPresets`: an empty league reads `league.startingAssets` as Set (SET-1 fallout).
- `composition.TestCommissionerWireMatchesGolden` (controls.json, rules.json) and
  `TestCommissionerReplacedRulesWire`: stale wire goldens.
- `scripts/check-frontend-transport.sh`: roomReads.ts:2 and localWorkflow.test.ts:63/78.
- Hosted door: `ExecuteRules` needs a host with `ExecuteRulesBatchWithFacts`. The hosted footballHost does not
  have one yet and refuses in plain words.
- The preview shows each change against the current ledger, not chained. The record step checks each change
  against the batch-so-far.
