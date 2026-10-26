package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
)

type MoveRefresh struct {
	Pause    func() func(context.Context, time.Duration) error
	Awaiting func(context.Context) ([]envelope.Envelope, error)
	Recent   func(envelope.Source, time.Time) bool
	Refresh  func(context.Context, envelope.Source) error
}

func (s MoveRefresh) Run(ctx context.Context, sources map[envelope.Source]bool) error {
	pause := s.Pause()
	entries, err := s.Awaiting(ctx)
	if err != nil {
		return fmt.Errorf("watch moves: latest hand-off: %w", err)
	}
	var latest time.Time
	for _, e := range entries {
		for _, entry := range e.Receipt().Audit {
			if entry.Event == envelope.HandOff && entry.At.After(latest) {
				latest = entry.At
			}
		}
	}
	first := true
	for _, source := range []envelope.Source{
		envelope.Rosters, envelope.Contracts, envelope.Transactions, envelope.Lineups, envelope.PendingTrades,
	} {
		if SkipMoveRefresh(source, sources) || s.Recent(source, latest) {
			continue
		}
		if !first {
			if err := pause(ctx, time.Second); err != nil {
				return fmt.Errorf("watch moves: spacing: %w", err)
			}
		}
		first = false
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("watch moves: refresh: %w", err)
		}
		if err := s.Refresh(ctx, source); err != nil {
			return err
		}
	}
	return nil
}

type MoveSourceRefresh struct {
	Override                      func(context.Context, envelope.Source) error
	Rosters                       func(context.Context) error
	Transactions, Lineups, Trades func(context.Context)
	Log                           func(string)
}

func (s MoveSourceRefresh) Run(ctx context.Context, source envelope.Source) error {
	if s.Override != nil {
		if err := s.Override(ctx, source); err != nil {
			return fmt.Errorf("watch moves: refresh %s: %w", source, err)
		}
		return nil
	}
	switch source {
	case envelope.Rosters, envelope.Contracts:
		if err := s.Rosters(ctx); err != nil {
			s.Log(fmt.Sprintf("watch moves: roster refresh: %v", err))
		}
	case envelope.Transactions:
		s.Transactions(ctx)
	case envelope.Lineups:
		s.Lineups(ctx)
	case envelope.PendingTrades:
		s.Trades(ctx)
	}
	return nil
}

type MoveSourceFeeds struct{ Transactions, Lineups, Pending report.Feed }

func (s MoveSourceFeeds) Recent(source envelope.Source, handed time.Time, now func() time.Time) bool {
	var feed report.Feed
	switch source {
	case envelope.Transactions:
		feed = s.Transactions
	case envelope.Lineups:
		feed = s.Lineups
	case envelope.PendingTrades:
		feed = s.Pending
	case envelope.Rosters, envelope.Contracts:
		return false
	default:
		return false
	}
	at := now()
	return feed.RefreshError == "" && feed.FetchedAt.After(handed) && !feed.FetchedAt.After(at) && at.Sub(feed.FetchedAt) < time.Minute
}

func ReadMoveObservation(ctx context.Context, sources map[envelope.Source]bool,
	refresh func(context.Context, map[envelope.Source]bool) error, read func(context.Context) (envelope.Observation, error)) (envelope.Observation, error) {
	if err := refresh(ctx, sources); err != nil {
		return envelope.Observation{}, err
	}
	return read(ctx)
}
