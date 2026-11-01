package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/fantasyscore/source"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// SyncScheduledScoring uses every league's ordinary published NFL season.
func SyncScheduledScoring(ctx context.Context, store *history.Store, transport *archive.Transport, league string, records []commissioner.Record) error {
	schedule, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return fmt.Errorf("read scoring season: %w", err)
	}
	if len(schedule.Weeks) == 0 || schedule.Season < 2021 {
		return nil
	}
	return SyncScoring(ctx, store, transport, schedule.Season)
}

// SyncScoring refreshes only the league season's scoring inputs, not the scouting backfill.
// The caller serializes it with Signals, which shares the weekly feed and load stamps.
func SyncScoring(ctx context.Context, store *history.Store, transport *archive.Transport, season int) error {
	if transport == nil {
		return fmt.Errorf("scoring acquisition needs the archived transport")
	}
	service := Service{Signals: store, Season: season, Transport: transport}
	reg := store.Registry()
	feed, ok := reg.Feed("stats_player_week")
	if !ok || season < feed.FirstSeason || season > 9999 {
		return fmt.Errorf("no weekly NFL statistics feed for season %d", season)
	}
	client := &http.Client{Timeout: RASFetchTimeout, Transport: transport}
	weekly := service.loadFeedFile(ctx, client, reg, feed, season)
	var weeklyErr error
	if weekly.Error != "" {
		weeklyErr = fmt.Errorf("weekly statistics: %s", weekly.Error)
	}
	return errors.Join(weeklyErr, service.loadScoringPlays(ctx, client, season))
}

func (a Service) loadScoringPlays(ctx context.Context, client *http.Client, season int) error {
	url := source.PlaysURL(season)
	due, err := a.due(ctx, url, season >= a.Season)
	if err != nil || !due {
		return err
	}
	body, err := ingestion.Get(ctx, client, url, 64<<20)
	if err != nil {
		return fmt.Errorf("fetch scoring plays: %w", err)
	}
	plays, err := source.ReadCompressedPlays(body)
	if err != nil {
		return fmt.Errorf("validate scoring plays: %w", err)
	}
	for _, play := range plays {
		if play.Season != fmt.Sprint(season) {
			return fmt.Errorf("scoring plays name a different season")
		}
	}
	hash := sha256.Sum256(body)
	_, err = a.Signals.Ingest(ctx, measures.Batch{Source: "nflverse", BodySHA256: hex.EncodeToString(hash[:])})
	if err != nil {
		return fmt.Errorf("record scoring plays load: %w", err)
	}
	return nil
}
