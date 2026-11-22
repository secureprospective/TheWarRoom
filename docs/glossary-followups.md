# Glossary Follow-ups

Places in the UI, code, CLI, and docs that still use an avoided word from [GLOSSARY.md](../GLOSSARY.md). This is a to-do list for later code/copy changes; do not change code in the glossary commit.

## 1. Co-commissioner / Co-commissioner order (avoid: `deputy priority`, `deputy order`, `commish.deputy.order`, `firstDeputy`)

| File:Line | Current word | Canonical word |
|---|---|---|
| `cmd/twr/verbs.gen.go:99` | `commish.deputy.order` / `"order league deputy"` | Co-commissioner order |
| `internal/commissioner/people.go:14` | `DeputyOrder = "commish.deputy.order"` | Co-commissioner order |
| `internal/governance/state.go:226` | `firstDeputy` | Co-commissioner order (`firstCoCommissioner`) |
| `internal/governance/succession.go:136` | `firstDeputy` | Co-commissioner order |
| `internal/governance/elections.go:150` | `firstDeputy` | Co-commissioner order |
| `internal/governance/policy.go:94` | `firstDeputy` | Co-commissioner order |
| `frontend/src/app/commissioner/actTable.ts:75` | `commish.deputy.order` | Co-commissioner order |
| `frontend/src/app/commissioner/HostedControlRoomPeople.tsx:183` | `commish.deputy.order` | Co-commissioner order |
| `frontend/src/app/commissioner/HostedControlRoomPeople.test.tsx:206` | `commish.deputy.order` | Co-commissioner order |
| `frontend/src/app/control/room/PeopleView.tsx:186` | `DEPUTY ORDER` / `commish.deputy.order` | Co-commissioner order |
| `frontend/src/app/registry/commissioner.gen.ts:654` | `commish.deputy.order` | Co-commissioner order |
| `frontend/src/app/commands/native.gen.ts:4343,4355,7872-7880` | `commish.deputy.order` | Co-commissioner order |
| `internal/app/commissioner/testdata/release-mutations.json:1607` | `commish.deputy.order` | Co-commissioner order |
| `docs/commissioner/README.md:271` | `Native deputy priority` | Co-commissioner order |
| `docs/ui/control-room/Control_Room_Tree_v1.md:12,141,236` | `deputy priority` / `commish.deputy.order` | Co-commissioner order |
| `docs/ui/data-catalog/claude-catalog-2026-10-30.md:364` | `deputy order` / `DEPUTY ORDER` | Co-commissioner order |
| `docs/ui/endpoint-registry.csv:292` | `commish.deputy.order` | Co-commissioner order |
| `docs/ui/mocks/control-room-mock-v1.html:843,876` | `Deputy order` / `DEPUTY ORDER` / `commish.deputy.order` | Co-commissioner order |
| `docs/ui/v3/v3-stubs.json:931,3043` | `commish.deputy.order` | Co-commissioner order |
| `docs/ui/v3/V3-STUB-INVENTORY.md:229,353` | `commish.deputy.order` | Co-commissioner order |
| `docs/ui/v3/UIUX-INVENTORY-2026-11-17.md:73` | `commish.deputy.order` | Co-commissioner order |

## 2. Twin-pool league (avoid: `Mirror league`, `16-Team Mirror`, `per-side`)

| File:Line | Current word | Canonical word |
|---|---|---|
| `internal/commissioner/presets.go:20,67-73` | `16-Team Mirror` (`PresetMirror16`) | Twin-pool league |
| `internal/leaguerules/mirror16.go:9-14,61,73` | `16-Team Mirror` | Twin-pool league |
| `internal/leaguerules/mirror16_test.go:8` | `16-Team Mirror` | Twin-pool league |
| `internal/engine/rookiedraft/startup_test.go:228` | `16-Team Mirror` | Twin-pool league |
| `internal/store/history/per_side_restore_test.go:14` | `16-Team Mirror` | Twin-pool league |
| `frontend/src/app/control/MoveMFL.tsx:62` | `16-Team Mirror` | Twin-pool league |
| `docs/ui/control-room/Control_Room_Tree_v1.md:158` | `16-Team Mirror` | Twin-pool league |
| `docs/ui/data-catalog/claude-catalog-2026-10-30.md:339` | `16-Team Mirror` | Twin-pool league |
| `docs/ui/data-catalog/terra-CR.md:195` | `16-Team Mirror` | Twin-pool league |
| `docs/ui/mocks/control-room-mock-v1.html:737` | `16-Team Mirror` | Twin-pool league |

## 3. Cap schedule / Dead-cap schedule & Trade desk (avoid: `salary ledger`, `dead-cap ledger`)

| File:Line | Current word | Canonical word |
|---|---|---|
| `internal/store/state/ledger.go:84` | `per-year salary ledger` | Cap schedule |
| `docs/transactions/Salary_Ledger_Design.md:30` | `per-year salary ledger` | Cap schedule |

## 4. Dead cap (avoid: `dead money`, `dead_money`)

| File:Line | Current word | Canonical word |
|---|---|---|
| `frontend/src/app/shell/HQPanels.tsx:198` | `dead money included in runway` | Dead cap |
| `frontend/src/app/shell/hqModel.ts:45` | `dead money` | Dead cap |
| `frontend/src/app/commissioner/hqReading.ts:765` | `What-if dead money` | Dead cap |
| `internal/commissioner/native_cap.go:83` | `dead money` | Dead cap |
| `internal/commissioner/native_runway.go:13` | `dead money` | Dead cap |
| `internal/commissioner/native_seed.go:102` | `"dead_money"` / `dead money` | Dead cap |
| `internal/commissioner/rollover_carry.go:12,65,84,91` | `"dead_money"` / `dead money` | Dead cap |
| `internal/app/commissioner/native_legality_api.go:73` | `dead money` | Dead cap |
| `internal/store/state/schema.go:128` | `dead money columns` | Dead cap |
| `internal/store/state/migration_test.go:87` | `dead money columns` | Dead cap |
| `docs/commissioner/README.md:238` | `dead money` | Dead cap |
| `docs/architecture/cap-runway.md:20,38` | `dead_money` / `dead money` | Dead cap |
| `docs/architecture/hq-wire.md:13` | `dead money` | Dead cap |
| `docs/architecture/native-moves.md:39` | `dead money` | Dead cap |
| `docs/architecture/system-narrative.md:168,683` | `dead money` | Dead cap |
| `docs/plans/FUTURE-PLATFORM-DIRECTION-2026-11-08.md:55` | `dead money` | Dead cap |
| `docs/research/tiny-models-2026-11-08.md:10` | `dead money` | Dead cap |

## 5. Cut / Vacate seat (avoid: `Release a player`, `join-link -release`)

| File:Line | Current word | Canonical word |
|---|---|---|
| `frontend/src/app/commissioner/HostedWorkflows.tsx:55` | `Release a player to waivers` | Cut |
| `frontend/src/app/shell/WarRoomMarket.tsx:291` | `Release a player if the roster is full` | Cut |
| `frontend/src/app/shell/WarRoomMarket.test.tsx:134` | `Release a player` | Cut |
| `cmd/twr-server/join_link.go:36` | `-release FRANCHISE` | Vacate seat |
| `cmd/twr-server/README.md:348,359` | `-release franchise.07` | Vacate seat |
| `docs/resume/RESUME-server-lab.md:147` | `join-link -release` | Vacate seat |
| `docs/resume/RESUME-storage.md:562` | `join-link -release` | Vacate seat |

## 6. Native league / Thin client (avoid: `app league`, `App league`, `Hosted (thin)` / `-native` client confusion)

| File:Line | Current word | Canonical word |
|---|---|---|
| `cmd/twr/schedule_backfill.go:22,59` | `app league` | Native league |
| `internal/commandcatalog/workflow.go:54` | `app league` | Native league |
| `internal/commissioner/migration.go:12` | `app league` | Native league |
| `internal/commissioner/native_reading.go:155,157,186` | `app league` / `App league` | Native league |
| `internal/commissioner/native_cap.go:56` | `app league` | Native league |
| `internal/commissioner/native_directory.go:43` | `App league` | Native league |
| `internal/commissioner/calendar.go:90` | `app league` | Native league |
| `internal/commissioner/schedule.go:98` | `app league` | Native league |
| `internal/commissioner/bracket_reading.go:103` | `app league` | Native league |
| `internal/commissioner/standings_reading.go:94` | `app league` | Native league |
| `internal/commissioner/rules_reading.go:70` | `app league` | Native league |
| `internal/commissioner/notes_reading.go:70` | `app league` | Native league |
| `internal/commissioner/player_score.go:154` | `app league` | Native league |
| `internal/commissioner/corrections_reading.go:34` | `app league` | Native league |
| `internal/commissioner/preview_operations.go:41,114,225,229,325` | `app league` | Native league |
| `internal/app/platform/api.go:34,37,91,102` | `app league` / `App leagues` | Native league |
| `internal/app/commissioner/commissioner_api.go:71` | `app league` | Native league |
| `internal/app/commissioner/commissioner_preview_api.go:66` | `app league` | Native league |
| `internal/app/commissioner/native/facts.go:93,106` | `app league` | Native league |
| `frontend/src/app/commissioner/ActionTray.tsx:10` | `Start an app league` | Native league |
| `frontend/src/app/commissioner/ControlGroupView.tsx:29-30` | `Start an app league` / `Start another app league` | Native league |
| `frontend/src/app/commissioner/LeagueLanding.tsx:118` | `app league` | Native league |
| `frontend/src/app/commissioner/LeagueLedger.tsx:43` | `app league` | Native league |
| `frontend/src/app/commissioner/StartLeagueForm.tsx:10,47` | `app league` | Native league |
| `frontend/src/app/commissioner/proxy.ts:84,210` | `App leagues` / `app league` | Native league |
| `frontend/src/app/draftroom/DraftRoom.tsx:75` | `app league` | Native league |
| `frontend/src/app/control/room/ControlRoom.tsx:58` | `app league` | Native league |
| `frontend/src/app/control/room/EditorTray.tsx:65` | `Start an app league` | Native league |
| `frontend/src/app/shell/HomeBrief.tsx:343` | `app league` | Native league |
| `frontend/src/app/shell/homeNeedsYou.ts:123` | `native app league` | Native league |
| `frontend/src/app/shell/WarRoom.tsx:101` | `app league` | Native league |
| `frontend/src/app/shell/warroomLens.ts:79` | `app league` | Native league |
| `frontend/src/app/shell/leaguePicker.interaction.test.tsx:18,41,45,81` | `App league` / `app league` | Native league |
| `docs/architecture/desktop-modes.md:13-16` | `Thin (hosted)` / `-native` | Thin client |
| `docs/architecture/system-narrative.md:105,168` | `app league` | Native league |

## 7. Items from the v3 branch (reported in seed draft, not yet in this worktree)

- `frontend/src/app/v3/HQContractsV3.tsx:47,123-125` — `"Release a player"` → **Cut**
- `frontend/src/app/v3/HQContractsV3.tsx:112` — `"Dead-cap ledger"` → **Dead-cap schedule**
- `frontend/src/app/v3/HQContractsV3.tsx:113` — `"No dead money on the books"` → **Dead cap**
- `frontend/src/app/v3/WarTradeV3.tsx:2,60` — trade desk `"YOU SEND / YOU GET"` columns called ledgers → **Trade desk** (**You send** / **You get**)
