# Native cap runway

`commissioner.NativeCapRunway` is the four-year HQ cap projection, also available to
War Room and Home through `TargetNativeCapCheck(...).franchises[].runway`.
Each row contains only season, committed cents, dead cents, cap cents and room cents.
Unknown amounts are null; negative room is preserved.

## Read path

Existing Commissioner API → leagueservice read admission → history.NativeCapCheck →
ReadNativeBase (validated, replayed native assets and current league rules) →
NativeCapRunway per franchise → nativecheck.CheckCap / CountedChange → existing IPC.
No writes, new endpoint, money rules, dependencies or changes to engine/store behavior.
Errors abort the read rather than returning a partial runway. Native authorization,
revision capture and unseeded-league behavior remain on the existing path.

Counted commitments include reserve shares and signed non-dead adjustments; dead
money is separated, not counted twice. Each amount is rounded by CheckCap, not by
this read. Cut entries are identified by their ledger command/entry identity, or
by an explicit dead_money kind; descriptions are never parsed. Expired contracts
drop out. Current flat salaries and squad shares project unchanged; standing
charges persist and dated charges count only in their recorded year. Cap rules are
those in force at the read: no guessed cap growth, renewals or future transactions.

## Evidence and limits

The previous IPC omitted runway; TestNativeCapRunwayIPC failed on that absence.
Table tests cover a cut of a multi-year contract using PriceContract, recorded dead
charges across years, expiration, SlotTerms/signRookie contracts, over-cap room,
franchise isolation, signed adjustments, reserve shares, per-entry rounding, missing
rulings, bad held contracts and unchanged inputs. The IPC tests cover actual native
cut/claim projection and physical reopen. Wire fixtures pin the additive response.
No UI/mock changes or live GUI test are claimed.

Suspected existing inconsistencies, deliberately not changed:
- Both native release pricing and transactions/deadcap charge all remaining-year
  penalties in the cut year. The mock's multi-year dead rows are not a new spread rule.
- Rollover carryCharges re-dates explicitly typed dead money into each next season;
  the dated-charge runway does not simulate that rollover mutation.
- Imported adjustments with no kind or native cut entry cannot be classified as dead
  from their text. They still count toward room as ordinary signed adjustments.
- The older current cap reading counts expired/pending roster contracts without an
  expiration filter; runway counts contracts only through expiration.

Debt: screen consumption and migration of other bid/trade/plan room callers are not
part of this read-only slice. Future pending rule changes are not forecast here.
