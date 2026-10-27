package control

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type SeasonRefresh struct {
	Changed                                                                                        func(context.Context)
	Transactions, Comms, Shell, Lineups, Trades, Injuries, Draft, Market, Race, Firsts, Revalidate func(context.Context)
	Pause                                                                                          func() func(context.Context, time.Duration) error
	Week                                                                                           func() int
	Schedule                                                                                       func(context.Context, int)
}

func (s SeasonRefresh) Run(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	defer s.Changed(ctx)
	s.Transactions(ctx)
	s.Comms(ctx)
	s.Shell(ctx)
	pause := s.Pause()
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	s.Lineups(ctx)
	s.Changed(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	s.Trades(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	s.Injuries(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	s.Draft(ctx)
	s.Market(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	s.Race(ctx)
	s.Firsts(ctx)
	if week := s.Week(); week >= 1 {
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		s.Schedule(ctx, week)
	}
	s.Revalidate(ctx)
}

func RefreshInjuries(ctx context.Context,
	read func(context.Context) ([]injuries.RawInjury, domain.Freshness, error),
	publish func([]leaguefeed.Injury, domain.Freshness, error)) {
	rows, fresh, err := read(ctx)
	var value []leaguefeed.Injury
	if err == nil {
		value, err = injuries.ToInjuries(rows)
	}
	publish(value, fresh, err)
}

func HeldInjuries(held report.HeldFeed[[]leaguefeed.Injury]) snapshot.Injuries {
	statuses := make(map[string]string, len(held.Value))
	for _, row := range held.Value {
		statuses[row.ID.String()] = row.Status
	}
	return snapshot.Injuries{Status: statuses, Provenance: held.Provenance("injuries", "injuries")}
}

func RefreshTransactions(ctx context.Context,
	read func(context.Context) ([]transactions.RawTransactions, domain.Freshness, error),
	publish func([]leaguefeed.Transaction, domain.Freshness, error)) {
	rows, fresh, err := read(ctx)
	var value []leaguefeed.Transaction
	if err == nil {
		value, err = transactions.ToTransactions(rows[0])
	}
	publish(value, fresh, err)
}

func CheckLineupWeek(raw livescoring.RawLineups, week int) error {
	if week != 0 && raw.Week != strconv.Itoa(week) {
		return fmt.Errorf("season lineups: requested week %d, received %s", week, raw.Week)
	}
	return nil
}

func LineupWeekParam(week int) string {
	if week == 0 {
		return ""
	}
	return strconv.Itoa(week)
}

func RefreshLineups(ctx context.Context,
	read func(context.Context) ([]livescoring.RawLineups, domain.Freshness, error),
	publish func(leaguefeed.Lineups, domain.Freshness, error)) {
	rows, fresh, err := read(ctx)
	var value leaguefeed.Lineups
	if err == nil {
		value, err = livescoring.ToLineups(rows[0])
	}
	publish(value, fresh, err)
}

func PublishLineups(held *report.HeldFeed[leaguefeed.Lineups], heldWeek *int, week int, value leaguefeed.Lineups, fresh domain.Freshness, err error) {
	if *heldWeek != week {
		*held = report.HeldFeed[leaguefeed.Lineups]{}
	}
	*heldWeek = week
	report.HoldFeed(held, value, fresh, err)
}

func PendingKey(ctx context.Context, store *mflkey.Store) error {
	if store == nil {
		return mflkey.ErrUnavailable
	}
	_, err := store.Get(ctx)
	return err //nolint:wrapcheck // Preserve key sentinel identity.
}

func RefreshPending(ctx context.Context, keyErr error,
	fetch func(context.Context) (pendingtrades.RawPendingTrades, error),
	publish func([]leaguefeed.PendingTrade, error)) {
	var value []leaguefeed.PendingTrade
	err := keyErr
	if err == nil {
		var raw pendingtrades.RawPendingTrades
		raw, err = fetch(ctx)
		if err == nil {
			value, err = pendingtrades.ToPendingTrades(raw)
		}
	}
	publish(value, err)
}

func ClearPending(held *report.HeldFeed[[]leaguefeed.PendingTrade]) {
	*held = report.HeldFeed[[]leaguefeed.PendingTrade]{RefreshError: "MFL not connected"}
}

// CurrentPending calls publication only while the caller holds the matching key generation.
func CurrentPending(generation, current uint64, publish func()) {
	if generation == current {
		publish()
	}
}

// PublishPending runs under both key and season locks.
func PublishPending(ctx context.Context, held *report.HeldFeed[[]leaguefeed.PendingTrade], value []leaguefeed.PendingTrade, err error, now func() time.Time,
	record func(context.Context, []leaguefeed.PendingTrade, time.Time, error) error, log func(string)) {
	switch {
	case errors.Is(err, mflkey.ErrNoKey):
		ClearPending(held)
	case errors.Is(err, mflkey.ErrUnavailable), errors.Is(err, mflkey.ErrTimeout):
		held.RefreshError = "MFL keyring is locked or unavailable"
	default:
		at := now()
		if rerr := record(ctx, value, at, err); rerr != nil && err == nil {
			log(rerr.Error())
		}
		report.HoldFeed(held, value, report.LiveFreshness(at), err)
	}
}

func PendingParent(worker context.Context, fallback func() context.Context) context.Context {
	if worker != nil {
		return worker
	}
	return fallback()
}
