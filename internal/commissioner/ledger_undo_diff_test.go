package commissioner

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

// oldUndoPlacement and oldRolloverReversalRefusal are the reference: the Undo placement before
// 533f2aa7, which walked the reversal chain from the entry itself on every call.
func oldUndoPlacement(records []Record, i int, p Projection, principal string, auth Authorizer) string {
	r := records[i]
	if r.Command.Command == Create || r.Command.Command == NativeSeed || liveDraftEntry(r.Command.Command) ||
		!p.Active[r.Receipt.Entry] {
		return ""
	}
	if _, err := auth.Allow(r.Command.League, principal, Revert); err != nil {
		return ""
	}
	if oldRolloverReversalRefusal(records, r.Receipt.Entry) != "" {
		return ""
	}
	if governed, ok := auth.(GovernanceAuthorizer); ok &&
		authorizeReversal(governed, r.Receipt.Entry, principal, records[:i+1]) != nil {
		return ""
	}
	return r.Command.Visibility
}

func oldRolloverReversalRefusal(records []Record, entry string) string {
	target := slices.IndexFunc(records, func(r Record) bool { return r.Receipt.Entry == entry })
	for target >= 0 && records[target].Command.Command == Revert {
		next := records[target].Command.Args.Entry
		target = slices.IndexFunc(records[:target], func(r Record) bool { return r.Receipt.Entry == next })
	}
	if target < 0 {
		return ""
	}
	active := Replay(records).Active
	later := records[target+1:]
	if records[target].Command.Command == SeasonRollover {
		if slices.ContainsFunc(later, func(r Record) bool {
			return active[r.Receipt.Entry] && rolloverKind(r.Command.Command)
		}) {
			return "later"
		}
		return ""
	}
	if rolloverKind(records[target].Command.Command) && slices.ContainsFunc(later, func(r Record) bool {
		return active[r.Receipt.Entry] && r.Command.Command == SeasonRollover
	}) {
		return "moved past"
	}
	return ""
}

// allowAll answers like an authorizer without league governance: every principal may send everything.
type allowAll struct{}

func (allowAll) Allow(string, string, string) (string, error) { return "commissioner", nil }

// undoLedger extends the visibility ledger with what Undo placement asks about: season rollovers and the
// changes they capture, rules, trade overrides, lineups and draft picks, transaction corrections, and the
// league's creation and seed.
func undoLedger(rng *rand.Rand, n int) []Record {
	records := visibilityLedger(rng, n)
	principals := []string{"creator", "cc", "owner", "stranger"}
	extra := []RawCommand{
		{Command: SeasonRollover}, {Command: ContractEdit}, {Command: CapAdjust}, {Command: WindowSet},
		{Command: RulesSet, Args: RawArgs{Field: "calendar.boundary"}}, {Command: TradeOverride},
		{Command: LineupSet}, {Command: DraftPick}, {Command: Create}, {Command: NativeSeed},
		{Command: CorrectTransaction},
	}
	for range n / 2 {
		c := extra[rng.IntN(len(extra))]
		c.Visibility = []string{"league", "private"}[rng.IntN(2)]
		if c.Command == CorrectTransaction {
			if len(records) == 0 {
				continue
			}
			c.Args.Correction = &RawCorrection{Entry: records[rng.IntN(len(records))].Receipt.Entry}
		}
		at := rng.IntN(len(records) + 1)
		records = slices.Insert(records, at, Record{Principal: principals[rng.IntN(len(principals))], Command: c})
	}
	// Reversals of the new entries, reversals of those, and of a missing entry, all after their targets.
	for range n / 2 {
		if len(records) == 0 {
			break
		}
		target := records[rng.IntN(len(records))].Receipt.Entry
		if rng.IntN(6) == 0 {
			target = "missing"
		}
		records = append(records, Record{Principal: principals[rng.IntN(len(principals))],
			Command: RawCommand{Command: Revert, Visibility: "league", Args: RawArgs{Entry: target}}})
	}
	for i := range records {
		records[i].Command.League = "native.test"
		if records[i].Receipt.Entry == "" {
			records[i].Receipt.Entry = fmt.Sprintf("x%02d", i)
		}
	}
	return records
}

// The root index Reading passes since 533f2aa7 must place Undo exactly where the chain walk did, for every
// entry and principal, with and without league governance.
func TestUndoPlacementMatchesChainWalk(t *testing.T) {
	t.Parallel()
	state, err := governance.Replay([]governance.Event{
		governance.LeagueCreated{Creator: "creator", Franchises: []governance.FranchiseID{"f1", "f2"}},
		governance.SeatReplaced{Franchise: "f1", Principal: "owner"},
		governance.SeatReplaced{Franchise: "f2", Principal: "owner-a"},
		governance.RoleGranted{Principal: "cc", Role: governance.CoCommissioner},
	})
	if err != nil {
		t.Fatal(err)
	}
	auths := []Authorizer{allowAll{}, GovernanceAuthorizer{State: state}}
	rng := rand.New(rand.NewPCG(533, 0x0dd0)) //nolint:gosec // a fixed test grid, not a secret
	checked := map[string]int{}
	for trial := range 3000 {
		records := undoLedger(rng, 2+rng.IntN(14))
		p := Replay(records)
		roots := reversalRoots(records)
		hasRollover := slices.ContainsFunc(records, func(r Record) bool { return r.Command.Command == SeasonRollover })
		for i, r := range records {
			for _, auth := range auths {
				for _, principal := range []string{"creator", "cc", "owner", "owner-a", "stranger"} {
					want := oldUndoPlacement(records, i, p, principal, auth)
					got := undoPlacement(records, i, roots[r.Receipt.Entry], p, principal, auth, hasRollover)
					if got != want {
						t.Fatalf("trial %d: %s (%s) for %s with %T: new %q, old %q\nledger: %+v",
							trial, r.Receipt.Entry, r.Command.Command, principal, auth, got, want, records)
					}
					checked[fmt.Sprint(r.Command.Command, want != "")]++
				}
			}
		}
	}
	// The grid must reach both answers for reversals, rollovers and the changes a rollover captures.
	for _, key := range []string{Revert + "true", Revert + "false", SeasonRollover + "true", SeasonRollover + "false",
		ContractEdit + "true", ContractEdit + "false", BidPlace + "true", BidPlace + "false"} {
		if checked[key] == 0 {
			t.Errorf("grid never produced %s", key)
		}
	}
}
