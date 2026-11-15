package commissioner

import (
	"strings"
	"testing"
)

func TestNotificationSettingsAreRulesThatNeedOnlyAConfirm(t *testing.T) {
	t.Parallel()
	app := opsApp(t)
	rules, err := app.TargetCommissionerRules(opsLeague)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if strings.HasPrefix(row.Field, "notify.") {
				found = append(found, row.Field+"="+row.Value)
				if row.Competitive || row.Safeguards.Now != "confirm" || row.Safeguards.NextRollover != "confirm" {
					t.Errorf("%s should need one confirm: %+v", row.Field, row.Safeguards)
				}
			}
		}
	}
	if strings.Join(found, ",") != "notify.transactions=On,notify.reminders=On" {
		t.Fatalf("the league starts with both on: %v", found)
	}
}
