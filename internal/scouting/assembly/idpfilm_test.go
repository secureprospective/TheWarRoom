package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/madden"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func idpFinite(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// --- pure composite math (maddenComposite / termValue) ---

// TestMaddenComposite_EqualWeightMeanCoverageAveraged pins the LOCKED K1 recipe: an
// equal-weight mean of the curated terms, with man+zone coverage averaged into ONE term
// (so coverage is NOT double-weighted vs the eight single CB terms).
func TestMaddenComposite_EqualWeightMeanCoverageAveraged(t *testing.T) {
	t.Parallel()
	// CB curated set = 9 terms; the 5th is {manCoverage, zoneCoverage}. Give every
	// single term the SAME rating (50) and the two coverage attrs 80/40 (avg 60), so the
	// composite = mean of eight 50/99 terms + one 60/99 coverage term.
	rc := madden.RawMaddenRating{Attributes: map[string]int{
		"speed": 50, "acceleration": 50, "press": 50, "strength": 50,
		"manCoverage": 80, "zoneCoverage": 40, // coverage term averages to 60
		"agility": 50, "playRecognition": 50, "catching": 50, "jumping": 50,
	}}
	got, ok := maddenComposite(rc, domain.PosCB)
	if !ok {
		t.Fatal("CB composite reported no signal")
	}
	want := (8*(50.0/99.0) + (60.0 / 99.0)) / 9.0
	if !idpFinite(got, want) {
		t.Fatalf("CB composite = %.6f, want %.6f (coverage must average man+zone, not count both)", got, want)
	}
	// A swap of the coverage term for TWO separate 80/40 terms would give a different
	// (higher-count) mean — prove counting both would diverge, so averaging matters.
	if bad := (8*(50.0/99.0) + 80.0/99.0 + 40.0/99.0) / 10.0; idpFinite(want, bad) {
		t.Fatal("test is toothless: averaged vs counted-both coverage resolve equal")
	}
}

// TestMaddenComposite_BoundaryAndPartial covers the non-IDP boundary, the fully-absent
// record, and the Data-Parity partial (average only the present terms).
func TestMaddenComposite_BoundaryAndPartial(t *testing.T) {
	t.Parallel()
	full := madden.RawMaddenRating{Attributes: map[string]int{"tackle": 90, "strength": 66}}

	if _, ok := maddenComposite(full, domain.PosWR); ok {
		t.Error("WR (offense) must report no IDP film signal")
	}
	if _, ok := maddenComposite(full, domain.PosQB); ok {
		t.Error("QB must report no IDP film signal")
	}
	if _, ok := maddenComposite(madden.RawMaddenRating{Attributes: map[string]int{"unrelated": 90}}, domain.PosLB); ok {
		t.Error("a record carrying none of the curated LB attrs must report absent")
	}
	// DT with only two of its seven curated attrs present → mean of the two present terms.
	got, ok := maddenComposite(full, domain.PosDT)
	if !ok {
		t.Fatal("partial DT record reported no signal")
	}
	if want := (90.0/99.0 + 66.0/99.0) / 2.0; !idpFinite(got, want) {
		t.Fatalf("partial DT composite = %.6f, want %.6f (average PRESENT terms only)", got, want)
	}
}

// TestMaddenComposite_ClampsBeyondCeiling proves a rating above the Madden ceiling can
// never push a term past 1.0 (the film composite must stay in [0,1]).
func TestMaddenComposite_ClampsBeyondCeiling(t *testing.T) {
	t.Parallel()
	rc := madden.RawMaddenRating{Attributes: map[string]int{"tackle": 150}}
	got, ok := maddenComposite(rc, domain.PosDT)
	if !ok || !idpFinite(got, 1.0) {
		t.Fatalf("over-ceiling term = %.6f (ok=%v), want clamped to 1.0", got, ok)
	}
}

// --- BuildIDPFilm end-to-end (crosswalk name+birth resolver + madden fixture) ---

// idpFilmCrosswalkCSV carries name + birthdate columns so MaddenResolver resolves each EA
// record to a gsis. mfl 1001 (DT) / 1002 (CB) / 1003 (WR, offense) → gsis G-1/G-2/G-3.
const idpFilmCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id,name,birthdate
1001,G-1,,,Al Pha,1997-09-03
1002,G-2,,,Bo Beta,1999-01-02
1003,G-3,,,Cy Gamma,2000-05-06
`

// maddenDoc builds one EA-shaped record: identity + each rating as "<attr>_rating".
func maddenDoc(first, last, birth string, ratings map[string]int) string {
	keys := make([]string, 0, len(ratings))
	for k := range ratings {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	fmt.Fprintf(&b, `{"firstName":%q,"lastName":%q,"team":"X","position":"NA","plyrBirthdate":%q,"plyrAssetname":"a_1","overall_rating":70`,
		first, last, birth)
	for _, k := range keys {
		fmt.Fprintf(&b, `,%q:%d`, k+"_rating", ratings[k])
	}
	b.WriteString("}")
	return b.String()
}

func maddenServer(t *testing.T, docs []string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		var slice []string
		if offset < len(docs) {
			slice = docs[offset:]
		}
		fmt.Fprintf(w, `{"count":%d,"docs":[%s]}`, len(docs), strings.Join(slice, ","))
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func TestBuildIDPFilm_JoinsAndComposesByPosition(t *testing.T) {
	cw := crosswalkFixture(t, idpFilmCrosswalkCSV)
	if cw.LenMaddenResolver() != 3 {
		t.Fatalf("resolver indexed %d entries, want 3", cw.LenMaddenResolver())
	}
	client, url := maddenServer(t, []string{
		// DT: two curated attrs present → composite = mean(60/99, 90/99).
		maddenDoc("Al", "Pha", "9/3/1997", map[string]int{"tackle": 60, "strength": 90}),
		// CB: only the coverage term present → composite = avg(man 88, zone 66)/99.
		maddenDoc("Bo", "Beta", "1/2/1999", map[string]int{"manCoverage": 88, "zoneCoverage": 66}),
		// WR (offense) — resolves to a gsis but maddenComposite rejects the position.
		maddenDoc("Cy", "Gamma", "5/6/2000", map[string]int{"speed": 95}),
	})

	pos := fakePosLookup{"1001": domain.PosDT, "1002": domain.PosCB, "1003": domain.PosWR}
	out, err := BuildIDPFilm(context.Background(), client, url, cw, []string{"1001", "1002", "1003"}, pos)
	if err != nil {
		t.Fatalf("BuildIDPFilm: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d composites, want 2 (WR must be skipped)", len(out))
	}
	dt, _ := playerid.New("1001")
	if want := (60.0/99.0 + 90.0/99.0) / 2.0; !idpFinite(out[dt], want) {
		t.Errorf("DT composite = %.6f, want %.6f", out[dt], want)
	}
	cb, _ := playerid.New("1002")
	if want := (88.0/99.0 + 66.0/99.0) / 2.0; !idpFinite(out[cb], want) {
		t.Errorf("CB composite = %.6f, want %.6f (coverage term = man/zone avg)", out[cb], want)
	}
}

func TestBuildIDPFilm_FetchFailureLoud(t *testing.T) {
	cw := crosswalkFixture(t, idpFilmCrosswalkCSV)
	// A server that returns an empty doc set → madden.Fetch resolves zero records → loud.
	client, url := maddenServer(t, nil)
	if _, err := BuildIDPFilm(context.Background(), client, url, cw,
		[]string{"1001"}, fakePosLookup{"1001": domain.PosDT}); err == nil {
		t.Fatal("expected a loud error on a zero-record Madden fetch")
	}
}

func TestBuildIDPFilm_GuardsNilDeps(t *testing.T) {
	if _, err := BuildIDPFilm(context.Background(), nil, "u", crosswalk.Map{},
		nil, fakePosLookup{}); err == nil {
		t.Error("expected an error for a nil *http.Client")
	}
	if _, err := BuildIDPFilm(context.Background(), &http.Client{}, "u", crosswalk.Map{},
		nil, nil); err == nil {
		t.Error("expected an error for a nil PositionLookup")
	}
}
