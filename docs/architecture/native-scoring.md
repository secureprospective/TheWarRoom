# Native scoring: the 32-team default as commissioner settings

## Resume here

Branch `session/revamp-D-scoring-engine` (stacked parts, PR only at the final part).

- **BLOCKER (part 1 end, not pushed):** `leaguerules.RuleSet` crosses the Wails binding, so the five
  new `ScoringRules` fields regenerate `frontend/wailsjs/go/models.ts` (+246 lines) and break seven
  wire fixtures: `internal/app/commissioner` `TestCommissionerControlsWire`, and in
  `internal/composition` `TestCommissionerReplacedRulesWire`, `TestCommissionerLegacyRulesWire`,
  `TestCommissionerRulesWire`, `TestCommissionerJobsWire`, `TestNativeScoreWireFixtureMatchesGo`,
  `TestNativeStatQueueWireFixtureMatchesGo`. The part-1 brief forbids binding changes and
  composition edits, so the push gate failed and the branch is local only. Needs Christopher's
  ruling: (a) accept the additive binding/wire change and regenerate after branch B lands, or
  (b) keep the new kinds off the wire shape. (a) is the design's intent: they are commissioner
  settings, so the UI must see them.

- **Part 1 (2026-10-26), done:** five new scoring rule kinds as commissioner settings in
  `internal/leaguerules`, the scorer for them in `internal/fantasyscore`, the locked DEFAULT-32
  spec shipped as `leaguerules.Default32Scoring()`, table tests for every spec row.
- **Golden, done:** 21 real 2024 player-weeks (every position, Barkley, Chase, McBride, Boswell,
  the 98-yard Westbrook-Ikhine week …) match the reference scorer to the cent
  (`internal/fantasyscore/default32_golden_test.go`, fixture `testdata/default32_2024.json`).
- **Held adapter, done:** `internal/fantasyscore/source` now emits `fg_made_distance` (weekly
  `fg_made_list`), `fg_missed_distance` (`fg_missed_list` + `fg_blocked_list`) and `chunk_play`
  (every counted rush and catch distance from play-by-play, same credits as Legacy's long plays).
  Every held week scores under the default (`TestHeldWeeksScoreUnderTheDefault`).
- **O-Line scorer golden, done:** all 1,164 played 2023–25 team-games (weeks 1–13) from the study's
  frozen components score exactly its Set B values; the clamp binds the reported 16 times
  (`internal/fantasyscore/oline_golden_test.go`).
- **O-Line held feed, done:** `source.ReadLinePlays` / `ReadLineRushing` / `LineGames`
  (`internal/fantasyscore/source/line.go`) build each team-game's `UnitLine` from held play-by-play
  plus PFR yards before contact, counting only players the caller's `isRB` says the league lists at
  RB. Two real 2024 games (`testdata/line_plays.csv`, `line_rushing.csv`) reproduce the study's
  components and Set B scores; a table test pins each §4.6 definition (kneels, laterals, defensive
  TDs, 3rd/4th and ≤ 2, both hitters, deleted and 2-point plays). A game with no advanced rushing
  rows yet leaves ybc absent, so the score is "not recorded yet", not a guessed zero.
- **Next part starts at:** wiring. (a) Store: the history store (`internal/store/history/
  football_stats.go`) must hold and serve the line plays and advanced rushing, and the
  per-player RB designation for `isRB`. (b) A league choosing the default at creation
  (`internal/commissioner/creation.go` seeds `LegacyNFLForTeams`): a `Default32` preset, with the
  spec's lineup (§1), roster 44/8/12, taxi and IR (§2). (c) The O-Line unit as a rosterable,
  startable thing (§3), which needs a position or slot kind the model does not have yet; that is
  a model change to plan, not to slip in. (d) Live feeds (spec §5/§7.1) are not mapped.
- **Open rulings** are listed at the bottom; do not resolve them by guessing.

## Owner framing

The 32-team default is a default scoring setting inside the existing commissioner rules. It is not
a separate engine. Everything below extends `leaguerules.ScoringRules` (data) and
`fantasyscore.Score` (the pure scorer). Neither package imports composition, Wails or a store.

## What already existed

| Piece | Path | Role |
|---|---|---|
| Rule data model | `internal/leaguerules/scoring.go` | `ScoringRules{Catalogue, RoundStatPoints}`; `Stat` = rate, flat-or-not (`Once`), positions, per-play threshold `Bonuses` |
| Rule ledger | `internal/leaguerules/resolve.go`, `rules.go` | every `ruleField` is a commissioner setting, seeded explicitly per league, changed by `Change` |
| Scorer | `internal/fantasyscore/score.go` | `Score(rules, position, StatLine)` → points + per-stat `LineItem` breakdown, exact ten-thousandths |
| Held nflverse adapter | `internal/fantasyscore/source/` | weekly stat lines + PBP distances/return TDs for Legacy |
| MFL rulebook mirror | `internal/store/rulebook` | raw MFL scoring rows (MFL-era, not the native engine) |
| MFL-era model | `internal/app/scoring` | ranking-side model, not the rules engine; untouched |

Callers: `internal/commissioner/player_score.go`, `stat_queue.go`, `stat_check.go`.

## Design

### Rule kinds (each its own commissioner setting)

New fields on `ScoringRules`, each a `Value[...]` with its own field path, so the commissioner edits
them like any other rule and Legacy records each as an empty, cited value (`internal/leaguerules/scoring_kinds.go`):

| Field path | Type | Expresses |
|---|---|---|
| `scoring.extras` | `map[StatID][]PositionExtra` | additive per-position rates on a catalogue row (TE catch +0.2, DT/DE tackle +1, CB/S IC +1 …). Each extra is its own line item, rounded on its own, as MFL rounds overlapping rows. |
| `scoring.ranges` | `map[StatID]StatRange` | MFL's inclusive range: a week value outside it scores 0 for that rule (TCH 20..99, OFC 1..10, yards −50..999). |
| `scoring.scales` | `map[StatID]DistanceScale` | per-play distance pricing: plays shorter than `From` score 0, others `Base + Above·max(0,d−Pivot) + Below·max(0,Pivot−d)`, optional `Cap`. The week's sum is the stat's value, priced by its catalogue rate (×1) and rounded once. Chunk (From 20, Base 1, Pivot 20, Above 0.025), FG made (Base 3, Pivot 40, Above 0.1), FG missed/blocked (Pivot 60, Below −0.1), 40+ completions (From 40, Base 1). |
| `scoring.totals` | `map[StatID][]StatID` | a stat counted as the sum of recorded stats (touches = carries + receptions). Absent when any part is absent; recording it directly is an error. |
| `scoring.lineUnit` | `LineUnit` | the team O-Line unit: `Weights` per component, `NoSack` bonus keyed to the `Sacks` component, `Clamped`/`Clamp`, `Labels`. |

Legacy is unchanged (golden `internal/leaguerules/testdata/legacy.json` grew only by the empty
fields; the two commissioner wire fixtures grew by the same fields plus their rules digest).

### The default rule set

`leaguerules.Default32Scoring()` (`internal/leaguerules/scoring_default32.go`) is the spec's §4
as data, cited "DEFAULT-32 spec §4, locked 2026-10-26": Legacy's IDs where the rule is the same
stat, new IDs where the stat is new (`chunk_play`, `touches`, `fg_made_distance`,
`fg_missed_distance`, `blocked_fg_td`, `blocked_punt_td`, `line_*`). Removed Legacy rows (R20/R40/
C20/C40, the FG buckets, MG) are simply absent, so a recorded bucket scores nothing.

### Inputs

- Player-week: `fantasyscore.StatLine` — `Quantities` per stat (ten-thousandths, so half sacks
  work) and `Events` per stat = the distance of every play/kick (chunk plays, made FGs, missed and
  blocked FGs, completions for 40+).
- Team-game: `fantasyscore.UnitLine{Played, Quantities}` with the seven `line_*` components.
  `Played=false` (bye / no game) scores 0. An absent component (e.g. ybc not yet published) makes
  the result Needs-a-ruling with the reason, never a guessed zero.

### Outputs

`(Points, []LineItem, error)`. Each `LineItem` carries the stat, the rule row that priced it
(`Rule`), the quantity, the label, the points, and flags `Extra`, `OutOfRange`, `Unused`. For the
O-Line, the no-sack bonus and the clamp adjustment are their own lines so the breakdown always
sums to the total.

### Arithmetic

Exact integer ten-thousandths throughout (spec §7.3: no floats). Each item is rounded to the cent
when `RoundStatPoints` is on; per-play values are summed for the week before rounding (§4.1).

## Golden test

Real 2024 player-weeks scored by the reference `legacy_scorer.score` under
`study4c.VARIANTS["C2 continuous + dyn kicker"]`, with per-play chunk distances pulled from
`play_by_play_2024` (rusher yards; receiver yards where `complete_pass = 1`) and FG lists from the
weekly `fg_made_list` / `fg_missed_list` / `fg_blocked_list`. Fixture
`internal/fantasyscore/testdata/default32_2024.json`; each row cites week, gsis and MFL id. The
extractor (kept out of the repo, run once with `python3 -I` against the study's held data) dropped any
row whose per-play chunk sum differed from the study's `CK2` or whose 40+ completions differed from
weekly `passing_40`; none did. The fixture keeps the study's event codes, and the test fails on any
code it does not map, so a silently dropped stat cannot pass.

## Needs a ruling

1. **Negative half-cent rounding.** Spec §4.1 says half-up toward +∞ (−0.125 → −0.12, as the
   reference `floor(x·100+0.5)/100`). The existing `roundCent` rounds half away from zero
   (−0.125 → −0.13) and `TestRoundsEachStat…` pins `-250 → -300`. Unreachable under the default
   (no negative multiplier yields a half-cent from integer inputs), but reachable under Legacy for
   negative odd return yards ×0.025. Left unchanged.
2. **Existing seeded leagues.** A ledger seeded before these fields existed resolves them as
   Needs-a-ruling, and the scorer then reports the whole score as Needs-a-ruling. Options: a seed
   migration that records Legacy's empty values, or the commissioner records them. Not chosen here.
3. **EM range.** The spec's EM row is one rule over missed + blocked PATs (range 0..10); the
   catalogue keeps `pat_missed` and `pat_blocked` as two rows each ranged 0..10. They differ only
   above 10 combined. Could be expressed exactly with a `totals` stat; not done without a ruling.
4. **#UT "other return TD"** (e.g. missed-FG return) is mapped to `punt_return_td`; the source
   adapter does not yet classify missed-FG returns.
5. **Legacy ranges.** Legacy's Go preset never applied MFL ranges (a −51-yard week scores
   −2.55, not 0). The default applies them. Whether Legacy should too is a parity question.
6. **TE catch extra range.** The study's TE +0.2 row is ranged 0..999; the base CC row 0..99. In
   this model an extra shares its base row's range. They differ only above 99 catches.
7. Spec §7 items (live mapping, ybc provisional score, stat corrections, EDGE designation) remain
   open as written there.
