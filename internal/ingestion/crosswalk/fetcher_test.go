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

// --- espn_id -> gsis bridge (second map) ---

// A header carrying espn_id (not first, not adjacent to the others) proves the
// optional column is bound by name and the bridge resolves alongside MFL->gsis.
const espnCSV = "name,gsis_id,espn_id,team,mfl_id\n" +
	"Caleb Williams,00-0039918,4431611,CHI,16579\n" + // both bridges resolve
	"NflVet NoEspn,00-0011000,,KC,13294\n" + // MFL resolves, espn absent -> espn miss
	"College Only,00-0044000,4500000,FA,\n" // espn resolves, no mfl -> MFL miss

func TestFetch_ESPNBridge(t *testing.T) {
	t.Parallel()
	m, err := serve(t, http.StatusOK, espnCSV)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// espn -> gsis resolves for a drafted rookie and for a college-only (no mfl) row.
	if gsis, ok := m.GSISForESPN("4431611"); !ok || gsis != "00-0039918" {
		t.Errorf("GSISForESPN(4431611) = %q,%v; want 00-0039918,true", gsis, ok)
	}
	if gsis, ok := m.GSISForESPN("4500000"); !ok || gsis != "00-0044000" {
		t.Errorf("GSISForESPN(4500000) = %q,%v; want 00-0044000,true (espn resolves even with no mfl_id)", gsis, ok)
	}
	if _, ok := m.GSISForESPN("9999999"); ok {
		t.Error("unknown espn id should miss, not match")
	}

	// The two maps are independent: the college-only row is in the espn bridge but not
	// MFL->gsis; the no-espn vet is in MFL->gsis but not the espn bridge.
	if m.Len() != 2 {
		t.Errorf("MFL Len = %d, want 2 (the college-only row has no mfl_id)", m.Len())
	}
	if m.LenESPN() != 2 {
		t.Errorf("espn LenESPN = %d, want 2 (the no-espn vet contributes only to MFL)", m.LenESPN())
	}
}

// A source that omits espn_id entirely still resolves MFL->gsis (foundation intact)
// and yields an empty espn bridge — the column is optional, not required.
func TestFetch_ESPNColumnOptional(t *testing.T) {
	t.Parallel()
	m, err := serve(t, http.StatusOK, goodCSV) // goodCSV has no espn_id column
	if err != nil {
		t.Fatalf("Fetch should not require espn_id: %v", err)
	}
	if m.Len() == 0 {
		t.Fatal("MFL bridge should still resolve without an espn_id column")
	}
	if m.LenESPN() != 0 {
		t.Errorf("espn bridge LenESPN = %d, want 0 when the column is absent", m.LenESPN())
	}
}

// An espn id mapping to two different gsis is AMBIGUOUS source data (the RAS
// combine-collision class): it is dropped from the bridge and stays dropped, while
// every other espn id resolves normally — Fetch does NOT error on it.
func TestFetch_ESPNConflictDropped(t *testing.T) {
	t.Parallel()
	body := "gsis_id,espn_id,mfl_id\n" +
		"00-0011000,4431611,13294\n" + // ambiguous espn, first sighting
		"00-0099999,4431611,17464\n" + // same espn, different gsis -> drop + poison
		"00-0077000,4431611,12300\n" + // a THIRD sighting must not resurrect it
		"00-0088000,4500000,531\n" // an unrelated espn must still resolve
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("an ambiguous espn id should drop-and-continue, not error: %v", err)
	}
	if _, ok := m.GSISForESPN("4431611"); ok {
		t.Error("the ambiguous espn id must be dropped from the bridge entirely")
	}
	if gsis, ok := m.GSISForESPN("4500000"); !ok || gsis != "00-0088000" {
		t.Errorf("unrelated espn = %q,%v; want 00-0088000,true (one bad id must not poison the rest)", gsis, ok)
	}
	if m.LenESPN() != 1 {
		t.Errorf("espn LenESPN = %d, want 1 (only the clean id survives)", m.LenESPN())
	}
}

// An NA/empty espn cell is skipped (not keyed as a literal "NA"), and an identical
// espn->gsis duplicate is deduplicated, not rejected.
func TestFetch_ESPNNAAndIdenticalDup(t *testing.T) {
	t.Parallel()
	body := "gsis_id,espn_id,mfl_id\n" +
		"00-0011000,NA,13294\n" + // NA espn -> skipped, mfl still resolves
		"00-0022000,4431611,531\n" +
		"00-0022000,4431611,531\n" // identical row -> dedup, no conflict
	m, err := serve(t, http.StatusOK, body)
	if err != nil {
		t.Fatalf("NA/identical-dup espn should not error: %v", err)
	}
	if m.LenESPN() != 1 {
		t.Errorf("espn LenESPN = %d, want 1 (NA skipped, identical dup deduped)", m.LenESPN())
	}
	if _, ok := m.GSISForESPN("NA"); ok {
		t.Error("an NA espn cell must never be keyed as a literal \"NA\"")
	}
}

// --- pfr_id -> gsis bridge (third map) ---

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

// A source omitting pfr_id still resolves MFL->gsis and yields an empty pfr bridge.
func TestFetch_PFRColumnOptional(t *testing.T) {
	t.Parallel()
	m, err := serve(t, http.StatusOK, goodCSV) // no pfr_id column
	if err != nil {
		t.Fatalf("Fetch should not require pfr_id: %v", err)
	}
	if m.LenPFR() != 0 {
		t.Errorf("LenPFR = %d, want 0 when the column is absent", m.LenPFR())
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
