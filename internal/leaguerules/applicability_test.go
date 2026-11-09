package leaguerules

import "testing"

func TestUnusedFollowsKnownControllingSettings(t *testing.T) {
	r := LegacyNFL()
	for _, path := range []FieldPath{"league.keepers", "cap.complianceDeadline"} {
		if !Unused(r, path) {
			t.Errorf("%s should not need a ruling", path)
		}
	}
	r.League.Dynasty = Known(false)
	r.Cap.Enforcement = Known(SoftCap)
	r.Trades.Review = Known(NoReview)
	if Unused(r, "league.keepers") || Unused(r, "cap.complianceDeadline") || !Unused(r, "trades.dotSeats") {
		t.Fatal("applicability did not follow settings")
	}
	r.League.Dynasty.State = NeedsRuling
	r.Cap.Enforcement.State = NeedsRuling
	r.Trades.Review.State = NeedsRuling
	for _, path := range []FieldPath{"league.keepers", "cap.complianceDeadline", "trades.dotSeats", "cap.amount"} {
		if Unused(r, path) {
			t.Errorf("%s suppressed without controlling evidence", path)
		}
	}
}
