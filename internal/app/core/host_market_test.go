package core_test

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

func TestMarketCaptureHoldsSeasonMu(t *testing.T) {
	s := coretest.New(t).Build()
	want := map[string]season.Feed[[]marketfeeds.Feed]{marketfeeds.ADP: {RefreshError: "offline"}}
	got := s.CaptureMarket(func() map[string]season.Feed[[]marketfeeds.Feed] {
		if s.SeasonMu.TryLock() {
			s.SeasonMu.Unlock()
			t.Error("capture lost the outer shared lock")
		}
		return want
	})
	if len(got) != 1 || got[marketfeeds.ADP].RefreshError != "offline" {
		t.Fatal(got)
	}
	if !s.SeasonMu.TryLock() {
		t.Fatal("capture retained SeasonMu")
	}
	s.SeasonMu.Unlock()
	s.Week = leagueweek.Week{Number: 6}
	if s.MarketWeek() != 6 || s.MarketZone() != time.UTC {
		t.Fatal("week or pinned zone")
	}
	s.LocalZone = nil
	if s.MarketZone() != time.Local {
		t.Fatal("unpinned zone is not local")
	}
}
