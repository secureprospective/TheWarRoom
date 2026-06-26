package collegedefense

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Long-format CFBD fixtures. Georgia LB (espn 1) owns 50% tackles, 60% sacks, 50% TFL;
// CB (espn 2) owns the PD/INT. Player 9 has production but is unresolvable (college-only)
// -> skipped. Sacks/TFL arrive fractional. SOLO/QB HUR/TD rows are present to prove they
// are ignored.
const defensiveFixture = `[
  {"playerId":"1","player":"A Backer","team":"Georgia","statType":"TOT","stat":"100"},
  {"playerId":"1","player":"A Backer","team":"Georgia","statType":"SACKS","stat":"6.0"},
  {"playerId":"1","player":"A Backer","team":"Georgia","statType":"TFL","stat":"10.0"},
  {"playerId":"1","player":"A Backer","team":"Georgia","statType":"SOLO","stat":"55"},
  {"playerId":"1","player":"A Backer","team":"Georgia","statType":"QB HUR","stat":"8"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"TOT","stat":"40"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"SACKS","stat":"0.0"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"TFL","stat":"2.0"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"PD","stat":"15"},
  {"playerId":"3","player":"C Other","team":"Georgia","statType":"TOT","stat":"60"},
  {"playerId":"3","player":"C Other","team":"Georgia","statType":"SACKS","stat":"4.0"},
  {"playerId":"3","player":"C Other","team":"Georgia","statType":"TFL","stat":"8.0"},
  {"playerId":"3","player":"C Other","team":"Georgia","statType":"PD","stat":"5"},
  {"playerId":"9","player":"Undrafted","team":"Georgia","statType":"TOT","stat":"20"}
]`

const interceptionsFixture = `[
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"INT","stat":"6"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"YDS","stat":"120"},
  {"playerId":"2","player":"B Corner","team":"Georgia","statType":"TD","stat":"1"},
  {"playerId":"3","player":"C Other","team":"Georgia","statType":"INT","stat":"4"}
]`

func serveStats(t *testing.T, wantKey string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Query().Get("category") {
		case categoryDefensive:
			_, _ = w.Write([]byte(defensiveFixture))
		case categoryInterceptions:
			_, _ = w.Write([]byte(interceptionsFixture))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func fakeResolver(espn string) (string, bool) {
	m := map[string]string{"1": "00-0000001", "2": "00-0000002", "3": "00-0000003"}
	g, ok := m[espn]
	return g, ok // espn "9" intentionally absent (undrafted, no gsis)
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestFetch_SharesCountsAndKeying(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "k")
	out, err := Fetch(context.Background(), client, url, "k", 2023, fakeResolver)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Three resolvable players (1,2,3); espn 9 unresolved and skipped.
	if len(out) != 3 {
		t.Fatalf("emitted %d records, want 3 (unresolved espn 9 skipped)", len(out))
	}

	// Team Georgia totals (resolvable + unresolved both count toward the denominator,
	// since the share is of the whole team's production): tackles 100+40+60+20=220,
	// sacks 6+0+4=10, TFL 10+2+8=20, PD 15+5=20, INT 6+4=10.
	lb := out["00-0000001"]
	if lb.Tackles != 100 || !approx(lb.Sacks, 6.0) || !approx(lb.TacklesForLoss, 10.0) {
		t.Errorf("LB raw counts wrong: %+v", lb)
	}
	if !approx(lb.TackleShare, 100.0/220.0) {
		t.Errorf("LB TackleShare = %v, want %v", lb.TackleShare, 100.0/220.0)
	}
	if !approx(lb.SackShare, 6.0/10.0) || !approx(lb.TFLShare, 10.0/20.0) {
		t.Errorf("LB sack/TFL share wrong: %+v", lb)
	}

	cb := out["00-0000002"]
	if cb.PassesDefended != 15 || cb.Interceptions != 6 {
		t.Errorf("CB raw counts wrong: %+v", cb)
	}
	if !approx(cb.PassDefShare, 15.0/20.0) || !approx(cb.InterceptionShare, 6.0/10.0) {
		t.Errorf("CB PD/INT share wrong: %+v", cb)
	}
}

func TestFetch_UnresolvableSkipped(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "k")
	out, err := Fetch(context.Background(), client, url, "k", 2023, fakeResolver)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	for gsis := range out {
		if gsis == "" {
			t.Fatal("empty gsis emitted")
		}
	}
	// espn 9 produced tackles but has no gsis -> never appears.
	for _, rc := range out {
		if rc.ESPNID == "9" {
			t.Fatal("unresolved espn 9 leaked into output")
		}
	}
}

func TestFetch_PoisonGsisDropped(t *testing.T) {
	t.Parallel()
	// Two distinct espn ids resolving to one gsis -> both dropped.
	poison := func(espn string) (string, bool) {
		switch espn {
		case "1", "2", "3":
			return "00-0000099", true // all collide
		}
		return "", false
	}
	client, url := serveStats(t, "k")
	_, err := Fetch(context.Background(), client, url, "k", 2023, poison)
	if err == nil {
		t.Fatal("expected errEmpty: the only gsis was poisoned by colliding espn ids")
	}
}

func TestFetch_MalformedStatFailsLoud(t *testing.T) {
	t.Parallel()
	bad := `[{"playerId":"1","player":"X","team":"Georgia","statType":"TOT","stat":"oops"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(bad))
	}))
	t.Cleanup(srv.Close)
	_, err := Fetch(context.Background(), srv.Client(), srv.URL, "k", 2023, fakeResolver)
	if err == nil {
		t.Fatal("expected a malformed-stat error on TOT=oops")
	}
}

func TestFetch_EmptyResolvesNothing(t *testing.T) {
	t.Parallel()
	none := func(string) (string, bool) { return "", false }
	client, url := serveStats(t, "k")
	_, err := Fetch(context.Background(), client, url, "k", 2023, none)
	if err == nil {
		t.Fatal("expected errEmpty when the resolver bridges nothing")
	}
}

func TestFetch_BadKeyFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "right")
	_, err := Fetch(context.Background(), client, url, "wrong", 2023, fakeResolver)
	if err == nil {
		t.Fatal("expected an auth error with the wrong bearer key")
	}
}
