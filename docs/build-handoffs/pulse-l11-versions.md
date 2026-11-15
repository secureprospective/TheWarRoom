# Workflow reconciliation seam

Flow: validated workflow + authenticated principal → league admission → command read admission →
recipient revision observation at the committed head → copied version tags → unchanged or workflow
engine/reply cache → fresh read admission → reply. The existing reply cache remains independently keyed.

`Hosted.Versions` is an optional `WorkflowVersions` owner. Its bounded head memo is per principal/league;
only a changed head recomputes `TargetCommissionerLedger(...).Revision`. The version source must share
observations with the stream publisher, return recipient-authorized copied snapshots, and never roll back
an observation. Read tags retain every matching published scope, not a synthesized aggregate key.
`commandcatalog.ReadDependencies` owns the matching policy. Tags are captured before engine execution;
a concurrent publication cannot relabel an older reply. Authentication is rechecked before either reply.

Production intentionally has no source: no tags, no unchanged, full existing replies. The head brain must
connect the L5 publisher through `VersionSource`; no L5 worktree was read or copied. The snapshot also
has live epoch and values run fields, currently empty without an adapter. The model run identities found
in `history.Store.LatestRun(season, RunModel)` are seasonal; selecting one independently of the actual
value engine snapshot would mislabel replies, so no speculative lookup was added.

`versions.read` is registered and generated but executes only through the hosted workflow, not the legacy
local native method binding. It returns the roster season, all authorized source topics, pin digest,
server time, and the public `ReadingEpoch` boundary. It always answers in full so reconciliation cannot
hide a changed authentication generation. The generation is SHA-256 of account, league, a domain-separated
session credential fingerprint, and the caller's sorted owner/caretaker seats and roles; no bearer or raw
ledger position is returned. `Requests.Unchanged` records conditional replies without an engine/cache hit.

Evidence: hosted tests exercise disconnected correction, exact scoped dependency bumps, immutable older
reply tags, restart resync, model-only publication, session replacement, revoked in-flight conditional
read, private recipient isolation, legacy full replies, payload cap, engine computation counts, and the
Go desktop decode/re-encode path. Frontend tests exercise conditional transport and tag validation; the
existing hosted-query tests cover crossed pings and stale/session-fenced results. This is not a live
publisher integration or all eight C5 scenarios at the server seam: stream crossing, membership removal
while an engine is blocked, and an event during engine execution still need integrated publisher evidence.

Known slop: registry reads assume a local Wails binding, forcing an explicit hosted-only refusal handler
for `versions.read`. A future hosted-read registration seam would avoid that dummy local exposure; do not
add a local publisher or alter composition/run.go to make a hosted reconciliation command pretend local.
