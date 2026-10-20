package ras

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// xwalk is the default pfr->gsis fixture map. A func (not a package var) keeps the
// test free of globals (gochecknoglobals) and hands each caller its own copy.
func xwalk() map[string]string {
	return map[string]string{
		"AlfoDa20": "00-0011000",
		"AbraJo00": "00-0022000",
	}
}

func serve(t *testing.T, status int, body string, m map[string]string) (map[string]RawCombine, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Fetch(context.Background(), srv.Client(), srv.URL, m)
}

// Header carries the nine bound columns among unbound ones in a mixed order.
const header = "pfr_id,pos,school,ht,wt,forty,bench,vertical,broad_jump,cone,shuttle,season\n"

func deref(t *testing.T, p *float64, name string) float64 {
	t.Helper()
	if p == nil {
		t.Fatalf("%s: expected a value, got nil", name)
	}
	return *p
}

func TestFetch_HappyPathFullAndSparse(t *testing.T) {
	t.Parallel()
	body := header +
		// full workout
		"AlfoDa20,OT,Cuse,6-4,334,5.56,23,25,94,8.48,4.98,2020\n" +
		// sparse: only ht/wt/forty; the other drills empty (absent, not 0)
		"AbraJo00,OLB,Sch,6-4,252,4.55,,,,,,2016\n"
	m, err := serve(t, http.StatusOK, body, xwalk())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 2 {
		t.Fatalf("got %d records, want 2", len(m))
	}

	full := m["00-0011000"]
	if deref(t, full.HeightIn, "HeightIn") != 76 { // 6-4 => 76 in
		t.Errorf("height = %v, want 76", *full.HeightIn)
	}
	if deref(t, full.Forty, "Forty") != 5.56 || deref(t, full.Cone, "Cone") != 8.48 {
		t.Errorf("full workout parse wrong: %+v", full)
	}

	sparse := m["00-0022000"]
	if deref(t, sparse.Forty, "Forty") != 4.55 {
		t.Errorf("sparse forty wrong: %v", *sparse.Forty)
	}
	if sparse.Bench != nil || sparse.Vertical != nil || sparse.Cone != nil || sparse.Shuttle != nil {
		t.Errorf("absent drills must be nil, not zero: %+v", sparse)
	}
}

func TestFetch_UnresolvedAndMissingPfrSkipped(t *testing.T) {
	t.Parallel()
	body := header +
		"AlfoDa20,OT,Cuse,6-4,334,5.56,23,25,94,8.48,4.98,2020\n" +
		"Ghost999,RB,X,6-0,200,4.5,,,,,,2021\n" + // pfr not in xwalk
		",RB,X,6-0,200,4.5,,,,,,2021\n" // empty pfr
	m, err := serve(t, http.StatusOK, body, xwalk())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(m) != 1 {
		t.Fatalf("got %d, want 1 (unresolved/missing pfr skipped)", len(m))
	}
}

func TestFetch_NoMeasurablesSkipped(t *testing.T) {
	t.Parallel()
	// Resolvable pfr but every drill absent -> no RAS signal -> skipped -> errEmpty.
	body := header + "AlfoDa20,OT,Cuse,,,,,,,,,2020\n"
	_, err := serve(t, http.StatusOK, body, xwalk())
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty (no measurables)", err)
	}
}

func TestFetch_GateViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body string
		m    map[string]string
	}{
		{"missing required column", "pfr_id,ht,wt\nAlfoDa20,6-4,334\n", xwalk()},
		{"malformed forty", header + "AlfoDa20,OT,X,6-4,334,fast,,,,,,2020\n", xwalk()},
		{"malformed height (no dash)", header + "AlfoDa20,OT,X,604,334,5.56,,,,,,2020\n", xwalk()},
		{"non-numeric height feet", header + "AlfoDa20,OT,X,x-4,334,5.56,,,,,,2020\n", xwalk()},
		{"ragged row", header + "AlfoDa20,OT\n", xwalk()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := serve(t, http.StatusOK, tc.body, tc.m); err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

// A gsis that receives 2+ combine rows is ambiguous (combine.csv shares one pfr_id
// across two distinct players upstream) and is dropped ENTIRELY — not failed, not
// first-wins — while clean players survive. A third colliding row proves it stays
// dropped.
func TestFetch_AmbiguousGsisDropped(t *testing.T) {
	t.Parallel()
	m := map[string]string{
		"AlfoDa20": "00-0011000", // these three all collide on one gsis
		"Twin0000": "00-0011000",
		"Trip0000": "00-0011000",
		"AbraJo00": "00-0022000", // clean, distinct gsis
	}
	body := header +
		"AlfoDa20,OT,X,6-4,334,5.56,,,,,,2020\n" +
		"Twin0000,OT,X,6-5,340,5.60,,,,,,2021\n" +
		"Trip0000,OT,X,6-6,350,5.70,,,,,,2022\n" +
		"AbraJo00,OLB,X,6-4,252,4.55,,,,,,2016\n"
	res, err := serve(t, http.StatusOK, body, m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("got %d, want 1 (ambiguous gsis dropped, clean kept)", len(res))
	}
	if _, ok := res["00-0011000"]; ok {
		t.Error("ambiguous gsis must be dropped entirely")
	}
	if _, ok := res["00-0022000"]; !ok {
		t.Error("clean player must survive")
	}
}

func TestFetch_EmptyIsSentinel(t *testing.T) {
	t.Parallel()
	body := header + "Ghost999,RB,X,6-0,200,4.5,,,,,,2021\n" // unresolved -> zero resolved
	_, err := serve(t, http.StatusOK, body, xwalk())
	if !errors.Is(err, errEmpty) {
		t.Fatalf("err = %v, want errEmpty", err)
	}
}

func TestFetch_Non200(t *testing.T) {
	t.Parallel()
	if _, err := serve(t, http.StatusInternalServerError, "boom", xwalk()); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
