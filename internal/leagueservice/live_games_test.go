package leagueservice

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
)

func TestLiveGamesHoldsADetachedBoardAndMarksItStale(t *testing.T) {
	at := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	clock := at
	games := &LiveGames{now: func() time.Time { return clock }}
	if board := games.Board(); len(board.Games) != 0 || board.Games == nil {
		t.Fatalf("empty holder must answer an empty, non-nil list: %+v", board)
	}
	held := []livegame.Game{{ID: "401", Status: "in", Home: livegame.Team{Abbreviation: "KC", Score: 7}}}
	games.Set(held, at, "ESPN")
	held[0].Home.Score = 99
	board := games.Board()
	if len(board.Games) != 1 || board.Games[0].Home.Score != 7 || board.Stale || board.Source != "ESPN" {
		t.Fatalf("board not held as set: %+v", board)
	}
	board.Games[0].Home.Score = 50
	if games.Board().Games[0].Home.Score != 7 {
		t.Fatal("a reader's copy leaked back into the holder")
	}
	clock = at.Add(liveGamesStale + time.Second)
	if !games.Board().Stale {
		t.Fatal("an old board must be labelled stale")
	}
	var none *LiveGames
	none.Set(held, at, "ESPN")
	if b := none.Board(); b.Games == nil || len(b.Games) != 0 {
		t.Fatalf("nil holder: %+v", b)
	}
}

func TestNFLGamesNeedsALeague(t *testing.T) {
	if _, err := New(nil).TargetNFLGames(""); err == nil {
		t.Fatal("empty league accepted")
	}
}

func TestNFLGameDetailsAreDetachedAndReplacedWithTheSlate(t *testing.T) {
	at := time.Date(2026, 11, 15, 18, 0, 0, 0, time.UTC)
	holder := &LiveGames{now: func() time.Time { return at.Add(liveGamesStale + time.Second) }}
	service := New(nil).WithLiveGames(holder)
	state := &livegame.State{Down: 2}
	game := livegame.Game{ID: "401", State: state, Plays: []livegame.Play{{ID: "1", End: state, Participants: []string{"1"}}}}
	holder.SetDetails(map[string]NFLGame{"401": {Game: &game, At: at}})
	game.State.Down = 4
	game.Plays[0].Participants[0] = "9"
	detail, err := service.TargetNFLGame("native.test", "401")
	if err != nil {
		t.Fatal(err)
	}
	if !detail.Stale || detail.Game.State.Down != 2 || detail.Game.Plays[0].Participants[0] != "1" {
		t.Fatalf("mutated detail: %+v", detail)
	}
	detail.Game.Plays[0].End.Down = 3
	reread, err := service.TargetNFLGame("native.test", "401")
	if err != nil {
		t.Fatal(err)
	}
	if reread.Game.Plays[0].End.Down != 2 {
		t.Fatal("reader changed held detail")
	}
	holder.SetDetails(map[string]NFLGame{})
	empty, err := service.TargetNFLGame("native.test", "401")
	if err != nil || empty.Game != nil || empty.Words == "" {
		t.Fatalf("old slate retained: %+v, %v", empty, err)
	}
}

func TestNFLGameDetailsClonePregameFields(t *testing.T) {
	at := time.Date(2026, 11, 22, 16, 0, 0, 0, time.UTC)
	holder := &LiveGames{now: func() time.Time { return at }}
	service := New(nil).WithLiveGames(holder)
	temp, precip := 66, 10
	game := livegame.Game{
		ID: "402", Status: "pre",
		Predictor: &livegame.Predictor{Home: 59.1, Away: 40.6},
		Venue:     &livegame.Venue{Name: "Highmark Stadium", City: "Orchard Park", State: "NY"},
		Weather: &livegame.Weather{
			Text: "Partly cloudy", TempF: &temp, PrecipPct: &precip, Source: "Open-Meteo",
		},
	}
	holder.SetDetails(map[string]NFLGame{"402": {Game: &game, At: at}})
	temp, precip = 99, 99
	game.Predictor.Home = 1
	game.Venue.Name = "Mutated"

	detail, err := service.TargetNFLGame("native.test", "402")
	if err != nil || detail.Game == nil {
		t.Fatalf("TargetNFLGame: %v %+v", err, detail)
	}
	if detail.Game.Predictor.Home != 59.1 || detail.Game.Venue.Name != "Highmark Stadium" ||
		*detail.Game.Weather.TempF != 66 || *detail.Game.Weather.PrecipPct != 10 {
		t.Fatalf("pregame fields mutated on set: %+v", detail.Game)
	}
	*detail.Game.Weather.TempF = 12
	reread, err := service.TargetNFLGame("native.test", "402")
	if err != nil || *reread.Game.Weather.TempF != 66 {
		t.Fatalf("reader mutated held weather: %v %+v", err, reread.Game.Weather)
	}
}
