# Review — M4 slice-3 pre-commit dollar breakdown

## LEAD 1 — CapRelief delta likely carries the RAW request amount, not the snapped one (HIGH)

**Failure scenario:** Commissioner requests a §13 relief of $3,006,000 (off the $10k grid). The store snaps to $3,010,000 on the ledger. But the pre-commit breakdown shows `−$3,006,000` (raw), so the quote the commissioner confirms does not match the figure that actually posts — the exact drift the design claims to prevent.

**Where the break is.** `deadcap.Relieve` constructs the entry with the raw parameter and returns it without snapping:

```go
// internal/transactions/deadcap/special.go:127-148
entry := state.CapReliefEntry{
    ...
    Amount: amount,   // raw request, no snapping in the body
    Reason: reason,
}
if err := w.AddCapRelief(ctx, entry); err != nil { ... }
return entry, nil    // entry.Amount is still `amount`
```

Then `CapRelief.apply` builds the delta from `entry.Amount`:

```go
// internal/transactions/request.go:393
Deltas: []CapDelta{{FranchiseID: c.FranchiseID, Cents: -entry.Amount, Reason: "cap relief §13"}}
```

Two structural reasons this is wrong even granting that *some* snapping exists in the store:

1. `w.AddCapRelief(ctx, entry)` takes `entry` **by value**. Any snapping the store does internally cannot mutate the caller's `entry`. The return value of `AddCapRelief` (if it even produces a snapped entry) is discarded — only `err` is captured.
2. The added test `TestPreview_CapReliefBreakdown` feeds `Amount: 3_006_000 * 100` and asserts `got.Cents != -3_010_000*100`. For that assertion to pass, `entry.Amount` must equal `3_010_000*100` at the call site in `apply`. Nothing in the visible diff produces that transformation. `domain.Money` is assigned via struct literal (`Amount: amount`), so no constructor/normalization runs on the assignment.

**Confidence:** HIGH that the visible diff does not snap; HIGH that this is a real bug (the comment "Use the store's SNAPPED amount (entry.Amount)" is false to the code as shown). Either the test fails on a real store, or the test was never run, or there is invisible snapping — but the code as written drifts from the ledger.

**Fix:** snap in `Relieve` before constructing the entry (e.g. `amount = amount.snapTo(10_000*100)`) so both the persisted row and the returned entry carry the snapped figure; or have `AddCapRelief` return the snapped entry and use that.

---

## LEAD 2 — CapRelief delta hardcodes the reason, inconsistent with Waive/Buyout/Retire (MEDIUM)

**Failure scenario:** Commissioner submits a relief with `Reason: "career-ending injury"`. The breakdown row shows `cap relief §13`, while the committed `cap_relief` ledger row carries `career-ending injury` (because `entry.Reason = reason` is what gets persisted). The quote and the ledger describe the same charge with different labels — the very thing `breakdown.go`'s `CapDelta.Reason` doc claims must never happen ("reused verbatim so the quote and the committed ledger never describe the same charge differently").

```go
// internal/transactions/request.go:393
Deltas: []CapDelta{{..., Reason: "cap relief §13"}}   // hardcoded, ignores entry.Reason
```

vs. the dead-cap path which uses the entry's own reason:

```go
// internal/transactions/breakdown.go:38
return []CapDelta{{..., Reason: e.Reason}}
```

The test only asserts `got.Reason == ""`, so it does not catch this.

**Confidence:** MEDIUM. If the store's true audit label for a relief row is literally `"cap relief §13"` (and `entry.Reason` is metadata that never reaches the ledger), then the hardcode is correct and consistent. If `entry.Reason` is what the store stamps, this is a real mismatch. The `state.CapReliefEntry.Reason` field existing at all leans toward the second reading — flag to confirm against `AddCapRelief`.

---

## LEAD 3 — `receiptResult` became a method; out-of-diff callers would break the build (LOW)

`receiptResult` changed from a package-level function to `(a *App).receiptResult`. The diff updates the call sites inside `PreviewTransaction`. Any other caller (another `PreviewXxx`, a test, or — most importantly — `ExecuteTransaction`, which the prompt claims "intentionally omits" CapDeltas) would now fail to compile if it still calls the free function.

Likewise `Pending` gained a required `capDeltas` field; the diff updates three `.tsx` stage paths, but any other `Pending` literal in the repo (another modal entry point, a test fixture) would now be a TS error.

**Confidence:** LOW that a bug ships (the compiler catches both), but worth confirming `make lint && make test && pnpm tsc --noEmit` were actually run — a stale `ExecuteTransaction` body that still calls `receiptResult` is the most likely shadow here.

---

## Things I checked and found correct

- **Rollback survival.** `coordinator.go` Preview captures `res = r` *before* `return errDryRun`. The `applyResult` is a return value, never a post-apply re-read, so the in-tx read wall is respected and the deltas survive the rollback. Correct.
- **Sign convention.** Dead-cap charges flow through `deadCapDeltas` as positive `e.DeadCap`; relief is explicitly negated `-entry.Amount`. Matches `CapUsed = Σcells + Σdead_cap − Σcap_relief`. Correct.
- **Death → no line.** `deadCapDeltas` returns `nil` for `e.DeadCap <= 0`, so the $0 Gaines-Adams case emits no row rather than a noisy `$0`. Correct.
- **PlayersAffected.** Every handler preserves its prior count (Trade `len(moves)`, the 1-player ops, the 0-player ops CapRelief/AdvancePhase/RolloverSeason/SetSigningWindow). No drift.
- **Receipt non-comparability.** Adding `[]CapDelta` made `Receipt` non-comparable; the `nonZeroReceipt` helper and the inlined equivalent in `TestExecute_BuyoutCounterFailRollsBack` correctly replace the old `rec != (Receipt{})` checks. Correct.
- **Frontend nil guards.** All three stage paths initialize `capDeltas: []` and fold in `res.capDeltas ?? []` on success. The modal renders deltas only when `pending.previewOK === true && (pending.capDeltas ?? []).length > 0`. The double-guard is belt-and-suspenders, not load-bearing — fine.
- **Color/sign mapping.** `d.cents < 0 ? green : red` matches the spec (credit = negative = good/green; charge = positive = bad/red). The `+`/`−` formatting is server-side in `capDeltaDTOs` (`+` prepended for positive, `Money.String` already prefixes `−` for negative), so the UI doesn't re-derive sign. Correct.
- **Single-writer law.** No new mutators, no new goroutines, no store-to-store import. All `apply` still runs inside the one `WriteTx`. No violation introduced.
- **Execute Receipt carrying CapDeltas.** Harmless — Execute's IPC mapping doesn't go through `receiptResult` per the prompt, so the deltas are simply unused on the commit path (post-commit refresh is authoritative). No bug from carrying them, assuming the ExecuteTransaction body genuinely doesn't call `a.receiptResult`.

---

**Bottom line:** LEAD 1 is the one to block on — either the test fails, or the dollar quote drifts from the ledger in the exact way the feature exists to prevent. LEAD 2 is a label-consistency question to resolve against the store schema. LEAD 3 is "confirm the build is green."
EXIT=0
