package history

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const autoLineupLead = time.Hour

// clockLineupAuthorizer is narrower than any human role: the lineup clock may only set lineups for
// franchises that have no human lineup for the week.
type clockLineupAuthorizer struct{}

func (clockLineupAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if principal != commissioner.LineupClockPrincipal || command != commissioner.LineupSet {
		return "", fmt.Errorf("the lineup clock can only set automatic lineups")
	}
	return "automatic", nil
}

func (c clockLineupAuthorizer) allowRecords(raw commissioner.RawCommand, principal string,
	records []commissioner.Record) (string, error) {
	role, err := c.Allow(raw.League, principal, raw.Command)
	if err != nil {
		return "", err
	}
	change, err := commissioner.DecodeLineupChange(raw.Args, raw.Command)
	if err != nil {
		return "", fmt.Errorf("decode automatic lineup: %w", err)
	}
	auto, err := commissioner.IsAutomaticLineupFranchise(records, change.Franchise, change.Season, change.Week)
	if err != nil {
		return "", fmt.Errorf("check automatic lineup eligibility: %w", err)
	}
	if !auto {
		return "", fmt.Errorf("franchise %s already has a human lineup for week %d", change.Franchise, change.Week)
	}
	return role, nil
}

type lineupWeekContext struct {
	season int
	week   leagueweek.Week
	rules  leaguerules.RuleSet
	waves  []time.Time
}

func (s *Store) lineupWeekState(ctx context.Context, league string, records []commissioner.Record,
	at time.Time) (lineupWeekContext, bool) {
	if len(records) == 0 || s.lineupEnv == nil {
		return lineupWeekContext{}, false
	}
	schedule, ok := s.lineupEnv.(interface {
		ScheduleDates(context.Context, int) ([]leagueweek.Week, error)
	})
	if !ok {
		return lineupWeekContext{}, false
	}
	base, err := commissioner.ReadNativeBase(league, records, at)
	if err != nil || !base.Assets.Seeded {
		return lineupWeekContext{}, false
	}
	instant := commissioner.ReadingInstant(records, at)
	season, err := commissioner.SeasonAt(records, instant)
	if err != nil {
		return lineupWeekContext{}, false
	}
	rules, err := commissioner.RulesInForce(records, season, instant)
	if err != nil || rules.Lineup.Lock.State != leaguerules.IsKnown {
		return lineupWeekContext{}, false
	}
	weeks, err := schedule.ScheduleDates(ctx, season)
	if err != nil {
		return lineupWeekContext{}, false
	}
	week, found := leagueweek.LineupWeek(at, weeks...)
	if !found || len(week.Games) == 0 {
		return lineupWeekContext{}, false
	}
	waves := commissioner.LineupWaveLockTimes(rules.Lineup.Lock.Value, week)
	if len(waves) == 0 {
		return lineupWeekContext{}, false
	}
	return lineupWeekContext{season: season, week: week, rules: rules, waves: waves}, true
}

// NextLineupDeadline returns the earliest auto-lineup wake time (kickoff minus 60 minutes) strictly
// after at for the league's current lineup week, or zero when none remains.
func (s *Store) NextLineupDeadline(ctx context.Context, league string, at time.Time) (time.Time, error) {
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return time.Time{}, err
	}
	state, ok := s.lineupWeekState(ctx, league, records, at)
	if !ok {
		return time.Time{}, nil
	}
	for _, kickoff := range state.waves {
		wake := kickoff.Add(-autoLineupLead)
		if wake.After(at) {
			return wake, nil
		}
	}
	return time.Time{}, nil
}

// ProcessDueLineups sets the best-projected healthy lineup for every AUTOMATIC franchise when a
// kickoff wave of the current lineup week is within 60 minutes. Locked starters are preserved.
func (s *Store) ProcessDueLineups(ctx context.Context, league string, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("stop lineup clock: %w", err)
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	state, ok := s.lineupWeekState(ctx, league, records, at)
	if !ok {
		return nil
	}
	wave, due := dueLineupWave(state.waves, at)
	if !due {
		return nil
	}
	env, err := s.weekFacts(ctx, state.season, state.week.Number)
	if err != nil {
		return err
	}
	if !env.ScheduleHeld {
		return nil
	}
	return s.applyDueLineups(ctx, league, records, state, wave, env, at)
}

// dueLineupWave returns the latest kickoff wave whose wake time (kickoff - 60m) has arrived while at
// least one wave remains unlocked.
func dueLineupWave(waves []time.Time, at time.Time) (time.Time, bool) {
	if len(waves) == 0 || !at.Before(waves[len(waves)-1]) {
		return time.Time{}, false
	}
	var chosen time.Time
	found := false
	for _, kickoff := range waves {
		if !at.Before(kickoff.Add(-autoLineupLead)) {
			chosen, found = kickoff, true
		}
	}
	return chosen, found
}

func (s *Store) applyDueLineups(ctx context.Context, league string, records []commissioner.Record,
	state lineupWeekContext, wave time.Time, env commissioner.LineupFacts, at time.Time) error {
	assets, franchises, err := autoLineupRoster(league, records, state.waves[0], at)
	if err != nil {
		return err
	}
	activeByTeam, allIDs := activePlayersByFranchise(assets.Snapshot.Players)
	if len(allIDs) == 0 {
		return nil
	}
	projMap, err := s.autoLineupProjections(ctx, league, state.season, state.week.Number, allIDs)
	if err != nil {
		return err
	}
	for _, franchise := range franchises {
		if err := s.setAutoLineupForFranchise(ctx, league, franchise, state, wave, env,
			activeByTeam[franchise], projMap, at); err != nil {
			return err
		}
	}
	return nil
}

func autoLineupRoster(league string, records []commissioner.Record, firstLock, at time.Time) (
	commissioner.NativeAssets, []string, error) {
	base, err := commissioner.ReadNativeBase(league, records, at)
	if err != nil {
		return commissioner.NativeAssets{}, nil, fmt.Errorf("read lineup base: %w", err)
	}
	assets := base.Assets
	if !at.Before(firstLock) {
		assets, err = commissioner.NativeAssetsAsOf(league, records, firstLock)
		if err != nil {
			return commissioner.NativeAssets{}, nil, fmt.Errorf("read kickoff roster: %w", err)
		}
	}
	franchises := make([]string, 0, len(base.Directory.Franchises))
	for _, f := range base.Directory.Franchises {
		franchises = append(franchises, f.Franchise)
	}
	return assets, franchises, nil
}

func activePlayersByFranchise(players []commissioner.NativePlayer) (map[string][]string, []string) {
	byTeam := map[string][]string{}
	seen := map[string]bool{}
	all := make([]string, 0, len(players))
	for _, p := range players {
		if p.Status != domain.RosterActive || p.Franchise == "" {
			continue
		}
		byTeam[p.Franchise] = append(byTeam[p.Franchise], p.ID)
		if !seen[p.ID] {
			seen[p.ID] = true
			all = append(all, p.ID)
		}
	}
	return byTeam, all
}

func (s *Store) autoLineupProjections(ctx context.Context, league string, season, week int,
	players []string) (map[string]NativeProjection, error) {
	if s.reg == nil {
		return map[string]NativeProjection{}, nil
	}
	if _, ok := s.reg.Feed("stats_player_week"); !ok {
		return map[string]NativeProjection{}, nil
	}
	reading, err := s.NativeProjections(ctx, league, season, week, players)
	if err != nil {
		return nil, fmt.Errorf("project automatic lineup players: %w", err)
	}
	out := make(map[string]NativeProjection, len(reading.Players))
	for _, p := range reading.Players {
		out[p.Player] = p
	}
	return out, nil
}

func (s *Store) setAutoLineupForFranchise(ctx context.Context, league, franchise string,
	state lineupWeekContext, wave time.Time, env commissioner.LineupFacts, roster []string,
	projMap map[string]NativeProjection, at time.Time) error {
	if len(roster) == 0 {
		return nil
	}
	records, err := s.nativeRecords(ctx, league)
	if err != nil {
		return err
	}
	auto, err := commissioner.IsAutomaticLineupFranchise(records, franchise, state.season, state.week.Number)
	if err != nil {
		return fmt.Errorf("check automatic franchise %s: %w", franchise, err)
	}
	if !auto {
		return nil
	}
	request := fmt.Sprintf("lineup-clock.%s.%d.w%d.%d.%s",
		league, state.season, state.week.Number, wave.UTC().Unix(), franchise)
	if hasLineupRequest(records, request) {
		return nil
	}
	held, found, err := commissioner.RecordedLineupFor(records, franchise, state.season, state.week.Number)
	if err != nil {
		return fmt.Errorf("read standing lineup for %s: %w", franchise, err)
	}
	candidates := buildAutoCandidates(roster, held.Starters, state.rules.Lineup.Lock.Value,
		state.waves[0], env, projMap, at)
	starters := nativecheck.SelectAutoLineup(state.rules.Lineup, candidates)
	if len(starters) == 0 || (found && sameStarterSet(held.Starters, starters)) {
		return nil
	}
	raw := commissioner.RawCommand{
		League:  league,
		Command: commissioner.LineupSet,
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{
			Season: state.season, Franchise: franchise, Week: state.week.Number, Starters: starters,
		}},
		Expected:   int64(len(records)),
		Request:    request,
		Effective:  "now",
		Visibility: "private",
	}
	if _, err := s.ExecuteCommissionerWithFacts(ctx, raw, commissioner.LineupClockPrincipal,
		clockLineupAuthorizer{}, env.Facts, env.Zone); err != nil {
		return fmt.Errorf("set automatic lineup for %s: %w", franchise, err)
	}
	return nil
}

func buildAutoCandidates(roster, standing []string, policy leaguerules.LineupLock, firstLock time.Time,
	env commissioner.LineupFacts, projMap map[string]NativeProjection,
	at time.Time) []nativecheck.AutoLineupCandidate {
	inStanding := make(map[string]bool, len(standing))
	for _, id := range standing {
		inStanding[id] = true
	}
	out := make([]nativecheck.AutoLineupCandidate, 0, len(roster))
	for _, id := range roster {
		f, _ := env.Facts.Of(id)
		kickoff, hasGame := playerGameKickoff(f, env.Locks)
		locked := playerLockedAt(policy, firstLock, kickoff, hasGame, at)
		proj := projMap[id]
		eligible := !locked && hasGame && !nativecheck.IneligibleDesignation(f.Designation)
		out = append(out, nativecheck.AutoLineupCandidate{
			ID:        id,
			Position:  f.Position,
			Median:    proj.Median,
			Ceiling:   proj.Ceiling,
			Available: proj.Available,
			Locked:    locked && inStanding[id],
			Eligible:  eligible,
		})
	}
	return out
}

func playerGameKickoff(f nativecheck.PlayerFacts, locks map[string]time.Time) (time.Time, bool) {
	if f.Team == "" || f.Team == "FA" {
		return time.Time{}, false
	}
	kickoff, ok := locks[f.Team]
	return kickoff, ok
}

func playerLockedAt(policy leaguerules.LineupLock, firstLock, kickoff time.Time,
	hasGame bool, at time.Time) bool {
	if policy == leaguerules.FirstKickoff {
		return !at.Before(firstLock)
	}
	return hasGame && !at.Before(kickoff)
}

func hasLineupRequest(records []commissioner.Record, request string) bool {
	for _, r := range records {
		if r.Command.Request == request {
			return true
		}
	}
	return false
}

func sameStarterSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa, sb := slices.Clone(a), slices.Clone(b)
	slices.Sort(sa)
	slices.Sort(sb)
	return slices.Equal(sa, sb)
}
