package core

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
)

// MarketContext is the app-lifetime context Market's held reads derive their bound from.
func (s *State) MarketContext() context.Context { return s.Ctx }

// LabFacts returns the shared Lab decode, or a fresh one for a State built without one.
func (s *State) LabFacts() (*lab.Data, error) {
	if s.LabData == nil {
		d, err := lab.Load()
		if err != nil {
			return nil, fmt.Errorf("load lab facts: %w", err)
		}
		return d, nil
	}
	return s.LabData()
}

// MarketWeek is the league week the market reports are read for.
func (s *State) MarketWeek() int {
	s.WeekMu.Lock()
	defer s.WeekMu.Unlock()
	return s.Week.Number
}

// CaptureMarket runs the holder's snapshot under SeasonMu, the lock every market report
// publication is made under. The snapshot returns detached copies.
func (s *State) CaptureMarket(snapshot func() map[string]season.Feed[[]marketfeeds.Feed]) map[string]season.Feed[[]marketfeeds.Feed] {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return snapshot()
}

// MarketLookup is the held players lookup; it is immutable and exposes no mutable storage.
func (s *State) MarketLookup() normalize.Lookup {
	s.LookupMu.Lock()
	defer s.LookupMu.Unlock()
	return s.Lookup
}

// MarketZone is the zone the market reports' times are written in.
func (s *State) MarketZone() *time.Location {
	if s.LocalZone == nil {
		return time.Local
	}
	return s.LocalZone
}

// RefreshMarket refreshes the market reports through the root's feed refresh, one MFL refresh at
// a time.
func (s *State) RefreshMarket(force bool) error {
	if err := s.Ready(); err != nil {
		return fmt.Errorf("market refresh: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(s.Ctx, 2*time.Minute)
	defer cancel()
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	s.RefreshMarketFeeds(ctx, force)
	return nil
}
