package governance

type Control struct {
	ID          string
	MinimumRole Role
	COnly       bool
}

// CensusControls returns independent permission data; succession requires its specific action.
func CensusControls() []Control {
	return []Control{
		{ID: "CM-001", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-002", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-003", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-004", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-005", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-006", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-007", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-008", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-009", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-010", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-011", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-012", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-013", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-014", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-015", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-016", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-017", MinimumRole: Commissioner, COnly: false},
		{ID: "CM-018", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-019", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-020", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-021", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-022", MinimumRole: "", COnly: false},
		{ID: "CM-023", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-024", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-025", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-026", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-027", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-028", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-029", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-030", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-031", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-032", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-033", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-034", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-035", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-036", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-037", MinimumRole: DOT, COnly: false},
		{ID: "CM-038", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-039", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-040", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-041", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-042", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-043", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-044", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-045", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-046", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-047", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-048", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-049", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-050", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-051", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-052", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-053", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-054", MinimumRole: CoCommissioner, COnly: false},
		{ID: "CM-055", MinimumRole: Commissioner, COnly: true},
		{ID: "CM-056", MinimumRole: Owner, COnly: false},
	}
}
func CensusControl(id string) (Control, bool) {
	for _, row := range CensusControls() {
		if row.ID == id {
			return row, true
		}
	}
	return Control{}, false
}
