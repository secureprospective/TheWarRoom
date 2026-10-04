// Package modelrun values the league with the measurables model (plan Stage 7). For every
// rostered player it reads his league seasons, his pre-NFL facts and the run's fitted params,
// computes on-field-now and dynasty, and writes them as one model run in history beside the
// board.
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
	"slices"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// FeatureReader reads facts from history as of a date.
type FeatureReader interface {
	Features(ctx context.Context, q history.FeatureQuery) ([]history.Feature, error)
}

// History is what a model run needs from the history store.
type History interface {
	FeatureReader
	WriteModelRun(ctx context.Context, nr history.NewModelRun) (history.Run, bool, error)
}

// Directory resolves a rostered mfl id to its players-DB facts. normalize.Lookup satisfies it.
type Directory interface {
	Facts(mflID string) (normalize.PlayerFacts, bool)
}

// Runner runs model passes. It reads league state and history and writes only model runs.
type Runner struct {
	state  state.Reader
	dir    Directory
	hist   History
	engine string
}

// New wires a Runner. engine is the build label recorded on every run.
func New(st state.Reader, dir Directory, hist History, engine string) (*Runner, error) {
	if st == nil || dir == nil || hist == nil || engine == "" {
		return nil, fmt.Errorf("modelrun: missing dependency (state=%t dir=%t history=%t engine=%t)",
			st != nil, dir != nil, hist != nil, engine != "")
	}
	return &Runner{state: st, dir: dir, hist: hist, engine: engine}, nil
}

// Spec is one pass: the season valued, the date facts are read as of, the params, and the
// league's last scoring week.
type Spec struct {
	Season   int
	AsOf     time.Time
	Params   params.Set
	LastWeek int
}

// Report is what one pass valued and which run holds it.
type Report struct {
	RunID     int64       `json:"runID"`
	Unchanged bool        `json:"unchanged"` // the pass matched the latest model run, so nothing was written
	Scored    int         `json:"scored"`
	Rookies   int         `json:"rookies"` // valued on the prior alone: no NFL season yet
	Excluded  []Exclusion `json:"excluded"`
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

// Run values every rostered player and writes the run.
func (r *Runner) Run(ctx context.Context, spec Spec) (Report, error) {
	obs, err := Observations(ctx, r.hist, spec.Season-lookback, spec.Season, spec.AsOf, model.RuntimeMeasures())
	if err != nil {
		return Report{}, err
	}
	lastWeek := map[int]int{}
	for y := spec.Season - lookback; y <= spec.Season; y++ {
		lastWeek[y] = spec.LastWeek
	}
	d := model.Build(obs, lastWeek)
	AtLeaguePositions(d, r.dir)
	pass, err := newPass(d, spec)
	if err != nil {
		return Report{}, err
	}
	var rep Report
	var scores []history.ModelScore
	var inputs []playerInput
	for _, fid := range r.state.Franchises() {
		roster, ok := r.state.Roster(fid)
		if !ok {
			return Report{}, fmt.Errorf("modelrun: franchise %q listed but has no roster (store drift)", fid)
		}
		for _, p := range roster {
			sc, in, reason := pass.value(r.dir, p.MFLID)
			if reason != "" {
				facts, _ := r.dir.Facts(p.MFLID)
				rep.Excluded = append(rep.Excluded, Exclusion{MFLID: p.MFLID, Name: facts.Name, Reason: reason})
				continue
			}
			if len(in.Past) == 0 && in.Current.Games == 0 {
				rep.Rookies++
			}
			if in.Mislinked {
				facts, _ := r.dir.Facts(p.MFLID)
				rep.Mislinked = append(rep.Mislinked, Exclusion{MFLID: p.MFLID, Name: facts.Name,
					Reason: "history's record is another player's: its first NFL season is far from MFL's draft year"})
			}
			scores, inputs = append(scores, sc), append(inputs, in)
		}
	}
	if len(scores) == 0 {
		return Report{}, fmt.Errorf("modelrun: no rostered player could be valued for season %d", spec.Season)
	}
	hash, err := inputsHash(inputs, rep.Excluded)
	if err != nil {
		return Report{}, err
	}
	run, written, err := r.hist.WriteModelRun(ctx, history.NewModelRun{
		Season: spec.Season, AsOf: spec.AsOf, Engine: r.engine, InputsHash: hash, Scores: scores,
		Params: history.ParamSet{Params: spec.Params.Model(), Measures: model.RuntimeMeasures()},
	})
	if err != nil {
		return Report{}, fmt.Errorf("modelrun: write %d scores (season %d): %w", len(scores), spec.Season, err)
	}
	rep.RunID, rep.Unchanged, rep.Scored = run.ID, !written, len(scores)
	return rep, nil
}

// pass is what every player in one run is valued with.
type pass struct {
	data    model.Data
	scales  map[domain.Position]map[int]model.Scale
	params  map[domain.Position]model.Params
	horizon model.Horizon
	season  int
}

func newPass(d model.Data, spec Spec) (pass, error) {
	p := pass{data: d, scales: d.Scales(), params: map[domain.Position]model.Params{}, season: spec.Season}
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
	return p, nil
}

// playerInput is everything the model read for one player; the run's inputs hash covers it.
type playerInput struct {
	MFLID     string
	Position  domain.Position
	Player    model.Player
	Past      []model.Past
	Current   model.Past
	Mislinked bool
}

// value scores one rostered player, or returns why it could not.
func (ps pass) value(dir Directory, mflID string) (history.ModelScore, playerInput, string) {
	facts, ok := dir.Facts(mflID)
	if !ok {
		return history.ModelScore{}, playerInput{}, "not in the players database (aggregate or unknown id)"
	}
	m, ok := ps.params[facts.Position]
	if !ok {
		return history.ModelScore{}, playerInput{}, fmt.Sprintf("no model for position %q", facts.Position)
	}
	pl, mislinked := ps.player(mflID, facts)
	in := playerInput{MFLID: mflID, Position: facts.Position, Player: pl, Current: model.Past{Year: ps.season},
		Mislinked: mislinked}
	for y := ps.season - 3; y <= ps.season; y++ {
		s := ps.data.Seasons[mflID][y]
		if s == nil || s.Games() == 0 {
			continue
		}
		past := model.Past{Year: y, Games: float64(s.Games()), Pct: ps.scale(facts.Position, y).Pct(s.PPG())}
		if y == ps.season {
			in.Current = past
		} else {
			in.Past = append(in.Past, past)
		}
	}
	scale := ps.scale(facts.Position, ps.season-1)
	if scale.Len() == 0 {
		return history.ModelScore{}, playerInput{}, fmt.Sprintf("no %s regulars in %d to read points from", facts.Position, ps.season-1)
	}
	v := m.Value(&pl, in.Past, in.Current, ps.horizon, scale)
	return history.ModelScore{MFLID: mflID, Position: string(facts.Position), Value: v, Inputs: inputNames(&pl, in)}, in, ""
}

// identitySlack is how many seasons history's first NFL season may sit from the year MFL has the
// player entering the league. On the 2026 roster 1,439 of 1,440 sit within one; the one outside
// sat 48 away.
const identitySlack = 2

// player is the history's facts for mflID, with the position he is rostered at; the players
// database fills a birth date history lacks. A history record whose first NFL season is far from
// MFL's draft year is another player's: the crosswalk tied him to the wrong ids (DynastyProcess
// gave a 2026 rookie DT the ids of a 1978 receiver). MFL wins: that record's birth date, rookie
// season, draft slot and measurements are set aside, his college seasons kept, and mislinked
// reports it.
func (ps pass) player(mflID string, facts normalize.PlayerFacts) (pl model.Player, mislinked bool) {
	pl = model.Player{ID: mflID}
	if h := ps.data.Players[mflID]; h != nil {
		pl = *h
		if facts.HasDraftYear && h.Rookie != 0 && abs(h.Rookie-facts.DraftYear) > identitySlack {
			pl, mislinked = model.Player{ID: mflID, College: h.College}, true
		}
	}
	pl.Position = facts.Position
	if pl.Birth.IsZero() && facts.HasBirthdate {
		pl.Birth = time.Unix(facts.Birthdate, 0).UTC()
	}
	return pl, mislinked
}

func abs(n int) int { return max(n, -n) }

// AtLeaguePositions moves each history player the league's players database lists at a position
// the model scores to that position, and returns how many moved. The league scores by its own
// position (a DE's tackle is worth 2.5, an LB's 1.5), so nflverse's label would put an edge
// rusher it lists as LB in the linebackers' scale and the linebackers' fit. A player the league
// does not list keeps nflverse's position.
func AtLeaguePositions(d model.Data, dir Directory) int {
	moved := 0
	for id, pl := range d.Players {
		facts, ok := dir.Facts(id)
		if !ok || facts.Position == pl.Position || !slices.Contains(model.Positions(), facts.Position) {
			continue
		}
		pl.Position = facts.Position
		moved++
	}
	return moved
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

// inputNames lists the inputs that fed a player's value: each known prior input and each
// season of league points.
func inputNames(pl *model.Player, in playerInput) []string {
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
