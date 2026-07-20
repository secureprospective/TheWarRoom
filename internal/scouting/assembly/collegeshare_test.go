package assembly

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// CFBD long-format fixtures (the same shape internal/ingestion/collegeshare tests
// use). Georgia: WR espn E-1 owns 80% of receiving yards; RB espn E-3 owns the
// rushing (1450 of 1500) plus a sliver of receiving. TE espn E-2 is a Georgia
// receiver too. Every teammate's production counts toward the team denominator.
const csReceivingFixture = `[
  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"REC","stat":"60"},
  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"YDS","stat":"800"},
  {"playerId":"E-2","player":"TE Two","team":"Georgia","statType":"REC","stat":"40"},
  {"playerId":"E-2","player":"TE Two","team":"Georgia","statType":"YDS","stat":"150"},
  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"REC","stat":"20"},
  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"YDS","stat":"50"}
]`

const csRushingFixture = `[
  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"CAR","stat":"250"},
  {"playerId":"E-3","player":"RB Three","team":"Georgia","statType":"YDS","stat":"1450"},
  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"CAR","stat":"5"},
  {"playerId":"E-1","player":"WR One","team":"Georgia","statType":"YDS","stat":"50"}
]`

// cfbdStatsServer serves the right college-production fixture per category query,
// asserting the bearer token arrived (matching the fetcher's contract).
func cfbdStatsServer(t *testing.T, wantKey string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Query().Get("category") {
		case "receiving":
			_, _ = w.Write([]byte(csReceivingFixture))
		case "rushing":
			_, _ = w.Write([]byte(csRushingFixture))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// crosswalkServer serves a db_playerids CSV closing BOTH bridges the join needs:
// mfl_id → gsis (Lookup) and espn_id → gsis (GSISForESPN). Each row carries the
// player's mfl id, gsis, and CFBD/espn playerId.
func crosswalkServer(t *testing.T, csv string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(csv))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// crosswalkFixture stands up the crosswalk fixture server and returns the RESOLVED Map.
// The app now fetches the crosswalk once and threads the Map down into every assembler, so
// the assembler tests mirror that by fetching here and passing the Map, not a URL.
func crosswalkFixture(t *testing.T, csv string) crosswalk.Map {
	t.Helper()
	srv := crosswalkServer(t, csv)
	cw, err := crosswalk.Fetch(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatalf("crosswalk fixture fetch: %v", err)
	}
	return cw
}

// csCrosswalkCSV maps the three fixture players end to end: mfl 1001/1002/1003 →
// gsis G-1/G-2/G-3 and espn E-1/E-2/E-3 → the SAME gsis, so a rostered mfl id
// resolves to the college-production row keyed by that gsis.
const csCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id
1001,G-1,E-1,
1002,G-2,E-2,
1003,G-3,E-3,
`

func csFinite(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestBuildCollegeShare_CollapseByPosition proves the position-defined v1 collapse
// end to end: WR/TE take receiving-yard share, RB takes the 70/30 rush/receiving
// blend, all joined through the crosswalk's two bridges.
func TestBuildCollegeShare_CollapseByPosition(t *testing.T) {
	stats := cfbdStatsServer(t, "k")
	cw := crosswalkFixture(t, csCrosswalkCSV)

	roster := []string{"1001", "1002", "1003"}
	pos := fakePosLookup{
		"1001": domain.PosWR,
		"1002": domain.PosTE,
		"1003": domain.PosRB,
	}

	out, err := BuildCollegeShare(context.Background(), stats.Client(), stats.URL, "k", cw, 2026, roster, pos)
	if err != nil {
		t.Fatalf("BuildCollegeShare: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3 collapsed shares, got %d: %+v", len(out), out)
	}

	// Georgia receiving-yard totals: 800 + 150 + 50 = 1000. Rushing-yard totals:
	// 1450 + 50 = 1500.
	wrYardShare := 800.0 / 1000.0 // 0.80
	teYardShare := 150.0 / 1000.0 // 0.15
	rbYardShare := 50.0 / 1000.0  // 0.05
	rbRushShare := 1450.0 / 1500.0

	id1001, _ := playerid.New("1001")
	id1002, _ := playerid.New("1002")
	id1003, _ := playerid.New("1003")

	if !csFinite(out[id1001], wrYardShare) {
		t.Errorf("WR (1001): got %v, want ReceivingYardShare %v", out[id1001], wrYardShare)
	}
	if !csFinite(out[id1002], teYardShare) {
		t.Errorf("TE (1002): got %v, want ReceivingYardShare %v", out[id1002], teYardShare)
	}
	wantRB := 0.70*rbRushShare + 0.30*rbYardShare
	if !csFinite(out[id1003], wantRB) {
		t.Errorf("RB (1003): got %v, want 0.70·rush(%v)+0.30·rec(%v)=%v", out[id1003], rbRushShare, rbYardShare, wantRB)
	}
}

// TestBuildCollegeShare_QBAndDefenseHaveNoSource: a QB (no passing share in this
// feed) and a defensive player (drawn from the separate collegedefense feed) both
// drop from the map — the position simply has no offense college-share source here,
// an ordinary skip rather than a spurious zero.
func TestBuildCollegeShare_QBAndDefenseHaveNoSource(t *testing.T) {
	stats := cfbdStatsServer(t, "k")
	cw := crosswalkFixture(t, csCrosswalkCSV)

	roster := []string{"1001", "1002", "1003"}
	pos := fakePosLookup{
		"1001": domain.PosWR, // collapses
		"1002": domain.PosQB, // no passing share → absent
		"1003": domain.PosLB, // IDP feed, not this one → absent
	}

	out, err := BuildCollegeShare(context.Background(), stats.Client(), stats.URL, "k", cw, 2026, roster, pos)
	if err != nil {
		t.Fatalf("BuildCollegeShare: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 collapsed share (the WR), got %d: %+v", len(out), out)
	}
	id1001, _ := playerid.New("1001")
	if _, ok := out[id1001]; !ok {
		t.Fatal("the WR must be present")
	}
	id1002, _ := playerid.New("1002")
	if _, ok := out[id1002]; ok {
		t.Error("a QB has no source in this feed and must be absent, not present-as-zero")
	}
}

// TestBuildCollegeShare_PlayerMissesAreOrdinary: an id with no crosswalk row, one
// with no college-production row, and one with no resolved position all drop quietly
// — the feed is healthy, so these are player-level misses, never an error.
func TestBuildCollegeShare_PlayerMissesAreOrdinary(t *testing.T) {
	stats := cfbdStatsServer(t, "k")
	cw := crosswalkFixture(t, csCrosswalkCSV)

	roster := []string{"1001", "1002", "9999", "10X4"}
	pos := fakePosLookup{
		"1001": domain.PosWR, // resolves
		// 1002 present in crosswalk + feed, but no position → miss
		"9999": domain.PosWR, // no crosswalk row → miss
		"10X4": domain.PosWR, // malformed id → miss
	}

	out, err := BuildCollegeShare(context.Background(), stats.Client(), stats.URL, "k", cw, 2026, roster, pos)
	if err != nil {
		t.Fatalf("player-level misses must NOT error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 collapsed share (1001), got %d: %+v", len(out), out)
	}
}

// TestBuildCollegeShare_FetchFailsLoud: a feed that resolves zero gsis-keyed records
// (empty CFBD arrays → the fetcher's errEmpty) surfaces as an error — a college-
// share-less league is visible, matching BuildRAS's posture.
func TestBuildCollegeShare_FetchFailsLoud(t *testing.T) {
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(empty.Close)
	cw := crosswalkFixture(t, csCrosswalkCSV)
	if _, err := BuildCollegeShare(context.Background(), empty.Client(), empty.URL, "k", cw, 2026, []string{"1001"}, fakePosLookup{"1001": domain.PosWR}); err == nil {
		t.Fatal("a zero-record college feed should fail loud")
	}
}

// TestBuildCollegeShare_BadKeyFailsLoud: a non-200 from CFBD (bad bearer token) is a
// genuine fetch failure and must error, not return empty.
func TestBuildCollegeShare_BadKeyFailsLoud(t *testing.T) {
	stats := cfbdStatsServer(t, "right")
	cw := crosswalkFixture(t, csCrosswalkCSV)
	if _, err := BuildCollegeShare(context.Background(), stats.Client(), stats.URL, "wrong", cw, 2026, []string{"1001"}, fakePosLookup{"1001": domain.PosWR}); err == nil {
		t.Fatal("a non-200 CFBD response should fail loud")
	}
}

// TestBuildCollegeShare_NilGuards: a nil dependency is a wiring error, never a silent
// no-signal league.
func TestBuildCollegeShare_NilGuards(t *testing.T) {
	if _, err := BuildCollegeShare(context.Background(), nil, "u", "k", crosswalk.Map{}, 2026, nil, fakePosLookup{}); err == nil {
		t.Fatal("nil client should error")
	}
	if _, err := BuildCollegeShare(context.Background(), http.DefaultClient, "u", "k", crosswalk.Map{}, 2026, nil, nil); err == nil {
		t.Fatal("nil PositionLookup should error")
	}
}

// TestCollapseCollegeShare_Rules pins the pure v1 collapse rule directly (no
// network): WR/TE = receiving-yard share, RB = 70/30 blend, QB + every defensive
// position have no source, and a non-finite raw share is skipped.
func TestCollapseCollegeShare_Rules(t *testing.T) {
	rc := collegeshare.RawCollegeShare{
		ReceptionShare:     0.50,
		ReceivingYardShare: 0.40,
		RushingYardShare:   0.60,
	}

	if got, ok := collapseCollegeShare(rc, domain.PosWR); !ok || !csFinite(got, 0.40) {
		t.Errorf("WR: got (%v,%v), want (0.40,true)", got, ok)
	}
	if got, ok := collapseCollegeShare(rc, domain.PosTE); !ok || !csFinite(got, 0.40) {
		t.Errorf("TE: got (%v,%v), want (0.40,true)", got, ok)
	}
	if got, ok := collapseCollegeShare(rc, domain.PosRB); !ok || !csFinite(got, 0.70*0.60+0.30*0.40) {
		t.Errorf("RB: got (%v,%v), want (%v,true)", got, ok, 0.70*0.60+0.30*0.40)
	}
	for _, p := range []domain.Position{domain.PosQB, domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS, domain.PosK} {
		if _, ok := collapseCollegeShare(rc, p); ok {
			t.Errorf("%s must have no offense college-share source in this feed", p)
		}
	}
	nan := collegeshare.RawCollegeShare{ReceivingYardShare: math.NaN()}
	if _, ok := collapseCollegeShare(nan, domain.PosWR); ok {
		t.Error("a non-finite raw share must be skipped, not passed through")
	}
}
