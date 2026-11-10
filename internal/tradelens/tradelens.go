// Package tradelens reads a proposed trade the way each side would: every asset is worth what it
// adds to that side's best N under that side's own direction. It fits nothing. It blends the two
// values the model already gives a player (this season and long term) and counts with the
// league's own starter number, so two depth pieces never add up to a star.
package tradelens

import (
	"fmt"
	"math"
	"slices"
	"sort"
)

// Direction is where an owner is taking the franchise. The owner picks it; until then Read gives ours.
type Direction string

const (
	Contend Direction = "contend"
	Retool  Direction = "retool"
	Rebuild Direction = "rebuild"
)

// Valid reports whether d is one of the three directions.
func (d Direction) Valid() bool { return d == Contend || d == Retool || d == Rebuild }

// NowWeight is how much of a lens is this season's value; the rest is long-term value. The three
// weights are the league's own ruling (2026-10-31, tested with its owners); every lens reads them here.
func (d Direction) NowWeight() float64 {
	switch d {
	case Contend:
		return 0.65
	case Rebuild:
		return 0.30
	case Retool:
		return 0.50
	}
	return 0.50
}

// even is the size under which a lens calls a trade level: values show one decimal.
const even = 0.05

// Standing is a franchise's record and the value-weighted age of its best N.
type Standing struct {
	Wins, Losses, Ties int
	Age                float64 // NaN when no age is known
}

// Read is our read of a franchise that has not set a direction: the record decides, and roster
// age settles the middle. With no game played there is nothing to read, so it is a retool.
func Read(s Standing, medianAge float64) Direction {
	games := s.Wins + s.Losses + s.Ties
	if games == 0 {
		return Retool
	}
	pct := (float64(s.Wins) + float64(s.Ties)/2) / float64(games)
	switch {
	case pct >= 0.6:
		return Contend
	case pct <= 0.35:
		return Rebuild
	case !math.IsNaN(s.Age) && !math.IsNaN(medianAge) && s.Age > medianAge && pct >= 0.5:
		return Contend
	}
	return Retool
}

// Comp is one player a league really took where a pick sits, as the model values him today.
type Comp struct {
	Position     string
	Now, Dynasty float64
}

// Asset is one player or pick as the model values it. A pick is worth what its Comps would add to
// the roster holding it, on average, times Weight: the wait for its draft and the league's market.
// Comps holds the players taken where it sits, then each later round's: whoever holds it could
// always take what a later pick takes, so it is never worth less than one.
type Asset struct {
	Token      string
	Franchise  string
	Position   string
	Pick       bool
	Now        float64
	Dynasty    float64
	Valued     bool
	Words      string
	Age        float64
	AgeKnown   bool
	SignedNext bool // under contract next season, or its contract end is not known
	Comps      [][]Comp
	Weight     float64
}

// Slot is one position's starter bounds in the league's lineup.
type Slot struct {
	Position string
	Min, Max int
}

// Side is one franchise in the trade.
type Side struct {
	Franchise string
	Direction Direction
	Set       bool // the owner picked it; false is our read
}

// Scope names one proposed trade: what the franchise gives and gets, as league player ids or pick
// tokens. With no partner it asks for the franchise's own shelf.
type Scope struct {
	League    string
	Season    int
	Franchise string
	Partner   string
	Give, Get []string
}

// Input is one proposed trade and everything it is judged against.
type Input struct {
	N         int // the league's starter count
	Teams     int
	Slots     []Slot
	Assets    []Asset // every rostered player and held pick in the league
	You, Them Side
	Give, Get []string
}

// AssetValue is one asset on either shelf: the model's two values, and what it is worth to each
// side: what it holds in its owner's best N, and what it would add to the other side's.
type AssetValue struct {
	Token    string  `json:"token"`
	Now      float64 `json:"now"`
	Dynasty  float64 `json:"dynasty"`
	You      float64 `json:"you"`
	Them     float64 `json:"them"`
	Valued   bool    `json:"valued"`
	Words    string  `json:"words"`
	Age      float64 `json:"age"`
	AgeKnown bool    `json:"ageKnown"`
}

// Need is one lineup position next season: ok, thin or hole.
type Need struct {
	Position string `json:"position"`
	State    string `json:"state"`
	Have     int    `json:"have"`
	Starts   int    `json:"starts"`
}

// SideReading is one franchise's side of the trade.
type SideReading struct {
	Franchise string    `json:"franchise"`
	Direction Direction `json:"direction"`
	Set       bool      `json:"set"`
	NowWeight float64   `json:"nowWeight"`
	Before    float64   `json:"before"`
	After     float64   `json:"after"`
	Net       float64   `json:"net"`
	Out       float64   `json:"out"` // what leaving assets take from the best N
	In        float64   `json:"in"`  // what arriving assets add once those have left
	Needs     []Need    `json:"needs"`
	Spare     []string  `json:"spare"`
}

// Suggestion is one single change that keeps the trade acceptable to the other side.
type Suggestion struct {
	Kind  string  `json:"kind"` // trim, ask or sweeten
	Token string  `json:"token"`
	You   float64 `json:"you"`
	Them  float64 `json:"them"`
}

// Reading is the trade through both lenses.
type Reading struct {
	Preparing bool         `json:"preparing,omitempty"`
	League    string       `json:"league"`
	Season    int          `json:"season"`
	StarterN  int          `json:"starterN"`
	You       SideReading  `json:"you"`
	Them      SideReading  `json:"them"`
	Assets    []AssetValue `json:"assets"`
	Verdict   string       `json:"verdict"` // empty, both, balk or bad
	Tune      []Suggestion `json:"tune"`
	Notes     []string     `json:"notes"`
}

// maxTune is how many suggestions a reading carries.
const maxTune = 6

func (a Asset) lens(d Direction) float64 {
	if !a.Valued {
		return 0
	}
	return d.blend(a.Now, a.Dynasty)
}

func (d Direction) blend(now, dynasty float64) float64 {
	w := d.NowWeight()
	return w*now + (1-w)*dynasty
}

// replacement is each position's replacement level under a lens: the best player there who would
// not start anywhere in the league. Every lineup seats its minimum at each position, and its
// flexible spots are shared among positions by the room each has above its minimum. Scarcity
// lives here: a position deep in good players has a high level, so its players are worth less.
func replacement(in Input, d Direction) map[string]float64 {
	flex, room := in.N, 0
	for _, s := range in.Slots {
		flex, room = flex-s.Min, room+s.Max-s.Min
	}
	out := map[string]float64{}
	for _, s := range in.Slots {
		seats := float64(s.Min)
		if room > 0 {
			seats += float64(max(flex, 0)) * float64(s.Max-s.Min) / float64(room)
		}
		var league []float64
		for _, a := range in.Assets {
			if a.plays(s.Position) {
				league = append(league, a.lens(d))
			}
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(league)))
		if bench := int(math.Round(seats * float64(in.Teams))); bench < len(league) {
			out[s.Position] = league[bench]
		}
	}
	return out
}

// over is a player's lens value over his position's replacement level, never below nothing.
func (r roster) over(position string, now, dynasty float64) float64 {
	return max(0, r.side.Direction.blend(now, dynasty)-r.floor[position])
}

// worth is what a pick adds to a roster whose last counted player is worth bar: each player taken
// where it sits counts for what he would add, so a pick is priced on its chance, not its average.
func (r roster) worth(a Asset, bar float64) float64 {
	if !a.Valued {
		return 0
	}
	top := 0.0
	for _, group := range a.Comps {
		sum := 0.0
		for _, c := range group {
			sum += max(0, r.over(c.Position, c.Now, c.Dynasty)-bar)
		}
		top = max(top, sum/float64(max(len(group), 1)))
	}
	return a.Weight * top
}

func (a Asset) value() AssetValue {
	return AssetValue{Token: a.Token, Now: a.Now, Dynasty: a.Dynasty, Valued: a.Valued, Words: a.Words,
		Age: a.Age, AgeKnown: a.AgeKnown}
}

// best is the total of the n largest values and the smallest of them: the bar a newcomer must
// beat. With fewer than n values every newcomer counts, so the bar is zero.
func best(values []float64, n int) (float64, float64) {
	sorted := slices.Clone(values)
	sort.Sort(sort.Reverse(sort.Float64Slice(sorted)))
	total := 0.0
	for _, v := range sorted[:min(n, len(sorted))] {
		total += v
	}
	if len(sorted) < n || n < 1 {
		return total, 0
	}
	return total, sorted[n-1]
}

type roster struct {
	side   Side
	n      int
	assets map[string]Asset
	floor  map[string]float64 // each position's replacement level under this side's lens
}

// count is the roster's best N players under its own lens, each over his position's replacement
// level, with out removed and in added, plus what each pick it then holds is worth against them.
func (r roster) count(out map[string]bool, in []Asset) float64 {
	held := make([]Asset, 0, len(r.assets)+len(in))
	for token, a := range r.assets {
		if !out[token] {
			held = append(held, a)
		}
	}
	held = append(held, in...)
	values := make([]float64, 0, len(held))
	for _, a := range held {
		if !a.Pick && a.Valued {
			values = append(values, r.over(a.Position, a.Now, a.Dynasty))
		}
	}
	total, bar := best(values, r.n)
	for _, a := range held {
		if a.Pick {
			total += r.worth(a, bar)
		}
	}
	return total
}

func set(tokens []string) map[string]bool {
	out := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		out[t] = true
	}
	return out
}

func pick(all map[string]Asset, tokens []string) []Asset {
	out := make([]Asset, 0, len(tokens))
	for _, t := range tokens {
		out = append(out, all[t])
	}
	return out
}

// Evaluate reads the trade. Every token in Give must be on your roster and every token in Get on
// theirs, once each.
func Evaluate(in Input) (Reading, error) {
	out := Reading{StarterN: in.N, Assets: []AssetValue{}, Tune: []Suggestion{}, Notes: []string{}}
	if in.N < 1 || in.You.Franchise == "" || !in.You.Direction.Valid() {
		return out, fmt.Errorf("trade lens: choose a franchise in a league with a starter count")
	}
	you, them := rosters(in)
	paired := in.Them.Franchise != "" && in.Them.Franchise != in.You.Franchise && in.Them.Direction.Valid()
	if err := owned(you, in.Give); err != nil {
		return out, err
	}
	if err := owned(them, in.Get); err != nil || (!paired && len(in.Give)+len(in.Get) > 0) {
		return out, fmt.Errorf("trade lens: choose a partner who holds what you ask for")
	}
	out.You = sideReading(you, in, in.Give, pick(them.assets, in.Get))
	if paired {
		out.Them = sideReading(them, in, in.Get, pick(you.assets, in.Give))
	}
	out.Assets = shelves(you, them, paired)
	out.Verdict = verdict(len(in.Give)+len(in.Get), out.You.Net, out.Them.Net)
	if paired && len(in.Give)+len(in.Get) > 0 {
		out.Tune = tune(you, them, in)
	}
	return out, nil
}

// rosters splits the league's assets into the two sides' holdings.
func rosters(in Input) (roster, roster) {
	you := roster{side: in.You, n: in.N, assets: map[string]Asset{}, floor: replacement(in, in.You.Direction)}
	them := roster{side: in.Them, n: in.N, assets: map[string]Asset{}, floor: replacement(in, in.Them.Direction)}
	for _, a := range in.Assets {
		switch {
		case a.Franchise == in.You.Franchise:
			you.assets[a.Token] = a
		case in.Them.Franchise != "" && a.Franchise == in.Them.Franchise:
			them.assets[a.Token] = a
		}
	}
	return you, them
}

func owned(r roster, tokens []string) error {
	seen := map[string]bool{}
	for _, t := range tokens {
		if _, ok := r.assets[t]; !ok || seen[t] {
			return fmt.Errorf("trade lens: %q is not on that roster, or is named twice", t)
		}
		seen[t] = true
	}
	return nil
}

func sideReading(r roster, in Input, leaving []string, arriving []Asset) SideReading {
	out := SideReading{Franchise: r.side.Franchise, Direction: r.side.Direction, Set: r.side.Set,
		NowWeight: r.side.Direction.NowWeight()}
	gone := set(leaving)
	out.Before = r.count(nil, nil)
	without := r.count(gone, nil)
	out.After = r.count(gone, arriving)
	out.Out, out.In, out.Net = out.Before-without, out.After-without, out.After-out.Before
	out.Needs, out.Spare = needs(r, in)
	return out
}

// shelves values every asset on both rosters for both sides, your roster first, each in token order.
func shelves(you, them roster, paired bool) []AssetValue {
	var out []AssetValue
	yours, theirs := you.count(nil, nil), 0.0
	if paired {
		theirs = them.count(nil, nil)
	}
	for _, t := range tokens(you.assets) {
		a := you.assets[t]
		row := a.value()
		row.You = yours - you.count(map[string]bool{t: true}, nil)
		if paired {
			row.Them = them.count(nil, []Asset{a}) - theirs
		}
		out = append(out, row)
	}
	if !paired {
		return out
	}
	for _, t := range tokens(them.assets) {
		a := them.assets[t]
		row := a.value()
		row.You, row.Them = you.count(nil, []Asset{a})-yours, theirs-them.count(map[string]bool{t: true}, nil)
		out = append(out, row)
	}
	return out
}

func tokens(assets map[string]Asset) []string {
	out := make([]string, 0, len(assets))
	for t := range assets {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func verdict(objects int, you, them float64) string {
	switch {
	case objects == 0:
		return "empty"
	case you > -even && them > -even:
		return "both"
	case you > -even:
		return "balk"
	}
	return "bad"
}

// tune tries every single change and keeps those the other side would still take: trim one thing
// you send, ask for one more of theirs, or, when they are short, add one more of yours.
func tune(you, them roster, in Input) []Suggestion {
	net := func(give, get []string) (float64, float64) {
		mine := you.count(set(give), pick(them.assets, get)) - you.count(nil, nil)
		theirs := them.count(set(get), pick(you.assets, give)) - them.count(nil, nil)
		return mine, theirs
	}
	mine, theirs := net(in.Give, in.Get)
	used := set(append(slices.Clone(in.Give), in.Get...))
	var out []Suggestion
	if len(in.Give) > 1 {
		for i, t := range in.Give {
			m, th := net(slices.Delete(slices.Clone(in.Give), i, i+1), in.Get)
			if th > -even {
				out = append(out, Suggestion{Kind: "trim", Token: t, You: m, Them: th})
			}
		}
	}
	for _, t := range tokens(them.assets) {
		if used[t] || !them.assets[t].Valued {
			continue
		}
		m, th := net(in.Give, append(slices.Clone(in.Get), t))
		if th > -even && m > mine+even {
			out = append(out, Suggestion{Kind: "ask", Token: t, You: m, Them: th})
		}
	}
	if theirs <= -even {
		for _, t := range tokens(you.assets) {
			if used[t] || !you.assets[t].Valued {
				continue
			}
			m, th := net(append(slices.Clone(in.Give), t), in.Get)
			if th > -even {
				out = append(out, Suggestion{Kind: "sweeten", Token: t, You: m, Them: th})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].You > out[j].You })
	return out[:min(maxTune, len(out))]
}

func (a Asset) plays(position string) bool { return !a.Pick && a.Valued && a.Position == position }

// starterBar is the long-term value of the last starter-grade player at a position: the league's
// (teams x minimum starters)th best. A league with fewer valued players there has no bar.
func starterBar(in Input, slot Slot) (float64, bool) {
	var league []float64
	for _, a := range in.Assets {
		if a.plays(slot.Position) {
			league = append(league, a.Dynasty)
		}
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(league)))
	rank := max(in.Teams, 1) * slot.Min
	if len(league) < rank {
		return 0, false
	}
	return league[rank-1], true
}

// needs reads next season's lineup position by position. A starter-grade player is one of the
// league's top (teams x the position's minimum starters) at it by long-term value who is signed
// for next season. Fewer than the lineup must start is a hole; exactly that many, at a position
// that can start more, is thin. Spare is more starter-grade players than the lineup can start.
func needs(r roster, in Input) ([]Need, []string) {
	out, spare := []Need{}, []string{}
	for _, slot := range in.Slots {
		if slot.Min < 1 {
			continue
		}
		bar, ok := starterBar(in, slot)
		if !ok {
			continue
		}
		have := 0
		for _, a := range r.assets {
			if a.plays(slot.Position) && a.SignedNext && a.Dynasty >= bar {
				have++
			}
		}
		state := "ok"
		switch {
		case have < slot.Min:
			state = "hole"
		case have == slot.Min && slot.Max > slot.Min:
			state = "thin"
		}
		out = append(out, Need{Position: slot.Position, State: state, Have: have, Starts: slot.Min})
		if have > slot.Max {
			spare = append(spare, slot.Position)
		}
	}
	return out, spare
}
