package market

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/labcompile"
)

// Board is C-01: what each player is worth on the field now and as a dynasty asset; it is
// main.AssetBoard. Every cell is worded; a value the app cannot support says why in place of a
// number.
type Board struct {
	Values        []Value `json:"values"`
	NowSource     string  `json:"nowSource"`
	DynastySource string  `json:"dynastySource"`
	How           string  `json:"how"`
}

// Value is one player's three cells: on-field-now, dynasty value and market; it is
// main.AssetValue. NowValue and DynastyValue order the board (points per game, discounted
// wins); zero when not held.
type Value struct {
	ID           string  `json:"id"`
	Now          Cell    `json:"now"`
	Dynasty      Cell    `json:"dynasty"`
	Market       Cell    `json:"market"`
	NowValue     float64 `json:"nowValue"`
	DynastyValue float64 `json:"dynastyValue"`
}

// Cell is a worded value: a short headline, the qualifier beneath it and the detail on hover;
// it is main.AssetCell. Held is false when the words say why there is no value.
type Cell struct {
	Text   string `json:"text"`
	Sub    string `json:"sub"`
	Detail string `json:"detail"`
	Held   bool   `json:"held"`
}

// Dynasty is one asset's Lab value at a moment: the shape of the Lab valuer's answer
// (internal/labcompile Model.Player). Price and its range are in win-equivalents: 1.00 is one
// expected win in a next-draft first-round pick.
type Dynasty struct {
	Discounted        float64
	Seasons           int
	Class, ClassLabel string
	Price, Low, High  *float64
	Valued            bool
	Reason            string
}

// Model values a player or a pick at a moment. The caller holds none until the Lab's value
// model is compiled.
type Model interface {
	Player(pid string, at time.Time) Dynasty
	Pick(year, round int, original string, at time.Time) Dynasty
}

// LabModel is the Lab's fitted value model (internal/labcompile) as the asset board reads it.
type LabModel struct{ Model *labcompile.Model }

// Pick values a draft pick of a season and round, by its original owner.
func (l LabModel) Pick(year, round int, original string, at time.Time) Dynasty {
	return DynastyFrom(l.Model.Pick(year, round, 0, original, at))
}

// Player values a player.
func (l LabModel) Player(pid string, at time.Time) Dynasty {
	return DynastyFrom(l.Model.Player(pid, at))
}

// DynastyFrom keeps the Lab's answer as given: its forecast seasons, class words, price and range.
func DynastyFrom(v labcompile.AssetValue) Dynasty {
	d := Dynasty{Discounted: v.Discounted, Seasons: len(v.Forecast), Class: v.Class, ClassLabel: v.Label,
		Price: v.Price, Valued: v.Valued, Reason: v.Reason}
	if v.Range != nil {
		d.Low, d.High = &v.Range.Low, &v.Range.High
	}
	return d
}

const boardHow = "On-field-now is points per game in this league's scoring over the finished weeks MFL has " +
	"scored, counting only weeks in which the player has a score, ranked against every player at his position " +
	"with a score this season. Dynasty value is the Lab's forecast of the wins he adds over five seasons (this " +
	"season's unplayed weeks and the four after), later seasons discounted; market is what the league has paid for " +
	"that in trades, in win-equivalents (1.00 is the price of one expected win from a next-draft first-round pick), " +
	"with the market's 95% range. Each player in a trade also costs a roster spot, so filler can price below zero. " +
	"Values only; this app moves nothing."

// Scored is one player's points over the finished weeks the caller holds, and the weeks he scored.
type Scored struct {
	ID, Position string
	Sum          float64
	Games        int
}

// OnField is each scored player's on-field-now cell and points per game, with the source line.
type OnField struct {
	Cells  map[string]Cell
	PPG    map[string]float64
	Source string
}

// OnFieldNow words each scored player's points per game and rank at his position. held is the
// number of the last finished weeks the caller holds, latest when the newest was read.
func OnFieldNow(rows []Scored, held, last int, latest time.Time, zone *time.Location) OnField {
	source := "Source: MFL fantasy points, this league's scoring · no finished week held yet"
	if held > 0 {
		source = fmt.Sprintf("Source: MFL fantasy points, this league's scoring · %d of %d finished weeks held · as of %s",
			held, last, report.Time(latest, zone))
	}
	byPosition := map[string][]Scored{}
	for _, r := range rows {
		byPosition[r.Position] = append(byPosition[r.Position], r)
	}
	out := OnField{Cells: make(map[string]Cell, len(rows)), PPG: make(map[string]float64, len(rows)), Source: source}
	for position, group := range byPosition {
		slices.SortFunc(group, func(a, b Scored) int {
			return cmp.Or(cmp.Compare(b.Sum/float64(b.Games), a.Sum/float64(a.Games)), cmp.Compare(a.ID, b.ID))
		})
		for i, r := range group {
			rank := i + 1
			for rank > 1 && group[rank-2].Sum/float64(group[rank-2].Games) == r.Sum/float64(r.Games) {
				rank-- // equal points per game share the higher rank
			}
			out.PPG[r.ID] = r.Sum / float64(r.Games)
			out.Cells[r.ID] = Cell{Held: true, Text: fmt.Sprintf("%.2f PPG", out.PPG[r.ID]),
				Sub:    fmt.Sprintf("%s %d/%d · top %d%%", position, rank, len(group), TopPercent(rank, len(group))),
				Detail: fmt.Sprintf("%.2f points in %s", r.Sum, report.Count(r.Games, "scored week"))}
		}
	}
	return out
}

// TopPercent is the share of the position at or above a rank, rounded up: rank 1 of 50 is top 2%.
func TopPercent(rank, of int) int {
	return (100*rank + of - 1) / of
}

// BuildBoard words C-01 for every player with an on-field cell, and every id in also, valued by
// model at a moment; model is nil until the Lab's value model is compiled.
func BuildBoard(now OnField, model Model, at time.Time, also []string) Board {
	b := Board{Values: []Value{}, How: boardHow, NowSource: now.Source}
	b.DynastySource = "Source: the Lab's value model, compiled from the checked league facts"
	if model == nil {
		b.DynastySource = "Source: the Lab's value model · not compiled yet"
	}
	cells := now.Cells
	ids := slices.Sorted(func(yield func(string) bool) {
		for id := range cells {
			if !yield(id) {
				return
			}
		}
	})
	for _, id := range also {
		if _, ok := cells[id]; !ok {
			ids = append(ids, id)
			cells[id] = Cell{Text: "No score", Sub: "this season"}
		}
	}
	for _, id := range ids {
		value := Value{ID: id, Now: cells[id], NowValue: now.PPG[id]}
		value.Dynasty, value.Market, value.DynastyValue = DynastyCells(model, id, at)
		b.Values = append(b.Values, value)
	}
	return b
}

// DynastyCells words the Lab's value and its market price, or why there is none.
func DynastyCells(model Model, id string, at time.Time) (Cell, Cell, float64) {
	if model == nil {
		missing := Cell{Text: "Not valued yet", Sub: "Lab model not compiled"}
		return missing, missing, 0
	}
	return ValueCells(model.Player(id, at))
}

// ValueCells words one Lab answer: the dynasty cell, the market cell and the discounted wins.
func ValueCells(d Dynasty) (Cell, Cell, float64) {
	if !d.Valued {
		reason := Cell{Text: "Not valued", Sub: d.Reason}
		return reason, reason, 0
	}
	dynasty := Cell{Held: true, Text: Signed(d.Discounted) + " wins",
		Sub:    fmt.Sprintf("next %d seasons", d.Seasons),
		Detail: fmt.Sprintf("%s · this season's unplayed weeks and the %d after, later ones discounted", d.ClassLabel, max(d.Seasons-1, 0))}
	market := Cell{Text: "No price", Sub: "no market for " + d.ClassLabel}
	if d.Price != nil {
		market = Cell{Held: true, Text: Signed(*d.Price), Sub: "win-equivalents",
			Detail: "1.00 is the price of one expected win from a next-draft first-round pick"}
		if d.Low != nil && d.High != nil {
			market.Sub = fmt.Sprintf("win-equivalents · 95%% %s to %s", Signed(*d.Low), Signed(*d.High))
		}
		if *d.Price < 0 {
			market.Detail += ". Below zero: in trades the league has valued this kind of player at less than the roster spot he takes"
		}
	}
	return dynasty, market, d.Discounted
}

// Signed writes a price with a true minus sign, so a range reads "−0.04 to −0.01". A value that
// rounds to zero reads 0.00, never −0.00.
func Signed(v float64) string {
	if text := fmt.Sprintf("%.2f", -v); v < 0 && text != "0.00" {
		return "−" + text
	}
	return fmt.Sprintf("%.2f", math.Abs(v))
}
