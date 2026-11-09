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

// Fill gives every setting r does not hold the value base holds for it. A league imported from another
// platform runs as that league ran: what the platform records comes from r, and the rest of how the
// league operates comes from base, a bundle of setting values the commissioner can change afterwards
// like any other.
func Fill(r, base RuleSet) RuleSet {
	out := cloneValue(r)
	for _, f := range ruleFields() {
		if DescribeSetting(f.read(out)).State == "Set" {
			continue
		}
		if b := f.read(base); DescribeSetting(b).State == "Set" {
			_ = f.apply(&out, b) // b is base's own value for this field, so it is always the right kind.
		}
	}
	return out
}
