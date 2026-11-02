# APP-MERGE-9 — copied-profile regression repairs

2026-11-02 · Bee · build/INTEGRATE-1102

**Partial acceptance:** targeted tests, frontend checks and plain build pass. The copied desktop
shows live players, recovered standings and named Home players. The requested five-minute live
acceptance does **not** pass: the desktop terminates with signal 9 before five minutes. Its memory
use needs investigation; do not treat the absence of deadline log lines after termination as success.

## Per item

### 1. ESPN summary numeric week — already committed, verified

- Origin: `2ad87c55f`; the summary's numeric `header.week` was decoded into `RawWeek`.
- Fix: inherited `f9af9632`, accepting the real summary shape without losing game season/week.
- Reverted only the decoder for the failing-before run, then restored it. The held-summary test
  `TestSummaryCarriesRealNumericWeek` fails with the exact numeric-week unmarshal error before
  the fix. The targeted livegame race batch passes afterward.
- Desktop evidence: **774 players from 14 games**, season 2026/week 8, rather than zero.

### 2. Launch read timeouts — lock scope and repeated batch folds fixed

- Lock origin: `98ce88c3b`. Its deferred schedule synchronization ran before the deferred
  `RefreshMu.Unlock`; launch synchronization also explicitly held `RefreshMu` across acquisition
  and scoring. Neither the five-minute history-worker delay nor local-seat credentials caused
  the initial two-minute failure: it reproduced before that worker could start.
- Amplifier: `a0e973be6`'s automatic-stat batch repeatedly checked/folded the entire growing
  ledger for each player's approval. Stack captures and the 1,200-player regression reproduce
  the final credential-admission deadline as a symptom of that preceding work.
- Fix: `6cc8ab15`. Release `RefreshMu` before acquisition/scoring; serialize schedule
  synchronizations with their own mutex. Validate the common locked ledger once per batch and
  fold manual stat pins once. Keep per-approval evidence checks, canonical preparation, retry
  checks, credential admission, sequential revisions, audit writes and atomic transaction scope.
- Finished the inherited `TestScheduleEvidenceDoesNotHoldRefreshReadLock`: it fails before
  the change because foreground lock acquisition is blocked, and passes afterward.
- `TestAutomaticStatFeedAdoptsSeasonWithinReadBudget`: two generations of 1,200 players;
  before, generation 0 exceeds its 15-second budget with the reported admission error; after,
  generations take **7.46 s / 8.52 s with race** (about 0.4 s without race).
- Added admission-revocation/atomic-rollback/retry coverage. Updated four reviewed boundary
  source fingerprints; no boundary exemption or authentication bypass.
- Desktop launch refresh completes in **28–38 seconds**, instead of the observed 120-second
  timeout. War Room's partner read recovers to 31 partners. Home recovers record/seed/week and
  live scores. See the runtime limitation below before claiming complete acceptance.

### 3. Player IDs — no independently dropped name resolver found

- Traced `a8a31c434` through Home's existing HQ identity resolution and the store's public
  side-qualified identities. The resolver remains present; it was not removed by the merge.
- No speculative resolver rewrite or separate functional commit. Item 2 unblocks the reads;
  early copied-profile screenshots still show stand-ins while startup reads are settling.
- A subsequent copied-profile desktop shows **McDuffie, Trent; Parrish, Jacob; Stokes, Eric;
  Surtain II, Patrick; Murphy, Byron** in the TOP 5 ticker, alongside week-8 live scores.
- Existing frontend regression asserts both THIS WEEK rows and TOP 5 resolve HQ identities over
  results-read stand-ins. Existing store identity tests preserve side-qualified names and avoid
  invented facts. These pass. War Room shelf names are not independently live-accepted after
  the final restart; this remains a verification gap, not a proven separate name-code regression.

### 4. Painted Home selection labels — fixed

- Origin: `5110ff1d3` routed the sibling overlay through `HomeAct`; absent children caused
  the shared control to render its accessible label as visible fallback content.
- Fix: `1239ba41`, explicit empty children on that overlay only. Its `aria-label`, sibling
  structure, focusability and selection behavior remain intact. `abee0930` wraps the source
  line to satisfy the frontend readability contract.
- Strengthened the shared-control test: before, `textContent` is `Select RACE` instead of empty;
  after, content is empty, accessible name is retained, clicking selects, and interactive
  descendants remain siblings. Copied desktop screenshots no longer paint selection labels.

## Verification

Listed tests first; targeted race runs used `-p 2`, one package at a time. No full Go suite.

| Check | Result |
|---|---|
| composition schedule/refresh selection | 13 PASS tests/subtests |
| history automatic-stat selection + public-identity regression | 18 PASS tests/subtests |
| commissioner stat-feed selection | 12 PASS tests/subtests |
| ingestion/livegame selection | 26 PASS tests/subtests |
| Vitest src/app | 180 files / 1,833 tests PASS |
| TypeScript no-emit | PASS |
| boundarycheck | PASS, 170 frozen functions/sources |
| plain `make build` | PASS |
| commit hooks | PASS; no bypass |
| five-minute copied-profile desktop acceptance | **BLOCKED: process killed before five minutes** |

Evidence remains untracked under `.task-evidence/app-merge-9/`: failing-before outputs,
per-package JSON results, frontend/build logs, stack captures and copied-desktop logs.
Disposable screenshots/profiles are in operator scratch, not the repository. The real profile
was never launched or modified. Temporary diagnostic test code was removed.

## Runtime debt / next move

- Uncapped final launch disappeared without a Go panic/deadline log. A second isolated launch
  used only the experimental `GOMEMLIMIT=1500MiB` change (not committed to configuration).
  It recovered Home names and scores, but also exited **137**, around 2m15s. Recorded parent
  RSS reached **4,252,340 KiB** at two minutes despite the soft Go limit. Memory pressure is
  suspected, not proved: kernel OOM records are inaccessible to this account.
- `capped-desktop.log`, `capped-resources.txt` and the scratch wrapper's exit receipt preserve
  that evidence. No desktop from this task remains running. Do not claim a clean five-minute
  run. First next step: profile peak heap/RSS during a copied-profile launch with live/read
  sources distinguished; then repeat the full five-minute acceptance without experimental
  environment changes and finish the War Room shelf-name check.
- A copied-history standings diagnostic also exceeded a 20-second read deadline in held
  score-evidence/directory work. Its CPU profile shows absence-depth reconstruction, scoring
  columns and GC. This is a lead for profiling, not a proved cause or authorization to weaken
  evidence checks.
- Inherited keyring-unavailable calendar fetch, backup-retention invalid-entry warning and
  Wails missing-model-alias build warnings remain. No new keyring/sign-in workaround added.
- Contrary to the brief's no-remote description, `origin` points to a local bundle, not a
  receiving repository. Commits remain local; nothing was pushed to that bundle.
