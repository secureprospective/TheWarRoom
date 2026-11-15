package composition

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	bidsapp "github.com/secureprospective/TheWarRoom/internal/app/mflref/bids"
	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// offersHost is unbound composition over the shared state and other features' readers.
type offersHost struct {
	*core.State
	app *App
}

func (h offersHost) OffersBuildStamp() string { return buildLabel() + " " + buildDate }
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
	in := h.app.heldResearchInput().snapshot()
	statuses := researchapp.PoolStrings(in)
	offseason := false
	if h.Whatif != nil {
		phase, err := h.Whatif.CurrentPhase(ctx)
		if err != nil {
			in.Finished = nil
			note += " Season phase is unavailable"
		} else {
			offseason = phase == domain.PhaseOffseason
		}
	}
	if offseason {
		end, err := strconv.Atoi(h.Rulebook.ActiveConfig().EndWeek)
		if err == nil {
			in.Held.Feeds = map[string]researchapp.Feed[[]researchfeeds.Feed]{}
			for week := 1; week <= end; week++ {
				period := strconv.Itoa(week)
				in.Held.Feeds[researchapp.Key("playerScores", period)] = h.app.researchArchiveFeeds().Read(ctx, h.Season-1, "playerScores", period)
			}
			in.Week = end + 1
			in.Finished = researchapp.FinishedScores(in).Rows
		} else {
			in.Finished = nil
		}
	}
	out := make(map[string]offersapp.AssetValue, len(ids))
	for _, player := range ids {
		id := player.String()
		value := values[id]
		status, points := statuses[id], "—"
		if status == "" {
			status = "—"
		}
		if score := in.Finished[id]; score != nil {
			points = fmt.Sprintf("%.1f", score.Sum)
		}
		out[id] = offersapp.AssetValue{String: status, Points: points, NowText: value.Now.Text, DynastyText: value.Dynasty.Text,
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
