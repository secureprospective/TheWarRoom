package kicking

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, status int, body string) (map[string]RawKicking, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL)
}

// headerCols is a deliberately mixed column order that interleaves the rate columns
// (fg_pct, pat_pct) the fetcher must ignore and ends with a quoted, comma-bearing
// fg_made_list column — proving (a) by-name binding survives order and unbound
// columns, and (b) CSV quoting on an unbound column does not derail parsing.
func headerCols() []string {
	return []string{
		"player_id", "season_type", "games", "fg_made", "fg_att", "fg_pct",
		"fg_missed", "fg_blocked", "fg_long",
		"fg_made_0_19", "fg_made_20_29", "fg_made_30_39", "fg_made_40_49", "fg_made_50_59", "fg_made_60_",
		"fg_missed_0_19", "fg_missed_20_29", "fg_missed_30_39", "fg_missed_40_49", "fg_missed_50_59", "fg_missed_60_",
		"pat_made", "pat_att", "pat_pct", "pat_missed", "pat_blocked", "fg_made_list",
	}
}

func header() string { return strings.Join(headerCols(), ",") + "\n" }

// row builds one CSV data row: every column defaults to "0", fg_made_list defaults to
// a quoted comma-bearing value (must be ignored), and overrides set specific cells.
func row(overrides map[string]string) string {
	cols := headerCols()
	vals := make([]string, len(cols))
	for i, c := range cols {
		switch {
		case overrides[c] != "":
			vals[i] = overrides[c]
		case c == "fg_made_list":
			vals[i] = `"21,35,48"` // quoted, embedded commas — unbound, must be ignored
		default:
			vals[i] = "0"
		}
	}
	return strings.Join(vals, ",") + "\n"
}

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	body := header() +
		row(map[string]string{
			"player_id": "00-0011000", "season_type": "REG", "games": "17",
			"fg_made": "30", "fg_att": "34", "fg_missed": "4", "fg_blocked": "0", "fg_long": "58",
			"fg_made_0_19": "1", "fg_made_20_29": "8", "fg_made_30_39": "10",
			"fg_made_40_49": "7", "fg_made_50_59": "3", "fg_made_60_": "1",
			"fg_missed_40_49": "2", "fg_missed_50_59": "2",
			"pat_made": "40", "pat_att": "41", "pat_missed": "1", "pat_blocked": "0",
			"fg_pct": "0.882", "pat_pct": "0.976", // rate columns must be ignored
		}) +
		row(map[string]string{
			"player_id": "00-0022000", "season_type": "REG", "games": "16",
			"fg_made": "20", "fg_att": "25", "fg_missed": "5", "fg_long": "61",
			"fg_made_60_": "1", "pat_made": "30", "pat_att": "30",
		})
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d records, want 2", len(m))
	}

	k, ok := m["00-0011000"]
	if !ok {
		t.Fatal("kicker row missing")
	}
	if k.Games != 17 || k.FGMade != 30 || k.FGAtt != 34 || k.FGMissed != 4 || k.FGLong != 58 {
		t.Errorf("scalar parse wrong: %+v", k)
	}
	wantMade := [numBuckets]int{1, 8, 10, 7, 3, 1}
	if k.FGMadeByDistance != wantMade {
		t.Errorf("FGMadeByDistance = %v, want %v", k.FGMadeByDistance, wantMade)
	}
	wantMissed := [numBuckets]int{0, 0, 0, 2, 2, 0}
	if k.FGMissedByDistance != wantMissed {
		t.Errorf("FGMissedByDistance = %v, want %v", k.FGMissedByDistance, wantMissed)
	}
	if k.PATMade != 40 || k.PATAtt != 41 || k.PATMissed != 1 {
		t.Errorf("PAT parse wrong: %+v", k)
	}
	if m["00-0022000"].FGMadeByDistance[5] != 1 {
		t.Errorf("second kicker 60+ bucket wrong: %+v", m["00-0022000"])
	}
}

// A traded kicker has a per-team REG row for each team and no source aggregate row;
// the fetcher sums them into one season record (counts + buckets add, fg_long = max).
func TestFetch_TradedKickerSummed(t *testing.T) {
	t.Parallel()
	body := header() +
		row(map[string]string{
			"player_id": "00-0034450", "season_type": "REG", "team": "NYG", "games": "6",
			"fg_made": "13", "fg_att": "16", "fg_long": "54", "fg_made_30_39": "5", "pat_made": "10",
		}) +
		row(map[string]string{
			"player_id": "00-0034450", "season_type": "REG", "team": "NYJ", "games": "1",
			"fg_made": "1", "fg_att": "1", "fg_long": "33", "fg_made_30_39": "1", "pat_made": "2",
		}) +
		row(map[string]string{
			"player_id": "00-0034450", "season_type": "REG", "team": "WAS", "games": "1",
			"fg_made": "2", "fg_att": "3", "fg_long": "58", "fg_made_50_59": "1", "pat_made": "1",
		})
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d records, want 1 (summed)", len(m))
	}
	k := m["00-0034450"]
	if k.Games != 8 || k.FGMade != 16 || k.FGAtt != 20 || k.PATMade != 13 {
		t.Errorf("sums wrong: %+v", k)
	}
	if k.FGLong != 58 { // max across teams, not last-write
		t.Errorf("FGLong = %d, want 58 (max)", k.FGLong)
	}
	if k.FGMadeByDistance[2] != 6 || k.FGMadeByDistance[4] != 1 { // 30-39 bucket summed; 50-59 bucket
		t.Errorf("distance buckets not summed: %+v", k.FGMadeByDistance)
	}
}

// Zero-leak / rate-ignored: RawKicking has no field for fg_pct/pat_pct, so even a
// populated rate column cannot reflect into the output — the row still parses cleanly.
func TestFetch_RateColumnsIgnored(t *testing.T) {
	t.Parallel()
	body := header() + row(map[string]string{
		"player_id": "00-0011000", "season_type": "REG", "fg_pct": "1.000", "pat_pct": "1.000",
	})
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if _, ok := m["00-0011000"]; !ok {
		t.Fatal("row with rate columns should still parse")
	}
}

func TestFetch_NonRegSkipped(t *testing.T) {
	t.Parallel()
	body := header() +
		row(map[string]string{"player_id": "00-0011000", "season_type": "REG", "fg_made": "10"}) +
		row(map[string]string{"player_id": "00-0011000", "season_type": "POST", "fg_made": "3"}) + // same gsis, POST — skipped, no dup error
		row(map[string]string{"player_id": "00-0033000", "season_type": "POST", "fg_made": "2"}) // POST-only kicker — skipped
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (only the REG row)", len(m))
	}
	if _, ok := m["00-0033000"]; ok {
		t.Error("POST-only kicker must not appear")
	}
}

func TestFetch_MissingCellIsZero(t *testing.T) {
	t.Parallel()
	// fg_made empty, fg_blocked "NA", a bucket empty — all legitimate zeros.
	body := header() + row(map[string]string{
		"player_id": "00-0011000", "season_type": "REG",
		"fg_made": "", "fg_blocked": "NA", "fg_made_30_39": "",
	})
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	k := m["00-0011000"]
	if k.FGMade != 0 || k.FGBlocked != 0 || k.FGMadeByDistance[2] != 0 {
		t.Errorf("missing/NA cells should be 0: %+v", k)
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"missing required column", "player_id,season_type,games\n00-0011000,REG,17\n"},
		{"malformed scalar", header() + row(map[string]string{"player_id": "00-0011000", "season_type": "REG", "fg_made": "not-a-number"})},
		{"malformed bucket", header() + row(map[string]string{"player_id": "00-0011000", "season_type": "REG", "fg_made_40_49": "bad"})},
		{"empty (header only)", header()},
		{"ragged row", header() + "00-0011000,REG,17\n"},
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
	body := header() + row(map[string]string{"player_id": "00-0011000", "season_type": "POST"})
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
