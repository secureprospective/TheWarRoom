package core

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (s *State) OffersContext() context.Context { return s.Ctx }
func (s *State) OffersPools() *db.Pools         { return s.Pools }
func (s *State) OffersMoves() *moves.Store      { return s.Moves }
func (s *State) OffersSeason() int              { return s.League.Season() }
func (s *State) OffersNow() time.Time           { return s.MovesNow() }
func (s *State) OffersChanged()                 { s.MovesChanged(s.Ctx) }
func (s *State) OffersZone() *time.Location {
	if s.LocalZone != nil {
		return s.LocalZone
	}
	return time.Local
}

func (s *State) OffersFreeAgents() report.HeldFeed[[]playerid.PlayerID] {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	feed := s.SeasonFreeAgents
	feed.Value = slices.Clone(feed.Value)
	return feed
}

func (s *State) OffersLookup() normalize.Lookup {
	s.LookupMu.Lock()
	defer s.LookupMu.Unlock()
	return s.Lookup // Immutable lookup exposes no mutable storage.
}

func (s *State) OffersRoster() state.MirrorSnapshot {
	mirror, _, _ := s.League.ContractsView()
	return mirror
}

func (s *State) OffersSnapshotNames() ([]snapshot.Player, []snapshot.Franchise, bool) {
	s.TargetSnapshotMu.Lock()
	defer s.TargetSnapshotMu.Unlock()
	players := slices.Clone(s.TargetSnapshot.Players.Value)
	for i := range players {
		players[i].Birthdate = clonePtr(players[i].Birthdate)
		players[i].IsRookie = clonePtr(players[i].IsRookie)
	}
	franchises := slices.Clone(s.TargetSnapshot.Franchises.Value)
	for i := range franchises {
		franchises[i].CapUsed = clonePtr(franchises[i].CapUsed)
		franchises[i].CapRoom = clonePtr(franchises[i].CapRoom)
	}
	return players, franchises, s.HasTargetSnapshot
}

func (s *State) OffersWeek() (int, bool) {
	s.WeekMu.Lock()
	defer s.WeekMu.Unlock()
	return s.Week.Number, !s.WeekFetchedAt.IsZero()
}

func (s *State) OffersCapRoom(franchise string) *domain.Money {
	raw := strings.TrimSpace(s.Rulebook.GetSalaryCap())
	if raw == "" {
		return nil
	}
	limit, err := domain.ParseMoneyMillions(raw)
	used, ok := s.League.CapUsed(franchise)
	if err != nil || !ok {
		return nil
	}
	room := limit - used
	return &room
}

// The callback retains the original load/check/save/log/event span, including revalidation.
func (s *State) SerializeOffersMove(run func() error) error {
	s.MovesMu.Lock()
	defer s.MovesMu.Unlock()
	return run()
}
