package core

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// SeasonFeeds copies the held transactions, lineups (with the week they were asked for) and
// pending trades under SeasonMu. The slices, each lineup row's players and starters are copied
// and nil stays nil; a refresh replaces the published rows, never edits them in place.
func (s *State) SeasonFeeds() (HeldFeed[[]leaguefeed.Transaction], HeldFeed[leaguefeed.Lineups], int,
	HeldFeed[[]leaguefeed.PendingTrade]) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	txs, lineups, trades := s.SeasonTransactions, s.SeasonLineups, s.SeasonPendingTrades
	txs.Value = slices.Clone(txs.Value)
	trades.Value = slices.Clone(trades.Value)
	lineups.Value = CloneLineups(lineups.Value)
	return txs, lineups, s.LineupsWeek, trades
}

// SetLeagueWindow writes one league date through the coordinator, then tells the views the clock
// changed. The caller has validated the request and waited for startup.
func (s *State) SetLeagueWindow(ctx context.Context, kind, at string) error {
	if err := s.Whatif.Err(); err != nil {
		return fmt.Errorf("set league date: state: %w", err)
	}
	var err error
	if kind == string(leagueclock.TradeDeadline) {
		err = s.setTradeWindow(ctx, at)
	} else {
		err = s.setCalendarWindow(ctx, kind, at)
	}
	if err != nil {
		return err
	}
	s.ClockChanged(ctx)
	return nil
}

func (s *State) setTradeWindow(ctx context.Context, at string) error {
	deadline := time.Time{}
	if at == "" {
		current, err := s.Whatif.TradeDeadline(ctx)
		if err != nil {
			return fmt.Errorf("read trade date: %w", err)
		}
		if current.IsZero() {
			return fmt.Errorf("trade deadline already follows the rulebook; there is nothing to reset")
		}
	} else {
		var err error
		deadline, err = time.Parse(time.RFC3339, at)
		if err != nil {
			return fmt.Errorf("trade date: %w", err)
		}
	}
	if _, err := s.Coordinator.Execute(ctx, transactions.SetTradeDeadline{
		Deadline: deadline, Note: "commissioner set trade_deadline",
	}); err != nil {
		return fmt.Errorf("set trade date: %w", err)
	}
	return nil
}

func (s *State) setCalendarWindow(ctx context.Context, kind, at string) error {
	heads, err := s.Whatif.CalendarEvents(ctx)
	if err != nil {
		return fmt.Errorf("read league dates: %w", err)
	}
	id := fmt.Sprintf("window:%d:%s", s.League.Season(), kind)
	var head *state.CalendarEvent
	for i := range heads {
		if heads[i].EventID == id {
			head = &heads[i]
			break
		}
	}
	event := transactions.CalendarEvent{EventID: id, EventKind: transactions.WindowDate,
		ScheduledAt: at, Payload: fmt.Sprintf(`{"window":%q}`, kind), Note: "commissioner set " + kind}
	var request transactions.Request = transactions.ScheduleEvent{Event: event}
	if at == "" {
		if head == nil || head.Status != state.CalStatusPlanned {
			return fmt.Errorf("this window already follows the rulebook; there is nothing to reset")
		}
		event.ScheduledAt = head.ScheduledAt
		request = transactions.CancelEvent{Event: event}
	} else if head != nil {
		request = transactions.RescheduleEvent{Event: event}
	}
	if _, err := s.Coordinator.Execute(ctx, request); err != nil {
		return fmt.Errorf("set league date: %w", err)
	}
	return nil
}
