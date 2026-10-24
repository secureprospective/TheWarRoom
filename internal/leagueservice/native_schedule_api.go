package leagueservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
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

func (a *Service) TargetNativeFranchiseSchedule(raw RawNativeScheduleScope) (commissioner.FranchiseSchedule,
	error) {
	reading, _, err := a.nativeFranchiseSchedule(raw, a.host.CommissionerNow(), "schedule.franchise.read")
	return reading, err
}

func (a *Service) TargetNativeWeek(raw RawNativeWeekScope) (commissioner.NativeWeek, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.NativeWeek{}, err
	}
	schedule, weeks, err := a.nativeFranchiseSchedule(raw.RawNativeScheduleScope, raw.At, "week.read")
	if err != nil {
		return commissioner.NativeWeek{}, err
	}
	reading, err := commissioner.NativeWeekReading(schedule, raw.At, weeks)
	if err != nil {
		return commissioner.NativeWeek{}, fmt.Errorf("native week: %w", err)
	}
	return reading, nil
}

func (a *Service) nativeFranchiseSchedule(raw RawNativeScheduleScope, at time.Time, operation string) (
	commissioner.FranchiseSchedule, []leagueweek.Week, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.FranchiseSchedule{}, nil, err
	}
	if err := a.admitRead(raw.League, operation, raw.Franchise); err != nil {
		return commissioner.FranchiseSchedule{}, nil, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("native schedule startup: %w", err)
	}
	records, err := a.host.CommissionerRecords(a.viewContext(a.host.CommissionerContext()), raw.League)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.FranchiseSchedule{}, nil, err
	}
	zone, err := time.LoadLocation(raw.LocalZone)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("read owner time zone: %w", err)
	}
	weeks, err := a.host.NativeScheduleDates(a.viewContext(a.host.CommissionerContext()), raw.Season, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.FranchiseSchedule{}, nil, err
	}
	reading, err := commissioner.FranchiseScheduleReading(raw.League, raw.Franchise, raw.Season,
		records, weeks, zone, at)
	if err != nil {
		return commissioner.FranchiseSchedule{}, nil, fmt.Errorf("native franchise schedule: %w", err)
	}
	reading.Revision = commissioner.ViewRevision(a.viewContext(a.host.CommissionerContext()), records, reading.Revision)
	return reading, weeks, nil
}
