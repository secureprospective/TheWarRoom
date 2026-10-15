package rookiedraft

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// Selection is an accepted owner selection, supplied in ledger order. The writer authorizes the owner
// through the native gate and judges roster and cap legality before recording it. Replay checks draft
// identity, ownership, uniqueness and clock placement independently. Terms is the contract fixed when the
// pick was made, so a later price change cannot reprice it; absent, the slot's current terms apply.
type Selection struct {
	Round    int
	Original string
	Owner    string
	Player   string
	At       time.Time
	Terms    *Terms
}

type Result struct {
	Pick
	Player    string     `json:"player,omitempty"`
	Skipped   bool       `json:"skipped"`
	Forfeited bool       `json:"forfeited"`
	MadeAt    *time.Time `json:"madeAt,omitempty"`
	Contract  *Terms     `json:"contract,omitempty"`
}

// Clock is the pick on the clock, with the contract it would sign at its slot's current terms.
type Clock struct {
	Pick
	OpenedAt time.Time `json:"openedAt"`
	Deadline time.Time `json:"deadline"`
	Contract Terms     `json:"contract"`
}

type Draft struct {
	Start     time.Time `json:"start"`
	Started   bool      `json:"started"`
	Exhausted bool      `json:"exhausted"`
	Picks     []Result  `json:"picks"`
	OnClock   *Clock    `json:"onClock"`
}

// Replay uses the supplied instant, never wall time. Expiry is inclusive: at the deadline the pick
// has already skipped. Successive overdue clocks start at the prior deadline, not at the read time.
// Exhausted does not seal the draft or issue new picks; closing is a separate commissioner phase.
func Replay(r leaguerules.DraftRules, order []RoundOrder, start, at time.Time,
	selections []Selection) (Draft, error) {
	setup, err := ResolveSetup(r)
	if err != nil {
		return Draft{}, err
	}
	picks, err := preparedPicks(r, setup, order)
	if err != nil {
		return Draft{}, err
	}
	out := Draft{Start: start, Started: !start.IsZero() && !at.Before(start), Picks: picks}
	if !out.Started {
		if len(selections) > 0 {
			return Draft{}, fmt.Errorf("a selection cannot precede the draft start")
		}
		for i := range out.Picks {
			out.Picks[i].Forfeited = false
		}
		return out, nil
	}
	cursor, opened := 0, start
	for _, selection := range selections {
		if selection.At.Before(opened) || selection.At.After(at) {
			return Draft{}, fmt.Errorf("draft selections must follow their clock in time order")
		}
		cursor, _ = advance(out.Picks, cursor, opened, selection.At, setup.Clock)
		if cursor == len(out.Picks) {
			return Draft{}, fmt.Errorf("there is no pick on the clock for this selection")
		}
		if err := applySelection(r, &out.Picks[cursor], selection, out.Picks); err != nil {
			return Draft{}, err
		}
		cursor++
		opened = selection.At
	}
	cursor, opened = advance(out.Picks, cursor, opened, at, setup.Clock)
	out.Exhausted = cursor == len(out.Picks)
	if out.Exhausted {
		return out, nil
	}
	terms, err := SlotTerms(r, out.Picks[cursor].Round, out.Picks[cursor].Slot)
	if err != nil {
		return Draft{}, err
	}
	out.OnClock = &Clock{Pick: out.Picks[cursor].Pick, OpenedAt: opened, Deadline: opened.Add(setup.Clock),
		Contract: terms}
	return out, nil
}

func preparedPicks(r leaguerules.DraftRules, setup Setup, order []RoundOrder) ([]Result, error) {
	if len(order) != setup.Rounds {
		return nil, fmt.Errorf("set every round's draft order before starting")
	}
	out := []Result{}
	counts := map[string]int{}
	for i, round := range order {
		if round.Round != i+1 || len(round.Picks) == 0 {
			return nil, fmt.Errorf("set round %d's draft order before starting", i+1)
		}
		if err := validateRound(round, order[0]); err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for slot, pick := range round.Picks {
			if pick.Round != round.Round || pick.Slot != slot+1 || pick.Owner == "" ||
				pick.Original == "" || seen[pick.Original] {
				return nil, fmt.Errorf("a draft round has an invalid or repeated pick")
			}
			if _, err := SlotTerms(r, pick.Round, pick.Slot); err != nil {
				return nil, err
			}
			seen[pick.Original] = true
			counts[pick.Owner]++
			forfeit := counts[pick.Owner] > setup.MaximumPicks
			if forfeit && !setup.ForfeitExtra {
				return nil, fmt.Errorf("a team has more than %d picks; set how extra picks are handled", setup.MaximumPicks)
			}
			out = append(out, Result{Pick: pick, Forfeited: forfeit})
		}
	}
	return out, nil
}

func validateRound(round, first RoundOrder) error {
	originals := make([]string, 0, len(first.Picks))
	teams := make([]string, 0, len(round.Picks))
	for _, pick := range first.Picks {
		originals = append(originals, pick.Original)
	}
	for _, pick := range round.Picks {
		teams = append(teams, pick.Original)
	}
	return validateOrder(teams, originals)
}

func advance(picks []Result, cursor int, opened, at time.Time, duration time.Duration) (int, time.Time) {
	for cursor < len(picks) {
		if picks[cursor].Forfeited {
			cursor++
			continue
		}
		deadline := opened.Add(duration)
		if at.Before(deadline) {
			break
		}
		picks[cursor].Skipped = true
		picks[cursor].MadeAt = &deadline
		cursor++
		opened = deadline
	}
	return cursor, opened
}

func applySelection(r leaguerules.DraftRules, result *Result, selection Selection, picks []Result) error {
	if selection.Round != result.Round || selection.Original != result.Original || selection.Owner != result.Owner {
		return fmt.Errorf("only the current owner may select for the pick on the clock")
	}
	id, err := playerid.New(selection.Player)
	if err != nil || id.String() != selection.Player {
		return fmt.Errorf("choose a valid rookie player")
	}
	if slices.ContainsFunc(picks, func(p Result) bool { return p.Player == selection.Player }) {
		return fmt.Errorf("this player has already been drafted")
	}
	terms := selection.Terms
	if terms == nil {
		slot, err := SlotTerms(r, result.Round, result.Slot)
		if err != nil {
			return err
		}
		terms = &slot
	}
	result.Player, result.MadeAt, result.Contract = selection.Player, &selection.At, terms
	return nil
}

// NextPick follows ownership, including traded picks, and excludes picks already used or forfeited.
func (d Draft) NextPick(franchise string) (Pick, bool) {
	for _, result := range d.Picks {
		if result.Owner == franchise && result.Player == "" && !result.Skipped && !result.Forfeited {
			return result.Pick, true
		}
	}
	return Pick{}, false
}
