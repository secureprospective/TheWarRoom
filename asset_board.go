package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	"github.com/secureprospective/TheWarRoom/internal/labcompile"
)

// AssetBoard is C-01: what each player is worth on the field now and as a dynasty asset. Go words
// every cell; a value the app cannot support says why in place of a number.
type AssetBoard struct {
	Values        []AssetValue `json:"values"`
	NowSource     string       `json:"nowSource"`
	DynastySource string       `json:"dynastySource"`
	How           string       `json:"how"`
}

// AssetValue is one player's three cells: on-field-now, dynasty value and market. NowValue and
// DynastyValue order the board (points per game, discounted wins); zero when not held.
type AssetValue struct {
	ID           string    `json:"id"`
	Now          AssetCell `json:"now"`
	Dynasty      AssetCell `json:"dynasty"`
	Market       AssetCell `json:"market"`
	NowValue     float64   `json:"nowValue"`
	DynastyValue float64   `json:"dynastyValue"`
}

// AssetCell is a worded value: a short headline, the qualifier beneath it and the detail on hover.
// Held is false when the words say why there is no value.
type AssetCell struct {
	Text   string `json:"text"`
	Sub    string `json:"sub"`
	Detail string `json:"detail"`
	Held   bool   `json:"held"`
}

// dynastyValue and assetModel are the service's Lab answer and value model; the App holds the
// model (assetModel) once the Lab compiles. Neither is on the wire.
type (
	dynastyValue = marketapp.Dynasty
	assetModel   = marketapp.Model
)

// buildAssetBoard words C-01 in the service for every player with a held score this season, and
// every id in also, and converts it to its wire type.
func buildAssetBoard(v researchView, model assetModel, at time.Time, also []string) AssetBoard {
	return wireAssetBoard(marketapp.BuildBoard(v.onField(), model, at, also))
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

// wireAssetBoard converts a board built by the Market service to its wire twin.
func wireAssetBoard(b marketapp.Board) AssetBoard {
	values := make([]AssetValue, len(b.Values))
	for i, v := range b.Values {
		values[i] = AssetValue{ID: v.ID, Now: AssetCell(v.Now), Dynasty: AssetCell(v.Dynasty), Market: AssetCell(v.Market),
			NowValue: v.NowValue, DynastyValue: v.DynastyValue}
	}
	return AssetBoard{Values: wireSlice(b.Values, values), NowSource: b.NowSource, DynastySource: b.DynastySource, How: b.How}
}

// TargetAssetBoard reads C-01 from held state: the weekly scores (from their archived copies when
// no refresh has run this launch) and, once compiled, the Lab's value model. Every rostered player
// has a row, scored or not. It never fetches.
func (a *App) TargetAssetBoard() (AssetBoard, error) {
	if err := a.ready(); err != nil {
		return AssetBoard{}, fmt.Errorf("asset board: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m4Timeout)
	defer cancel()
	return a.assetBoard(ctx)
}

func (a *App) assetBoard(ctx context.Context) (AssetBoard, error) {
	a.hydrateResearchContext(ctx)
	in := a.heldResearchInput()
	rostered := make([]string, 0, len(in.mirror.Players))
	for _, p := range in.mirror.Players {
		rostered = append(rostered, p.MFLID)
	}
	var model assetModel
	if m := a.assetModel.Load(); m != nil {
		model = *m
	}
	return buildAssetBoard(researchView{researchInput: in}, model, time.Now(), rostered), nil
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
	var model assetModel = marketapp.LabModel{Model: m}
	a.assetModel.Store(&model)
	log.Printf("the war room: value model fitted in %s", time.Since(began).Round(time.Millisecond))
	a.core.SeasonChanged(ctx)
}
