package history

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNativeEvidenceUsesDirectoryAndNewestHeldRevision(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "identity.db"))
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "0099"},
	}); err != nil {
		t.Fatal(err)
	}
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	newest := putStatBody(t, s, statBody(t, "245", true), "", t0().Add(time.Hour))
	scope := source.RawNativeScope{Player: "0099", Season: 2025, Week: 1}
	evidence, err := source.ReadNativeEvidence(t.Context(), s, s, scope)
	if err != nil || evidence.GSIS != "00-0023459" || evidence.Held == nil || evidence.Held.Week == nil ||
		evidence.Held.SHA256 != newest || evidence.Held.Week.Line.Quantities["passing_yards"] != 2450000 {
		t.Fatalf("directory to held statistics: %+v %v", evidence, err)
	}
	if len(evidence.Missing) != 1 || !strings.Contains(evidence.Missing[0], "play-by-play") {
		t.Fatalf("missing evidence hidden: %+v", evidence)
	}
	missing := putStatBody(t, s, statBody(t, "245", false), "", t0().Add(2*time.Hour))
	evidence, err = source.ReadNativeEvidence(t.Context(), s, s, scope)
	if err != nil || evidence.Held == nil || evidence.Held.SHA256 != missing || evidence.Held.Week != nil {
		t.Fatalf("absent newest player replaced with older line: %+v %v", evidence, err)
	}
}

func TestNativeEvidenceMissingIdentityIsNotAZeroWeek(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "missing.db"))
	putStatBody(t, s, statBody(t, "245", true), "", t0())
	evidence, err := source.ReadNativeEvidence(t.Context(), s, s,
		source.RawNativeScope{Player: "0099", Season: 2025, Week: 1})
	if err != nil || evidence.GSIS != "" || evidence.Held != nil || len(evidence.Missing) != 1 ||
		evidence.Missing[0] != "This player has no matching NFL statistics identity" {
		t.Fatalf("missing match became a zero week: %+v %v", evidence, err)
	}
}

func TestNativeEvidenceUnusableIdentityIsMissingNotAnError(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "unusable.db"))
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: "THO581952", PlayerID: "0099"},
	}); err != nil {
		t.Fatal(err)
	}
	putStatBody(t, s, statBody(t, "245", true), "", t0())
	evidence, err := source.ReadNativeEvidence(t.Context(), s, s,
		source.RawNativeScope{Player: "0099", Season: 2025, Week: 1})
	if err != nil || evidence.Held != nil || len(evidence.Missing) != 1 ||
		evidence.Missing[0] != "This player has no matching NFL statistics identity" {
		t.Fatalf("unusable directory id stopped the read: %+v %v", evidence, err)
	}
}

func TestNativeEvidenceScopeAndUnloadedSeason(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "scope.db"))
	for _, scope := range []source.RawNativeScope{{Player: "99", Season: 2025, Week: 1},
		{Player: "0099", Season: 1919, Week: 1}, {Player: "0099", Season: 2025, Week: 0}} {
		if _, err := source.ReadNativeEvidence(t.Context(), s, s, scope); err == nil {
			t.Fatalf("invalid native scope accepted: %+v", scope)
		}
	}
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0023459", PlayerID: "0099"},
	}); err != nil {
		t.Fatal(err)
	}
	evidence, err := source.ReadNativeEvidence(t.Context(), s, s,
		source.RawNativeScope{Player: "0099", Season: 2025, Week: 1})
	if err != nil || evidence.Held != nil || len(evidence.Missing) != 1 ||
		!strings.Contains(evidence.Missing[0], "No NFL statistics") {
		t.Fatalf("no held files became evidence: %+v %v", evidence, err)
	}
}

func TestNativeEvidenceCarriesPlayerRecordScoringPosition(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "position.db"))
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{{IDType: "gsis", IDValue: "00-0023459", PlayerID: "0099"}}); err != nil {
		t.Fatal(err)
	}
	putStatBody(t, s, statBody(t, "244", true), "", t0())
	s.SetLineupEnvironment(&lineupEnvFake{facts: nativecheck.Facts{"0099": {Position: leaguerules.QB}}})
	evidence, err := s.nativeEvidence(t.Context(), s, source.RawNativeScope{Player: "0099", Season: 2025, Week: 1})
	if err != nil || evidence.ListedPosition != "QB" {
		t.Fatalf("existing record not supplied: %+v %v", evidence, err)
	}
}

// An MFL import pins players by MFL id alone; before kickoff a linked player must still read as identified.
func TestNativeEvidenceMFLImportPinReadsDirectoryIdentity(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "mfl-pin.db"))
	if _, err := s.LinkPlayerIDs(t.Context(), "dynastyprocess", []PlayerIDLink{
		{IDType: "gsis", IDValue: "00-0040563", PlayerID: "17269"},
	}); err != nil {
		t.Fatal(err)
	}
	s.SetPinnedFootball(&PinnedFootball{digest: strings.Repeat("a", 64), raw: RawFootballEvidence{Season: 2026,
		Players: map[string]RawFootballPlayer{"17269": {Identity: MFLIdentityPrefix + "17269"}, "17270": {Identity: MFLIdentityPrefix + "17270"}}}})
	scope := source.RawNativeScope{Player: "17269", Season: 2026, Week: 10}
	evidence, err := s.nativeEvidence(t.Context(), s, scope)
	if err != nil || evidence.GSIS != "00-0040563" {
		t.Fatalf("linked MFL import player read as unidentified: %+v %v", evidence, err)
	}
	scope.Player = "17270"
	if evidence, err = s.nativeEvidence(t.Context(), s, scope); err != nil || evidence.GSIS != "" {
		t.Fatalf("unlinked MFL import player gained an identity: %+v %v", evidence, err)
	}
}
