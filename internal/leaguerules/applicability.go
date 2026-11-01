package leaguerules

// Unused says a setting does nothing in this league as it stands, so it waits for no ruling: keepers in a
// dynasty league (every player stays), a soft cap deadline under a hard cap, review seats with no review panel.
func Unused(r RuleSet, path FieldPath) bool {
	switch path {
	case "league.keepers":
		return r.League.Dynasty.State == IsKnown && r.League.Dynasty.Value
	case "cap.complianceDeadline":
		return r.Cap.Enforcement.State == IsKnown && r.Cap.Enforcement.Value == HardCap
	case "trades.dotSeats":
		return r.Trades.Review.State == IsKnown && r.Trades.Review.Value != DOTReview
	}
	return false
}
