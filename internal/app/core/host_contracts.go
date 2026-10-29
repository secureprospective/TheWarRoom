package core

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func (s *State) ContractsContext() context.Context { return s.Ctx }

func (s *State) ContractsBuildStamp() string { return s.BuildStamp }

// ContractsStores are set by startup and lock internally; read them after Ready.
func (s *State) ContractsStores() (*state.Mirror, *state.Store, *db.Pools, *moves.Store, *rulebook.Store) {
	return s.League, s.Whatif, s.WhatifPools, s.Moves, s.Rulebook
}

func (s *State) ContractsNow() time.Time { return s.MovesNow() }

func (s *State) NotifyMoves(ctx context.Context) { s.MovesChanged(ctx) }

// WakeMoves asks the moves watcher for a pass without waiting for it.
func (s *State) WakeMoves() {
	select {
	case s.MovesWake <- struct{}{}:
	default:
	}
}

func (s *State) OpenMoveURL(ctx context.Context, target string) { s.OpenURL(ctx, target) }

// The serializers keep each contracts binding's original operation locks and their order.
func (s *State) SerializeContractsRead(fn func()) {
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	fn()
}

func (s *State) SerializeContractActs(fn func()) {
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	s.MovesMu.Lock()
	defer s.MovesMu.Unlock()
	fn()
}

func (s *State) SerializeReleasePlans(fn func()) {
	s.WhatifMu.Lock()
	defer s.WhatifMu.Unlock()
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	s.MovesMu.Lock()
	defer s.MovesMu.Unlock()
	fn()
}

func (s *State) SerializeWhatIf(fn func()) {
	s.WhatifMu.Lock()
	defer s.WhatifMu.Unlock()
	s.RefreshMu.Lock()
	defer s.RefreshMu.Unlock()
	fn()
}

func (s *State) SerializeMoves(fn func()) {
	s.MovesMu.Lock()
	defer s.MovesMu.Unlock()
	fn()
}

// ContractsDirectory is the held target snapshot, detached, and whether one is loaded.
func (s *State) ContractsDirectory() (snapshot.Snapshot, bool) {
	s.TargetSnapshotMu.Lock()
	defer s.TargetSnapshotMu.Unlock()
	return cloneSnapshot(s.TargetSnapshot), s.HasTargetSnapshot
}

// ContractsLineups is MFL's held saved lineups, detached, with the feed's read state.
func (s *State) ContractsLineups() (leaguefeed.Lineups, report.Feed) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	held := s.SeasonLineups
	out := held.Value
	out.Matchups = slices.Clone(out.Matchups)
	out.Franchises = slices.Clone(out.Franchises)
	for i := range out.Franchises {
		f := &out.Franchises[i]
		f.Players, f.Starters, f.NonStarters = slices.Clone(f.Players), slices.Clone(f.Starters), slices.Clone(f.NonStarters)
	}
	return out, held.FeedState()
}

// ContractsWeek is the held league week, detached.
func (s *State) ContractsWeek() leagueweek.Week {
	s.WeekMu.Lock()
	defer s.WeekMu.Unlock()
	return leagueweek.Week{Number: s.Week.Number, Games: slices.Clone(s.Week.Games)}
}

func (s *State) ContractsLeagueHost() (string, int) {
	host := ""
	if s.MFLClient != nil {
		host = s.MFLClient.LeagueHost()
	}
	return host, s.Season
}

func cloneSnapshot(in snapshot.Snapshot) snapshot.Snapshot {
	out := in
	out.League.Value.SalaryCap = clonePtr(in.League.Value.SalaryCap)
	out.Franchises.Value = slices.Clone(in.Franchises.Value)
	for i := range out.Franchises.Value {
		f := &out.Franchises.Value[i]
		f.CapUsed, f.CapRoom = clonePtr(f.CapUsed), clonePtr(f.CapRoom)
	}
	out.Rosters.Value = slices.Clone(in.Rosters.Value)
	for i := range out.Rosters.Value {
		r := &out.Rosters.Value[i]
		r.Players = slices.Clone(r.Players)
		for j := range r.Players {
			r.Players[j].YearsRemaining = clonePtr(r.Players[j].YearsRemaining)
		}
	}
	out.Players.Value = slices.Clone(in.Players.Value)
	for i := range out.Players.Value {
		p := &out.Players.Value[i]
		p.Birthdate, p.IsRookie = clonePtr(p.Birthdate), clonePtr(p.IsRookie)
	}
	out.Eligibility.Players = maps.Clone(in.Eligibility.Players)
	out.Eligibility.Feeds = maps.Clone(in.Eligibility.Feeds)
	return out
}

// PendingTrades copies the held pendingTrades feed under SeasonMu. The value is never nil, as
// before, and every offer's assets are detached from the held feed.
func (s *State) PendingTrades() snapshot.Sourced[[]leaguefeed.PendingTrade] {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	held := s.SeasonPendingTrades
	out := make([]leaguefeed.PendingTrade, len(held.Value))
	for i, t := range held.Value {
		t.Gives, t.Gets = cloneAssets(t.Gives), cloneAssets(t.Gets)
		out[i] = t
	}
	return snapshot.Sourced[[]leaguefeed.PendingTrade]{
		Value:      out,
		Provenance: held.Provenance("pendingTrades", "pending trades"),
	}
}

// TradeFuturePicks copies the held futureDraftPicks value under SeasonMu, keeping nil nil.
func (s *State) TradeFuturePicks() []draftreports.FuturePick {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return slices.Clone(s.SeasonFuturePicks.Value)
}

// TradeFuturePicksFeed reads when futureDraftPicks was last fetched and its last refresh error.
func (s *State) TradeFuturePicksFeed() (time.Time, string) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return s.SeasonFuturePicks.FetchedAt, s.SeasonFuturePicks.RefreshError
}
