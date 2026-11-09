# TheWarRoom bone list — draft for Christopher, 2026-11-09

Measured on `session/merge-prep-1109` @ f28b449dc (PR #220 head). Estimates are from line counts and
the import graph (`go list -deps` for the three binaries), not from trial deletions.

## Totals (thousand lines)

| Area | Now | After (est.) |
|---|---|---|
| Go production | 141.6 | ~106 |
| Go tests | 136.6 | ~36 |
| Frontend source (35k generated) | 72.8 | ~50 |
| Frontend tests | 22.0 | ~6 |
| Go test data | 133.4 | ~35 |
| Frontend JSON fixtures | 101.3 | ~35 |
| Docs | 101.6 | ~25 |
| **All tracked** | **725** | **~295** |

## Key fact

`twr-server` links zero packages from `internal/app`, `internal/composition`, `internal/mflref`,
`internal/transactions`. The desktop links all of them (61 packages server, 118 desktop).
Desktop-only Go is about 42k production lines.

## KEEP

- Ledger and rules: `commissioner`, `store/history`, `store/state`, `leaguerules`, `governance`,
  `fantasyscore`, `nativecheck`, `leagueclock`, `leagueweek`, `leaguefeed`, `leagueload`.
- Commands and server: `commandcatalog`, `leagueservice`, `serverhttp`, `serverclient`,
  `serverstatus`, `serverversion`, `cmd/twr-server`, `cmd/twr`.
- Durability, privacy, accounts: `db`, `accounts`, `store/accounts`, `offsite`, `leaguetransfer`,
  `leaguerecovery`, `releasetrust`.
- Engines (ruled keep 11-09): `model`, `model/fit`, `modelrun`, `appvalue`, `powerrankings`,
  `m2service`, `measures`, `projection`, `pickvalue`, `lineupdelta`, `store/params`, `tools/fit`,
  `engine/*`, `app/scoring`, `app/sources`, and signal pipelines (`crosswalk`, `college`, `feeds`,
  `players`, `playerscores`, `leaguearchive`, `archive`, `playerrecord`).
- MFL import: `leaguemigration` + `mflsource`, `mfl`, `mflkey`, `normalize`, the `ingestion`
  packages the server links.
- Desktop host that stays: `app/core`, `app/platform`, `app/control`, `app/commissioner`,
  `app/season`, `composition` (shrinks as cut features leave).
- Frontend: shell, Home, Franchise HQ, War Room, League Pulse, Control Room, commissioner hosted
  screens, draft room, command registry, generated bindings.

## CUT

- MFL-as-MFL path: `app/mflref` 6.2k, `mflref` 2.2k, `transactions` 2.1k, `store/mflref` 0.6k,
  `snapshot` 0.7k, `store/rulebook` 0.8k, MFL live-mirror feeds ~1.2k, `app/offers` 1.5k, plus tests.
- Frontend non-native side of all 19 `native ?` forks, MFL connection, old endpoint registry
  (`registry/`, 6.8k), specimen, MFL command tables.
- Dead and stray: ~750 lines dead Go, `playeridentity`, `seams`, `teamneeds`, migration assistant,
  `tools/d11replay`, `d27replay`, `d33replay`, fixtures compiled into release binaries.
- Tests: all but the KEEP set (replay parity, privacy, durability, rosters/contracts, auth, reply
  cache). Composition tests 28.5k go nearly whole.
- Test data: `testdata/wire` golden snapshots (~72k), frontend test-only JSON (~77k), about half of
  package testdata.
- Docs: old mock versions (keep latest of each), spent handoffs, plans, resumes, audits, wireframe
  briefs.
- Comments: about 2k lines of narration to get under 5%.

## REBUILD-LATER (delete now, comes back thin)

- Research: `app/research` 2.8k, `researchfeeds`, `injuries` (ruling 10-28).
- Now screen and live game feeds: `app/now` 2.7k, `livegame` 1.3k, `livestats`, `livescoring`.
- Market and draft reports: `app/market` 1.4k, `marketfeeds`, `app/draft`, `draftreports`.
- Race: `app/race`, `engine/race` stays.

## FLAGS

1. What-if desk (trade builder, release plans, bid plans on the MFL-mirror league) is cut with
   `transactions`. README features it.
2. Now, Market, Race screens leave the desktop until rebuilt.
3. Go production only falls ~25% this session. The rest needs the desktop rewired onto the command
   registry the server already uses; that is rebuild work, not deletion.
4. 23.8k lines of JSON under `frontend/src/app/data/fixtures` are imported by shipping code
   (`data/provider.ts`). Needs a look before cutting.
5. `commissioner` (27k) and `store/history` (10.5k) are kept nearly whole because replay parity is law.
6. `teamneeds` and `model/fit` are linked by no binary; `model/fit` kept as an engine tool,
   `teamneeds` cut.

## REVISION 1 — Christopher's ruling, 2026-11-09 (supersedes the CUT and FLAG entries it touches)

Ruling: the project moved from MFL mirror to MFL→Native import. The MFL mirror stays ONLY as the
solo-owner framework and is kept simple. The import must be flawless for any league size or type;
the goal is to pull in as many leagues as possible.

- MFL-as-MFL path (`app/mflref`, `mflref`, `transactions`, `store/mflref`, `snapshot`,
  `store/rulebook`, mirror feeds, `app/offers`): moves CUT → KEEP-SIMPLE. Solo-owner tools only;
  trim, do not delete. What-if desk, Now, Market, Race stay as solo-owner tools.
- Research stays REBUILD-LATER (ruling 10-28) unless told otherwise.
- MFL→Native import (`leaguemigration`, `mflsource`, server-linked `ingestion`) joins the test KEEP
  set as a first-class item: league size and league type coverage.
- `native ?` forks: now a legitimate solo/native seam; target is one seam at the mount, not 19 inline forks.
- Revised estimates (k lines): Go prod 142 → ~128; Go tests 137 → ~40; frontend 73 → ~62;
  frontend tests 22 → ~7; data 235 → ~75; docs 102 → ~25. Code alone 373 → ~237 (-36%);
  all tracked 725 → ~337 (-53%).

## REVISION 2 — Christopher, 2026-11-09: PWA-centric

The PWA (the same React app served by `twr-server` to a browser/phone) is the main way people use
TWR. The desktop app stays first-class and supported. Source in repo:
`docs/plans/CLIENT-SERVER-PLAN-2026-11-05.md` (rows "Runtime" and "4 — Complete PWA, not another application").

- Where a desktop-only path and a hosted/browser path do the same job, the hosted path is the bone
  and the desktop one is the candidate to thin.
- KEEP explicitly: `serverhttp/browser.go`, cookie/CSRF/session code, the browser command catalogue
  (`commandcatalog/browser.go`, `commands/browser.gen.ts`), SSE/invalidations, the reply cache, CSP checks.
- Frontend screens are shared by both; nothing is cut for being "not desktop".
- Step 5 proof adds: the PWA loads from the lab server in a phone-width browser, logs in, reads owner
  screens, previews and executes a lineup.

## REVISION 3 — the frame, Christopher 2026-11-09: AI/CLI-first

TWR houses a league and its human interaction as rule-set data; an AI agent does everything through
the CLI; screens and minigames are outputs. 337k is still too much under this lens.

Measured (production Go lines linked, `go list -deps`): `cmd/twr` 86.3k, `cmd/twr-server` 89.0k,
desktop 129.7k, union 134.5k. The CLI already exists as a thin client: ~100 generated verbs,
preview/execute, MCP mode, over the 137-command catalogue.

League-shape code found inside the rules/ledger core (file-name families, production lines):
sides/mirror 466, O-Line card 518, conferences/brackets/playoffs 1,054, presets/seeds 934;
`maxFranchises = 32` in `commissioner/creation.go`. Rules are a typed Go struct (`leaguerules.RuleSet`),
so each new rule kind is code in ~20 files. NOT YET DONE: end-to-end read of the import and
`commissioner` by family, and the git history of the week that built per-league-type code.
