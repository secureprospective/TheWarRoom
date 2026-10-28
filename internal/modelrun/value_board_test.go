//go:build valueboard

package modelrun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawBoardRequest struct {
	Season    int
	LastWeek  int
	AsOf      time.Time
	Roster    []state.PlayerState
	Overrides map[string]float64
}

func (r RawBoardRequest) Validate() error {
	if r.Season < 2000 || r.Season > 2100 || r.LastWeek < 1 || r.LastWeek > 18 || r.AsOf.IsZero() || len(r.Roster) == 0 {
		return fmt.Errorf("value board: invalid replay period or empty roster")
	}
	for _, p := range r.Roster {
		id, err := playerid.New(p.MFLID)
		if err != nil {
			return fmt.Errorf("value board: roster ID: %w", err)
		}
		if id.String() != p.MFLID {
			return fmt.Errorf("value board: non-canonical roster ID %q", p.MFLID)
		}
	}
	for _, v := range r.Overrides {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("value board: non-finite override")
		}
	}
	return nil
}

type boardCapture struct {
	FeatureReader
	run history.NewModelRun
}

func (c *boardCapture) WriteModelRun(_ context.Context, run history.NewModelRun) (history.Run, bool, error) {
	c.run = run
	return history.Run{ID: 1, Kind: history.RunModel}, true, nil
}

type boardPlayer struct {
	Score      history.ModelScore
	Facts      normalize.PlayerFacts
	Age        *float64
	LastPoints *float64
	ThisPoints *float64
	LastGames  int
	CurrentPPG *float64
	Rule       string
	AgePull    *float64
}

type boardResult struct {
	Report     Report
	Players    []boardPlayer
	Parameters map[string]float64
}

// TestValueBoardReplay calls the production runner, capturing its write in memory. Only the
// held features are read from a disposable export; no database writer or fetcher is involved.
func TestValueBoardReplay(t *testing.T) {
	root := boardRoot(t)
	var req RawBoardRequest
	boardReadJSON(t, root, &req)
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	body, err := root.ReadFile("players.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := players.Parse(t.Context(), body)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	features := boardFeatures(t, root)
	capture := &boardCapture{FeatureReader: features}
	runner, err := New(fakeState{roster: req.Roster}, Held{dir, features}, capture, "offline-value-board")
	if err != nil {
		t.Fatal(err)
	}
	effective := params.DefaultSet().Values()
	for k, v := range req.Overrides {
		if _, ok := effective[k]; !ok {
			t.Fatalf("value board: unknown override %s", k)
		}
		effective[k] = v
	}
	set := params.SetOf(effective)
	spec := Spec{Season: req.Season, AsOf: req.AsOf, Params: set, LastWeek: req.LastWeek, FreeAgents: boardFreeAgents(t, root)}
	report, err := runner.Run(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	book, err := Held{dir, features}.Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	pass, err := newPass(book, spec)
	if err != nil {
		t.Fatal(err)
	}
	data := book.Data
	result := boardResult{Report: report, Parameters: effective}
	for _, sc := range capture.run.Scores {
		facts, _ := dir.Facts(sc.MFLID)
		id, _ := book.ID(sc.MFLID)
		pl := book.Player(id)
		row := boardPlayer{Score: sc, Facts: facts, Rule: "Fitted model"}
		age := pl.AgeAt(req.Season)
		if !math.IsNaN(age) {
			row.Age = &age
		}
		if !pass.params[facts.Position].BoardKept {
			row.Rule = "Last-season points × age pull ÷ 17"
			pull, err := AgePull(facts.Position, pl.AgeAt(req.Season), pass.decay)
			if err != nil {
				t.Fatal(err)
			}
			row.AgePull = &pull
		}
		seasons := data.Seasons[id]
		if s := seasons[req.Season-1]; s != nil {
			row.LastGames = s.Games()
			row.LastPoints = boardPoints(s)
		}
		if s := seasons[req.Season]; s != nil {
			row.ThisPoints = boardPoints(s)
			if s.HasTotal && s.Games() > 0 {
				ppg := s.PPG()
				row.CurrentPPG = &ppg
			}
		}
		result.Players = append(result.Players, row)
	}
	body, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("replay.json", body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func boardPoints(s *model.Season) *float64 {
	if s.HasTotal {
		v := s.Total
		return &v
	}
	return nil
}

func boardReadJSON(t *testing.T, root *os.Root, out *RawBoardRequest) {
	t.Helper()
	body, err := root.ReadFile("request.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		t.Fatal(err)
	}
}

func TestValueBoardMeasures(t *testing.T) {
	root := boardRoot(t)
	body, err := json.Marshal(struct {
		Measures []string
		Lookback int
	}{model.RuntimeMeasures(), lookback})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("measures.json", body, 0o600); err != nil {
		t.Fatal(err)
	}
}

type boardFeatureReader map[int][]history.Feature

func (f boardFeatureReader) Features(_ context.Context, q history.FeatureQuery) ([]history.Feature, error) {
	var out []history.Feature
	for _, row := range f[q.Season] {
		if !row.AsOf.After(q.AsOf) && slices.Contains(q.Measures, row.Measure) {
			out = append(out, row)
		}
	}
	return out, nil
}

func boardFeatures(t *testing.T, root *os.Root) boardFeatureReader {
	t.Helper()
	file, err := root.Open("features.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	}()
	out := boardFeatureReader{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var row history.Feature
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatal(err)
		}
		if row.PlayerID == "" || row.Measure == "" || row.AsOf.IsZero() || math.IsNaN(row.Value) || math.IsInf(row.Value, 0) {
			t.Fatal("value board: invalid held feature")
		}
		out[row.Season] = append(out[row.Season], row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func boardRoot(t *testing.T) *os.Root {
	t.Helper()
	path := os.Getenv("TWR_VALUE_BOARD_DIR")
	if path == "" {
		t.Skip("use scripts/value_board.py for offline replay")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	return root
}

func boardFreeAgents(t *testing.T, root *os.Root) []string {
	t.Helper()
	body, err := root.ReadFile("free-agents.json")
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	raw, err := freeagents.Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := freeagents.ToPlayerIDs(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	slices.Sort(out)
	return out
}

func TestValueBoardFreeAgentArchiveUsesProductionParser(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	if got := boardFreeAgents(t, root); len(got) != 0 {
		t.Fatalf("missing feed: %v", got)
	}
	body := []byte(`{"freeAgents":{"leagueUnit":[{"player":{"id":"0001"}},{"player":[{"id":"0001"},{"id":"0002"}]}]}}`)
	if err := root.WriteFile("free-agents.json", body, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := boardFreeAgents(t, root); !slices.Equal(got, []string{"0001", "0002"}) {
		t.Fatalf("parsed free agents = %v", got)
	}
}

func TestValueBoardRequestValidation(t *testing.T) {
	valid := RawBoardRequest{Season: 2026, LastWeek: 17, AsOf: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		Roster: []state.PlayerState{{MFLID: "0816"}}}
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*RawBoardRequest){
		"season":       func(r *RawBoardRequest) { r.Season = 0 },
		"last week":    func(r *RawBoardRequest) { r.LastWeek = 19 },
		"cutoff":       func(r *RawBoardRequest) { r.AsOf = time.Time{} },
		"empty roster": func(r *RawBoardRequest) { r.Roster = nil },
		"short ID":     func(r *RawBoardRequest) { r.Roster = []state.PlayerState{{MFLID: "816"}} },
		"invalid ID":   func(r *RawBoardRequest) { r.Roster = []state.PlayerState{{MFLID: "not-a-player"}} },
		"non-finite":   func(r *RawBoardRequest) { r.Overrides = map[string]float64{"dynasty.discount": math.NaN()} },
	} {
		t.Run(name, func(t *testing.T) {
			request := valid
			mutate(&request)
			if err := request.Validate(); err == nil {
				t.Fatal("invalid replay request accepted")
			}
		})
	}
}

func TestValueBoardPointsDontInferZeroFromSnaps(t *testing.T) {
	season := &model.Season{Points: map[int]float64{1: 0}}
	if got := boardPoints(season); got != nil {
		t.Fatalf("missing total shown as %v", *got)
	}
	season.HasTotal = true
	if got := boardPoints(season); got == nil || *got != 0 {
		t.Fatalf("explicit zero total lost: %v", got)
	}
	season.Total = 12.5
	if got := boardPoints(season); got == nil || *got != 12.5 {
		t.Fatalf("known total lost: %v", got)
	}
}

// TestPickValueReplay fits Pick Value v1 on the replayed values and the replay's records, logs
// the slot table and pins its scale on the real league: an early 1st lands among solid starters
// (70th–95th percentile of the valued players), not superstars; a late 3rd among bench players
// (20th–55th).
func TestPickValueReplay(t *testing.T) {
	root := boardRoot(t)
	var req RawBoardRequest
	boardReadJSON(t, root, &req)
	body, err := root.ReadFile("players.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := players.Parse(t.Context(), body)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := normalize.NewLookup(raw)
	if err != nil {
		t.Fatal(err)
	}
	features := boardFeatures(t, root)
	capture := &boardCapture{FeatureReader: features}
	runner, err := New(fakeState{roster: req.Roster}, Held{dir, features}, capture, "offline-value-board")
	if err != nil {
		t.Fatal(err)
	}
	set := params.DefaultSet()
	spec := Spec{Season: req.Season, AsOf: req.AsOf, Params: set, LastWeek: req.LastWeek, FreeAgents: boardFreeAgents(t, root)}
	if _, err := runner.Run(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	book, err := Held{dir, features}.Book(t.Context(), BookQuery(spec))
	if err != nil {
		t.Fatal(err)
	}
	valued := map[string]float64{}
	var values []float64
	for _, sc := range capture.run.Scores {
		valued[sc.MFLID] = sc.DynastyPPG
		values = append(values, sc.DynastyPPG)
	}
	slices.Sort(values)
	var cohort []pickvalue.Player
	for _, r := range book.Records {
		v, ok := valued[r.MFL]
		cohort = append(cohort, pickvalue.Player{Draft: r.Draft, Value: v, Valued: ok})
	}
	discount, err := set.GetGlobal(params.KeyDynastyDiscount)
	if err != nil {
		t.Fatal(err)
	}
	hist, err := pickvalue.History()
	if err != nil {
		t.Fatal(err)
	}
	m, err := pickvalue.Fit(hist, cohort, req.Season, discount)
	if err != nil {
		t.Fatal(err)
	}
	pct := func(v float64) float64 {
		i, _ := slices.BinarySearch(values, v)
		return 100 * float64(i) / float64(len(values))
	}
	t.Logf("valued players: p50 %.2f · p75 %.2f · p90 %.2f · p98 %.2f dynasty PPG", values[len(values)/2],
		values[len(values)*75/100], values[len(values)*90/100], values[len(values)*98/100])
	type row struct {
		Pick                        string
		PPG, Pct, BandLow, BandHigh float64
		Undrafted                   float64
		Players                     int
	}
	var table []row
	for r := 1; r <= 5; r++ {
		for _, s := range []int{1, 3, 8, 16, 24, 30, 0} {
			v, err := m.Pick(req.Season, r, s, 32)
			if err != nil {
				t.Fatal(err)
			}
			name := fmt.Sprintf("%d.%02d", r, s)
			if s == 0 {
				name = fmt.Sprintf("round %d", r)
			}
			table = append(table, row{name, v.PPG, pct(v.PPG), v.BandLow, v.BandHigh, v.Undrafted, v.Players})
			t.Logf("%-8s %6.2f PPG  p%-3.0f NFL %3.0f–%3.0f  undrafted %3.0f%%  n=%d", name, v.PPG, pct(v.PPG), v.BandLow,
				v.BandHigh, 100*v.Undrafted, v.Players)
		}
	}
	for y := 1; y <= 2; y++ {
		v, _ := m.Pick(req.Season+y, 1, 0, 32)
		t.Logf("round 1, %d (%d out): %.2f PPG", req.Season+y, y, v.PPG)
	}
	early, _ := m.Pick(req.Season, 1, 3, 32)
	late, _ := m.Pick(req.Season, 3, 30, 32)
	if p := pct(early.PPG); p < 70 || p > 95 {
		t.Errorf("early 1st at p%.0f, want a solid starter (p70–p95)", p)
	}
	if p := pct(late.PPG); p < 20 || p > 55 {
		t.Errorf("late 3rd at p%.0f, want a bench player (p20–p55)", p)
	}
	body, err = json.Marshal(table)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.WriteFile("pick-values.json", body, 0o600); err != nil {
		t.Fatal(err)
	}
}
