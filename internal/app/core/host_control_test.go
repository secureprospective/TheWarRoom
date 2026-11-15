package core_test

import (
	"testing"

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
