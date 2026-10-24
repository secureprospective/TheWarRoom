package composition

import (
	"context"
	"fmt"
	"log"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// This is a local reference read. It has no fetch path and never reads a fantasy league's schedule.
func (a *App) nativeScheduleDates(parent context.Context, season int, at time.Time) ([]leagueweek.Week,
	error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	var weeks []leagueweek.Week
	seen := make(map[int]bool)
	_, err := a.core.History.ReferenceBodies(ctx, "TYPE="+nflschedule.Export,
		func(src string, body []byte, _ time.Time) bool {
			if !sameExport(src, nflschedule.Export, season, true) {
				return false
			}
			raw, err := nflschedule.Parse(body)
			if err != nil {
				return false
			}
			week, err := nflschedule.ToWeek(raw, at)
			if err == nil && !seen[week.Number] {
				weeks = append(weeks, week)
				seen[week.Number] = true
			}
			return false
		})
	if err != nil {
		return nil, fmt.Errorf("read held NFL week dates: %w", err)
	}
	return weeks, nil
}

type appBidEnv struct{ app *App }

func (e appBidEnv) BidEnvironment(ctx context.Context, league string,
	at time.Time) (commissioner.BidEnvironment, error) {
	records, err := e.app.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return commissioner.BidEnvironment{}, fmt.Errorf("read the league season: %w", err)
	}
	weeks, err := e.app.nativeScheduleDates(ctx, season, at)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	phase := nativeapp.WeekAt(weeks, at)
	if !phase.WeekKnown && !phase.Offseason {
		phase.Offseason, err = commissioner.BidOffseasonAt(records, at, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding offseason: %w", err)
		}
	}
	field := map[string]nativecheck.PlayoffStanding{}
	if phase.WeekKnown {
		bracket, err := e.app.core.History.NativeBracket(ctx, league, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding playoff field: %w", err)
		}
		directory, err := commissioner.NativeDirectoryReading(league, records, at)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding franchises: %w", err)
		}
		field = nativeapp.PlayoffField(bracket, directory)
	}
	return commissioner.BidEnvironment{Season: phase, Playoff: field}, nil
}

type nativeDueClock struct{ now func() time.Time }

func (c nativeDueClock) Now() time.Time { return c.now() }
func (nativeDueClock) Trusted() error   { return nil }

func (a *App) nativeWaiverClock(ctx context.Context) {
	worker := leagueservice.NewDueWorker(a.core.History, a.nativeFacts, nativeDueClock{now: a.timeNow})
	worker.Run(ctx, func(err error) { log.Printf("the war room: process native deadlines: %v", err) })
}
