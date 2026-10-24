package governance

import (
	"reflect"
	"testing"
	"time"
)

func deputyEvents() []Event {
	return []Event{
		LeagueCreated{Creator: "creator", Franchises: []FranchiseID{"one", "two"}},
		RoleGranted{Principal: "z", Role: CoCommissioner},
		RoleGranted{Principal: "a", Role: CoCommissioner},
		RoleGranted{Principal: "m", Role: CoCommissioner},
	}
}

func TestOrderedDeputiesReplayAndRemoval(t *testing.T) {
	events := deputyEvents()
	s, err := Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.CoCommissioners(), []Principal{"z", "a", "m"}) {
		t.Fatal("priority was sorted instead of granted")
	}
	rejected(t, s, RoleGranted{Principal: "z", Role: CoCommissioner})
	before := s.Checkpoint()
	s = change(t, s, RoleRevoked{Principal: "a", Role: CoCommissioner})
	if !s.HasRole("z", CoCommissioner) || !s.HasRole("m", CoCommissioner) || s.HasRole("a", CoCommissioner) {
		t.Fatal("removal changed another deputy")
	}
	if !reflect.DeepEqual(s.CoCommissioners(), []Principal{"z", "m"}) || !reflect.DeepEqual(before.CoCommissioners, []Principal{"z", "a", "m"}) {
		t.Fatal("order or parent changed")
	}
	order := s.CoCommissioners()
	order[0] = "forged"
	s = change(t, s, RoleGranted{Principal: "a", Role: CoCommissioner})
	if !reflect.DeepEqual(s.CoCommissioners(), []Principal{"z", "m", "a"}) {
		t.Fatal("regrant did not append")
	}
	events = append(events, RoleRevoked{Principal: "a", Role: CoCommissioner}, RoleGranted{Principal: "a", Role: CoCommissioner})
	replayed, err := Replay(events)
	if err != nil || !reflect.DeepEqual(s, replayed) {
		t.Fatalf("replay differs: %v", err)
	}
	for _, p := range s.CoCommissioners() {
		if _, err := Allow(s, p, CommandRef{ID: "CM-021", MinimumRole: Commissioner, COnly: true}); err == nil {
			t.Fatal("deputy granted commissioner-only powers")
		}
	}
}

func TestOrderedDeputySuccession(t *testing.T) {
	for _, removed := range []bool{false, true} {
		s, err := Replay(deputyEvents())
		if err != nil {
			t.Fatal(err)
		}
		s = change(t, s, StepDown{Principal: "creator"})
		if !s.HasRole("z", Commissioner) || s.HasRole("a", Commissioner) {
			t.Fatal("first deputy did not succeed")
		}
		next := Event(SuccessorDeclined{Principal: "z"})
		if removed {
			next = RoleRevoked{Principal: "z", Role: CoCommissioner}
		}
		raw := s.Checkpoint()
		s = change(t, s, next)
		resumed, err := Resume(raw, []Event{next}, 5)
		if err != nil || !reflect.DeepEqual(s, resumed) {
			t.Fatalf("succession checkpoint differs: %v", err)
		}
		if !s.HasRole("a", Commissioner) || s.HasRole("z", Commissioner) || s.ElectionNeeded() {
			t.Fatal("second deputy did not succeed")
		}
		s = change(t, s, SuccessorDeclined{Principal: "a"})
		if !s.HasRole("m", Commissioner) {
			t.Fatal("third deputy did not succeed")
		}
		s = change(t, s, SuccessorDeclined{Principal: "m"})
		if !s.ElectionNeeded() {
			t.Fatal("exhausted list did not require election")
		}
	}
}

func TestOrderedDeputyAppealAndInactivity(t *testing.T) {
	s, err := Replay(deputyEvents())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		parties []Principal
		want    Principal
	}{
		{nil, "creator"}, {[]Principal{"creator"}, "z"}, {[]Principal{"creator", "z"}, "a"},
	} {
		got, err := AppealJudge(s, tc.parties)
		if err != nil || got != tc.want {
			t.Fatalf("appeal judge %s: %v", got, err)
		}
	}
	at := electionTime()
	s = change(t, s, ActivityObserved{Principal: "creator", At: at})
	takeover := InactivityTakeover{Principal: "a", Commissioner: "creator", Now: at.Add(30 * 24 * time.Hour)}
	rejected(t, s, takeover)
	s = change(t, s, RoleRevoked{Principal: "z", Role: CoCommissioner})
	s = change(t, s, takeover)
	if !s.HasRole("a", Commissioner) {
		t.Fatal("highest remaining deputy could not take over")
	}
}

func TestOrderedDeputyRunoffTie(t *testing.T) {
	s := tiedRunoff(t)
	// A candidate is ineligible; remaining priority must not fall back to DOT or lexical order.
	for _, p := range []Principal{"a", "z", "m"} {
		s = change(t, s, RoleGranted{Principal: p, Role: CoCommissioner})
	}
	v, _ := s.Vote("e")
	rejected(t, s, ElectionTieBreakCast{ID: "e", Principal: "m", Choice: "a", Now: v.EndsAt})
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "z", Choice: "a", Now: v.EndsAt})
	s = change(t, s, RoleRevoked{Principal: "z", Role: CoCommissioner})
	rejected(t, s, VoteClosed{ID: "e", Winner: "a", Now: v.EndsAt})
	s = change(t, s, ElectionTieBreakCast{ID: "e", Principal: "m", Choice: "b", Now: v.EndsAt})
	s = closeRound(t, s, "b")
	if !s.HasRole("b", Commissioner) {
		t.Fatal("remaining deputy ruling ignored")
	}
}

func TestDeputyCheckpointValidation(t *testing.T) {
	s, err := Replay(deputyEvents())
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range [][]Principal{{"z", "z", "m"}, {"z", "a"}, {"z", "a", "outsider"}} {
		raw := s.Checkpoint()
		raw.CoCommissioners = order
		if err := raw.Validate(); err == nil {
			t.Fatal("invalid deputy order accepted")
		}
	}
	raw := s.Checkpoint()
	resumed, err := Resume(raw, nil, 4)
	if err != nil || !reflect.DeepEqual(s, resumed) {
		t.Fatalf("order not checkpointed: %v", err)
	}
	raw.CoCommissioners[0] = "forged"
	if resumed.CoCommissioners()[0] != "z" || s.CoCommissioners()[0] != "z" {
		t.Fatal("checkpoint shared deputy slice")
	}
}

func TestLegacyDeputyCheckpoint(t *testing.T) {
	s := change(t, league(t, 1), RoleGranted{Principal: "cc", Role: CoCommissioner})
	raw := s.Checkpoint()
	raw.CoCommissioners, raw.CoCommissioner = nil, "cc"
	resumed, err := Resume(raw, nil, 2)
	if err != nil || !reflect.DeepEqual(s, resumed) {
		t.Fatalf("legacy summary changed: %v", err)
	}
	raw.CoCommissioners = []Principal{"cc"}
	if err := raw.Validate(); err == nil {
		t.Fatal("ambiguous summary accepted")
	}
}
