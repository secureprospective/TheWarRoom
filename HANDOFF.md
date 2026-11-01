# Handoff

## Baton
Review — 2026-11-01, build/WR-wire. Local commits only; no remote contact.

## Where it stands
- PICKER-6: d903e283 keeps native HQ/War Room inside the normal shell; 01178962 adds the per-league footer franchise chooser; a4f34846 fixes control/render gates and removes seat-based permission inference.
- Flow: selected league → checked people/seats → saved valid franchise (otherwise held seat) → native HQ/War Room reads via localWorkflow. The workflow authorizer, not the viewed franchise, determines acting rights. No import-source branches added.
- No-seat render tests exercise the real shell and native screens with a fake port: chooser, roster/contracts, War Room, navigation, held-seat default, invalid saved choice and league-isolated persistence.
- Final make build (dev/picker6), frontend typecheck, 1704 frontend tests / 154 files, bundle budget (149081 / 321818 bytes), CSP and diff checks pass. Mechanical UI detector returned no findings before the final gate corrections.
- One full Go run: `go test -race -timeout 40m ./...`; 130 packages pass, 1 fails, 20 have no tests. Failure: internal/leaguetransfer TestEncryptedExportRoundtripFrozenReplayParity, transfer_test.go:211, “roster differs”. No Go source changed. Logs: frontend/.tmp/picker6/.
- heavy-ok passed three standalone checks. One combined invocation reported STOP (one suite running) and mistakenly continued into the suite; its shell command itself contained the matched suite text. No second full suite was launched.
- Handwritten production footprint, e0be8be4 → a4f34846, 12 affected files: 2905 → 2931 physical lines; 183597 → 185466 Unicode code points including LF. Tests/docs excluded.

## Next move / blocked on
- Reviewer performs the requested desktop walk: app-league HQ/War Room must retain rail/picker/navigation/footer, franchise choice must display native roster/contracts and survive switching/relaunch, and refused actions must remain read-only.
- Investigate the Go transfer parity failure before declaring the full gate green. No desktop walk was performed here.
- Prior unrelated debt remains: partner-position enrichment needs a separate read-only wire field; overwriting recorded snapshot names broke composition/golden-wire tests.

## Tried and rejected
- Empty seat is not a permission rule: removed the local read-only wrapper; existing backend authorization stays authoritative.
- Existing untracked .golangci-cache/, .picker-state.md and .picker-tmp/ left untouched.
