package agetrajectory

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func serve(t *testing.T, status int, body string) (map[string]RawAge, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL)
}

// Header carries the two bound columns among unbound ones in a mixed order — to
// prove by-name binding survives column order and that other columns are not read.
const header = "gsis_id,display_name,position,birth_date,status\n"

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	body := header +
		"00-0011000,Joe Back,RB,1998-03-14,ACT\n" +
		"00-0022000,Sam Wideout,WR,2000-11-02,ACT\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d records, want 2", len(m))
	}
	want := time.Date(1998, time.March, 14, 0, 0, 0, 0, time.UTC)
	if got := m["00-0011000"].BirthDate; !got.Equal(want) {
		t.Errorf("birth date = %v, want %v", got, want)
	}
}

func TestFetch_MissingFieldsSkipped(t *testing.T) {
	t.Parallel()
	body := header +
		"00-0011000,Joe Back,RB,1998-03-14,ACT\n" +
		"00-0022000,No DOB,WR,,ACT\n" + // empty birth_date
		"00-0033000,NA DOB,TE,NA,ACT\n" + // NA birth_date
		",No GSIS,QB,1995-01-01,ACT\n" // empty gsis
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (rows missing gsis or DOB are skipped)", len(m))
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"missing required column", "gsis_id,display_name\n00-0011000,Joe\n"},
		{"malformed birth_date", header + "00-0011000,Joe,RB,03/14/1998,ACT\n"},
		{"impossible date", header + "00-0011000,Joe,RB,1998-13-40,ACT\n"},
		{"duplicate gsis", header +
			"00-0011000,Joe,RB,1998-03-14,ACT\n" +
			"00-0011000,Joe Again,RB,1998-03-14,ACT\n"},
		{"empty (header only)", header},
		{"ragged row", header + "00-0011000,Joe\n"},
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
	// Only a DOB-less row -> zero resolved -> errEmpty.
	body := header + "00-0011000,No DOB,RB,,ACT\n"
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
