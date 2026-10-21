package core_test

import (
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func controlLineups() leaguefeed.Lineups {
	return leaguefeed.Lineups{Week: 5, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}},
		Franchises: []leaguefeed.Lineup{{Franchise: "0001", Players: []leaguefeed.PlayerScore{{Score: 1}}}}}
}

func TestControlSeasonFeedsDetachAndKeepNil(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	txs, lineups, week, trades := s.SeasonFeeds()
	if txs.Value != nil || trades.Value != nil || lineups.Value.Franchises != nil || week != 0 {
		t.Fatal("unheld feeds must stay nil; the API words them as lists")
	}
	s.SeasonMu.Lock()
	s.SeasonTransactions.Value = []leaguefeed.Transaction{{Kind: "TRADE"}}
	s.SeasonPendingTrades.Value = []leaguefeed.PendingTrade{}
	s.SeasonLineups.Value, s.LineupsWeek = controlLineups(), 5
	s.SeasonMu.Unlock()
	txs, lineups, week, trades = s.SeasonFeeds()
	if trades.Value == nil || len(trades.Value) != 0 || week != 5 {
		t.Fatal("an empty held feed must stay empty, not nil", trades.Value, week)
	}
	txs.Value[0].Kind = "changed"
	lineups.Value.Matchups[0].Home = "changed"
	lineups.Value.Franchises[0].Franchise = "changed"
	lineups.Value.Franchises[0].Players[0].Score = 9
	want := controlLineups()
	held := s.SeasonLineups.Value
	if s.SeasonTransactions.Value[0].Kind != "TRADE" || held.Matchups[0] != want.Matchups[0] ||
		held.Franchises[0].Franchise != "0001" || held.Franchises[0].Players[0].Score != 1 {
		t.Fatal("season feeds alias the held copies")
	}
}

func TestControlSeasonFeedsTakeTheSeasonLock(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	s.SeasonMu.Lock()
	done := make(chan struct{})
	go func() { s.SeasonFeeds(); close(done) }()
	select {
	case <-done:
		t.Fatal("season feeds read without SeasonMu")
	case <-time.After(50 * time.Millisecond):
	}
	s.SeasonMu.Unlock()
	<-done
}

func TestControlSeasonFeedsRacePublication(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 200 {
			s.SeasonMu.Lock()
			s.SeasonLineups = core.HeldFeed[leaguefeed.Lineups]{Value: controlLineups()}
			s.LineupsWeek = i
			s.SeasonTransactions.Value = append([]leaguefeed.Transaction{}, leaguefeed.Transaction{Kind: "ADD"})
			s.SeasonMu.Unlock()
		}
	})
	for range 4 {
		wg.Go(func() {
			for range 200 {
				txs, lineups, _, _ := s.SeasonFeeds()
				if len(lineups.Value.Franchises) > 0 {
					lineups.Value.Franchises[0].Players[0].Score = 7
				}
				if len(txs.Value) > 0 {
					txs.Value[0].Kind = "reader"
				}
			}
		})
	}
	wg.Wait()
	if s.SeasonLineups.Value.Franchises[0].Players[0].Score != 1 || s.SeasonTransactions.Value[0].Kind != "ADD" {
		t.Fatal("a reader's write reached the held feeds")
	}
}
