package assembly

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegedefense"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// CFBD long-format DEFENSIVE fixtures (the same shape internal/ingestion/collegedefense
// tests use). Georgia defense: LB espn E-1, CB espn E-2, S espn E-3, DT espn E-4. Team
// denominators are the sum of every listed defender's production. The defensive category
// carries TOT/SACKS/TFL/PD; INT lives in the separate interceptions category.
const cdDefensiveFixture = `[
  {"playerId":"E-1","player":"LB One","team":"Georgia","statType":"TOT","stat":"100"},
  {"playerId":"E-1","player":"LB One","team":"Georgia","statType":"SACKS","stat":"10"},
  {"playerId":"E-1","player":"LB One","team":"Georgia","statType":"TFL","stat":"20"},
  {"playerId":"E-1","player":"LB One","team":"Georgia","statType":"PD","stat":"3"},
  {"playerId":"E-2","player":"CB Two","team":"Georgia","statType":"TOT","stat":"40"},
  {"playerId":"E-2","player":"CB Two","team":"Georgia","statType":"TFL","stat":"2"},
  {"playerId":"E-2","player":"CB Two","team":"Georgia","statType":"PD","stat":"12"},
  {"playerId":"E-3","player":"S Three","team":"Georgia","statType":"TOT","stat":"60"},
  {"playerId":"E-3","player":"S Three","team":"Georgia","statType":"SACKS","stat":"2"},
  {"playerId":"E-3","player":"S Three","team":"Georgia","statType":"TFL","stat":"4"},
  {"playerId":"E-3","player":"S Three","team":"Georgia","statType":"PD","stat":"5"},
  {"playerId":"E-4","player":"DT Four","team":"Georgia","statType":"TOT","stat":"20"},
  {"playerId":"E-4","player":"DT Four","team":"Georgia","statType":"SACKS","stat":"5"},
  {"playerId":"E-4","player":"DT Four","team":"Georgia","statType":"TFL","stat":"8"}
]`

const cdInterceptionsFixture = `[
  {"playerId":"E-2","player":"CB Two","team":"Georgia","statType":"INT","stat":"4"},
  {"playerId":"E-3","player":"S Three","team":"Georgia","statType":"INT","stat":"3"},
  {"playerId":"E-1","player":"LB One","team":"Georgia","statType":"INT","stat":"1"}
]`

// Team denominators from the fixtures above.
const (
	cdTeamTackles = 100.0 + 40.0 + 60.0 + 20.0 // 220
	cdTeamSacks   = 10.0 + 0.0 + 2.0 + 5.0     // 17
	cdTeamTFL     = 20.0 + 2.0 + 4.0 + 8.0     // 34
	cdTeamPD      = 3.0 + 12.0 + 5.0 + 0.0     // 20
	cdTeamINT     = 4.0 + 3.0 + 1.0            // 8
)

// cfbdDefenseServer serves the right defensive fixture per category query, asserting the
// bearer token arrived (matching the shared CFBD fetcher's contract).
func cfbdDefenseServer(t *testing.T, wantKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Query().Get("category") {
		case "defensive":
			_, _ = w.Write([]byte(cdDefensiveFixture))
		case "interceptions":
			_, _ = w.Write([]byte(cdInterceptionsFixture))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// cdCrosswalkCSV maps the four fixture defenders end to end: mfl 1001..1004 → gsis
// G-1..G-4 and espn E-1..E-4 → the SAME gsis, so a rostered mfl id resolves to the
// defensive-production row keyed by that gsis.
const cdCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id
1001,G-1,E-1,
1002,G-2,E-2,
1003,G-3,E-3,
1004,G-4,E-4,
`

func cdFinite(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestBuildCollegeDefense_CollapseByPosition proves the locked position-defined combine
// end to end: LB = mean(tackle,sack,TFL), CB = mean(PD,INT), S = mean(INT,tackle),
// DT = mean(TFL,sack), all joined through the crosswalk's two bridges.
func TestBuildCollegeDefense_CollapseByPosition(t *testing.T) {
	stats := cfbdDefenseServer(t, "k")
	cw := crosswalkServer(t, cdCrosswalkCSV)

	roster := []string{"1001", "1002", "1003", "1004"}
	pos := fakePosLookup{
		"1001": domain.PosLB,
		"1002": domain.PosCB,
		"1003": domain.PosS,
		"1004": domain.PosDT,
	}

	out, err := BuildCollegeDefense(context.Background(), stats.Client(), stats.URL, cw.URL, "k", 2026, roster, pos)
	if err != nil {
		t.Fatalf("BuildCollegeDefense: %v", err)
	}
	if len(out) != 4 {
		t.Fatalf("want 4 collapsed shares, got %d: %+v", len(out), out)
	}

	id1001, _ := playerid.New("1001")
	id1002, _ := playerid.New("1002")
	id1003, _ := playerid.New("1003")
	id1004, _ := playerid.New("1004")

	wantLB := (100.0/cdTeamTackles + 10.0/cdTeamSacks + 20.0/cdTeamTFL) / 3.0
	wantCB := (12.0/cdTeamPD + 4.0/cdTeamINT) / 2.0
	wantS := (3.0/cdTeamINT + 60.0/cdTeamTackles) / 2.0
	wantDT := (8.0/cdTeamTFL + 5.0/cdTeamSacks) / 2.0

	if !cdFinite(out[id1001], wantLB) {
		t.Errorf("LB (1001): got %v, want mean(tackle,sack,TFL)=%v", out[id1001], wantLB)
	}
	if !cdFinite(out[id1002], wantCB) {
		t.Errorf("CB (1002): got %v, want mean(PD,INT)=%v", out[id1002], wantCB)
	}
	if !cdFinite(out[id1003], wantS) {
		t.Errorf("S (1003): got %v, want mean(INT,tackle)=%v", out[id1003], wantS)
	}
	if !cdFinite(out[id1004], wantDT) {
		t.Errorf("DT (1004): got %v, want mean(TFL,sack)=%v", out[id1004], wantDT)
	}
}

// TestBuildCollegeDefense_OffenseHasNoSource: an offense player (drawn from the separate
// collegeshare feed) and a kicker both drop from the map — the position simply has no
// defensive college-share source here, an ordinary skip rather than a spurious zero.
func TestBuildCollegeDefense_OffenseHasNoSource(t *testing.T) {
	stats := cfbdDefenseServer(t, "k")
	cw := crosswalkServer(t, cdCrosswalkCSV)

	roster := []string{"1001", "1002", "1003"}
	pos := fakePosLookup{
		"1001": domain.PosLB, // collapses
		"1002": domain.PosWR, // offense feed, not this one → absent
		"1003": domain.PosK,  // no college-production share → absent
	}

	out, err := BuildCollegeDefense(context.Background(), stats.Client(), stats.URL, cw.URL, "k", 2026, roster, pos)
	if err != nil {
		t.Fatalf("BuildCollegeDefense: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 collapsed share (the LB), got %d: %+v", len(out), out)
	}
	id1002, _ := playerid.New("1002")
	if _, ok := out[id1002]; ok {
		t.Error("a WR has no source in this feed and must be absent, not present-as-zero")
	}
}

// TestBuildCollegeDefense_PlayerMissesAreOrdinary: an id with no crosswalk row, one with
// no resolved position, and a malformed id all drop quietly — the feed is healthy, so
// these are player-level misses, never an error.
func TestBuildCollegeDefense_PlayerMissesAreOrdinary(t *testing.T) {
	stats := cfbdDefenseServer(t, "k")
	cw := crosswalkServer(t, cdCrosswalkCSV)

	roster := []string{"1001", "1002", "9999", "10X4"}
	pos := fakePosLookup{
		"1001": domain.PosLB, // resolves
		// 1002 present in crosswalk + feed, but no position → miss
		"9999": domain.PosCB, // no crosswalk row → miss
		"10X4": domain.PosCB, // malformed id → miss
	}

	out, err := BuildCollegeDefense(context.Background(), stats.Client(), stats.URL, cw.URL, "k", 2026, roster, pos)
	if err != nil {
		t.Fatalf("player-level misses must NOT error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 collapsed share (1001), got %d: %+v", len(out), out)
	}
}

// TestBuildCollegeDefense_FetchFailsLoud: a feed that resolves zero gsis-keyed records
// (empty CFBD arrays → the fetcher's errEmpty) surfaces as an error — a defense-share-less
// league is visible, matching BuildRAS/BuildCollegeShare's posture.
func TestBuildCollegeDefense_FetchFailsLoud(t *testing.T) {
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(empty.Close)
	cw := crosswalkServer(t, cdCrosswalkCSV)
	if _, err := BuildCollegeDefense(context.Background(), empty.Client(), empty.URL, cw.URL, "k", 2026, []string{"1001"}, fakePosLookup{"1001": domain.PosLB}); err == nil {
		t.Fatal("a zero-record college-defense feed should fail loud")
	}
}

// TestBuildCollegeDefense_BadKeyFailsLoud: a non-200 from CFBD (bad bearer token) is a
// genuine fetch failure and must error, not return empty.
func TestBuildCollegeDefense_BadKeyFailsLoud(t *testing.T) {
	stats := cfbdDefenseServer(t, "right")
	cw := crosswalkServer(t, cdCrosswalkCSV)
	if _, err := BuildCollegeDefense(context.Background(), stats.Client(), stats.URL, cw.URL, "wrong", 2026, []string{"1001"}, fakePosLookup{"1001": domain.PosLB}); err == nil {
		t.Fatal("a non-200 CFBD response should fail loud")
	}
}

// TestBuildCollegeDefense_NilGuards: a nil dependency is a wiring error, never a silent
// no-signal league.
func TestBuildCollegeDefense_NilGuards(t *testing.T) {
	if _, err := BuildCollegeDefense(context.Background(), nil, "u", "u", "k", 2026, nil, fakePosLookup{}); err == nil {
		t.Fatal("nil client should error")
	}
	if _, err := BuildCollegeDefense(context.Background(), http.DefaultClient, "u", "u", "k", 2026, nil, nil); err == nil {
		t.Fatal("nil PositionLookup should error")
	}
}

// TestCollapseCollegeDefense_Rules pins the pure locked combine directly (no network):
// each defensive position averages its named component shares, every offense position + K
// have no source, and a non-finite component is skipped.
func TestCollapseCollegeDefense_Rules(t *testing.T) {
	rc := collegedefense.RawCollegeDefense{
		TackleShare:       0.20,
		SackShare:         0.30,
		TFLShare:          0.40,
		PassDefShare:      0.50,
		InterceptionShare: 0.60,
	}

	if got, ok := collapseCollegeDefense(rc, domain.PosCB); !ok || !cdFinite(got, (0.50+0.60)/2) {
		t.Errorf("CB: got (%v,%v), want mean(PD,INT)=%v", got, ok, (0.50+0.60)/2)
	}
	if got, ok := collapseCollegeDefense(rc, domain.PosS); !ok || !cdFinite(got, (0.60+0.20)/2) {
		t.Errorf("S: got (%v,%v), want mean(INT,tackle)=%v", got, ok, (0.60+0.20)/2)
	}
	if got, ok := collapseCollegeDefense(rc, domain.PosLB); !ok || !cdFinite(got, (0.20+0.30+0.40)/3) {
		t.Errorf("LB: got (%v,%v), want mean(tackle,sack,TFL)=%v", got, ok, (0.20+0.30+0.40)/3)
	}
	if got, ok := collapseCollegeDefense(rc, domain.PosDT); !ok || !cdFinite(got, (0.40+0.30)/2) {
		t.Errorf("DT: got (%v,%v), want mean(TFL,sack)=%v", got, ok, (0.40+0.30)/2)
	}
	if got, ok := collapseCollegeDefense(rc, domain.PosDE); !ok || !cdFinite(got, (0.40+0.30)/2) {
		t.Errorf("DE: got (%v,%v), want mean(TFL,sack)=%v", got, ok, (0.40+0.30)/2)
	}
	for _, p := range []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK} {
		if _, ok := collapseCollegeDefense(rc, p); ok {
			t.Errorf("%s must have no defensive college-share source in this feed", p)
		}
	}
	nan := collegedefense.RawCollegeDefense{TFLShare: math.NaN(), SackShare: 0.3}
	if _, ok := collapseCollegeDefense(nan, domain.PosDT); ok {
		t.Error("a non-finite component must be skipped, not passed through")
	}
}
