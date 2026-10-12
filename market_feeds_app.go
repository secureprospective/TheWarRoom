package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

func marketKinds() []string {
	return []string{marketfeeds.Projections, marketfeeds.Ranks, marketfeeds.ADP, marketfeeds.AAV}
}

func (a *App) heldMarket() map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	return a.heldMarketForWeek(a.marketWeek())
}
func (a *App) heldMarketForWeek(week int) map[string]heldSeasonFeed[[]marketfeeds.Feed] {
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	held := map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	for kind, feed := range a.seasonMarket {
		if kind == marketfeeds.Projections && len(feed.value) == 1 && feed.value[0].Week != week {
			feed = heldSeasonFeed[[]marketfeeds.Feed]{}
		}
		held[kind] = feed
	}
	return held
}

func (a *App) marketWeek() int {
	a.weekMu.Lock()
	defer a.weekMu.Unlock()
	return a.week.Number
}

func (a *App) refreshMarketFeeds(ctx context.Context, force bool) {
	week := a.marketWeek()
	held := a.heldMarketForWeek(week)
	for _, kind := range marketKinds() {
		feed := held[kind]
		sameWeek := kind != marketfeeds.Projections || (len(feed.value) == 1 && feed.value[0].Week == week)
		if !force && sameWeek && feed.refreshError == "" && time.Since(feed.fetchedAt) < 24*time.Hour {
			continue
		}
		pause := a.seasonPause
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
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[marketfeeds.Feed]{
		export: kind, leagueless: kind != marketfeeds.Projections,
		fetch: func(ctx context.Context) ([]marketfeeds.Feed, error) {
			feed, ferr := marketfeeds.Fetch(ctx, a.mflClient, kind, strconv.Itoa(a.season), ingestion.LeagueID, week)
			if ferr != nil {
				return nil, fmt.Errorf("market refresh: %w", ferr)
			}
			return []marketfeeds.Feed{feed}, nil
		}, parse: parse,
		bodies: func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
			return a.history.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
				if !marketArchiveScope(src, kind, week) {
					return false
				}
				return accept(src, body, at)
			})
		},
	}, a.fallbackLog)
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if a.seasonMarket == nil {
		a.seasonMarket = map[string]heldSeasonFeed[[]marketfeeds.Feed]{}
	}
	held := a.seasonMarket[kind]
	if kind == marketfeeds.Projections && len(held.value) == 1 && held.value[0].Week != week {
		held = heldSeasonFeed[[]marketfeeds.Feed]{}
	}
	holdSeasonFeed(&held, rows, fresh, err)
	a.seasonMarket[kind] = held
}

func marketArchiveScope(src, kind string, week int) bool {
	u, err := url.Parse(src)
	if err != nil {
		return false
	}
	switch kind {
	case marketfeeds.Projections:
		return u.Query().Get("W") == strconv.Itoa(week)
	case marketfeeds.ADP:
		return u.Query().Get("PERIOD") == "ALL"
	default:
		return true
	}
}

func (a *App) TargetMarketReportsRefresh(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("market refresh: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	a.refreshMarketFeeds(ctx, force)
	return nil
}

func (a *App) TargetMarketReports() (ReportBoard, error) {
	if err := a.ready(); err != nil {
		return ReportBoard{}, fmt.Errorf("market reports: startup: %w", err)
	}
	week := a.marketWeek()
	held := a.heldMarketForWeek(week)
	a.lookupMu.Lock()
	lookup := a.lookup
	a.lookupMu.Unlock()
	return buildMarketReports(held, marketNames(held, lookup), week, time.Local), nil
}

// marketNames labels every player the held reports list: name, position and team for a player,
// and the name alone for a team unit (a defense, a kicking unit) that MFL-wide reports price.
func marketNames(held map[string]heldSeasonFeed[[]marketfeeds.Feed], lookup normalize.Lookup) map[string]string {
	names := map[string]string{}
	for _, feed := range held {
		if len(feed.value) != 1 {
			continue
		}
		for _, p := range feed.value[0].Players {
			id := p.ID.String()
			if f, ok := lookup.Facts(id); ok {
				names[id] = f.Name + " · " + string(f.Position) + " · " + f.Team
			} else if name, ok := lookup.Name(id); ok {
				names[id] = name + " · team unit"
			}
		}
	}
	return names
}
