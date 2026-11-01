# League value preparation

Desktop composition owns one `appvalue.Preparation` for the application lifetime.
`finishStartup` starts its single worker after stores are published, without waiting for
valuation. It discovers seeded leagues at startup and once per second thereafter; each
league's current roster season is prepared. Screen-selected league/season scopes also join
the worker. A sweep catches refresh commits, rule/roster/contract edits, params changes,
new leagues and rollovers without relying on individual mutation callers remembering a hook.

## Read and freshness contract

`commissionerHost.NativeValues` → `Preparation.Read` → `Source.Read` in prepared-only mode.
The screen reads the current records and derives the same inputs as the producer; it never
reads the weekly archive, builds a player Book or runs valuation. Each league/season retains
its own completed pass; franchise headers and inspectors share it.

A pass is usable only when the SHA-256 key matches the history connection's commit revision,
all valuation params and the league inputs (season, scoring, lineup, ownership, contracts,
franchises and scoring-week limit). Its as-of interval starts at preparation and ends at the
next future observation. Clock rewind, observation visibility, any history commit, and
changed league inputs or params refuse reuse. A second revision check fences commits during
screen assembly. The producer also checks the revision after computing and does not publish
across a commit. Params/input changes during production cannot match a later screen's key.

A miss returns a scoped, non-error `preparing: true` reading, with empty players and age bands
and unavailable Best N; no fabricated known zero or stale values. Missing wiring and actual
input/preparation failures remain errors. A producer failure is associated with that key and
retried by the worker; it is not mislabelled as ongoing preparation. Successful publication
emits the existing season-change signal. HQ and War Room reload under their existing request
and scope fences; only value regions say “Values are being prepared”. Other regions stay
independently readable. The app cancels and joins the producer before closing databases.

No engine/model math, league-origin branches, persistent tables or dependencies changed.
History's latest-per-source query now uses SQLite's unique MAX(as_of) row instead of a
redundant ROW_NUMBER window; highest-priority source selection remains the second window.

## Evidence and limits

Copied-profile composition harness: cold-after-startup direct read 49.167 ms, first workflow
40.269 ms (preparing); background ready at 18,499.677 ms; first prepared workflow 33.489 ms,
repeat 31.025 ms. Bills: 35/35 players, Best 21 243.26284592650194, rank 3/32 unchanged.
Before preparation: cold 19,143.493 ms; query-only optimization 18,077.569 ms.
These timings exclude DB opening and do not claim values finish computing in 200 ms.

Race tests exercise startup-before-first-read, no screen-side production, shared in-flight
work, rejected publication across a commit, visible producer errors, and roster, contract and
scoring-setting invalidation against a direct fresh valuation. Imported 25-player model
parity remains exact. DOM tests exercise preparing → publication → ready for both workspaces,
retain the cap region, and reject preparing payloads containing completed values.

Remaining cost is mostly history resolution and league scoring, not per-player fitting.
Preparation is in memory, so a restart or unrelated history commit requires another pass.
Single-worker sweeps serialize multiple leagues; memory holds tracked scopes for the session.
Real desktop visual verification and hosted-server valuation plumbing remain unverified/debt.

Boundary review: `finishStartup` guards worker launch only on acquired-resource success;
`startValuePreparation` binds the lifetime context, worker group, source ports and existing
notification/log sinks. Its only branch suppresses cancellation logging. League discovery,
seeded-roster selection and freshness decisions live in appvalue, not composition. Only
these two adapters and their changed source fingerprints were accepted; unrelated pre-existing
migration/platform/binding fingerprint mismatches were not refreshed.

Additional gates: `make lint` remains blocked by the inherited tiny-file baseline mismatch
(26 both at f954d1f6 and here, baseline 25). Standalone placement/transport gates pass;
`make boundary` still rejects the untouched migrationInputs fingerprint. These are not
claimed passing or silently waived. Full commit lint/secret/interface hooks still apply.

Affected handwritten production footprint versus f954d1f6 (physical splitlines, full files,
including added service): 1,960 → 2,151 lines. Tests separately: 467 → 753. Generated bindings,
docs and evidence are excluded from those counts. This is an ownership/freshness improvement,
not a source-size reduction. A screen-side-computation mutant was rejected by the cold-read
assertion; restored source matches the race-tested candidate.
