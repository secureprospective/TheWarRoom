package race

import (
	"fmt"
	"slices"
	"strings"
)

// Key is one step of MFL's standingsSort.
type Key string

const (
	KeyPct     Key = "PCT"          // win percentage
	KeyH2H     Key = "H2H"          // record in games among the tied teams
	KeyDiv     Key = "DIVPCT"       // division win percentage
	KeyConf    Key = "CONFPCT"      // conference win percentage
	KeyAllPlay Key = "ALL_PLAY_PCT" // all-play win percentage
	KeyPoints  Key = "PTS"          // points for
	KeyPower   Key = "PWR"          // MFL's power rating
)

// ParseSort reads MFL's comma-separated order. A step the app does not know is an error: ranking
// on a guess would put the wrong team in the playoffs.
func ParseSort(raw string) ([]Key, error) {
	keys := []Key{}
	for _, part := range strings.Split(raw, ",") {
		k := Key(strings.TrimSpace(part))
		switch k {
		case "":
			continue
		case KeyPct, KeyH2H, KeyDiv, KeyConf, KeyAllPlay, KeyPoints, KeyPower:
			keys = append(keys, k)
		default:
			return nil, fmt.Errorf("race: the tiebreak order names %q, which the app does not count", k)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("race: the league lists no tiebreak order")
	}
	return keys, nil
}

// Outcome is what one step did to a tied group.
type Outcome int

const (
	// Skipped: the step does not apply to this group (they did not all meet, or are in
	// different divisions or conferences, or MFL's power rating is not held).
	Skipped  Outcome = iota
	Level            // every team had the same value
	Split            // the step separated the group
	Unbroken         // every step was level; the group stays in MFL's franchise order
)

// Value is a team's number at a step: a record for the percentage steps, an amount in
// hundredths for points and power.
type Value struct {
	Record    Record
	Amount    int64
	IsRecord  bool
	HasAmount bool
}

// Step is one step applied to a group, Depth 0 for the whole tie and deeper for a group a step
// split off, which restarts the order from its first tiebreak.
type Step struct {
	Depth   int
	Key     Key
	Teams   []string
	Values  []Value
	Outcome Outcome
}

// Tie is a group level on the order's first step, with every step it took to separate them.
type Tie struct {
	Teams []string // in the order the steps put them
	Steps []Step
}

// Ordering is a group in the order the league's steps put it, and the ties those steps broke.
type Ordering struct {
	Order []string
	Ties  []Tie
}

// Order ranks ids by the league's steps: the first step groups them, and every group still
// level on it is broken by the remaining steps in order. When a step splits a group, each part
// that is still tied restarts from the first tiebreak, so head-to-head is read among the teams
// still level. Head-to-head applies only when every tied team played every other; division and
// conference percentage only when all the tied teams share that division or conference.
func (t Table) Order(ids []string) Ordering {
	if len(t.keys) == 0 || len(ids) == 0 {
		return Ordering{Order: slices.Clone(ids), Ties: []Tie{}}
	}
	o := Ordering{Order: []string{}, Ties: []Tie{}}
	for _, tier := range t.tiers(t.keys[0], ids) {
		if len(tier) == 1 {
			o.Order = append(o.Order, tier...)
			continue
		}
		tie := Tie{Steps: []Step{}}
		tie.Teams = t.breakTie(tier, 0, &tie)
		o.Order = append(o.Order, tie.Teams...)
		o.Ties = append(o.Ties, tie)
	}
	return o
}

func (t Table) breakTie(group []string, depth int, tie *Tie) []string {
	for _, k := range t.keys[1:] {
		step := Step{Depth: depth, Key: k, Teams: slices.Clone(group)}
		if !t.applies(k, group) {
			tie.Steps = append(tie.Steps, step)
			continue
		}
		for _, id := range group {
			step.Values = append(step.Values, t.value(k, id, group))
		}
		tiers := t.tiers(k, group)
		if len(tiers) == 1 {
			step.Outcome = Level
			tie.Steps = append(tie.Steps, step)
			continue
		}
		step.Outcome = Split
		tie.Steps = append(tie.Steps, step)
		out := []string{}
		for _, tier := range tiers {
			if len(tier) == 1 {
				out = append(out, tier...)
			} else {
				out = append(out, t.breakTie(tier, depth+1, tie)...)
			}
		}
		return out
	}
	out := slices.Clone(group)
	slices.Sort(out)
	tie.Steps = append(tie.Steps, Step{Depth: depth, Teams: out, Outcome: Unbroken})
	return out
}

// tiers splits group by k, best first; ids keep their incoming order within a tier.
func (t Table) tiers(k Key, group []string) [][]string {
	sorted := slices.Clone(group)
	slices.SortStableFunc(sorted, func(a, b string) int { return -t.compare(k, a, b, group) })
	out := [][]string{}
	for i, id := range sorted {
		if i == 0 || t.compare(k, sorted[i-1], id, group) != 0 {
			out = append(out, []string{})
		}
		out[len(out)-1] = append(out[len(out)-1], id)
	}
	return out
}

func (t Table) compare(k Key, a, b string, group []string) int {
	va, vb := t.value(k, a, group), t.value(k, b, group)
	if va.IsRecord {
		return comparePct(va.Record, vb.Record)
	}
	switch {
	case va.Amount > vb.Amount:
		return 1
	case va.Amount < vb.Amount:
		return -1
	}
	return 0
}

func (t Table) applies(k Key, group []string) bool {
	first := t.lines[group[0]]
	for i, id := range group {
		l := t.lines[id]
		switch k {
		case KeyH2H:
			for _, other := range group[i+1:] {
				if t.HeadToHead(id, other).Games() == 0 {
					return false
				}
			}
		case KeyDiv:
			if l.Division == "" || l.Division != first.Division {
				return false
			}
		case KeyConf:
			if l.Conference == "" || l.Conference != first.Conference {
				return false
			}
		case KeyPower:
			if !l.HasPower {
				return false
			}
		case KeyPct, KeyAllPlay, KeyPoints:
		}
	}
	return true
}

func (t Table) value(k Key, id string, group []string) Value {
	l := t.lines[id]
	switch k {
	case KeyH2H:
		r := Record{}
		for _, other := range group {
			h := t.HeadToHead(id, other)
			r.W, r.L, r.T = r.W+h.W, r.L+h.L, r.T+h.T
		}
		return Value{Record: r, IsRecord: true}
	case KeyDiv:
		return Value{Record: l.InDivision, IsRecord: true}
	case KeyConf:
		return Value{Record: l.InConference, IsRecord: true}
	case KeyAllPlay:
		return Value{Record: l.AllPlay, IsRecord: true}
	case KeyPoints:
		return Value{Amount: int64(l.For), HasAmount: true}
	case KeyPower:
		return Value{Amount: l.Power, HasAmount: true}
	case KeyPct:
	}
	return Value{Record: l.Overall, IsRecord: true}
}
