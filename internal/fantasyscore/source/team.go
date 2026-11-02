package source

// Canonical returns a team code in ESPN's form, whichever source supplied it. The week strip is
// seeded from MFL's schedule and the live board comes from ESPN, so one game must be the same
// game under either source's codes.
func CanonicalTeam(team string) string {
	switch team {
	case "GBP":
		return "GB"
	case "KCC":
		return "KC"
	case "LVR":
		return "LV"
	case "NEP":
		return "NE"
	case "NOS":
		return "NO"
	case "SFO":
		return "SF"
	case "TBB":
		return "TB"
	case "JAC":
		return "JAX"
	case "WAS":
		return "WSH"
	}
	return team
}
