package contracts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/contracts/contractstest"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/moves"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// fakeHost is a Host over real stores: its serializers are real mutexes in the root's order and it
// records each notice, wake and browser hand-off.
type fakeHost struct {
	*contractstest.Stores
	readyErr error
	stamp    string

	refreshMu, movesMu, whatifMu sync.Mutex

	mu      sync.Mutex
	snap    snapshot.Snapshot
	loaded  bool
	now     func() time.Time
	opened  []string
	notices atomic.Int64
	wakes   atomic.Int64
}

// planHost is the release fixture behind the API: franchise 0001 holds 0042–0046 at $10M, no
// directory snapshot is loaded and the plan clock is the wall clock in UTC.
func planHost(t *testing.T) (*API, *fakeHost) {
	t.Helper()
	h := &fakeHost{Stores: contractstest.Release(t), now: func() time.Time { return time.Now().UTC() }}
	return New(h), h
}

func (h *fakeHost) phase(t *testing.T, phase domain.Phase) {
	t.Helper()
	h.SetPhase(t, phase)
}

func (h *fakeHost) Ready() error                      { return h.readyErr }
func (h *fakeHost) ContractsContext() context.Context { return h.Ctx }
func (h *fakeHost) ContractsBuildStamp() string       { return h.stamp }
func (h *fakeHost) ContractsStores() (*state.Mirror, *state.Store, *db.Pools, *moves.Store, *rulebook.Store) {
	return h.League, h.Whatif, h.WhatifPools, h.Moves, h.Rulebook
}
func (h *fakeHost) ContractsNow() time.Time     { return h.now() }
func (h *fakeHost) NotifyMoves(context.Context) { h.notices.Add(1) }
func (h *fakeHost) WakeMoves()                  { h.wakes.Add(1) }
func (h *fakeHost) OpenMoveURL(_ context.Context, target string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.opened = append(h.opened, target)
}

func (h *fakeHost) ContractsDirectory() (snapshot.Snapshot, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.snap, h.loaded
}

func (h *fakeHost) SerializeContractsRead(fn func()) {
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	fn()
}

func (h *fakeHost) SerializeContractActs(fn func()) {
	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	h.movesMu.Lock()
	defer h.movesMu.Unlock()
	fn()
}

func (h *fakeHost) SerializeReleasePlans(fn func()) {
	h.whatifMu.Lock()
	defer h.whatifMu.Unlock()
	h.SerializeContractActs(fn)
}

func (h *fakeHost) SerializeWhatIf(fn func()) {
	h.whatifMu.Lock()
	defer h.whatifMu.Unlock()
	h.SerializeContractsRead(fn)
}

func (h *fakeHost) SerializeMoves(fn func()) {
	h.movesMu.Lock()
	defer h.movesMu.Unlock()
	fn()
}

func (h *fakeHost) ContractsLineups() (leaguefeed.Lineups, report.Feed) {
	return leaguefeed.Lineups{}, report.Feed{}
}
func (h *fakeHost) ContractsWeek() leagueweek.Week     { return leagueweek.Week{} }
func (h *fakeHost) ContractsLeagueHost() (string, int) { return "www47.myfantasyleague.com", 2026 }
func (h *fakeHost) RulesFreshness() domain.Freshness {
	return domain.Freshness{State: domain.FreshStale, Note: "stored league settings"}
}
func (h *fakeHost) ContractsSeason() (snapshot.Sourced[leaguefeed.Lineups], error) {
	return snapshot.Sourced[leaguefeed.Lineups]{}, nil
}
func (h *fakeHost) RosterSnapshot(string) (snapshot.Snapshot, error) {
	return snapshot.Snapshot{}, fmt.Errorf("personal: snapshot not loaded")
}
func (h *fakeHost) RosterFeeds(context.Context, int) (Scores, Scores, Byes) {
	return Scores{}, Scores{}, Byes{}
}
func (h *fakeHost) ReleaseTarget() envelope.Target {
	return envelope.Target{Kind: envelope.Mapped, URL: "https://www47.myfantasyleague.com/2026/options?L=14432&O=257"}
}
func (h *fakeHost) ValidateMoveURL(target string) error {
	if !strings.HasPrefix(target, "https://www47.myfantasyleague.com/") {
		return fmt.Errorf("move URL: HTTPS MFL host without user info required")
	}
	return nil
}

var _ Host = (*fakeHost)(nil)
