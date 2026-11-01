package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestMirrorRulesReadingWordsHeldSettingsInTheNativeShape(t *testing.T) {
	t.Parallel()
	got := MirrorRulesReading("14432", "Legacy NFL", leaguerules.LegacyNFL(), 1700000000)
	if got.League != "14432" || got.Name != "Legacy NFL" || got.Revision != 1700000000 || len(got.Sections) == 0 {
		t.Fatalf("mirror rules keep league, name and the held revision: %+v", got)
	}
	rows := 0
	for _, s := range got.Sections {
		for _, r := range s.Rows {
			rows++
			if r.Citation != "Held from MFL" || r.Pending == nil || len(r.Pending) != 0 || r.Label == "" {
				t.Fatalf("every mirror row cites MFL and has nothing pending: %+v", r)
			}
		}
	}
	if rows != len(leaguerules.Catalogue()) {
		t.Fatalf("one row per catalogued setting: %d of %d", rows, len(leaguerules.Catalogue()))
	}
}
