package lineup

import (
	"cmp"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func realStarters(t *testing.T) league.Starters {
	t.Helper()
	body, err := os.ReadFile("testdata/league.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		League struct {
			Starters struct {
				Count     string `json:"count"`
				IOP       string `json:"iop_starters"`
				IDP       string `json:"idp_starters"`
				Positions []struct {
					Name  string `json:"name"`
					Limit string `json:"limit"`
				} `json:"position"`
			} `json:"starters"`
		} `json:"league"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	s := raw.League.Starters
	out := league.Starters{Count: s.Count, IOPStarters: s.IOP, IDPStarters: s.IDP}
	for _, p := range s.Positions {
		out.Positions = append(out.Positions, league.PositionLimit{Name: p.Name, Limit: p.Limit})
	}
	return out
}

func TestParseRealRules(t *testing.T) {
	r, err := ParseRules(realStarters(t))
	if err != nil {
		t.Fatal(err)
	}
	want := Rules{OffenseCap: 8, DefenseTotal: 12, StarterCount: 21, Positions: []PositionRule{
		{domain.PosQB, "QB", 1, 1}, {domain.PosRB, "RB", 1, 3}, {domain.PosWR, "WR", 2, 5},
		{domain.PosTE, "TE", 1, 3}, {domain.PosK, "PK", 1, 1}, {domain.PosDT, "DT", 2, 4},
		{domain.PosDE, "DE", 2, 4}, {domain.PosLB, "LB", 2, 4}, {domain.PosCB, "CB", 2, 4},
		{domain.PosS, "S", 2, 4},
	}}
	if !reflect.DeepEqual(r, want) {
		t.Fatalf("got %+v", r)
	}
}

func TestParseRuleFields(t *testing.T) {
	for _, tc := range []struct {
		name, limit string
		bad         bool
		min, max    int
	}{
		{"QB", "1", false, 1, 1}, {"DT", "2-4", false, 2, 4}, {"PK", "1", false, 1, 1},
		{"WR", "4-2", true, 0, 0}, {"WR", "x", true, 0, 0}, {"WR", "-1", true, 0, 0}, {"Z", "1", true, 0, 0},
	} {
		t.Run(tc.name+tc.limit, func(t *testing.T) {
			raw := realStarters(t)
			raw.Positions = []league.PositionLimit{{Name: tc.name, Limit: tc.limit}}
			r, err := ParseRules(raw)
			if tc.bad {
				if err == nil || !strings.Contains(err.Error(), tc.name) {
					t.Fatalf("missing field error: %v", err)
				}
				return
			}
			if err != nil || r.Positions[0].Min != tc.min || r.Positions[0].Max != tc.max {
				t.Fatalf("%+v %v", r, err)
			}
			if tc.name == "PK" && r.Positions[0].Position != domain.PosK {
				t.Fatal("PK not mapped")
			}
		})
	}
	for _, field := range []string{"count", "iop_starters", "idp_starters"} {
		raw := realStarters(t)
		switch field {
		case "count":
			raw.Count = "x"
		case "iop_starters":
			raw.IOPStarters = "x"
		default:
			raw.IDPStarters = "x"
		}
		if _, err := ParseRules(raw); err == nil || !strings.Contains(err.Error(), field) {
			t.Fatal(err)
		}
	}
}

func TestCheck(t *testing.T) {
	rules, err := ParseRules(realStarters(t))
	if err != nil {
		t.Fatal(err)
	}
	// Selected from the archived 2026 players export; ids are 0025's real saved week-5 starters.
	body, err := os.ReadFile("testdata/players-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var players []struct{ Position string }
	if err := json.Unmarshal(body, &players); err != nil {
		t.Fatal(err)
	}
	full := []domain.Position{}
	for _, p := range players {
		pos, _ := normalize.PositionFromMFL(p.Position)
		full = append(full, pos)
	}
	slices.SortStableFunc(full, func(a, b domain.Position) int {
		return cmp.Compare(rules.Order(a), rules.Order(b))
	})

	for _, tc := range []struct {
		name        string
		positions   []domain.Position
		full, legal bool
		field, kind string
	}{
		{"full", full, true, true, "", ""},
		{"partial", full[:19], false, true, "S", "short"},
		{"WR over", append(extra(full, domain.PosWR), domain.PosWR), false, false, "WR", "over"},
		{"offense", extra(full, domain.PosRB), false, false, "QB/RB/WR/TE", "over"},
		{"defense", append(append([]domain.Position{}, full...), domain.PosLB), false, false, "Defense", "over"},
		{"unknown", extra(full, domain.Position("")), false, false, "Position", "unknown"},
		{"flag", extra(full, domain.PosFlag), false, false, "FLAG", "unknown"},
		{"empty", nil, false, true, "WR", "short"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Check(rules, tc.positions)
			if r.Full != tc.full || r.Legal != tc.legal {
				t.Fatalf("%+v", r)
			}
			if tc.kind != "" {
				found := false
				for _, p := range r.Problems {
					if p.Subject == tc.field && p.Kind == tc.kind {
						found = true
					}
				}
				if !found {
					t.Fatalf("missing %s %s: %+v", tc.field, tc.kind, r)
				}
			}
			if !reflect.DeepEqual(r, Check(rules, tc.positions)) {
				t.Fatal("nondeterministic")
			}
		})
	}
}

func extra(full []domain.Position, pos domain.Position) []domain.Position {
	return append(append([]domain.Position{}, full...), pos)
}

func TestCheckMessages(t *testing.T) {
	rules, err := ParseRules(realStarters(t))
	if err != nil {
		t.Fatal(err)
	}
	starters := []domain.Position{domain.PosQB, domain.PosWR, domain.PosK, ""}
	for range 9 {
		starters = append(starters, domain.PosRB)
	}
	got := []string{}
	for _, p := range Check(rules, starters).Problems {
		got = append(got, p.Kind+" "+p.Message)
	}
	want := []string{
		"over RB: 9 starting, at most 3",
		"short WR: 1 starting, needs at least 2",
		"short TE: 0 starting, needs at least 1",
		"short DT: 0 starting, needs at least 2",
		"short DE: 0 starting, needs at least 2",
		"short LB: 0 starting, needs at least 2",
		"short CB: 0 starting, needs at least 2",
		"short S: 0 starting, needs at least 2",
		"unknown Position unknown: 1 starting",
		"over QB/RB/WR/TE: 11 starting, at most 8",
		"short Defense: 0 starting, needs at least 12",
		"short Total: 13 starting, needs at least 21",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}
