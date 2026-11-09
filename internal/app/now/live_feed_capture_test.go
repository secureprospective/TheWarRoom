package now

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

func capturedLiveFeed(t *testing.T, name string) []byte {
	t.Helper()
	f, err := os.Open("testdata/" + name + ".json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestCapturedSleeperWeekFeedsAcquisition(t *testing.T) {
	body := capturedLiveFeed(t, "sleeper-week8-live")
	rows, err := source.LiveSleeperLines(source.RawSleeperWeek{Season: 2026, Week: 8, Body: string(body)})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 500 {
		t.Fatalf("only %d converted rows", len(rows))
	}
	// Use a real athlete row, not a team aggregate or fantasy-points field.
	id := ""
	for key, line := range rows {
		if key[0] >= '0' && key[0] <= '9' && line.Quantities["passing_yards"] > 2000000 {
			id = key
			break
		}
	}
	if id == "" {
		t.Fatal("captured passing evidence missing")
	}
	held := &capturedSleeperHistory{id: id}
	network := &sleeperTransport{t: t, status: 200, body: string(body)}
	lines := map[string]source.PlayerWeek{}
	if !fillLiveSleeper(t.Context(), LiveStatsRun{Network: network}, held,
		[]livegame.Game{{Season: 2026, Week: 8, SeasonType: 2, Status: "in"}}, lines) {
		t.Fatal("acquisition failed")
	}
	if lines["0099"].Line.Quantities["passing_yards"] != rows[id].Quantities["passing_yards"] || lines["0099"].Provenance != source.Sleeper {
		t.Fatalf("captured statistics lost: %+v", lines)
	}
}

type capturedSleeperHistory struct {
	livePollHistory
	id string
}

func (h *capturedSleeperHistory) SourceIDs(context.Context, string) (map[string]string, error) {
	return map[string]string{"0099": h.id}, nil
}

func TestCapturedScoreboardReachesLiveReadWithoutLegacyWindows(t *testing.T) {
	body := capturedLiveFeed(t, "espn-week8-live")
	games, err := livegame.ParseScoreboard(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 14 || games[0].Status != "in" || games[0].Week != 8 {
		t.Fatalf("captured board: %+v", games)
	}
	at := games[0].Kickoff.Add(30 * time.Minute)
	held := &Held{}
	publications := 0
	var poll LiveStatsPoller
	// Stop stat acquisition after the board: clocks must survive even an identity outage.
	poll.poll(t.Context(), LiveStatsRun{Season: 2026, Week: func() int { return 8 },
		Network: livePollTransport{body: string(body)}, Board: func(week int, board livegame.Held) {
			publications++
			board.At = at
			held.PublishScoreboard(week, board)
		}}, &livePollHistory{fail: true})
	if publications != 1 {
		t.Fatal("scoreboard not published")
	}
	live := held.SnapshotLive()
	view, err := RefreshLiveGame(t.Context(), &live, true, "", "", LiveSource{
		Start: func(*livegame.Adapter) error { return nil }, Now: func() time.Time { return at },
		Week: func() int { return 8 }, Windows: func() []livegame.Window { return nil },
		Poll:   func(context.Context, *LiveGame, time.Time, bool) {},
		Spines: func(context.Context, *LiveGameView, livegame.Adapter, string) error { return nil },
	})
	if err != nil || !view.Active || len(view.Games) != 14 || view.Games[0].Clock != "15:00" || view.Games[0].Period != 2 {
		t.Fatalf("live board lost: %+v %v", view, err)
	}
	// Publication and snapshots cannot alias the worker's mutable data.
	live.Adapter.Board.Games[0].Clock = "0:00"
	if held.SnapshotLive().Adapter.Board.Games[0].Clock != "15:00" {
		t.Fatal("snapshot aliased board")
	}
}

func TestSharedScoreboardPreservesDetailsAndRejectsOlderPublication(t *testing.T) {
	at := time.Now()
	held := &Held{}
	held.PublishLive(LiveGame{Week: 8, Adapter: livegame.Adapter{
		Board: livegame.Held{At: at}, Details: map[string]livegame.Detail{"1": {At: at}},
	}})
	board := livegame.Held{At: at.Add(time.Second), Source: "ESPN", Games: []livegame.Game{{ID: "1", Clock: "3:00"}}}
	held.PublishScoreboard(8, board)
	board.Games[0].Clock = "0:00"
	live := held.SnapshotLive()
	if live.Adapter.Board.Games[0].Clock != "3:00" || live.Adapter.Details["1"].At != at {
		t.Fatal("publication aliased board or lost followed details")
	}
	held.PublishScoreboard(7, livegame.Held{At: at})
	if held.SnapshotLive().Week != 8 {
		t.Fatal("older publication reset week")
	}
	held.PublishScoreboard(9, livegame.Held{At: at.Add(2 * time.Second)})
	live = held.SnapshotLive()
	if live.Week != 9 || len(live.Adapter.Details) != 0 {
		t.Fatal("new week retained old details")
	}
}
