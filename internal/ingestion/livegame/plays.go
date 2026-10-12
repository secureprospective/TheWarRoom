package livegame

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"
)

type RawReference struct {
	ID  string `json:"id"`
	Ref string `json:"$ref"`
}
type RawState struct {
	Down      *int         `json:"down"`
	Distance  *int         `json:"distance"`
	YardLine  *int         `json:"yardLine"`
	ToEndzone *int         `json:"yardsToEndzone"`
	Team      RawReference `json:"team"`
}
type RawPlay struct {
	ID        string `json:"id"`
	Sequence  string `json:"sequenceNumber"`
	Text      string `json:"text"`
	HomeScore *int   `json:"homeScore"`
	AwayScore *int   `json:"awayScore"`
	Period    struct {
		Number *int `json:"number"`
	} `json:"period"`
	Clock struct {
		DisplayValue string `json:"displayValue"`
	} `json:"clock"`
	Wallclock    string    `json:"wallclock"`
	Start        *RawState `json:"start"`
	End          *RawState `json:"end"`
	Participants []struct {
		Athlete RawReference `json:"athlete"`
	} `json:"participants"`
}
type RawCore struct {
	Count     *int      `json:"count"`
	PageCount *int      `json:"pageCount"`
	Items     []RawPlay `json:"items"`
}

func (r RawState) Validate() error {
	for _, v := range []struct {
		name   string
		value  *int
		lo, hi int
	}{
		{"down", r.Down, -1, 4}, {"distance", r.Distance, 0, 100}, {"yardLine", r.YardLine, 0, 100}, {"yardsToEndzone", r.ToEndzone, 0, 100},
	} {
		if v.value == nil {
			return fmt.Errorf("%s: missing", v.name)
		}
		if *v.value < v.lo || *v.value > v.hi {
			return fmt.Errorf("%s: out of range", v.name)
		}
	}
	switch {
	case r.Team.Ref != "":
		if _, err := referenceID(r.Team.Ref, "teams"); err != nil {
			return err
		}
	case r.Team.ID == "" && *r.Down == 0 && *r.Distance == 0 && *r.YardLine == 0 && *r.ToEndzone == 0:
		return nil
	case !digits(r.Team.ID):
		return fmt.Errorf("team.id: invalid")
	}
	return nil
}
func (r RawPlay) validateScoreClock() error {
	for name, v := range map[string]*int{"homeScore": r.HomeScore, "awayScore": r.AwayScore, "period": r.Period.Number} {
		if v == nil {
			return fmt.Errorf("play.%s: missing", name)
		}
		maxValue := 200
		minValue := 0
		if name == "period" {
			maxValue = 10
			minValue = 1
		}
		if *v < minValue || *v > maxValue {
			return fmt.Errorf("play.%s: out of range", name)
		}
	}
	if !clockOK(r.Clock.DisplayValue) {
		return fmt.Errorf("play.clock: invalid")
	}
	if r.Wallclock != "" {
		if _, err := time.Parse(time.RFC3339, r.Wallclock); err != nil {
			return fmt.Errorf("play.wallclock: %w", err)
		}
	}
	return nil
}
func (r RawPlay) Validate() error {
	if !digits(r.ID) || !digits(r.Sequence) || r.Text == "" {
		return fmt.Errorf("play.id/sequenceNumber/text: missing or invalid")
	}
	if _, err := strconv.Atoi(r.Sequence); err != nil {
		return fmt.Errorf("sequenceNumber: %w", err)
	}
	if err := r.validateScoreClock(); err != nil {
		return err
	}
	for name, state := range map[string]*RawState{"start": r.Start, "end": r.End} {
		if state != nil {
			if err := state.Validate(); err != nil {
				return fmt.Errorf("play.%s: %w", name, err)
			}
		}
	}
	for i, p := range r.Participants {
		if _, err := referenceID(p.Athlete.Ref, "athletes"); err != nil {
			return fmt.Errorf("participants[%d].athlete: %w", i, err)
		}
	}
	return nil
}
func (r RawCore) Validate() error {
	if r.Count == nil || r.PageCount == nil || *r.PageCount != 1 || *r.Count != len(r.Items) {
		return fmt.Errorf("core.count/pageCount/items: incomplete play page")
	}
	return validatePlays(r.Items)
}
func validatePlays(plays []RawPlay) error {
	seen := map[string]bool{}
	for i, p := range plays {
		if seen[p.ID] {
			return fmt.Errorf("plays[%d].id: duplicate", i)
		}
		seen[p.ID] = true
		if err := p.Validate(); err != nil {
			return fmt.Errorf("plays[%d]: %w", i, err)
		}
	}
	return nil
}
func (r *RawState) state() *State {
	if r == nil {
		return nil
	}
	id := r.Team.ID
	if r.Team.Ref != "" {
		id, _ = referenceID(r.Team.Ref, "teams")
	}
	return &State{Possession: id, Down: *r.Down, Distance: *r.Distance, YardLine: *r.YardLine, ToEndzone: *r.ToEndzone}
}
func (r RawPlay) play() Play {
	n, _ := strconv.Atoi(r.Sequence)
	p := Play{ID: r.ID, Sequence: n, Text: r.Text, Clock: r.Clock.DisplayValue, Period: *r.Period.Number, HomeScore: *r.HomeScore, AwayScore: *r.AwayScore, Start: r.Start.state(), End: r.End.state(), Participants: []string{}}
	if r.Wallclock != "" {
		p.At, _ = time.Parse(time.RFC3339, r.Wallclock)
	}
	for _, v := range r.Participants {
		id, _ := referenceID(v.Athlete.Ref, "athletes")
		if !slices.Contains(p.Participants, id) {
			p.Participants = append(p.Participants, id)
		}
	}
	return p
}
func ParseCore(body []byte) ([]Play, error) {
	var r RawCore
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("ESPN core decode: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	out := make([]Play, 0, len(r.Items))
	for _, p := range r.Items {
		out = append(out, p.play())
	}
	slices.SortFunc(out, func(a, b Play) int { return a.Sequence - b.Sequence })
	return out, nil
}
