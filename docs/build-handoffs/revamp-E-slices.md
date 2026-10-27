# Revamp E — executable slices (plan only, 2026-10-27)

## Status and first move

No implementation in this change. Execute E01–E20 (including a/b sub-slices) **serially**, one commit per slice;
each must compile and pass its targeted tests before handing off. Times are estimates,
not measured performance. SOL means genuinely cross-cutting or arithmetic/state risk;
TERRA means a bounded literal implementation. Do not combine slices to save dispatches.
Read this document's contracts before implementing an individual slice.

The new ruling replaces HANDOFF.md's part-1 blocker: weekly RTL, prior-season fixed
position rate, no carry floor; −0.3124 is expressly allowed. Do not build a settlement,
current-season rate, rolling rate, or retroactive seed upgrade to Package A.

Source-traced base: branch `session/revamp-E-edge-scoring`, D base `391933f3`, part-1
handoff `6dd43bea`. `git pull --ff-only` refused (no upstream); no alternate pull,
rebase or push attempted. The requested part-1 stdout.log is **empty (0 bytes)**;
HANDOFF.md and `.scratch/revampE/reproduction.log` are the available part-1 evidence.
No test suite ran for this planning task. The arithmetic command in §Golden actually ran.

**One owner question before E10/E19 acceptance:** which source defines the *prior*
season's TFL numerator? Recommendation and computed reference below use complete prior
REG nflverse PBP TFL / nflverse weekly carries, at the league's prior-season positions.
Current-week RTL counts remain Sleeper `rush_tkl_loss`, as the edge-stat study did.
These sources are not identical, especially for QBs. The ruling fixes time and strength,
not this source choice. Do not silently call this the Sleeper baseline. If the owner wants
Sleeper for both, obtain held full-season prior Sleeper bodies and recompute §Golden;
only weeks 1–13 are held here. E01–E09 and model/engine work can proceed independently.
The five totals below are exact **conditional reference targets**, not claimed accepted
production baselines or a passing Go golden. This is an explicit blocker, not permission
to fill missing weeks with zero.

## Data flow and owners

| Responsibility | Contract / effects / failure |
|---|---|
| `source.ReadPlays` → `AttachPlays` | Validated held PBP → X3/X4/Q3/Q4/RZA/RZT/RZP quantities on existing player weeks. Pure, no network/store. Missing columns remain absent, not zero. |
| New `source.ReadSleeperEdges` → `AttachSleeperEdges` | Typed Raw JSON, scoped season/week and checked Sleeper→GSIS bridge → only the 14 non-PBP edge quantities including raw rushing TFL. Pure. No name joins or team rows used as players. |
| `history.RawFootballStats` / `RawFootballWeek` → `scoringFile` | Embed the exact Sleeper bytes and bridge in the pinned revision/week part. Validate before attaching. Existing outer evidence SHA / part SHA / revision SHA / LineDigest chain includes additions. No archive-retention dependence. |
| New offline `cmd/rtl-rates` → rate artifact | Read held files only; complete prior-season coverage and league-position crosswalk → deterministic integer TFL/carries per position with source hashes, coverage and exclusions. No DB writes or network. |
| `history.RawFootballEvidence` → pinned baseline artifact | Embed the small artifact, verify canonical digest and target/prior season. Rule admission must compare the rate values with this artifact; a caller-supplied digest alone is not proof. |
| `leaguerules` → `commissioner` → `fantasyscore.Score` | Versioned per-season/per-position rational rates + coefficient → exact single weekly RTL item. Rule admission freezes existing seasons before kickoff. Engine has no store/I/O. |
| `NativePlayerScoreWith` / `withCorrections` | Carry known week season into the line; corrected carries/TFL recompute RTL with the same frozen rate. Existing score-delta and counted-version semantics remain. |

**Source gaps today (do not assume capabilities):** RawPlay has no conversion/RZ fields;
Sleeper scoring adapter/bridge/body fields do not exist; Score has no season context;
`multiply` refuses inexact ten-thousandths (it cannot price a rational-rate residual);
ScoringRules has no rate kind; CheckRulesAppend has no kickoff or pinned-baseline argument;
`lineupFactsFor` currently gathers facts only for lineup/playoff commands. The generic
rules UI's ordinary `now`/`next_rollover` policy is **not** a historical-rate freeze.
The fetch-archive `Store.PlayerWeekStats` holds no PBP, and must continue saying incomplete.
This plan extends pinned imports, not live acquisition or a second ingestion registry.

### Invariants shared by all slices

* PBP: preserve existing long plays, kicking, O-Line and TD classification. Skip deleted
  and two-point plays just as AttachPlays does; preserve REG/POST and exact game matching.
  Add counts only for a reported weekly player; never fabricate a player-week.
* Conversion credit: `third_down_converted` / `fourth_down_converted = 1`;
  rusher with `rush_attempt=1` gets X; receiver with `complete_pass=1` gets X;
  passer with `complete_pass=1` gets Q. TDs/chunks still receive the separate conversion.
  No inference from yards/description; no general first-down count substituted.
* RZ: `yardline_100 <= 20` (inclusive); rush_attempt credits rusher RZA;
  pass_attempt credits passer RZP and named receiver RZT, **even on an incompletion**.
  Missing receiver does not create a target. Sack behavior follows pass_attempt exactly.
* New PBP columns are an optional **whole capability** for old held bodies. No new columns
  means no edge keys; partial columns or malformed nonempty fields fail before mutation.
  NA yardline means no RZ location. Valid complete game capability initializes recorded zeros.
* Sleeper rows are sparse: a present, validated numeric player row with an omitted supported
  key records 0; an absent player row/file/identity mapping remains absent evidence. Document
  this upstream convention. Unsupported keys ignored, never parse through `any`/interface{}.
  Allow signed completion air yards/YAC where supplied; reject negative counts/snaps/TFL,
  NaN/Inf, >4-place quantities, duplicate player/JSON keys, conflicting bridge entries and
  cross-season/week attachment. A zero-valued row is different from no row.
* Attaching twice is idempotent (replace owned quantities); neither adapter overwrites the
  other's stats. Validate every row and coverage before touching caller maps.
* Legacy, existing seeded defaults and old raw evidence keep old values and old encodings.
  Optional import fields use omitempty; preserve the exact old revision SHA formula when
  absent. No existing seed-upgrade step is edited. No automatic new-scoring seed upgrade.
* All active new rules require their applicable inputs: missing edge data/rate is incomplete,
  never a silent 0 or a complete score. Inapplicable-position data needs no missing warning.
  Missing data must not stop unrelated Legacy scoring. Never treat an absent rate as 0.
* No new dependencies, generated binding edits, live downloads, DB schema or size-cap change.
  New command directory requires its SYSTEM_MAP/placement entries in the same slice.

## Code/rate table (all of Package A)

`O` = QB/RB/WR/TE; `D` = DT/DE/LB/CB/S; `A` = all ten true player
positions incl. PK, not the synthetic OL card. These are p6.py WHO restrictions.
The DSNP override comes from **p6_run.py**, not p6.py's earlier candidate 0.002.
All ordinary edge rows use inclusive −999..9999 (study range), not the old 99-count bands.
Raw rushing TFL is an input, not an additional −0.15 event penalty.

| Code | New StatID | Source field / definition | Points | Positions |
|---|---|---|---:|---|
| X3 | third_down_conversion | PBP conversion credit above | .25 | O |
| X4 | fourth_down_conversion | PBP conversion credit above | .40 | O |
| Q3 | third_down_passing_conversion | PBP passer credit | .10 | QB |
| Q4 | fourth_down_passing_conversion | PBP passer credit | .15 | QB |
| RZA | red_zone_carry | PBP RZ | .08 | O |
| RZT | red_zone_target | PBP RZ | .08 | O |
| RZP | red_zone_pass_attempt | PBP RZ | .04 | QB |
| RYAC | rushing_yards_after_contact | Sleeper rush_yac | .003 | O |
| CYAC | receiving_yards_after_catch | Sleeper rec_yar | .003 | O |
| RAY | receiving_completion_air_yards | Sleeper rec_air_yd (completions only) | .002 | O |
| PAY | passing_completion_air_yards | Sleeper pass_air_yd (completions only) | .001 | QB |
| BTK | rushing_broken_tackle | Sleeper rush_btkl | .08 | O |
| OSNP | offensive_snap | Sleeper off_snp | .002 | O |
| DROP | receiving_drop | Sleeper rec_drop | −.30 | O |
| RTL | rush_loss_adjustment | Sleeper rush_tkl_loss → raw rusher_tackled_for_loss + weekly carries + frozen rate | −.3124 × (TFL − n/d × carries) | O |
| PEN | player_penalty | Sleeper penalty | −.25 | A |
| PENY | player_penalty_yards | Sleeper penalty_yd | −.01 | A |
| SKY | sack_yards | Sleeper idp_sack_yd | .05 | D |
| STK | special_teams_solo_tackle_topup | Sleeper st_tkl_solo | .25 | D + PK |
| DSNP | defensive_snap | Sleeper def_snp | .008 | D |
| SSNP | special_teams_snap | Sleeper st_snp | .003 | D + PK |

Neither Sleeper down_3_conv/down_4_conv nor def_st_tkl_solo may enter a player's line.
Ignore its RZA/RZT/RZP twins; PBP owns these three. STK adds only .25; the existing
TK base and position extras already pay the tackle. Do not re-add a tackle for STK.
None of these StatIDs exists on this branch; existing RA/TK/TD/chunk rows stay unchanged.

### Rate representation and time contract

Add `ScoringRules.RushLoss Value[RushLossRule]`, field `scoring.rushLoss`.
`RushLossRule` holds Used, coefficient Decimal, and a season→position→rate map.
Each rate holds integer numerator TFL, positive integer denominator carries, prior season,
and canonical evidence digest. Carry the scoring season in `StatLine.Season` (0 means unknown).
`Default32Scoring` sets Used=true, **Coefficient=−3124** (−0.3124), but no guessed rates;
Legacy/old seeds get the known disabled kind. Existing rule field encode/decode/clone/catalogue
machinery is generic, but its validators, descriptions, menus and roundtrip tests need tracing.
Preserve Legacy's JSON golden using the existing optional-value pattern if a plain new field
would serialize; never regenerate that golden to hide a default-only addition.

Use integer/rational arithmetic (`math/big`, already used by the scorer), not a 4-decimal
rounded `r_pos` or rounded residual. Compute the entire RTL award, then round **once** to cents,
half away from zero, with RoundStatPoints=true. With rounding off, return exact Decimal only
if representable; otherwise the same explicit representability error as existing arithmetic.
Example n/d=1/10, carries=10, TFL=0 → +.3124 → +.31; TFL=2 → −.31.
At carries=1,TFL=0 → +.03; no 40-carry threshold. n/d=1/3, carries=1,TFL=0
→ +.104133… → +.10, **not** an inexact multiply error when rounding is on.
Raw TFL stays an unused input; one special `rush_loss_adjustment` LineItem contains the award.
Reject a caller-recorded adjustment (same principle as derived totals). Never independently
round/prices the positive-carry baseline and negative-TFL pieces.

Before first NFL REG kickoff of target season Y, allow installing/correcting Y's rate from
Y−1's pinned artifact. After kickoff, **no insertion, overwrite, deletion, coefficient/Used
change or revert that changes Y's applicable RTL rule**. Adding future season Z preserves
all locked season entries byte-for-byte. No generic G3 exception bypass. Retrying an already
committed request is still allowed and returns the original receipt. Old leagues with RTL
disabled stay disabled; enabling new scoring midseason needs a separate owner decision, not
an upgrade. Unknown kickoff/facts refuses activation or change, never guesses an instant.
A rate/tool rerun or newer imported evidence never selects itself into scoring.
Normal explicit stat/score corrections remain possible and use the *same* frozen rate.

The artifact records source hashes, REG coverage/game manifest, prior/target seasons,
position-map digest, cohort policy and excluded/unjoined records. Require full prior REG
coverage (2024: weeks 1–18, 272 games), not merely max(week)=18 or one play per game.
No 40-carry cohort exclusion. A position with no carries has **no** rate, not 0/0 or an
inferred other-position rate; missing needed position rates block completion.
Only players with a known league position join a positional cohort; report every exclusion.
One unjoined carry exists in this run's 2024 denominator and is explicitly excluded.

## Slices

Commands below are **builder acceptance commands, not commands claimed run here**.
Prefix every command with `GOTOOLCHAIN=go1.26.10 GOMAXPROCS=4`; Go tests use -race.
Do not run a full suite manually. If a slice overruns, stop with a compiling commit and
split its remaining work; do not substitute a mock for the boundary test.
New paths below are proposed, not assumed to exist. Existing named source files were traced.

### E01 — PBP edge input capability — TERRA, 25 min
* Goal: validate new PBP fields without changing scores.
* Files: `internal/fantasyscore/source/plays.go`; new `plays_edge_test.go` in that package.
  Add flags third/fourth_down_converted, rush/pass_attempt, complete_pass, yardline_100;
  retain capability distinction for absent old headers. Reuse validated decimal/flag idioms.
* Tests: `TestReadPlaysEdgeCapability` (complete/absent accepted, partial rejected),
  `TestReadPlaysRejectsMalformedEdgeFields` (bad flags/yardline, NA accepted).
* Command: `go test -race ./internal/fantasyscore/source/ -run 'TestReadPlays|TestAttach'`.
* done = old CSVs still parse; invalid new capability never reaches business counting.

### E02 — Third/fourth-down player credits — TERRA, 30 min
* Goal: attach X3/X4/Q3/Q4 counts only.
* Files: `source/plays.go`; new `source/plays_edges.go`, `source/plays_edges_test.go`
  (all under `internal/fantasyscore/`). Reset only these four owned quantities.
* Tests: `TestAttachConversionCredits` (one third rush + one fourth completion:
  rusher X3=1, receiver X4=1, passer Q4=1; passer X4=0),
  `TestConversionTDChunkStacks` (TD/chunk and conversion retained),
  `TestConversionAttachmentIsIdempotent` (two calls still 1), deleted/2pt/other game excluded.
* Command: `go test -race ./internal/fantasyscore/source/ -run 'Test.*Conversion|TestAttach'`.
* done = all four credits match the stated flags with no season/week/game leakage.

### E03 — PBP red-zone counts — TERRA, 25 min
* Goal: attach RZA/RZT/RZP, not Sleeper twins.
* Files: `internal/fantasyscore/source/plays_edges.go`, `plays_edges_test.go`.
* Tests: `TestRedZoneBoundaryAndCredits` (yardline20 carry=1; yardline21=0;
  incomplete pass at20: RZT=1/RZP=1; unnamed receiver: RZT absent credit),
  `TestRedZoneExclusions` (NA/deleted/2pt excluded; idempotent; old capability absent).
* Command: `go test -race ./internal/fantasyscore/source/ -run 'Test.*RedZone|TestAttach'`.
* done = inclusive 20-yard counts come only from validated PBP player credits.

### E04 — PBP fingerprint regression — SOL, 30 min
* Goal: prove saved-body→revision→player-line identity changes with derived counts.
* Files: `internal/store/history/football_stats_test.go`, `football_week_test.go`;
  `internal/fantasyscore/source/revisions_test.go`; new fixtures under their testdata dirs.
  Change production `football_stats.go` only if a test exposes a real gap.
* Tests: `TestPBPConversionCorrectionChangesPinnedRevision` (one X3→X4 changes
  revision SHA and LineDigest; counted old version retained),
  `TestPBPRedZoneCorrectionChangesPinnedRevision` (21→20),
  `TestOldPBPBodyKeepsFingerprint` (no new keys for absent capability).
* Command: `go test -race ./internal/store/history/ ./internal/fantasyscore/source/ -run 'Test.*PBP|Test.*Revision|Test.*Digest'`.
* done = derived stat changes cannot bypass the existing pinned fingerprint/version chain.
* Why SOL: archive identity and correction baselines, not merely parser counts.

### E05 — Sleeper typed decoder — TERRA, 35 min
* Goal: decode only the supported 14 Sleeper edge fields plus explicit identity bridge.
* Files: new `internal/fantasyscore/source/sleeper.go`, `sleeper_test.go`,
  `source/testdata/sleeper_edges.json`. Raw numeric JSON uses json.Number, not float64/any;
  typed envelope supplies season/week because upstream weekly response does not.
* Tests: `TestReadSleeperEdgeFields` (all mapping-table fields incl RTL, team rows
  never player credits), `TestSleeperSparseZeroVersusMissingRow`,
  `TestSleeperEdgesRejectBadNumbersAndDuplicates`, `TestSleeperBridgeIsUnambiguous`.
* Command: `go test -race ./internal/fantasyscore/source/ -run 'Test.*Sleeper'`.
* done = validated player edges exist independently of scoring and no team-only stat leaks.

### E06 — Attach Sleeper edge facts — TERRA, 25 min
* Goal: merge edges into reported PlayerWeeks with deterministic absence semantics.
* Files: `internal/fantasyscore/source/sleeper.go`, `sleeper_test.go`.
* Tests: `TestAttachSleeperEdges` (32 off_snp→320000; 15 rush_yac→150000;
  2 rush_tkl_loss→20000), `TestSleeperDoesNotReplacePBPRedZone`,
  `TestAttachSleeperRejectsWrongScopeWithoutMutation`, `TestSleeperAttachTwice`.
* Command: `go test -race ./internal/fantasyscore/source/ -run 'Test.*Sleeper'`.
* done = only owned quantities merge; no row or bridge means no fabricated zero player-week.

### E07 — Pin inline Sleeper evidence — SOL, 35 min
* Goal: hold exact JSON/bridge and attach it on inline revision loading.
* Files: `internal/store/history/football_stats.go`, `football_import.go` (only if shared
  bridge type belongs there); new `football_edges.go`, `football_edges_test.go`.
  Optional scoped body records contain season/week, raw JSON, bridge; validate unique scopes.
  Extend inline revision digest with unambiguous length-delimited edge content only when present.
* Tests: `TestInlineSleeperEvidenceIsPinned` (body or bridge change→new SHA/LineDigest),
  `TestInlineSleeperSurvivesArchivePruning`, `TestInlineWithoutEdgesKeepsOldDigest`,
  `TestInvalidInlineEdgesDoNotImport` (archive state unchanged).
* Command: `go test -race ./internal/store/history/ -run 'Test.*Sleeper|Test.*Edges|Test.*Football'`.
* done = the selected evidence, not a live Sleeper response, owns inline edge facts.
* Why SOL: revision identity, immutable import and backwards-compatible byte contract.

### E08 — Pin split-week Sleeper evidence — SOL, 30 min
* Goal: same edge path for per-week parts, preserving the 8 MiB cap.
* Files: `internal/store/history/football_week.go`, `football_stats.go`,
  `football_edges.go`, `football_edges_test.go`, `football_week_test.go`.
  `RawFootballWeek` gains optional scoped edge content; parts and inline bodies cannot mix.
* Tests: `TestSplitWeekSleeperMatchesInline`, `TestSplitSleeperPartDigestMismatch`,
  `TestSplitSleeperRejectsOtherWeek`, `TestOldWeekPartBytesUnchanged`.
* Command: `go test -race ./internal/store/history/ -run 'Test.*Split|Test.*WeekPart|Test.*Sleeper'`.
* done = inline and split facts agree, tampered/mis-scoped parts fail before persistence.
* Why SOL: two existing evidence representations must preserve the same SHA chain.

### E09 — Baseline artifact contract — TERRA, 30 min
* Goal: validated deterministic rate artifact, without rule activation.
* Files: new `internal/fantasyscore/source/rush_loss_rates.go`, `rush_loss_rates_test.go`.
  Raw artifact type/Validate and canonical digest (stdlib only); fields as rate contract above.
* Tests: `TestRushLossArtifactRoundtrip`, `TestRushLossArtifactRejectsInvalidRates`
  (wrong prior year, negative n, d=0, n>d, malformed digest, duplicate positions),
  `TestRushLossArtifactDigest` (position map/coverage/source change changes digest).
* Command: `go test -race ./internal/fantasyscore/source/ -run 'TestRushLossArtifact'`.
* done = rate provenance has a small typed, hashable wire contract.

### E10 — Offline r_pos production tool — SOL, 40 min
* Goal: produce the artifact from held full prior-season files; no hand-entered rate.
* Files: new `cmd/rtl-rates/main.go`, `main_test.go`; extend
  `internal/fantasyscore/source/rush_loss_rates.go`/test for aggregation;
  `SYSTEM_MAP.md`, `docs/architecture/placement.tsv` for the new command row.
  CLI flags explicitly identify prior season, target season, weekly/PBP/league-position and
  complete schedule/game-manifest files. Output artifact on stdout, exclusions on stderr.
* Tests: `TestBuildPriorSeasonPositionRates` (RB TFL2/carries20=1/10; one-carry
  player included; league position overrides source position),
  `TestRatesRequireCompleteRegularSeason` (missing week/game/incomplete manifest rejected),
  `TestRatesDeterministicAndReadOnly` (input order independent; zero filesystem writes).
  Golden-source acceptance is blocked on the numerator-source question above.
* Command: `go test -race ./cmd/rtl-rates/ ./internal/fantasyscore/source/ -run 'Test.*Rates|TestRushLossArtifact'`.
* done = repeated held-input runs emit byte-identical prior-season artifacts or explicit refusal.
* Why SOL: cohort/completeness and source choice determine every future player's RTL award.

### E11 — Commissioner-owned season rate kind — SOL, 40 min
* Goal: register/validate/serialize `scoring.rushLoss`, disabled everywhere initially.
* Files: `internal/leaguerules/scoring.go`, `scoring_kinds.go`, `seed_upgrade.go`,
  `catalogue.go`, `menus.go` only where their generic paths need extension;
  new `scoring_rush_loss.go`, `scoring_rush_loss_test.go`, existing `preset_test.go`.
  Add an append-only seed step defaulting absent kind to disabled, not new scoring.
* Tests: `TestRushLossRuleStoredRoundtrip`, `TestRushLossRuleValidation`,
  `TestPreEdgeSeedKeepsRushLossDisabled`, `TestLegacyEncodingUnchanged`;
  cover deep-clone isolation, recognized menu/catalogue field and multi-season entries.
* Command: `go test -race ./internal/leaguerules/`.
* done = the parameter is a real editable/versioned rule and old seeds/Legacy stay identical.
* Why SOL: seed upgrades and stored JSON replay are compatibility boundaries.

### E12 — Exact weekly RTL pricing — SOL, 40 min
* Goal: implement the pure single RTL item, with no default activation yet.
* Files: `internal/fantasyscore/score.go`, `kinds.go`; new `rush_loss.go`,
  `rush_loss_test.go`; `StatLine.Season` added here. No I/O or source imports.
* Tests: `TestRushLossWeekly` (1/10, 10 carries,TFL0→+.31; TFL2→−.31;
  one carry→+.03; zero/zero→0), `TestRushLossRationalRounding` (1/3→+.10;
  signed half-cent cases away from zero; no premature residual rounding),
  `TestRushLossMissingRateOrInputs` (NeedsRuling, not complete 0),
  `TestRushLossRejectsRecordedAdjustment`, `TestRushLossUnroundedRepresentability`,
  `TestRushLossLeavesInputsAndLegacyAlone` (overflow/error has no mutation).
* Command: `go test -race ./internal/fantasyscore/ -run 'TestRushLoss|TestScore'`.
* done = exact rational weekly RTL prices correctly at any carry volume with one rounding.
* Why SOL: existing multiply cannot represent arbitrary residuals; rational rounding is material.

### E13 — Pin baseline artifact with football evidence — SOL, 30 min
* Goal: imported artifact is immutable evidence; later imports do not activate it.
* Files: `internal/store/history/football_import.go`, `football_evidence.go`;
  new `football_rates.go`, `football_rates_test.go`.
  Add optional embedded artifact(s), validate digest/season, expose defensive-copy read for
  rule admission using existing pinned evidence selection. Do not alter existing required APIs.
* Tests: `TestPinnedRushLossBaseline`, `TestPinnedRatesRejectDigestOrSeasonMismatch`,
  `TestRateArtifactDoesNotActivateItself`, `TestFootballWithoutRatesKeepsBytes`.
* Command: `go test -race ./internal/store/history/ -run 'Test.*RushLoss|Test.*Rates|TestFootballWithout'`.
* done = baseline bytes/provenance survive archive pruning and are not auto-selected rules.
* Why SOL: joins a derived tool artifact to the selected evidence and immutable rule lifecycle.

### E14 — Pure rate-freeze admission policy — SOL, 40 min
* Goal: judge proposed vs existing rate rules with explicit held kickoff/artifact facts.
* Files: new `internal/commissioner/rush_loss_rules.go`, `rush_loss_rules_test.go`;
  `internal/commissioner/rules.go` only for calling the policy when explicit facts supplied.
  Keep old CheckRulesAppend API valid; add narrow facts-based check, not another resolver.
* Tests: `TestRateChangeBeforeKickoff` (matches pinned n/d/digest),
  `TestRateFreezeAtAndAfterKickoff` (insert/edit/delete/Used/coefficient/revert refused),
  `TestFutureRateLeavesPastScoresAlone` (Y unchanged, add Y+1 accepted),
  `TestUnknownRateFactsRefuse`, `TestRateRuleRejectsDigestOnlyForgery`.
* Command: `go test -race ./internal/commissioner/ -run 'TestRate|TestFutureRate|TestUnknownRate'`.
* done = pure admission refuses every way of changing an already-started season's RTL rule.
* Why SOL: current resolver reads season rules at the reading instant; without admission freezing,
  setting a rate later would retroactively change historical score reads.

### E15a — Gather and enforce rate facts at the writer — SOL, 35 min
* Goal: no writer bypasses E14, and no partial persistence on rejection.
* Files: `internal/store/history/commissioner.go`, `native_facts.go`, `native_lineup.go`;
  new `native_rates.go`, `native_rates_test.go`. Gather target-season first REG kickoff
  and selected pinned artifact; recheck against records under the existing append lock.
  Reuse pinned schedule and environment; missing schedule refuses, no guessed date.
  Trace seed, rules.set, change reversal and rollover paths; do not change general APIs blindly.
* Tests: `TestRateFactsUseSelectedEvidence` (not newest unrelated import),
  `TestRateFactsRequireFirstRegularKickoff` (absent schedule refuses),
  `TestRateFreezeThroughAppendAndRestart` (before accepted, at kickoff refused,
  ledger hash/count unchanged; preserved after reload).
* Command: `go test -race ./internal/store/history/ -run 'TestRateFacts|TestRateFreezeThrough'`.
* done = the actual locked append path checks E14 using selected pinned facts.
* Why SOL: joins pure policy to held evidence and the authoritative writer.

### E15b — Close rate revert/retry/concurrency paths — SOL, 35 min
* Goal: prove non-obvious writer paths cannot undo a started season's frozen rate.
* Files: `internal/store/history/native_rates.go`, `native_rates_test.go`,
  `commissioner.go`; `internal/commissioner/rush_loss_rules.go` if reversal needs
  explicit proposed-rule comparison. Do not reimplement generic retry identity.
* Tests: `TestRateFreezeRejectsRevert`, `TestCommittedRateRetryAfterKickoff`,
  `TestConcurrentRateChangesRecheckStandingRule`,
  `TestFutureRateInstallKeepsHistoricalScore` (same historical total before/after/restart).
* Command: `go test -race ./internal/store/history/ -run 'Test.*Rate'`.
* done = all write/reversal/retry routes enforce the freeze without rejected-write persistence.
* Why SOL: reverse/retry semantics and under-lock revalidation are state-integrity guarantees.

### E16 — Season context and missing-edge completeness — SOL, 35 min
* Goal: connect known season and required applicable inputs to real score readings.
* Files: `internal/fantasyscore/source/weekly.go`, `revisions.go` (only if digest input changes);
  `internal/commissioner/player_score.go`, `stat_ledger.go`, `stat_check.go`, `stat_queue.go` where traced
  copying/correcting StatLine drops Season; `internal/fantasyscore/kinds.go`;
  new `internal/commissioner/edge_score_test.go`, extend `score_test.go`.
  Preserve old LineDigest for old evidence (Season already scoped by PlayerWeek);
  if using an optional digest field, omit its zero value. Audit every StatLine copy.
* Tests: `TestNativeEdgeScoreUsesWeekSeason`, `TestCorrectedCarryRepricesFrozenRTL`
  (n/d1/10, TFL0, 10→20 carries changes RTL .31→.62),
  `TestMissingSleeperOrPBPMakesDefaultIncomplete`, `TestLegacyNeedsNoEdgeEvidence`,
  `TestInapplicableEdgeDoesNotBlockPK`, `TestMissingRateNotReportedAsCompleteZero`.
* Command: `go test -race ./internal/fantasyscore/ ./internal/fantasyscore/source/ ./internal/commissioner/ -run 'Test.*Edge|Test.*RushLoss|Test.*CorrectedCarry|Test.*MissingRate|TestLegacyNeeds|TestInapplicable'`.
* done = real reading/correction paths cannot silently omit applicable new points or season context.
* Why SOL: source → pure scorer → corrections/readings spans three distinct contracts.

### E17 — Activate every ruled Default32 value — TERRA, 40 min
* Goal: turn on the exact 21-code table, ST top-up and rational RTL kind for new defaults.
* Files: `internal/leaguerules/scoring_default32.go`, `default32_test.go`;
  `internal/fantasyscore/default32_test.go`, `default32_golden_test.go`,
  `internal/fantasyscore/source/source_test.go` where old-default completeness assertions live;
  new `internal/fantasyscore/testdata/default32_pre_edges_rules.json` (captured **before** edit).
* Tests: `TestDefault32PackageARows` (every rate, range, position; coefficient−3124),
  `TestDefault32PackageAUnitAwards` (X3 2→.50, X4 2→.80, Q3 2→.20, Q4 2→.30;
  RZA/RZT 2→.16, RZP2→.08; RYAC/CYAC10→.03, RAY10→.02,PAY10→.01;
  BTK2→.16, OSNP50→.10, DROP2→−.60,PEN2→−.50,PENY5→−.05;
  SKY5→.25,STK2→.50,DSNP50→.40,SSNP50→.15),
  `TestSTTopupDoesNotDoubleTackle` (LB tackle1+STK1→2.25, PK likewise),
  `TestPackageAPositionRestrictions` (WR DSNP/STK unused; PK SSNP50→.15).
* Unit-price tests use complete recorded-zero edge lines plus a named valid season/rate;
  a single raw quantity alone is deliberately incomplete after E16.
* Preserve old 21-row 2024 golden values as **frozen pre-edge scoring** using the captured
  actual rule snapshot; rename test accordingly, never inject fake known-zero edges into real
  old evidence and call it the new default. Old held-default tests without new fields must now
  test Incomplete under the new default and retain their old-rule complete-score assertion.
  This is an explicitly changed contract, not a dropped/weakened assertion. The new actual
  Default32 golden is E19. Run existing tests in the three packages to find all literal counts.
* Command: `go test -race ./internal/leaguerules/ ./internal/fantasyscore/ ./internal/fantasyscore/source/`.
* done = new default owns all ruled values; old evidence stays honestly incomplete under it.

### E18a — Extract small auditable golden evidence — TERRA, 35 min
* Goal: commit raw five-week evidence and independent expected values, not only study pickles.
* Files: new `internal/fantasyscore/testdata/tools/edge_golden.py`,
  `testdata/tools/README.md`; new source fixtures `internal/fantasyscore/source/testdata/edge_2025_{weekly,plays}.csv`,
  `edge_2025_sleeper.json`, `edge_2025_bridge.json`, `edge_2024_rates.json`.
  Tool reads held originals only and writes only these explicitly selected fixture paths.
* Checks: tool verifies source hashes below, extracts original rows/header columns and exact
  Sleeper player rows, game-scope PBP (for coverage), bridge, prior artifact and metadata.
  Compare per-code raw extraction to this run's numbers; fail, do not tune on discrepancy.
  Record exclusions/source metadata; no runtime access to study directories.
* Tests: `TestEdgeFixtureHasFiveSourceIdentities` (listed five, distinct scopes, digests),
  `TestEdgeFixtureExtractionDeterministic` (repeat extraction byte-identical).
  Put these in new `internal/fantasyscore/source/edge_fixture_test.go`.
* Command: `python3 -I internal/fantasyscore/testdata/tools/edge_golden.py` then
  `go test -race ./internal/fantasyscore/source/ -run 'TestEdgeFixture'`.
* done = a clean checkout holds every small byte input needed for the five-player comparison.

### E18b — Freeze independent expected awards — TERRA, 25 min
* Goal: commit the arithmetic oracle separately from extracting inputs.
* Files: `internal/fantasyscore/testdata/tools/edge_golden.py`, new
  `internal/fantasyscore/testdata/edge_2025.json`, `testdata/tools/README.md`;
  `internal/fantasyscore/source/edge_fixture_test.go`.
* Tests: `TestEdgeExpectedAwardsHaveIndependentProvenance` (five expected totals,
  source/rate digests and per-code quantities/awards, no engine-produced expectation).
  Extractor must reproduce the approved §Golden arithmetic, or stop with the discrepancy.
* Command: `python3 -I internal/fantasyscore/testdata/tools/edge_golden.py` then
  `go test -race ./internal/fantasyscore/source/ -run 'TestEdgeExpected|TestEdgeFixture'`.
* done = the reviewed, independently generated oracle is frozen with its exact inputs.

### E19a — Real adapter→engine golden — SOL, 30 min
* Goal: score the five players through both actual adapters and actual Default32Scoring.
* Files: new `internal/fantasyscore/source/edge_golden_test.go`.
  Consume E18a/E18b fixtures; configure only the explicitly approved pinned 2024 baseline parameter.
* Tests: `TestDefault32EdgeGolden2025`: Hunt2025w4=9.96, CMCw7=48.27,
  Hurtsw1=31.05, Keenanw1=20.87, Dartw6=31.87 **if PBP baseline approved**.
  Assert each edge/RTL subtotal as well as total, tolerance ≤.01 (prefer exact cents).
  `TestP6ConversionLeaderReproduction` checks X-only totals in §Golden separately.
  Mutations worth trying: drop X4, use Sleeper RZ, coefficient−.15, current-year rate,
  restore 40-carry floor, double-pay STK; each must fail a relevant golden/unit test.
* Command: `go test -race ./internal/fantasyscore/source/ -run 'TestDefault32EdgeGolden|TestP6Conversion'`.
* done = real held-input adapter totals match the independent approved arithmetic, not hand-built lines.
* Why SOL: integration oracle must distinguish intentionally changed p6 sources from actual defects.

### E19b — Pinned reading golden — SOL, 35 min
* Goal: prove one real reference week survives the complete pinned store/reading path.
* Files: new `internal/store/history/football_edge_golden_test.go`; reuse E18 fixtures.
* Tests: `TestPinnedDefault32EdgeGolden` (Hunt9.96 via split import and NativePlayerScore,
  inline parity, old counted revision stable after a later edge correction/restart),
  `TestPinnedGoldenMissingEdgesIsIncomplete` (never the same 9.96 with fabricated zeros).
* Command: `go test -race ./internal/store/history/ -run 'TestPinnedDefault32EdgeGolden|TestPinnedGoldenMissing'`.
* done = held SHA/part/revision/rules/reading integration reproduces the approved golden.
* Why SOL: actual pinned-evidence selection and historical baselines can differ from direct scoring.

### E20 — Ship amended Default32 scoring contract — TERRA, 30 min
* Goal: documentation matches what passed, with no live-feed claims.
* Files: `docs/architecture/native-scoring.md`, new
  `docs/scoring-engine/DEFAULT-32-SPEC.md`, `docs/sources/Approved_Sources.md` (held Sleeper
  scoring scope, not new network authorization), `docs/architecture/system-narrative.md`;
  update `docs/build-handoffs/revamp-E-slices.md` with build/evidence status, `HANDOFF.md`.
  The original DEFAULT-32-SPEC is outside this worktree, so do **not** edit it there.
  Put the maintained in-repo copy at the path above, preserving existing roster/OL rulings
  and marking §4's edge amendments/date, superseding old per-event RTL and rounding wording.
* Checks: every table code maps once, all 21 values/position restrictions match E17;
  cite tests actually run and exact final output; document prior numerator decision, complete
  cohort/coverage, coefficient location, freeze/no retroactivity, missing evidence and live gaps.
* Command: `git diff --check` and a targeted `rg -n 'RTL|STK|DSNP|rushLoss|0.3124' docs/architecture/native-scoring.md docs/scoring-engine/DEFAULT-32-SPEC.md` against the table.
* done = Default32's spec, actual settings and evidence agree, and remaining debt is explicit.

## Golden arithmetic computed in this run

Command actually executed (final run adds raw-Sleeper equality assertions):

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -I .scratch/revampE/p2-calculate.py \
  | tee .scratch/revampE/p2-calculation.log
```

The complete research-only script is embedded below so the ignored scratch is not required
for reproduction. It reads the held p6 cache/pickles and read-only league-history SQLite;
it does not invoke cache-writing study loaders or fetch anything. Base = running the frozen
p4c reference rule list on its held event table **in this run**. New PBP counts are independently
recounted from raw CSV here (deleted/2pt excluded). New non-PBP facts use the held p6 merged
table; the final run independently compared **all 14 fields for all five chosen rows** with
raw held Sleeper JSON and printed RAW SLEEPER MATCH five times. E18 retains this assertion
and commits the extracted raw bytes. Base event-table→raw-nflverse equivalence is still
for E18/E19 acceptance, not established merely by running the old reference scorer.

Prior 2024 **full REG weeks 1–18**, league-position joined:

| Position | PBP rushing TFL | weekly carries | n/d |
|---|---:|---:|---|
| RB | 1106 | 11832 | 1106/11832 = .09347532116294794… |
| QB | 79 | 2390 | 79/2390 = .03305439330543933… |
| WR | 62 | 430 | 62/430 = .1441860465116279… |
| TE | 6 | 23 | 6/23 = .2608695652173913… |

The input has all REG weeks, but this planning computation did **not** independently certify
all game-final completeness; E10 requires the manifest. Unjoined carries=1, unjoined TFL=0.
These are not the TFL study's 2024 rates (.0935 RB, **.0703 QB**, .1475 WR, .25 TE): that
study used Sleeper, joined players and weeks1–13. Do not claim those as this full-season cohort.

Each new ordinary stat is rounded individually to cents; RTL is one rational award rounded
once. Base arithmetic and edge arithmetic below are already-rounded line amounts.

| Player / IDs / week | Base arithmetic | Ordinary edges arithmetic | RTL arithmetic | Total |
|---|---|---|---|---:|
| Hunt / 13138 / 00-0033923 / 2025w4 | RY3.40+RA1.95+CY1.30+CC1=7.65 | X3 .75 + X4 .80 + RZA .24 + RYAC .05 + CYAC .02 + RAY .01 + OSNP .06 =1.93 | round[−.3124×(0−1106/11832×13)] =+.38 | **9.96** |
| McCaffrey / 13130 / 00-0033280 / w7 | rushingTD12+RY12.90+RA3.60+CY7.20+CC7+touches3=45.70 | X3 1.25+RZA .40+RZT .08+RYAC .14+CYAC .17+RAY .03+BTK .32+OSNP .10=2.49 | round[−.3124×(2−1106/11832×24)]=+.08 | **48.27** |
| Hurts / 14783 / 00-0036389 / w1 | PY7.60+P40 1+rushingTD12+RY6.20+RA2.10=28.90 | X3 1.25+Q3 .30+RZA .24+RZP .04+RYAC .10+PAY .09+BTK .16+OSNP .13−PEN .25−PENY .05=2.01 | round[−.3124×(0−79/2390×14)]=+.14 | **31.05** |
| Keenan Allen / 11222 / 00-0030279 / w1 | receivingTD6+CY6.80+CC7=19.80 | X3 .75+RZT .08+CYAC .06+RAY .10+OSNP .08=1.07 | zero carries/TFL=0 | **20.87** |
| Dart / 17032 / 00-0040691 / w6 | passingTD5+PY9.75+rushingTD6+RY5.80+RA1.95+chunk1=29.50 | X3 1+Q3 .60+RZA .24+RZP .12+RYAC .03+PAY .12+OSNP .13=2.24 | round[−.3124×(0−79/2390×13)]=+.13 | **31.87** |

Raw new quantities, omitted codes=recorded zero for these particular held player rows:

| Player | Nonzero quantities |
|---|---|
| Hunt | X3=3,X4=2,RZA=3,RYAC=15,CYAC=8,RAY=5,OSNP=32; carries13,TFL0 |
| McCaffrey | X3=5,RZA=5,RZT=1,RYAC=48,CYAC=56,RAY=16,BTK=4,OSNP=50; carries24,TFL2 |
| Hurts | X3=5,Q3=3,RZA=3,RZP=1,RYAC=34,PAY=91,BTK=2,OSNP=63,PEN=1,PENY=5; carries14,TFL0 |
| Keenan | X3=3,RZT=1,CYAC=20,RAY=48,OSNP=40; carries0,TFL0 |
| Dart | X3=4,Q3=6,RZA=3,RZP=3,RYAC=11,PAY=119,OSNP=67; carries13,TFL0 |

Separate p6 §5 reproduction over REG weeks1–13, **conversions only**, not whole scores:
Hunt 17×.25+11×.40=8.65 /12=.720833…→.72;
CMC 29×.25+1×.40=7.65 /13=.588461…→.59;
Hurts 12×.25+8×.40=6.20 /12=.516666…→.52;
Keenan 23×.25+1×.40=6.15 /12=.5125→.51;
Dart 14×.25+4×.40=5.10 /10=.51.
Those exactly reproduce the printed conversion PPG; p6 §5 does not publish total player-week
scores, so matching unchanged published whole-week totals is not possible or claimed.

### Research command source

```python
# Research only: independent Decimal arithmetic; no application implementation.
import sys, os, csv, gzip, json, pickle, sqlite3, hashlib
from collections import defaultdict
from decimal import Decimal as D, ROUND_HALF_UP
P='/home/chris/fleet/runs/config-study-p6-2026-10-27'
sys.path.insert(0,P)
import study4c, legacy_scorer as L
C=P+'/cache'
con=sqlite3.connect('file:'+L.DB+'?mode=ro',uri=True)
pos={y:dict(con.execute('select pid, position from player_season where year=?',(y,))) for y in (2024,2025)}
names=dict(con.execute('select pid,name from player_season where year=2025'))
bridge={}; m2s={}
with open(C+'/db_playerids.csv') as f:
 for r in csv.DictReader(f):
  if r['gsis_id'] not in ('','NA') and r['mfl_id'] not in ('','NA'): bridge[r['gsis_id']]=r['mfl_id']
  if r['sleeper_id'] not in ('','NA') and r['mfl_id'] not in ('','NA'): m2s[r['mfl_id']]=r['sleeper_id']
car=defaultdict(int); tfl=defaultdict(int); unknown=defaultdict(int)
with gzip.open(C+'/stats_player_week_2024.csv.gz','rt') as f:
 for r in csv.DictReader(f):
  if r['season_type']!='REG': continue
  n=int(D(r['carries'] or '0')); p=pos[2024].get(bridge.get(r['player_id']))
  if p: car[p]+=n
  elif n: unknown['carries']+=n
with gzip.open(C+'/play_by_play_2024.csv.gz','rt') as f:
 for r in csv.DictReader(f):
  if r['season_type']!='REG' or r['play_deleted']=='1' or r['two_point_attempt']=='1': continue
  if r['rush_attempt']=='1' and r['tackled_for_loss']=='1':
   p=pos[2024].get(bridge.get(r['rusher_player_id']))
   if p: tfl[p]+=1
   else: unknown['TFL']+=1
print('PRIOR 2024 FULL REG, PBP numerator/weekly carries denominator, league positions')
for p in ('RB','QB','WR','TE'): print(p,tfl[p],car[p],str(D(tfl[p])/D(car[p])))
print('UNJOINED',dict(unknown))
# Read the frozen tables, never invoke study loaders that can write caches.
ev=pickle.load(open(P+'/events4c_2025.pkl','rb'))
ns=pickle.load(open(P+'/p6cache/newstats_2025.pkl','rb'))[0]
rules=study4c.VARIANTS['C2 continuous + dyn kicker'](study4c.V.BASE)
prices={'X3':'.25','X4':'.40','Q3':'.10','Q4':'.15','RZA':'.08','RZT':'.08','RZP':'.04','RYAC':'.003','CYAC':'.003','RAY':'.002','PAY':'.001','BTK':'.08','OSNP':'.002','DROP':'-.30','PEN':'-.25','PENY':'-.01','SKY':'.05','STK':'.25','DSNP':'.008','SSNP':'.003'}
off={'QB','RB','WR','TE'}; idp={'DT','DE','LB','CB','S'}
who={k:off for k in prices}
for k in ('Q3','Q4','RZP','PAY'): who[k]={'QB'}
for k in ('PEN','PENY'): who[k]=off|idp|{'PK'}
who['SKY']=who['DSNP']=idp
who['STK']=who['SSNP']=idp|{'PK'}
pbp=defaultdict(lambda:defaultdict(int))
with gzip.open(C+'/play_by_play_2025.csv.gz','rt') as f:
 for r in csv.DictReader(f):
  if r['season_type']!='REG' or int(r['week'])>13 or r['play_deleted']=='1' or r['two_point_attempt']=='1': continue
  def credit(g,k):
   m=bridge.get(g)
   if m: pbp[(int(r['week']),m)][k]+=1
  for flag,code,q in [('third_down_converted','X3','Q3'),('fourth_down_converted','X4','Q4')]:
   if r[flag]=='1':
    if r['rush_attempt']=='1': credit(r['rusher_player_id'],code)
    if r['complete_pass']=='1': credit(r['receiver_player_id'],code); credit(r['passer_player_id'],q)
  if r['yardline_100'] not in ('','NA') and D(r['yardline_100'])<=20:
   if r['rush_attempt']=='1': credit(r['rusher_player_id'],'RZA')
   if r['pass_attempt']=='1': credit(r['passer_player_id'],'RZP'); credit(r['receiver_player_id'],'RZT')
quant=lambda x:x.quantize(D('.01'),rounding=ROUND_HALF_UP)
want=['Hunt, Kareem','McCaffrey, Christian','Hurts, Jalen','Allen, Keenan','Dart, Jaxson']
for name in want:
 pid=next(k for k,v in names.items() if v==name); p=pos[2025][pid]
 keys=[k for k in ev if k[1]==pid]
 key=max(keys,key=lambda k:(D('.25')*pbp[k]['X3']+D('.40')*pbp[k]['X4'],-k[0]))
 e=ev[key]; n=ns.get(key,{})
 sleeper_fields=dict(off_snp='OSNP',def_snp='DSNP',st_snp='SSNP',rec_drop='DROP',rush_btkl='BTK',pass_air_yd='PAY',rec_air_yd='RAY',rush_yac='RYAC',rec_yar='CYAC',rush_tkl_loss='RTL',penalty='PEN',penalty_yd='PENY',idp_sack_yd='SKY',st_tkl_solo='STK')
 raw=json.load(open(f'{P}/p6cache/sleeper_2025_{key[0]}.json'))[m2s[pid]]
 for field,code in sleeper_fields.items():
  assert D(str(raw.get(field,0)))==D(str(n.get(code,0))),(name,field)
 print('RAW SLEEPER MATCH',pid,key[0],m2s[pid],len(sleeper_fields),'fields')
 vals={k:D(str(pbp[key].get(k,0) if k in ('X3','X4','Q3','Q4','RZA','RZT','RZP') else n.get(k,0))) for k in prices}
 parts={k:quant(vals[k]*D(v)) for k,v in prices.items() if p in who[k]}
 rtl=quant(-D('.3124')*(D(str(n.get('RTL',0)))-D(tfl[p])*D(str(e.get('RA',0)))/D(car[p])))
 base=D(str(L.score(e,p,rules))); edge=sum(parts.values()); total=base+edge+rtl
 base_parts=[(r['event'],L.score(e,p,[r])) for r in rules if L.score(e,p,[r])]
 print(json.dumps(dict(name=name,mfl=pid,gsis=next(g for g,m in bridge.items() if m==pid),week=key[0],position=p,base=str(base),base_parts=base_parts,edge=str(edge),RTL=str(rtl),expected=str(total),carries=e.get('RA',0),TFL=n.get('RTL',0),counts={k:str(v) for k,v in vals.items() if v},parts={k:str(v) for k,v in parts.items() if v})))
 # All played weeks: retain p6 conversion-only check separately from new whole-week totals.
 x3=sum(pbp[k]['X3'] for k in keys); x4=sum(pbp[k]['X4'] for k in keys)
 print('CONVERSIONS',name,len(keys),x3,x4,str(D('.25')*x3+D('.40')*x4))
for f in ['cache/play_by_play_2024.csv.gz','cache/stats_player_week_2024.csv.gz','cache/play_by_play_2025.csv.gz','cache/db_playerids.csv','events4c_2025.pkl','p6cache/newstats_2025.pkl']:
 print('SHA256',f,hashlib.sha256(open(P+'/'+f,'rb').read()).hexdigest())

```

### Observed input fingerprints

Paths below are relative to the held `config-study-p6-2026-10-27` run directory;
script location and read-only database location are explicit in the command source above.
These hashes were computed in the same run, not copied from the old handoff.

```text
cache/play_by_play_2024.csv.gz 23370d5d10f8104d80d46a1fc5e61f4f6f5a3263fe96fe2dd629913cfcb08c06
cache/stats_player_week_2024.csv.gz 61fc9a44706522218a4448001706e3fec65ebbb97c1d9c5746ceef364989125d
cache/play_by_play_2025.csv.gz 2f135887790a013fd004e609e37096bb4816d5cc80b9f19122e1bad478961978
cache/db_playerids.csv e8b5e30b1b62c4fef25ef84c77dc2994a07334e797e7fc3ab9a14c2db46c19b2
events4c_2025.pkl 3ef92c21066407a58e71b3d1676624c16b32f92b3b68c58eadeeed5be906a827
p6cache/newstats_2025.pkl 13b2e53a6d4b96fed563a91a2fb36606d976226cb8f4f2c16f658a39a6bc2c66
```

## Debt left / exclusions

* This commit is only a plan; all scoring, imports, freeze enforcement, tests and amended
  spec remain to build. No test/build/live acceptance claimed.
* Prior TFL-source decision and independent complete-game manifest still needed before
  accepting production rate artifacts and the conditional golden totals.
* Raw Sleeper values for the five selected rows were verified (14/14 each); committing
  those bytes and verifying frozen base events against raw nflverse is E18/E19 work.
* No live Sleeper/nflverse downloader, feed registry, kickoff schedule acquisition or
  full-prior-Sleeper cache is added here. Pinned imported evidence is the supported path.
* No parity/balance re-study: PBP RZ and prior-rate RTL intentionally differ from p6.
  Do not claim p6's aggregate acceptance results automatically carry over.
* No retrofit of already-started leagues, historical scoring rewrite, missing-rate zero,
  floor, current-season normalization or settlement. Any requested midseason adoption
  requires a separate owner ruling compatible with the no-retroactivity rule.
