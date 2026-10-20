package commissioner

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"
)

// oldLedgerEntryVisible, oldPrivateWaiverRecord and oldPrivateBidRecord are the reference: the
// visibility check before 533f2aa7, which walked each record's own reversal chain.
func oldLedgerEntryVisible(r Record, records []Record, principal string, privileged bool) bool {
	if oldPrivateWaiverRecord(r, records) {
		return (r.Principal == principal && r.Command.Command == WaiverClaim) ||
			(privileged && processedWaiverClaim(r, records))
	}
	if oldPrivateBidRecord(r, records) {
		return r.Principal == principal || (privileged && decidedBidRecord(r, records))
	}
	return r.Command.Visibility != "private" || privileged
}

func oldPrivateWaiverRecord(r Record, records []Record) bool {
	if r.Command.Command == WaiverClaim {
		return true
	}
	for r.Command.Command == Revert {
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == r.Command.Args.Entry })
		if index < 0 {
			return false
		}
		r = records[index]
	}
	return r.Command.Command == WaiverClaim
}

func oldPrivateBidRecord(r Record, records []Record) bool {
	for range len(records) {
		if r.Command.Command != Revert {
			break
		}
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == r.Command.Args.Entry })
		if index < 0 {
			return false
		}
		r = records[index]
	}
	return OwnerBidAction(r.Command.Command) != ""
}

// visibilityLedger builds a well-formed ledger: hidden bids and raises, withdrawals, waiver claims,
// decisions (the reveal), public and private notes, and reversals of any earlier entry, reversals
// included, sent by owners or the commissioner. One reversal names an entry that is not recorded.
func visibilityLedger(rng *rand.Rand, n int) []Record {
	owners := []string{"owner-a", "owner-b", "commish"}
	var records []Record
	add := func(principal, command, visibility string, args RawArgs) {
		records = append(records, Record{Principal: principal, Receipt: Receipt{Entry: fmt.Sprintf("e%02d", len(records))},
			Command: RawCommand{Command: command, Visibility: visibility, Args: args}})
	}
	pick := func(command string) string {
		var ids []string
		for _, r := range records {
			if r.Command.Command == command {
				ids = append(ids, r.Receipt.Entry)
			}
		}
		if len(ids) == 0 {
			return ""
		}
		return ids[rng.IntN(len(ids))]
	}
	players, windows := []string{"p1", "p2"}, []string{"w1", "w2"}
	for range n {
		owner := owners[rng.IntN(2)]
		switch rng.IntN(9) {
		case 0:
			add(owner, BidPlace, "private", RawArgs{Bid: &RawBidArgs{Player: players[rng.IntN(2)]}})
		case 1:
			if bid := pick(BidPlace); bid != "" {
				add(owner, []string{BidRaise, BidWithdraw}[rng.IntN(2)], "private", RawArgs{Bid: &RawBidArgs{Bid: bid}})
			}
		case 2:
			add(owner, WaiverClaim, "private", RawArgs{Waiver: &RawWaiverArgs{Window: windows[rng.IntN(2)]}})
		case 3:
			add("commish", WaiverProcess, "league", RawArgs{Waiver: &RawWaiverArgs{Window: windows[rng.IntN(2)]}})
		case 4:
			add("commish", BidsProcess, "league", RawArgs{Bid: &RawBidArgs{Player: players[rng.IntN(2)]}})
		case 5:
			add(owners[rng.IntN(3)], "commish.note.add", []string{"league", "private"}[rng.IntN(2)], RawArgs{})
		case 6:
			add(owners[rng.IntN(3)], Revert, []string{"league", "private"}[rng.IntN(2)], RawArgs{Entry: "missing"})
		default:
			if len(records) > 0 {
				target := records[rng.IntN(len(records))].Receipt.Entry
				add(owners[rng.IntN(3)], Revert, []string{"league", "private"}[rng.IntN(2)], RawArgs{Entry: target})
			}
		}
	}
	return records
}

// The root lookup Reading uses since 533f2aa7 must hide and show exactly what the chain walk did,
// for every entry and principal, before and after the reveal.
func TestLedgerVisibilityMatchesChainWalk(t *testing.T) {
	t.Parallel()
	rng := rand.New(rand.NewPCG(533, 0xf2aa7)) //nolint:gosec // a fixed test grid, not a secret
	checked := map[string]int{}
	for trial := range 4000 {
		records := visibilityLedger(rng, 2+rng.IntN(14))
		roots := reversalRoots(records)
		for _, r := range records {
			for _, principal := range []string{"owner-a", "owner-b", "commish"} {
				for _, privileged := range []bool{false, true} {
					want := oldLedgerEntryVisible(r, records, principal, privileged)
					got := ledgerEntryVisible(r, records[roots[r.Receipt.Entry]], records, principal, privileged)
					if got != want {
						t.Fatalf("trial %d: %s (%s by %s) for %s privileged=%v: new %v, old %v\nledger: %+v",
							trial, r.Receipt.Entry, r.Command.Command, r.Principal, principal, privileged, got, want, records)
					}
					checked[fmt.Sprint(r.Command.Command, want)]++
				}
			}
		}
	}
	// The grid must reach both answers for hidden bids, claims and their reversals.
	for _, key := range []string{BidPlace + "true", BidPlace + "false", WaiverClaim + "true", WaiverClaim + "false",
		Revert + "true", Revert + "false", BidRaise + "true", BidRaise + "false"} {
		if checked[key] == 0 {
			t.Errorf("grid never produced %s", key)
		}
	}
}
