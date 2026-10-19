package commissioner

import (
	"strings"
	"testing"
)

func TestPreviewCatalogueUnknownCounts(t *testing.T) {
	catalogue, err := ReadPreviewCatalogue("id,name,gm_question,governs\n" +
		"CM-003,Roster limits,How large may a roster be?,\n" +
		"CM-056,Ledger,What changed?,ledger-wide\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, control := range []string{"CM-003", "CM-056", "CM-999"} {
		affects := catalogue.affects(control)
		if affects.Count != nil || len(affects.Readings) != 0 || affects.Words != "Not known yet" {
			t.Fatalf("invented affected count for %s: %+v", control, affects)
		}
	}
}

func TestPreviewCatalogueReadsRegistryEdges(t *testing.T) {
	catalogue, err := ReadPreviewCatalogue("id,name,gm_question,governs\n" +
		"M-001,Player points,What did my player score?,\n" +
		"M-002,Standings,Where does my team stand?,\n" +
		"CM-006,Scoring,What is each stat worth?,M-001 M-002\n")
	if err != nil {
		t.Fatal(err)
	}
	affects := catalogue.affects("CM-006")
	if affects.Count == nil || *affects.Count != 2 || strings.Join(affects.Readings, ";") !=
		"Player points;Standings" || catalogue.questions["CM-006"] != "What is each stat worth?" {
		t.Fatalf("catalogue did not supply affected rows and question: %+v", affects)
	}
	// A caller cannot change the catalogue through the returned slice.
	affects.Readings[0] = "changed"
	if catalogue.affects("CM-006").Readings[0] != "Player points" {
		t.Fatal("preview mutated the catalogue")
	}
}

func TestPreviewCatalogueRefusesBrokenRegistry(t *testing.T) {
	for _, body := range []string{"", "id,name\n", "id,name,gm_question,governs\nCM-003,Roster,Why?,M-999\n",
		"id,name,gm_question,governs\n\"unterminated"} {
		if _, err := ReadPreviewCatalogue(body); err == nil {
			t.Fatalf("unreadable registry accepted: %q", body)
		}
	}
}

func TestPreviewReadingNamesHideInternalMarkers(t *testing.T) {
	for raw, want := range map[string]string{
		"Rosters O=07": "Rosters", "League Rules O=09 → League Settings": "League Rules",
		"Salaries & Contracts R=SALARIES": "Salaries & Contracts", "Commissioner News ⛔": "Commissioner News",
		"By-Laws (custom module) ◆": "By-Laws (custom module)", "Contracts handler": "Contracts plans",
	} {
		if got := previewReadingName(raw); got != want {
			t.Errorf("%q: got %q, want %q", raw, got, want)
		}
	}
}
