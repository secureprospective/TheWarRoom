package commissioner

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// TestFrozenResultsEncodeAsFresh: freezing a reading never changes its JSON, with or without a bye, evidence
// or empty lists, and the per-read fields set after freezing still show.
func TestFrozenResultsEncodeAsFresh(t *testing.T) {
	t.Parallel()
	line := map[leaguerules.StatID]leaguerules.Decimal{"PY": 250000, "RY": 15000, "<b>": 1}
	scored := NativeResults{League: "L&1", Season: 2026, Week: 3, Published: true, Words: "a <week>",
		Games: []NativeGame{{Week: 3, Home: "0001", Away: "0002", Why: []string{"x"}}},
		Bye:   "0003", ByeName: "Byes & co",
		Scores: []FranchiseWeekScore{{Franchise: "0001", Starters: []StarterScore{{Player: "0123", Line: line,
			FieldSources: map[leaguerules.StatID]string{"PY": "nflverse"}}}}},
		Missed: []MissedLineup{{Franchise: "0004", Flagged: true}}}
	for name, reading := range map[string]NativeResults{"full": scored, "empty": {League: "L"},
		"no bye": {League: "L", Games: scored.Games, Scores: scored.Scores}} {
		frozen := reading
		if err := frozen.Freeze(); err != nil {
			t.Fatal(err)
		}
		frozen.Revision, reading.Revision = 42, 42
		frozen.Evidence = &ScoreEvidence{Live: "g1", AsOf: time.Unix(1, 0).UTC()}
		reading.Evidence = frozen.Evidence
		type plain NativeResults
		want, err := json.Marshal(plain(reading))
		if err != nil {
			t.Fatal(err)
		}
		viaMarshal, err := json.Marshal(frozen)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := frozen.EncodedJSON()
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(viaMarshal, want) || !bytes.Equal(encoded, want) {
			t.Fatalf("%s: frozen JSON differs:\nmarshal %s\nencoded %s\n   want %s", name, viaMarshal, encoded, want)
		}
	}
}
