# Pick Value v2 and franchise power

The 2026-10-28 ruling supersedes the earlier display-only pick decision: the franchise power
board counts picks at `power.pick_weight` (default 0.6, range 0–1). The season board counts none.
Both views default to the legal starting lineup. The approved player valuation model is unchanged.

## Data flow

- `app/market.LoadNative` reads a named run's dynasty PPG and player records. MFL history extends
  the embedded 2017–2026 sample with checked held drafts. A native-scoped source instead reads
  only that league's replayed rookie drafts through `history.PickValueDrafts`; startup drafts,
  extra selections, skips, forfeits, unknown capital and future selections do not train it.
  Corrections/restores come from the existing commissioner replay, not a second ledger fold.
- `pickvalue.Fit` is pure. The band's denominator is every integer NFL draft slot in the band
  across four recent classes, including missing records as zero. A running minimum makes the
  fitted 512-selection curve non-increasing; later selections retain its floor. This does not
  introduce an outside market curve or change any player value.
- `Native.Pick` uses a recorded slot keyed by year, round and **original** franchise, independent
  of who now owns it; otherwise it uses the round mean. `pickvalue.discount` (default 0.75,
  range 0–1) discounts each year out, independently of `dynasty.discount`.
- `market.PickTotals` sums the next two drafts by current owner. Nil inventory/model means unknown;
  a known empty inventory means zero. An unreadable pick blanks that owner's total.
- `composition.powerPicks` fits against each compared model run, using current inventory/settings.
  `scoring.Source` supplies picks only to the franchise view. `m2service.BuildBoard` adds
  weight × pick total **after** counting the lineup and **before** standardization. Picks never
  enter age. `season.View.Power` shows unweighted Picks beside roster value (which includes the
  weighted addition) and names the weight and method. Missing/non-finite totals never fail a board.
- Control Room › League exposes both settings through the existing `data.parameter` command and
  validated params store, including reset. Successful writes rebuild held franchise power and
  republish the pick model when needed, without fetching MFL or rerunning player valuations.

## Exercised and limits

Tests cover the whole monotone curve, missing records, discount independence, known/unknown slots,
owner/horizon totals, failure preservation, lineup/age behavior, settings persistence/range/reset,
report wording, commissioner replay and database restart. Native orders are read even before a
start date is set. Replay evidence compares all 2,278 approved run #5 player Score fields exactly.

The current asset/power composition is still MFL-scoped. The native-scoped loader/store path is
exercised end to end, but selecting a native league on the existing commissioner screen does not
switch the MFL asset/power model. Native league model selection remains integration debt; histories
are deliberately not pooled across unrelated leagues. MFL's held future-pick inventory has no
explicit original-franchise slot order, so its 2027–2028 picks correctly remain round means.
No GUI acceptance is claimed.
