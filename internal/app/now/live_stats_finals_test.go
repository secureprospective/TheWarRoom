package now

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
)

func TestLiveStatLinesRereadsFinishedGamesOnlyEveryTenMinutes(t *testing.T) {
	body, err := os.ReadFile("../../ingestion/livegame/testdata/espn-summary-401872991.json")
	if err != nil {
		t.Fatal(err)
	}
	game, err := livegame.ParseSummary(body)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := livestats.Lines(game)
	if err != nil || len(lines) == 0 {
		t.Fatalf("fixture lines: %d, %v", len(lines), err)
	}
	byESPN := map[string]string{lines[0].SourceID: "15263"}
	reads := map[string]int{}
	adapter := livegame.Adapter{Fetch: func(_ context.Context, _ string, id string) ([]byte, error) {
		reads[id]++
		return body, nil
	}}
	games := []livegame.Game{{ID: "done", Status: "post", Season: 2026, Week: 8, SeasonType: 2}, {ID: "playing", Status: "in", Season: 2026, Week: 8, SeasonType: 2}, {ID: "later", Status: "pre"}}
	var finals liveFinals
	start := time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC)
	for minute := 0; minute < 10; minute++ {
		out, _ := liveStatLines(context.Background(), adapter, games, byESPN, &finals, start.Add(time.Duration(minute)*time.Minute))
		if got := out["15263"]; got.Week != 8 || got.Season != 2026 {
			t.Fatalf("minute %d: player line missing or misfiled: %+v", minute, got)
		}
	}
	if reads["done"] != 1 || reads["playing"] != 10 || reads["later"] != 0 {
		t.Fatalf("ten polls read finished %d, playing %d, not started %d times; want 1, 10, 0", reads["done"], reads["playing"], reads["later"])
	}
	liveStatLines(context.Background(), adapter, games, byESPN, &finals, start.Add(liveFinalReread))
	if reads["done"] != 2 {
		t.Fatalf("finished game not read again after %s: %d reads", liveFinalReread, reads["done"])
	}
}

func TestLiveStatLinesUseEachGamesScope(t *testing.T) {
	var finals liveFinals = map[string]liveFinal{
		"old": {at: time.Now(), lines: []livestats.Line{{SourceID: "1"}}},
		"new": {at: time.Now(), lines: []livestats.Line{{SourceID: "2"}}},
	}
	games := []livegame.Game{
		{ID: "old", Status: "post", Season: 2025, Week: 18, SeasonType: 2},
		{ID: "new", Status: "post", Season: 2026, Week: 1, SeasonType: 2},
		{ID: "unknown", Status: "post"},
		{ID: "postseason", Status: "post", Season: 2026, Week: 1, SeasonType: 3},
	}
	out, _ := liveStatLines(t.Context(), livegame.Adapter{}, games, map[string]string{"1": "0099", "2": "0100"}, &finals, time.Now())
	if len(out) != 2 || out["0099"].Season != 2025 || out["0099"].Week != 18 || out["0100"].Week != 1 {
		t.Fatalf("misfiled lines: %+v", out)
	}
}
