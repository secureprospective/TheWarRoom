package composition

import (
	"context"
	"log"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	researchapp "github.com/secureprospective/TheWarRoom/internal/app/research"
	"github.com/secureprospective/TheWarRoom/internal/labcompile"
)

// AssetBoard, AssetValue and AssetCell are Market's wire types; the pool and trade surfaces
// still built here carry them.
type (
	AssetBoard = marketapp.AssetBoard
	AssetValue = marketapp.AssetValue
	AssetCell  = marketapp.AssetCell
)

// dynastyValue and assetModel are the service's Lab answer and value model; the App holds the
// model (assetModel) once the Lab compiles. Neither is on the wire.
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

func (h marketHost) LabStamp() string { return labStamp() }
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

// fitAssetModel fits the Lab's value model on the verified facts warmArchive just read, so C-01
// values players on the same frame and market as the Lab's trade legs. Until it lands the board
// says the model is not compiled; when it lands the board reloads.
func (a *App) fitAssetModel(ctx context.Context) {
	began := time.Now()
	a.core.FactsMu.Lock()
	cache := a.archiveCache
	a.core.FactsMu.Unlock()
	if cache.SHA == "" {
		return
	}
	m, _, err := labcompile.Build(ctx, cache.Facts, labcompile.Evidence{Built: began, ArchiveThrough: cache.Through})
	if err != nil {
		log.Printf("the war room: value model: %v", err)
		return
	}
	a.marketHeld.PublishModel(marketapp.LabModel{Model: m})
	log.Printf("the war room: value model fitted in %s", time.Since(began).Round(time.Millisecond))
	a.core.SeasonChanged(ctx)
}
