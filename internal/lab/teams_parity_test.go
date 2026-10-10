package lab

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"testing"
)

type teamsGoldenHeader struct {
	DataSHA   string `json:"dataSHA"`
	EngineSHA string `json:"engineSHA"`
	TeamsSHA  string `json:"teamsSHA"`
	Node      string `json:"node"`
}
type teamsFixture struct {
	Name   string                     `json:"name"`
	Scope  TeamScope                  `json:"scope"`
	Values map[string]json.RawMessage `json:"values"`
}

// projectSides rewrites every "sides" array of side objects as the golden's
// compact tradeID:fid references, the same projection the generator applies.
func projectSides(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			if k == "sides" {
				if sides, ok := val.([]any); ok {
					refs := make([]any, len(sides))
					for i, s := range sides {
						m, ok := s.(map[string]any)
						if !ok {
							refs[i] = projectSides(s)
							continue
						}
						refs[i] = fmt.Sprintf("%.0f:%s", m["tradeID"], m["fid"])
					}
					out[k] = refs
					continue
				}
			}
			out[k] = projectSides(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = projectSides(val)
		}
		return out
	}
	return v
}

func TestTeamsParity(t *testing.T) {
	d := loadData(t)
	b := readGoldenBytes(t, "testdata/goldens/teams.json.gz")
	var golden struct {
		Header     teamsGoldenHeader `json:"header"`
		TradeSides []any             `json:"tradeSides"`
		TierOf     []any             `json:"tierOf"`
		Fixtures   []teamsFixture    `json:"fixtures"`
	}
	if err := json.Unmarshal(b, &golden); err != nil {
		t.Fatal(err)
	}
	if golden.Header.DataSHA != DataSHA || golden.Header.EngineSHA != "d7a914bcf5f12c22b004a513d0c1a443356fad2bffc3ef6080f788283d296cfd" || golden.Header.Node != "v24.21.0" {
		t.Fatal("invalid teams golden provenance")
	}
	if golden.Header.TeamsSHA != "dc9c99d432b89ed11edc9ed0a1e8080433804932d7c5226bb683d31567b98ca2" {
		t.Fatal("teams golden stamped against a different teams.js")
	}
	t.Run("tradeSides", func(t *testing.T) {
		maxError := 0.0
		if err := parity(golden.TradeSides, jsonValue(t, TradeSides(d)), "tradeSides", &maxError); err != nil {
			t.Fatal(err)
		}
		t.Logf("tradeSides max absolute error %.17g", maxError)
	})
	t.Run("tierOf", func(t *testing.T) {
		got := make([]string, 7)
		for i, rank := range []int{0, 1, 10, 11, 22, 23, 32} {
			got[i] = TierOf(rank)
		}
		maxError := 0.0
		if err := parity(golden.TierOf, jsonValue(t, got), "tierOf", &maxError); err != nil {
			t.Fatal(err)
		}
	})
	for _, fixture := range golden.Fixtures {
		t.Run(fixture.Name, func(t *testing.T) {
			league := SidesInScope(d, TeamScope{From: fixture.Scope.From, To: fixture.Scope.To})
			sides := SidesInScope(d, fixture.Scope)
			computed := map[string]any{
				"sidesInScope":  jsonValue(t, sideRefs(sides)),
				"behaviour":     jsonValue(t, Behaviour(sides, nil)),
				"againstLeague": jsonValue(t, Behaviour(sides, league)),
				"tierTrends":    jsonValue(t, TierTrends(sides)),
				"tierPairs":     jsonValue(t, TierPairs(sides)),
				"teamRows":      projectSides(jsonValue(t, TeamRows(sides, league))),
			}
			profiles := make(map[string]any)
			var profileFids map[string]json.RawMessage
			if err := json.Unmarshal(fixture.Values["teamProfile"], &profileFids); err != nil {
				t.Fatal(err)
			}
			for fid := range profileFids {
				profiles[fid] = projectSides(jsonValue(t, TeamProfile(d, sides, league, fid)))
			}
			computed["teamProfile"] = profiles
			for _, name := range keysRaw(fixture.Values) {
				var want any
				if err := json.Unmarshal(fixture.Values[name], &want); err != nil {
					t.Fatal(err)
				}
				maxError := 0.0
				if err := parity(want, computed[name], name, &maxError); err != nil {
					t.Fatalf("%s: %v", fixture.Name, err)
				}
				t.Logf("%s %s max absolute error %.17g", fixture.Name, name, maxError)
			}
		})
	}
}

func sideRefs(sides []TeamSide) []string {
	out := make([]string, len(sides))
	for i, s := range sides {
		out[i] = fmt.Sprintf("%d:%s", s.TradeID, s.Fid)
	}
	return out
}

func TestTeamsMeasured(t *testing.T) {
	d := loadData(t)
	sides := TradeSides(d)
	valued := 0
	for _, trade := range d.Trades {
		if trade.ANet != nil {
			valued++
		}
	}
	if len(sides) != 2*valued {
		t.Fatal("each valued trade must have two sides")
	}
	sum := 0.0
	for _, side := range sides {
		sum += side.Now
		if !slices.Contains(TeamPostures(), side.Posture) {
			t.Fatal("unknown posture")
		}
	}
	if math.Abs(sum) >= 1e-6 {
		t.Fatal("on-the-day sides must cancel")
	}
	scope := SidesInScope(d, TeamScope{From: 2017, To: 2026, Tiers: []string{"rebuild"}, Windows: []string{"season"}})
	if len(scope) == 0 {
		t.Fatal("empty measured rebuild scope")
	}
	for _, side := range scope {
		if side.Tier != "rebuild" || side.Window != "season" {
			t.Fatal("scope leaked")
		}
	}
	league := SidesInScope(d, TeamScope{From: 2017, To: 2026})
	rows := TeamRows(league, league)
	if len(rows) != 32 || len(TierTrends(league).Tiers) != 3 {
		t.Fatal("measured teams or tiers missing")
	}
	total := 0
	for _, row := range rows {
		total += row.N
	}
	if total != len(league) {
		t.Fatal("team counts lost sides")
	}
	profile := TeamProfile(d, league, league, rows[0].Fid)
	total = 0
	for _, season := range profile.BySeason {
		total += season.N
	}
	if len(profile.Sides) != rows[0].N || total != rows[0].N {
		t.Fatal("season counts lost sides")
	}
	share := Behaviour(league, nil).Posture["swap"].P
	if share == nil || *share <= 0.4 {
		t.Fatal("measured like-for-like share must exceed 40%")
	}
}
