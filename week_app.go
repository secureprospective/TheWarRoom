package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

func (a *App) refreshWeeksInBackground(parent context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		if parent.Err() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(parent, refreshTimeout)
		a.refreshWeek(ctx)
		cancel()
		ctx, cancel = context.WithTimeout(parent, refreshTimeout)
		a.refreshSeason(ctx)
		cancel()
		for _, rule := range leagueclock.Rules(a.lastRegularWeek()) {
			if rule.Week == 0 {
				continue
			}
			ctx, cancel = context.WithTimeout(parent, refreshTimeout)
			a.refreshNowWeek(ctx, rule.Week, false)
			cancel()
		}
		ctx, cancel = context.WithTimeout(parent, refreshTimeout)
		a.refreshFinishedScores(ctx)
		cancel()
		a.refreshEngine(parent)
		a.core.ClockChanged(parent)
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
			a.launchRefresh(parent)
		}
	}
}

func (a *App) refreshWeek(ctx context.Context) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	now := a.timeNow().UTC()
	w, err := a.fetchLineupWeek(ctx, now)
	fetched := time.Now().UTC()
	a.core.WeekMu.Lock()
	if err != nil {
		a.core.WeekRefreshError = err.Error()
	} else {
		a.core.Week, a.core.WeekFetchedAt, a.core.WeekRefreshError = w, fetched, ""
	}
	a.core.WeekMu.Unlock()
	if err != nil {
		log.Printf("the war room: NFL schedule refresh failed: %v", err)
	} else if w.Number != 0 {
		a.deriveWeekPhase(ctx, w, fetched)
	}
	a.revalidateReleaseAfterRefresh(ctx)
	a.core.ClockChanged(ctx)
}

func (a *App) fetchLineupWeek(ctx context.Context, now time.Time) (leagueweek.Week, error) {
	year := strconv.Itoa(a.core.League.Season())
	return lineupWeekFrom(now, func(requested int) (nflschedule.RawWeek, error) {
		return nflschedule.Fetch(ctx, a.core.MFLClient, year, requested)
	})
}

func lineupWeekFrom(now time.Time, read func(int) (nflschedule.RawWeek, error)) (leagueweek.Week, error) {
	for requested := 0; ; {
		raw, err := read(requested)
		if err != nil {
			return leagueweek.Week{}, fmt.Errorf("week refresh: %w", err)
		}
		w, err := nflschedule.ToWeek(raw, now)
		if err != nil {
			return leagueweek.Week{}, fmt.Errorf("week refresh: convert: %w", err)
		}
		if w.Number > 18 || (requested != 0 && w.Number != requested) {
			return leagueweek.Week{}, fmt.Errorf("week refresh: requested %d, received %d", requested, w.Number)
		}
		if lineup, ok := leagueweek.LineupWeek(now, w); ok {
			return lineup, nil
		}
		if w.Number == 18 {
			return leagueweek.Week{}, nil
		}
		requested = w.Number + 1
	}
}

// seedCurrentWeek applies the live refresh's finality rule to held schedules, without fetching.
func (a *App) seedCurrentWeek(ctx context.Context) {
	var fetched time.Time
	w, err := lineupWeekFrom(a.timeNow().UTC(), func(requested int) (nflschedule.RawWeek, error) {
		week := ""
		if requested != 0 {
			week = strconv.Itoa(requested)
		}
		rows, at, found, err := readArchivedFeed(ctx, a.core.Season, archivedFeed[nflschedule.RawWeek]{
			Export: nflschedule.Export, Leagueless: true,
			Parse: func(body []byte) ([]nflschedule.RawWeek, error) {
				raw, err := nflschedule.Parse(body)
				if err != nil {
					return nil, fmt.Errorf("held NFL week: %w", err)
				}
				return []nflschedule.RawWeek{raw}, nil
			},
			Bodies: a.scopedBodies("W", week),
		})
		if err != nil {
			return nflschedule.RawWeek{}, fmt.Errorf("held NFL week: %w", err)
		}
		if !found || len(rows) != 1 {
			return nflschedule.RawWeek{}, fmt.Errorf("held NFL week: no schedule for week %d", requested)
		}
		fetched = at
		return rows[0], nil
	})
	if err != nil {
		a.core.FallbackLog(err.Error())
		return
	}
	a.core.WeekMu.Lock()
	defer a.core.WeekMu.Unlock()
	a.core.Week, a.core.WeekFetchedAt, a.core.WeekRefreshError = w, fetched, ""
}

func (a *App) deriveWeekPhase(ctx context.Context, w leagueweek.Week, fetched time.Time) {
	cfg := a.core.Rulebook.ActiveConfig()
	bounds, err := leagueweek.ParseBounds(cfg.StartWeek, cfg.LastRegularSeasonWeek, cfg.EndWeek)
	if err != nil {
		log.Printf("the war room: derive phase: %v", err)
		return
	}
	phase, err := leagueweek.Phase(w.Number, bounds)
	if errors.Is(err, leagueweek.ErrSeasonOver) {
		log.Printf("the war room: derive phase: rollover needs a person: %v", err)
		return
	}
	if err != nil {
		log.Printf("the war room: derive phase: %v", err)
		return
	}
	current, err := a.core.Whatif.CurrentPhase(ctx)
	if err != nil {
		log.Printf("the war room: derive phase: read current: %v", err)
		return
	}
	if phase == current {
		return
	}
	note := fmt.Sprintf("derived from NFL week %d (bounds %d/%d/%d), schedule fetched %s",
		w.Number, bounds.Start, bounds.LastRegular, bounds.End, fetched.Format(time.RFC3339))
	if _, err := a.core.Coordinator.Execute(ctx, transactions.AdvancePhase{To: phase, Note: note}); err != nil {
		log.Printf("the war room: derive phase: advance: %v", err)
	}
}

func (a *App) weekSnapshot() controlapp.WeekSnapshot {
	a.core.WeekMu.Lock()
	defer a.core.WeekMu.Unlock()
	w := a.core.Week
	w.Games = slices.Clone(w.Games)
	return controlapp.WeekSnapshot{Week: w, FetchedAt: a.core.WeekFetchedAt, RefreshError: a.core.WeekRefreshError}
}

func (a *App) weekClock(now time.Time) (*leagueclock.LineupLock, Freshness) {
	return controlapp.WeekClock(a.weekSnapshot(), now)
}

func (a *App) lastRegularWeek() int {
	last, err := strconv.Atoi(a.core.Rulebook.ActiveConfig().LastRegularSeasonWeek)
	if err != nil || last < 2 || last > 18 {
		return 0
	}
	return last
}

func (a *App) windowSnapshot() controlapp.WindowSnapshot {
	in := controlapp.WindowSnapshot{LastRegular: a.lastRegularWeek(), Schedules: map[int][]nflschedule.RawWeek{}}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	for _, rule := range leagueclock.Rules(in.LastRegular) {
		rows := slices.Clone(a.now.schedules[rule.Week].Value)
		for i := range rows {
			rows[i].Matchups = slices.Clone(rows[i].Matchups)
		}
		in.Schedules[rule.Week] = rows
	}
	return in
}

func (a *App) windowInputs(now time.Time) leagueclock.WindowInputs {
	return controlapp.WindowInputs(a.windowSnapshot(), now)
}

// timeNow is the league clock's current time: the wall clock unless a test pins it.
func (a *App) timeNow() time.Time {
	if a.core.ClockNow != nil {
		return a.core.ClockNow()
	}
	return time.Now()
}
