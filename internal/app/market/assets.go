package market

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
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
// main.AssetValue. NowValue and DynastyValue order the board (points per game, dynasty points
// per game); zero when not held.
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

// Dynasty is one asset's native value at a moment, in dynasty points per game: the discounted
// mean of its expected league points per game over Seasons seasons, this one first (a player's
// from the latest model run, a pick's from Pick Value v1).
type Dynasty struct {
	Discounted    float64
	Seasons       int
	ClassLabel    string
	Valued        bool
	Reason        string
	ReducedInputs bool
}

// Model values a player or a pick at a moment. The caller holds none until the first model run
// is read.
type Model interface {
	Player(pid string, at time.Time) Dynasty
	Pick(year, round int, original string, at time.Time) Dynasty
}

const boardHow = "On-field-now is points per game in this league's scoring over the finished weeks MFL has " +
	"scored, counting only weeks in which the player has a score, ranked against every player at his position " +
	"with a score this season. Dynasty value is the native value model's: his expected league points per game " +
	"over this season and the ones after, later seasons discounted, from his record (draft capital, age, college, " +
	"combine) and his seasons. A pick is valued in the same units by " + pickvalue.Version + ": what the players " +
	"typically taken at its slot are worth. Trades are not priced. Values only; this app moves nothing."

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
// model at a moment; model is nil until the first model run is read.
func BuildBoard(now OnField, model Model, at time.Time, also []string) Board {
	b := Board{Values: []Value{}, How: boardHow, NowSource: now.Source}
	b.DynastySource = "Source: the native value model's latest run, and " + pickvalue.Version + " for picks"
	if model == nil {
		b.DynastySource = "Source: the native value model · no model run read yet"
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

// DynastyCells words the native value and the market cell, or why there is none.
func DynastyCells(model Model, id string, at time.Time) (Cell, Cell, float64) {
	if model == nil {
		missing := Cell{Text: "Not valued yet", Sub: "no model run read yet"}
		return missing, missing, 0
	}
	return ValueCells(model.Player(id, at))
}

// ValueCells words one native value: the dynasty cell, the market cell and the value itself.
func ValueCells(d Dynasty) (Cell, Cell, float64) {
	if !d.Valued {
		reason := Cell{Text: "Not valued", Sub: d.Reason}
		return reason, reason, 0
	}
	dynasty := Cell{Held: true, Text: Signed(d.Discounted) + " PPG",
		Sub: fmt.Sprintf("dynasty · next %d seasons", d.Seasons),
		Detail: fmt.Sprintf("%s · expected league points per game this season and the %d after, later ones discounted",
			d.ClassLabel, max(d.Seasons-1, 0))}
	if d.ReducedInputs {
		dynasty.Sub += " · reduced inputs"
		dynasty.Detail += "; " + d.Reason
	}
	market := Cell{Text: "No price", Sub: "trades are not priced"}
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

func PoolAssetValues(now OnField, model Model, ids []playerid.PlayerID, at time.Time) (map[string]AssetValue, string) {
	also := make([]string, 0, len(ids))
	for _, id := range ids {
		also = append(also, id.String())
	}
	board := WireAssetBoard(BuildBoard(now, model, at, also))
	out := make(map[string]AssetValue, len(board.Values))
	for _, v := range board.Values {
		out[v.ID] = v
	}
	return out, board.NowSource
}

func PickValue(model Model, year, round int, original string, at time.Time) *AssetCell {
	if model == nil {
		return &AssetCell{Text: "Not valued", Sub: "no model run read yet"}
	}
	cell, _, _ := ValueCells(model.Pick(year, round, original, at))
	out := AssetCell(cell)
	return &out
}
