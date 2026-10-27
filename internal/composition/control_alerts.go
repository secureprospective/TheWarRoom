package composition

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// Alerts gathers the alert tray's alerts unsorted; control.API sorts them and words the refusal.
// Each held input is read under its own lock, in the binding's original order.
func (h controlHost) Alerts(franchiseID string) (controlapp.AlertReading, error) {
	a := h.app
	var saved contractsapp.LineupReading
	var trades tradesapp.TradeReading
	//nolint:wrapcheck // Preserve existing feature refusal text.
	return controlapp.GatherAlerts(controlapp.AlertSource{
		Lineup: func() error {
			var err error
			saved, err = a.contracts().TargetLineup(franchiseID)
			return err //nolint:wrapcheck // The Control API supplies the refusal context.
		},
		Trades: func() error {
			var err error
			trades, err = a.tradesAPI().TargetTrades(franchiseID)
			return err //nolint:wrapcheck // The Control API supplies the refusal context.
		},
		Now: a.core.MovesNow, IR: func(r *controlapp.AlertReading) { a.irAlerts(r, franchiseID) },
		SavedLineup: func(r *controlapp.AlertReading, now time.Time) { a.lineupAlert(r, saved, now) },
		Byes:        func(r *controlapp.AlertReading, now time.Time) { a.byeAlerts(r, franchiseID, now) },
		TradeOffers: func(r *controlapp.AlertReading, now time.Time) { tradeAlert(r, trades, now) },
	})

}

func (a *App) lineupAlert(r *controlapp.AlertReading, saved contractsapp.LineupReading, now time.Time) {
	a.core.WeekMu.Lock()
	week, failed := a.core.Week, a.core.WeekRefreshError
	a.core.WeekMu.Unlock()
	controlapp.LineupAlert(r, controlapp.LineupAlertInput{Week: saved.Week, Check: saved.Check, Provenance: saved.Provenance, RulesSource: saved.RulesSource}, week, failed, now)
}

func tradeAlert(r *controlapp.AlertReading, trades tradesapp.TradeReading, now time.Time) {
	offers := make([]controlapp.TradeAlertOffer, len(trades.Offers))
	for i, offer := range trades.Offers {
		offers[i] = controlapp.TradeAlertOffer{Direction: offer.Direction, Expires: offer.Expires}
	}
	controlapp.TradeAlert(r, controlapp.TradeAlertInput{Offers: offers, Provenance: trades.Provenance}, now)
}

func (a *App) irAlerts(r *controlapp.AlertReading, franchiseID string) {
	a.core.TargetSnapshotMu.Lock()
	snap, loaded := a.core.TargetSnapshot, a.core.HasTargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	injuries := a.heldInjuries()
	controlapp.IRAlerts(r, franchiseID, snap, loaded, injuries, func() string { target, _ := a.rosterMoveTarget("18"); return target.URL })
}

func (a *App) byeAlerts(r *controlapp.AlertReading, mine string, now time.Time) {
	a.core.TargetSnapshotMu.Lock()
	snap := a.core.TargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	a.core.WeekMu.Lock()
	week, failed := a.core.Week, a.core.WeekRefreshError
	a.core.WeekMu.Unlock()
	ctx, cancel := context.WithTimeout(a.core.Ctx, m4Timeout)
	defer cancel()
	byes := archivedOnly(ctx, a, nflschedule.ByeExport, true, nflschedule.ParseByes)
	controlapp.ByeAlerts(r, controlapp.ByeAlertInput{Snapshot: snap, Franchise: mine, Byes: byes, Week: week, WeekFailure: failed, Starters: a.core.Rulebook.ActiveConfig().Starters, RulesFreshness: func() domain.Freshness { return contractsapp.RulesFreshness(a.core.RulesCheckedAt.Load()) }, Now: now}, func(w int) time.Time { return a.byeWeekLock(ctx, week, w, now) })
}

func (a *App) byeWeekLock(ctx context.Context, current leagueweek.Week, week int, now time.Time) time.Time {
	return controlapp.ByeWeekLock(current, week, now, func(w int) ([]nflschedule.RawWeek, bool, error) {
		feed := a.scheduleFeed(w)
		feed.Bodies = a.core.History.ReferenceBodies
		rows, _, found, err := readArchivedFeed(ctx, a.core.Season, feed)
		return rows, found, err
	})
}
