package main

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TestSeasonInABoxPlayoff follows the season box through the playoff command door. The regular-season fixture
// cannot resolve its seeded bracket: the shared fixture has no conference assignment for its divisions.
// The test records the exact hosted refusal rather than pretending the bracket is seeded.
func TestSeasonInABoxPlayoff(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the twr-server binary")
	}
	b := startBinaryBox(t)
	b.ledgerRevision(t)
	b.quickDraft(t)
	b.setOpponentLineup(t)
	b.setClock(t, b.at.Add(25*time.Hour))

	frames := replayFrames(t)
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-01-progress.json")
	b.awaitProvisional(t, func(points int64) bool { return points > 0 })
	b.useLiveFrame(t, frames, "espn-401873037-synthetic-02-progress.json")
	b.awaitProvisional(t, func(points int64) bool { return points > 0 })
	b.useLiveFrame(t, frames, "espn-401873037-raw-final.json")
	b.awaitNFLFinal(t)
	b.setClock(t, b.at.Add(6*time.Hour))
	b.useOfficial(t, "stats_player_week_2026.csv.gz")
	b.awaitPoints(t, func(int64) bool { return true })
	b.setClock(t, b.at.Add(time.Hour))
	b.useOfficial(t, "stats_player_week_2026_corrected.csv.gz")
	b.awaitPoints(t, func(points int64) bool { return points > 0 })

	build := commissioner.RawCommand{League: "native.hosted", Command: commissioner.BracketBuild,
		Request: "box.playoff.build", Effective: "now", Visibility: "league", Expected: b.ledgerRevision(t),
		Reason: "The fake clock has passed the playoff kickoff", Args: commissioner.RawArgs{Bracket: &commissioner.RawBracketChange{
			Season: 2026, Started: true, Preset: "legacy_nfl"}}}
	body, err := json.Marshal(build)
	if err != nil {
		t.Fatal(err)
	}
	printed, err := b.twr(t, string(body), "preview", "-")
	if err != nil {
		t.Fatalf("bracket.build preview door: %v (%s)", err, printed)
	}
	var preview struct {
		Reply json.RawMessage `json:"reply"`
	}
	if err := json.Unmarshal([]byte(printed), &preview); err != nil {
		t.Fatalf("decode bracket.build preview %q: %v", printed, err)
	}
	t.Logf("PLAYOFF STEP 1 bracket.build preview: %s", preview.Reply)

	var bracket commissioner.NativeBracket
	b.read(t, "bracket.read", map[string]int{"season": 2026}, &bracket)
	want := `Needs a ruling: which conference the division "Practice division 1" is in`
	if bracket.Status != "Needs a ruling" || bracket.Words != want {
		t.Fatalf("expected hosted bracket read refusal %q; got %q: %s", want, bracket.Status, bracket.Words)
	}
	t.Logf("PLAYOFF STEP 2 bracket.read: status %q: %s", bracket.Status, bracket.Words)
	notReached(t, 3, fmt.Sprintf("cannot build/seed from standings because the hosted read cannot resolve the fixture's divisions: %s", bracket.Words))
	notReached(t, 4, "playoff week scoring and advancement depend on a completed seeded bracket")
	notReached(t, 5, "championship final and champion depend on playoff advancement")
}
