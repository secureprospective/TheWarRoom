package commissioner

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ruleOwner maps a rule field (exact path, or a section ending in ".") to the census row that owns it.
type ruleOwner struct{ prefix, control, gravity string }

func ruleOwners() []ruleOwner {
	return []ruleOwner{
		{"league.teams", "CM-001", "G2"}, {"league.conferences", "CM-001", "G2"},
		{"league.divisions", "CM-001", "G2"}, {"league.format", "CM-001", "G2"},
		{"league.initialRosters", "CM-001", "G2"}, {"league.alignment", "CM-002", "G2"},
		{"league.divisionConferences", "CM-002", "G2"},
		{"league.dynasty", "CM-005", "G2"}, {"league.carry", "CM-005", "G2"},
		{"league.keepers", "CM-005", "G2"}, {"roster.", "CM-003", "G2"},
		{"lineup.", "CM-004", "G2"}, {"scoring.", "CM-006", "G2"},
		{"standings.", "CM-007", "G2"}, {"cap.", "CM-008", "G2"},
		{"contracts.", "CM-009", "G2"}, {"ir.", "CM-010", "G2"},
		{"taxi.", "CM-011", "G2"}, {"waivers.", "CM-012", "G2"},
		{"bids.", "CM-013", "G2"}, {"trades.", "CM-014", "G2"},
		{"schedule.", "CM-015", "G2"}, {"playoffs.", "CM-016", "G2"},
		{"house.", "CM-017", "G1"}, {"calendar.windows", "CM-023", "G2"},
		{"calendar.timezone", "CM-024", "G1"}, {"calendar.boundary", "CM-024", "G2"},
		{"calendar.rollover", "CM-027", "G3"}, {"calendar.rolloverDate", "CM-027", "G3"},
		{"calendar.tradeDeadline", "CM-025", "G2"}, {"calendar.tradeDeadlineWeek", "CM-025", "G2"},
		{"draft.mode", "CM-040", "G2"}, {"draft.clock", "CM-040", "G2"},
		{"draft.onExpire", "CM-040", "G2"}, {"draft.timeouts", "CM-040", "G2"},
		{"draft.rounds", "CM-040", "G2"}, {"draft.maximumPicks", "CM-040", "G2"},
		{"draft.forfeitExtra", "CM-040", "G2"}, {"draft.pricing", "CM-040", "G2"},
		{"notify.", "CM-051", "G1"}, {"draft.lengths", "CM-040", "G2"}, {"draft.rfaAfter", "CM-040", "G2"},
		{"draft.orderSource", "CM-041", "G2"}, {"draft.order", "CM-041", "G2"},
		{"draft.laterPattern", "CM-041", "G2"}, {"draft.tieRule", "CM-041", "G2"},
	}
}

func (o ruleOwner) owns(path leaguerules.FieldPath) bool {
	if strings.HasSuffix(o.prefix, ".") {
		return strings.HasPrefix(string(path), o.prefix)
	}
	return string(path) == o.prefix
}

// ownerFor refuses a field that no row, or more than one row, owns.
func ownerFor(path leaguerules.FieldPath) (ruleOwner, error) {
	var found []ruleOwner
	for _, owner := range ruleOwners() {
		if owner.owns(path) {
			found = append(found, owner)
		}
	}
	if len(found) != 1 {
		return ruleOwner{}, fmt.Errorf("choose a recognised league rule")
	}
	return found[0], nil
}

func catalogueField(path string) (leaguerules.CatalogueField, error) {
	if field, ok := leaguerules.Field(leaguerules.FieldPath(path)); ok {
		return field, nil
	}
	return leaguerules.CatalogueField{}, fmt.Errorf("choose a recognised league rule")
}
