package lineup

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

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
