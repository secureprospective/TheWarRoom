package core

import (
	"context"
	"time"

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
