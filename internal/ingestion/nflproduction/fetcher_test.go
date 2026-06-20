package nflproduction

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, status int, body string) (map[string]RawProduction, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL)
}

// Header includes fantasy_points + fantasy_points_ppr (the leak columns) and an
// extra unbound column, in a deliberately mixed order — to prove (a) by-name
// binding survives order and (b) the leak columns are simply never read.
const header = "player_id,season_type,fantasy_points,games,completions,attempts," +
	"passing_yards,passing_tds,interceptions,carries,rushing_yards,rushing_tds," +
	"receptions,targets,receiving_yards,receiving_tds,fantasy_points_ppr,team\n"

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	body := header +
		// a QB REG line; fantasy_points 380.5 / ppr present but must be ignored
		"00-0011000,REG,380.5,17,400,600,4500.5,35,8,20,60,1,0,0,0,0,401.2,KC\n" +
		// a WR REG line, mostly receiving
		"00-0022000,REG,210,16,0,0,0,0,0,5,40,0,90,130,1200.7,9,255,CIN\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d records, want 2", len(m))
	}

	qb, ok := m["00-0011000"]
	if !ok {
		t.Fatal("QB row missing")
	}
	if qb.PassingYards != 4500.5 || qb.PassingTDs != 35 || qb.Interceptions != 8 || qb.Games != 17 {
		t.Errorf("QB parse wrong: %+v", qb)
	}
	wr := m["00-0022000"]
	if wr.Receptions != 90 || wr.Targets != 130 || wr.ReceivingYards != 1200.7 || wr.ReceivingTDs != 9 {
		t.Errorf("WR parse wrong: %+v", wr)
	}
}

// Zero-leak structural guarantee: RawProduction has no field that can hold fantasy
// points, so even with the leak columns populated the output cannot reflect them.
// (Compile-time: there is simply no field to assert against — this test documents
// the guarantee and that fantasy-bearing rows still parse cleanly.)
func TestFetch_FantasyColumnsIgnored(t *testing.T) {
	t.Parallel()
	body := header + "00-0011000,REG,999.9,1,0,0,0,0,0,0,0,0,0,0,0,0,999.9,KC\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := m["00-0011000"]; !ok {
		t.Fatal("row with fantasy points should still parse")
	}
}

func TestFetch_NonRegSkipped(t *testing.T) {
	t.Parallel()
	body := header +
		"00-0011000,REG,10,1,0,0,0,0,0,0,0,0,0,0,0,0,10,KC\n" +
		"00-0011000,POST,5,1,0,0,0,0,0,0,0,0,0,0,0,0,5,KC\n" + // same gsis, POST — skipped, no dup error
		"00-0033000,POST,5,1,0,0,0,0,0,0,0,0,0,0,0,0,5,BUF\n" // POST-only player — skipped
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (only the REG row)", len(m))
	}
	if _, ok := m["00-0033000"]; ok {
		t.Error("POST-only player must not appear")
	}
}

func TestFetch_MissingCellIsZero(t *testing.T) {
	t.Parallel()
	// targets and receiving_yards empty, interceptions "NA" — all legitimate zeros.
	body := header + "00-0011000,REG,10,1,0,0,0,0,NA,0,0,0,0,,,0,10,KC\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	r := m["00-0011000"]
	if r.Targets != 0 || r.ReceivingYards != 0 || r.Interceptions != 0 {
		t.Errorf("missing/NA cells should be 0: %+v", r)
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"missing required column", "player_id,season_type,games\n00-0011000,REG,17\n"},
		{"malformed numeric", header + "00-0011000,REG,10,not-a-number,0,0,0,0,0,0,0,0,0,0,0,0,10,KC\n"},
		{"malformed yardage", header + "00-0011000,REG,10,1,0,0,bad,0,0,0,0,0,0,0,0,0,10,KC\n"},
		{"duplicate REG row", header +
			"00-0011000,REG,10,1,0,0,0,0,0,0,0,0,0,0,0,0,10,KC\n" +
			"00-0011000,REG,11,1,0,0,0,0,0,0,0,0,0,0,0,0,11,KC\n"},
		{"empty (header only)", header},
		{"ragged row", header + "00-0011000,REG,10\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := serve(t, http.StatusOK, tc.body); err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

func TestFetch_EmptyIsSentinel(t *testing.T) {
	t.Parallel()
	// A header with only POST rows resolves zero REG records -> errEmpty.
	body := header + "00-0011000,POST,5,1,0,0,0,0,0,0,0,0,0,0,0,0,5,KC\n"
	_, err := serve(t, http.StatusOK, body)
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty", err)
	}
}

func TestFetch_Non200(t *testing.T) {
	t.Parallel()
	if _, err := serve(t, http.StatusInternalServerError, "boom"); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
