package core

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (s *State) ResearchNames() map[string]string {
	return s.Rulebook.FranchiseNames()
}

func (s *State) ResearchDestination() (string, int) {
	return s.MFLClient.LeagueHost(), s.Season
}

func (s *State) ResearchDirectory() normalize.Lookup {
	s.LookupMu.Lock()
	lookup := s.Lookup // Immutable lookup; it exposes no mutable storage.
	s.LookupMu.Unlock()
	return lookup
}

func (s *State) ResearchRoster() state.MirrorSnapshot {
	mirror, _, _ := s.League.ContractsView()
	return mirror
}

func (s *State) ResearchConfig() league.RawConfig {
	return s.Rulebook.ActiveConfig()
}

func (s *State) ResearchZone() *time.Location {
	if s.LocalZone != nil {
		return s.LocalZone
	}
	return time.Local
}

func (s *State) ResearchContext() context.Context {
	return s.Ctx
}

func (s *State) CurrentWeek() int {
	s.WeekMu.Lock()
	defer s.WeekMu.Unlock()
	return s.Week.Number
}

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
