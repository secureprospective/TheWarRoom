package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"math"
	"slices"
	"strconv"
	"time"

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

// dynastyValue is one asset's Lab value at a moment: the shape of the Lab valuer's answer
// (internal/labcompile Model.Player). Price and its range are in win-equivalents: 1.00 is one
// expected win in a next-draft first-round pick.
type dynastyValue struct {
	Discounted        float64
	Seasons           int
	Class, ClassLabel string
	Price, Low, High  *float64
	Valued            bool
	Reason            string
}

// assetModel values a player at a moment. Nil until the Lab's value model is compiled.
type assetModel interface {
	player(pid string, at time.Time) dynastyValue
	pick(year, round int, original string, at time.Time) dynastyValue
}

const assetBoardHow = "On-field-now is points per game in this league's scoring over the finished weeks MFL has " +
	"scored, counting only weeks in which the player has a score, ranked against every player at his position " +
	"with a score this season. Dynasty value is the Lab's forecast of the wins he adds over five seasons (this " +
	"season's unplayed weeks and the four after), later seasons discounted; market is what the league has paid for " +
	"that in trades, in win-equivalents (1.00 is the price of one expected win from a next-draft first-round pick), " +
	"with the market's 95% range. Each player in a trade also costs a roster spot, so filler can price below zero. " +
	"Values only; this app moves nothing."

// buildAssetBoard words C-01 for every player with a held score this season, and every id in also.
func buildAssetBoard(v researchView, model assetModel, at time.Time, also []string) AssetBoard {
	b := AssetBoard{Values: []AssetValue{}, How: assetBoardHow}
	now, ppg, source := v.onFieldNow()
	b.NowSource = source
	b.DynastySource = "Source: the Lab's value model, compiled from the checked league facts"
	if model == nil {
		b.DynastySource = "Source: the Lab's value model · not compiled yet"
	}
	ids := slices.Sorted(func(yield func(string) bool) {
		for id := range now {
			if !yield(id) {
				return
			}
		}
	})
	for _, id := range also {
		if _, ok := now[id]; !ok {
			ids = append(ids, id)
			now[id] = AssetCell{Text: "No score", Sub: "this season"}
		}
	}
	for _, id := range ids {
		value := AssetValue{ID: id, Now: now[id], NowValue: ppg[id]}
		value.Dynasty, value.Market, value.DynastyValue = dynastyCells(model, id, at)
		b.Values = append(b.Values, value)
	}
	return b
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
		if len(feed.value) == 0 {
			continue
		}
		held++
		if feed.fetchedAt.After(latest) {
			latest = feed.fetchedAt
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

// onFieldNow is each scored player's points per game and rank at his position.
func (v researchView) onFieldNow() (map[string]AssetCell, map[string]float64, string) {
	rows, held, latest := v.finishedScores()
	last := max(v.week-1, 0)
	source := "Source: MFL fantasy points, this league's scoring · no finished week held yet"
	if held > 0 {
		source = fmt.Sprintf("Source: MFL fantasy points, this league's scoring · %d of %d finished weeks held · as of %s",
			held, last, reportTime(latest, v.zone))
	}
	byPosition := map[string][]*onFieldRow{}
	for _, r := range rows {
		byPosition[r.position] = append(byPosition[r.position], r)
	}
	out, ppg := make(map[string]AssetCell, len(rows)), make(map[string]float64, len(rows))
	for position, group := range byPosition {
		slices.SortFunc(group, func(a, b *onFieldRow) int {
			return cmp.Or(cmp.Compare(b.sum/float64(b.games), a.sum/float64(a.games)), cmp.Compare(a.id, b.id))
		})
		for i, r := range group {
			rank := i + 1
			for rank > 1 && group[rank-2].sum/float64(group[rank-2].games) == r.sum/float64(r.games) {
				rank-- // equal points per game share the higher rank
			}
			ppg[r.id] = r.sum / float64(r.games)
			out[r.id] = AssetCell{Held: true, Text: fmt.Sprintf("%.2f PPG", ppg[r.id]),
				Sub:    fmt.Sprintf("%s %d/%d · top %d%%", position, rank, len(group), topPercent(rank, len(group))),
				Detail: fmt.Sprintf("%.2f points in %s", r.sum, countText(r.games, "scored week"))}
		}
	}
	return out, ppg, source
}

// topPercent is the share of the position at or above a rank, rounded up: rank 1 of 50 is top 2%.
func topPercent(rank, of int) int {
	return (100*rank + of - 1) / of
}

// dynastyCells words the Lab's value and its market price, or why there is none.
func dynastyCells(model assetModel, id string, at time.Time) (AssetCell, AssetCell, float64) {
	if model == nil {
		missing := AssetCell{Text: "Not valued yet", Sub: "Lab model not compiled"}
		return missing, missing, 0
	}
	return dynastyValueCells(model.player(id, at))
}

func dynastyValueCells(d dynastyValue) (AssetCell, AssetCell, float64) {
	if !d.Valued {
		reason := AssetCell{Text: "Not valued", Sub: d.Reason}
		return reason, reason, 0
	}
	dynasty := AssetCell{Held: true, Text: signed(d.Discounted) + " wins",
		Sub:    fmt.Sprintf("next %d seasons", d.Seasons),
		Detail: fmt.Sprintf("%s · this season's unplayed weeks and the %d after, later ones discounted", d.ClassLabel, max(d.Seasons-1, 0))}
	market := AssetCell{Text: "No price", Sub: "no market for " + d.ClassLabel}
	if d.Price != nil {
		market = AssetCell{Held: true, Text: signed(*d.Price), Sub: "win-equivalents",
			Detail: "1.00 is the price of one expected win from a next-draft first-round pick"}
		if d.Low != nil && d.High != nil {
			market.Sub = fmt.Sprintf("win-equivalents · 95%% %s to %s", signed(*d.Low), signed(*d.High))
		}
		if *d.Price < 0 {
			market.Detail += ". Below zero: in trades the league has valued this kind of player at less than the roster spot he takes"
		}
	}
	return dynasty, market, d.Discounted
}

// signed writes a price with a true minus sign, so a range reads "−0.04 to −0.01". A value that
// rounds to zero reads 0.00, never −0.00.
func signed(v float64) string {
	if text := fmt.Sprintf("%.2f", -v); v < 0 && text != "0.00" {
		return "−" + text
	}
	return fmt.Sprintf("%.2f", math.Abs(v))
}

// TargetAssetBoard reads C-01 from held state: the weekly scores (from their archived copies when
// no refresh has run this launch) and, once compiled, the Lab's value model. Every rostered player
// has a row, scored or not. It never fetches.
func (a *App) TargetAssetBoard() (AssetBoard, error) {
	if err := a.ready(); err != nil {
		return AssetBoard{}, fmt.Errorf("asset board: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m4Timeout)
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

// labAssets is the Lab's fitted value model (internal/labcompile) as the asset board reads it.
type labAssets struct{ model *labcompile.Model }

func (l labAssets) pick(year, round int, original string, at time.Time) dynastyValue {
	return dynastyFrom(l.model.Pick(year, round, 0, original, at))
}

func (l labAssets) player(pid string, at time.Time) dynastyValue {
	return dynastyFrom(l.model.Player(pid, at))
}

// dynastyFrom keeps the Lab's answer as given: its forecast seasons, class words, price and range.
func dynastyFrom(v labcompile.AssetValue) dynastyValue {
	d := dynastyValue{Discounted: v.Discounted, Seasons: len(v.Forecast), Class: v.Class, ClassLabel: v.Label,
		Price: v.Price, Valued: v.Valued, Reason: v.Reason}
	if v.Range != nil {
		d.Low, d.High = &v.Range.Low, &v.Range.High
	}
	return d
}

// fitAssetModel fits the Lab's value model on the verified facts warmArchive just read, so C-01
// values players on the same frame and market as the Lab's trade legs. Until it lands the board
// says the model is not compiled; when it lands the board reloads.
func (a *App) fitAssetModel(ctx context.Context) {
	began := time.Now()
	a.factsMu.Lock()
	cache := a.archiveCache
	a.factsMu.Unlock()
	if cache.sha == "" {
		return
	}
	m, _, err := labcompile.Build(ctx, cache.facts, labcompile.Evidence{Built: began, ArchiveThrough: cache.through})
	if err != nil {
		log.Printf("the war room: value model: %v", err)
		return
	}
	var model assetModel = labAssets{m}
	a.assetModel.Store(&model)
	log.Printf("the war room: value model fitted in %s", time.Since(began).Round(time.Millisecond))
	a.seasonChanged(ctx)
}
