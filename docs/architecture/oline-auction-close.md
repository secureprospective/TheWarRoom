# Closing the O-Line start auction

`oline.auction.close` ("Close O-Line auction") is a commissioner-only registry command.
It uses the league's O-Line roster setting, not its creation preset. There is no deadline setting.
An O-Line-off league refuses it without changing assets. Settle standing bids and ties first;
close never gives a contested unit away or rewrites an auction winner's contract.

Leftover NFL units are ordered alphabetically by their team abbreviation; unit-less franchises
are ordered by their stable franchise ID. Pair those lists, preserving existing owners. This is
neutral and reproducible without inventing rankings or consuming random state. Each assignment
has one year at the lowest known minimum bidding salary when the auction closes.

## Ownership and failure routes

| Path | Contract and effects |
| --- | --- |
| Registry → service → history append | Raw arguments admit no caller-supplied decision. Current commissioner authority and revision are checked inside the write transaction. |
| `PrepareOLineClose` | Reads current assets, bid book and settings; reserves deterministic pairs. Insufficient units or unsettled bids refuse the entire close. |
| Cap/roster room check → outcome | Counts reserve shares, rounding and current-season charges with the existing cap calculator. Never assigns above the cap, even under soft enforcement or an exception. The startup roster minimum is not a prerequisite; the roster maximum remains in force. |
| Outcome → native projector and ledger | Persists assignments and pending reservations atomically. Replay uses recorded contracts, not today's settings. Ledger descriptions name both unit and franchise in plain words. Close is permanent; individual assets remain correctable. |
| Pending → jobs | A public team-specific cut prompt remains until assignment completes. There is no automatic cut or cut deadline. |
| Owner cut → due worker | A pending, unit-less franchise can reduce its roster without being refused for the missing unit. Its ordinary cut calendar and remaining cap checks still apply. |
| `ProcessDueOLines` → append | The restricted clock identity cannot open an auction. It rechecks a closed reservation and available room under the append lock. A blocked pass writes nothing. Startup, post-commit wakes and periodic reconciliation retry after room opens, including after restart. O-Line off suspends retries. |

The close also stops further bids on O-Line units. Other market actions keep their existing rules.

## Evidence

`TestOLineAuctionCloseFillsEveryEmptyRoster` was observed failing before implementation because
franchise.01 held zero units after close. Tests cover all 32 rosters, a retained winner, stable
ordering with reversed input, ledger wording, foreign arguments and decisions, outstanding bids,
O-Line off, unauthorized close, full-cap blocking, the requested cut, restart completion,
idempotency and a worker that cannot open the auction. The worker test uses Legacy settings
with O-Line enabled by an ordinary rule change, not the Default32 preset.

No GUI click-through is claimed. The existing jobs surface carries the cut prompt; this slice
adds the close command to the registry/CLI, not a new dedicated frontend button.

The final focused batch has 10 passing top-level tests under `-race`, including a real timed bid
award retained by close and the owner's public ledger reading. Generated catalogs and TypeScript
checks pass. Handwritten production Go across all 18 affected files grows from 4,497 to 4,873
physical lines (161,603 → 175,592 code points including LF; 161,613 → 175,602 UTF-8 bytes).
This is a behavior addition, not a debloat claim; tests, docs and generated files are excluded.

## Player pools per side

Native start reads `players.pools` and the counted conference sides, not a preset name. An OL-bounded
league receives all 32 NFL unit lots in each pool (blank side only for a one-pool league). Admission
requires each unit exactly once per side. Older unscoped O-Line seed lots expand on the detached read
without rewriting the ledger; existing single-pool bytes remain unchanged.

Owner bid readings show only that owner's side's auctions and unbid units; awarded/closed units are
not offered as fresh lots. The league-wide close reserves and checks each franchise's unit on its
own side. Its cap check, one-year minimum contract, cut prompt and retry use that franchise's copy;
one side holding or reserving a unit does not consume the other's. The close command itself remains
league-wide, so standing bids anywhere must be settled first. No new durable outcome shape is needed.

Table tests: `TestOLinePoolsAndCloseFollowPoolSetting`, `TestOLineCapReservationsAndRetryArePerSide`,
`TestStartingOLineLotsAreScopedAndCloseHidesThem`; old-seed detachment and single-pool digest tests
cover compatibility. Tests exercise a recorded roster correction making room, not a GUI cut workflow.
