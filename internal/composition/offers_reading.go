package composition

import (
	"context"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	bidsapp "github.com/secureprospective/TheWarRoom/internal/app/offers/bids"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// offersHost is unbound composition over the shared state and other features' readers.
type offersHost struct {
	*core.State
	app *App
}

func (h offersHost) OffersHistoryContext() context.Context            { return h.app.fallbackParent() }
func (h offersHost) OffersBuildStamp() string                         { return buildLabel() + " " + buildDate }
func (h offersHost) OffersLab() (*lab.Data, error)                    { return h.app.lab() }
func (h offersHost) OffersProvenance(d *lab.Data) snapshot.Provenance { return counterProvenance(d) }
func (h offersHost) OffersDirectory(ctx context.Context) (normalize.Lookup, error) {
	return h.app.directory(ctx)
}
func (h offersHost) OffersRosterTarget(option string) (envelope.Target, []string) {
	return h.app.rosterMoveTarget(option)
}
func (h offersHost) OffersLog(e envelope.Envelope)      { h.app.logMove(e) }
func (h offersHost) OffersWake()                        { h.app.wakeMoves() }
func (h offersHost) RefreshOffersPool(force bool) error { return h.app.refreshOffersPool(force) }
func (h offersHost) OffersTradeBait(ctx context.Context) report.HeldFeed[[]ingestion.RawTradeBait] {
	feed := readShellFeed(ctx, h.app, &h.app.core.SeasonTradeBait, "tradeBait", ingestion.ParseTradeBait)
	feed.Value = slices.Clone(feed.Value)
	return feed
}
func (h offersHost) OffersInjuries() (snapshot.Provenance, map[string]string) {
	held := h.app.heldInjuries()
	return held.Provenance, held.Status
}
func (h offersHost) OffersMarketFacts() map[string][]report.MarketFact {
	return offersMarketFacts(marketFacts(h.app.heldMarket()))
}
func (h offersHost) OffersEngineValues(ctx context.Context) (map[string]float64, string) {
	return h.app.draftReportValues(ctx)
}
func (h offersHost) OffersAssetValues(ctx context.Context, ids []playerid.PlayerID, at time.Time) (map[string]offersapp.AssetValue, string) {
	values, note := h.app.poolAssetValues(ctx, ids, at)
	out := make(map[string]offersapp.AssetValue, len(values))
	for id, value := range values {
		out[id] = offersapp.AssetValue{NowText: value.Now.Text, DynastyText: value.Dynasty.Text,
			NowHeld: value.Now.Held, DynastyHeld: value.Dynasty.Held, NowValue: value.NowValue, DynastyValue: value.DynastyValue}
	}
	return out, note
}

func offersMarketFacts(facts map[string][]MarketFact) map[string][]report.MarketFact {
	out := make(map[string][]report.MarketFact, len(facts))
	for id, rows := range facts {
		if rows == nil {
			out[id] = nil
			continue
		}
		out[id] = make([]report.MarketFact, len(rows))
		for i, fact := range rows {
			cell := report.MarketFact{Label: fact.Label, Text: fact.Text, Reason: fact.Reason}
			if fact.value != nil {
				value := *fact.value
				cell.Value = &value
			}
			if fact.Range != nil {
				value := report.Range{Max: fact.Range.Max, Interval: report.Interval(fact.Range.Interval), Text: fact.Range.Text}
				cell.Range = &value
			}
			out[id][i] = cell
		}
	}
	return out
}

// Research and roster actions still share this rule and the pool's market wording.
func heldCutCharge(p state.MirrorPlayer, season int) (domain.Money, bool) {
	return bidsapp.HeldCutCharge(p, season)
}
func poolMarketLines(facts []MarketFact) []string {
	return offersapp.MarketLines(offersMarketFacts(map[string][]MarketFact{"": facts})[""])
}

// The caller holds seasonMu, so the ending evidence and the pending read form one observation.
func (a *App) recordOffers(ctx context.Context, pending []leaguefeed.PendingTrade, at time.Time, readErr error) error {
	return offersapp.RecordOffers(ctx, pending, at, readErr, a.core.Pools != nil, //nolint:wrapcheck // Preserve original read/save errors.
		func() (string, error) {
			d, err := a.lab()
			if err != nil {
				return "", err
			}
			return lab.OfferOwner(d), nil
		}, func(ctx context.Context, owner string, pending []leaguefeed.PendingTrade, at time.Time) error {
			return personal.New(a.core.Pools).ObserveOffers(ctx, owner, a.core.Season, pending, a.core.SeasonTransactions.Value, at)
		})

}

func counterProvenance(d *lab.Data) snapshot.Provenance {
	return snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: d.Built, Note: "Frozen historical input; not current league truth"}}
}

// archivedOnly reads already held exports; opening a list or the clock never calls MFL.
func archivedOnly[T any](ctx context.Context, a *App, export string, leagueless bool,
	parse func([]byte) ([]T, error)) heldSeasonFeed[[]T] {
	return core.ArchivedOnly(ctx, a.core, export, leagueless, parse)
}

func (a *App) personalSnapshot(mine string) (snapshot.Snapshot, error) {
	a.core.TargetSnapshotMu.Lock()
	snap, loaded := a.core.TargetSnapshot, a.core.HasTargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	return offersapp.PersonalSnapshot(snap, loaded, mine) //nolint:wrapcheck // Preserve the personal snapshot refusal.
}

func (h offersHost) OffersCapRoom(franchise string) *domain.Money {
	return offersapp.CapRoom(h.Rulebook.GetSalaryCap(), func() (domain.Money, bool) { return h.League.CapUsed(franchise) })
}
