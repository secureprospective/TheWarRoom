# Legacy NFL behavior explorer

Independent, read-only PWA for the 32-team dynasty league's fourteen-season MFL
archive. Explore observed franchise choices, compare like-for-like peers, then
inspect receipts. No runtime dependencies, external fonts or MFL calls. The existing
WarRoom application, databases and raw archive are untouched.

## Open

On Beelink: **http://localhost:8765/**.

From the repository root, generate the private index and start the local server:

```sh
python3 tools/league-history/build_archive.py
python3 tools/league-history/serve.py
```

The server defaults to loopback. Chromium browsers can install from localhost or
HTTPS; wait for **Offline ready** before disconnecting. All seven views, not just
the shell, work offline. Remote-device installation needs private HTTPS or a local
tunnel. No HTTPS host or reboot-autostart service is configured. Restart the Python
server after a reboot. Do not launch a second copy while port 8765 is occupied.

## Explore

- **Calendar:** fourteen January–December rows. Click a month heading for that month
  across years, a cell for its month, then a day for its receipts. Switch between
  recorded deals, directed pick/player quantities, net picks and per-30-calendar-date
  counts. Unavailable measurements show dashes, not fabricated zeroes.
- **Compare:** the same period, subject, policy and facets across franchise slots.
  Receiving shares show numerators/denominators, matched peer medians and percentage-
  point differences. Inspect counts, partner concentration/HHI and package patterns.
- **Franchise:** received/sent assets, counterparties and partner matrix, encoded
  exchange shapes, package-size distribution, seasonal trajectories and descriptive
  whole-season outcome associations. Annual trajectories preserve the selected
  month/day and explicit date bounds; only the clicked year is cleared.
- **Weekly:** score/optimal reconciliation and legal-lineup checks under archived
  constraints. Inspect unused points and optimal-set overlap as hindsight benchmarks,
  never ex-ante skill. Acquisitions have a censored three-week roster/start follow-up.
- **League:** source-season summaries, recorded Super Bowl winners, standings and
  recorded draft selections. This auxiliary draft view labels its distinct source-
  season scope and ignores trade-only facets; missing draft results are disclosed.
- **Players:** search by name or exact MFL ID; inspect observed scores, drafts,
  movements and rosters. Latest directory attributes are labelled separately.
- **Evidence:** source readiness, method/eligibility contracts, exclusions, missing
  inputs and searchable inventory of all 1,705 files.

One shared lens carries franchise/player, UTC dates and calendar selection across
analysis views. Prior-result and record-band filters require a selected franchise. Expand **Refine the evidence** for source season, phase, asset kind,
received/sent direction, pick round, draft-cycle horizon, season-listed position,
partner, exchange shape, prior result/record band and exact pick token. Asset,
partner and package facets constrain completed trades only; they never hide draft
or manual records in other channels. Weekly observations ignore trade-only facets;
calendar dates select team-weeks by first NFL kickoff. Receipt dates select
acquisitions, whose follow-up can extend beyond that receipt date range.

Questions supply starting lenses. Save, reopen and delete lenses in this browser;
URLs carry the complete analysis context. CSV exports include all filtered receipts,
not just the displayed page. Findings JSON includes context, definitions, denominators,
exclusions, peer/distribution summaries, weekly reports where applicable, and source-
path provenance. On secure origins, the export fingerprints the exact loaded index
bytes; insecure LAN HTTP reports an unavailable fingerprint rather than guessing. CSV blanks mean unavailable/not applicable; early unencoded-pick
periods and non-trade channels do not export invented pick zeroes. Draft selections
and recorded add/drop IDs retain their own columns.

## Evidence policy and interpretation

Default: completed TRADE records with two encoded sides and no ambiguity flags.
Use **all reported** or **ambiguous only** for sensitivity checks. Empty sides are
unknown consideration, not zero-price exchanges. Unknown assets, comment-pick
references and non-token consideration notes remain inspectable. Commissioner entry
is neither proof of initiation nor grounds for automatic exclusion: 2,610 of 2,634
completed records were commissioner-entered. Manual roster adjustments and proposal/
status notices are separate channels, never completed-trade denominators.

- Pick receiving shares use contextual deals in source seasons with observed pick
  encoding, **before asset facets**. Matching facets constrain numerator legs.
  2013–2016 have no observed structured pick tokens; quantities/shares and complete-
  package classification are unavailable there, not proof of no pick preference.
- Player-position shares exclude receipts whose incoming positions are unobserved.
  Position attributes are source-season listed values, not verified decision-time
  attributes. Counts describe known matching legs, not unobserved attributes.
- Per-30-date measures divide by selected calendar dates intersecting the archive
  and any verified tenure. They are **not permitted-trading-opportunity rates**.
- Phase and draft-cycle anchors come from recorded draft spans and listed NFL
  schedules; they are archive proxies, not verified trading deadlines. DP tokens
  retain the source-season year; past/unresolved cycle encodings are not relabelled.
- Prior records are retrospectively reported franchise context. Require complete
  prior regular-week history and at least four games for record bands; exclude
  regular-labelled consolation games after the first archived playoff week.
- Result availability is last listed NFL kickoff +24 hours, **not an archived final
  result clock**. Post-result recording windows end at the next result or seven
  days, clip to selected dates and stop at UTC midnight one day before the capture
  date because the capture timezone is unknown. Mature weekly cohorts use that
  conservative cutoff too. Future T-coded forecasts are not counted as played ties.
- Hindsight benchmarks require roster membership, unique players, legal total/
  position/offense/defense counts, and actual/optimal score reconciliation within
  0.02 after explicit adjustment. Kicker is excluded from offense counts. Omitted
  individual scores may be zero only through both reconciliations. Missing optimizer
  fields exclude the benchmark, not an independently scored win/loss observation.
  Unexplained 2014/2015 home-score +3 residuals remain excluded, not guessed away.
- Usage observes the next three matured league weeks in the receipt's analysis
  season, starting after its timestamp. Missing team-weeks, recorded exits, tenure
  boundaries and archive-end truncation censor follow-up; later weeks never fill
  missing ones. It is not injury/eligibility-adjusted usage or continuous custody.
- Package size is complete encoded inventory, **not value, price or trade quality**.
  Associations do not establish causality, stable traits, panic, collusion or winners.

2026 is partial. Standings use wins/points ordering, not the complete MFL tiebreaker.
Roster snapshots never establish uninterrupted ownership. Offer linkage, initiator/
response clocks, historical deadlines/exceptions, dated values, age/opportunity/cap
inputs and continuous custody remain unavailable or unverified. Their dependent
metrics stay blocked rather than approximated with unrelated proxies. The roadmap
and 32 metric contracts are in `research/`; not every planned inference is shipped.

## Verified owner tenures (optional)

Without a historical who-owned-which-franchise-and-when map, profiles are **franchise-
slot histories**, not fourteen-year manager biographies. Aliases never prove owner
continuity. Supply the ignored `tools/league-history/data/tenures.json` and rebuild:

```json
[
  {"id":"owner-a-0025","name":"Owner A","team":"0025",
   "from":"2018-01-01","to":"2026-01-01",
   "evidence":"Verified league ownership record; UTC boundaries confirmed"}
]
```

Exactly those six fields are accepted. IDs must be unique; teams must exist; dates
are half-open UTC intervals `[from,to)`. Invalid dates, overlaps and extra fields
fail compilation. Leave gaps/transition days unmapped when boundaries are uncertain.
The owner filter clips events and matches peer dates. Weekly benchmarks require
whole anchored weeks within the tenure; follow-up never credits a successor's starts
or exits. This maps observed franchise activity under ownership, not offer initiation.
Keep evidence free of contacts and secrets. Current deployment has no tenure map.

## Data and privacy

Schema **2**, behavior definitions **1**. Generated `data/archive.json` and
`data/revision.json` are necessary for deployment and ignored by git. Latest manifest
entry wins per file. Current build: 19,907 events, 7,921 players, approximately 26 MB;
1,559 usable sources, 97 expected errors, 49 empty. Raw totals reconcile to 2,634
completed trade records, 1,725 structured-pick deals and 3,780 encoded pick transfers.
Broader ambiguity flags identify 137 comment-pick-reference records, 11 non-token
consideration notes and 361 empty encoded sides. Hindsight eligibility is 4,215 of
5,590 scored team-weeks; 320 future/unscored rows are excluded.

Owner contacts, credentials, messages and arbitrary transaction comments are omitted;
draft notes are retained. The allowlist server denies source scripts, raw data,
tenure input, directory listing and traversal. Host checks reject browser DNS-
rebinding hosts. For private reverse-proxy hosting, use a permitted upstream Host;
do not serve the entire repository or publish private generated data without approval.

Compiler output is replaced atomically, preserving the previous complete index on
failure. The service worker validates fetched index format before replacing its
cached copy; scope is this app only. It caches the full index, refreshes by content
revision and falls back offline. Bump shell/cache versions for incompatible releases;
bump schema/definition versions when contracts change. Clear this origin's site
storage to remove cached data and saved lenses.

## Verify

```sh
cd tools/league-history
python3 -m unittest -v
node --test model.test.mjs behavior.test.mjs
PLAYWRIGHT_CORE=/path/to/playwright-core BROWSER_PATH=/path/to/chromium \
  node browser.test.mjs
```

Browser dependencies are optional test tooling, not installed by this project.
Tests cover raw invariants, 168 cells, unknown versus zero, ambiguity policies,
directions/denominators, peer context, trajectories, legal reconciliation, censored
usage, tenure boundaries, saved lenses/deletion, focus, CSV/JSON/source paths,
1440/390px layouts, all channels, full offline reload and source/host denial.
Private screenshots and exports remain in ignored `.impeccable/review/`.
Repository verification: `PATH=/usr/local/go/bin:$HOME/go/bin:$PATH GOMEMLIMIT=1500MiB
make verify` from the repository root. Independent review is not claimed; Christopher
owns acceptance and merge.

**Acceptance check:** reset; select Arizona, Picks received; refine to pick assets,
round 1, received direction; click March 2024. Expect **2 firsts received**, **1 sent**,
and receiving share **2/6 (33.3%)** before asset facets. Compare and Franchise must
retain the lens and expose both receipts. Reset to March 2017: all-policy 143 trades
versus default-policy 10. After Offline ready, disconnect and reload all seven views.

## Extension points

- `build_archive.py`: source normalization, deduplication, provenance, atomic output.
- `behavior_archive.py`: behavioral flags/anchors, dated positions, weekly constraints,
  prior-result chronology, optional validated tenures.
- `behavior.mjs`: pure subject-relative ledger, shared selection, denominators, cohorts,
  trajectories, windows, follow-up and CSV receipts.
- `behavior_views.mjs`: analysis views; `history_views.mjs`: league/player/source views.
- `model.mjs`: shared presentation/CSV primitives and compatible raw-filter helpers;
  those raw helpers do not implement behavioral eligibility policies.
- `app.js`: URL/local-lens state, validation, interaction and findings export.
- `style.css`, `sw.js`, `serve.py`: inherited visual system, offline and local boundary.

Keep raw reads and new provider adapters in the compiler. Give each new metric an
explicit unit, eligible population, denominator, source coordinates, missingness and
interpretation limit. Do not copy final-season attributes into decision-time inputs.
Icons are original calendar-grid artwork; no third-party image/font assets ship.
