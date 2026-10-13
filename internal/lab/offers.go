package lab

import (
	"fmt"
	"math"
	"slices"
)

type OfferScope struct {
	From      int      `json:"from"`
	To        int      `json:"to"`
	Direction string   `json:"direction"`
	Windows   []string `json:"windows"`
	Team      string   `json:"team"`
}
type OfferPhase struct {
	ID     string `json:"id"`
	Window string `json:"window"`
	Label  string `json:"label"`
}

func OfferPhases() []OfferPhase {
	return []OfferPhase{{"winter", "spring", "After the season"}, {"opening", "spring", "Before the NFL draft"}, {"between", "spring", "NFL draft to rookie draft"}, {"rookieDraft", "summer", "During the rookie draft"}, {"summer", "summer", "Rookie draft to kickoff"}, {"early", "season", "Weeks 1–4"}, {"midseason", "season", "Midseason"}, {"deadline", "season", "Deadline run-in"}, {"closed", "season", "After the trade deadline"}}
}

type OfferStretch struct {
	Phase     string         `json:"phase"`
	N         int            `json:"n"`
	PerSeason float64        `json:"perSeason"`
	Counts    map[string]int `json:"counts"`
	Known     int            `json:"known"`
	Rate      *float64       `json:"rate"`
	SE        *float64       `json:"se"`
	Best      bool           `json:"best"`
	Worst     bool           `json:"worst"`
}
type OfferResult struct {
	Owner     string         `json:"owner"`
	Direction string         `json:"dir"`
	Offers    []Offer        `json:"offers"`
	Stretches []OfferStretch `json:"stretches"`
	Seasons   int            `json:"seasons"`
	Rate      *float64       `json:"rate"`
}

// Ties follow first appearance in the archived offers, matching the reference's stable Map sort.
func OfferOwner(d *Data) string {
	if d.OfferOwnerID != "" {
		return d.OfferOwnerID
	}
	counts := make(map[string]int)
	order := []string{}
	for _, o := range d.Offers {
		for _, id := range []string{o.By, o.To} {
			if counts[id] == 0 {
				order = append(order, id)
			}
			counts[id]++
		}
	}
	owner := ""
	for _, id := range order {
		if counts[id] > counts[owner] {
			owner = id
		}
	}
	return owner
}
func OfferStretches(d *Data, s OfferScope) OfferResult { return offerStretches(d, s, 10, "unknown") }
func offerStretches(d *Data, s OfferScope, minimum int, unknownEnding string) OfferResult {
	out := OfferResult{Owner: OfferOwner(d), Direction: s.Direction, Offers: []Offer{}, Stretches: []OfferStretch{}}
	if out.Direction == "" {
		out.Direction = "all"
	}
	out.Offers, out.Seasons = offersInScope(d, s, out.Owner, out.Direction)
	known, accepted := 0, 0
	for _, p := range OfferPhases() {
		if s.Windows != nil && !slices.Contains(s.Windows, p.Window) {
			continue
		}
		st := countOfferStretch(out.Offers, p.ID, out.Seasons, unknownEnding)
		if p.ID == "closed" && st.N == 0 {
			continue
		}
		known += st.Known
		accepted += st.Counts["accepted"]
		out.Stretches = append(out.Stretches, st)
	}
	if known > 0 {
		out.Rate = ptr(float64(accepted) / float64(known))
	}
	offerExtremes(out.Stretches, minimum)
	return out
}
func offerExtremes(st []OfferStretch, minimum int) {
	usable := []int{}
	for i, s := range st {
		if s.Known >= minimum {
			usable = append(usable, i)
		}
	}
	if len(usable) < 2 {
		return
	}
	hi, lo := usable[0], usable[0]
	for _, i := range usable {
		if *st[i].Rate > *st[hi].Rate {
			hi = i
		}
		if *st[i].Rate < *st[lo].Rate {
			lo = i
		}
	}
	best, worst := true, true
	for _, i := range usable {
		if i != hi && *st[hi].Rate-*st[i].Rate <= 1.96*math.Hypot(*st[hi].SE, *st[i].SE) {
			best = false
		}
		if i != lo && *st[i].Rate-*st[lo].Rate <= 1.96*math.Hypot(*st[lo].SE, *st[i].SE) {
			worst = false
		}
	}
	st[hi].Best = best
	st[lo].Worst = worst
}

func offersInScope(d *Data, s OfferScope, owner, direction string) ([]Offer, int) {
	offers := []Offer{}
	phases := OfferPhases()
	years := make(map[int]bool)
	for _, o := range d.Offers {
		if !offerVisibleTo(o, owner) {
			continue
		}
		if o.Season < s.From || o.Season > s.To {
			continue
		}
		years[o.Season] = true
		p := slices.IndexFunc(phases, func(p OfferPhase) bool { return p.ID == o.Phase })
		if p < 0 || (s.Windows != nil && !slices.Contains(s.Windows, phases[p].Window)) {
			continue
		}
		if !offerDirectionMatches(o, owner, direction) {
			continue
		}
		if s.Team != "" && s.Team != owner && o.By != s.Team && o.To != s.Team {
			continue
		}
		offers = append(offers, o)
	}
	return offers, max(1, len(years))
}

func countOfferStretch(offers []Offer, phase string, seasons int, unknownEnding string) OfferStretch {
	st := OfferStretch{Phase: phase, Counts: map[string]int{"accepted": 0, "rejected": 0, "withdrawn": 0, "expired": 0, "unknown": 0}}
	for _, o := range offers {
		if o.Phase == phase {
			st.N++
			outcome := o.Outcome
			if outcome == "unknown" {
				outcome = unknownEnding
			}
			st.Counts[outcome]++
		}
	}
	st.Known = st.N - st.Counts["unknown"] - st.Counts["pending"]
	st.PerSeason = float64(st.N) / float64(seasons)
	if st.Known > 0 {
		rate := float64(st.Counts["accepted"]) / float64(st.Known)
		st.Rate = ptr(rate)
		st.SE = ptr(math.Sqrt(math.Max(rate*(1-rate), 1e-9) / float64(st.Known)))
	}
	return st
}

func validateOfferValues(r RawOfferColumns) error {
	for i, outcome := range r.Outcome {
		if !slices.Contains([]string{"accepted", "rejected", "withdrawn", "expired", "unknown"}, outcome) {
			return fmt.Errorf("offers[%d]: invalid outcome %q", i, outcome)
		}
		if r.By[i] == "" || r.To[i] == "" || r.Season[i] < 1900 || r.Season[i] > 2200 {
			return fmt.Errorf("offers[%d]: invalid endpoint or season", i)
		}
	}
	return nil
}

func offerVisibleTo(o Offer, owner string) bool { return o.By == owner || o.To == owner }

func offerDirectionMatches(o Offer, owner, direction string) bool {
	return (direction != "sent" || o.By == owner) && (direction != "received" || o.To == owner)
}
