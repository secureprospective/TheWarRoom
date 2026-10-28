package rookiedraft

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
	// Extra counts the picks a person inserted after this pick's place in the order; 0 is a scheduled pick.
	Extra     int        `json:"extra,omitempty"`
	Player    string     `json:"player,omitempty"`
	Skipped   bool       `json:"skipped"`
	Forfeited bool       `json:"forfeited"`
	MadeAt    *time.Time `json:"madeAt,omitempty"`
	Contract  *Terms     `json:"contract,omitempty"`
	// How says how the pick was used: by its owner, by the commissioner, inserted, run out, skipped, or
	// chosen by the commissioner after the clock skipped it. Changed says the rookie was later swapped.
	How     string `json:"how,omitempty"`
	Changed bool   `json:"changed,omitempty"`
}

// Clock is the pick on the clock, with the contract it would sign at its slot's current terms. Remaining
// is the time left at the instant read, and while Paused it is the time the clock holds.
type Clock struct {
	Pick
	OpenedAt  time.Time     `json:"openedAt"`
	Deadline  time.Time     `json:"deadline"`
	Contract  Terms         `json:"contract"`
	Paused    bool          `json:"paused"`
	Remaining time.Duration `json:"remaining"`
}

type Draft struct {
	Start     time.Time `json:"start"`
	Started   bool      `json:"started"`
	Exhausted bool      `json:"exhausted"`
	Picks     []Result  `json:"picks"`
	OnClock   *Clock    `json:"onClock"`
}

// Replay applies owner selections only; ReplayActs applies every kind of act.
func Replay(r leaguerules.DraftRules, order []RoundOrder, start, at time.Time,
	selections []Selection) (Draft, error) {
	acts := make([]Act, len(selections))
	for i, s := range selections {
		acts[i] = Act{Kind: Select, At: s.At, Pick: PickRef{Round: s.Round, Original: s.Original}, Owner: s.Owner,
			Player: s.Player, Terms: s.Terms}
	}
	return ReplayActs(r, order, start, at, acts)
}

// ReplayActs uses the supplied instant, never wall time. Expiry is inclusive: at the deadline the pick
// has already skipped. Successive overdue clocks start at the prior deadline, not at the read time. A
// Restart is the one act that does not first let the clock run out: it puts the draft back as it stood.
// Exhausted does not seal the draft or issue new picks; closing is a separate commissioner phase.
func ReplayActs(r leaguerules.DraftRules, order []RoundOrder, start, at time.Time, acts []Act) (Draft, error) {
	return replayActs(r, order, start, at, acts, false)
}

// ReplaySideActs replays one side's share of a validated league order. Traded picks follow their owners,
// so rounds can have different original franchises or no picks at all on this side.
func ReplaySideActs(r leaguerules.DraftRules, order []RoundOrder, start, at time.Time,
	acts []Act) (Draft, error) {
	return replayActs(r, order, start, at, acts, true)
}

func replayActs(r leaguerules.DraftRules, order []RoundOrder, start, at time.Time,
	acts []Act, perSide bool) (Draft, error) {
	setup, err := ResolveSetup(r)
	if err != nil {
		return Draft{}, err
	}
	picks, err := preparedPicks(r, setup, order, perSide)
	if err != nil {
		return Draft{}, err
	}
	out := Draft{Start: start, Started: !start.IsZero() && !at.Before(start), Picks: picks}
	if !out.Started {
		if len(acts) > 0 {
			return Draft{}, fmt.Errorf("a selection cannot precede the draft start")
		}
		for i := range out.Picks {
			out.Picks[i].Forfeited = false
		}
		return out, nil
	}
	line := newTimeline(out.Picks, start, setup.Clock)
	for _, act := range acts {
		if act.At.After(at) {
			return Draft{}, fmt.Errorf("draft selections must follow their clock in time order")
		}
		if err := line.apply(r, act); err != nil {
			return Draft{}, err
		}
	}
	line.advance(at)
	out.Picks, out.Exhausted = line.picks, !line.onClock()
	if out.OnClock, err = line.clockAt(r, at); err != nil {
		return Draft{}, err
	}
	return out, nil
}

func preparedPicks(r leaguerules.DraftRules, setup Setup, order []RoundOrder, perSide bool) ([]Result, error) {
	if len(order) != setup.Rounds {
		return nil, fmt.Errorf("set every round's draft order before starting")
	}
	out := []Result{}
	counts := map[string]int{}
	for i, round := range order {
		if err := validateReplayRound(round, order[0], i+1, perSide); err != nil {
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

func validateReplayRound(round, first RoundOrder, number int, perSide bool) error {
	if round.Round != number || (!perSide && len(round.Picks) == 0) {
		return fmt.Errorf("set round %d's draft order before starting", number)
	}
	if perSide {
		return nil
	}
	return validateRound(round, first)
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

// NextPick follows ownership, including traded picks, and excludes picks already used or forfeited.
func (d Draft) NextPick(franchise string) (Pick, bool) {
	for _, result := range d.Picks {
		if result.Owner == franchise && result.Player == "" && !result.Skipped && !result.Forfeited {
			return result.Pick, true
		}
	}
	return Pick{}, false
}
