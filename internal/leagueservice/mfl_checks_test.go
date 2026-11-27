package leagueservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/mflref/moves"
)

// mflDefaultLeagueData is recorded-shape MFL data for team 0001: `active` filler players at $0.05M plus a
// lineup squad, under a $10M cap with IR counting half. Team 0002 holds one player.
func mflDefaultLeagueData(active int) mflmirror.LeagueData {
	rows := []mflmirror.RosterRow{
		{Franchise: "0001", Player: "1001", Salary: 100_000_00, ContractYear: 2027, Status: "ROSTER"},
		{Franchise: "0001", Player: "1002", Salary: 100_000_00, ContractYear: 2027, Status: "ROSTER"},
		{Franchise: "0001", Player: "1003", Salary: 100_000_00, ContractYear: 2027, Status: "ROSTER"},
		{Franchise: "0001", Player: "1004", Salary: 100_000_00, ContractYear: 2027, Status: "ROSTER"},
		{Franchise: "0001", Player: "1005", Salary: 100_000_00, ContractYear: 2027, Status: "INJURED_RESERVE"},
		{Franchise: "0001", Player: "1006", Salary: 100_000_00, ContractYear: 2027, Status: "TAXI_SQUAD"},
		{Franchise: "0001", Player: "1008", Salary: 2_000_000_00, ContractYear: 2028, Status: "ROSTER"},
		{Franchise: "0002", Player: "1007", Salary: 100_000_00, ContractYear: 2027, Status: "ROSTER"},
	}
	for i := 0; i < active-5; i++ { // 1001-1004 and 1008 are active already
		rows = append(rows, mflmirror.RosterRow{Franchise: "0001", Player: fmt.Sprintf("9%03d", i), Salary: 50_000_00,
			ContractYear: 2026, Status: "ROSTER"})
	}
	config, _ := json.Marshal(league.RawConfig{SalaryCapAmount: "10", RosterSize: "48", InjuredReserve: "12", TaxiSquad: "8", IncludeTaxiWithSalary: "0", IncludeIRWithSalary: "50",
		Starters: league.Starters{Count: "3", Positions: []league.PositionLimit{{Name: "QB", Limit: "1"},
			{Name: "RB", Limit: "1"}, {Name: "WR", Limit: "1"}}}})
	return mflmirror.LeagueData{Rosters: rows, Byes: map[string]int{"MIA": 9, "BUF": 12}, Config: config}
}

// mflOwnerFixture refreshes and picks team 0001 for owner.one.
func mflOwnerFixture(t *testing.T, data *mflmirror.LeagueData) mflMovesFixture {
	t.Helper()
	f := newMflMovesFixture(t, "owner.one")
	f.mfl.data = data
	ctx := t.Context()
	if _, err := f.service.MflMovesRefreshContext(ctx, RawMflMovesScope{League: f.league}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.MflMovesTeamSetContext(ctx, RawMflTeamSet{League: f.league, Team: "0001"}); err != nil {
		t.Fatal(err)
	}
	return f
}

func hasProblem(problems []string, part string) bool {
	for _, p := range problems {
		if strings.Contains(p, part) {
			return true
		}
	}
	return false
}

func TestMflChecksSayPickYourTeamFirst(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	ctx := t.Context()
	if _, err := f.service.MflMovesRefreshContext(ctx, RawMflMovesScope{League: f.league}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.MflMovesBidCheckContext(ctx, RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000}); err == nil ||
		!strings.Contains(err.Error(), "Pick your MFL team first") {
		t.Fatalf("bid check with no team: %v", err)
	}
	if _, err := f.service.MflMovesLineupCheckContext(ctx, RawMflLineupCheck{League: f.league, Week: 3, Starters: []string{"1001"}}); err == nil ||
		!strings.Contains(err.Error(), "Pick your MFL team first") {
		t.Fatalf("lineup check with no team: %v", err)
	}
}

func TestMflBidCheckPassesAGoodBid(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000, Years: 1})
	if err != nil || !got.OK || len(got.Problems) != 0 {
		t.Fatalf("good bid = %+v, %v", got, err)
	}
	if got.CapRoomAfter == nil || got.ActiveAfter != 41 {
		t.Fatalf("figures = %+v", got)
	}
}

func TestMflBidCheckCapRoomAfterTheBid(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	// Held: 2.0M + 0.4M (4 x 0.1M active) + 0.05M IR half + 43 x 0.05M = ~4.2M of 10M; a 6M bid goes over.
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 600_000_000, Years: 3})
	if err != nil || got.OK || !hasProblem(got.Problems, "salary cap") {
		t.Fatalf("over-cap bid = %+v, %v", got, err)
	}
	// Dropping the $2M player on the roster makes room: his salary frees, less 35% of one remaining year.
	ok, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 600_000_000, Years: 3, Drop: []string{"1008"}})
	if err != nil || hasProblem(ok.Problems, "salary cap") {
		t.Fatalf("bid with a drop = %+v, %v", ok, err)
	}
}

func TestMflBidCheckRejectsMoreThanOneDropInPlainWords(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	_, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league,
		Add: "0531", AmountCents: 100_000_000, Drop: []string{"1001", "1002"}})
	if err == nil || !strings.Contains(err.Error(), "a bid can hold one drop") {
		t.Fatalf("two-drop bid error = %v", err)
	}
}

func TestMflBidCheckRosterSizeAfterDrops(t *testing.T) {
	data := mflDefaultLeagueData(48)
	f := mflOwnerFixture(t, &data)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000})
	if err != nil || got.ActiveAfter != 49 || !hasProblem(got.Problems, "48 active roster spots") {
		t.Fatalf("49 active = %+v, %v", got, err)
	}
	dropped, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000, Drop: []string{"1001"}})
	if err != nil || dropped.ActiveAfter != 48 || hasProblem(dropped.Problems, "48 active roster spots") {
		t.Fatalf("drop makes room = %+v, %v", dropped, err)
	}
	small := mflDefaultLeagueData(20)
	f2 := mflOwnerFixture(t, &small)
	below, err := f2.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f2.league, Add: "0531", AmountCents: 100_000_000})
	// MFL names no roster minimum, so none is invented: the check says it did not look.
	if err != nil || hasProblem(below.Problems, "in-season minimum") || !hasProblem(below.NotChecked, "no roster minimum") {
		t.Fatalf("below 35 = %+v, %v", below, err)
	}
}

// config is a copy of the default league whose rules change by the given edit.
func mflLeagueWith(active int, edit func(*league.RawConfig)) mflmirror.LeagueData {
	data := mflDefaultLeagueData(active)
	var c league.RawConfig
	_ = json.Unmarshal(data.Config, &c)
	edit(&c)
	data.Config, _ = json.Marshal(c)
	return data
}

func TestMflBidCheckUsesTheLeaguesOwnRosterSize(t *testing.T) {
	// 25 active players is under the desktop's 48 but a 26th is over this league's 25.
	data := mflLeagueWith(25, func(c *league.RawConfig) { c.RosterSize = "25" })
	f := mflOwnerFixture(t, &data)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000})
	if err != nil || got.ActiveAfter != 26 || !hasProblem(got.Problems, "fit 25 active roster spots") {
		t.Fatalf("26 active in a 25-player league = %+v, %v", got, err)
	}
	fits, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000, Drop: []string{"1001"}})
	if err != nil || hasProblem(fits.Problems, "active roster spots") {
		t.Fatalf("a drop makes room = %+v, %v", fits, err)
	}
}

func TestMflBidCheckWithNoRosterSizeSaysItDidNotLook(t *testing.T) {
	data := mflLeagueWith(60, func(c *league.RawConfig) { c.RosterSize = "" })
	f := mflOwnerFixture(t, &data)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000})
	if err != nil || hasProblem(got.Problems, "active roster spots") || !hasProblem(got.NotChecked, "no roster size") {
		t.Fatalf("no roster size = %+v, %v", got, err)
	}
}

func TestMflBidCheckFlagsReserveSlotsOverTheLeagueLimit(t *testing.T) {
	data := mflLeagueWith(30, func(c *league.RawConfig) { c.InjuredReserve = "0"; c.TaxiSquad = "0" })
	f := mflOwnerFixture(t, &data)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 100_000_000})
	if err != nil || !hasProblem(got.Problems, "injured reserve holds 1") || !hasProblem(got.Problems, "taxi squad holds 1") {
		t.Fatalf("reserve limits = %+v, %v", got, err)
	}
}

func TestMflBidCheckMinimumSalary(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	// 0531 was drafted 2020: six years, minimum $530,000.
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 10_000_000})
	if err != nil || !hasProblem(got.Problems, "minimum salary for this player is $530,000") {
		t.Fatalf("below minimum salary = %+v, %v", got, err)
	}
}

func TestMflBidCheckBidLimit(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	// The Legacy rules cap a one year bid; a very large one year bid breaks the limit or the cap, and a
	// long bid at the same salary must not trip the length rule.
	long, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 200_000_000, Years: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(long.NotChecked) != 0 {
		t.Skipf("fixture rules do not set the bid limits: %v", long.NotChecked)
	}
	short, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: 1_300_000_000, Years: 1})
	if err != nil || !hasProblem(short.Problems, "1 year bid can be no more than") {
		t.Fatalf("one year limit = %+v, %v", short, err)
	}
}

func TestMflBidCheckPlayerOnAnotherTeam(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "1007", AmountCents: 100_000_000})
	if err != nil || got.OK || !hasProblem(got.Problems, "another team's roster") {
		t.Fatalf("rostered player = %+v, %v", got, err)
	}
}

func lineupCheck(t *testing.T, f mflMovesFixture, week int, starters ...string) MflLineupCheck {
	t.Helper()
	got, err := f.service.MflMovesLineupCheckContext(t.Context(), RawMflLineupCheck{League: f.league, Week: week, Starters: starters})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func hasKind(got MflLineupCheck, kind string) bool {
	for _, p := range got.Problems {
		if p.Kind == kind {
			return true
		}
	}
	return false
}

func TestMflLineupCheckSlotCounts(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	if got := lineupCheck(t, f, 3, "1001", "1002", "1003"); !got.OK || !got.Full || len(got.Problems) != 0 {
		t.Fatalf("legal lineup = %+v", got)
	}
	over := lineupCheck(t, f, 3, "1001", "1002", "1003", "1008") // two WR where one is allowed
	if over.OK || !hasKind(over, lineup.KindOver) {
		t.Fatalf("too many WR = %+v", over)
	}
	short := lineupCheck(t, f, 3, "1001", "1002") // no WR: partial but allowed
	if !short.OK || short.Full || !hasKind(short, lineup.KindShort) {
		t.Fatalf("a slot short = %+v", short)
	}
}

func TestMflLineupCheckBye(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	if got := lineupCheck(t, f, 9, "1001", "1002", "1004"); got.OK || !hasKind(got, "bye") {
		t.Fatalf("player on bye in week 9 = %+v", got)
	}
	if got := lineupCheck(t, f, 10, "1001", "1002", "1004"); !got.OK {
		t.Fatalf("same player, other week = %+v", got)
	}
}

func TestMflLineupCheckInjuredReserveAndTaxi(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	if got := lineupCheck(t, f, 3, "1001", "1002", "1005"); got.OK || !hasKind(got, "injured-reserve") {
		t.Fatalf("player on IR = %+v", got)
	}
	if got := lineupCheck(t, f, 3, "1001", "1006", "1003"); got.OK || !hasKind(got, "taxi-squad") {
		t.Fatalf("player on taxi = %+v", got)
	}
	if got := lineupCheck(t, f, 3, "1001", "1002", "1007"); got.OK || !hasKind(got, "not-on-roster") {
		t.Fatalf("another team's player = %+v", got)
	}
}

func planBody(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func saveBid(t *testing.T, f mflMovesFixture, add string, dollars int64) (MflPlan, error) {
	t.Helper()
	return f.service.MflMovesPlanSaveContext(t.Context(), RawMflPlanSave{League: f.league, Kind: "bid",
		Body: planBody(map[string]any{"add": add, "amountCents": dollars * 100, "years": 1})})
}

func TestMflPlanSaveRerunsTheCheckAndKeepsItPrivate(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	ctx := t.Context()
	if _, err := saveBid(t, f, "0531", 100_000); err == nil || !strings.Contains(err.Error(), "minimum salary") {
		t.Fatalf("a failing bid was saved: %v", err)
	}
	plan, err := saveBid(t, f, "0531", 1_000_000)
	if err != nil || plan.Kind != "bid" || !plan.CanDone || !plan.CanWithdraw || plan.PlayerName != "Doe, John" {
		t.Fatalf("saved bid = %+v, %v", plan, err)
	}
	single, err := f.service.MflMovesPlansReadContext(ctx, RawMflMovesScope{League: f.league})
	if err != nil || single.Words != "1 plan. Nothing here was sent to MFL." {
		t.Fatalf("one-plan wording = %+v, %v", single, err)
	}
	if _, err := saveBid(t, f, "0531", 1_200_000); err == nil || !strings.Contains(err.Error(), "already have an open bid plan") {
		t.Fatalf("second plan on the same player: %v", err)
	}
	lp, err := f.service.MflMovesPlanSaveContext(ctx, RawMflPlanSave{League: f.league, Kind: "lineup",
		Body: planBody(map[string]any{"week": 3, "starters": []string{"1001", "1002", "1003"}})})
	if err != nil || lp.Kind != "lineup" || lp.Week != 3 || len(lp.Starters) != 3 {
		t.Fatalf("saved lineup = %+v, %v", lp, err)
	}
	if _, err := f.service.MflMovesPlanSaveContext(ctx, RawMflPlanSave{League: f.league, Kind: "lineup",
		Body: planBody(map[string]any{"week": 9, "starters": []string{"1001", "1002", "1004"}})}); err == nil ||
		!strings.Contains(err.Error(), "bye") {
		t.Fatalf("a lineup with a player on bye was saved: %v", err)
	}
	mine, err := f.service.MflMovesPlansReadContext(ctx, RawMflMovesScope{League: f.league})
	if err != nil || len(mine.Plans) != 2 {
		t.Fatalf("my plans = %+v, %v", mine, err)
	}
	// Another seat sees none of them, and cannot change, link or finish them.
	host := f.service.host.(StoredHost)
	host.Principal = "owner.two"
	other := NewHosted(host, admissionFunc(func(context.Context, commissioner.Access) error { return nil }))
	theirs, err := other.MflMovesPlansReadContext(ctx, RawMflMovesScope{League: f.league})
	if err != nil || len(theirs.Plans) != 0 {
		t.Fatalf("another seat reads %d plans (%v)", len(theirs.Plans), err)
	}
	scope := RawMflPlanScope{League: f.league, Plan: plan.ID}
	if _, err := other.MflMovesPlanLinkContext(ctx, scope); err == nil || !strings.Contains(err.Error(), mflPlanNotFound) {
		t.Fatalf("another seat linked my plan: %v", err)
	}
	if _, err := other.MflMovesPlanDoneContext(ctx, scope); err == nil || !strings.Contains(err.Error(), mflPlanNotFound) {
		t.Fatalf("another seat finished my plan: %v", err)
	}
	if _, err := other.MflMovesPlanWithdrawContext(ctx, scope); err == nil || !strings.Contains(err.Error(), mflPlanNotFound) {
		t.Fatalf("another seat withdrew my plan: %v", err)
	}
	if _, err := f.service.MflMovesPlanSaveContext(ctx, RawMflPlanSave{League: f.league, Franchise: "franchise.02", Kind: "bid",
		Body: planBody(map[string]any{"add": "0531", "amountCents": 100_000_000})}); err == nil ||
		!strings.Contains(err.Error(), "no request franchise") {
		t.Fatalf("a request naming a franchise saved: %v", err)
	}
}

func TestMflPlanWithdrawOnlyFromPlanned(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	ctx := t.Context()
	a, err := saveBid(t, f, "0531", 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	done, err := f.service.MflMovesPlanDoneContext(ctx, RawMflPlanScope{League: f.league, Plan: a.ID})
	if err != nil || done.CanWithdraw || done.CanDone || !strings.Contains(done.Words, "You said you did it") {
		t.Fatalf("done = %+v, %v", done, err)
	}
	if _, err := f.service.MflMovesPlanWithdrawContext(ctx, RawMflPlanScope{League: f.league, Plan: a.ID}); err == nil ||
		!strings.Contains(err.Error(), "still planned") {
		t.Fatalf("a plan marked done was withdrawn: %v", err)
	}
	if _, err := f.service.MflMovesPlanDoneContext(ctx, RawMflPlanScope{League: f.league, Plan: a.ID}); err == nil {
		t.Fatal("a plan was marked done twice")
	}
	b, err := f.service.MflMovesPlanSaveContext(ctx, RawMflPlanSave{League: f.league, Kind: "lineup",
		Body: planBody(map[string]any{"week": 3, "starters": []string{"1001", "1002", "1003"}})})
	if err != nil {
		t.Fatal(err)
	}
	gone, err := f.service.MflMovesPlanWithdrawContext(ctx, RawMflPlanScope{League: f.league, Plan: b.ID})
	if err != nil || gone.State != "withdrawn" || gone.CanWithdraw {
		t.Fatalf("withdraw a planned lineup = %+v, %v", gone, err)
	}
	if _, err := f.service.MflMovesPlanWithdrawContext(ctx, RawMflPlanScope{League: f.league, Plan: b.ID}); err == nil {
		t.Fatal("a withdrawn plan was withdrawn again")
	}
}

func TestMflPlanLinkIsAValidatedMflAddress(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	ctx := t.Context()
	lp, err := f.service.MflMovesPlanSaveContext(ctx, RawMflPlanSave{League: f.league, Kind: "lineup",
		Body: planBody(map[string]any{"week": 3, "starters": []string{"1001", "1002", "1003"}})})
	if err != nil {
		t.Fatal(err)
	}
	link, err := f.service.MflMovesPlanLinkContext(ctx, RawMflPlanScope{League: f.league, Plan: lp.ID})
	if err != nil || !strings.HasPrefix(link.URL, "https://www.myfantasyleague.com/2026/lineup?L=14432&WEEK=3&F=0001") {
		t.Fatalf("lineup link = %+v, %v", link, err)
	}
	bp, err := saveBid(t, f, "0531", 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if link, err = f.service.MflMovesPlanLinkContext(ctx, RawMflPlanScope{League: f.league, Plan: bp.ID}); err != nil ||
		!strings.HasPrefix(link.URL, "https://www.myfantasyleague.com/") {
		t.Fatalf("bid link = %+v, %v", link, err)
	}
}

// The golden files were shared with the former desktop contracts path.

func TestMflLineupCheckGivesTheDesktopVerdict(t *testing.T) {
	body, err := os.ReadFile("../mflref/lineup/testdata/hosted_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Cases []struct {
			Name  string   `json:"name"`
			IDs   []string `json:"ids"`
			Full  bool     `json:"full"`
			Legal bool     `json:"legal"`
			Kinds []string `json:"kinds"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(body, &g); err != nil {
		t.Fatal(err)
	}
	f := mflOwnerFixture(t, nil)
	for _, c := range g.Cases {
		got := lineupCheck(t, f, 3, c.IDs...)
		kinds := []string{}
		for _, p := range got.Problems {
			kinds = append(kinds, p.Kind)
		}
		if got.Full != c.Full || got.OK != c.Legal || (len(kinds)+len(c.Kinds) != 0 && !slices.Equal(kinds, c.Kinds)) {
			t.Fatalf("%s: hosted says full=%v ok=%v kinds=%v, desktop golden says full=%v legal=%v kinds=%v",
				c.Name, got.Full, got.OK, kinds, c.Full, c.Legal, c.Kinds)
		}
	}
}

func TestMflBidCheckGivesTheDesktopVerdict(t *testing.T) {
	body, err := os.ReadFile("../nativecheck/testdata/acquire_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		Cases []struct {
			Name    string   `json:"name"`
			Players int      `json:"players"`
			Amount  float64  `json:"amount"`
			After   int64    `json:"after"`
			Active  int      `json:"active"`
			Reasons []string `json:"reasons"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(body, &g); err != nil {
		t.Fatal(err)
	}
	for _, c := range g.Cases {
		data := mflDefaultLeagueData(c.Players)
		f := mflOwnerFixture(t, &data)
		got, err := f.service.MflMovesBidCheckContext(t.Context(), RawMflBidCheck{League: f.league, Add: "0531", AmountCents: domain.Money(c.Amount * 100), Years: 3})
		if err != nil {
			t.Fatal(err)
		}
		if c.After != 0 && int64(got.CapUsedAfter) != c.After || got.ActiveAfter != c.Active {
			t.Fatalf("%s: hosted cap after %d, active %d; golden %d, %d", c.Name, got.CapUsedAfter, got.ActiveAfter, c.After, c.Active)
		}
		if len(c.Reasons) == 1 && strings.Contains(c.Reasons[0], "in-season minimum") {
			// The desktop holds a fixed 35; MFL names no minimum, so the hosted check leaves it out and says so.
			if hasProblem(got.Problems, "in-season minimum") || !hasProblem(got.NotChecked, "no roster minimum") {
				t.Fatalf("%s: hosted invented a minimum: %+v", c.Name, got)
			}
			continue
		}
		shared := []string{}
		for _, p := range got.Problems {
			if strings.HasPrefix(p, "The add ") || strings.HasPrefix(p, "The roster ") {
				shared = append(shared, p)
			}
		}
		if !slices.Equal(shared, c.Reasons) {
			t.Fatalf("%s: hosted reasons %v, desktop golden %v", c.Name, shared, c.Reasons)
		}
	}
}

func TestMflPlanLinkRefusesAnAddressOffMfl(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	ctx := t.Context()
	mirror := f.service.host.(StoredHost).Mirror
	store := moves.New(mirror.Pools())
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	id, _ := playerid.New("1001")
	e, err := envelope.New(f.clock.UTC(), envelope.Spec{Intent: "mfl.lineup.set", LeagueID: f.league, FranchiseID: "franchise.01",
		Subject:  envelope.Subject{Players: []playerid.PlayerID{id}},
		Expected: envelope.ExpectedChange{Lineup: &envelope.ExpectedLineup{Week: 3, Starters: []playerid.PlayerID{id}}},
		Gravity:  envelope.G1, Undo: envelope.Reversible,
		Target: envelope.Target{Kind: envelope.Mapped, URL: "https://evil.example.com/2026/lineup?L=14432"}},
		func() string { return "lineup-planted" })
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, e); err != nil {
		t.Fatal(err)
	}
	if link, err := f.service.MflMovesPlanLinkContext(ctx, RawMflPlanScope{League: f.league, Plan: "lineup-planted"}); err == nil {
		t.Fatalf("a link off MFL was returned: %+v", link)
	}
}

func TestMflRosterReadGivesThePickedTeamsRoster(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	got, err := f.service.MflMovesRosterReadContext(t.Context(), RawMflMovesScope{League: f.league})
	if err != nil || got.Team != "0001" {
		t.Fatalf("roster = %+v, %v", got, err)
	}
	status := map[string]string{}
	for _, p := range got.Players {
		status[p.ID] = p.Status
		if p.ID == "1001" && (p.Name != "Quinn QB" || p.Position != "QB" || p.Team != "BUF" || p.Salary != 100_000_00 || p.ContractThrough != 2027) {
			t.Fatalf("player 1001 = %+v", p)
		}
	}
	if status["1005"] != "IR" || status["1006"] != "taxi" || status["1008"] != "active" {
		t.Fatalf("statuses = %v", status)
	}
	if _, other := status["1007"]; other {
		t.Fatal("another team's player is on my roster")
	}
}

func TestMflRosterReadNeedsAPickedTeamAndRefusesARequestFranchise(t *testing.T) {
	f := newMflMovesFixture(t, "owner.one")
	if _, err := f.service.MflMovesRefreshContext(t.Context(), RawMflMovesScope{League: f.league}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.MflMovesRosterReadContext(t.Context(), RawMflMovesScope{League: f.league}); err == nil ||
		!strings.Contains(err.Error(), "Pick your MFL team first") {
		t.Fatalf("no team: %v", err)
	}
	if _, err := f.service.MflMovesRosterReadContext(t.Context(), RawMflMovesScope{League: f.league, Franchise: "franchise.02"}); err == nil {
		t.Fatal("a request naming a franchise was accepted")
	}
}

func TestMflLineupPlanSupersedesTheOlderPlanForTheSameWeek(t *testing.T) {
	f := mflOwnerFixture(t, nil)
	save := func(week int, ids ...string) MflPlan {
		t.Helper()
		plan, err := f.service.MflMovesPlanSaveContext(t.Context(), RawMflPlanSave{League: f.league, Kind: "lineup",
			Body: planBody(map[string]any{"week": week, "starters": ids})})
		if err != nil {
			t.Fatalf("save week %d: %v", week, err)
		}
		return plan
	}
	first := save(3, "1001", "1002", "1003")
	otherWeek := save(4, "1001", "1002", "1003")
	second := save(3, "1001", "1002", "1008")
	plans, err := f.service.MflMovesPlansReadContext(t.Context(), RawMflMovesScope{League: f.league})
	if err != nil {
		t.Fatal(err)
	}
	live := map[string]bool{}
	for _, p := range plans.Plans {
		live[p.ID] = p.CanWithdraw
	}
	if live[first.ID] || !live[second.ID] || !live[otherWeek.ID] {
		t.Fatalf("first week-3 plan still live=%v, newer=%v, week 4=%v", live[first.ID], live[second.ID], live[otherWeek.ID])
	}
}
