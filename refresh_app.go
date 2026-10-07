package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/salaryadjustments"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// refreshTimeout bounds one MFL refresh: the league, rules, rosters and salary adjustments.
const refreshTimeout = 2 * time.Minute

// RefreshResult reports one pull of MFL truth. Changed is false when MFL held nothing new.
type RefreshResult struct {
	OK           bool   `json:"ok"`
	Error        string `json:"error"`
	Season       int    `json:"season"`
	RulesChanged bool   `json:"rulesChanged"`
	Changed      bool   `json:"changed"`
	Players      int    `json:"players"`
}

// discoverSource fetches the league config for MFL's current season; the rulebook seeds from it
// on a fresh database.
type discoverSource struct{ app *App }

func (s discoverSource) Fetch(ctx context.Context) (league.RawConfig, error) {
	cfg, err := league.Discover(ctx, s.app.mflClient, ingestion.LeagueID, s.app.seasonGuess())
	if err != nil {
		return league.RawConfig{}, fmt.Errorf("app: %w", err)
	}
	return cfg, nil
}

// seasonGuess is where season discovery starts: the season held, else the calendar year.
func (a *App) seasonGuess() int {
	if a.season != 0 {
		return a.season
	}
	return time.Now().Year()
}

// RefreshLeague pulls the league from MFL into the mirror (MFL wins, R2). The what-if league is
// left as it is.
func (a *App) RefreshLeague() RefreshResult {
	if err := a.ready(); err != nil {
		return RefreshResult{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	res, err := a.refreshLeague(ctx, a.rulebook, a.league)
	if err != nil {
		return RefreshResult{Error: err.Error()}
	}
	return res
}

// refreshLeague syncs the rules, then replaces the mirror with MFL's rosters and salary
// adjustments for the current season. Every fetch is archived by the transport.
func (a *App) refreshLeague(
	ctx context.Context, rb *rulebook.Store, mirror *state.Mirror,
) (RefreshResult, error) {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	return a.refreshLeagueLocked(ctx, rb, mirror)
}

// The caller holds refreshMu, including when the watcher refreshes several feeds.
func (a *App) refreshLeagueLocked(
	ctx context.Context, rb *rulebook.Store, mirror *state.Mirror,
) (RefreshResult, error) {
	cfg, err := league.Discover(ctx, a.mflClient, ingestion.LeagueID, a.seasonGuess())
	if err != nil {
		return RefreshResult{}, fmt.Errorf("refresh: %w", err)
	}
	rulesChanged, err := rb.Sync(ctx, cfg)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("refresh: rules: %w", err)
	}
	a.rulesCheckedAt.Store(time.Now().Unix())
	snap, err := a.fetchLeague(ctx, cfg.CurrentSeason)
	if err != nil {
		return RefreshResult{}, err
	}
	changed, err := mirror.Replace(ctx, snap)
	if err != nil {
		return RefreshResult{}, fmt.Errorf("refresh: %w", err)
	}
	a.rostersCheckedAt = time.Now().UTC()
	return RefreshResult{OK: true, Season: snap.Season, RulesChanged: rulesChanged, Changed: changed,
		Players: len(snap.Players)}, nil
}

// fetchLeague pulls one season's rosters and salary adjustments as a mirror snapshot. It needs no
// players directory, so MFL's once-a-day players limit never blocks a refresh.
func (a *App) fetchLeague(ctx context.Context, season int) (state.MirrorSnapshot, error) {
	year := strconv.Itoa(season)
	raws, err := rosters.Fetch(ctx, a.mflClient, year, ingestion.LeagueID)
	if err != nil {
		return state.MirrorSnapshot{}, fmt.Errorf("refresh: rosters: %w", err)
	}
	snap := state.MirrorSnapshot{Season: season}
	for _, raw := range raws {
		p, err := normalize.Contract(raw)
		if err != nil {
			return state.MirrorSnapshot{}, fmt.Errorf("refresh: rosters: %w", err)
		}
		snap.Players = append(snap.Players, state.MirrorPlayer{
			MFLID: p.MFLID.String(), FranchiseID: p.FranchiseID, RosterStatus: p.RosterStatus,
			Salary: p.Salary, ContractYear: p.ContractYear, ContractStatus: p.ContractStatus,
			ContractInfo: p.ContractInfo,
		})
	}
	adjs, err := salaryadjustments.Fetch(ctx, a.mflClient, year, ingestion.LeagueID)
	if err != nil {
		return state.MirrorSnapshot{}, fmt.Errorf("refresh: salary adjustments: %w", err)
	}
	for _, adj := range adjs {
		amt, err := domain.ParseSignedMoneyMillions(adj.Amount)
		if err != nil {
			return state.MirrorSnapshot{}, fmt.Errorf("refresh: salary adjustment %s: %w", adj.ID, err)
		}
		snap.Adjustments = append(snap.Adjustments, state.Adjustment{
			ID: adj.ID, FranchiseID: adj.FranchiseID, Amount: amt, Description: adj.Description})
	}
	return snap, nil
}

// refreshInBackground runs the launch refresh off the startup path, so an MFL outage never
// delays or blocks opening the app. It waits for startup without holding up the caller, Wails'
// message loop, and skips the refresh when startup failed or already refreshed. The result goes
// to the log.
func (a *App) refreshInBackground(parent context.Context) {
	a.weekStart.Do(func() {
		parent, a.weekCancel = context.WithCancel(parent)
		a.weekWorkers.Add(1)
		go func() {
			defer a.weekWorkers.Done()
			if a.ready() == nil {
				a.watchMoves(parent)
			}
		}()
		a.weekWorkers.Add(1)
		go func() {
			defer a.weekWorkers.Done()
			if a.ready() != nil {
				return
			}
			if a.launchRefreshDue {
				a.launchRefresh(parent)
			}
			a.refreshWeeksInBackground(parent)
		}()
	})
}

func (a *App) launchRefresh(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	res, err := a.refreshLeague(ctx, a.rulebook, a.league)
	if err != nil {
		log.Printf("the war room: launch refresh failed (the league held is kept): %v", err)
		return
	}
	log.Printf("the war room: launch refresh: season %d, %d players, league changed %t, rules changed %t",
		res.Season, res.Players, res.Changed, res.RulesChanged)
}
