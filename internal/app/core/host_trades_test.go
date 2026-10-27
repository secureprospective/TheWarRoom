package core_test

import (
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func heldOffer(t *testing.T) leaguefeed.PendingTrade {
	t.Helper()
	id, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	return leaguefeed.PendingTrade{ID: "77", Offering: "0001", OfferedTo: "0025",
		Gives: []leaguefeed.Asset{{Player: &id}, {FuturePick: &leaguefeed.FuturePick{Franchise: "0001", Year: 2027, Round: 1}}},
		Gets:  []leaguefeed.Asset{{Player: &id}}, Expires: time.Date(2026, 10, 7, 16, 0, 0, 0, time.UTC)}
}

func TestPendingTradesDetachedAndNeverNil(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	if got := s.PendingTrades(); got.Value == nil || len(got.Value) != 0 || got.Provenance.Freshness.Note != "pending trades not fetched yet" {
		t.Fatal(got)
	}
	offer := heldOffer(t)
	input := []leaguefeed.PendingTrade{offer}
	s.SeasonPendingTrades = core.HeldFeed[[]leaguefeed.PendingTrade]{Value: input, FetchedAt: offer.Expires}
	before, err := json.Marshal(s.SeasonPendingTrades.Value)
	if err != nil {
		t.Fatal(err)
	}
	got := s.PendingTrades()
	if len(got.Value) != 1 || got.Provenance.Source != "pendingTrades" || got.Value[0].Gives[1].FuturePick.Year != 2027 {
		t.Fatal(got)
	}
	got.Value[0].ID = "mutated"
	got.Value[0].Gives[1].FuturePick.Year = 1999
	other, err := playerid.New("0100")
	if err != nil {
		t.Fatal(err)
	}
	*got.Value[0].Gets[0].Player = other
	got.Value[0].Gives[0] = leaguefeed.Asset{}
	after, err := json.Marshal(s.SeasonPendingTrades.Value)
	if err != nil || string(before) != string(after) {
		t.Fatalf("egress aliased the held feed:\n%s\n%s", before, after)
	}
	if again := s.PendingTrades(); again.Value[0].Gives[1].FuturePick.Year != 2027 || again.Value[0].Gets[0].Player.String() != "0099" {
		t.Fatal("second read saw the first read's mutation", again)
	}
}

func TestPendingTradesReadDuringPublication(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	offer := heldOffer(t)
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 200 {
			next := offer
			next.Comments = strconv.Itoa(i)
			s.SeasonMu.Lock()
			s.SeasonPendingTrades = core.HeldFeed[[]leaguefeed.PendingTrade]{Value: []leaguefeed.PendingTrade{next}, FetchedAt: offer.Expires}
			s.SeasonMu.Unlock()
		}
	})
	for range 200 {
		got := s.PendingTrades()
		if len(got.Value) == 1 {
			got.Value[0].Gives[1].FuturePick.Round = 9
		}
	}
	wg.Wait()
	if got := s.PendingTrades(); got.Value[0].Comments != "199" || got.Value[0].Gives[1].FuturePick.Round != 1 {
		t.Fatal(got)
	}
}

func TestTradeFuturePicksKeepNilAndDetach(t *testing.T) {
	t.Parallel()
	s := coretest.New(t).Build()
	if got := s.TradeFuturePicks(); got != nil {
		t.Fatal("nil picks became", got)
	}
	s.SeasonFuturePicks.Value = []draftreports.FuturePick{}
	if got := s.TradeFuturePicks(); got == nil {
		t.Fatal("empty picks became nil")
	}
	s.SeasonFuturePicks.Value = []draftreports.FuturePick{{Year: 2027, Round: 1, Original: "0001", Owner: "0001"}}
	s.TradeFuturePicks()[0].Owner = "0025"
	if s.SeasonFuturePicks.Value[0].Owner != "0001" {
		t.Fatal("picks aliased")
	}
	at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	s.SeasonFuturePicks.FetchedAt, s.SeasonFuturePicks.RefreshError = at, "offline"
	if fetched, refresh := s.TradeFuturePicksFeed(); !fetched.Equal(at) || refresh != "offline" {
		t.Fatal(fetched, refresh)
	}
}
