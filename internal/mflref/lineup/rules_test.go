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
	r, err := ParseRules(league.RawConfig{Starters: realStarters(t)})
	if err != nil {
		t.Fatal(err)
	}
	if r.StarterCount != 21 {
		t.Fatalf("starter count %d", r.StarterCount)
	}
	order := []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDT, domain.PosDE, domain.PosLB, domain.PosCB, domain.PosS}
	for i, pos := range order {
		if r.Order(pos) != i {
			t.Errorf("%s ordered %d, want %d", pos, r.Order(pos), i)
		}
	}
	if r.Order(domain.PosFlag) != len(order) {
		t.Error("an unlisted position was not placed last")
	}
	mins := map[domain.Position]int{domain.PosQB: 1, domain.PosRB: 1, domain.PosWR: 2, domain.PosK: 1, domain.PosS: 2}
	for pos, want := range mins {
		if got := r.minimum(pos); got != want {
			t.Errorf("%s minimum %d, want %d", pos, got, want)
		}
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
			r, err := ParseRules(league.RawConfig{Starters: raw})
			if tc.bad {
				if err == nil || !strings.Contains(err.Error(), tc.name) && !strings.Contains(err.Error(), tc.limit) {
					t.Fatalf("missing field error: %v", err)
				}
				return
			}
			pos, _ := normalize.PositionFromMFL(tc.name)
			b := r.Native.Positions.Value[native(pos)]
			if err != nil || b.Min.Value != tc.min || b.Max.Value != tc.max {
				t.Fatalf("%+v %v", r, err)
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
		if _, err := ParseRules(league.RawConfig{Starters: raw}); err == nil || !strings.Contains(err.Error(), `"x"`) {
			t.Fatal(err)
		}
	}
}

func TestCheck(t *testing.T) {
	rules, err := ParseRules(league.RawConfig{Starters: realStarters(t)})
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
		{"offense", extra(full, domain.PosRB), false, false, "Offense", "over"},
		{"defense", append(append([]domain.Position{}, full...), domain.PosLB), false, false, "Defense", "over"},
		{"unknown", extra(full, domain.Position("")), false, false, "", "unknown"},
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
	rules, err := ParseRules(league.RawConfig{Starters: realStarters(t)})
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
		"over Offense: 11 starting, needs exactly 8",
		"short Defense: 0 starting, needs exactly 12",
		"short Total: 13 starting, needs exactly 21",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
}
