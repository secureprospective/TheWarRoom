package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/rulebook"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

const legalityLeague = "native.seed"

// legalityLineup is a legal Legacy lineup: its eight offensive, one kicker and twelve defensive starters.
func legalityLineup() []string {
	return []string{"QB", "RB", "RB", "WR", "WR", "WR", "TE", "TE", "PK", "DT", "DT", "DE", "DE",
		"LB", "LB", "LB", "LB", "CB", "CB", "S", "S"}
}

// legalityIDs names the players of the fixture's franchise 0002 (native franchise.02):
//
//	1000..1081  eighty-two active players; the first twenty-one fill a legal lineup
//	2001, 2002  on IR: the NFL lists the first as out, and lists the second with nothing
//	3001..3003  on the taxi squad, drafted 2025, 2024 and 2022 (the last is in his fifth season)
func legalityID(base, i int) string { return fmt.Sprintf("%04d", base+i) }

type legalityFixture struct {
	app   *App
	path  string
	pools *db.Pools
}

func legalityRaw(id, position, draft string) players.RawPlayer {
	return players.RawPlayer{ID: id, Name: "Player, " + id, Position: position, Team: "AAA", DraftYear: draft}
}

func legalityLookup(t *testing.T) normalize.Lookup {
	t.Helper()
	positions := legalityLineup()
	var raws []players.RawPlayer
	for i := 0; i < 82; i++ {
		position := "WR"
		if i < len(positions) {
			position = positions[i]
		}
		raws = append(raws, legalityRaw(legalityID(1000, i), position, "2018"))
	}
	for i := 0; i < 40; i++ {
		raws = append(raws, legalityRaw(legalityID(4000, i), "WR", "2019"))
	}
	for i := 1; i <= 6; i++ {
		raws = append(raws, legalityRaw(fmt.Sprintf("%04d", 5000+i), "WR", "2027"))
	}
	raws = append(raws, legalityRaw("2001", "LB", "2019"), legalityRaw("2002", "CB", "2019"),
		legalityRaw("3001", "WR", "2025"), legalityRaw("3002", "RB", "2024"), legalityRaw("3003", "TE", "2022"))
	lookup, err := normalize.NewLookup(raws)
	if err != nil {
		t.Fatal(err)
	}
	return lookup
}

func legalityMirror(t *testing.T, a *App) {
	t.Helper()
	p, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "mirror.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = p.Close() })
	a.rulebook = rulebook.New(p)
	cfg := league.RawConfig{CurrentSeason: 2026}
	for i := 32; i >= 1; i-- {
		cfg.Franchises = append(cfg.Franchises, league.Franchise{ID: fmt.Sprintf("%04d", i),
			Name: fmt.Sprintf("Held team %d", i), Abbrev: fmt.Sprintf("H%d", i), Division: "division.1"})
	}
	if err := a.rulebook.Initialize(t.Context(), lineupRulesSource{cfg: cfg}); err != nil {
		t.Fatal(err)
	}
	a.league = state.NewMirror(p, a.rulebook)
	if err := a.league.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	a.season = 2026
	snap := state.MirrorSnapshot{Season: a.season}
	add := func(id, franchise string, status domain.RosterStatus) {
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: id, FranchiseID: franchise, RosterStatus: status,
			Salary: 1000000, ContractYear: 2028, ContractStatus: domain.CStatusUFA})
	}
	for i := 0; i < 82; i++ {
		add(legalityID(1000, i), "0002", domain.RosterActive)
	}
	for i := 0; i < 40; i++ {
		add(legalityID(4000, i), "0001", domain.RosterActive)
	}
	add("2001", "0002", domain.RosterIR)
	add("2002", "0002", domain.RosterIR)
	for _, id := range []string{"3001", "3002", "3003"} {
		add(id, "0002", domain.RosterTaxi)
	}
	if _, err := a.league.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
}

// legalityApp is a seeded Legacy league on a database that can be physically reopened, with the held
// players directory and injuries feed a native check reads its facts from.
func legalityApp(t *testing.T) *legalityFixture {
	t.Helper()
	a := commissionerTestApp(t)
	path := filepath.Join(t.TempDir(), "legality.db")
	store, pools := openDirectoryHistory(t, path)
	a.history = store
	legalityMirror(t, a)
	a.lookup, a.hasLookup = legalityLookup(t), true
	at := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)
	out, err := playerid.New("2001")
	if err != nil {
		t.Fatal(err)
	}
	a.seasonInjuries = heldSeasonFeed[[]leaguefeed.Injury]{fetchedAt: at,
		value: []leaguefeed.Injury{{ID: out, Status: "Out"}}}
	a.seasonDraftResults = heldSeasonFeed[[]draftreports.Pick]{fetchedAt: at, value: []draftreports.Pick{}}
	a.seasonFuturePicks = heldSeasonFeed[[]draftreports.FuturePick]{fetchedAt: at, value: []draftreports.FuturePick{}}
	input := createNativeSeedLeague(t, a, commissioner.PresetLegacy)
	seedNativeTest(t, a, input)
	return &legalityFixture{app: a, path: path, pools: pools}
}

// reopen closes the database for real and builds a fresh App on the same file: no mirror, no rulebook.
func (f *legalityFixture) reopen(t *testing.T) {
	t.Helper()
	old := f.app
	if err := f.pools.Close(); err != nil {
		t.Fatal(err)
	}
	store, pools := openDirectoryHistory(t, f.path)
	f.pools = pools
	restarted := &App{ctx: t.Context(), history: store, started: make(chan struct{}),
		commissionerSession: testCommissionerSession{"creator"}}
	close(restarted.started)
	restarted.lookup, restarted.hasLookup, restarted.seasonInjuries = old.lookup, old.hasLookup, old.seasonInjuries
	f.app = restarted
}

// set records a rule change through the real command and returns its receipt.
func (f *legalityFixture) set(t *testing.T, field, value, effective string) commissioner.Receipt {
	t.Helper()
	houses, err := f.app.TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague,
		Command: commissioner.RulesSet, Args: commissioner.RawArgs{Field: field, Value: json.RawMessage(value)},
		Request: fmt.Sprintf("legality.%d", houses.Revision), Expected: houses.Revision,
		Reason: "Our league's ruling", Effective: effective, Visibility: "league"})
}

func (f *legalityFixture) revert(t *testing.T, receipt commissioner.Receipt) {
	t.Helper()
	houses, err := f.app.TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague, Command: commissioner.Revert,
		Args: commissioner.RawArgs{Entry: receipt.Entry}, Request: fmt.Sprintf("undo.%d", houses.Revision),
		Expected: houses.Revision, Reason: "Undo that ruling", Effective: "now", Visibility: "league"})
}

func (f *legalityFixture) roster(t *testing.T, franchise string) commissioner.NativeRosterFranchise {
	t.Helper()
	check, err := f.app.TargetNativeRosterCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range check.Franchises {
		if row.Franchise == franchise {
			return row
		}
	}
	t.Fatalf("no roster row for %s", franchise)
	return commissioner.NativeRosterFranchise{}
}

func (f *legalityFixture) reserves(t *testing.T) commissioner.NativeReserveFranchise {
	t.Helper()
	check, err := f.app.TargetNativeReserveCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range check.Franchises {
		if row.Franchise == "franchise.02" {
			return row
		}
	}
	t.Fatal("no reserve row for franchise.02")
	return commissioner.NativeReserveFranchise{}
}

func (f *legalityFixture) lineup(t *testing.T, starters []string) commissioner.NativeLineupCheck {
	t.Helper()
	check, err := f.app.TargetNativeLineupCheck(commissioner.RawNativeLineup{League: legalityLeague,
		Franchise: "franchise.02", Submitted: starters != nil, Starters: starters})
	if err != nil {
		t.Fatal(err)
	}
	return check
}

func firstStarters(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = legalityID(1000, i)
	}
	return out
}

// governs runs the survey's round trip for one rule: read, set through the real command, read the
// consequence, make the compensating revert, read again; then reopen the database and do it all again.
func (f *legalityFixture) governs(t *testing.T, field, value string, read func(*testing.T, *legalityFixture) string, before, after string) {
	t.Helper()
	for pass, name := range []string{"live", "after reopening the database"} {
		if pass == 1 {
			f.reopen(t)
		}
		if got := read(t, f); got != before {
			t.Fatalf("%s: before the ruling: %q, want %q", name, got, before)
		}
		receipt := f.set(t, field, value, "now")
		if got := read(t, f); got != after {
			t.Fatalf("%s: after the ruling: %q, want %q", name, got, after)
		}
		f.revert(t, receipt)
		if got := read(t, f); got != before {
			t.Fatalf("%s: after the compensating revert: %q, want %q", name, got, before)
		}
	}
}

func TestRosterLimitGovernsTheRosterVerdict(t *testing.T) {
	f := legalityApp(t)
	read := func(t *testing.T, f *legalityFixture) string { return f.roster(t, "franchise.02").Words }
	f.governs(t, "roster.limits", `{"min":18,"max":90}`, read, "Active roster 82 of 80: 2 over", "Within 18 to 90")
	f.governs(t, "roster.limits", `{"min":85,"max":90}`, read, "Active roster 82 of 80: 2 over", "Active roster 82 of 85: 3 under")
}

func TestRosterVerdictIgnoresReservesUntilTheLeagueCountsThem(t *testing.T) {
	f := legalityApp(t)
	row := f.roster(t, "franchise.02")
	if row.Counted != 82 || row.Fits {
		t.Fatalf("IR and taxi players were counted: %+v", row)
	}
	f.governs(t, "roster.reserveExcluded", `false`,
		func(t *testing.T, f *legalityFixture) string { return f.roster(t, "franchise.02").Words },
		"Active roster 82 of 80: 2 over", "Whole roster 87 of 80: 7 over")
	if other := f.roster(t, "franchise.01"); !other.Fits || other.Words != "Within 18 to 80" || other.Counted != 40 {
		t.Fatalf("%+v", other)
	}
}

func TestRosterPositionLimitsGovernTheRosterVerdict(t *testing.T) {
	f := legalityApp(t)
	f.governs(t, "roster.positionBounds", `{"QB":{"min":2,"max":3}}`,
		func(t *testing.T, f *legalityFixture) string {
			row := f.roster(t, "franchise.01")
			return fmt.Sprintf("%v %v", row.Fits, row.Problems)
		}, "true []", "false [QB 0 of 2: 2 under]")
}

func TestLineupRulesGovernTheLineupVerdict(t *testing.T) {
	f := legalityApp(t)
	read := func(t *testing.T, f *legalityFixture) string { return f.lineup(t, firstStarters(21)).Verdict.Words }
	f.governs(t, "lineup.groups", `{"offense":{"min":0,"max":8},"defense":{"min":13,"max":13},"total":{"min":21,"max":21}}`,
		read, "Valid lineup", "Invalid lineup: Defense: 12 starting, needs exactly 13")
	f.governs(t, "lineup.positions", `{"QB":{"min":1,"max":1},"RB":{"min":1,"max":1},"WR":{"min":2,"max":5},"TE":{"min":1,"max":3},`+
		`"PK":{"min":1,"max":1},"DT":{"min":2,"max":4},"DE":{"min":2,"max":4},"LB":{"min":2,"max":4},"CB":{"min":2,"max":4},"S":{"min":2,"max":4}}`,
		read, "Valid lineup", "Invalid lineup: RB: 2 starting, needs exactly 1")
	check := f.lineup(t, firstStarters(21))
	if check.Verdict.Status != "valid" || len(check.Starters) != 21 || check.Starters[0].Position != "QB" ||
		check.Starters[8].Position != "PK" {
		t.Fatalf("the Legacy lineup was not judged by positions held for it: %+v", check)
	}
}

func TestInvalidLineupPenaltyAndMissedFlagAreTheLeaguesChoice(t *testing.T) {
	f := legalityApp(t)
	penalty := func(t *testing.T, f *legalityFixture) string { return f.lineup(t, firstStarters(20)).Verdict.Penalty }
	f.governs(t, "lineup.invalidPenalty", `"none"`, penalty,
		"The highest-scoring starter's points are set to zero", "No penalty")
	missed := func(t *testing.T, f *legalityFixture) string {
		v := f.lineup(t, nil).Verdict
		return fmt.Sprintf("%s %v %v", v.Status, v.Flagged, v.Penalty == "")
	}
	f.governs(t, "lineup.missedFlag", `false`, missed, "missed true true", "missed false true")
}

func TestInjuredReserveRulesGovernWhoMustLeave(t *testing.T) {
	f := legalityApp(t)
	standing := func(id string) func(*testing.T, *legalityFixture) string {
		return func(t *testing.T, f *legalityFixture) string {
			for _, p := range f.reserves(t).Players {
				if p.Player == id {
					return string(p.Standing)
				}
			}
			t.Fatalf("player %s is on no reserve squad", id)
			return ""
		}
	}
	f.governs(t, "ir.mustLeaveHealthy", `false`, standing("2002"), "must_leave", "stays")
	f.governs(t, "ir.eligible", `["PUP"]`, standing("2001"), "stays", "must_leave")
	slots := func(t *testing.T, f *legalityFixture) string { return f.reserves(t).IR.Words }
	f.governs(t, "ir.slots", `1`, slots, "IR 2 of 12", "IR 2 of 1: 1 over")
}

func TestTaxiRulesGovernWhoMustLeave(t *testing.T) {
	f := legalityApp(t)
	standing := func(id string) func(*testing.T, *legalityFixture) string {
		return func(t *testing.T, f *legalityFixture) string {
			for _, p := range f.reserves(t).Players {
				if p.Player == id {
					return string(p.Standing)
				}
			}
			t.Fatalf("player %s is on no reserve squad", id)
			return ""
		}
	}
	f.governs(t, "taxi.maximumYears", `5`, standing("3003"), "must_leave", "stays")
	f.governs(t, "taxi.maximumYears", `1`, standing("3002"), "stays", "must_leave")
	// The held data has no career games, so a league that measures by them cannot tell, rather than
	// falling back on draft years.
	f.governs(t, "taxi.basis", `"career_games"`, standing("3003"), "must_leave", "unclear")
	f.governs(t, "taxi.slots", `2`, func(t *testing.T, f *legalityFixture) string { return f.reserves(t).Taxi.Words },
		"Taxi 3 of 8", "Taxi 3 of 2: 1 over")
}

func TestHouseRulesAreReadOnlyTextTheCommissionerWrites(t *testing.T) {
	f := legalityApp(t)
	read := func(t *testing.T, f *legalityFixture) string {
		h, err := f.app.TargetNativeHouseRules(legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("%s %v", h.Words, h.Rules)
	}
	f.governs(t, "house.texts", `{"b-trades":"No trades in week 17","a-conduct":"Be kind"}`, read,
		"1 house rule [{legacy-reference docs/league-rules/Official_Rulebook.md v2026 (read-only reference)}]",
		"2 house rules [{a-conduct Be kind} {b-trades No trades in week 17}]")
}

func TestNativeLegalityWire(t *testing.T) {
	f := legalityApp(t)
	roster, err := f.app.TargetNativeRosterCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	reserves, err := f.app.TargetNativeReserveCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	house, err := f.app.TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []struct {
		name string
		body json.RawMessage
	}{
		{"native-roster-check", legalityJSON(t, roster)},
		{"native-reserve-check", legalityJSON(t, reserves)},
		{"native-house-rules", legalityJSON(t, house)},
		{"native-lineup-check", legalityJSON(t, f.lineup(t, firstStarters(20)))},
	}
	for _, fixture := range fixtures {
		checkWire(t, "internal/commissioner/testdata/"+fixture.name+".wire.json", fixture.body, fixture.name)
	}
}

func legalityJSON[T commissioner.NativeRosterCheck | commissioner.NativeReserveCheck |
	commissioner.NativeHouseRules | commissioner.NativeLineupCheck](t *testing.T, value T) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestNativeLineupRefusesForeignReserveAndDuplicateStarters(t *testing.T) {
	f := legalityApp(t)
	for _, ids := range [][]string{{"4000"}, {"2001"}, {"3001"}, {"1000", "1000"}, {"missing"}} {
		_, err := f.app.TargetNativeLineupCheck(commissioner.RawNativeLineup{League: legalityLeague,
			Franchise: "franchise.02", Submitted: true, Starters: ids})
		if err == nil {
			t.Fatalf("invalid starters accepted: %v", ids)
		}
	}
}

func TestNativeRosterWaitsForRolloverAndRulesShareSeedAlignment(t *testing.T) {
	f := legalityApp(t)
	f.set(t, "calendar.rollover", `"2027-03-01T05:00:00Z"`, "now")
	pending := f.set(t, "roster.limits", `{"min":18,"max":90}`, "next_rollover")
	if got := f.roster(t, "franchise.02").Words; got != "Active roster 82 of 80: 2 over" {
		t.Fatalf("waiting rule counted early: %q", got)
	}
	f.reopen(t)
	if got := f.roster(t, "franchise.02").Words; got != "Active roster 82 of 80: 2 over" {
		t.Fatalf("waiting rule counted early after replay: %q", got)
	}
	f.app.history.SetClock(func() time.Time { return time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC) })
	if got := f.roster(t, "franchise.02").Words; got != "Within 18 to 90" {
		t.Fatalf("rollover did not activate rule: %q", got)
	}
	f.revert(t, pending)
	if got := f.roster(t, "franchise.02").Words; got != "Active roster 82 of 80: 2 over" {
		t.Fatalf("compensation did not restore rule: %q", got)
	}
	rules, err := f.app.TargetCommissionerRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	directory, err := f.app.TargetNativeDirectory(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range rules.Sections {
		for _, row := range section.Rows {
			if row.Field != "league.alignment" {
				continue
			}
			var alignment map[string]string
			if err := json.Unmarshal(row.Current, &alignment); err != nil {
				t.Fatal(err)
			}
			for _, seat := range directory.Franchises {
				if alignment[seat.Franchise] != seat.Division {
					t.Fatalf("rules and directory disagree for %s", seat.Franchise)
				}
			}
			return
		}
	}
	t.Fatal("no alignment rule row")
}

func TestNativeGovernedCatalogueHasNoOrphanedFields(t *testing.T) {
	f := legalityApp(t)
	roster, err := f.app.TargetNativeRosterCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	reserves, err := f.app.TargetNativeReserveCheck(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	lineup := f.lineup(t, firstStarters(21))
	edges := map[string][]commissioner.RuleLine{
		"roster.": roster.Rules, "lineup.": lineup.Rules, "ir.": reserves.Rules, "taxi.": reserves.Rules,
	}
	seen := map[string]bool{"house.texts": true}
	for _, field := range leaguerules.Catalogue() {
		for prefix, reading := range edges {
			if !strings.HasPrefix(string(field.Path), prefix) {
				continue
			}
			matches := 0
			for _, line := range reading {
				if line.Label == field.Label {
					matches++
				}
			}
			if matches != 1 {
				t.Fatalf("field %s reaches %d reading rules", field.Path, matches)
			}
			seen[string(field.Path)] = true
		}
	}
	proved := []string{"roster.limits", "roster.reserveExcluded", "roster.positionBounds",
		"lineup.positions", "lineup.groups", "lineup.invalidPenalty", "lineup.missedFlag", "ir.slots",
		"ir.eligible", "ir.mustLeaveHealthy", "taxi.slots", "taxi.maximumYears", "taxi.basis", "house.texts"}
	for _, path := range proved {
		if !seen[path] {
			t.Fatalf("round-trip proof has no reading edge: %s", path)
		}
		delete(seen, path)
	}
	if len(seen) != 0 {
		t.Fatalf("reading fields have no round-trip proof: %v", seen)
	}
}
