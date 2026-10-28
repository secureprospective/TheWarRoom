package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// sideDraftKeys expands a shared start into each side's draft. Live acts already name their side.
func sideDraftKeys(records []Record, key DraftKey) []DraftKey {
	if key.Side != "" {
		return []DraftKey{key}
	}
	keys := startupKeys(records)
	for i := range keys {
		keys[i].Season, keys[i].Startup = key.Season, key.Startup
	}
	return keys
}

func checkDraftSide(records []Record, key DraftKey) error {
	sides, err := snapshotSides(records)
	if err != nil {
		return err
	}
	if len(sides) == 0 && key.Side == "" {
		return nil
	}
	if key.Side == "" || !slices.Contains(sideNames(sides), key.Side) {
		return fmt.Errorf("choose the side whose draft you are changing")
	}
	return nil
}

// rookieSideOrder keeps the league's order and pick identities, moving traded picks with their current
// owner. Ordering the whole league first validates originals and owners even when a trade crosses sides.
func rookieSideOrder(records []Record, key DraftKey,
	order []rookiedraft.RoundOrder) ([]rookiedraft.RoundOrder, error) {
	if key.Side == "" {
		return order, nil
	}
	if err := checkDraftSide(records, key); err != nil {
		return nil, err
	}
	sides, err := snapshotSides(records)
	if err != nil {
		return nil, err
	}
	for i := range order {
		picks := []rookiedraft.Pick{}
		for _, pick := range order[i].Picks {
			if sides[pick.Owner] == key.Side {
				pick.Slot = len(picks) + 1
				picks = append(picks, pick)
			}
		}
		order[i].Picks = picks
	}
	return order, nil
}

func rookieClosed(records []Record, season int) bool {
	for _, key := range sideDraftKeys(records, rookieDraft(season)) {
		if _, closed := closeRecord(records, key); !closed {
			return false
		}
	}
	return true
}

func rookieOrderWords(order []rookiedraft.RoundOrder) string {
	for _, round := range order {
		if len(round.Picks) == 0 && round.Words != "Draft order set" {
			return round.Words
		}
	}
	return ""
}

func rookieOrder(records []Record, key DraftKey, rules leaguerules.DraftRules,
	standings rookiedraft.FinalStandings) ([]rookiedraft.RoundOrder, error) {
	owned, err := draftOwnership(key.Season, records)
	if err != nil {
		return nil, err
	}
	owned = slices.DeleteFunc(owned, func(p rookiedraft.OwnedPick) bool {
		return rules.Rounds.State == leaguerules.IsKnown && p.Round > rules.Rounds.Value
	})
	order, err := rookiedraft.Order(rules, LeagueFranchises(records), owned, standings)
	if err != nil {
		//nolint:wrapcheck // Keep the engine's ruling words unchanged in the draft reading.
		return nil, err
	}
	return rookieSideOrder(records, key, order)
}
