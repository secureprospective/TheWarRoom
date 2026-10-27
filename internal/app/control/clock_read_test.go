package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type clockReadSource struct{}

func (clockReadSource) CurrentPhase(context.Context) (domain.Phase, error) {
	return domain.PhaseRegularSeason, nil
}
func (clockReadSource) CalendarEvents(context.Context) ([]state.CalendarEvent, error) {
	return nil, nil
}
func (clockReadSource) TradeDeadline(context.Context) (time.Time, error) { return time.Time{}, nil }

func TestClockReadAdmissionAndHeldMerge(t *testing.T) {
	failure := errors.New("unavailable")
	s := ClockRead{Ready: func() error { return failure }, StateError: func() error { t.Fatal("read state before ready"); return nil }}
	if _, err := s.Read(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	s.Ready = func() error { return nil }
	s.StateError = func() error { return failure }
	if _, err := s.Read(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	s.StateError = func() error { return nil }
	now := time.Now()
	merged := false
	s.Timeout = time.Minute
	s.Now = func() time.Time { return now }
	s.Season = func() int { return 2026 }
	s.Source = func() snapshot.ClockSource { return clockReadSource{} }
	s.Lineup = func(at time.Time) (*leagueclock.LineupLock, domain.Freshness) {
		if !at.Equal(now) {
			t.Fatal(at)
		}
		return nil, domain.Freshness{State: domain.FreshLive}
	}
	s.Windows = func(time.Time) leagueclock.WindowInputs { return leagueclock.WindowInputs{} }
	s.Merge = func(ctx context.Context, at time.Time, clock *snapshot.Sourced[leagueclock.Reading]) {
		if _, ok := ctx.Deadline(); !ok || !at.Equal(now) || clock.Value.Season != 2026 {
			t.Fatal("clock merge input", clock)
		}
		merged = true
	}
	if _, err := s.Read(context.Background()); err != nil || !merged {
		t.Fatal(err, merged)
	}
}
