package composition

import (
	"context"
	"time"

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
	saved, err := a.contracts().TargetLineup(franchiseID)
	if err != nil {
		return controlapp.AlertReading{}, err //nolint:wrapcheck // control.API prefixes the refusal
	}
	trades, err := a.tradesAPI().TargetTrades(franchiseID)
	if err != nil {
		return controlapp.AlertReading{}, err //nolint:wrapcheck // control.API prefixes the refusal
	}
	now := a.core.MovesNow()
	r := controlapp.AlertReading{Alerts: []controlapp.Alert{}, Unavailable: []controlapp.Unavailable{}}
	a.irAlerts(&r, franchiseID)
	a.lineupAlert(&r, saved, now)
	a.byeAlerts(&r, franchiseID, now)
	tradeAlert(&r, trades, now)
	return r, nil
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
	controlapp.ByeAlerts(r, controlapp.ByeAlertInput{Snapshot: snap, Franchise: mine, Byes: byes, Week: week, WeekFailure: failed, Starters: a.core.Rulebook.ActiveConfig().Starters, RulesFreshness: a.core.RulesFreshness, Now: now}, func(w int) time.Time { return a.byeWeekLock(ctx, week, w, now) })
}

func (a *App) byeWeekLock(ctx context.Context, current leagueweek.Week, week int, now time.Time) time.Time {
	return controlapp.ByeWeekLock(current, week, now, func(w int) ([]nflschedule.RawWeek, bool, error) {
		feed := a.scheduleFeed(w)
		feed.Bodies = a.core.History.ReferenceBodies
		rows, _, found, err := readArchivedFeed(ctx, a.core.Season, feed)
		return rows, found, err
	})
}
