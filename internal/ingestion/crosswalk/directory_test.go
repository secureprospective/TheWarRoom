package crosswalk

import (
	"maps"
	"net/http"
	"slices"
	"testing"
)

// Rows shaped on the live league (2026-10-03): 0816 is Stephen Gosnell on MFL but Dre' Bly in
// DynastyProcess; 13502 and 8463 share a gsis id; 17471 is a rookie with no gsis yet; 0300 is a
// retired player MFL no longer lists.
const directoryCSV = "mfl_id,gsis_id,espn_id,pfr_id,name\n" +
	"13294,00-0011000,4040,VetxPa00,Pat Veteran Jr.\n" +
	"816,,2111,BlyxDr00,Dre' Bly\n" +
	"13502,00-0031636,,,Justin Hamilton\n" +
	"8463,00-0031636,9999,,Justin Hamilton\n" +
	"17471,,5150,,Diego Pavia\n" +
	"300,00-0002000,,BarbRo00,Ronde Barber\n"

func TestLinksGuardAgainstMisattribution(t *testing.T) {
	t.Parallel()
	m, err := serve(t, http.StatusOK, directoryCSV)
	if err != nil {
		t.Fatal(err)
	}
	onMFL := map[string]string{
		"13294": "Veteran, Pat", "0816": "Gosnell, Stephen", "13502": "Hamilton, Justin",
		"8463": "Hamilton, Justin", "17471": "Pavia, Diego",
	}
	links, rejected := m.Links(func(id string) (string, bool) { n, ok := onMFL[id]; return n, ok })

	got := map[Link]bool{}
	for _, l := range links {
		got[l] = true
	}
	for _, want := range []Link{
		{"gsis", "00-0011000", "13294"}, {"espn", "4040", "13294"}, {"pfr", "VetxPa00", "13294"},
		{"espn", "9999", "8463"},                                    // its own espn id still links
		{"espn", "5150", "17471"},                                   // no gsis yet, other ids link
		{"gsis", "00-0002000", "0300"}, {"pfr", "BarbRo00", "0300"}, // not on MFL: DynastyProcess's word
	} {
		if !got[want] {
			t.Errorf("missing link %+v", want)
		}
	}
	for _, l := range links {
		if l.MFLID == "0816" || l.IDValue == "00-0031636" {
			t.Errorf("linked %+v: a guard should have stopped it", l)
		}
	}
	if len(links) != 7 {
		t.Errorf("links = %d, want 7: %+v", len(links), links)
	}
	wantRejected := map[string]string{"0816": ReasonOtherPlayer, "13502": ReasonSharedGSIS, "8463": ReasonSharedGSIS}
	for id, why := range wantRejected {
		if rejected[id] != why {
			t.Errorf("rejected[%s] = %q, want %q", id, rejected[id], why)
		}
	}
	if len(rejected) != len(wantRejected) {
		t.Errorf("rejected = %v", rejected)
	}
	if e, ok := m.Entry("17471"); !ok || e.Name != "Diego Pavia" || e.IDs["gsis"] != "" {
		t.Errorf("Entry(17471) = %+v, %t; want a row without gsis", e, ok)
	}
}

func TestSameName(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		mfl, dp string
		same    bool
	}{
		{"Harrison Jr., Marvin", "Marvin Harrison", true},
		{"St. Brown, Amon-Ra", "Amon-Ra St. Brown", true},
		{"Smith-Njigba, Jaxon", "Jaxon Smith-Njigba", true},
		{"Gosnell, Stephen", "Dre' Bly", false},
		{"Steelers, Pittsburgh", "Ronde Barber", false},
	} {
		if got := sameName(c.mfl, c.dp); got != c.same {
			t.Errorf("sameName(%q, %q) = %t", c.mfl, c.dp, got)
		}
	}
}

func TestIDColumnsTakeEveryIDButMFL(t *testing.T) {
	t.Parallel()
	cols := idColumns([]string{"mfl_id", "gsis_id", "name", "sleeper_id", "_id"})
	got := slices.Sorted(maps.Keys(cols))
	if !slices.Equal(got, []string{"gsis", "sleeper"}) {
		t.Errorf("idColumns = %v", got)
	}
}
