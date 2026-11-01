# APP-SCHED-8 — missing weekly rows

## Delivered / limits

Backend only; no scoring math, league-origin branches or counted-revision changes.
Local commits: `62a9d08e`, `0ab75324`, `e1deddf5`, `4f638287`, `7c2f5733`, `d7ce8249`.

- Validated named play credits feed missing player-weeks through the ordinary held revision,
  LineBook, feed selection and StatApprove path. Existing weekly rows are never overwritten.
  Defensive solo/assisted tackles, full/half sacks, QB hits, TFL, PD, interceptions, forced
  fumbles, recoveries, returns and associated ordinary offensive credits are supported.
- An absent weekly row with affirmative positive snaps or exact-week Out evidence, a declared
  END GAME and complete player-event columns can become a recorded empty line. Overtime
  finals do not require an expired clock. Any named player event prevents this zero route.
  The empty line retains recorded zeros so configured zero-stat awards still run normally.
- Fallback revisions carry `nflverse missing row` and deciding evidence into the stat queue,
  even after automatic feed approval. Manual approval/rejection retains its existing authority.
- No ESPN or Sleeper bodies are held in this copied profile. If unsupported secondary bodies
  are held, zero admission refuses rather than ignoring them. General secondary-source fallback
  integration is **not delivered**; it remains debt, not a claimed completed adapter.
- Confirmed game-day inactive and historically verified bye evidence were not available.
  Those unknowns stay unavailable. Identity 0835 was not guessed or set to zero.

## Actual slice / ownership

`source.ReadPlays` validates retained player IDs, distances and clock; `MissingPlayWeeks`
aggregates only absent weeks. Store completion combines archived weekly/PBP evidence and
validated snap/injury proofs, fingerprinting raw bodies and resolved identities. `AddProvenZeros`
requires full event coverage and explicit finality; it rejects conflicting event evidence.
The archive store owns state and revisions; source adapters have no I/O. `LineBook` and direct
held reads retain the same provenance/review metadata. `queueBuilder.candidate` adds the
missing-row review without changing `FeedVersion`, corrections or scoring formulas.
Pinned football gets PBP fallback too, but has no embedded snap/injury proof for this zero route.

Source-traced responsibilities and caller paths were checked locally. Independently exercised:
CSV missing-row fixture → held fallback → ordinary feed approval → 1 solo LB tackle = 1.5
points → open review → commissioner approval clears review. Removing CSV player-credit
retention made that test fail; restoration passes. No independent reviewer or GUI test claimed.

## Copied-profile comparison

SQLite backup API copied the scratch profile into ignored `.tmp/sched8-fallback/profile`.
Ordinary backfill/acquisition/feed/results/standings, same 148-row cohort as part 7.
No remote git contact or production profile writes. One-use harness removed from the test tree;
pre-existing measurement harness and cache left untouched.

| Week | Rows | Exact | Within .01 | Differ | Unavailable |
|---|---:|---:|---:|---:|---:|
| 1 | 15 | 3 | 1 | 2 | 9 |
| 2 | 15 | 4 | 0 | 2 | 9 |
| 3 | 18 | 7 | 0 | 2 | 9 |
| 4 | 22 | 7 | 0 | 3 | 12 |
| 5 | 23 | 10 | 0 | 0 | 13 |
| 6 | 24 | 5 | 0 | 1 | 18 |
| 7 | 31 | 9 | 0 | 3 | 19 |
| **Total** | **148** | **45** | **1** | **13** | **89** |

Part 7: 24 exact / 0 within .01 / 9 differ / 115 unavailable. Now 26 more complete weeks.
Full rows: `APP-SCHED-8-comparison.tsv`; player-week dispositions: `APP-SCHED-8-absent.tsv`.

| Missing player-week disposition | Count | Deciding evidence |
|---|---:|---|
| Scored from PBP stat credits | 0 | No nonblank missing GSIS/week occurs in held PBP player-ID columns |
| Scored from ESPN | 0 | No held box scores; no ESPN adapter built |
| Recorded zero: positive snaps | 141 | Exact player/game snap row + completed PBP + no named event |
| Recorded zero: Out | 53 | Exact GSIS/REG week Out + matching completed game + no named event |
| Still unresolved | 242 | No affirmative exact-week proof |
| Still bye candidates | 36 | Historical affiliation not established |
| Still identity gap | 7 | 0835 has no verified GSIS link |

All 472 formerly absent player-weeks have MFL player points zero. That comparison is **not**
used as proof for zero admission. There are no demonstrated positive-stat gaps requiring an
ESPN scorer in this cohort; **285 player-weeks still have evidence/identity gaps**. ESPN/inactive
and historical-team acquisition could address availability, but has not been implemented here.

### Top five complete differences (app minus MFL)

| Week / team | MFL | App | Delta | Observed cause |
|---|---:|---:|---:|---|
| 1 NY Giants | 208.20 | 179.70 | -28.50 | Native invalid-lineup penalty zeroes Za'Darius Smith (26); Roquan Smith -2.50 residual |
| 7 Cincinnati | 145.60 | 117.50 | -28.10 | Native invalid-lineup penalty zeroes Jameson Williams (28.10) |
| 2 NY Giants | 205.65 | 182.20 | -23.45 | Native invalid-lineup penalty zeroes Cam Ward (23.45) |
| 7 Tennessee | 146.50 | 143.00 | -3.50 | Markquese Bell 7.00 versus MFL 10.50; credit/rule root cause unverified |
| 6 Atlanta | 214.15 | 217.65 | +3.50 | Cooper DeJean +2 and D.J. Reader +1.50 residuals; root cause unverified |

The new large differences were previously unavailable, not newly changed scoring math.
MFL weeklyResults lacks stat-credit breakdowns; residuals are not diagnosed as rule errors.

**Bills:** MFL **6–1 / PF 1912.35**; native **unavailable, 0/7 complete weeks**.
Its displayed 0–0/PF 0 is not an observed W-L/PF. Zane Gonzalez W1–6 and other unresolved
starters still block totals; no unsupported zero was inserted to manufacture parity.

## Verification / execution

- Final source package: **39 PASS / 4 opt-in SKIP**, race; final commissioner feed/queue slice:
  **8 PASS**, race. Includes populated CSV integration, zero-yard return, overtime final,
  schema loss, identity gap, conflicting stat, weekly precedence and zero-stat award.
- Full earlier affected-package race batch: history **256 PASS / 4 SKIP**; commissioner market
  **28 PASS**; commissioner **271 PASS / 2 FAIL**. Not a green full suite.
  `TestSinglePoolStartupDraftIsUnchanged` reports digest `8a6936145d37309e`, reproduced at
  starting commit `c37754d1`. `TestOnlyTheCatalogNamesAPreset` scans pre-existing
  `.tmp/sched4/baseline` and the temporary baseline snapshot; that snapshot's isolated test passes.
  No unrelated tests or gates weakened. Final overtime/source changes also passed targeted
  history race tests; full batch was not repeated on unchanged unrelated behavior.
- All commit hooks passed after correcting genuine lint failures; no bypass.
- Final plain `make build`: **PASS**, 25.02 s; binary not launched.
- Initial full copied backfill/PBP measurement PASS 294.53 s. Intermediate zero attempt was
  interrupted during redundant backfill; resumed ordinary feed/results on the same copy.
  Zero/feed proof PASS 415.50 s, idempotent final-schema proof PASS 147.05 s, final overtime
  proof PASS **154.32 s**. Non-race copied measurements, not race/GUI evidence.
- Complete affected handwritten production footprint: **1744 → 2399 physical lines**, ten files
  against `c37754d1`; tests/docs counted separately. This is added behavior, not a debloat claim.

Latest held raw body digests:

| Body | SHA256 |
|---|---|
| Weekly | `501eb74105287596fdf138f3bdc7d0786af71dccc386f811563359ffff203b97` |
| PBP | `653bf713cb9851a8a9d37787388f27a3a339c0e3429d0e64f611402d4328fe6f` |
| Snaps | `03d803cce40868395fb6d7a3903e5e10f2d17651ca2c3391218dde85d5da148e` |
| Injuries | `16d5a88e806dc6427c49b6f2de442426fc7d1bbaa018e5cb19d3185ce20e4b03` |

## Debt left

285 unavailable player-weeks; secondary ESPN/Sleeper fallback integration; inactive/historical
bye proof; rare lateral/split-return and two-point/kicker fallback completeness; pinned absence
proof embedding; costly whole-season feed/adoption; score residuals and lineup-penalty parity;
76 historical-roster backfill refusals; inherited test failures; independent review/live validation.
A held unsupported secondary source blocks zeros conservatively until its adapter is supplied.
