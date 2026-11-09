# APP-SCHED-9

## Intended slice / findings

1. Missing-row admission currently cannot use complete snap files and dated depth membership.
   Decode/validate full snap rows, require internally reconciled offense/defense/special-team
   totals for both game teams, join dated held depth snapshots by GSIS, then admit only through
   existing final-PBP/event-conflict gates. Missing/partial files and unknown identities stay
   pending. Archive store owns reading/fingerprints; source owns pure coverage checks; existing
   held revision, feed and queue retain provenance. No scoring math changes.
2. Schedule byes need exact-week affiliation. Use held schedule bye declarations and dated depth
   membership; never current player-directory teams. Same recorded-zero/queue path.
3. 0835 has only its MFL external ID. Inspect held crosswalk/source IDs; do not join by name alone.
4. Imported lineup penalty inherits `zero_top_starter` from the base despite explicit partial
   lineups allowed. Preserve the setting-driven scoring path; translate the held setting, not
   league identity. Tests must show explicit permission overrides the base and unsupported values
   do not invent a penalty.

Verification planned: test rejection first, targeted race batches, plain `make build`, copied
SQLite profile's ordinary feed/results/standings, same 148-row cohort. No production writes,
independent reviewer or GUI evidence claimed. Existing untracked harness/cache remain untouched.

Starting ref: 48bacfac. `git pull --ff-only` refused: branch has no upstream.

## Item 3 — identity blocked, evidence-backed no change

Held DynastyProcess crosswalk has **zero rows for MFL 0835/835**. The canonical directory
has only provider=mfl/value=0835 for that player; no GSIS, ESPN, Sleeper or PFR bridge.
The same-name row is **MFL 17372**, GSIS `00-0040044`, ESPN `4685841`, PFR `ChilZi00`,
Sleeper missing. Those three external IDs already belong to the distinct canonical player
for MFL 17372. Historical depth also carries that GSIS/ESPN pair, not MFL 0835.

Exact missing link: authoritative evidence that league-local **0835 aliases 17372** (or one
of its three external IDs), followed by the normal identity merge/link flow. Neither a name
match nor DAL affiliation authorizes merging two canonical players. All seven 0835 weeks
remain pending; no zero or identity mutation made. Existing identity-gap zero-rejection tests
pass in the targeted source batch. Source-traced crosswalk guard rejects reused/misattributed
MFL IDs; no guard was weakened.

## Delivered / exercised

- **1: implemented, cohort partly closed.** 55 additional zero player-weeks carry exactly
  `complete snaps + pbp, no snap`. Both teams must have unique player rows and all three
  units' counts must reconcile to eleven player-snaps per uniquely recoverable snap total.
  Rounded percentages admitting adjacent totals are insufficient. This deliberately leaves
  legitimate short-handed/otherwise unreconciled games pending rather than asserting completeness.
  Dated depth membership must be in the newest snapshot before that game's date, at most seven
  days old. A departed player's older row is not membership. Existing complete-final-PBP and
  no-event gates still apply; present snap rows, missing PFR/GSIS and missing team games refuse.
- **2: implemented, cohort partly closed.** 32 additional zeros carry exactly `team bye per schedule`.
  Positive season-scoped held NFL bye declarations plus dated depth membership are required;
  absence from a partial schedule alone is never evidence. Week dating uses that week's held
  game dates. Any held PBP game for the team, player event or snap-game conflict blocks the bye.
  Four previous bye candidates lack dated membership and remain pending (Gonzalez W6; Browning,
  Neal and Milano W7). Legacy non-GSIS depth identifiers stay pending, not a fatal whole-feed error.
- **4: fixed setting translation, no scoring fork.** `league.partialLineupAllowed="YES"` now maps
  to ordinary `lineup.invalidPenalty="none"`, overriding the supplied base's `zero_top_starter`.
  The previous value came from `LegacyNFL().Lineup.InvalidPenalty`, citation Rulebook §3, through
  `leaguerules.Fill`, not an imported MFL penalty. Raw decode previously dropped the permission.
  Blank/NO retain the supplied base; unknown permission is a translation problem. Scoring's
  `FranchiseWeekScore.judge -> nativecheck.ApplyPenalty` is unchanged and checks only settings.
  Existing recorded leagues are not silently rewritten: the disposable copy applies the translated
  setting through the ordinary commissioner rule command (revision 8742).

| Penalty row | MFL | Before app | After app | Explanation |
|---|---:|---:|---:|---|
| W1 NY Giants | 208.20 | 179.70 | 205.70 | Restores Za'Darius Smith 26; Roquan Smith -2.50 residual remains |
| W2 NY Giants | 205.65 | 182.20 | 205.65 | Restores Cam Ward 23.45 |
| W7 Cincinnati | 145.60 | 117.50 | 145.60 | Restores Jameson Williams 28.10 |

Held MFL weeklyResults counts those starters' points; the imported settings and observed totals
provide no evidence of an automatic highest-starter deduction for these partial lineups. We did
not alter scoring formulas to match MFL or infer other residuals' credit/rule root causes.

## Same 148-row comparison

| Week | Exact | Within .01 | Differ | Unavailable |
|---|---:|---:|---:|---:|
| 1 | 4 | 1 | 2 | 8 |
| 2 | 6 | 0 | 1 | 8 |
| 3 | 8 | 0 | 2 | 8 |
| 4 | 9 | 0 | 4 | 9 |
| 5 | 13 | 0 | 0 | 10 |
| 6 | 8 | 0 | 2 | 14 |
| 7 | 13 | 0 | 4 | 14 |
| **Before** | **45** | **1** | **13** | **89** |
| **After** | **61** | **1** | **15** | **71** |

18 additional team-weeks complete: 14 exact, four differ. Two previously complete differences
also become exact when the penalty is corrected. Full rows: `APP-SCHED-9-comparison.tsv`.
`APP-SCHED-9-absent.tsv` records each disposition, deciding evidence and new zeros' GSIS/game,
team and depth snapshot. Its denominator is **479** starter player-week occurrences: the prior
report's 472 stat-absence occurrences **plus seven identity gaps**, not 472 including identity.

| Disposition | Before | After |
|---|---:|---:|
| Positive-snap zero | 141 | 141 |
| Exact-week Out zero | 53 | 53 |
| Complete snaps + PBP, no snap zero | 0 | 55 |
| Dated schedule-bye zero | 0 | 32 |
| Unresolved | 242 | 187 |
| Bye candidate, affiliation missing | 36 | 4 |
| Identity gap | 7 | 7 |
| **Unavailable total** | **285** | **198** |

**Bills: MFL 6–1 / PF 1912.35; native remains unavailable, 0/7 complete.**
Zane Gonzalez lacks membership in the deciding historical depth snapshots for W1–6;
current MIA membership cannot be carried backwards. W4 also lacks Terry McLaurin, W6 Tyler
Warren, W7 Calvin Ridley. Displayed 0–0/PF 0 is not a computed season result.

## Verification / limitations

Candidate production ref **db6b642bc**:
- Full touched source/history packages, `go test -race`: **43 PASS / 4 SKIP** source and
  **259 PASS / 4 SKIP** history, counting top-level tests. Including subtests: 122 / 424 PASS.
- League raw decoding + settings translation race: **10 PASS / 1 SKIP**, **7 PASS** respectively.
  Total **319 top-level PASS / 9 opt-in SKIP**, zero failing tests in these batches.
- Actual failing-first regressions: unknown bye evidence; ambiguous rounded partial snap file;
  conflicting completed game. Initial new API tests also failed to compile before implementation.
- SQLite backup of the previous disposable profile, ordinary feed/results/standings:
  initial successful run PASS 516.32 s. Final candidate replay PASS **169.18 s**; all seven
  result JSONs and standings byte-identical, ledger idempotent at **10478** records. Non-race
  measurement; not GUI or independently reviewed evidence. No original profile writes.
- Plain **`make build` PASS**, Wails binary built in 21.559 s; not launched.
- Commit hooks PASS, no bypass. **`make lint` is NOT green**: inherited bloat tiny_files=26
  exceeds baseline 25. A read-only count of all starting-ref production files reproduces 26.
  Additional component checks: placement manifest/root ratchet and frontend transport PASS;
  inherited source-branch gate rejects mflsource/schedule.go and native_forks=20 > 19;
  inherited boundary gate rejects commissioner_host.go startValuePreparation. All implicated
  files/allowlists/baselines are unchanged from 48bacfac. No unrelated gate was weakened.
  Full verify not claimed.
- Handwritten affected production footprint: **1034 -> 1391 physical lines**, eight files,
  against 48bacfac. Tests/docs separate. This is added behavior, not a debloat claim.
- One-use measurement tests removed after checking callers. Pre-existing untracked sched8
  harness and cache untouched. Ignored artifacts/logs remain in `.tmp/sched9/`.

Shared weekly/PBP/snaps/injury bodies are unchanged from part 8. Added deciding raw hashes:
- Historical depth: `ecc33eb6f332f6604c02ccd065b0c5efe22d91b4e517de1a2d234ed58c3e93db`
- Season bye declaration: `c73046ebb93c03e62f519cae64138ecfaed63ddfcfe5ddd5e194619dfc1ac4d1`

## Debt left

198 unavailable player-weeks / 71 unavailable comparison rows, including seven Childress alias
gaps; missing historical membership and unreconciled snap coverage; Bills season result still
blocked. 15 complete score differences remain (largest absolute team residual 3.50), not diagnosed
as scoring-rule errors. Secondary ESPN/Sleeper adapters, pinned absence embedding, rare return/
lateral/two-point/kicker completeness, whole-season read cost, prior 76 roster backfill refusals,
inherited gates and independent/live review remain. No remote push: branch has no upstream.
