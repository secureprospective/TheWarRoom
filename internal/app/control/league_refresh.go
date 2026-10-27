package control

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/salaryadjustments"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type LeagueRefreshResult struct {
	Season       int  `json:"season"`
	RulesChanged bool `json:"rulesChanged"`
	Changed      bool `json:"changed"`
	Players      int  `json:"players"`
}

// LeagueRefresh stages both held stores before their single atomic commit.
// The caller owns the refresh lock and checked-time/dirty publication.
type LeagueRefresh struct {
	Discover                 func(context.Context) (league.RawConfig, error)
	Fetch                    func(context.Context, int) (state.MirrorSnapshot, error)
	StageRules               func(league.RawConfig) (db.PreparedWrite, rulebook.CapPercents, bool, error)
	StageMirror              func(state.MirrorSnapshot, state.CapDiscounts) (db.PreparedWrite, bool, error)
	Commit                   func(context.Context, ...db.PreparedWrite) error
	RulesChecked             func(int64)
	Dirty                    func(bool)
	RostersChecked           func(time.Time)
	Now                      func() time.Time
	DraftReports, Revalidate func(context.Context)
}

func (s LeagueRefresh) Refresh(ctx context.Context) (LeagueRefreshResult, error) {
	res, err := s.Run(ctx)
	if err != nil {
		return res, err
	}
	s.DraftReports(ctx)
	s.Revalidate(ctx)
	return res, nil
}

func (s LeagueRefresh) Run(ctx context.Context) (LeagueRefreshResult, error) {
	cfg, err := s.Discover(ctx)
	if err != nil {
		return LeagueRefreshResult{}, fmt.Errorf("refresh: %w", err)
	}
	snap, err := s.Fetch(ctx, cfg.CurrentSeason)
	if err != nil {
		return LeagueRefreshResult{}, err
	}
	rulesWrite, caps, rulesChanged, err := s.StageRules(cfg)
	if err != nil {
		return LeagueRefreshResult{}, fmt.Errorf("refresh: rules: %w", err)
	}
	mirrorWrite, changed, err := s.StageMirror(snap, caps)
	if err != nil {
		rulesWrite.Release()
		return LeagueRefreshResult{}, fmt.Errorf("refresh: %w", err)
	}
	if err := s.Commit(ctx, rulesWrite, mirrorWrite); err != nil {
		return LeagueRefreshResult{}, fmt.Errorf("refresh: %w", err)
	}
	s.RulesChecked(s.Now().Unix())
	if changed || rulesChanged {
		s.Dirty(true)
	}
	s.RostersChecked(s.Now().UTC())
	return LeagueRefreshResult{Season: snap.Season, RulesChanged: rulesChanged, Changed: changed,
		Players: len(snap.Players)}, nil
}

func PullLeague(ctx context.Context, season int, readRosters func(context.Context, int) ([]rosters.RawRoster, error), readAdjustments func(context.Context, int) ([]salaryadjustments.RawSalaryAdjustment, error)) (state.MirrorSnapshot, error) {
	raws, err := readRosters(ctx, season)
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
	adjs, err := readAdjustments(ctx, season)
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

type BackgroundRefresh struct {
	Ready                       func() error
	Start                       func(func())
	Due                         func() bool
	Clock, Watch, Launch, Weeks func(context.Context)
}

func (s BackgroundRefresh) StartWorkers(ctx context.Context) {
	s.Start(func() {
		if s.Ready() == nil {
			s.Clock(ctx)
		}
	})
	s.Start(func() {
		if s.Ready() == nil {
			s.Watch(ctx)
		}
	})
	s.Start(func() {
		if s.Ready() != nil {
			return
		}
		if s.Due() {
			s.Launch(ctx)
		}
		s.Weeks(ctx)
	})
}

func LaunchLeagueRefresh(parent context.Context, timeout time.Duration,
	refresh func(context.Context) (LeagueRefreshResult, error), dirty func(bool), logLine func(string)) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	res, err := refresh(ctx)
	if err != nil {
		logLine(fmt.Sprintf("the war room: launch refresh failed (the league held is kept): %v", err))
		return
	}
	if res.Changed || res.RulesChanged {
		dirty(true)
	}
	logLine(fmt.Sprintf("the war room: launch refresh: season %d, %d players, league changed %t, rules changed %t", res.Season, res.Players, res.Changed, res.RulesChanged))
}
