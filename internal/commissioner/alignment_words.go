package commissioner

import (
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
	state, err := GovernanceState(records)
	if err != nil {
		return "", fmt.Errorf("read alignment names: %w", err)
	}
	seats := readSeats(state, LeagueFranchises(records))
	applyNativeNames(seats, records)
	names := make(map[string]string, len(seats))
	for _, seat := range seats {
		if seat.Name != "" {
			names[seat.Franchise] = seat.Name
		}
	}
	ids := make([]string, 0, len(alignment.Value))
	for id := range alignment.Value {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		name := names[id]
		if name == "" {
			name = id
		}
		parts = append(parts, name+": "+alignment.Value[id])
	}
	if len(parts) == 0 {
		return "None", nil
	}
	return strings.Join(parts, "; "), nil
}
