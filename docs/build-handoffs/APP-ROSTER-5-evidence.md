# APP-ROSTER-5 — incomplete; copied-profile baseline only

No production code changed and no transaction was appended. This is not delivery
of the importer, pre-seed reconstruction, or refusal cleanup. Local starting ref:
`dca039191`. No remote contact, network request, live-config access, clock override,
SQL ledger write, or real MFL move occurred.

## Steps and missing implementation

1. **Not implemented:** transaction translation, stable replay identities, import
   wiring, and the independent transaction-follow league setting.
2. **Not implemented:** backward-consistent reconstruction and unverified contracts.
   `internal/commissioner/native_reading.go:40` explicitly refuses an as-of read
   before the seed. `internal/commissioner/correction_projector.go:46` refuses an
   add without a previously recorded contract. These are implementation gaps,
   not evidence that Christopher's authorized extension is impossible.
3. **Baseline exercised; post-fix proof unavailable:** results below. All exact
   refusal messages are in `APP-ROSTER-5-refusals.txt` beside this document.

**Policy question for Christopher/ClaudeBox before assigning unknown contracts:**
`internal/leaguerules/contracts.go:5` defines maximum years and experience-based
minimum salaries but no default unknown-acquisition salary/term. The only matching
helper, `internal/leaguerules/draft.go:148`, expressly supplies the startup draft's
lowest minimum. The brief authorizes an unknown-contract fallback, but does not
specify its salary/term or identify an existing default setting. Is the intended
fallback one year at the lowest league minimum, or an experience-based minimum?
Do not silently equate these policies. Queue integration is also still unwritten.
No fallback value was invented in this run.

## Copied-profile measurements

SQLite backup API copied both requested scratch-profile databases into disposable
worktree-local profiles. Archived bodies were read from that copy; existing Go
parsers/validators checked transactions, players, weekly results, and held NFL
schedules. The validated transaction archive still contains **895** rows. Source
transactions, roster reference, and weekly-result body hashes exactly match the
prior APP-ROSTER-1/2 evidence inputs. The copied followed ledger started with two
entries: create and seed.

### Comparable baseline without kickoff schedules

The existing ordinary `BackfillLineups` path, with real receipt time:

| Pass | Appended | Refused | Full direct importer call |
| --- | ---: | ---: | ---: |
| First | 148 | 76 | 40.639965552 s |
| Repeat | 0 | 76 | 10.446388703 s |

Bills weeks **1–7 all recorded** here. Refused baseline/repeat: **76 → 76**.
These are not before/after an implementation change and not app-launch timings.
No transaction-replay idempotency was proved; zero applies only to lineup repeats.

### Important limitation exposed by held kickoff schedules

A separate fresh backup used the archived, validated **2026 week-7 and week-8** NFL
kickoffs in the normal `LineupEnvironment`. Other historical weeks had no held
kickoffs; none were invented.

| Pass | Historical appended | Historical refused | Full direct importer call |
| --- | ---: | ---: | ---: |
| First | 117 | 107 | 32.164532778 s |
| Repeat | 0 | 107 | 9.955029799 s |

All 32 week-7 lineups now refuse with **"Roster history before the starting
observation is not recorded"**. The former schedule-free baseline refused one of
those teams, so this adds 31 refusals. Bills weeks 1–6 remain recorded, but **week 7
is not recorded** on this schedule-aware copy. The starting observation is
`2026-11-01T03:09:10.095212087Z`; week 7 and week 8 started before it.

This proves the previous 76 figure is not a complete kickoff-aware acceptance
baseline. Merely translating forward moves cannot close this gap: generic
pre-observation reconstruction is necessary.

### Explicit week-8 starters

The archived weekly-results body contains week-8 starter selections but not scored
week-8 results. `mflsource.Lineups` intentionally skips unscored sides at
`internal/leaguemigration/mflsource/lineups.go:23`. Consequently a historical
backfill does **not** exercise the requested in-play week.

A separate explicit selection of those archived week-8 starters was mapped to
ordinary `RawLineupChange` inputs and sent through `leaguemigration.SyncLineups`.
This did not change source `Scored` flags or claim these were final results.

- Without kickoff schedules: first appended 32, refused 0, 9.017469177 s; repeat
  appended 0, refused 0, 0.054990028 s.
- With held week-8 kickoffs: first appended 0, refused **32**, 1.359937072 s; repeat
  appended 0, refused **32**, 1.373222894 s. Every refusal is the pre-observation
  history guard, including franchise.18, Panthers, Dolphins, and Saints.

These are archived-copy importer measurements, **not** a current live week-8 check,
League Pulse UI proof, launch measurement, or integration-branch worker test.
The addendum's live 15263 membership refusal was not reproduced from this older
archive. No claims are made about the current remote/reference state.

## Four franchise differences explained, not repaired

The original membership/status baseline remains **28/32 equal**. No roster entry
changed during these measurements. Exactly three validated transactions follow
the seed observation and explain all four listed differences:

- Unix 1793505951: FREE_AGENT adds 10976 to 0003.
- Unix 1793506094: FREE_AGENT adds 15263 to 0018.
- Unix 1793506821: TRADE sends 14467 from 0025 to 0015; the other side gives two
  future picks (0017/2027/4 and 0025/2027/5).

Thus 0003/0015/0018 are missing their reference additions and 0025 retains 14467.
This is an explanation from validated source evidence, not an end-of-ledger replay
or contract/pick equality proof. The authoritative baseline TSV remains
`APP-ROSTER-1-baseline.tsv`. **32-team end-of-replay equality remains unproved.**

## Checks and hygiene

- Targeted race batch: **26 test/subtest PASS, 0 FAIL, 0 SKIP**. Selection:
  `Test(RealTransactions|SyntheticTransactions|PlayedStartingLineups|MigrationLineupSync|NativeAssetsAsOf|LineupRosterAt)`
  over transaction ingestion, migration packages, and commissioner. Some names
  matched no tests; only emitted test PASS events are counted.
- The pre-test heavy check printed STOP because another full race suite was running;
  the shell command erroneously continued into this targeted batch. This was a
  workflow error, not a test failure. Subsequent heavy checks passed before jobs.
- Plain `make build`: PASS, packaged; binary not launched. Existing Wails binding
  warnings remain. No frontend source changed or separate tsc run performed.
- No full suite, independent review, GUI test, integration/cherry-pick, or feature
  acceptance occurred. Production files, including both protected composition
  files, remain unchanged.
- Disposable profile copies and one-off Go proof harnesses were checked for active
  processes/open files and deleted after the measurements. Ignored text logs remain
  under `.tmp/roster5/`. Pre-existing `.golangci-cache/` was left alone.

## Debt left / next move

Resolve the fallback salary/term explicitly, then implement the already-authorized
generic pre-seed history/contract intake and inverse-consistent projection. Translate
all supported transaction kinds through ordinary commands with stable identities,
wire import and the independent sync setting, and rerun both schedule-aware and
schedule-free copies. Prove transaction repeat-zero, 32-franchise replay equality,
Bills 1–7, week 8, and actual full-sync before/after time. No implementation step
is complete in this run.
