package commissioner

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestMustCutAllowsSuccessiveHumanCutsWithoutOpeningAcquisitions(t *testing.T) {
	rules := leaguerules.Default32()
	rules.Roster.Limits.Value.Max = leaguerules.Known(1)
	before := &RawNativeSnapshot{Players: []NativePlayer{{ID: "1000", Franchise: "a", Status: domain.RosterActive}, {ID: "1001", Franchise: "a", Status: domain.RosterActive}, {ID: "1002", Franchise: "a", Status: domain.RosterActive}}}
	after := *before
	after.Players = after.Players[:2]
	records := []Record{{Principal: ReserveClockPrincipal, Command: RawCommand{Command: RosterReserve, Args: RawArgs{Waiver: &RawWaiverArgs{Franchise: "a", Player: "1002", Bucket: "active"}}}, Receipt: Receipt{Entry: "auto"}}}
	permitted := AllowDebtReduction(before, &after, records, rules, "a")
	if permitted.Roster.Limits.Value.Max.Value != 2 || rules.Roster.Limits.Value.Max.Value != 1 {
		t.Fatal("successive cut blocked or rules mutated")
	}
	unchanged := AllowDebtReduction(before, before, records, rules, "a")
	if unchanged.Roster.Limits.Value.Max.Value != 1 {
		t.Fatal("non-reducing move got debt exemption")
	}
	if AllowDebtReduction(before, &after, nil, rules, "a").Roster.Limits.Value.Max.Value != 1 {
		t.Fatal("ordinary overfull roster got automatic-return exemption")
	}
}
