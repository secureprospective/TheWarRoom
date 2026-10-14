package governance

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func electionTime() time.Time { return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC) }
func electionLeague(t *testing.T, n int) State {
	t.Helper()
	s := league(t, n)
	s = change(t, s, RoleGranted{Principal: "dot", Role: DOT})
	for i := 0; i < n; i++ {
		s = change(t, s, SeatReplaced{
			Franchise: FranchiseID(fmt.Sprintf("f%02d", i)), Principal: Principal(fmt.Sprintf("p%02d", i)),
		})
	}
	s = change(t, s, StepDown{Principal: "creator"})
	return change(t, s, ElectionOpened{ID: "e", Principal: "dot", Now: electionTime()})
}
func castChoices(t *testing.T, s State, choices []Principal) State {
	t.Helper()
	v, _ := s.Vote("e")
	for i, choice := range choices {
		s = change(t, s, BallotCast{
			ID: "e", Principal: Principal(fmt.Sprintf("p%02d", i)), Choice: choice, Now: v.OpenedAt,
		})
	}
	return s
}
func closeRound(t *testing.T, s State, winner Principal) State {
	t.Helper()
	v, _ := s.Vote("e")
	return change(t, s, VoteClosed{ID: "e", Winner: winner, Now: v.EndsAt})
}
func TestSingleCoCommissioner(t *testing.T) {
	s := change(t, league(t, 2), RoleGranted{Principal: "z", Role: CoCommissioner})
	for _, p := range []Principal{"a", "z"} {
		rejected(t, s, RoleGranted{Principal: p, Role: CoCommissioner})
		_, err := Apply(s, RoleGranted{Principal: p, Role: CoCommissioner})
		if err == nil || !strings.Contains(err.Error(), "This league already has a co-commissioner: revoke them first") {
			t.Fatalf("second seat: %v", err)
		}
	}
	_, err := Replay([]Event{
		LeagueCreated{Creator: "creator", Franchises: []FranchiseID{"f"}},
		RoleGranted{Principal: "z", Role: CoCommissioner}, RoleGranted{Principal: "a", Role: CoCommissioner},
	})
	if err == nil || !strings.Contains(err.Error(), "change 3") {
		t.Fatalf("two-seat history: %v", err)
	}
	s = change(t, s, RoleRevoked{Principal: "z", Role: CoCommissioner})
	s = change(t, s, RoleGranted{Principal: "a", Role: CoCommissioner})
	judge, err := AppealJudge(s, []Principal{"creator"})
	if err != nil || judge != "a" {
		t.Fatalf("single appeal seat: %s %v", judge, err)
	}
	s = change(t, s, StepDown{Principal: "creator"})
	if !s.HasRole("a", Commissioner) || s.HasRole("z", Commissioner) {
		t.Fatal("wrong successor")
	}
}
func TestElectionAllFranchises(t *testing.T) {
	for _, yes := range []int{16, 17} {
		t.Run(fmt.Sprint(yes), func(t *testing.T) {
			s := electionLeague(t, 32)
			for i := 22; i < 32; i++ {
				s = change(t, s, SeatRemoved{Franchise: FranchiseID(fmt.Sprintf("f%02d", i))})
			}
			choices := make([]Principal, 22)
			for i := range choices {
				choices[i] = "b"
				if i < yes {
					choices[i] = "a"
				}
			}
			s = castChoices(t, s, choices)
			winner := Principal("")
			if yes == 17 {
				winner = "a"
			}
			s = closeRound(t, s, winner)
			v, _ := s.Vote("e")
			if v.Runoff != (yes == 16) || v.Closed != (yes == 17) || v.Winner != winner {
				t.Fatalf("%d of 32 with vacancies: %+v", yes, v)
			}
			if s.HasRole("a", Commissioner) != (yes == 17) {
				t.Fatal("majority transfer wrong")
			}
		})
	}
	for _, tc := range []struct{ n, want int }{{1, 1}, {2, 2}, {3, 2}, {4, 3}, {32, 17}} {
		if ElectionThreshold(tc.n) != tc.want {
			t.Fatal("majority threshold wrong")
		}
	}
}
func TestRunoffCandidatesAndPlurality(t *testing.T) {
	for _, tc := range []struct {
		name  string
		votes []Principal
		want  []Principal
	}{
		{"second", []Principal{"a", "a", "a", "b", "b", "c", "c", "d"}, []Principal{"a", "b", "c"}},
		{"first", []Principal{"a", "a", "b", "b", "c", "c", "d"}, []Principal{"a", "b", "c"}},
		{"pair", []Principal{"a", "a", "a", "b", "b", "c"}, []Principal{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := castChoices(t, electionLeague(t, 10), tc.votes)
			s = closeRound(t, s, "")
			v, _ := s.Vote("e")
			if !reflect.DeepEqual(v.Candidates, tc.want) || len(v.FirstBallots) != len(tc.votes) ||
				len(v.Ballots) != 0 || v.EndsAt.Sub(v.OpenedAt) != 7*24*time.Hour {
				t.Fatalf("runoff: %+v", v)
			}
			rejected(t, s, BallotCast{ID: "e", Principal: "p00", Choice: "d", Now: v.OpenedAt})
			// Three votes out of ten win a runoff; no all-franchise majority is needed.
			s = castChoices(t, s, []Principal{"b", "b", "b", "a", "a"})
			rejected(t, s, VoteClosed{ID: "e", Winner: "b", Now: v.EndsAt.Add(-time.Second)})
			s = closeRound(t, s, "b")
			v, _ = s.Vote("e")
			if !v.Closed || !v.Passed || v.Winner != "b" || !s.HasRole("b", Commissioner) {
				t.Fatal("plurality winner not installed")
			}
		})
	}
}
func tiedRunoff(t *testing.T) State {
	t.Helper()
	s := castChoices(t, electionLeague(t, 8), []Principal{"a", "a", "b", "b", "c"})
	s = closeRound(t, s, "")
	return castChoices(t, s, []Principal{"a", "a", "b", "b"})
}
func TestRunoffTieCoCommissioner(t *testing.T) {
	s := tiedRunoff(t)
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	v, _ := s.Vote("e")
	rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: "dot", Choice: "b", Now: v.EndsAt})
	rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "c", Now: v.EndsAt})
	rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "b", Now: v.EndsAt.Add(-time.Second)})
	before := s
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "b", Now: v.EndsAt})
	previous, _ := before.Vote("e")
	if len(previous.TieBallots) != 0 {
		t.Fatal("ruling mutated its parent state")
	}
	rejected(t, s, VoteClosed{ID: "e", Winner: "a", Now: v.EndsAt})
	if !strings.Contains(s.ElectionStatus("e", v.EndsAt), "b wins") {
		t.Fatal("ruling not projected")
	}
	rejected(t, s, BallotCast{ID: "e", Principal: "p00", Choice: "a", Now: v.EndsAt})
	s = closeRound(t, s, "b")
	if !s.HasRole("b", Commissioner) {
		t.Fatal("CC ruling ignored")
	}
	rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "a", Now: v.EndsAt})
}
func TestRunoffTieDOT(t *testing.T) {
	for _, cc := range []Principal{"", "a"} {
		t.Run(string(cc), func(t *testing.T) {
			s := tiedRunoff(t)
			if cc != "" {
				s = change(t, s, RoleGranted{Principal: cc, Role: CoCommissioner})
			}
			for _, p := range []Principal{"d2", "d3", "d4", "a", "b"} {
				s = change(t, s, RoleGranted{Principal: p, Role: DOT})
			}
			v, _ := s.Vote("e")
			for _, p := range []Principal{"a", "b", "outsider"} {
				rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: p, Choice: "b", Now: v.EndsAt})
			}
			for _, p := range []Principal{"dot", "d2"} {
				s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: p, Choice: "b", Now: v.EndsAt})
			}
			rejected(t, s, VoteClosed{ID: "e", Winner: "b", Now: v.EndsAt})
			s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "d3", Choice: "b", Now: v.EndsAt})
			s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "dot", Choice: "a", Now: v.EndsAt})
			rejected(t, s, VoteClosed{ID: "e", Winner: "b", Now: v.EndsAt})
			s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "dot", Choice: "b", Now: v.EndsAt})
			// Three of four eligible judges decide; the two candidates are excluded.
			s = closeRound(t, s, "b")
			if !s.HasRole("b", Commissioner) {
				t.Fatal("DOT majority ignored")
			}
		})
	}
}
func TestRunoffTieNeedsRuling(t *testing.T) {
	s := tiedRunoff(t)
	s = change(t, s, RoleRevoked{Principal: "dot", Role: DOT})
	for _, p := range []Principal{"a", "b"} {
		s = change(t, s, RoleGranted{Principal: p, Role: DOT})
	}
	s = change(t, s, RoleGranted{Principal: "a", Role: CoCommissioner})
	v, _ := s.Vote("e")
	want := "Needs a ruling: the runoff is tied and no one is eligible to break it"
	if s.ElectionStatus("e", v.EndsAt) != want {
		t.Fatal(s.ElectionStatus("e", v.EndsAt))
	}
	rejected(t, s, VoteClosed{ID: "e", Now: v.EndsAt})
	v, _ = s.Vote("e")
	if v.Closed || !s.ElectionNeeded() {
		t.Fatal("unresolved runoff closed")
	}
	// Eligible judge added later can resolve the still-open runoff explicitly.
	s = change(t, s, RoleGranted{Principal: "late", Role: DOT})
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "late", Choice: "b", Now: v.EndsAt})
	s = closeRound(t, s, "b")
	if !s.HasRole("b", Commissioner) {
		t.Fatal("late resolution failed")
	}
}
func TestElectionWinnerChecked(t *testing.T) {
	s := castChoices(t, electionLeague(t, 4), []Principal{"a", "a", "a", "b"})
	v, _ := s.Vote("e")
	rejected(t, s, VoteClosed{ID: "e", Winner: "b", Now: v.EndsAt})
	s = closeRound(t, s, "")
	if !s.HasRole("a", Commissioner) {
		t.Fatal("omitted claim did not compute winner")
	}
}
func TestElectionPeriods(t *testing.T) {
	s := electionLeague(t, 4)
	v, _ := s.Vote("e")
	if v.EndsAt.Sub(v.OpenedAt) != 7*24*time.Hour {
		t.Fatal("default is not seven days")
	}
	s = castChoices(t, s, []Principal{"a", "a", "a"})
	rejected(t, s, VoteClosed{ID: "e", Winner: "a", Now: v.EndsAt.Add(-time.Second)})
	rejected(t, s, VoteClosed{ID: "e", Winner: "a"})
	rejected(t, s, BallotCast{ID: "e", Principal: "p00", Choice: "b", Now: v.OpenedAt.Add(-time.Second)})
	rejected(t, s, BallotCast{ID: "e", Principal: "p00", Choice: "b"})
	s = change(t, s, ElectionPeriodsSet{Election: time.Hour, Runoff: 2 * time.Hour})
	held, _ := s.Vote("e")
	if !held.EndsAt.Equal(v.EndsAt) {
		t.Fatal("setting changed a running election")
	}
	s = closeRound(t, s, "a")
	if !s.HasRole("a", Commissioner) {
		t.Fatal("boundary close failed")
	}
	s = change(t, s, StepDown{Principal: "a"})
	s = change(t, s, ElectionOpened{ID: "next", Principal: "dot", Now: v.EndsAt})
	next, _ := s.Vote("next")
	if next.EndsAt.Sub(next.OpenedAt) != time.Hour {
		t.Fatal("election setting ignored")
	}
	s = change(t, s, BallotCast{ID: "next", Principal: "p00", Choice: "x", Now: next.OpenedAt})
	s = change(t, s, BallotCast{ID: "next", Principal: "p01", Choice: "y", Now: next.OpenedAt})
	s = change(t, s, VoteClosed{ID: "next", Now: next.EndsAt})
	next, _ = s.Vote("next")
	if next.EndsAt.Sub(next.OpenedAt) != 2*time.Hour {
		t.Fatal("runoff setting ignored")
	}
	rejected(t, s, ElectionPeriodsSet{Election: 0, Runoff: time.Hour})
	rejected(t, s, ElectionPeriodsSet{Election: time.Hour, Runoff: -time.Hour})
}

func TestElectionReplayAndIsolation(t *testing.T) {
	at := electionTime()
	events := []Event{
		LeagueCreated{Creator: "creator", Franchises: []FranchiseID{"one", "two"}},
		RoleGranted{Principal: "cc", Role: CoCommissioner},
		StepDown{Principal: "creator"}, SuccessorDeclined{Principal: "cc"},
		SeatReplaced{Franchise: "one", Principal: "owner"},
		ElectionOpened{ID: "e", Principal: "cc", Now: at},
		BallotCast{ID: "e", Principal: "owner", Choice: "a", Now: at},
		BallotCast{ID: "e", Principal: "owner", Choice: "b", Now: at},
		VoteClosed{ID: "e", Now: at.Add(7 * 24 * time.Hour)},
		// With one candidate and no runoff turnout, that candidate has most votes (zero).
		VoteClosed{ID: "e", Now: at.Add(14 * 24 * time.Hour)},
	}
	s, err := Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasRole("b", Commissioner) || s.HasRole("a", Commissioner) {
		t.Fatal("recast ignored")
	}
	fold := State{}
	for _, e := range events {
		fold = change(t, fold, e)
	}
	if !reflect.DeepEqual(s, fold) {
		t.Fatal("election replay differs")
	}
	v, _ := s.Vote("e")
	v.Candidates[0] = "forged"
	v.FirstBallots["one"] = "forged"
	v.Ballots["one"] = "forged"
	v.TieBallots["cc"] = "forged"
	held, _ := s.Vote("e")
	if held.Candidates[0] != "b" || held.FirstBallots["one"] != "b" ||
		held.Ballots["one"] != "" || held.TieBallots["cc"] != "" {
		t.Fatal("vote reading mutated state")
	}
	rejected(t, s, VoteClosed{ID: "e", Winner: "a", Now: at.Add(14 * 24 * time.Hour)})
}
func TestRunoffZeroTurnoutAndRulingRecast(t *testing.T) {
	s := castChoices(t, electionLeague(t, 8), []Principal{"a", "b"})
	s = closeRound(t, s, "")
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	v, _ := s.Vote("e")
	rejected(t, s, VoteClosed{ID: "e", Now: v.EndsAt})
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "a", Now: v.EndsAt})
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "cc", Choice: "b", Now: v.EndsAt})
	rejected(t, s, BallotCast{ID: "e", Principal: "p00", Choice: "a", Now: v.OpenedAt})
	rejected(t, s, VoteClosed{ID: "e", Winner: "a", Now: v.EndsAt})
	s = closeRound(t, s, "b")
	if !s.HasRole("b", Commissioner) {
		t.Fatal("zero-vote tie or ruling recast ignored")
	}
}
