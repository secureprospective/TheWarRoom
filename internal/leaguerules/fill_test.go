package leaguerules

import (
	"reflect"
	"testing"
)

// An imported league runs as it ran: what the platform holds stays, and every setting it does not hold
// comes from the bundle the league runs by.
func TestFillKeepsHeldAndTakesTheRestFromBase(t *testing.T) {
	t.Parallel()
	imported := RuleSet{}
	imported.League.Teams = cited(12, "Your MFL league")
	base := LegacyNFL()
	got := Fill(imported, base)
	if got.League.Teams.Value != 12 {
		t.Fatalf("a held setting was overwritten: %+v", got.League.Teams)
	}
	if !reflect.DeepEqual(got.Contracts.MaximumYears, base.Contracts.MaximumYears) ||
		!reflect.DeepEqual(got.Trades.MaximumParties, base.Trades.MaximumParties) {
		t.Errorf("unheld settings were not taken from the base: %+v %+v", got.Contracts.MaximumYears, got.Trades.MaximumParties)
	}
	for _, f := range ruleFields() {
		if DescribeSetting(f.read(base)).State == "Set" && DescribeSetting(f.read(got)).State != "Set" {
			t.Errorf("%s is set in the base but still waits for a ruling", f.pathName())
		}
	}
	if imported.Contracts.MaximumYears.State == IsKnown {
		t.Error("Fill changed its input")
	}
}
