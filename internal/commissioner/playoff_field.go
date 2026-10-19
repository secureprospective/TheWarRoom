package commissioner

import (
	"fmt"
	"maps"
	"slices"
)

// Why a team sits where it does in a conference's field.
const (
	kindDivisionWinner = "Division winner"
	kindWildCard       = "Wild card"
	kindByHand         = "Set by the commissioner"
)

// FieldSeed is one team in a conference's playoff field.
type FieldSeed struct {
	Seed      int    `json:"seed"`
	Franchise string `json:"franchise"`
	Name      string `json:"name"`
	Division  string `json:"division"`
	Rank      int    `json:"rank"`
	Kind      string `json:"kind"`
	Bye       bool   `json:"bye"`
	ByHand    bool   `json:"byHand"`
}

// ConferenceField is a conference's seeded playoff teams, best seed first.
type ConferenceField struct {
	Conference string      `json:"conference"`
	Seeds      []FieldSeed `json:"seeds"`
}

type fieldTeam struct{ Name, Division, Conference string }

// fieldInput is the final standings order with each team's place in the league, and the seeds the
// commissioner set by hand for each conference.
type fieldInput struct {
	order       []string
	teams       map[string]fieldTeam
	conferences []string
	rules       PlayoffRules
	hand        map[string]map[int]string
}

// seedField builds every conference's field from the final standings: division winners take the first
// seeds in standings order, the best of the rest take the wild cards, and a hand seed replaces one slot.
// It returns plain words instead of a field when the standings cannot fill the rules' field.
func seedField(in fieldInput) ([]ConferenceField, string) {
	rank := map[string]int{}
	for i, id := range in.order {
		rank[id] = i + 1
	}
	out := make([]ConferenceField, 0, len(in.conferences))
	for _, conference := range in.conferences {
		members := []string{}
		for _, id := range in.order {
			if in.teams[id].Conference == conference {
				members = append(members, id)
			}
		}
		ordered, words := in.policyOrder(conference, members)
		if words != "" {
			return nil, words
		}
		seeds, words := in.placeHand(conference, ordered, members)
		if words != "" {
			return nil, words
		}
		field := ConferenceField{Conference: conference}
		for i, id := range seeds {
			seed := FieldSeed{Seed: i + 1, Franchise: id, Name: in.teams[id].Name, Division: in.teams[id].Division,
				Rank: rank[id], Kind: in.kindOf(id, ordered), Bye: i < in.rules.ByeSeeds}
			seed.ByHand = in.hand[conference][i+1] == id
			if seed.ByHand {
				seed.Kind = kindByHand
			}
			field.Seeds = append(field.Seeds, seed)
		}
		out = append(out, field)
	}
	return out, ""
}

// policyOrder lists a conference's teams in seeding order: its division winners, best first, then everyone else.
func (in fieldInput) policyOrder(conference string, members []string) ([]string, string) {
	winners, seenDivision := []string{}, map[string]bool{}
	for _, id := range members {
		if division := in.teams[id].Division; !seenDivision[division] {
			seenDivision[division] = true
			winners = append(winners, id)
		}
	}
	if len(members) < in.rules.PerConference || len(winners) < in.rules.DivisionWinners {
		return nil, fmt.Sprintf("The %s conference has %d teams in %d divisions, too few for %d playoff teams with %d division winners",
			conference, len(members), len(winners), in.rules.PerConference, in.rules.DivisionWinners)
	}
	winners = winners[:in.rules.DivisionWinners]
	ordered := slices.Clone(winners)
	for _, id := range members {
		if !slices.Contains(winners, id) {
			ordered = append(ordered, id)
		}
	}
	return ordered, ""
}

// placeHand puts each hand-set team in its slot and fills the other slots, in order, from the rest.
func (in fieldInput) placeHand(conference string, ordered, members []string) ([]string, string) {
	hand := in.hand[conference]
	seeds := make([]string, in.rules.PerConference)
	taken := map[string]bool{}
	for _, slot := range slices.Sorted(maps.Keys(hand)) {
		id := hand[slot]
		if slot < 1 || slot > len(seeds) || !slices.Contains(members, id) || taken[id] {
			return nil, fmt.Sprintf("A seed set by hand in the %s conference no longer fits: %s is not a team there, or its slot is gone",
				conference, id)
		}
		seeds[slot-1], taken[id] = id, true
	}
	rest := []string{}
	for _, id := range ordered {
		if !taken[id] {
			rest = append(rest, id)
		}
	}
	for i := range seeds {
		if seeds[i] == "" {
			seeds[i], rest = rest[0], rest[1:]
		}
	}
	return seeds, ""
}

func (in fieldInput) kindOf(id string, ordered []string) string {
	if slices.Index(ordered, id) < in.rules.DivisionWinners {
		return kindDivisionWinner
	}
	return kindWildCard
}
