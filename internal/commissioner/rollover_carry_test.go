package commissioner

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestRolloverRoutesExpiringPlayersByRookieRound(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets.Players = []NativePlayer{
		{ID: "0001", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2026, RookieRound: 1},
		{ID: "0002", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2026, RookieRound: 2},
		{ID: "0003", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2026, RookieRound: 5},
		{ID: "0004", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2026, RookieRound: 6},
		{ID: "0005", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2026},
	}
	plan := requireRollover(t, state, rules)
	routes := map[string]string{}
	for _, pending := range plan.Pending {
		routes[pending.Player] = pending.Route
	}
	for id, want := range map[string]string{"0001": "Unrestricted free agent", "0002": "Restricted free agent",
		"0003": "Restricted free agent", "0004": "Unrestricted free agent", "0005": "cannot be determined"} {
		if !strings.Contains(routes[id], want) {
			t.Fatalf("player %s route %q, want %q", id, routes[id], want)
		}
	}
	for _, player := range plan.After.Players {
		if !player.Pending || player.Franchise != "f01" {
			t.Fatalf("expiring player not kept and marked pending: %+v", player)
		}
	}
}

func TestRolloverLeavesAnAlreadyEndedContractAlone(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets.Players = []NativePlayer{
		{ID: "0001", Franchise: "f01", Status: domain.RosterActive, ExpirationYear: 2024, Pending: true}}
	plan := requireRollover(t, state, rules)
	if len(plan.Pending) != 0 || !plan.After.Players[0].Pending ||
		!strings.Contains(strings.Join(plan.Lines, "\n"), "the contract ended in 2024 and is unchanged") {
		t.Fatalf("an old pending contract was counted as newly expiring: %+v", plan.Pending)
	}
}

func TestRolloverArchivesTheChampion(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets.Champion = "f07"
	state.Assets.Champions = []NativeChampion{{Season: 2025, Franchise: "f03"}}
	plan := requireRollover(t, state, rules)
	want := []NativeChampion{{Season: 2025, Franchise: "f03"}, {Season: 2026, Franchise: "f07"}}
	if !reflect.DeepEqual(plan.After.Champions, want) || plan.After.Champion != "" {
		t.Fatalf("champion not archived once: %+v %q", plan.After.Champions, plan.After.Champion)
	}
	if !strings.Contains(strings.Join(plan.Lines, "\n"), "Team f07, the season 2026 champion, is archived.") {
		t.Fatal("the plan does not name the archived champion")
	}
	if state.Assets.Champion != "f07" || len(state.Assets.Champions) != 1 {
		t.Fatal("archiving changed the before-state")
	}
	state.Assets.Champion = "nobody"
	if _, err := Rollover(state, rules, 2026, 2027); err == nil {
		t.Fatal("accepted a champion that is not a franchise")
	}
}

func TestRolloverCarriesDeadMoneyAndEndsSeasonCharges(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets.Adjustments = []NativeCharge{
		{ID: "dead-old", Kind: "dead_money", Franchise: "f01", Amount: 500, Season: 2026, Description: "Cut player"},
		{ID: "dead-standing", Kind: "dead_money", Franchise: "f01", Amount: 600, Description: "Held dead money"},
		{ID: "dead-later", Kind: "dead_money", Franchise: "f02", Amount: 700, Season: 2028, Description: "Later"},
		{ID: "bonus-old", Franchise: "f02", Amount: -800, Season: 2026, Description: "One-year credit"},
		{ID: "bonus-standing", Franchise: "f02", Amount: 900, Description: "Standing charge"},
	}
	plan := requireRollover(t, state, rules)
	seasons := map[string]int{}
	for _, charge := range plan.After.Adjustments {
		seasons[charge.ID] = charge.Season
	}
	want := map[string]int{"dead-old": 2027, "dead-standing": 0, "dead-later": 2028, "bonus-old": 2026,
		"bonus-standing": 0}
	if !reflect.DeepEqual(seasons, want) {
		t.Fatalf("charge seasons after rollover: %v, want %v", seasons, want)
	}
	lines := strings.Join(plan.Lines, "\n")
	for _, line := range []string{"Team f01: dead money (Cut player, $5.00) carries into season 2027.",
		"Team f02: the salary adjustment (One-year credit, -$8.00) ends with season 2026; the cap resets.",
		"Team f01: dead money (Held dead money, $6.00) keeps counting in season 2027."} {
		if !strings.Contains(lines, line) {
			t.Fatalf("missing %q in\n%s", line, lines)
		}
	}
	if state.Assets.Adjustments[0].Season != 2026 {
		t.Fatal("carrying dead money changed the before-state")
	}
}

func TestRolloverWithoutAssetsMovesOnlyRulesAndCalendar(t *testing.T) {
	state, rules := rolloverInputs()
	state.Assets = RawNativeSnapshot{Season: 2026}
	state.NoAssets = true
	plan := requireRollover(t, state, rules)
	if plan.Blocked || len(plan.After.Players) != 0 || plan.Rules.Season != 2027 ||
		!strings.Contains(strings.Join(plan.Lines, "\n"), "No starting assets are recorded") {
		t.Fatalf("an unseeded league's rollover: %+v", plan)
	}
	state.NoAssets = false
	if _, err := Rollover(state, rules, 2026, 2027); err == nil {
		t.Fatal("an empty asset snapshot passed for a league that has assets")
	}
}
