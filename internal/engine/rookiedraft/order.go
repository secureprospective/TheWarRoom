package rookiedraft

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// FinalStandings is the standings branch's seam. FirstRound must return every franchise, worst
// first, applying the supplied tie rule. It must refuse missing final results or unresolved ties.
// The draft validates membership; it never estimates standings or implements a hidden tie breaker.
type FinalStandings interface {
	FirstRound(tie leaguerules.Value[leaguerules.TieRule]) ([]string, error)
}

type MissingFinalStandings struct{}

type missingStandingsError struct{}

func (missingStandingsError) Error() string {
	return "The draft order needs the final standings, which are not built yet."
}

func (MissingFinalStandings) FirstRound(_ leaguerules.Value[leaguerules.TieRule]) ([]string, error) {
	return nil, missingStandingsError{}
}

// OwnedPick retains the original franchise as its identity while following its current owner.
// Ownership arrives from the native asset ledger, never from the standings order.
type OwnedPick struct {
	Round    int    `json:"round"`
	Original string `json:"original"`
	Owner    string `json:"owner"`
}

type Pick struct {
	OwnedPick
	Slot int `json:"slot"`
}

type RoundOrder struct {
	Round int    `json:"round"`
	Words string `json:"words"`
	Picks []Pick `json:"picks"`
}

// Order can expose round one even when a later round still needs a ruling. Unresolved rounds have
// no fabricated slots. An explicit round override does not need an automatic pattern or tie rule.
func Order(r leaguerules.DraftRules, franchises []string, owned []OwnedPick,
	standings FinalStandings) ([]RoundOrder, error) {
	if r.Rounds.State != leaguerules.IsKnown || r.Rounds.Value < 1 {
		return nil, fmt.Errorf("needs a ruling: draft rounds")
	}
	if err := validateFranchises(franchises); err != nil {
		return nil, err
	}
	owners, err := indexOwners(franchises, owned, r.Rounds.Value)
	if err != nil {
		return nil, err
	}
	first, firstWords := firstRound(r, franchises, standings)
	out := make([]RoundOrder, 0, r.Rounds.Value)
	for round := 1; round <= r.Rounds.Value; round++ {
		teams, words := roundTeams(r, round, first, firstWords)
		row := RoundOrder{Round: round, Words: words, Picks: []Pick{}}
		if words == "" {
			if err := validateOrder(teams, franchises); err != nil {
				return nil, fmt.Errorf("round %d: %w", round, err)
			}
			row.Picks, err = placeOwners(round, teams, owners)
			if err != nil {
				return nil, err
			}
			row.Words = "Draft order set"
		}
		out = append(out, row)
	}
	return out, nil
}

func firstRound(r leaguerules.DraftRules, franchises []string, standings FinalStandings) ([]string, string) {
	if r.Order.State == leaguerules.IsKnown && len(r.Order.Value[1]) > 0 {
		return slices.Clone(r.Order.Value[1]), ""
	}
	if r.OrderSource.State != leaguerules.IsKnown || r.OrderSource.Value != leaguerules.WorstStandingsFirst {
		return nil, "Needs a ruling: first round draft order"
	}
	if standings == nil {
		standings = MissingFinalStandings{}
	}
	teams, err := standings.FirstRound(r.TieRule)
	if err != nil {
		return nil, err.Error()
	}
	if err := validateOrder(teams, franchises); err != nil {
		return nil, err.Error()
	}
	return slices.Clone(teams), ""
}

func roundTeams(r leaguerules.DraftRules, round int, first []string, firstWords string) ([]string, string) {
	if r.Order.State == leaguerules.IsKnown && len(r.Order.Value[round]) > 0 {
		return slices.Clone(r.Order.Value[round]), ""
	}
	if round == 1 {
		return slices.Clone(first), firstWords
	}
	if r.LaterPattern.State != leaguerules.IsKnown {
		return nil, "Needs a ruling: later round draft pattern"
	}
	if firstWords != "" {
		return nil, firstWords
	}
	teams := slices.Clone(first)
	switch r.LaterPattern.Value {
	case leaguerules.Linear:
	case leaguerules.Snake:
		if round%2 == 0 {
			slices.Reverse(teams)
		}
	default:
		return nil, "Needs a ruling: this later round draft pattern is not built yet"
	}
	return teams, ""
}

func validateFranchises(franchises []string) error {
	if len(franchises) < 2 {
		return fmt.Errorf("record at least two franchises before ordering the draft")
	}
	seen := map[string]bool{}
	for _, franchise := range franchises {
		if franchise == "" || seen[franchise] {
			return fmt.Errorf("each draft franchise must have one unique identity")
		}
		seen[franchise] = true
	}
	return nil
}

func validateOrder(teams, franchises []string) error {
	if len(teams) != len(franchises) {
		return fmt.Errorf("list every franchise exactly once in the draft order")
	}
	seen := map[string]bool{}
	for _, team := range teams {
		if !slices.Contains(franchises, team) || seen[team] {
			return fmt.Errorf("list every franchise exactly once in the draft order")
		}
		seen[team] = true
	}
	return nil
}

func indexOwners(franchises []string, owned []OwnedPick, rounds int) (map[OwnedPick]string, error) {
	out := map[OwnedPick]string{}
	for _, pick := range owned {
		if pick.Round < 1 || pick.Round > rounds || !slices.Contains(franchises, pick.Original) ||
			!slices.Contains(franchises, pick.Owner) {
			return nil, fmt.Errorf("a draft pick has an unknown round, original franchise or owner")
		}
		key := OwnedPick{Round: pick.Round, Original: pick.Original}
		if out[key] != "" {
			return nil, fmt.Errorf("a draft pick has more than one recorded owner")
		}
		out[key] = pick.Owner
	}
	return out, nil
}

func placeOwners(round int, teams []string, owners map[OwnedPick]string) ([]Pick, error) {
	out := make([]Pick, 0, len(teams))
	for slot, original := range teams {
		owner := owners[OwnedPick{Round: round, Original: original}]
		if owner == "" {
			return nil, fmt.Errorf("record the owner of round %d's pick belonging to %s", round, original)
		}
		out = append(out, Pick{OwnedPick: OwnedPick{Round: round, Original: original, Owner: owner}, Slot: slot + 1})
	}
	return out, nil
}
