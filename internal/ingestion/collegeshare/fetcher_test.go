package collegeshare

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Long-format CFBD fixtures. Georgia: WR A (espn 1) owns 60% of receptions and 80% of
// receiving yards; RB C (espn 3) owns the rushing. Alabama: WR D (espn 4). Player A
// also has a few carries, so a player spans both categories. Player 9 has production
// but the resolver can't bridge it (college-only / undrafted) -> skipped.
const receivingFixture = `[
  {"playerId":"1","player":"A Receiver","team":"Georgia","statType":"REC","stat":"60"},
  {"playerId":"1","player":"A Receiver","team":"Georgia","statType":"YDS","stat":"800"},
  {"playerId":"1","player":"A Receiver","team":"Georgia","statType":"TD","stat":"9"},
  {"playerId":"2","player":"B Receiver","team":"Georgia","statType":"REC","stat":"40"},
  {"playerId":"2","player":"B Receiver","team":"Georgia","statType":"YDS","stat":"200"},
  {"playerId":"9","player":"Undrafted Guy","team":"Georgia","statType":"REC","stat":"5"},
  {"playerId":"4","player":"D Receiver","team":"Alabama","statType":"REC","stat":"50"},
  {"playerId":"4","player":"D Receiver","team":"Alabama","statType":"YDS","stat":"700"}
]`

const rushingFixture = `[
  {"playerId":"3","player":"C Back","team":"Georgia","statType":"CAR","stat":"200"},
  {"playerId":"3","player":"C Back","team":"Georgia","statType":"YDS","stat":"1450"},
  {"playerId":"1","player":"A Receiver","team":"Georgia","statType":"CAR","stat":"10"},
  {"playerId":"1","player":"A Receiver","team":"Georgia","statType":"YDS","stat":"50"}
]`

// serveStats returns a client + base URL serving the right fixture per category query,
// asserting the bearer token arrived.
func serveStats(t *testing.T, wantKey string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Query().Get("category") {
		case categoryReceiving:
			_, _ = w.Write([]byte(receivingFixture))
		case categoryRushing:
			_, _ = w.Write([]byte(rushingFixture))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

// fakeResolver bridges the fixture espn ids to test gsis; espn "9" is intentionally
// absent (an undrafted college-only player with no gsis).
func fakeResolver(espn string) (string, bool) {
	m := map[string]string{"1": "00-0000001", "2": "00-0000002", "3": "00-0000003", "4": "00-0000004"}
	g, ok := m[espn]
	return g, ok
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestFetch_SharesAndKeying(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "k")
	out, err := Fetch(context.Background(), client, url, "k", 2023, fakeResolver)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// Four resolvable players (1,2,3,4); espn 9 is unresolved and skipped.
	if len(out) != 4 {
		t.Fatalf("emitted %d records, want 4 (the unresolved espn 9 must be skipped)", len(out))
	}
	if _, ok := out["00-0000009"]; ok {
		t.Error("an unresolved espn id must not appear in the output")
	}

	a := out["00-0000001"]
	// Georgia receiving totals include EVERY receiver, even the unresolved espn 9 (his
	// 5 catches still count toward the team denominator — market share is share of the
	// team's real production): REC 60+40+5=105, YDS 800+200+0=1000. So A owns 60/105.
	if !approx(a.ReceptionShare, 60.0/105.0) {
		t.Errorf("A ReceptionShare = %v, want %v (denominator includes the unresolved teammate)", a.ReceptionShare, 60.0/105.0)
	}
	if !approx(a.ReceivingYardShare, 0.80) {
		t.Errorf("A ReceivingYardShare = %v, want 0.80", a.ReceivingYardShare)
	}
	// Georgia rushing yards: C 1450 + A 50 = 1500; A owns 50/1500.
	if !approx(a.RushingYardShare, 50.0/1500.0) {
		t.Errorf("A RushingYardShare = %v, want %v", a.RushingYardShare, 50.0/1500.0)
	}
	// Raw counts carried through.
	if a.Receptions != 60 || a.ReceivingYards != 800 || a.RushingCarries != 10 {
		t.Errorf("A raw counts = {rec %d, recYds %v, car %d}, want {60, 800, 10}", a.Receptions, a.ReceivingYards, a.RushingCarries)
	}
	if a.Season != 2023 || a.ESPNID != "1" {
		t.Errorf("A meta = {season %d, espn %q}, want {2023, 1}", a.Season, a.ESPNID)
	}

	// A pure rusher (C) has a 0 reception share, not a divide-by-zero or a spurious value.
	if c := out["00-0000003"]; c.ReceptionShare != 0 || !approx(c.RushingYardShare, 1450.0/1500.0) {
		t.Errorf("C shares = {rec %v, rushYd %v}, want {0, %v}", c.ReceptionShare, c.RushingYardShare, 1450.0/1500.0)
	}

	// Alabama is summed independently of Georgia: D owns 100% of Alabama receiving.
	if d := out["00-0000004"]; !approx(d.ReceptionShare, 1.0) || !approx(d.ReceivingYardShare, 1.0) {
		t.Errorf("D Alabama shares = {rec %v, yd %v}, want {1, 1}", d.ReceptionShare, d.ReceivingYardShare)
	}
}

// Two distinct CFBD players resolving to the same gsis is ambiguous (rare many-to-one
// bridge) and the gsis is dropped from the output, not silently mis-attributed.
func TestFetch_DuplicateGSISDropped(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "k")
	// Resolve espn 1 AND espn 2 to the same gsis -> that gsis is poisoned/dropped.
	collide := func(espn string) (string, bool) {
		if espn == "1" || espn == "2" {
			return "00-0000001", true
		}
		return fakeResolver(espn)
	}
	out, err := Fetch(context.Background(), client, url, "k", 2023, collide)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := out["00-0000001"]; ok {
		t.Error("a gsis two distinct players resolve to must be dropped, not mis-attributed")
	}
	// The unrelated players (C=3, D=4) still resolve.
	if len(out) != 2 {
		t.Errorf("emitted %d, want 2 (only the colliding gsis dropped)", len(out))
	}
}

func TestFetch_MalformedStatFailsLoud(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("category") == categoryReceiving {
			_, _ = w.Write([]byte(`[{"playerId":"1","player":"X","team":"Georgia","statType":"REC","stat":"not-a-number"}]`))
			return
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	if _, err := Fetch(context.Background(), srv.Client(), srv.URL, "k", 2023, fakeResolver); err == nil {
		t.Fatal("expected a fail-loud error on a present-but-unparseable stat")
	}
}

func TestFetch_BadKeyFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveStats(t, "right")
	if _, err := Fetch(context.Background(), client, url, "wrong", 2023, fakeResolver); err == nil {
		t.Fatal("expected a non-200 error on a bad bearer token")
	}
}

func TestFetch_EmptyIsSentinel(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	if _, err := Fetch(context.Background(), srv.Client(), srv.URL, "k", 2023, fakeResolver); err == nil {
		t.Fatal("expected errEmpty when no record resolves")
	}
}
