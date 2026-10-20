package native

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

type heldLookup map[string]normalize.PlayerFacts

func (l heldLookup) IDs() []string {
	ids := []string{}
	for id := range l {
		ids = append(ids, id)
	}
	return append(ids, "9999")
}

func (l heldLookup) Facts(id string) (normalize.PlayerFacts, bool) {
	f, ok := l[id]
	return f, ok
}

func TestPlayerFactsMarkOnlyARealDraftYearAsHeld(t *testing.T) {
	t.Parallel()
	facts := PlayerFacts(heldLookup{
		"0042": {Name: "Kicker, A", Team: "AAA", Position: domain.PosK, DraftYear: 2020, HasDraftYear: true},
		"0043": {Name: "Backer, B", Team: "BBB", Position: domain.PosLB, DraftYear: 1970, HasDraftYear: true},
		"0044": {Name: "Safety, C", Team: "CCC", Position: domain.PosS},
	})
	want := nativecheck.Facts{
		"0042": {Name: "Kicker, A", Team: "AAA", Position: leaguerules.PK, DraftYear: 2020, DraftYearHeld: true},
		"0043": {Name: "Backer, B", Team: "BBB", Position: leaguerules.Position("LB"), DraftYear: 1970},
		"0044": {Name: "Safety, C", Team: "CCC", Position: leaguerules.Position("S")},
	}
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("an id the directory cannot resolve is skipped, and MFL's 1970 is no draft year:\n got %+v\nwant %+v", facts, want)
	}
	Designate(facts, map[string]string{"0042": " IR-PUP ", "0043": "Questionable"})
	for id, want := range map[string]leaguerules.NFLStatus{"0042": leaguerules.PUP, "0043": "Questionable", "0044": ""} {
		if f := facts[id]; !f.DesignationHeld || f.Designation != want {
			t.Fatalf("%s: listing %+v, want held %q", id, f, want)
		}
	}
}

func TestPositionAndDesignationSpeakTheLeagueRules(t *testing.T) {
	t.Parallel()
	for p, want := range map[domain.Position]leaguerules.Position{domain.PosK: leaguerules.PK, domain.PosFlag: "",
		domain.PosDT: leaguerules.Position("DT")} {
		if got := Position(p); got != want {
			t.Errorf("%s: %q, want %q", p, got, want)
		}
	}
	for _, c := range []struct {
		status string
		want   leaguerules.NFLStatus
	}{{"", ""}, {"IR", leaguerules.InjuredReserve}, {"ir-r", leaguerules.InjuredReserve}, {"IR-NFI", leaguerules.NFI},
		{"Out", leaguerules.Out}, {"suspended", leaguerules.Suspended}, {" Doubtful ", "Doubtful"}} {
		if got := Designation(c.status); got != c.want {
			t.Errorf("%q: %q, want %q", c.status, got, c.want)
		}
	}
}

func TestDecodeLineupSetReadsOneLineupForAnAppLeague(t *testing.T) {
	t.Parallel()
	in, err := DecodeLineupSet([]byte(`{"league":"native.seed","franchise":"franchise.02","season":2026,"week":1,
		"starters":["1000"],"expected":3,"request":"r"}`))
	want := LineupSet{League: "native.seed", Franchise: "franchise.02", Season: 2026, Week: 1,
		Starters: []string{"1000"}, Expected: 3, Request: "r"}
	if err != nil || !reflect.DeepEqual(in, want) {
		t.Fatalf("%+v %v", in, err)
	}
	for body, words := range map[string]string{
		`{"league":"native.seed","request":"r","extra":1}`:     `read the lineup: json: unknown field "extra"`,
		`{"league":"native.seed","request":"r"} {}`:            "send exactly one lineup",
		`{"league":"native.","request":"r"}`:                   "choose an app league and a valid request and league version",
		`{"league":"native.seed ","request":"r"}`:              "choose an app league and a valid request and league version",
		`{"league":"native.seed","request":" "}`:               "choose an app league and a valid request and league version",
		`{"league":"native.seed","request":"r","expected":-1}`: "choose an app league and a valid request and league version",
	} {
		if _, err := DecodeLineupSet([]byte(body)); err == nil || err.Error() != words {
			t.Errorf("%s: %v, want %q", body, err, words)
		}
	}
}
