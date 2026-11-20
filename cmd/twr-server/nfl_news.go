package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflnews"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// nflNewsSource answers the news read from the public feed, using the reference store's id crosswalk
// to find each rostered player's Sleeper and ESPN ids.
type nflNewsSource struct {
	store *history.Store
	feed  *nflnews.Feed
}

// startHostedNews starts the public news feed for the server's lifetime. It fetches nothing until a
// league's news read names a rostered player.
func startHostedNews(ctx context.Context, store *history.Store) nflNewsSource {
	feed := nflnews.NewFeed(nil, time.Now)
	go nflnews.Run(ctx, feed)
	return nflNewsSource{store: store, feed: feed}
}

func (s nflNewsSource) ReadNFLNews(ctx context.Context, players []leagueservice.NFLNewsPlayer) (leagueservice.NFLNewsReading, error) {
	sleeperIDs, err := s.store.SourceIDs(ctx, "sleeper")
	if err != nil {
		return leagueservice.NFLNewsReading{}, fmt.Errorf("read Sleeper ids: %w", err)
	}
	espnIDs, err := s.store.SourceIDs(ctx, "espn")
	if err != nil {
		return leagueservice.NFLNewsReading{}, fmt.Errorf("read ESPN ids: %w", err)
	}
	asked := make([]nflnews.Player, 0, len(players))
	for _, p := range players {
		asked = append(asked, nflnews.Player{ID: p.Player, Name: p.Name, Position: p.Position,
			SleeperID: sleeperIDs[p.Player], ESPNID: espnIDs[p.Player]})
	}
	held := s.feed.Read(asked)
	out := leagueservice.NFLNewsReading{AsOf: held.AsOf, Stale: held.Stale, Items: make([]leagueservice.NFLNewsItem, 0, len(held.Items))}
	for _, item := range held.Items {
		out.Items = append(out.Items, leagueservice.NFLNewsItem{At: item.At, Player: item.Player, Name: item.Name,
			Position: item.Position, NFLTeam: item.NFLTeam, Tag: item.Tag, Text: item.Text, Source: item.Source})
	}
	return out, nil
}
