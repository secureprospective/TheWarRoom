package assembly

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/agetrajectory"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Multi-season CFBD fixtures. Georgia team totals are held at 1000 receiving / 1000
// rushing yards each season (a filler "OTH" absorbs the remainder) so a share is the
// player's YDS ÷ 1000. WR E-1 crosses the 0.20 dominator line in 2020 (0.30) — NOT
// 2019 (0.10) — so his EARLIEST breakout season is 2020. RB E-3 crosses on RUSHING
// share only in 2021 (0.50), below in 2019/2020 — proving the position-specific share
// selector and a later crossing. TE E-2 never crosses (0.05 every year) → absent. On a
// separate team, WR E-4 crosses in 2020 but has NO birthdate row → absent (the DOB gate).
func baReceiving() map[int]string {
	return map[int]string{
		2019: `[
	  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"YDS","stat":"100"},
	  {"playerId":"E-2","player":"TE Two","team":"Georgia","statType":"YDS","stat":"50"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"850"}
	]`,
		2020: `[
	  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"YDS","stat":"300"},
	  {"playerId":"E-2","player":"TE Two","team":"Georgia","statType":"YDS","stat":"50"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"650"},
	  {"playerId":"E-4","player":"WR Four","team":"Bama","statType":"YDS","stat":"300"},
	  {"playerId":"OTH2","player":"Filler2","team":"Bama","statType":"YDS","stat":"700"}
	]`,
		2021: `[
	  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"YDS","stat":"400"},
	  {"playerId":"E-2","player":"TE Two","team":"Georgia","statType":"YDS","stat":"50"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"550"}
	]`,
	}
}

func baRushing() map[int]string {
	return map[int]string{
		2019: `[
	  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"YDS","stat":"100"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"900"}
	]`,
		2020: `[
	  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"YDS","stat":"150"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"850"}
	]`,
		2021: `[
	  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"YDS","stat":"500"},
	  {"playerId":"OTH","player":"Filler","team":"Georgia","statType":"YDS","stat":"500"}
	]`,
	}
}

// baCrosswalkCSV closes both bridges for four players end to end.
const baCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id
1001,G-1,E-1,
1002,G-2,E-2,
1003,G-3,E-3,
1004,G-4,E-4,
`

// baBirthCSV gives DOBs for G-1/G-2/G-3 but deliberately OMITS G-4 (the no-DOB path).
// G-1 born 2000-09-01 → exactly 20.0y at the 2020-09-01 reference; G-3 born 1999-09-01
// → exactly 22.0y at the 2021-09-01 reference (both spans land on 365.25-year integers).
const baBirthCSV = `gsis_id,display_name,position,birth_date,status
G-1,WR One,WR,2000-09-01,ACT
G-2,TE Two,TE,2000-01-01,ACT
G-3,RB Three,RB,1999-09-01,ACT
`

func baSeasons() []int { return []int{2019, 2020, 2021} }

// cfbdSeasonServer serves the right per-year, per-category fixture, asserting the bearer
// token arrived. An unknown (year, category) is a 400 — the fetcher should never ask.
func cfbdSeasonServer(t *testing.T, wantKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		year, _ := strconv.Atoi(r.URL.Query().Get("year"))
		var body string
		switch r.URL.Query().Get("category") {
		case "receiving":
			body = baReceiving()[year]
		case "rushing":
			body = baRushing()[year]
		}
		if body == "" {
			body = "[]" // a played season is never empty; an unexpected ask returns nothing
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func baBirthServer(t *testing.T, csv string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(csv))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// baBirthFixture stands up the birthdate fixture server and returns the RESOLVED birthdate
// map. Birthdates are now fetched once by the app (behind the CFBD-key gate) and threaded
// into BuildBreakoutAge, so the test mirrors that by fetching here and passing the map.
func baBirthFixture(t *testing.T, csv string) map[string]agetrajectory.RawAge {
	t.Helper()
	srv := baBirthServer(t, csv)
	ages, err := agetrajectory.Fetch(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("birthdate fixture fetch: %v", err)
	}
	return ages
}

func baPosLookup() fakePosLookup {
	return fakePosLookup{"1001": domain.PosWR, "1002": domain.PosTE, "1003": domain.PosRB, "1004": domain.PosWR}
}

func baApprox(a, b float64) bool { return math.Abs(a-b) < 0.02 }

// TestBuildBreakoutAge_EarliestCrossing proves the end-to-end join: the WR's breakout is
// his EARLIEST crossing season (2020, not 2021), the RB crosses only on rushing share in
// 2021, the never-crossing TE is absent, and the crossing-but-DOB-less WR is absent.
func TestBuildBreakoutAge_EarliestCrossing(t *testing.T) {
	t.Parallel()
	stats := cfbdSeasonServer(t, "SECRET")
	cw := crosswalkFixture(t, baCrosswalkCSV)
	ages := baBirthFixture(t, baBirthCSV)

	out, err := BuildBreakoutAge(context.Background(), stats.Client(),
		stats.URL, "SECRET", cw, ages, baSeasons(),
		[]string{"1001", "1002", "1003", "1004"}, baPosLookup())
	if err != nil {
		t.Fatalf("BuildBreakoutAge: %v", err)
	}

	if len(out) != 2 {
		t.Fatalf("got %d breakout ages, want 2 (WR + RB; TE never crosses, DOB-less WR absent): %v", len(out), out)
	}
	wr, _ := playerid.New("1001")
	rb, _ := playerid.New("1003")
	te, _ := playerid.New("1002")
	nodob, _ := playerid.New("1004")
	if !baApprox(out[wr], 20.0) {
		t.Errorf("WR breakout age = %v, want ~20.0 (born 2000-09-01, earliest cross 2020)", out[wr])
	}
	if !baApprox(out[rb], 22.0) {
		t.Errorf("RB breakout age = %v, want ~22.0 (born 1999-09-01, cross 2021 on rushing)", out[rb])
	}
	if _, ok := out[te]; ok {
		t.Errorf("TE should be absent (never crosses 0.20)")
	}
	if _, ok := out[nodob]; ok {
		t.Errorf("DOB-less WR should be absent (no birthdate to derive age)")
	}
}

// TestBuildBreakoutAge_FetchFailsLoud proves a real fetch failure on ANY season surfaces
// (a breakout-less league must be visible, not silently neutral).
func TestBuildBreakoutAge_FetchFailsLoud(t *testing.T) {
	t.Parallel()
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`)) // resolves zero records → collegeshare errEmpty
	}))
	t.Cleanup(empty.Close)
	cw := crosswalkFixture(t, baCrosswalkCSV)
	ages := baBirthFixture(t, baBirthCSV)

	_, err := BuildBreakoutAge(context.Background(), empty.Client(),
		empty.URL, "SECRET", cw, ages, baSeasons(), []string{"1001"}, baPosLookup())
	if err == nil {
		t.Fatal("expected a loud error when a season's college feed resolves zero records")
	}
}

// TestBuildBreakoutAge_BadKeyFailsLoud proves a rejected bearer token fails loud.
func TestBuildBreakoutAge_BadKeyFailsLoud(t *testing.T) {
	t.Parallel()
	stats := cfbdSeasonServer(t, "SECRET")
	cw := crosswalkFixture(t, baCrosswalkCSV)
	ages := baBirthFixture(t, baBirthCSV)

	_, err := BuildBreakoutAge(context.Background(), stats.Client(),
		stats.URL, "WRONG-KEY", cw, ages, baSeasons(), []string{"1001"}, baPosLookup())
	if err == nil {
		t.Fatal("expected a loud error when the CFBD bearer token is rejected")
	}
}

// TestBuildBreakoutAge_Guards proves the nil-client, nil-pos, and empty-seasons guards.
func TestBuildBreakoutAge_Guards(t *testing.T) {
	t.Parallel()
	if _, err := BuildBreakoutAge(context.Background(), nil, "u", "k", crosswalk.Map{}, nil, baSeasons(), nil, baPosLookup()); err == nil {
		t.Error("expected error for nil *http.Client")
	}
	if _, err := BuildBreakoutAge(context.Background(), http.DefaultClient, "u", "k", crosswalk.Map{}, nil, baSeasons(), nil, nil); err == nil {
		t.Error("expected error for nil PositionLookup")
	}
	if _, err := BuildBreakoutAge(context.Background(), http.DefaultClient, "u", "k", crosswalk.Map{}, nil, nil, nil, baPosLookup()); err == nil {
		t.Error("expected error for empty seasons")
	}
}

// TestBreakoutShare_Rules pins the position → share-source mapping: WR/TE draw the
// receiving-yard share, RB the rushing-yard share, and every other position has no
// offense breakout source in v1 (QB passing is unfetched, K has none, defense is 4b).
func TestBreakoutShare_Rules(t *testing.T) {
	t.Parallel()
	rc := collegeshare.RawCollegeShare{ReceivingYardShare: 0.42, RushingYardShare: 0.11}

	rec := []domain.Position{domain.PosWR, domain.PosTE}
	for _, p := range rec {
		sel, ok := breakoutShare(p)
		if !ok || sel(rc) != 0.42 {
			t.Errorf("%s: want receiving-yard share 0.42, ok=true; got ok=%v", p, ok)
		}
	}
	if sel, ok := breakoutShare(domain.PosRB); !ok || sel(rc) != 0.11 {
		t.Errorf("RB: want rushing-yard share 0.11, ok=true; got ok=%v", ok)
	}
	for _, p := range []domain.Position{domain.PosQB, domain.PosK, domain.PosCB,
		domain.PosS, domain.PosLB, domain.PosDT, domain.PosDE, domain.PosFlag} {
		if _, ok := breakoutShare(p); ok {
			t.Errorf("%s: want no offense breakout source (ok=false), got ok=true", p)
		}
	}
}

// TestEarliestBreakoutAge_Logic proves the scan picks the earliest crossing, skips
// below-threshold and rowless seasons, and rejects a non-finite/negative derived age
// (a birthdate after the season = corrupt join, never a real breakout age).
func TestEarliestBreakoutAge_Logic(t *testing.T) {
	t.Parallel()
	byseason := map[int]map[string]collegeshare.RawCollegeShare{
		2019: {"G-9": {ReceivingYardShare: 0.10}}, // below
		2020: {"G-9": {ReceivingYardShare: 0.25}}, // FIRST crossing
		2021: {"G-9": {ReceivingYardShare: 0.90}}, // later crossing, ignored
	}
	seasons := []int{2019, 2020, 2021}
	born := time.Date(2000, time.September, 1, 0, 0, 0, 0, time.UTC)

	age, ok := earliestBreakoutAge(seasons, byseason, "G-9", born, receivingShare)
	if !ok || !baApprox(age, 20.0) {
		t.Fatalf("earliest crossing: got age=%v ok=%v, want ~20.0 (2020 season), ok=true", age, ok)
	}

	// Never crosses → absent.
	low := map[int]map[string]collegeshare.RawCollegeShare{2020: {"G-9": {ReceivingYardShare: 0.05}}}
	if _, ok := earliestBreakoutAge([]int{2020}, low, "G-9", born, receivingShare); ok {
		t.Error("below-threshold every season should be absent")
	}
	// No row for the player any season → absent.
	if _, ok := earliestBreakoutAge(seasons, byseason, "MISSING", born, receivingShare); ok {
		t.Error("a player with no college row should be absent")
	}
	// Birthdate AFTER the season (negative age) → rejected.
	future := time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, ok := earliestBreakoutAge(seasons, byseason, "G-9", future, receivingShare); ok {
		t.Error("a negative derived age (birth after season) must be rejected")
	}
}
