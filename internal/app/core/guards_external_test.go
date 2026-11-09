package core_test

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

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

func TestOffersFeedPublicationRacesDetachedReads(t *testing.T) {
	s := coretest.New(t).Build()
	ids := make([]playerid.PlayerID, 2)
	for i, raw := range []string{"0042", "13593"} {
		id, err := playerid.New(raw)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 500 {
			s.SeasonMu.Lock()
			s.SeasonFreeAgents = core.HeldFeed[[]playerid.PlayerID]{Value: slices.Clone(ids), FetchedAt: time.Now()}
			s.SeasonMu.Unlock()
		}
	})
	for range 3 {
		workers.Go(func() {
			for range 500 {
				feed := s.OffersFreeAgents()
				if len(feed.Value) > 0 {
					if !slices.Equal(feed.Value, ids) {
						t.Error("publication torn")
					}
					feed.Value[0] = playerid.PlayerID{}
				}
			}
		})
	}
	workers.Wait()
}
