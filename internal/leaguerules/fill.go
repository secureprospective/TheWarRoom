package leaguerules

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
