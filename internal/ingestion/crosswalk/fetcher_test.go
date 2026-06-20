package crosswalk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// serve spins an httptest server returning the given status + body and returns a
// Fetch result against it, so every test drives the real public surface (Fetch),
// including status handling and the LimitReader boundary.
func serve(t *testing.T, status int, body string) (Map, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL)
}

// mustID builds a validated MFL PlayerID for lookups, or fails the test.
func mustID(t *testing.T, raw string) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New(raw)
	if err != nil {
		t.Fatalf("playerid.New(%q): %v", raw, err)
	}
	return id
}

// A header with the two depended-on columns deliberately NOT first and NOT
// adjacent, to prove by-name binding survives arbitrary column order.
const goodCSV = "name,gsis_id,team,position,mfl_id,age\n" +
	"Pat Veteran,00-0011000,KC,QB,13294,30\n" +
	"Lo Rookie,00-0022000,DET,RB,531,22\n" + // mfl_id 531 must normalize to 0531
	"NflOnly Guy,00-0033000,SF,WR,,25\n" + // no mfl_id -> skipped
	"MflOnly Guy,,FA,TE,9999,28\n" // no gsis_id -> skipped

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	m, err := serve(t, http.StatusOK, goodCSV)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if m.Len() != 2 {
		t.Fatalf("Len = %d, want 2 (two one-sided rows skipped)", m.Len())
	}

	gsis, ok := m.Lookup(mustID(t, "13294"))
	if !ok || gsis != "00-0011000" {
		t.Errorf("Lookup vet = %q,%v; want 00-0011000,true", gsis, ok)
	}
	// RISK-003: a bare mfl_id under 1000 is zero-padded by the validating
	// constructor, so the key is "0531" — a lookup with the raw or padded form both
	// resolve through playerid.New.
	gsis, ok = m.Lookup(mustID(t, "531"))
	if !ok || gsis != "00-0022000" {
		t.Errorf("Lookup rookie = %q,%v; want 00-0022000,true", gsis, ok)
	}
	if _, ok := m.Lookup(mustID(t, "9999")); ok {
		t.Error("MFL-only row (no gsis) should not be in the crosswalk")
	}
	if _, ok := m.Lookup(mustID(t, "404040")); ok {
		t.Error("missing MFL id should miss, not match")
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
	}{
		{"missing mfl_id column", "name,gsis_id,team\nFoo,00-0011000,KC\n"},
		{"missing gsis_id column", "name,mfl_id,team\nFoo,13294,KC\n"},
		{"malformed mfl_id", "gsis_id,mfl_id\n00-0011000,not-a-number\n"},
		{"conflicting gsis for one mfl", "gsis_id,mfl_id\n00-0011000,13294\n00-0099999,13294\n"},
		{"empty (header only)", "gsis_id,mfl_id\n"},
		{"ragged row (short field count)", "gsis_id,mfl_id\n00-0011000\n"},
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

// An identical duplicate (same mfl -> same gsis) is deduplicated, not rejected;
// only a CONFLICTING duplicate (one mfl -> two gsis) is an integrity error.
func TestFetch_IdenticalDuplicateIsAllowed(t *testing.T) {
	t.Parallel()
	body := "gsis_id,mfl_id\n00-0011000,13294\n00-0011000,13294\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("identical duplicate should not error: %v", err)
	}
	if m.Len() != 1 {
		t.Errorf("Len = %d, want 1", m.Len())
	}
}

// The R-generated source encodes a missing cell as the literal "NA" (found live):
// the reverse direction gsis->mfl is one-to-many, but more practically many rows
// carry "NA" ids. Rows with an "NA" in either column must be skipped, never keyed.
func TestFetch_NASentinelSkipped(t *testing.T) {
	t.Parallel()
	body := "gsis_id,mfl_id\n" +
		"00-0055000,NA\n" + // NA mfl — skipped
		"00-0066000,NA\n" + // another NA mfl — must not collide as one "NA" key
		"00-0011000,13294\n" +
		"NA,17464\n" // NA gsis — skipped
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("NA rows should be skipped, not error: %v", err)
	}
	if m.Len() != 1 {
		t.Errorf("Len = %d, want 1 (only the real pair resolves)", m.Len())
	}
	if _, ok := m.Lookup(mustID(t, "13294")); !ok {
		t.Error("the one real pair should resolve")
	}
}

func TestFetch_EmptyIsSentinel(t *testing.T) {
	t.Parallel()
	_, err := serve(t, http.StatusOK, "gsis_id,mfl_id\n")
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty", err)
	}
}

func TestFetch_Non200(t *testing.T) {
	t.Parallel()
	if _, err := serve(t, http.StatusNotFound, "whatever"); err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}

// A UTF-8 BOM on the first header cell must not defeat by-name column binding.
func TestFetch_HeaderBOM(t *testing.T) {
	t.Parallel()
	body := "\ufeffmfl_id,gsis_id\n13294,00-0011000\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("BOM header should still parse: %v", err)
	}
	if _, ok := m.Lookup(mustID(t, "13294")); !ok {
		t.Error("expected the row to resolve despite the BOM")
	}
}

func TestFetch_ContextCancelled(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(goodCSV))
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before the call
	if _, err := Fetch(ctx, srv.Client(), srv.URL); err == nil {
		t.Fatal("expected error from a cancelled context")
	}
}
