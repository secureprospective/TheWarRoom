package assembly

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Multi-season DEFENSIVE fixtures, one defender per team so each player's within-team
// shares are independent of the others (an "OTH" filler sets each team's denominators).
//   - LB E-1 (Georgia): collapse mean(Tk,Sk,TFL). 2019 shares .10/.05/.05 → mean .067 (below
//     0.12); 2020 shares .15/.15/.15 → mean .15 (CROSSES); 2021 higher. Earliest cross = 2020.
//   - CB E-2 (Bama): collapse mean(PD,INT). 2019 .05/.05, 2020 .08/.08 (both below), 2021
//     .20/.20 → mean .20 (CROSSES). Proves a LATER earliest crossing + a different collapse.
//   - DE E-4 (Miami): collapse mean(TFL,Sk) = .05 every season → NEVER crosses → absent.
//
// PD lives in the "defensive" category; INT lives in the separate "interceptions" category.
func idpDefensive() map[int]string {
	return map[int]string{
		2019: `[
		  {"playerId":"E-1","team":"Georgia","statType":"TOT","stat":"10"},
		  {"playerId":"E-1","team":"Georgia","statType":"SACKS","stat":"5"},
		  {"playerId":"E-1","team":"Georgia","statType":"TFL","stat":"5"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TOT","stat":"90"},
		  {"playerId":"OTHG","team":"Georgia","statType":"SACKS","stat":"95"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TFL","stat":"95"},
		  {"playerId":"E-2","team":"Bama","statType":"PD","stat":"5"},
		  {"playerId":"OTHB","team":"Bama","statType":"PD","stat":"95"},
		  {"playerId":"E-4","team":"Miami","statType":"SACKS","stat":"5"},
		  {"playerId":"E-4","team":"Miami","statType":"TFL","stat":"5"},
		  {"playerId":"OTHM","team":"Miami","statType":"SACKS","stat":"95"},
		  {"playerId":"OTHM","team":"Miami","statType":"TFL","stat":"95"}
		]`,
		2020: `[
		  {"playerId":"E-1","team":"Georgia","statType":"TOT","stat":"15"},
		  {"playerId":"E-1","team":"Georgia","statType":"SACKS","stat":"15"},
		  {"playerId":"E-1","team":"Georgia","statType":"TFL","stat":"15"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TOT","stat":"85"},
		  {"playerId":"OTHG","team":"Georgia","statType":"SACKS","stat":"85"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TFL","stat":"85"},
		  {"playerId":"E-2","team":"Bama","statType":"PD","stat":"8"},
		  {"playerId":"OTHB","team":"Bama","statType":"PD","stat":"92"},
		  {"playerId":"E-4","team":"Miami","statType":"SACKS","stat":"5"},
		  {"playerId":"E-4","team":"Miami","statType":"TFL","stat":"5"},
		  {"playerId":"OTHM","team":"Miami","statType":"SACKS","stat":"95"},
		  {"playerId":"OTHM","team":"Miami","statType":"TFL","stat":"95"}
		]`,
		2021: `[
		  {"playerId":"E-1","team":"Georgia","statType":"TOT","stat":"20"},
		  {"playerId":"E-1","team":"Georgia","statType":"SACKS","stat":"20"},
		  {"playerId":"E-1","team":"Georgia","statType":"TFL","stat":"20"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TOT","stat":"80"},
		  {"playerId":"OTHG","team":"Georgia","statType":"SACKS","stat":"80"},
		  {"playerId":"OTHG","team":"Georgia","statType":"TFL","stat":"80"},
		  {"playerId":"E-2","team":"Bama","statType":"PD","stat":"20"},
		  {"playerId":"OTHB","team":"Bama","statType":"PD","stat":"80"},
		  {"playerId":"E-4","team":"Miami","statType":"SACKS","stat":"5"},
		  {"playerId":"E-4","team":"Miami","statType":"TFL","stat":"5"},
		  {"playerId":"OTHM","team":"Miami","statType":"SACKS","stat":"95"},
		  {"playerId":"OTHM","team":"Miami","statType":"TFL","stat":"95"}
		]`,
	}
}

func idpInterceptions() map[int]string {
	return map[int]string{
		2019: `[
		  {"playerId":"E-2","team":"Bama","statType":"INT","stat":"5"},
		  {"playerId":"OTHB","team":"Bama","statType":"INT","stat":"95"}
		]`,
		2020: `[
		  {"playerId":"E-2","team":"Bama","statType":"INT","stat":"8"},
		  {"playerId":"OTHB","team":"Bama","statType":"INT","stat":"92"}
		]`,
		2021: `[
		  {"playerId":"E-2","team":"Bama","statType":"INT","stat":"20"},
		  {"playerId":"OTHB","team":"Bama","statType":"INT","stat":"80"}
		]`,
	}
}

// idpCrosswalkCSV maps LB/CB/DE defenders and one offense WR (E-5, not in the defensive
// feed) end to end. The WR proves the offense position is skipped by the collapse gate.
const idpCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id
1001,G-1,E-1,
1002,G-2,E-2,
1004,G-4,E-4,
1005,G-5,E-5,
`

// idpBirthCSV: LB G-1 born 2000-09-01 → exactly 20.0y at the 2020-09-01 reference; CB G-2
// born 1999-09-01 → exactly 22.0y at the 2021-09-01 reference. DE G-4 and WR G-5 carry DOBs
// too (their absence is by threshold/position, not a missing birthdate).
const idpBirthCSV = `gsis_id,display_name,position,birth_date,status
G-1,LB One,LB,2000-09-01,ACT
G-2,CB Two,CB,1999-09-01,ACT
G-4,DE Four,DE,2000-01-01,ACT
G-5,WR Five,WR,2001-01-01,ACT
`

func idpPosLookup() fakePosLookup {
	return fakePosLookup{"1001": domain.PosLB, "1002": domain.PosCB, "1004": domain.PosDE, "1005": domain.PosWR}
}

func idpSeasons() []int { return []int{2019, 2020, 2021} }

func idpApprox(a, b float64) bool { return math.Abs(a-b) < 0.02 }

// cfbdDefenseSeasonServer serves the right per-year, per-category defensive fixture,
// asserting the bearer token arrived.
func cfbdDefenseSeasonServer(t *testing.T, wantKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		year, _ := strconv.Atoi(r.URL.Query().Get("year"))
		var body string
		switch r.URL.Query().Get("category") {
		case "defensive":
			body = idpDefensive()[year]
		case "interceptions":
			body = idpInterceptions()[year]
		}
		if body == "" {
			body = "[]"
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestBuildBreakoutAgeIDP_EarliestCrossing proves the defensive end-to-end join: the LB's
// breakout is his EARLIEST crossing (2020, not 2021), the CB crosses only in 2021 on the
// mean(PD,INT) collapse, the never-crossing DE is absent, and the offense WR is skipped by
// the collapse gate (disjoint from the offense breakout feed).
func TestBuildBreakoutAgeIDP_EarliestCrossing(t *testing.T) {
	t.Parallel()
	stats := cfbdDefenseSeasonServer(t, "SECRET")
	cw := crosswalkFixture(t, idpCrosswalkCSV)
	ages := baBirthFixture(t, idpBirthCSV)

	out, err := BuildBreakoutAgeIDP(context.Background(), stats.Client(),
		stats.URL, "SECRET", cw, ages, idpSeasons(),
		[]string{"1001", "1002", "1004", "1005"}, idpPosLookup())
	if err != nil {
		t.Fatalf("BuildBreakoutAgeIDP: %v", err)
	}

	if len(out) != 2 {
		t.Fatalf("got %d breakout ages, want 2 (LB + CB; DE never crosses, WR is offense): %v", len(out), out)
	}
	lb, _ := playerid.New("1001")
	cb, _ := playerid.New("1002")
	de, _ := playerid.New("1004")
	wr, _ := playerid.New("1005")
	if !idpApprox(out[lb], 20.0) {
		t.Errorf("LB breakout age = %v, want ~20.0 (born 2000-09-01, earliest cross 2020)", out[lb])
	}
	if !idpApprox(out[cb], 22.0) {
		t.Errorf("CB breakout age = %v, want ~22.0 (born 1999-09-01, cross 2021 on mean(PD,INT))", out[cb])
	}
	if _, ok := out[de]; ok {
		t.Errorf("DE should be absent (never crosses 0.12 on mean(TFL,Sack))")
	}
	if _, ok := out[wr]; ok {
		t.Errorf("offense WR must be absent (collapse returns no defensive source)")
	}
}

// TestBuildBreakoutAgeIDP_FetchFailsLoud proves a real fetch failure on ANY season surfaces.
func TestBuildBreakoutAgeIDP_FetchFailsLoud(t *testing.T) {
	t.Parallel()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`)) // resolves zero records → collegedefense errEmpty
	}))
	t.Cleanup(empty.Close)
	cw := crosswalkFixture(t, idpCrosswalkCSV)
	ages := baBirthFixture(t, idpBirthCSV)

	_, err := BuildBreakoutAgeIDP(context.Background(), empty.Client(),
		empty.URL, "SECRET", cw, ages, idpSeasons(), []string{"1001"}, idpPosLookup())
	if err == nil {
		t.Fatal("expected a loud error when a season's defensive feed resolves zero records")
	}
}

// TestBuildBreakoutAgeIDP_BadKeyFailsLoud proves a rejected bearer token fails loud.
func TestBuildBreakoutAgeIDP_BadKeyFailsLoud(t *testing.T) {
	t.Parallel()
	stats := cfbdDefenseSeasonServer(t, "SECRET")
	cw := crosswalkFixture(t, idpCrosswalkCSV)
	ages := baBirthFixture(t, idpBirthCSV)

	_, err := BuildBreakoutAgeIDP(context.Background(), stats.Client(),
		stats.URL, "WRONG-KEY", cw, ages, idpSeasons(), []string{"1001"}, idpPosLookup())
	if err == nil {
		t.Fatal("expected a loud error when the CFBD bearer token is rejected")
	}
}

// TestBuildBreakoutAgeIDP_Guards proves the nil-client, nil-pos, and empty-seasons guards.
func TestBuildBreakoutAgeIDP_Guards(t *testing.T) {
	t.Parallel()
	if _, err := BuildBreakoutAgeIDP(context.Background(), nil, "u", "k", crosswalk.Map{}, nil, idpSeasons(), nil, idpPosLookup()); err == nil {
		t.Error("expected error for nil *http.Client")
	}
	if _, err := BuildBreakoutAgeIDP(context.Background(), http.DefaultClient, "u", "k", crosswalk.Map{}, nil, idpSeasons(), nil, nil); err == nil {
		t.Error("expected error for nil PositionLookup")
	}
	if _, err := BuildBreakoutAgeIDP(context.Background(), http.DefaultClient, "u", "k", crosswalk.Map{}, nil, nil, nil, idpPosLookup()); err == nil {
		t.Error("expected error for empty seasons")
	}
}
