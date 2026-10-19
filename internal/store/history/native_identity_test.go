package history

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
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
