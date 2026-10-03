package state

import (
	"sync"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// leagueView is a league held in memory: each franchise's players and cap, indexed by player.
// Store and Mirror both serve their reads from one, replaced whole under mu.
type leagueView struct {
	mu         sync.RWMutex
	franchises map[string]*FranchiseState
	byPlayer   map[string]string
}

func newLeagueView() leagueView {
	return leagueView{franchises: map[string]*FranchiseState{}, byPlayer: map[string]string{}}
}

// FranchiseState returns a deep copy of one franchise's state.
func (s *leagueView) FranchiseState(franchiseID string) (FranchiseState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return FranchiseState{}, false
	}
	return cloneFranchise(fs), true
}

// Roster returns a deep copy of one franchise's players.
func (s *leagueView) Roster(franchiseID string) ([]PlayerState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return nil, false
	}
	return clonePlayers(fs.Players), true
}

// CapUsed returns one franchise's derived cap usage.
func (s *leagueView) CapUsed(franchiseID string) (domain.Money, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fs, ok := s.franchises[franchiseID]
	if !ok {
		return 0, false
	}
	return fs.CapUsed, true
}

// Player returns a copy of one player's state; ok is false if unrostered.
func (s *leagueView) Player(mflID string) (PlayerState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fid, ok := s.byPlayer[mflID]
	if !ok {
		return PlayerState{}, false
	}
	for _, p := range s.franchises[fid].Players {
		if p.MFLID == mflID {
			return p, true
		}
	}
	return PlayerState{}, false
}

// Franchises lists the franchise ids held, sorted. A franchise exists here only while it holds a
// player or carries cap charges; nothing asserts there are always 32.
func (s *leagueView) Franchises() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return sortedKeys(s.franchises)
}
