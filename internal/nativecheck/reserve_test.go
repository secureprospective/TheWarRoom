package nativecheck

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func legacyIR() leaguerules.IRRules     { return leaguerules.LegacyNFL().IR }
func legacyTaxi() leaguerules.TaxiRules { return leaguerules.LegacyNFL().Taxi }

func TestInjuredReserveVerdicts(t *testing.T) {
	for _, c := range []struct {
		name  string
		facts PlayerFacts
		stand Standing
		words string
	}{
		{"listed on injured reserve", PlayerFacts{DesignationHeld: true, Designation: leaguerules.InjuredReserve},
			Stays, "Stays on IR: the NFL lists him on injured reserve"},
		{"listed out", PlayerFacts{DesignationHeld: true, Designation: leaguerules.Out},
			Stays, "Stays on IR: the NFL lists him as out"},
		{"healthy", PlayerFacts{DesignationHeld: true},
			MustLeave, "Must come off IR: the NFL no longer lists him as injured"},
		{"listing not held", PlayerFacts{}, Unclear, "Can't tell: the NFL's listing of him is not held"},
	} {
		check := CheckReserves(legacyIR(), legacyTaxi(), 2026,
			[]RosterPlayer{{ID: "p", Status: domain.RosterIR}}, Facts{"p": c.facts})
		got := check.Players[0]
		if got.Standing != c.stand || got.Words != c.words || got.Squad != "IR" {
			t.Errorf("%s: %+v", c.name, got)
		}
	}
}

func TestHealthyPlayerMayStayWhenTheLeagueSaysSo(t *testing.T) {
	rules := legacyIR()
	rules.MustLeaveHealthy = leaguerules.Known(false)
	check := CheckReserves(rules, legacyTaxi(), 2026, []RosterPlayer{{ID: "p", Status: domain.RosterIR}},
		Facts{"p": {DesignationHeld: true}})
	if got := check.Players[0]; got.Standing != Stays || !strings.HasPrefix(got.Words, "May stay on IR") {
		t.Fatalf("%+v", got)
	}
	rules.MustLeaveHealthy = leaguerules.Needs[bool]("")
	check = CheckReserves(rules, legacyTaxi(), 2026, []RosterPlayer{{ID: "p", Status: domain.RosterIR}},
		Facts{"p": {DesignationHeld: true}})
	if got := check.Players[0]; got.Standing != Unclear || !strings.HasPrefix(got.Words, "Needs a ruling") {
		t.Fatalf("%+v", got)
	}
}

func TestListingTheLeagueDoesNotAcceptCannotHoldASpot(t *testing.T) {
	rules := legacyIR()
	rules.Eligible = leaguerules.Known([]leaguerules.NFLStatus{leaguerules.InjuredReserve})
	facts := PlayerFacts{DesignationHeld: true, Designation: leaguerules.Suspended}
	check := CheckReserves(rules, legacyTaxi(), 2026, []RosterPlayer{{ID: "p", Status: domain.RosterIR}}, Facts{"p": facts})
	if got := check.Players[0]; got.Standing != MustLeave || !strings.Contains(got.Words, "does not accept for IR") {
		t.Fatalf("%+v", got)
	}
	if e := CanEnterIR(rules, facts, 0); e.Allowed || !e.Known {
		t.Fatalf("%+v", e)
	}
}

func TestEnteringInjuredReserve(t *testing.T) {
	listed := PlayerFacts{DesignationHeld: true, Designation: leaguerules.PUP}
	healthy := PlayerFacts{DesignationHeld: true}
	for _, c := range []struct {
		name    string
		facts   PlayerFacts
		used    int
		allowed bool
		known   bool
		words   string
	}{
		{"listed with room", listed, 11, true, true, "May go on IR: the NFL lists him as physically unable to perform"},
		{"listed, squad full", listed, 12, false, true, "Cannot go on IR: all 12 IR spots are taken"},
		{"healthy", healthy, 0, false, true, "Cannot go on IR: the NFL no longer lists him as injured"},
		{"listing not held", PlayerFacts{}, 0, false, false, "Can't tell: the NFL's listing of him is not held"},
	} {
		e := CanEnterIR(legacyIR(), c.facts, c.used)
		if e.Allowed != c.allowed || e.Known != c.known || e.Words != c.words {
			t.Errorf("%s: %+v", c.name, e)
		}
	}
}

func TestSquadsReportHowFullTheyAre(t *testing.T) {
	var all []RosterPlayer
	all = append(all, players(domain.RosterIR, 13, "i")...)
	all = append(all, players(domain.RosterTaxi, 8, "t")...)
	all = append(all, players(domain.RosterActive, 5, "a")...)
	check := CheckReserves(legacyIR(), legacyTaxi(), 2026, all, nil)
	if check.IR.Words != "IR 13 of 12: 1 over" || !check.IR.Judged || check.Taxi.Words != "Taxi 8 of 8" || len(check.Players) != 21 {
		t.Fatalf("%+v", check)
	}
	rules := legacyIR()
	rules.Slots = leaguerules.Needs[int]("")
	if check := CheckReserves(rules, legacyTaxi(), 2026, all, nil); check.IR.Judged || !strings.HasPrefix(check.IR.Words, "Needs a ruling") {
		t.Fatalf("%+v", check.IR)
	}
}

func taxiVerdict(rules leaguerules.TaxiRules, f PlayerFacts) ReserveVerdict {
	return CheckReserves(legacyIR(), rules, 2026, []RosterPlayer{{ID: "p", Status: domain.RosterTaxi}}, Facts{"p": f}).Players[0]
}

func TestTaxiExperienceByCalendarYears(t *testing.T) {
	for _, c := range []struct {
		draft int
		stand Standing
		words string
	}{
		{2026, Stays, "May stay on the taxi squad: in his 1st NFL season; the limit is 3"},
		{2024, Stays, "May stay on the taxi squad: in his 3rd NFL season; the limit is 3"},
		{2023, MustLeave, "Must come off the taxi squad: in his 4th NFL season; the limit is 3"},
	} {
		// Career games are deliberately the opposite of what the calendar says: the rule is calendar years.
		got := taxiVerdict(legacyTaxi(), PlayerFacts{DraftYear: c.draft, DraftYearHeld: true,
			CareerGames: 200 * boolInt(c.stand == Stays), CareerGamesHeld: true})
		if got.Standing != c.stand || got.Words != c.words {
			t.Errorf("drafted %d: %+v", c.draft, got)
		}
	}
	if got := taxiVerdict(legacyTaxi(), PlayerFacts{CareerGamesHeld: true}); got.Standing != Unclear ||
		got.Words != "Can't tell: his draft year is not held" {
		t.Fatalf("%+v", got)
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestTaxiExperienceByCareerGames(t *testing.T) {
	rules := legacyTaxi()
	rules.Basis = leaguerules.Known(leaguerules.CareerGames)
	for _, c := range []struct {
		games int
		stand Standing
	}{{0, Stays}, {51, Stays}, {52, MustLeave}} {
		// The draft year says rookie; only the games decide.
		got := taxiVerdict(rules, PlayerFacts{DraftYear: 2026, DraftYearHeld: true, CareerGames: c.games, CareerGamesHeld: true})
		if got.Standing != c.stand {
			t.Errorf("%d games: %+v", c.games, got)
		}
	}
	if got := taxiVerdict(rules, PlayerFacts{DraftYear: 2020, DraftYearHeld: true}); got.Standing != Unclear ||
		!strings.Contains(got.Words, "career games played are not held") {
		t.Fatalf("a held draft year stood in for games played: %+v", got)
	}
}

func TestEnteringTheTaxiSquad(t *testing.T) {
	rookie := PlayerFacts{DraftYear: 2025, DraftYearHeld: true}
	veteran := PlayerFacts{DraftYear: 2021, DraftYearHeld: true}
	if e := CanEnterTaxi(legacyTaxi(), rookie, 2026, 7); !e.Allowed || !strings.HasPrefix(e.Words, "May go on the taxi squad") {
		t.Fatalf("%+v", e)
	}
	if e := CanEnterTaxi(legacyTaxi(), rookie, 2026, 8); e.Allowed || e.Words != "Cannot go on the taxi squad: all 8 taxi spots are taken" {
		t.Fatalf("%+v", e)
	}
	if e := CanEnterTaxi(legacyTaxi(), veteran, 2026, 0); e.Allowed || !e.Known || !strings.Contains(e.Words, "6th NFL season") {
		t.Fatalf("%+v", e)
	}
	if e := CanEnterTaxi(legacyTaxi(), PlayerFacts{}, 2026, 0); e.Allowed || e.Known {
		t.Fatalf("%+v", e)
	}
	rules := legacyTaxi()
	rules.Basis = leaguerules.Needs[leaguerules.ExperienceBasis]("")
	if e := CanEnterTaxi(rules, rookie, 2026, 0); e.Allowed || e.Known || !strings.HasPrefix(e.Words, "Can't tell: Needs a ruling") {
		t.Fatalf("%+v", e)
	}
}

func TestOrdinals(t *testing.T) {
	for n, want := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st", 22: "22nd"} {
		if got := ordinal(n); got != want {
			t.Errorf("%d: %s", n, got)
		}
	}
}
