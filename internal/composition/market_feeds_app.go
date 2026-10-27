package composition

import (
	"context"
	"strconv"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
)

// MarketFact is one market cell of a pool row, the wire twin of marketapp.Fact; value is the
// number behind the text, for the pool's own lines.
type MarketFact struct {
	value  *float64
	Label  string       `json:"label"`
	Text   string       `json:"text"`
	Reason string       `json:"reason"`
	Range  *MarketRange `json:"range"`
}

func marketKinds() []string { return marketapp.Kinds() }

func (a *App) heldMarket() map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	return a.heldMarketForWeek(a.marketWeek())
}
func (a *App) heldMarketForWeek(week int) map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	feeds := a.core.CaptureMarket(a.marketHeld.SnapshotFeeds)
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, feed := range marketapp.ForWeek(feeds, week) {
		held[kind] = heldSeasonFeed[[]marketfeeds.Feed](feed)
	}
	return held
}

func (a *App) marketWeek() int { return a.core.MarketWeek() }

func (a *App) refreshMarketFeeds(ctx context.Context, force bool) {
	week := a.marketWeek()
	held := a.heldMarketForWeek(week)
	marketapp.RefreshFeeds(ctx, force, week, marketHeld(held), time.Since,
		func(ctx context.Context, wait time.Duration) error {
			pause := a.core.SeasonPause
			if pause == nil {
				pause = seasonPause
			}
			return pause(ctx, wait)
		}, a.refreshMarketFeed)

}

func (a *App) refreshMarketFeed(ctx context.Context, kind string, week int) {
	rows, fresh, err := marketapp.ReadFeed(ctx, a.fallbackParent(), a.core.Season, kind, week,
		func(ctx context.Context) (marketfeeds.Feed, error) {
			return marketfeeds.Fetch(ctx, a.core.MFLClient, kind, strconv.Itoa(a.core.Season), ingestion.LeagueID, week)
		}, a.core.History.ArchivedBodies, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	held := a.marketHeld.SnapshotFeeds()[kind]
	a.marketHeld.PublishFeed(kind, marketapp.RefreshedFeed(kind, week, held, rows, fresh, err))

}

// buildMarketReports builds the board in the feature service and converts it to its wire type.
func buildMarketReports(held map[string]heldSeasonFeed[[]marketfeeds.Feed], names map[string]string, week int, zone *time.Location) ReportBoard {
	return wireBoard(marketapp.Reports(marketHeld(held), names, week, zone))
}

func marketFacts(held map[string]heldSeasonFeed[[]marketfeeds.Feed]) map[string][]MarketFact {
	return marketFactsAt(held, time.Local)
}

// marketFactsAt is every listed player's market cells, built in the service, as wire facts.
func marketFactsAt(held map[string]heldSeasonFeed[[]marketfeeds.Feed], zone *time.Location) map[string][]MarketFact {
	built := marketapp.Facts(marketHeld(held), zone)
	out := make(map[string][]MarketFact, len(built))
	for id, facts := range built {
		out[id] = make([]MarketFact, len(facts))
		for i, f := range facts {
			out[id][i] = wireFact(f)
		}
	}
	return out
}

func wireFact(f marketapp.Fact) MarketFact {
	return MarketFact{value: f.Value, Label: f.Label, Text: f.Text, Reason: f.Reason, Range: wireRange(f.Range)}
}

func marketWeekText(week int) string { return marketapp.WeekText(week) }

func marketHeld(held map[string]heldSeasonFeed[[]marketfeeds.Feed]) map[string]marketapp.HeldFeed {
	out := make(map[string]marketapp.HeldFeed, len(held))
	for kind, feed := range held {
		out[kind] = marketapp.HeldFeed{Value: feed.Value, Feed: feed.FeedState()}
	}
	return out
}
