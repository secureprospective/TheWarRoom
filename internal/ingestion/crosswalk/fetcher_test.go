package crosswalk

import (
	"context"
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

// --- pfr_id -> gsis bridge (second map) ---

// A header carrying pfr_id (optional, by-name) resolves the pfr bridge alongside the
// others; PFRMap exposes it and LenPFR counts it.
func TestFetch_PFRBridge(t *testing.T) {
	t.Parallel()
	body := "name,gsis_id,pfr_id,team,mfl_id\n" +
		"Pat Veteran,00-0011000,VetePa00,KC,13294\n" + // pfr + mfl resolve
		"NflVet NoPfr,00-0022000,,DET,531\n" + // mfl resolves, pfr absent
		"PfrOnly Guy,00-0033000,OnlyPf00,FA,\n" // pfr resolves, no mfl
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	pfr := m.PFRMap()
	if pfr["VetePa00"] != "00-0011000" || pfr["OnlyPf00"] != "00-0033000" {
		t.Errorf("PFRMap wrong: %v", pfr)
	}
	if _, ok := pfr["nope"]; ok {
		t.Error("unknown pfr id should miss")
	}
	if m.LenPFR() != 2 {
		t.Errorf("LenPFR = %d, want 2", m.LenPFR())
	}
	// PFRMap is a defensive copy: mutating it must not affect the crosswalk.
	pfr["VetePa00"] = "tampered"
	if again := m.PFRMap(); again["VetePa00"] != "00-0011000" {
		t.Error("PFRMap must return a copy, not the internal map")
	}
}

// A pfr id mapping to two different gsis is dropped (shared drop-ambiguous policy),
// without erroring the fetch.
func TestFetch_PFRConflictDropped(t *testing.T) {
	t.Parallel()
	body := "gsis_id,pfr_id,mfl_id\n" +
		"00-0011000,DupePf00,13294\n" +
		"00-0099999,DupePf00,17464\n" + // same pfr, different gsis -> drop
		"00-0088000,GoodPf00,531\n"
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("ambiguous pfr should drop-and-continue, not error: %v", err)
	}
	if _, ok := m.PFRMap()["DupePf00"]; ok {
		t.Error("ambiguous pfr id must be dropped")
	}
	if m.PFRMap()["GoodPf00"] != "00-0088000" {
		t.Error("unrelated pfr id must still resolve")
	}
}

// Len is the number of MFL -> gsis entries.
func (m Map) Len() int { return len(m.byMFL) }

// LenPFR is the number of pfr -> gsis entries.
func (m Map) LenPFR() int { return len(m.byPFR) }
