package labcompile

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

type rawOracleColumn struct {
	Dictionary []json.RawMessage `json:"dictionary"`
	Tokens     []int             `json:"tokens"`
	Values     []json.RawMessage `json:"values"`
}
type rawAnalysisOracle struct {
	Facts            map[string]map[string]rawOracleColumn `json:"facts"`
	Analysis         RawAnalysis                           `json:"analysis"`
	Trades           []Trade                               `json:"trades"`
	Draft            []Pick                                `json:"draft"`
	DraftSpans       map[int64][2]int64                    `json:"draftSpans"`
	Known, Estimated []struct {
		Year int64
		Fid  string
		Slot int64
	}
	Samples map[string][]rawMethodSample `json:"samples"`
	Fits    []struct {
		Last        int64
		PointInTime bool
		Coef        []struct {
			Group         string
			Horizon, Fold int
			Values        []float64
		}
	}
	Replacement []struct {
		Position string
		Year     int64
		Value    float64
	}
	Spread       map[int64]float64
	BacktestRows [][3]float64
	ValuedTrades []tradeValue
}

func (r rawAnalysisOracle) Validate() error {
	if err := r.Analysis.Validate(); err != nil {
		return err
	}
	for table, cols := range r.Facts {
		n := -1
		for name, col := range cols {
			size := len(col.Values)
			if col.Dictionary != nil {
				size = len(col.Tokens)
				for _, token := range col.Tokens {
					if token < 0 || token >= len(col.Dictionary) {
						return fmt.Errorf("oracle %s.%s has invalid dictionary token", table, name)
					}
				}
			}
			if n >= 0 && n != size {
				return fmt.Errorf("oracle %s has unequal columns", table)
			}
			n = size
		}
	}
	for _, fit := range r.Fits {
		for _, coef := range fit.Coef {
			if len(coef.Values) != 7 || coef.Horizon < 0 || coef.Horizon >= horizon || coef.Fold < 0 || coef.Fold > allFolds {
				return fmt.Errorf("invalid forecast coefficient input")
			}
		}
	}
	return nil
}
func oracleFacts(t *testing.T, r rawAnalysisOracle) facts.Facts {
	t.Helper()
	tables := map[string]map[string][]json.RawMessage{}
	for table, cols := range r.Facts {
		tables[table] = map[string][]json.RawMessage{}
		for name, col := range cols {
			values := col.Values
			if col.Dictionary != nil {
				values = make([]json.RawMessage, len(col.Tokens))
				for i, token := range col.Tokens {
					values[i] = col.Dictionary[token]
				}
			}
			tables[table][name] = values
		}
	}
	body, err := json.Marshal(tables)
	if err != nil {
		t.Fatal(err)
	}
	var raw rawMethods
	if err := json.Unmarshal(body, &raw.Facts); err != nil {
		t.Fatal(err)
	}
	if err := raw.Validate(); err != nil {
		t.Fatal(err)
	}
	fillOmittedColumns(&raw.Facts)
	return raw.Facts
}
func marshalOracle[T []tradeValue | Analysis | map[string]float64 | map[int64]float64 | []AnalysisTeam | []float64 | []*float64 | [][3]float64](t *testing.T, v T) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
func TestAnalysisGoldens(t *testing.T) {
	var r rawAnalysisOracle
	if err := json.Unmarshal(readGzip(t, "testdata/analysis-goldens.json.gz"), &r); err != nil {
		t.Fatal(err)
	}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	in, _ := frozenEvidence(t)
	l, err := newLab(oracleFacts(t, r), in.ArchiveThrough.Unix())
	if err != nil {
		t.Fatal(err)
	}
	spans := map[int64]draftSpan{}
	for year, span := range r.DraftSpans {
		spans[year] = draftSpan{span[0], span[1]}
	}
	l.buildCalendar(r.Trades, spans)
	known, estimated := map[teamYear]int64{}, map[teamYear]int64{}
	for _, k := range r.Known {
		known[teamYear{k.Year, k.Fid}] = k.Slot
	}
	for _, k := range r.Estimated {
		estimated[teamYear{k.Year, k.Fid}] = k.Slot
	}
	w := newWins(l)
	t.Run("wins", func(t *testing.T) {
		for _, replacement := range r.Replacement {
			near(t, w.replacement[positionYear{replacement.Position, replacement.Year}], replacement.Value)
		}
		compareJSON(t, marshalOracle(t, w.spread), marshalOracle(t, r.Spread), "spread")
		compareJSON(t, marshalOracle(t, orderedAllPlay(w.allPlay())), marshalOracle(t, orderedAllPlay(r.Analysis.AllPlay)), "allPlay")
		if w.season("absent", 2020) != 0 || w.seasonWeeks(2030) != 13 {
			t.Fatal("missing season result")
		}
	})
	for _, fit := range r.Fits {
		t.Run(fmt.Sprintf("forecast/%d/%t", fit.Last, fit.PointInTime), func(t *testing.T) {
			var through *int64
			if fit.PointInTime {
				through = &fit.Last
			}
			f := newForecast(w, through)
			if f.last != fit.Last {
				t.Fatalf("last=%d want %d", f.last, fit.Last)
			}
			for _, coef := range fit.Coef {
				compareJSON(t, marshalOracle(t, f.coef[coefficientKey{coef.Group, coef.Horizon, coef.Fold}]), marshalOracle(t, coef.Values), fmt.Sprintf("coef/%s/%d/%d", coef.Group, coef.Horizon, coef.Fold))
			}
		})
	}
	f := newForecast(w, nil)
	v := newValuer(w, f, newPicks(w, r.Draft, f.last), known, estimated, 0)
	for name, samples := range r.Samples {
		t.Run(name, func(t *testing.T) {
			for i, s := range samples {
				compareJSON(t, runValuerOracle(t, v, name, s.Args), s.Result, fmt.Sprintf("%s[%d]", name, i))
			}
			t.Logf("samples=%d", len(samples))
		})
	}
	t.Run("tradeSides", func(t *testing.T) {
		rows := valueTrades(l, r.Trades, v, rookieSalaries(l, r.Draft))
		byIndex := map[int]tradeValue{}
		for _, row := range rows {
			byIndex[row.Index] = row
		}
		for _, want := range r.ValuedTrades {
			compareJSON(t, marshalOracle(t, []tradeValue{byIndex[want.Index]}), marshalOracle(t, []tradeValue{want}), fmt.Sprintf("trade/%d", want.Index))
		}
		t.Logf("sampled trade sides=%d", len(r.ValuedTrades))
	})
	a, _, err := analyse(t.Context(), l, r.Trades, r.Draft, known, estimated)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("analysis", func(t *testing.T) {
		a.AllPlay = orderedAllPlay(a.AllPlay)
		want := Analysis(r.Analysis)
		want.AllPlay = orderedAllPlay(want.AllPlay)
		compareJSON(t, marshalOracle(t, a), marshalOracle(t, want), "analysis")
	})
	bt, err := backtest(t.Context(), l, r.Trades, r.Draft, known, estimated, w)
	if err != nil {
		t.Fatal(err)
	}
	tuples := make([][3]float64, len(bt.Rows))
	for i, row := range bt.Rows {
		tuples[i] = [3]float64{row.Expected, row.Realized, float64(row.Season)}
	}
	t.Run("backtestTrades", func(t *testing.T) {
		compareJSON(t, marshalOracle(t, tuples), marshalOracle(t, r.BacktestRows), "backtestRows")
	})
}
func runValuerOracle(t *testing.T, v *valuer, name string, args []json.RawMessage) json.RawMessage {
	t.Helper()
	season := argument[int64](args, 1)
	switch name {
	case "player":
		return marshalOracle(t, v.player(argument[string](args, 0), season, int(argument[int64](args, 2))))
	case "realized_player":
		return marshalOracle(t, v.realizedPlayer(argument[string](args, 0), season, int(argument[int64](args, 2))))
	default:
		var a Asset
		if err := json.Unmarshal(args[0], &a); err != nil {
			t.Fatal(err)
		}
		if name == "pick" {
			return marshalOracle(t, v.pick(a, season))
		}
		return marshalOracle(t, v.realizedPick(a, season))
	}
}

// Python's all-play dictionary has identity keys, not an observable row order.
func orderedAllPlay(rows []AnalysisTeam) []AnalysisTeam {
	rows = slices.Clone(rows)
	slices.SortFunc(rows, func(a, b AnalysisTeam) int {
		if a.Year != b.Year {
			return int(a.Year - b.Year)
		}
		return strings.Compare(a.Fid, b.Fid)
	})
	return rows
}
