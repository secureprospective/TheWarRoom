// Package modelrun values the league with the measurables model (plan Stage 7). For every
// rostered player it reads his record (internal/playerrecord): his league seasons, his draft
// capital and pre-NFL facts, all keyed by our permanent player ID; with the run's fitted params
// it computes on-field-now and dynasty, and writes them as one model run in history beside the
// board. MFL IDs appear only at the edges: the league's rosters name players by them, and the
// stored scores carry them for the screens that still read by them.
//
// Missing facts are normal: a player without a combine, a college season or a birth date is
// valued on what exists, and each score lists the inputs that fed it.
package modelrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// FeatureReader reads facts from history as of a date.
type FeatureReader interface {
	Features(ctx context.Context, q history.FeatureQuery) ([]history.Feature, error)
}

// Writer is where a model run is written.
type Writer interface {
	WriteModelRun(ctx context.Context, nr history.NewModelRun) (history.Run, bool, error)
}

// Records serves the player record and the production read through it, keyed by our ID.
type Records interface {
	Book(ctx context.Context, q playerrecord.Query) (playerrecord.Book, error)
}

// Held is the player record built from held feeds, MFL's listing and history's facts, with no
// identity store: an offline replay or a test. It resolves records exactly as the stored record.
type Held struct {
	Listing  playerrecord.Listing
	Features FeatureReader
}

// Book reads the held facts and resolves their records.
func (h Held) Book(ctx context.Context, q playerrecord.Query) (playerrecord.Book, error) {
	obs, err := Observations(ctx, h.Features, q.From, q.To, q.AsOf, q.Measures)
	if err != nil {
		return playerrecord.Book{}, err
	}
	b, err := playerrecord.HeldBook(h.Listing, obs, q.LastWeek)
	if err != nil {
		return playerrecord.Book{}, fmt.Errorf("modelrun: %w", err)
	}
	return b, nil
}

// Runner runs model passes. It reads league state and the player record and writes only model
// runs.
type Runner struct {
	state   state.Reader
	records Records
	out     Writer
	engine  string
}

// New wires a Runner. engine is the build label recorded on every run.
func New(st state.Reader, records Records, out Writer, engine string) (*Runner, error) {
	if st == nil || records == nil || out == nil || engine == "" {
		return nil, fmt.Errorf("modelrun: missing dependency (state=%t records=%t writer=%t engine=%t)",
			st != nil, records != nil, out != nil, engine != "")
	}
	return &Runner{state: st, records: records, out: out, engine: engine}, nil
}

// Spec is one pass: the season valued, the date facts are read as of, the params, and the
// league's last scoring week.
type Spec struct {
	Season     int
	AsOf       time.Time
	Params     params.Set
	LastWeek   int
	FreeAgents []string
	Refresh    bool
}

// Report is what one pass valued and which run holds it.
type Report struct {
	RunID             int64       `json:"runID"`
	Unchanged         bool        `json:"unchanged"` // the pass matched the latest model run, so nothing was written
	Scored            int         `json:"scored"`
	RosteredScored    int         `json:"rosteredScored"`
	FreeAgentScored   int         `json:"freeAgentScored"`
	RosteredExcluded  int         `json:"rosteredExcluded"`
	FreeAgentExcluded int         `json:"freeAgentExcluded"`
	Rookies           int         `json:"rookies"` // valued on the prior alone: no NFL season yet
	Excluded          []Exclusion `json:"excluded"`
	// Mislinked are players whose history record was another player's; they are valued on MFL's
	// facts and their college seasons.
	Mislinked []Exclusion `json:"mislinked"`
}

// Exclusion is a rostered player the pass could not value, with the reason.
type Exclusion struct {
	MFLID  string `json:"mflID"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// lookback is how many seasons before the valued one are read: enough for the oldest rostered
// veteran's last college season, which his prior reads.
const lookback = 16

// BookQuery is the read a pass makes of the record for spec.
func BookQuery(spec Spec) playerrecord.Query {
	lastWeek := map[int]int{}
	for y := spec.Season - lookback; y <= spec.Season; y++ {
		lastWeek[y] = spec.LastWeek
	}
	return playerrecord.Query{From: spec.Season - lookback, To: spec.Season, AsOf: spec.AsOf,
		Measures: model.RuntimeMeasures(), LastWeek: lastWeek}
}

// Run values every rostered player and writes the run.
func (r *Runner) Run(ctx context.Context, spec Spec) (Report, error) {
	book, err := r.records.Book(ctx, BookQuery(spec))
	if err != nil {
		return Report{}, fmt.Errorf("modelrun: read the player record: %w", err)
	}
	pass, err := newPass(book, spec)
	if err != nil {
		return Report{}, err
	}
	var rep Report
	var scores []history.ModelScore
	var inputs []playerInput
	universe, err := r.universe(book, spec.FreeAgents)
	if err != nil {
		return Report{}, err
	}
	for _, p := range universe {
		sc, in, reason := pass.value(p.id)
		if !rep.record(book, p, in, reason) {
			continue
		}
		scores, inputs = append(scores, sc), append(inputs, in)
	}
	if len(scores) == 0 {
		return Report{}, fmt.Errorf("modelrun: no player could be valued for season %d", spec.Season)
	}
	if err := r.tilt(book, scores, inputs, spec, pass); err != nil {
		return Report{}, err
	}
	hash, err := inputsHash(inputs, rep.Excluded)
	if err != nil {
		return Report{}, err
	}
	run, written, err := r.out.WriteModelRun(ctx, history.NewModelRun{
		Refresh: spec.Refresh, Season: spec.Season, AsOf: spec.AsOf, Engine: r.engine, InputsHash: hash, Scores: scores,
		Params: history.ParamSet{Params: spec.Params.Model(), Measures: model.RuntimeMeasures()},
	})
	if err != nil {
		return Report{}, fmt.Errorf("modelrun: write %d scores (season %d): %w", len(scores), spec.Season, err)
	}
	rep.RunID, rep.Unchanged, rep.Scored = run.ID, !written, len(scores)
	return rep, nil
}

// tilt applies the contract tilt to the rostered players' dynasty values. The league's rosters
// name players by MFL ID; each contract is filed under our ID before it is read.
func (r *Runner) tilt(book playerrecord.Book, scores []history.ModelScore, inputs []playerInput, spec Spec, ps pass) error {
	maxTilt, err := spec.Params.GetGlobal(params.KeyDynastyContractTilt)
	if err != nil {
		return fmt.Errorf("modelrun: %w", err)
	}
	roster := map[string]state.PlayerState{}
	for _, fid := range r.state.Franchises() {
		players, _ := r.state.Roster(fid)
		for _, p := range players {
			if id, ok := book.ID(p.MFLID); ok {
				roster[id] = p
			}
		}
	}
	ids := make([]string, len(inputs))
	for i, in := range inputs {
		ids[i] = in.ID
	}
	scale := func(pos domain.Position) func(float64) float64 {
		s := ps.scale(pos, spec.Season-1)
		if s.Len() == 0 {
			return nil
		}
		return s.Pct
	}
	tilted := contractTilt(scores, ids, roster, spec.Season, maxTilt, scale)
	for i := range scores {
		if _, ok := tilted[ids[i]]; ok {
			scores[i].Inputs = append(scores[i].Inputs, "contract_price")
		}
	}
	return nil
}

func (rep *Report) record(book playerrecord.Book, p universePlayer, in playerInput, reason string) bool {
	name := book.Records[p.id].Name
	if reason != "" {
		rep.Excluded = append(rep.Excluded, Exclusion{MFLID: p.mfl, Name: name, Reason: reason})
		if p.free {
			rep.FreeAgentExcluded++
		} else {
			rep.RosteredExcluded++
		}
		return false
	}
	if p.free {
		rep.FreeAgentScored++
	} else {
		rep.RosteredScored++
	}
	if len(in.Past) == 0 && in.Current.Games == 0 {
		rep.Rookies++
	}
	if in.Mislinked {
		rep.Mislinked = append(rep.Mislinked, Exclusion{MFLID: p.mfl, Name: name,
			Reason: "history's record is another player's: its first NFL season is far from MFL's draft year"})
	}
	return true
}

// universePlayer is one player the pass values: our ID ("" when no record holds his MFL link)
// and the MFL ID the league named him by.
type universePlayer struct {
	id, mfl string
	free    bool
}

// universe is every rostered player, then every free agent, each once, resolved to our ID.
func (r *Runner) universe(book playerrecord.Book, free []string) ([]universePlayer, error) {
	var out []universePlayer
	seen := map[string]bool{}
	add := func(mfl string, fa bool) {
		if !seen[mfl] {
			id, _ := book.ID(mfl)
			out = append(out, universePlayer{id: id, mfl: mfl, free: fa})
			seen[mfl] = true
		}
	}
	for _, fid := range r.state.Franchises() {
		roster, ok := r.state.Roster(fid)
		if !ok {
			return nil, fmt.Errorf("modelrun: franchise %q listed but has no roster (store drift)", fid)
		}
		for _, p := range roster {
			add(p.MFLID, false)
		}
	}
	for _, id := range free {
		add(id, true)
	}
	return out, nil
}

// pass is what every player in one run is valued with.
type pass struct {
	book    playerrecord.Book
	data    model.Data
	scales  map[domain.Position]map[int]model.Scale
	params  map[domain.Position]model.Params
	horizon model.Horizon
	season  int
	decay   float64 // the engine's age-pull rate, for positions valued by today's board rule
}

func newPass(book playerrecord.Book, spec Spec) (pass, error) {
	p := pass{book: book, data: book.Data, scales: book.Data.Scales(), params: map[domain.Position]model.Params{}, season: spec.Season}
	for _, pos := range model.Positions() {
		m, err := model.ParamsFrom(pos, func(key string) (float64, error) {
			return spec.Params.GetPosition(key, string(pos))
		})
		if err != nil {
			return pass{}, fmt.Errorf("modelrun: params at %s: %w", pos, err)
		}
		p.params[pos] = m
	}
	seasons, err := spec.Params.GetGlobal(params.KeyDynastySeasons)
	if err != nil {
		return pass{}, fmt.Errorf("modelrun: %w", err)
	}
	discount, err := spec.Params.GetGlobal(params.KeyDynastyDiscount)
	if err != nil {
		return pass{}, fmt.Errorf("modelrun: %w", err)
	}
	p.horizon = model.Horizon{Seasons: int(math.Round(seasons)), Discount: discount}
	if p.decay, err = spec.Params.GetGlobal(params.KeyLayer3DecayRate); err != nil {
		return pass{}, fmt.Errorf("modelrun: %w", err)
	}
	return p, nil
}

// playerInput is everything the model read for one player; the run's inputs hash covers it.
type playerInput struct {
	ID        string // our permanent ID
	MFLID     string
	Position  domain.Position
	Player    model.Player
	Past      []model.Past
	Current   model.Past
	Mislinked bool
	// TodayTotal, TodayCurrentPPG and TodayPull feed today's board rule where the fit did not
	// keep the model; this season's pace stands in only when there is no total last season.
	TodayTotal, TodayCurrentPPG, TodayPull float64
}

// value scores one player by our ID, or returns why it could not. Every input it reads is the
// player record's or filed under our ID in it.
func (ps pass) value(id string) (history.ModelScore, playerInput, string) {
	rec, ok := ps.book.Records[id]
	if !ok || !rec.Listed {
		return history.ModelScore{}, playerInput{}, "not in the players database (aggregate or unknown id)"
	}
	pos := rec.Position
	m, ok := ps.params[pos]
	if !ok {
		return history.ModelScore{}, playerInput{}, fmt.Sprintf("no model for position %q", pos)
	}
	pl := ps.book.Player(id)
	in := playerInput{ID: id, MFLID: rec.MFL, Position: pos, Player: pl, Current: model.Past{Year: ps.season},
		Mislinked: rec.Mislinked}
	for y := ps.season - 3; y <= ps.season; y++ {
		s := ps.data.Seasons[id][y]
		if s == nil || s.Games() == 0 {
			continue
		}
		past := model.Past{Year: y, Games: float64(s.Games()), Pct: ps.scale(pos, y).Pct(s.PPG())}
		if y == ps.season {
			in.Current = past
		} else {
			in.Past = append(in.Past, past)
		}
	}
	scale := ps.scale(pos, ps.season-1)
	if scale.Len() == 0 {
		return history.ModelScore{}, playerInput{}, fmt.Sprintf("no %s regulars in %d to read points from", pos, ps.season-1)
	}
	v := m.Value(&pl, in.Past, in.Current, ps.horizon, scale)
	if !m.BoardKept {
		pull, err := AgePull(pos, pl.AgeAt(ps.season), ps.decay)
		if err != nil {
			return history.ModelScore{}, playerInput{}, err.Error()
		}
		if last := ps.data.Seasons[id][ps.season-1]; last != nil && last.Games() > 0 {
			in.TodayTotal = last.PPG() * float64(last.Games())
		}
		if cur := ps.data.Seasons[id][ps.season]; cur != nil && cur.Games() > 0 {
			in.TodayCurrentPPG = cur.PPG()
		}
		in.TodayPull = pull
		v = model.TodayRule(v, in.TodayTotal, in.TodayCurrentPPG, pull, scale)
	}
	return history.ModelScore{MFLID: rec.MFL, Position: string(pos), Value: v, Inputs: inputNames(m, &pl, in)}, in, ""
}

// AgePull is the engine's age pull that today's board applies at age; an unknown age is the arc's
// centre, as the survival arc reads it.
func AgePull(pos domain.Position, age, decay float64) (float64, error) {
	if math.IsNaN(age) {
		age = model.ArcCenter
	}
	pull, err := engine.ApplyDecay(age, engine.PeakLimit(pos), decay)
	if err != nil {
		return 0, fmt.Errorf("modelrun: age pull: %w", err)
	}
	return pull, nil
}

// scale is the position's scale for a season; a season with too few regulars yet (the first
// weeks of the current one) borrows the season before.
func (ps pass) scale(pos domain.Position, year int) model.Scale {
	if s := ps.scales[pos][year]; s.Len() >= minRegulars {
		return s
	}
	return ps.scales[pos][year-1]
}

// minRegulars is the fewest regulars a season's scale is drawn from.
const minRegulars = 20

// inputNames lists the inputs that fed a player's value: each known prior input, his age, the
// contract he is on where m's survival arc weighs one, and each season of league points.
func inputNames(m model.Params, pl *model.Player, in playerInput) []string {
	out := []string{}
	x, known := pl.PriorInputs()
	for i, name := range model.PriorFeatures(pl.Position) {
		if known[i] && i < len(x) {
			out = append(out, name)
		}
	}
	if !pl.Birth.IsZero() {
		out = append(out, "age")
	}
	if m.ReadsContract() && pl.TenureAt(in.Current.Year).Known {
		out = append(out, "contract")
	}
	for _, s := range append(in.Past, in.Current) {
		if s.Games > 0 {
			out = append(out, "season."+strconv.Itoa(s.Year))
		}
	}
	return out
}

// inputsHash is the sha256 of every player's model input, in roster order, plus who was
// excluded and why.
func inputsHash(inputs []playerInput, excluded []Exclusion) (string, error) {
	enc, err := json.Marshal(struct {
		Inputs   []playerInput
		Excluded []Exclusion
	}{inputs, excluded})
	if err != nil {
		return "", fmt.Errorf("modelrun: encode inputs: %w", err)
	}
	sum := sha256.Sum256(enc)
	return hex.EncodeToString(sum[:]), nil
}

// Observations reads every value the model reads for seasons from..to, plus the facts with no
// season, as known at asOf.
func Observations(ctx context.Context, h FeatureReader, from, to int, asOf time.Time, measures []string) ([]model.Obs, error) {
	var obs []model.Obs
	for _, season := range append([]int{0}, seasonsBetween(from, to)...) {
		feats, err := h.Features(ctx, history.FeatureQuery{AsOf: asOf, Season: season, Measures: measures})
		if err != nil {
			return nil, fmt.Errorf("modelrun: read season %d: %w", season, err)
		}
		for _, f := range feats {
			obs = append(obs, model.Obs{Player: f.PlayerID, Season: f.Season, Week: f.Week, Measure: f.Measure,
				Value: f.Value, Text: f.Text})
		}
	}
	return obs, nil
}

func seasonsBetween(from, to int) []int {
	out := make([]int, 0, max(to-from+1, 0))
	for y := max(from, 1); y <= to; y++ {
		out = append(out, y)
	}
	return out
}
