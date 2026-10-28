package commissioner

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// alignmentWords uses the same current identities as the league directory.
func alignmentWords(records []Record, alignment leaguerules.Value[map[string]string]) (string, error) {
	if alignment.State != leaguerules.IsKnown {
		return leaguerules.DescribeField(alignmentField, alignment).Words, nil
	}
	names, err := franchiseNames(records)
	if err != nil {
		return "", err
	}
	ids := make([]string, 0, len(alignment.Value))
	for id := range alignment.Value {
		ids = append(ids, id)
	}
	// Franchises read in seat order, so franchise.100 follows franchise.99.
	slices.SortFunc(ids, func(a, b string) int {
		return cmp.Or(cmp.Compare(franchiseNumber(a), franchiseNumber(b)), cmp.Compare(a, b))
	})
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, cmpName(names, id)+": "+alignment.Value[id])
	}
	if len(parts) == 0 {
		return "None", nil
	}
	return strings.Join(parts, "; "), nil
}

// franchiseNames is each franchise's current name, as the league directory shows it.
func franchiseNames(records []Record) (map[string]string, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return nil, fmt.Errorf("read alignment names: %w", err)
	}
	seats := readSeats(state, LeagueFranchises(records))
	applyNativeNames(seats, records)
	names := make(map[string]string, len(seats))
	for _, seat := range seats {
		if seat.Name != "" {
			names[seat.Franchise] = seat.Name
		}
	}
	return names, nil
}
