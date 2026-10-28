package composition

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	recordstore "github.com/secureprospective/TheWarRoom/internal/store/playerrecord"
)

// AssetBoard, AssetValue and AssetCell are Market's wire types; the pool and trade surfaces
// still built here carry them.
type (
	AssetBoard = marketapp.AssetBoard
	AssetValue = marketapp.AssetValue
	AssetCell  = marketapp.AssetCell
)

// dynastyValue and assetModel are the service's native value and value model; the App holds the
// model (assetModel) once a model run is read. Neither is on the wire.
type (
	dynastyValue = marketapp.Dynasty
	assetModel   = marketapp.Model
)

// buildAssetBoard words C-01 in the service for every player with a held score this season, and
// every id in also, and converts it to its wire type.
func buildAssetBoard(v researchView, model assetModel, at time.Time, also []string) AssetBoard {
	return marketapp.WireAssetBoard(marketapp.BuildBoard(v.onField(), model, at, also))
}

// onField is each scored player's points per game and rank at his position, in the service.
func (v researchView) onField() marketapp.OnField {
	finished := researchapp.FinishedScores(v.scoreInput())
	scored := make([]marketapp.Scored, 0, len(finished.Rows))
	for id, r := range finished.Rows {
		scored = append(scored, marketapp.Scored{ID: id, Position: r.Position, Sum: r.Sum, Games: r.Games})
	}
	return marketapp.OnFieldNow(scored, finished.Held, finished.Weeks, finished.Latest, v.zone)
}

// onFieldNow is onField as wire cells.
func (v researchView) onFieldNow() (map[string]AssetCell, map[string]float64, string) {
	now := v.onField()
	cells := make(map[string]AssetCell, len(now.Cells))
	for id, c := range now.Cells {
		cells[id] = AssetCell(c)
	}
	return cells, now.PPG, now.Source
}

// marketHost composes Market's Host: shared state from core, and the Research scores and the
// build stamp still held at the root. It is never bound.
type marketHost struct {
	*core.State
	app *App
}

func (h marketHost) AssetInput(ctx context.Context) (marketapp.OnField, []string) {
	return h.app.assetInput(ctx)
}

// newMarketAPI is the Market API over this App's state and its one Market holder.
func newMarketAPI(app *App) *marketapp.API {
	return marketapp.New(marketHost{State: app.core, app: app}, app.marketHeld)
}

// assetBoard is C-01 for the trade builder: the Research scores and rostered ids, and the held
// value model.
func (a *App) assetBoard(ctx context.Context) (AssetBoard, error) {
	now, rostered := a.assetInput(ctx)
	return marketapp.WireAssetBoard(marketapp.BuildBoard(now, a.marketHeld.SnapshotModel(), time.Now(), rostered)), nil
}

// assetInput hydrates Research's held scores and reads them, with every rostered id, for Market.
func (a *App) assetInput(ctx context.Context) (marketapp.OnField, []string) {
	a.hydrateResearchContext(ctx)
	in := a.heldResearchInput()
	return researchView{researchInput: in}.onField(), researchapp.RosteredAssetIDs(in.mirror)

}

// publishAssetModel reads the latest model run and the player record and publishes the native
// value model C-01 values players and picks with. Until a run exists the board says so.
func (a *App) publishAssetModel(ctx context.Context) {
	h := a.core.History
	if h == nil || a.core.League == nil || a.core.Params == nil {
		return
	}
	run, ok, err := h.LatestRun(ctx, a.core.Season, history.RunModel)
	if err != nil || !ok {
		if err != nil {
			log.Printf("the war room: native value model: %v", err)
		}
		return
	}
	scores, err := h.ModelScores(ctx, run.ID)
	if err != nil {
		log.Printf("the war room: native value model: %v", err)
		return
	}
	records, err := recordstore.New(a.core.HistPools, h).Records(ctx)
	if err != nil {
		log.Printf("the war room: native value model: %v", err)
		return
	}
	set := a.core.Params.Snapshot()
	seasons, err := set.GetGlobal(params.KeyDynastySeasons)
	if err != nil {
		log.Printf("the war room: native value model: %v", err)
		return
	}
	discount, err := set.GetGlobal(params.KeyDynastyDiscount)
	if err != nil {
		log.Printf("the war room: native value model: %v", err)
		return
	}
	values := make(map[string]float64, len(scores))
	for _, sc := range scores {
		values[sc.MFLID] = sc.DynastyPPG
	}
	a.marketHeld.PublishModel(marketapp.NewNative(marketapp.NativeInput{Values: values, Records: records,
		Season: run.Season, Seasons: int(math.Round(seasons)), Discount: discount, Teams: len(a.core.League.Reader().Franchises())}))
}

func (h marketHost) RefreshMarket(force bool) error {
	//nolint:wrapcheck // Preserve Market startup refusal and refresh context.
	return marketapp.RefreshReports(force, h.Ready, func() context.Context { return h.Ctx }, func(ctx context.Context, force bool) {
		h.RefreshMu.Lock()
		defer h.RefreshMu.Unlock()
		h.RefreshMarketFeeds(ctx, force)
	})
}
