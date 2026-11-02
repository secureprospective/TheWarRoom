package composition

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livestats"
)

func TestLiveStatLinesRereadsFinishedGamesOnlyEveryTenMinutes(t *testing.T) {
	body, err := os.ReadFile("internal/ingestion/livegame/testdata/espn-summary-401872991.json")
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
	games := []livegame.Game{{ID: "done", Status: "post"}, {ID: "playing", Status: "in"}, {ID: "later", Status: "pre"}}
	var finals liveFinals
	start := time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC)
	for minute := 0; minute < 10; minute++ {
		out := liveStatLines(context.Background(), adapter, games, byESPN, 2026, 8, &finals, start.Add(time.Duration(minute)*time.Minute))
		if got := out["15263"]; got.Week != 8 || got.Season != 2026 {
			t.Fatalf("minute %d: player line missing or misfiled: %+v", minute, got)
		}
	}
	if reads["done"] != 1 || reads["playing"] != 10 || reads["later"] != 0 {
		t.Fatalf("ten polls read finished %d, playing %d, not started %d times; want 1, 10, 0", reads["done"], reads["playing"], reads["later"])
	}
	liveStatLines(context.Background(), adapter, games, byESPN, 2026, 8, &finals, start.Add(liveFinalReread))
	if reads["done"] != 2 {
		t.Fatalf("finished game not read again after %s: %d reads", liveFinalReread, reads["done"])
	}
}
