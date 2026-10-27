package commissioner

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func default32League(t *testing.T) []Record {
	t.Helper()
	raw, err := (RawCommand{League: "native.test", Command: Create,
		Args:   RawArgs{Name: "Ours", Preset: PresetDefault32, Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	if err := raw.Args.validateCreation(); err != nil {
		t.Fatal(err)
	}
	return []Record{{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: Receipt{Entry: "create", Revision: 1, RecordedAt: noChangeTime().Add(-time.Hour)}}}
}

// A new 32-Team Default league holds every spec value as a commissioner setting, read back through
// the seed upgrade unchanged, and everything else is Legacy's.
func TestDefault32PresetSeedsTheSpec(t *testing.T) {
	records := default32League(t)
	if got := len(records[0].Command.Args.Franchises); got != 32 {
		t.Fatalf("franchises: %d, want 32", got)
	}
	got, err := RulesInForce(records, 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	want := leaguerules.Default32()
	if !reflect.DeepEqual(got.Scoring, want.Scoring) || !reflect.DeepEqual(got.Lineup, want.Lineup) ||
		!reflect.DeepEqual(got.Roster, want.Roster) || !reflect.DeepEqual(got.IR, want.IR) || !reflect.DeepEqual(got.Taxi, want.Taxi) {
		t.Fatal("a new 32-Team Default league's rules differ from the spec preset")
	}
	legacy, err := RulesInForce(legacyLeague(t), 2026, noChangeTime())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Cap, legacy.Cap) || !reflect.DeepEqual(got.Contracts, legacy.Contracts) ||
		!reflect.DeepEqual(got.Playoffs, legacy.Playoffs) || !reflect.DeepEqual(got.Trades, legacy.Trades) {
		t.Fatal("the 32-Team Default must take everything outside the spec from Legacy")
	}
	r, i := got.Roster, got.IR
	if r.Limits.Value.Max.Value != 44 || r.PositionBounds.Value[leaguerules.OL].Min.Value != 1 ||
		r.PositionBounds.Value[leaguerules.OL].Max.Value != 1 || got.Taxi.Slots.Value != 8 || got.Taxi.MaximumYears.Value != 3 ||
		i.Slots.Value != 12 || !reflect.DeepEqual(i.Eligible.Value, []leaguerules.NFLStatus{leaguerules.InjuredReserve, leaguerules.PUP}) {
		t.Fatalf("roster, taxi or IR differ from spec §2: %+v %+v %+v", r, got.Taxi, i)
	}
	if !strings.Contains(recordLabel(records[0]), "32-Team Default") {
		t.Fatalf("history label: %q", recordLabel(records[0]))
	}
}

func TestDefault32PresetIsOnlyFor32Teams(t *testing.T) {
	for _, count := range []int{2, 16, 31} {
		_, err := (RawCommand{League: "native.test", Command: Create,
			Args: RawArgs{Name: "Ours", Preset: PresetDefault32, FranchiseCount: count, Season: 2026}}).Prepared()
		if err == nil {
			t.Errorf("%d teams accepted", count)
		}
	}
	records := default32League(t)
	records[0].Command.Args.Seed = nil
	if err := records[0].Command.Args.validateCreation(); err == nil {
		t.Fatal("a 32-Team Default league without its values was accepted")
	}
}

// Spec §1: 19 starters with the O-Line card, TE only through the flex, no QB in the flex.
func TestDefault32LineupTable(t *testing.T) {
	rules := leaguerules.Default32().Lineup
	lineup := func(extra ...leaguerules.Position) []nativecheck.Starter {
		base := []leaguerules.Position{leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.PK, leaguerules.OL,
			leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.CB, leaguerules.S}
		var out []nativecheck.Starter
		for i, p := range append(base, extra...) {
			out = append(out, nativecheck.Starter{ID: string(rune('a' + i)), Position: p})
		}
		return out
	}
	idp := []leaguerules.Position{leaguerules.LB, leaguerules.LB, leaguerules.S, leaguerules.CB, leaguerules.DE}
	for name, c := range map[string]struct {
		starters []nativecheck.Starter
		valid    bool
	}{
		"three TE flex":       {lineup(append([]leaguerules.Position{leaguerules.TE, leaguerules.TE, leaguerules.TE}, idp...)...), true},
		"RB WR TE flex":       {lineup(append([]leaguerules.Position{leaguerules.RB, leaguerules.WR, leaguerules.TE}, idp...)...), true},
		"QB in the flex":      {lineup(append([]leaguerules.Position{leaguerules.QB, leaguerules.WR, leaguerules.TE}, idp...)...), false},
		"no O-Line card":      {lineup(append([]leaguerules.Position{leaguerules.RB, leaguerules.WR, leaguerules.TE}, idp...)...)[:4], false},
		"two O-Line cards":    {lineup(append([]leaguerules.Position{leaguerules.RB, leaguerules.WR, leaguerules.OL}, idp...)...), false},
		"four offensive flex": {lineup(append([]leaguerules.Position{leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.TE}, idp[:4]...)...), false},
	} {
		v := nativecheck.JudgeLineup(rules, true, c.starters)
		if (v.Status == nativecheck.LineupValid) != c.valid {
			t.Errorf("%s: %s", name, v.Words)
		}
	}
}
