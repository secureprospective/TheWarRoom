package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"
	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
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

type onFieldRow struct {
	id, position string
	sum          float64
	games        int
}

// finishedScores is the one finished-week scope shared by cards and player research.
func (v researchView) finishedScores() (map[string]*onFieldRow, int, time.Time) {
	last := max(v.week-1, 0)
	rows := map[string]*onFieldRow{}
	held, latest := 0, time.Time{}
	for w := 1; w <= last; w++ {
		feed := v.feed("playerScores", strconv.Itoa(w))
		if len(feed.Value) == 0 {
			continue
		}
		held++
		if feed.FetchedAt.After(latest) {
			latest = feed.FetchedAt
		}
		for _, e := range researchRows(feed) {
			id := e.ID.String()
			r := rows[id]
			if r == nil {
				r = &onFieldRow{id: id, position: v.player(id).Position}
				rows[id] = r
			}
			r.sum += e.Value
			r.games++
		}
	}
	return rows, held, latest
}

// onField is each scored player's points per game and rank at his position, in the service.
func (v researchView) onField() marketapp.OnField {
	rows, held, latest := v.finishedScores()
	scored := make([]marketapp.Scored, 0, len(rows))
	for _, r := range rows {
		scored = append(scored, marketapp.Scored{ID: r.id, Position: r.position, Sum: r.sum, Games: r.games})
	}
	return marketapp.OnFieldNow(scored, held, max(v.week-1, 0), latest, v.zone)
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

// dynastyValueCells words one Lab answer in the service, as wire cells.
func dynastyValueCells(d dynastyValue) (AssetCell, AssetCell, float64) {
	dynasty, market, wins := marketapp.ValueCells(d)
	return AssetCell(dynasty), AssetCell(market), wins
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
	rostered := make([]string, 0, len(in.mirror.Players))
	for _, p := range in.mirror.Players {
		rostered = append(rostered, p.MFLID)
	}
	return researchView{researchInput: in}.onField(), rostered
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
