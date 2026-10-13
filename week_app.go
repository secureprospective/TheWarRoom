package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

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
		a.clockChanged(parent)
		select {
		case <-parent.Done():
			return
		case <-ticker.C:
			a.launchRefresh(parent)
		}
	}
}

func (a *App) refreshWeek(ctx context.Context) {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	now := a.timeNow().UTC()
	w, err := a.fetchLineupWeek(ctx, now)
	fetched := time.Now().UTC()
	a.weekMu.Lock()
	if err != nil {
		a.weekRefreshError = err.Error()
	} else {
		a.week, a.weekFetchedAt, a.weekRefreshError = w, fetched, ""
	}
	a.weekMu.Unlock()
	if err != nil {
		log.Printf("the war room: NFL schedule refresh failed: %v", err)
	} else if w.Number != 0 {
		a.deriveWeekPhase(ctx, w, fetched)
	}
	a.revalidateReleaseAfterRefresh(ctx)
	a.clockChanged(ctx)
}

func (a *App) fetchLineupWeek(ctx context.Context, now time.Time) (leagueweek.Week, error) {
	year := strconv.Itoa(a.league.Season())
	for requested := 0; ; {
		raw, err := nflschedule.Fetch(ctx, a.mflClient, year, requested)
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

func (a *App) deriveWeekPhase(ctx context.Context, w leagueweek.Week, fetched time.Time) {
	cfg := a.rulebook.ActiveConfig()
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
	current, err := a.whatif.CurrentPhase(ctx)
	if err != nil {
		log.Printf("the war room: derive phase: read current: %v", err)
		return
	}
	if phase == current {
		return
	}
	note := fmt.Sprintf("derived from NFL week %d (bounds %d/%d/%d), schedule fetched %s",
		w.Number, bounds.Start, bounds.LastRegular, bounds.End, fetched.Format(time.RFC3339))
	if _, err := a.coordinator.Execute(ctx, transactions.AdvancePhase{To: phase, Note: note}); err != nil {
		log.Printf("the war room: derive phase: advance: %v", err)
	}
}

// weekClock is the clock's lineup lock and provenance from the held week. A missing or failed
// schedule degrades only the lineup part: the phase and calendar still read, so it never marks
// the clock failed (the strip would say "Clock unavailable").
func (a *App) weekClock(now time.Time) (*leagueclock.LineupLock, Freshness) {
	a.weekMu.Lock()
	defer a.weekMu.Unlock()
	fresh := liveFreshness(now)
	if !a.weekFetchedAt.IsZero() {
		fresh = liveFreshness(a.weekFetchedAt)
	}
	if a.weekRefreshError != "" {
		fresh.State = FreshStale
	}
	fresh.Note = a.weekNote()
	if lock, ok := leagueweek.NextLock(now, a.week); ok {
		return &leagueclock.LineupLock{Week: a.week.Number, At: lock}, fresh
	}
	if a.week.Number != 0 && len(a.week.Games) > 0 {
		return &leagueclock.LineupLock{Week: a.week.Number, At: leagueweek.LastLock(a.week)}, fresh
	}
	return nil, fresh
}

// weekNote says what the held schedule is, or why there is none. The caller holds weekMu.
func (a *App) weekNote() string {
	held := "no lineup week"
	if a.week.Number != 0 {
		held = "week " + strconv.Itoa(a.week.Number)
	}
	fetched := a.weekFetchedAt.Format(time.RFC3339)
	switch {
	case a.weekRefreshError != "" && a.weekFetchedAt.IsZero():
		return "NFL schedule refresh failed: " + a.weekRefreshError
	case a.weekRefreshError != "":
		return fmt.Sprintf("NFL schedule refresh failed: %s; showing %s fetched %s",
			a.weekRefreshError, held, fetched)
	case a.weekFetchedAt.IsZero():
		return "NFL schedule not fetched yet"
	}
	return "NFL schedule: " + held + ", fetched " + fetched
}

func (a *App) lastRegularWeek() int {
	last, err := strconv.Atoi(a.rulebook.ActiveConfig().LastRegularSeasonWeek)
	if err != nil || last < 2 || last > 18 {
		return 0
	}
	return last
}

func (a *App) windowInputs(now time.Time) leagueclock.WindowInputs {
	in := leagueclock.WindowInputs{LastRegular: a.lastRegularWeek(), FirstKickoff: map[int]time.Time{}}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	for _, rule := range leagueclock.Rules(in.LastRegular) {
		for _, raw := range a.now.schedules[rule.Week].value {
			w, err := nflschedule.ToWeek(raw, now)
			if err == nil && w.Number == rule.Week && len(w.Games) > 0 {
				in.FirstKickoff[rule.Week] = leagueweek.FirstLock(w)
			}
		}
	}
	return in
}

// timeNow is the league clock's current time: the wall clock unless a test pins it.
func (a *App) timeNow() time.Time {
	if a.clockNow != nil {
		return a.clockNow()
	}
	return time.Now()
}
