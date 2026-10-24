# Client/server 27: patched toolchain and B acceptance

## Scope and result

The module now requires Go 1.26.10 (above the approved 1.26.9 minimum).
CI setup-go already reads go.mod; both Go jobs now report the selected version
and use `GOTOOLCHAIN=local`, so an older runner fails rather than silently
switching compilers. Existing build commands inherit the module floor; no new
build wrapper or application rewrite is needed.

`go mod tidy` removes the now-redundant equal-version toolchain directive and
classifies the already-used x/crypto v0.55.0 as direct. No dependency version or
checksum changes. No accounts, bindings, placement or runtime files changed.

Flow: go.mod floor → setup-go/local Go selection → existing vulnerability,
lint, race and frontend gates. An older selected compiler must refuse before
loading/building application packages. This changes build admission, not data.

## Exercised evidence

Baseline: `7bbc2da773ff776e2c9ec69c6dd62983d95d550e`. Commands below ran on the
slice's changed Go/CI files with Go 1.26.10, linux/amd64. Local raw output is in
`.tmp/cs27/` (ignored; not durable acceptance evidence).

- `cd frontend && pnpm install --frozen-lockfile && pnpm build`: pass.
- `GOTOOLCHAIN=go1.26.4 go list -m`: intentionally exits 1 with
  `go.mod requires go >= 1.26.10`; this is the old-toolchain rejection proof.
- `govulncheck ./...`: pass, zero reachable vulnerabilities; three vulnerabilities
  in required modules are reported as not called. This is not a claim that every
  transitive module is vulnerability-free.
- `go test -race -count=1 ./internal/db ./internal/mfl
  ./internal/ingestion/leaguearchive ./internal/domain ./internal/store/state`:
  pass. Covers migrations/rollback, newer-file byte preservation, matched-backup
  barriers, eleven restore interruption points, retention, weekly checks,
  host/redirect/body limits, archive rejection and failed mirror publication.
- Targeted composition race tests: shutdown cancellation/join before pool close,
  failed refresh preserves rules and mirror, bid deadline, startup capture,
  maintenance daily capture, offline restore CLI including instance-lock refusal,
  owner-only files, same-home roster/commissioner restore and different-home
  identity finding. Pass. Selection:
  `TestShutdownWaitsForFactsWorker`, `TestShutdownCancelsJoinsThenClosesDatabases`,
  `TestRefreshFailureLeavesRulesAndMirrorTogether`,
  `TestNativeBidActionsRefusedAtAuctionDeadline`,
  `TestMatchedRestoreSameHomeKeepsCommissionerAndNativeRoster`,
  `TestBackupSnapshotDifferentHomeCommissionerRightsFinding`,
  `TestStartupCapturesMatchedSetBeforeOpeningPools`,
  `TestDatabaseMaintenanceCreatesDailySet`,
  `TestRestoreFlagRestoresASetWithTheAppClosed`, `TestConfigDirAndLogsAreOwnerOnly`.
- `go test -race -count=1 ./internal/store/history -run
  '^TestMatchedRestoreKeepsPopulatedNativeResults$'`: pass; populated result
  replay after restore.
- Final full-suite gate is the mandatory pre-push `make verify` (this slice changes
  CI). Its `go test -race -timeout 60m ./...` is the end-of-slice all-package test,
  not an additional manual full run. See PR checks/report for outcome.

A desktop binary was compiled without regenerating bindings, using
`go build -tags desktop,production,webkit2_41 -ldflags ... -o .tmp/cs27/thewarroom .`.
`go version -m` reports go1.26.10. It was not launched. This preliminary artifact
is not a live acceptance result; build the clean candidate below for the gate.

## B is not accepted

| B requirement | Current disposition / next evidence |
|---|---|
| Patched Go and vulnerability check | Exercised above; CI must pass on the PR candidate. |
| Migrations, wrong-state, shutdown, backup/input regressions | Exercised above; no second implementation. |
| Different-home commissioner rights | Known failure, not an unknown: `localCommissionerSession.Principal` still hashes dataDir + uid in `internal/composition/commissioner_host.go`. The passing finding test expects Visitor. Accounts/aliases must resolve this before claiming the original B rights guarantee; do not silently waive it. |
| Idle CPU / due-only clock | Desktop `nativeWaiverClock` still ticks each second in `internal/composition/native_schedule_app.go`. The hosted due worker is not desktop acceptance. No target CPU profile or <2% result established here. |
| Hourly unchanged-input skip / fast reads / overlap cut | No integrated B measurements or per-test overlap report established by this slice. Existing cache/skip code is not measured acceptance. Collect unchanged-hourly-work logs, command latency and overlap evidence; bound any remaining fixes separately. |
| Integrated live/restore gate and Christopher click | NOT RUN. Requires the exact clean candidate, throwaway profile, evidence and Christopher's disposition below. Slice 28 remains blocked on B acceptance. |

This one-day closeout cannot honestly mean all B work is done. Remaining B effort
is unmeasured; the coordinator must size any fixes and record contingency use.

## Christopher / Claude-OS: exact remaining gate

Use a disposable Claude-OS desktop with its network disconnected (no real MFL
calls), never Christopher's live desktop/profile. Keep it offline throughout.
Choose an approved **verified matched production backup set** produced by
`db.Backups.Capture` (three standalone DBs plus manifest), not copied active
DB/WAL files. Supply its path as `SOURCE_SET` below; do not put private data in git.
The backup must be compatible with the candidate schema. Retain its manifest.

Build from the clean PR head in this worktree, using existing built frontend:

```sh
export PATH=/usr/local/go/bin:$HOME/go/bin:$HOME/vendor/gh/bin:$PATH
export GOTOOLCHAIN=go1.26.10
SHA=$(git rev-parse HEAD)
test -z "$(git status --porcelain)"
PKG=github.com/secureprospective/TheWarRoom/internal/composition
go build -tags desktop,production,webkit2_41 \
  -ldflags "-X $PKG.version=cs27-gate -X $PKG.commit=$SHA" \
  -o .tmp/cs27/thewarroom .
go version -m .tmp/cs27/thewarroom
sha256sum .tmp/cs27/thewarroom
```

Transfer that binary to Claude-OS through the existing deployment channel,
without replacing the installed/live executable. On the VM, set `BIN` to its
absolute path, and `SOURCE_SET` to the approved backup directory, then run:

```sh
export DISPLAY=:0
export XDG_CONFIG_HOME=$(mktemp -d "$HOME/cs27-gate.XXXXXX")
DATA="$XDG_CONFIG_HOME/TheWarRoom"
"$BIN" -version
sha256sum "$BIN"                 # must match the built candidate
"$BIN" -probe                   # initialize disposable current DBs; offline
mkdir -p "$DATA/backups"
cp -a "$SOURCE_SET" "$DATA/backups/"
SET=$(basename "$SOURCE_SET")
"$BIN" -restore "$SET"          # validates manifest and restores offline
"$BIN" -probe
"$BIN" -backups
"$BIN"                          # open and inspect; close normally to return
```

Record league/rules/roster/results and Data Sources database-check readings,
startup logs and screenshots. In that disposable profile only, create a clearly
named test native league (no MFL action). While the app is open, from another
terminal with the **same** BIN/XDG_CONFIG_HOME, run `"$BIN" -restore "$SET"`:
expect exit 1, “app closed”, and no state change. Close normally, then run:

```sh
"$BIN" -restore "$SET"
"$BIN" -backups                 # must include a pre-restore safety set
"$BIN" -probe
"$BIN"                         # original readings agree; added test league gone
```

Reopen once more after normal close: no torn state, shutdown hang or migration
error. Record file modes (`stat -c '%a %n' "$DATA" "$DATA"/*.db`), logs,
manifest, binary SHA-256, source commit, CI URL and Christopher's click/disposition
outside git if private. Expect directory 700 and DBs 600. The different-home
Visitor result remains a blocker, not a successful rights-preservation claim.
Measure five minutes idle and an unchanged hourly cycle on this same binary;
record CPU, wake/refresh work and command latency rather than asserting them.
Any candidate change repeats affected checks and this integrated gate.

## Plan corrections

The preferred toolchain was already 1.26.10; the unresolved gap was the module's
1.26.4 floor, not an absent toolchain selection or absent govulncheck gate.
Block B's different-home identity promise still contradicts the documented
Visitor behavior. Neither passing restore tests nor a patched compiler closes it.
