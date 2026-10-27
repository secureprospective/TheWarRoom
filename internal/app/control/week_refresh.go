package control

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type WeekRefresh struct {
	Now                 func() time.Time
	Read                func(context.Context, time.Time) (leagueweek.Week, error)
	Publish             func(leagueweek.Week, time.Time, error)
	Derive              func(context.Context, leagueweek.Week, time.Time)
	Revalidate, Changed func(context.Context)
	Log                 func(string)
}

func (s WeekRefresh) Run(ctx context.Context) {
	w, err := s.Read(ctx, s.Now().UTC())
	fetched := time.Now().UTC()
	s.Publish(w, fetched, err)
	if err != nil {
		s.Log(fmt.Sprintf("the war room: NFL schedule refresh failed: %v", err))
	} else if w.Number != 0 {
		s.Derive(ctx, w, fetched)
	}
	s.Revalidate(ctx)
	s.Changed(ctx)
}

func HoldWeek(held *leagueweek.Week, at *time.Time, failure *string, next leagueweek.Week, fetched time.Time, err error) {
	if err != nil {
		*failure = report.FailureText(err)
	} else {
		*held, *at, *failure = next, fetched, ""
	}
}

type WeekWorker struct {
	Timeout                                         time.Duration
	Week, Season, Finished, Engine, Changed, Launch func(context.Context)
	LastRegular                                     func() int
	Schedule                                        func(context.Context, int)
}

func (s WeekWorker) Run(parent context.Context, tick <-chan time.Time) {
	for {
		if parent.Err() != nil {
			return
		}
		s.bounded(parent, s.Week)
		s.bounded(parent, s.Season)
		for _, rule := range leagueclock.Rules(s.LastRegular()) {
			if rule.Week == 0 {
				continue
			}
			s.bounded(parent, func(ctx context.Context) { s.Schedule(ctx, rule.Week) })
		}
		s.bounded(parent, s.Finished)
		s.Engine(parent)
		s.Changed(parent)
		select {
		case <-parent.Done():
			return
		case <-tick:
			s.Launch(parent)
		}
	}
}

func (s WeekWorker) bounded(parent context.Context, run func(context.Context)) {
	ctx, cancel := context.WithTimeout(parent, s.Timeout)
	run(ctx)
	cancel()
}

func SeedWeek(now time.Time,
	read func(int) ([]nflschedule.RawWeek, time.Time, bool, error),
	publish func(leagueweek.Week, time.Time), log func(string)) {
	var fetched time.Time
	w, err := LineupWeekFrom(now.UTC(), func(requested int) (nflschedule.RawWeek, error) {
		rows, at, found, err := read(requested)
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
		log(err.Error())
		return
	}
	publish(w, fetched)
}

func DeriveWeekPhase(ctx context.Context, w leagueweek.Week, fetched time.Time, cfg league.RawConfig,
	read func(context.Context) (domain.Phase, error),
	execute func(context.Context, transactions.Request) (transactions.Receipt, error), log func(string)) {
	bounds, err := leagueweek.ParseBounds(cfg.StartWeek, cfg.LastRegularSeasonWeek, cfg.EndWeek)
	if err != nil {
		log(fmt.Sprintf("the war room: derive phase: %v", err))
		return
	}
	phase, err := leagueweek.Phase(w.Number, bounds)
	if errors.Is(err, leagueweek.ErrSeasonOver) {
		log(fmt.Sprintf("the war room: derive phase: rollover needs a person: %v", err))
		return
	}
	if err != nil {
		log(fmt.Sprintf("the war room: derive phase: %v", err))
		return
	}
	current, err := read(ctx)
	if err != nil {
		log(fmt.Sprintf("the war room: derive phase: read current: %v", err))
		return
	}
	if phase == current {
		return
	}
	note := fmt.Sprintf("derived from NFL week %d (bounds %d/%d/%d), schedule fetched %s", w.Number, bounds.Start, bounds.LastRegular, bounds.End, fetched.Format(time.RFC3339))
	if _, err := execute(ctx, transactions.AdvancePhase{To: phase, Note: note}); err != nil {
		log(fmt.Sprintf("the war room: derive phase: advance: %v", err))
	}
}
