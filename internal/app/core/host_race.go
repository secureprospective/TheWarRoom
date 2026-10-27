package core

import (
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func (s *State) RaceInput(snapshot func() season.Held) season.Snapshot {
	in := s.CaptureRace(snapshot)
	s.LookupMu.Lock()
	in.Lookup, in.HasLookup = s.Lookup, s.HasLookup
	s.LookupMu.Unlock()
	in.Config = s.Rulebook.ActiveConfig()
	in.Zone = s.LocalZone
	if in.Zone == nil {
		in.Zone = time.Local
	}
	return in
}

// CloneLineups detaches the shared feed while its caller holds SeasonMu.
func CloneLineups(in leaguefeed.Lineups) leaguefeed.Lineups {
	in.Matchups = slices.Clone(in.Matchups)
	in.Franchises = slices.Clone(in.Franchises)
	for i := range in.Franchises {
		f := &in.Franchises[i]
		f.Players = slices.Clone(f.Players)
		f.Starters = slices.Clone(f.Starters)
		f.NonStarters = slices.Clone(f.NonStarters)
	}
	return in
}

func (s *State) RaceRefreshState(snapshot func() season.Held) season.Held {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return snapshot()
}
