package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type RawNativeScheduleScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise"`
	Season    int    `json:"season"`
	LocalZone string `json:"localZone"`
}

func (r RawNativeScheduleScope) Validate() error {
	if err := (RawLeagueScheduleScope{League: r.League}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Franchise) == "" || strings.TrimSpace(r.Franchise) != r.Franchise ||
		r.Season < 1900 || r.Season > 9999 {
		return fmt.Errorf("choose a franchise and season")
	}
	if r.LocalZone == "" || r.LocalZone == "Local" {
		return fmt.Errorf("choose your local time zone")
	}
	if _, err := time.LoadLocation(r.LocalZone); err != nil {
		return fmt.Errorf("read owner time zone: %w", err)
	}
	return nil
}

type RawNativeWeekScope struct {
	RawNativeScheduleScope
	At time.Time `json:"at"`
}

func (r RawNativeWeekScope) Validate() error {
	if err := r.RawNativeScheduleScope.Validate(); err != nil {
		return err
	}
	if r.At.IsZero() {
		return fmt.Errorf("choose the time to read this week")
	}
	return nil
}

func (a *App) TargetNativeFranchiseSchedule(raw RawNativeScheduleScope) (commissioner.FranchiseSchedule, error) {
	reading, _, err := a.nativeFranchiseSchedule(raw, a.timeNow())
	return reading, err
}

func (a *App) TargetNativeWeek(raw RawNativeWeekScope) (commissioner.NativeWeek, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.NativeWeek{}, err
	}
	schedule, weeks, err := a.nativeFranchiseSchedule(raw.RawNativeScheduleScope, raw.At)
	if err != nil {
		return commissioner.NativeWeek{}, err
	}
	reading, err := commissioner.NativeWeekReading(schedule, raw.At, weeks)
	if err != nil {
		return commissioner.NativeWeek{}, fmt.Errorf("native week: %w", err)
	}
	return reading, nil
}

func (a *App) nativeFranchiseSchedule(raw RawNativeScheduleScope, at time.Time) (
	commissioner.FranchiseSchedule, []leagueweek.Week, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.FranchiseSchedule{}, nil, err
	}
	if err := a.ready(); err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("native schedule startup: %w", err)
	}
	records, err := a.leagueRecords(a.core.Ctx, raw.League)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, err
	}
	zone, err := time.LoadLocation(raw.LocalZone)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("read owner time zone: %w", err)
	}
	weeks, err := a.nativeScheduleDates(a.core.Ctx, raw.Season, at)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, err
	}
	reading, err := commissioner.FranchiseScheduleReading(raw.League, raw.Franchise, raw.Season,
		records, weeks, zone, at)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("native franchise schedule: %w", err)
	}
	return reading, weeks, nil
}

// This is a local reference read. It has no fetch path and never reads a fantasy league's schedule.
func (a *App) nativeScheduleDates(parent context.Context, season int, at time.Time) ([]leagueweek.Week, error) {
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

func (e appBidEnv) BidEnvironment(ctx context.Context, league string, at time.Time) (commissioner.BidEnvironment, error) {
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

func (a *App) processNativeBidsDue(ctx context.Context) {
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native bid leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.core.History.ProcessDueBids(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: award native bids for %s: %v", league.ID, err)
		}
	}
}

const draftLookEvery = 15 * time.Second

// nativeWaiverClock resumes persisted waiver windows, bid clocks and finished drafts without any network feed.

func (a *App) nativeWaiverClock(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastDrafts time.Time
	for {
		a.processNativeWaiversDue(ctx)
		a.processNativeBidsDue(ctx)
		// A finished draft is found by reading each draft, so that look is taken every few seconds.
		if time.Since(lastDrafts) >= draftLookEvery {
			lastDrafts = time.Now()
			a.processNativeDraftsDue(ctx)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *App) processNativeWaiversDue(ctx context.Context) {
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native waiver leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.core.History.ProcessDueWaivers(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: process native waivers for %s: %v", league.ID, err)
		}
	}
}

// A replay does not mutate a roster, so a closed calendar does not discard a recorded receipt.
