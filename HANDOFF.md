# Handoff

## Baton
Review — 2026-10-28, `session/revamp-cleanup`. No push requested or attempted.

## Where it stands
- C1: Archive removal `0ba00108`; orphan Sources views and authoritative endpoint retirement
  `6e20d3bc`; registry absence assertions `9539f060`. Counterparties/offer-history/Lab APIs were
  already absent at the starting revision. Live shared services stay; see
  `docs/architecture/research-cleanup.md` for callers and retained data.
- C2(a): `7887f4be` writes player records at listing/signals publication, not valuation.
- C2(b): `3d4a4cd6` retains supplied starting-season and All-Pro/Pro Bowl counts, with provenance.
  Migration 12 adds career facts without deleting records; server release 5 names the schema.
  No registered held feed supplies game starts or player wins; these are not fabricated.
- C2(c), partial: `420b9a71`, `8134d167` extend Pick Value v1 from held post-2026 MFL drafts,
  preserving each draft's own slot width. Native-ledger rookie drafts are not connected.
- C3: `49d56975` gives each counted side its own O-Line lots, owner bid readings and close
  reservations; old unscoped seed lots expand on detached reads. Test fixture correction
  `82cd9b75`. One-pool/two-pool close, cap prompt and retry tests pass.
- Approved held 10-28 value-board replay: 2,278 players, zero movers; all Score fields equal.
  Affected handwritten production: 6,921 → 5,450 physical lines versus `e3bc9d41`;
  tests/docs/generated output excluded.

## Evidence and limits
- Full `make verify` ran once, Go 1.26.10, isolated lint cache. Lint, boundary, vulnerability,
  bindings freshness and typecheck passed. Go: 119 packages passed, two failed; composition
  passed its full race run. Verify did not reach its frontend stage and is NOT claimed green.
- The two failures: incomplete new O-Line test fixture (fixed; market's 26 top-level race tests
  pass), and a Unix socket exceeding its path limit under this worktree's long temp path.
  The socket test passes with a directly run race test binary and relative in-worktree GOTMPDIR.
- Final touched-package evidence: C2(c) 32 top-level race PASS; final composition slice 19 PASS;
  C1 retirement 1 PASS / eight cases. Earlier record/source batch 53 PASS; career/feed/version
  batch 24 PASS; migration slice nine PASS. Logs are in `.tmp/` and are not committed.
- Frontend ran once at the end: 1,575 Vitest PASS / 132 files; build, bundle budget and CSP checks
  pass. Final Go build passes. No GUI acceptance or per-side worker/restart click-through.

## Next move
Review the commits, connect native-ledger draft slots if required, then live-test both O-Line
sides and the cut/retry flow on a disposable league. A passing cut frees only its franchise's
room and issues its reserved one-year unit exactly once; the other side's copy is unchanged.

## Blocked on
No technical blocker to review. Full verify has no final all-green run; it was not rerun after
fixes, per the brief's once-only requirement. Native draft slot input and GUI acceptance remain.

## Tried and rejected
- Lazy record writes during valuation: removed, so the runner cannot invent its own record.
- Removing Sources/Offers packages wholesale: live ingestion, health, settings, pool/bid and
  personal-list/refresh commands still call them. No stored history or schema was destroyed.
- Treating starting seasons as game starts, or weighted approximate value as wins: not the feed's
  meaning. Missing facts stay unknown and do not change the approved valuation inputs.
