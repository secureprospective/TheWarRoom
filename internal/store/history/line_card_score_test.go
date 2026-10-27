package history

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// The score reader scores an O-Line card from the pinned line games: OFF in Legacy, then, once the
// commissioner sets the line unit, PIT's game scores Set B, BAL's (no yards before contact yet) waits,
// ARI (no game in a held week) is a bye, and an unheld week is not held.
func TestScoreReaderScoresAnOLineCard(t *testing.T) {
	s, _ := statStore(t, filepath.Join(t.TempDir(), "card.db"))
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, heldSeedCommand(t), "creator")
	pinned := installScoringImport(t, s, lineImport(t))
	read := func(card string, week int) commissioner.NativePlayerScore {
		t.Helper()
		score, err := s.NativePlayerScore(t.Context(), "native.test", source.RawNativeScope{Player: card, Season: 2026, Week: week})
		if err != nil {
			t.Fatal(err)
		}
		return score
	}
	if got := read("990027", 1); got.Points != nil || got.Words != "This league does not score an O-Line" {
		t.Fatalf("Legacy scored a card: %+v", got)
	}
	unit, err := json.Marshal(leaguerules.Default32Scoring().LineUnit.Value)
	if err != nil {
		t.Fatal(err)
	}
	runScoreCommand(t, s, ruleCommand("scoring.lineUnit", string(unit), "now", "Score the line", 2), "creator")
	held, _, err := pinned.LineGames(t.Context(), 2026)
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := fantasyscore.ScoreUnit(leaguerules.Default32Scoring(), held.Games[source.LineGame{GameID: "2025_01_PIT_NYJ", Team: "PIT"}])
	if err != nil || want.State != leaguerules.IsKnown {
		t.Fatalf("PIT line: %+v %v", want, err)
	}
	pit := read("990027", 1)
	if pit.Status != "Complete" || pit.Points == nil || *pit.Points != want.Value || pit.Position != leaguerules.OL || pit.SHA256 != held.SHA {
		t.Fatalf("PIT card %+v, want %d", pit, want.Value)
	}
	if bal := read("990003", 1); bal.Status != "Incomplete" || bal.Points != nil {
		t.Fatalf("BAL scored without yards before contact: %+v", bal)
	}
	if ari := read("990001", 1); ari.Status != "Complete" || ari.Points == nil || *ari.Points != 0 {
		t.Fatalf("ARI bye: %+v", ari)
	}
	if later := read("990027", 2); later.Points != nil || later.Words != "No O-Line statistics are held for this week" {
		t.Fatalf("unheld week: %+v", later)
	}
}
