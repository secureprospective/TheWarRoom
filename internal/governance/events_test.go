package governance

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestInvitesSeatsReplay(t *testing.T) {
	events := []Event{
		LeagueCreated{Creator: "creator", Franchises: []FranchiseID{"f00", "f01"}},
		FranchiseIdentitySet{Franchise: "f00", Name: "Franchise", Abbreviation: "FR"},
		InviteIssued{ID: "one", Franchise: "f00", Redeemer: "owner"},
		InviteRedeemed{ID: "one", Principal: "owner"},
		SeatReplaced{Franchise: "f00", Principal: "replacement"},
		InviteIssued{ID: "two", Franchise: "f01", Redeemer: "second"},
		InviteRevoked{ID: "two"}, RoleGranted{Principal: "cc", Role: CoCommissioner},
		AbilitySet{All: true, Ability: "trades", On: true}, VisibilitySet{Report: "cap", Audience: "all"},
		ActivityObserved{Principal: "creator", At: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
	}
	s, err := Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []Event{
		InviteRedeemed{ID: "one", Principal: "owner"}, InviteRedeemed{ID: "two", Principal: "second"},
		InviteIssued{ID: "one", Franchise: "f01", Redeemer: "third"},
		SeatReplaced{Franchise: "f01", Principal: "replacement"},
	} {
		rejected(t, s, e)
	}
	f, _ := s.Franchise("f00")
	if f.ID != "f00" || f.Name != "Franchise" || f.Seat != "replacement" {
		t.Fatal("replacement changed franchise identity")
	}
	fold := State{}
	for _, event := range events {
		fold = change(t, fold, event)
	}
	if !reflect.DeepEqual(s, fold) {
		t.Fatal("replay differs from incremental fold")
	}
	replay, err := Replay(events)
	if err != nil || !reflect.DeepEqual(s, replay) {
		t.Fatal("replay differs")
	}
	s = change(t, s, SeatRemoved{Franchise: "f00"})
	rejected(t, s, InviteRedeemed{ID: "one", Principal: "owner"})
	f, _ = s.Franchise("f00")
	if f.Seat != "" || f.Name != "Franchise" {
		t.Fatal("removal destroyed franchise")
	}
	s = change(t, s, InviteIssued{ID: "bound", Franchise: "f01", Redeemer: "right"})
	rejected(t, s, InviteRedeemed{ID: "bound", Principal: "wrong"})
	s = change(t, s, InviteRedeemed{ID: "bound", Principal: "right"})
	invite, _ := s.Invite("bound")
	if !invite.Redeemed || invite.Franchise != "f01" {
		t.Fatal("invite binding lost")
	}
}
func votingLeague(t *testing.T) State {
	t.Helper()
	s := league(t, 32)
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	for i := 0; i < 22; i++ {
		s = change(t, s, SeatReplaced{Franchise: FranchiseID(fmt.Sprintf("f%02d", i)), Principal: Principal(fmt.Sprintf("p%02d", i))})
	}
	return s
}
func TestVoteOutAllFranchises(t *testing.T) {
	for _, yes := range []int{21, 22} {
		t.Run(fmt.Sprint(yes), func(t *testing.T) {
			s := votingLeague(t)
			rejected(t, s, VoteOutOpened{ID: "owner-open", Principal: "p00", Commissioner: "creator"})
			s = change(t, s, VoteOutOpened{ID: "vote", Principal: "cc", Commissioner: "creator"})
			rejected(t, s, BallotCast{ID: "vote", Principal: "unseated", Choice: "yes"})
			for i := 0; i < yes; i++ {
				s = change(t, s, BallotCast{ID: "vote", Principal: Principal(fmt.Sprintf("p%02d", i)), Choice: "yes"})
			}
			s = change(t, s, BallotCast{ID: "vote", Principal: "p00", Choice: "no"})
			v, _ := s.Vote("vote")
			if len(v.Ballots) != yes || v.Ballots["f00"] != "no" {
				t.Fatal("recast did not replace")
			}
			s = change(t, s, BallotCast{ID: "vote", Principal: "p00", Choice: "yes"})
			s = change(t, s, VoteClosed{ID: "vote"})
			v, _ = s.Vote("vote")
			if v.Passed != (yes == 22) || !v.Closed {
				t.Fatalf("%d yes with 10 vacant: %+v", yes, v)
			}
			if s.HasRole("creator", Commissioner) != (yes == 21) {
				t.Fatal("vote result did not change commissioner")
			}
			rejected(t, s, BallotCast{ID: "vote", Principal: "p00", Choice: "no"})
		})
	}
	for _, tc := range []struct{ n, want int }{{1, 1}, {2, 2}, {3, 2}, {4, 3}, {32, 22}} {
		if VoteOutThreshold(tc.n) != tc.want {
			t.Fatal("two-thirds ceiling wrong")
		}
	}
}
func TestSuccession(t *testing.T) {
	s := league(t, 2)
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	s = change(t, s, StepDown{Principal: "creator"})
	if !s.HasRole("cc", Commissioner) || s.HasRole("creator", Commissioner) {
		t.Fatal("co-commissioner did not take over first")
	}
	s = change(t, s, SuccessorDeclined{Principal: "cc"})
	if !s.ElectionNeeded() || s.HasRole("creator", Commissioner) || s.HasRole("cc", Commissioner) {
		t.Fatal("decline did not require election without restoring outgoing authority")
	}
	rejected(t, s, ElectionOpened{ID: "election", Principal: "owner"})
	s = change(t, s, SeatReplaced{Franchise: "f00", Principal: "owner"})
	s = change(t, s, ElectionOpened{ID: "election", Principal: "cc"})
	s = change(t, s, BallotCast{ID: "election", Principal: "owner", Choice: "new"})
	rejected(t, s, VoteClosed{ID: "election", Winner: "unvoted"})
	s = change(t, s, VoteClosed{ID: "election", Winner: "new"})
	if !s.HasRole("new", Commissioner) || s.HasRole("creator", Commissioner) || s.ElectionNeeded() {
		t.Fatal("elected successor not installed")
	}
	for _, tc := range []struct {
		successor Principal
		cc        bool
	}{{"named", false}, {"", true}} {
		s = league(t, 1)
		if tc.cc {
			s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
		}
		s = change(t, s, StepDown{Principal: "creator", Successor: tc.successor})
		if tc.cc {
			s = change(t, s, SuccessorAccepted{Principal: "cc"})
		}
		if s.ElectionNeeded() || s.HasRole("creator", Commissioner) {
			t.Fatal("succession not completed")
		}
	}
	s = league(t, 1)
	s = change(t, s, StepDown{Principal: "creator"})
	if !s.ElectionNeeded() {
		t.Fatal("no co-commissioner did not require election")
	}
}
func TestInactivityTakeover(t *testing.T) {
	s := league(t, 1)
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	s = change(t, s, ActivityObserved{Principal: "creator", At: at})
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	for _, e := range []Event{
		InactivityTakeover{Principal: "owner", Commissioner: "creator", Now: at.Add(14 * 24 * time.Hour), InSeason: true},
		InactivityTakeover{Principal: "cc", Commissioner: "creator", Now: at.Add(14*24*time.Hour - time.Second), InSeason: true},
	} {
		rejected(t, s, e)
	}
	s = change(t, s, InactivityTakeover{Principal: "cc", Commissioner: "creator", Now: at.Add(14 * 24 * time.Hour), InSeason: true})
	if !s.HasRole("cc", Commissioner) || s.HasRole("creator", Commissioner) {
		t.Fatal("takeover not recorded")
	}
}
