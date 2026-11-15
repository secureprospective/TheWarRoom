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
