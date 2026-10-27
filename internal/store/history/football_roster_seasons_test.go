package history

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestPinnedNFLSeasonsCountRosterCalendarNotElapsedYearsOrGames(t *testing.T) {
	body, _ := footballBody(t)
	var raw RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	seasons := []int{2020, 2022, 2026} // Years out and other leagues are not NFL roster appearances.
	p := raw.Players["0099"]
	p.NFLRosterSeasons = &seasons
	p.Facts.DraftYear, p.Facts.DraftYearHeld, p.Facts.CareerGames = 2020, true, 1
	raw.Players["0099"] = p
	store, _ := statStore(t, filepath.Join(t.TempDir(), "history.db"))
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	digest := footballDigest(body)
	if err := store.ImportFootball(t.Context(), body, digest); err != nil {
		t.Fatal(err)
	}
	held, err := store.LoadFootball(t.Context(), digest)
	if err != nil {
		t.Fatal(err)
	}
	f := held.PlayerFacts()["0099"]
	if !f.NFLSeasonsHeld || f.NFLSeasons != 3 {
		t.Fatalf("roster count: %+v", f)
	}
	verdict := nativecheck.CanEnterTaxi(leaguerules.Default32().Taxi, f, 2026, 0)
	if !verdict.Allowed {
		t.Fatalf("calendar gap or single game changed eligibility: %+v", verdict)
	}
	for _, years := range [][]int{{2020, 2020}, {1919}, {2027}} {
		p.NFLRosterSeasons = &years
		raw.Players["0099"] = p
		if err := raw.Validate(); err == nil {
			t.Fatalf("accepted invalid seasons %v", years)
		}
	}
}

func TestPinnedOfficialDesignationsAreNotInferred(t *testing.T) {
	body, _ := footballBody(t)
	var raw RawFootballEvidence
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, status := range []leaguerules.NFLStatus{leaguerules.InjuredReserve, leaguerules.PUP, leaguerules.NFI, leaguerules.Out, ""} {
		p := raw.Players["0099"]
		p.Facts.Designation = status
		raw.Players["0099"] = p
		if err := raw.Validate(); err != nil {
			t.Fatal(err)
		}
		facts := (&PinnedFootball{raw: raw}).PlayerFacts()["0099"]
		if facts.Designation != status {
			t.Fatalf("official status %q reclassified as %q", status, facts.Designation)
		}
		entry := nativecheck.CanEnterIR(leaguerules.Default32().IR, facts, 0)
		if entry.Allowed != (status == leaguerules.InjuredReserve || status == leaguerules.PUP) {
			t.Fatalf("designation %q: %+v", status, entry)
		}
	}
}
