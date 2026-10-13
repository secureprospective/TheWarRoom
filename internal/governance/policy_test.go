package governance

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func league(t *testing.T, n int) State {
	t.Helper()
	ids := make([]FranchiseID, n)
	for i := range ids {
		ids[i] = FranchiseID(fmt.Sprintf("f%02d", i))
	}
	return change(t, State{}, LeagueCreated{Creator: "creator", Franchises: ids})
}
func change(t *testing.T, s State, e Event) State {
	t.Helper()
	n, err := Apply(s, e)
	if err != nil {
		t.Fatal(err)
	}
	return n
}
func rejected(t *testing.T, s State, e Event) {
	t.Helper()
	before := s.clone()
	n, err := Apply(s, e)
	if err == nil {
		t.Fatalf("accepted forbidden %T", e)
	}
	if !reflect.DeepEqual(s, before) || !reflect.DeepEqual(n, before) {
		t.Fatal("refusal mutated state")
	}
}
func TestCreatorLastCommissioner(t *testing.T) {
	s := league(t, 2)
	if !s.HasRole("creator", Commissioner) {
		t.Fatal("creator not commissioner")
	}
	rejected(t, s, RoleRevoked{Principal: "creator", Role: Commissioner})
	s = change(t, s, SeatReplaced{Franchise: "f00", Principal: "creator"})
	for _, e := range []Event{SeatRemoved{Franchise: "f00"}, SeatReplaced{Franchise: "f00", Principal: "other"}} {
		rejected(t, s, e)
	}
	s = change(t, s, RoleGranted{Principal: "other", Role: Commissioner})
	s = change(t, s, RoleRevoked{Principal: "creator", Role: Commissioner})
	if s.HasRole("creator", Commissioner) {
		t.Fatal("revocation ignored")
	}
}
func TestCensusPermissions(t *testing.T) {
	s := change(t, league(t, 2), RoleGranted{Principal: "cc", Role: CoCommissioner})
	rows := CensusControls()
	if len(rows) != 56 {
		t.Fatalf("census has %d controls", len(rows))
	}
	blocked := []string{"CM-001", "CM-002", "CM-005", "CM-021", "CM-027", "CM-033", "CM-055"}
	for _, row := range rows {
		if row.MinimumRole == "" {
			continue
		}
		t.Run(row.ID, func(t *testing.T) {
			_, err := Allow(s, "cc", CommandRef{ID: row.ID, MinimumRole: row.MinimumRole, COnly: row.COnly})
			want := false
			for _, id := range blocked {
				want = want || id == row.ID
			}
			if (err != nil) != want {
				t.Fatalf("co-commissioner permission: %v", err)
			}
		})
	}
	_, err := Allow(s, "cc", CommandRef{ID: "CM-021", MinimumRole: Owner})
	if err == nil {
		t.Fatal("downgraded census permissions accepted")
	}
	s = change(t, s, RoleGranted{Principal: "dot", Role: DOT})
	s = change(t, s, SeatReplaced{Franchise: "f00", Principal: "owner"})
	s = change(t, s, AbilitySet{Franchise: "f00", Ability: "lineups", On: true})
	s = change(t, s, AbilitySet{Franchise: "f00", Ability: "trades", On: false})
	for _, tc := range []struct {
		p       Principal
		c       CommandRef
		allowed bool
	}{
		{"dot", CommandRef{ID: "CM-037", MinimumRole: DOT}, true},
		{"dot", CommandRef{ID: "succession.voteout.open", MinimumRole: DOT}, true},
		{"dot", CommandRef{ID: "succession.election.open", MinimumRole: DOT}, true},
		{"owner", CommandRef{ID: "vote.cast", MinimumRole: Owner}, true},
		{"dot", CommandRef{ID: "CM-018", MinimumRole: CoCommissioner}, false},
		{"owner", CommandRef{ID: "lineup.set", MinimumRole: Owner, Ability: "lineups", Franchise: "f00"}, true},
		{"owner", CommandRef{ID: "lineup.set", MinimumRole: Owner, Ability: "lineups", Franchise: "f01"}, false},
		{"owner", CommandRef{ID: "trade.offer", MinimumRole: Owner, Ability: "trades"}, false},
		{"owner", CommandRef{ID: "CM-056", MinimumRole: Owner}, true},
		{"outsider", CommandRef{ID: "CM-056", MinimumRole: Owner}, false},
	} {
		_, err := Allow(s, tc.p, tc.c)
		if (err == nil) != tc.allowed {
			t.Fatalf("%s %s: %v", tc.p, tc.c.ID, err)
		}
	}
}
func TestDOTReviewAndAppeal(t *testing.T) {
	s := league(t, 2)
	members := []Principal{}
	for i := 0; i < 10; i++ {
		p := Principal(fmt.Sprintf("dot%d", i))
		members = append(members, p)
		s = change(t, s, RoleGranted{Principal: p, Role: DOT})
		if i == 2 {
			err := TradeReviewReady(s)
			if err == nil || err.Error() != "Needs a ruling: DOT has 3 of 4 seats" {
				t.Fatalf("three seats: %v", err)
			}
			if _, err := Reviewers(s, nil); err == nil {
				t.Fatal("underfilled DOT reviewed")
			}
		}
	}
	rejected(t, s, RoleGranted{Principal: "eleventh", Role: DOT})
	reviewers, err := Reviewers(s, []Principal{"dot0", "dot1", "dot2"})
	if err != nil || len(reviewers) != 7 || reviewers[0] != "dot3" {
		t.Fatalf("recusal: %v %v", reviewers, err)
	}
	if _, err := Reviewers(s, members); err == nil {
		t.Fatal("all recused did not fail closed")
	}
	s = change(t, s, RoleGranted{Principal: "cc", Role: CoCommissioner})
	for _, tc := range []struct {
		parties []Principal
		judge   Principal
		fail    bool
	}{
		{nil, "creator", false}, {[]Principal{"creator"}, "cc", false},
		{[]Principal{"creator", "cc"}, "", true},
	} {
		p, err := AppealJudge(s, tc.parties)
		if p != tc.judge || (err != nil) != tc.fail {
			t.Fatalf("appeal: %s %v", p, err)
		}
	}
}
func TestInactivityEvidence(t *testing.T) {
	s := league(t, 1)
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	if ok, err := InactivityEligible(s, "creator", at.Add(1000*time.Hour), true); ok || err == nil {
		t.Fatal("inferred activity")
	}
	s = change(t, s, ActivityObserved{Principal: "creator", At: at})
	for _, tc := range []struct {
		season bool
		days   int
	}{{true, 14}, {false, 30}} {
		for _, delta := range []time.Duration{-time.Second, 0} {
			ok, err := InactivityEligible(s, "creator", at.Add(time.Duration(tc.days)*24*time.Hour+delta), tc.season)
			if err != nil || ok != (delta == 0) {
				t.Fatalf("threshold %d %v: %v %v", tc.days, delta, ok, err)
			}
		}
	}
	s = change(t, s, ActivityObserved{Principal: "creator", At: at.Add(-time.Hour)})
	if s.activity["creator"] != at {
		t.Fatal("old observation replaced latest activity")
	}
	s = change(t, s, InactivitySet{InSeason: time.Hour, Offseason: 2 * time.Hour})
	if ok, err := InactivityEligible(s, "creator", at.Add(time.Hour), true); !ok || err != nil {
		t.Fatal("settings ignored")
	}
}
func TestPrivacyVisibilityIdentity(t *testing.T) {
	s := league(t, 2)
	for _, id := range []FranchiseID{"f00", "f01"} {
		f, _ := s.Franchise(id)
		for _, ability := range []string{"lineups", "add/drop", "drop-only", "trades", "pick trades",
			"IR", "taxi", "articles", "board", "chat", "polls"} {
			if !f.Abilities[ability] {
				t.Fatalf("%s default %s is off", id, ability)
			}
		}
		if f.Abilities["customize franchise"] || f.Abilities["home page"] {
			t.Fatal("customization default on")
		}
	}
	s = change(t, s, FranchiseIdentitySet{Franchise: "f00", Name: "New name", Abbreviation: "NEW"})
	s = change(t, s, VisibilitySet{Report: "cap", Audience: "commish"})
	s = change(t, s, AbilitySet{All: true, Ability: "trades", On: true})
	f, _ := s.Franchise("f00")
	if f.Name != "New name" || f.Abbreviation != "NEW" || !f.Abilities["trades"] || s.Audience("cap") != "commish" {
		t.Fatal("identity, visibility or ability not projected")
	}
	f.Abilities["trades"] = false
	f, _ = s.Franchise("f00")
	if !f.Abilities["trades"] {
		t.Fatal("read leaked mutable abilities")
	}
	for _, e := range []Event{RoleGranted{}, RoleRevoked{}, InviteIssued{}, InviteRedeemed{}, InviteRevoked{}} {
		if !e.Private() {
			t.Fatalf("%T leaked to owner ledger", e)
		}
	}
	for _, e := range []Event{SeatRemoved{}, SeatReplaced{}, AbilitySet{}, VisibilitySet{}, StepDown{}} {
		if e.Private() {
			t.Fatalf("%T hides league state", e)
		}
	}
	rejected(t, s, VisibilitySet{Report: "cap", Audience: "unknown"})
	if _, err := Allow(s, "", CommandRef{}); err == nil || !strings.Contains(err.Error(), "identify") {
		t.Fatal("anonymous permission")
	}
}
