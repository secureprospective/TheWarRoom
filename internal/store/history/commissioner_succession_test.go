package history

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type successionFixture struct {
	store    *Store
	clock    *clock
	revision int64
}

func newSuccession(t *testing.T, seats int, cc bool) *successionFixture {
	t.Helper()
	s, c, _ := newStore(t)
	f := &successionFixture{store: s, clock: c}
	raw := createCommand()
	raw.Args.FranchiseCount = seats
	if _, err := s.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	f.revision = 1
	for i := range seats {
		f.people(t, "commish.seat.replace", commissioner.RawPeople{Franchise: fmt.Sprintf("franchise.%02d", i+1),
			Principal: fmt.Sprintf("owner%d", i)})
	}
	if cc {
		f.people(t, "commish.role.grant", commissioner.RawPeople{Principal: "cc", Role: governance.CoCommissioner})
	}
	f.people(t, "commish.role.grant", commissioner.RawPeople{Principal: "judge", Role: governance.DOT})
	return f
}

func (f *successionFixture) people(t *testing.T, command string, args commissioner.RawPeople) {
	t.Helper()
	executePeople(t, f.store, "creator", command, args, f.revision)
	f.revision++
}

func successionCommand(command string, args commissioner.RawPeople, revision int64) commissioner.RawCommand {
	raw := peopleCommand(command, args, revision)
	if command == "commish.vote.cast" {
		raw.Visibility = "private"
	}
	return raw
}

func (f *successionFixture) act(t *testing.T, principal, command string,
	args commissioner.RawPeople) commissioner.Receipt {
	t.Helper()
	raw := successionCommand(command, args, f.revision)
	receipt, err := f.store.ExecuteCommissioner(t.Context(), raw, principal, nil)
	if err != nil {
		t.Fatalf("%s by %s: %v", command, principal, err)
	}
	f.revision++
	return receipt
}

func (f *successionFixture) reading(t *testing.T, principal string) commissioner.People {
	t.Helper()
	out, err := f.store.CommissionerPeople(t.Context(), "native.test", principal)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (f *successionFixture) state(t *testing.T) governance.State {
	t.Helper()
	events, err := f.store.Events(t.Context(), "native.test", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	records, err := commissionerRecords(events)
	if err != nil {
		t.Fatal(err)
	}
	state, err := commissioner.GovernanceState(records)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func (f *successionFixture) restart(t *testing.T, principal string) {
	t.Helper()
	before := f.reading(t, principal)
	ledger, err := f.store.CommissionerLedger(t.Context(), "native.test", principal, nil)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	if err := f.store.pools.Write().QueryRowContext(t.Context(),
		`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err := f.store.pools.Close(); err != nil {
		t.Fatal(err)
	}
	pools, err := db.Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	f.store.pools = pools
	if after := f.reading(t, principal); !reflect.DeepEqual(before, after) {
		t.Fatalf("reopen differs: %+v %+v", before, after)
	}
	after, err := f.store.CommissionerLedger(t.Context(), "native.test", principal, nil)
	if err != nil || !reflect.DeepEqual(ledger, after) {
		t.Fatalf("ledger reopen differs: %+v %v", after, err)
	}
}

func TestLedgerSuccessionAcceptDecline(t *testing.T) {
	t.Parallel()
	for _, accept := range []bool{true, false} {
		t.Run(fmt.Sprint(accept), func(t *testing.T) {
			f := newSuccession(t, 2, true)
			f.act(t, "creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "creator"})
			if f.reading(t, "cc").Succession != "The co-commissioner has taken over; acceptance is pending" {
				t.Fatal("pending acceptance not shown")
			}
			command := "commish.succession.decline"
			if accept {
				command = "commish.succession.accept"
			}
			f.act(t, "cc", command, commissioner.RawPeople{Principal: "cc"})
			state := f.state(t)
			if state.HasRole("creator", governance.Commissioner) || state.HasRole("cc", governance.Commissioner) != accept {
				t.Fatal("incorrect commissioner succession")
			}
			if !accept && f.reading(t, "owner0").Succession != "A commissioner election is needed" {
				t.Fatal("decline did not expose election needed")
			}
			f.restart(t, "owner0")
		})
	}
}

func (f *successionFixture) ballot(t *testing.T, index int, choice string) commissioner.Receipt {
	t.Helper()
	p := fmt.Sprintf("owner%d", index)
	return f.act(t, p, "commish.vote.cast", commissioner.RawPeople{Principal: p, ID: "election", Choice: choice})
}

func TestLedgerElectionMajorityRunoff(t *testing.T) {
	t.Parallel()
	for _, majority := range []bool{true, false} {
		t.Run(fmt.Sprint(majority), func(t *testing.T) {
			f := newSuccession(t, 32, true)
			f.act(t, "creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "creator"})
			f.act(t, "cc", "commish.succession.decline", commissioner.RawPeople{Principal: "cc"})
			f.act(t, "cc", "commish.succession.election.open", commissioner.RawPeople{Principal: "cc", ID: "election"})
			first := 16
			if majority {
				first = 17
			}
			for i := range first {
				f.ballot(t, i, "owner0")
			}
			for i := first; i < 24; i++ {
				f.ballot(t, i, "owner1")
			}
			for i := 24; i < 32; i++ {
				f.ballot(t, i, "owner2")
			}
			read := f.reading(t, "owner0")
			if len(read.Votes) != 1 || read.Votes[0].BallotsCast != 32 || read.Votes[0].VotesRequired != 17 ||
				len(read.Votes[0].VotedFranchises) != 0 {
				t.Fatalf("public election counts: %+v", read.Votes)
			}
			f.restart(t, "cc")
			f.clock.now = f.clock.now.Add(7 * 24 * time.Hour)
			f.act(t, "cc", "commish.vote.close", commissioner.RawPeople{ID: "election"})
			v, _ := f.state(t).Vote("election")
			if majority {
				if !v.Closed || v.Winner != "owner0" || !f.state(t).HasRole("owner0", governance.Commissioner) {
					t.Fatalf("17 did not win: %+v", v)
				}
			} else {
				if !v.Runoff || v.Closed || !reflect.DeepEqual(v.Candidates, []governance.Principal{"owner0", "owner1", "owner2"}) {
					t.Fatalf("tied cutoff not retained: %+v", v)
				}
				f.ballot(t, 0, "owner0")
				f.ballot(t, 1, "owner1")
				f.clock.now = f.clock.now.Add(7 * 24 * time.Hour)
				raw := successionCommand("commish.vote.close", commissioner.RawPeople{ID: "election"}, f.revision)
				if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil ||
					!strings.Contains(err.Error(), "Needs a ruling") {
					t.Fatalf("unruled tie: %v", err)
				}
				f.act(t, "cc", "commish.vote.tiebreak", commissioner.RawPeople{ID: "election", Principal: "cc", Choice: "owner1"})
				public, err := f.store.CommissionerLedger(t.Context(), "native.test", "owner5", nil)
				if err != nil || !slices.ContainsFunc(public.Entries, func(e commissioner.Entry) bool {
					return strings.Contains(e.What, "tied runoff")
				}) {
					t.Fatalf("the tie ruling must be visible to owners: %v", err)
				}
				f.act(t, "cc", "commish.vote.close", commissioner.RawPeople{ID: "election"})
				v, _ = f.state(t).Vote("election")
				if !v.Closed || v.Winner != "owner1" {
					t.Fatalf("CC tie ruling not counted: %+v", v)
				}
			}
			f.restart(t, "owner0")
		})
	}
}

func TestLedgerVoteOutThreshold(t *testing.T) {
	t.Parallel()
	for _, yes := range []int{21, 22} {
		t.Run(fmt.Sprint(yes), func(t *testing.T) {
			f := newSuccession(t, 32, true)
			f.act(t, "judge", "commish.succession.voteout.open", commissioner.RawPeople{
				ID: "removal", Principal: "judge", Commissioner: "creator"})
			for i := range yes {
				p := fmt.Sprintf("owner%d", i)
				f.act(t, p, "commish.vote.cast", commissioner.RawPeople{ID: "removal", Principal: p, Choice: "yes"})
			}
			public := f.reading(t, "owner0")
			private := f.reading(t, "cc")
			if public.Votes[0].VotesRequired != 22 || public.Votes[0].BallotsCast != yes ||
				len(public.Votes[0].VotedFranchises) != 0 || len(private.Votes[0].VotedFranchises) != yes {
				t.Fatalf("participation privacy/counts: %+v %+v", public.Votes, private.Votes)
			}
			ledger, err := f.store.CommissionerLedger(t.Context(), "native.test", "owner0", nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range ledger.Entries {
				if strings.Contains(entry.What, "ballot") {
					t.Fatal("public ballot identity leak")
				}
			}
			closed := f.act(t, "cc", "commish.vote.close", commissioner.RawPeople{ID: "removal"})
			if f.state(t).HasRole("creator", governance.Commissioner) != (yes == 21) {
				t.Fatal("wrong removal threshold")
			}
			f.restart(t, "cc")
			raw := reverseCommand(closed.Entry, f.revision, "undo.close")
			if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "cc", nil); err != nil {
				t.Fatal(err)
			}
			f.revision++
			v, _ := f.state(t).Vote("removal")
			if v.Closed || !f.state(t).HasRole("creator", governance.Commissioner) {
				t.Fatal("close compensation failed")
			}
		})
	}
}

func TestLedgerSuccessionRefusalsAndBallotRevert(t *testing.T) {
	t.Parallel()
	f := newSuccession(t, 2, true)
	f.act(t, "cc", "commish.succession.voteout.open", commissioner.RawPeople{
		Principal: "cc", Commissioner: "creator", ID: "removal"})
	cases := []struct {
		principal, command string
		args               commissioner.RawPeople
		honesty            bool
	}{
		{"owner0", "commish.vote.cast", commissioner.RawPeople{ID: "removal", Principal: "owner1", Choice: "yes"}, true},
		{"creator", "commish.succession.stepdown", commissioner.RawPeople{Principal: "cc"}, true},
		{"creator", "commish.succession.accept", commissioner.RawPeople{Principal: "cc"}, true},
		{"creator", "commish.succession.takeover", commissioner.RawPeople{Principal: "cc", Commissioner: "creator"}, true},
		{"owner0", "commish.succession.periods.set", commissioner.RawPeople{ElectionDays: 2, RunoffDays: 3}, false},
		{"owner0", "commish.vote.close", commissioner.RawPeople{ID: "removal"}, false},
		{"owner0", "commish.succession.election.open", commissioner.RawPeople{ID: "election", Principal: "owner0"}, false},
	}
	for _, c := range cases {
		raw := successionCommand(c.command, c.args, f.revision)
		_, err := f.store.ExecuteCommissioner(t.Context(), raw, c.principal, nil)
		if err == nil {
			t.Fatalf("accepted %s", c.command)
		}
		if c.honesty && err.Error() != "league change refused: Other owners act from their own app; this desktop can only act as you" {
			t.Fatalf("wrong honesty refusal: %v", err)
		}
		if f.reading(t, "cc").Revision != f.revision {
			t.Fatal("refusal appended")
		}
	}
	receipt := f.act(t, "owner0", "commish.vote.cast", commissioner.RawPeople{ID: "removal", Principal: "owner0", Choice: "yes"})
	raw := reverseCommand(receipt.Entry, f.revision, "undo.ballot")
	raw.Visibility = "private"
	if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "owner0", nil); err == nil {
		t.Fatal("owner reversed without correction permission")
	}
	if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "cc", nil); err != nil {
		t.Fatal(err)
	}
	f.revision++
	v, _ := f.state(t).Vote("removal")
	if len(v.Ballots) != 0 {
		t.Fatal("ballot compensation did not remove vote")
	}
	f.restart(t, "cc")
}

func TestLedgerInactivityAndPeriods(t *testing.T) {
	t.Parallel()
	f := newSuccession(t, 2, true)
	f.act(t, "creator", "commish.succession.inactivity.set", commissioner.RawPeople{InSeasonDays: 3, OffseasonDays: 2})
	f.act(t, "creator", "commish.succession.activity", commissioner.RawPeople{Principal: "creator"})
	// Past the 2-day offseason period but short of the 3-day in-season one: the longer applies.
	f.clock.now = f.clock.now.Add(3*24*time.Hour - time.Second)
	raw := successionCommand("commish.succession.takeover", commissioner.RawPeople{
		Principal: "cc", Commissioner: "creator"}, f.revision)
	if _, err := f.store.ExecuteCommissioner(t.Context(), raw, "cc", nil); err == nil ||
		!strings.Contains(err.Error(), "longer inactivity period") {
		t.Fatalf("takeover before the longer period: %v", err)
	}
	f.clock.now = f.clock.now.Add(time.Second)
	f.act(t, "cc", raw.Command, *raw.Args.People)
	if !f.state(t).HasRole("cc", governance.Commissioner) {
		t.Fatal("recorded inactivity ignored")
	}
	f.act(t, "cc", "commish.succession.periods.set", commissioner.RawPeople{ElectionDays: 2, RunoffDays: 3})
	f.act(t, "cc", "commish.succession.stepdown", commissioner.RawPeople{Principal: "cc"})
	f.act(t, "judge", "commish.succession.election.open", commissioner.RawPeople{ID: "election", Principal: "judge"})
	v, _ := f.state(t).Vote("election")
	if v.EndsAt.Sub(v.OpenedAt) != 2*24*time.Hour {
		t.Fatal("election setting ignored")
	}
	f.ballot(t, 0, "owner0")
	f.ballot(t, 1, "owner1")
	f.restart(t, "owner0")
}

func TestLedgerSuccessionGuards(t *testing.T) {
	t.Parallel()
	f := newSuccession(t, 2, true)
	refused := func(principal, command string, args commissioner.RawPeople, want string) {
		t.Helper()
		raw := successionCommand(command, args, f.revision)
		_, err := f.store.ExecuteCommissioner(t.Context(), raw, principal, nil)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s by %s: want %q, got %v", command, principal, want, err)
		}
		if f.reading(t, "cc").Revision != f.revision {
			t.Fatal("refusal appended")
		}
	}
	// The co-commissioner may not set the period that lets them take over.
	refused("cc", "commish.succession.inactivity.set", commissioner.RawPeople{InSeasonDays: 1, OffseasonDays: 1},
		"only the commissioner")
	refused("owner0", "commish.succession.activity", commissioner.RawPeople{Principal: "owner0"},
		"only the commissioner")
	f.act(t, "creator", "commish.succession.inactivity.set", commissioner.RawPeople{InSeasonDays: 1, OffseasonDays: 1})

	// Any league change by the commissioner counts as activity, without a separate check-in.
	f.clock.now = f.clock.now.Add(2 * 24 * time.Hour)
	f.act(t, "creator", "commish.franchise.edit", commissioner.RawPeople{Franchise: "franchise.01", Name: "Kings",
		Abbreviation: "KNG"})
	takeover := commissioner.RawPeople{Principal: "cc", Commissioner: "creator"}
	refused("cc", "commish.succession.takeover", takeover, "inactivity period")
	f.clock.now = f.clock.now.Add(24 * time.Hour)
	f.act(t, "cc", "commish.succession.takeover", takeover)
	if !f.state(t).HasRole("cc", governance.Commissioner) {
		t.Fatal("takeover after the period did not seat the co-commissioner")
	}

	// The person a removal vote is about cannot close it; a ballot needs no typed reason.
	f.act(t, "judge", "commish.succession.voteout.open", commissioner.RawPeople{
		ID: "removal", Principal: "judge", Commissioner: "cc"})
	ballot := successionCommand("commish.vote.cast", commissioner.RawPeople{ID: "removal", Principal: "owner0",
		Choice: "no"}, f.revision)
	ballot.Reason = ""
	if _, err := f.store.ExecuteCommissioner(t.Context(), ballot, "owner0", nil); err != nil {
		t.Fatalf("ballot without a reason: %v", err)
	}
	f.revision++
	refused("cc", "commish.vote.close", commissioner.RawPeople{ID: "removal"}, "cannot close it")
	f.act(t, "judge", "commish.vote.close", commissioner.RawPeople{ID: "removal"})
	f.restart(t, "owner0")
}
