package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
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
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, feed := range a.core.SeasonMarket {
		if kind == marketfeeds.Projections && len(feed.Value) == 1 && feed.Value[0].Week != week {
			feed = heldSeasonFeed[[]marketfeeds.Feed]{}
		}
		held[kind] = feed
	}
	return held
}

func (a *App) marketWeek() int {
	a.core.WeekMu.Lock()
	defer a.core.WeekMu.Unlock()
	return a.core.Week.Number
}

func (a *App) refreshMarketFeeds(ctx context.Context, force bool) {
	week := a.marketWeek()
	held := a.heldMarketForWeek(week)
	for _, kind := range marketKinds() {
		feed := held[kind]
		sameWeek := kind != marketfeeds.Projections || (len(feed.Value) == 1 && feed.Value[0].Week == week)
		if !force && sameWeek && feed.RefreshError == "" && time.Since(feed.FetchedAt) < 24*time.Hour {
			continue
		}
		pause := a.core.SeasonPause
		if pause == nil {
			pause = seasonPause
		}
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		a.refreshMarketFeed(ctx, kind, week)
	}
}

func (a *App) refreshMarketFeed(ctx context.Context, kind string, week int) {
	parse := func(body []byte) ([]marketfeeds.Feed, error) {
		feed, err := marketfeeds.Parse(kind, body)
		if err != nil {
			return nil, fmt.Errorf("market archive: %w", err)
		}
		if kind == marketfeeds.Projections && feed.Week != week {
			return nil, fmt.Errorf("projection archive: wrong week")
		}
		return []marketfeeds.Feed{feed}, nil
	}
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[marketfeeds.Feed]{
		Export: kind, Leagueless: kind != marketfeeds.Projections,
		Fetch: func(ctx context.Context) ([]marketfeeds.Feed, error) {
			feed, ferr := marketfeeds.Fetch(ctx, a.core.MFLClient, kind, strconv.Itoa(a.core.Season), ingestion.LeagueID, week)
			if ferr != nil {
				return nil, fmt.Errorf("market refresh: %w", ferr)
			}
			return []marketfeeds.Feed{feed}, nil
		}, Parse: parse,
		Bodies: func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
			return a.core.History.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
				if !marketArchiveScope(src, kind, week) {
					return false
				}
				return accept(src, body, at)
			})
		},
	}, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if a.core.SeasonMarket == nil {
		a.core.SeasonMarket = map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	}
	held := a.core.SeasonMarket[kind]
	if kind == marketfeeds.Projections && len(held.Value) == 1 && held.Value[0].Week != week {
		held = heldSeasonFeed[[]marketfeeds.Feed]{}
	}
	holdSeasonFeed(&held, rows, fresh, err)
	a.core.SeasonMarket[kind] = held
}

func marketArchiveScope(src, kind string, week int) bool {
	return marketapp.ArchiveScope(src, kind, week)
}

func (a *App) TargetMarketReportsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("market refresh: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshMarketFeeds(ctx, force)
	return nil
}

func (a *App) TargetMarketReports() (ReportBoard, error) {
	if err := a.ready(); err != nil {
		return ReportBoard{}, fmt.Errorf("market reports: startup: %w", err)
	}
	week := a.marketWeek()
	held := a.heldMarketForWeek(week)
	a.core.LookupMu.Lock()
	lookup := a.core.Lookup
	a.core.LookupMu.Unlock()
	return buildMarketReports(held, marketNames(held, lookup), week, time.Local), nil
}

// marketNames labels every player the held reports list, through the service.
func marketNames(held map[string]heldSeasonFeed[[]marketfeeds.Feed], lookup normalize.Lookup) map[string]string {
	return marketapp.Names(marketHeld(held), lookup)
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
