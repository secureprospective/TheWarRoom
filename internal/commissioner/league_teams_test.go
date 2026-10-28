package commissioner

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// nativeLedger is a native league of n teams started the way creation starts it; edit changes the start.
func nativeLedger(t *testing.T, n int, edit func(*RawNativeSnapshot)) *ledgerBuilder {
	t.Helper()
	create, err := RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours", Preset: PresetEmpty,
		Source: SourceNative, Season: 2026, FranchiseCount: n}, Request: "create", Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	seed, err := NativeStart("native.test", b.records, seasonMoment(1))
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(&seed)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(1))
	return b
}

// change records a rule change after the same check the store runs, and returns that check's refusal.
func (b *ledgerBuilder) change(field, value string, day int) error {
	b.t.Helper()
	raw := ruleRaw(field, value, "now", "Agreed")
	raw.Request = fmt.Sprintf("r%d", len(b.records))
	if err := CheckRulesAppend(raw, b.records, seasonMoment(day)); err != nil {
		return err
	}
	b.add(raw, seasonMoment(day))
	return nil
}

func teamsPreview(t *testing.T, b *ledgerBuilder, field, value string) []PreviewField {
	t.Helper()
	preview, err := RulePreview(ruleRaw(field, value, "now", "Agreed"), b.records, PreviewCatalogue{}, seasonMoment(20))
	if err != nil {
		t.Fatal(err)
	}
	return preview.Fields
}

func TestTeamsChangeResizesTheLeague(t *testing.T) {
	b := nativeLedger(t, 12, nil)
	if got := teamsPreview(t, b, "league.teams", `14`); got[0].Before != "12" || got[0].After != "14 (adds Franchise 13, Franchise 14)" {
		t.Fatalf("preview up: %+v", got)
	}
	if err := b.change("league.teams", `14`, 2); err != nil {
		t.Fatal(err)
	}
	checkTeams(t, b, 14)
	directory, err := NativeDirectoryReading("native.test", b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	if seat := directory.Franchises[13]; seat.Franchise != "franchise.14" || seat.Name != "Franchise 14" ||
		seat.Owner != "Vacant" || seat.Division != "Needs a ruling" {
		t.Fatalf("new seat: %+v", seat)
	}
	point, ok, err := RosterRestorePoint("native.test", b.records)
	if err != nil || !ok || point.Validate() != nil || len(point.Snapshot.Franchises) != 14 {
		t.Fatalf("restore point: %v %v %+v", ok, err, point.Snapshot.Franchises)
	}

	if got := teamsPreview(t, b, "league.teams", `12`); got[0].After != "12 (removes Franchise 13, Franchise 14)" {
		t.Fatalf("preview down: %+v", got)
	}
	if err := b.change("league.teams", `12`, 4); err != nil {
		t.Fatal(err)
	}
	checkTeams(t, b, 12)
	state, err := GovernanceState(b.records)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := state.Franchise("franchise.13"); ok {
		t.Fatal("a removed team kept its seat")
	}
	if err := CheckRulesAppend(RawCommand{League: "native.test", Command: Revert, Args: RawArgs{Entry: "entry.4"},
		Request: "undo", Effective: "now", Visibility: "league"}, b.records, seasonMoment(5)); err == nil ||
		!strings.Contains(err.Error(), "change Teams again instead") {
		t.Fatalf("reverse a Teams change: %v", err)
	}
}

// checkTeams reads the franchise list from every reader that holds one.
func checkTeams(t *testing.T, b *ledgerBuilder, n int) {
	t.Helper()
	ids := LeagueFranchises(b.records)
	directory, err := NativeDirectoryReading("native.test", b.records, seasonMoment(20))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	rules, err := RulesNow(b.records, seasonMoment(20))
	if err != nil {
		t.Fatal(err)
	}
	people, err := PeopleReading("native.test", "creator", b.records)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{"list": len(ids), "directory": len(directory.Franchises), "assets": len(assets.Snapshot.Franchises),
		"rules": rules.League.Teams.Value, "people": len(people.Seats), "draft": len(draftOwners(b.records))}
	for reader, count := range got {
		if count != n {
			t.Errorf("%s reads %d teams, want %d", reader, count, n)
		}
	}
	if ids[n-1] != fmt.Sprintf("franchise.%02d", n) {
		t.Errorf("last franchise: %s", ids[n-1])
	}
}

func TestTeamsDownRefused(t *testing.T) {
	player := func(seed *RawNativeSnapshot) {
		seed.Players = append(seed.Players, NativePlayer{ID: "0001", Name: "Kept", Franchise: "franchise.12",
			Status: domain.RosterActive, Salary: 100, ExpirationYear: 2027})
	}
	for _, tc := range []struct {
		name  string
		edit  func(*RawNativeSnapshot)
		setup func(*ledgerBuilder)
		value string
		want  string
	}{
		{name: "player", edit: player, value: `10`,
			want: "Franchise 12 still has players; move or release them before removing the team"},
		{name: "owner", value: `11`, setup: func(b *ledgerBuilder) {
			b.add(RawCommand{League: "native.test", Command: "commish.seat.replace",
				Args: RawArgs{People: &RawPeople{Franchise: "franchise.12", Principal: "owner.twelve"}}}, seasonMoment(2))
		}, want: "Franchise 12 still has an owner; free the seat before removing the team"},
		{name: "game scored", value: `14`, setup: func(b *ledgerBuilder) {
			b.add(RawCommand{League: "native.test", Command: ScoreFranchise}, seasonMoment(2))
		}, want: "teams can't change once the season has games or player moves"},
		{name: "too many", value: `33`, want: "choose between 2 and 32 teams"},
		{name: "rollover", value: `14`, want: "Teams change takes effect now"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := nativeLedger(t, 12, tc.edit)
			if tc.setup != nil {
				tc.setup(b)
			}
			raw := ruleRaw("league.teams", tc.value, "now", "Agreed")
			if tc.name == "rollover" {
				raw.Effective = "next_rollover"
			}
			err := CheckRulesAppend(raw, b.records, seasonMoment(3))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal: %v", err)
			}
		})
	}
}

func TestDivisionsChangeRewritesAlignment(t *testing.T) {
	b := nativeLedger(t, 12, nil)
	one := map[string]string{}
	for _, id := range LeagueFranchises(b.records) {
		one[id] = "League"
	}
	body, _ := json.Marshal(one)
	b.set("league.alignment", string(body), "now", seasonMoment(2))
	b.set("league.divisionConferences", `{"League":"AFC"}`, "now", seasonMoment(3))
	if err := b.change("league.divisions", `1`, 4); err == nil || !strings.Contains(err.Error(), "already the rule") {
		t.Fatalf("same count: %v", err)
	}
	if err := b.change("league.divisions", `13`, 4); err == nil || !strings.Contains(err.Error(), "between 1 and 12 divisions") {
		t.Fatalf("more divisions than teams: %v", err)
	}
	fields := teamsPreview(t, b, "league.divisions", `4`)
	if len(fields) != 2 || fields[1].Label != "Division alignment" || !strings.Contains(fields[1].After, "Franchise 1: Division 1") {
		t.Fatalf("preview: %+v", fields)
	}
	if err := b.change("league.divisions", `4`, 4); err != nil {
		t.Fatal(err)
	}
	reading, err := RulesReading("native.test", b.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range reading.Sections {
		for _, row := range section.Rows {
			if row.Field == alignmentField && !strings.Contains(row.Value, "Franchise 1: Division 1") {
				t.Fatalf("alignment listing: %s", row.Value)
			}
		}
	}
	rules, err := RulesNow(b.records, seasonMoment(5))
	if err != nil {
		t.Fatal(err)
	}
	if rules.League.Divisions.Value != 4 {
		t.Fatalf("divisions: %+v", rules.League.Divisions)
	}
	for i, id := range LeagueFranchises(b.records) {
		if want := fmt.Sprintf("Division %d", i/3+1); rules.League.Alignment.Value[id] != want {
			t.Fatalf("%s in %q, want %q", id, rules.League.Alignment.Value[id], want)
		}
	}
	if c := rules.League.DivisionConferences; c.State != leaguerules.IsKnown || len(c.Value) != 0 {
		t.Fatalf("a vanished division kept its conference: %+v", c)
	}
}

func TestTeamsAndDivisionsFollowAnImport(t *testing.T) {
	b := newLedger(t, PresetLegacy)
	for _, field := range []string{"league.teams", "league.divisions"} {
		for _, effective := range []string{"now", "next_rollover"} {
			err := CheckRulesAppend(ruleRaw(field, `12`, effective, "Agreed"), b.records, seasonMoment(2))
			if err == nil || !strings.Contains(err.Error(), "follows the league's MFL franchises in an imported league") {
				t.Fatalf("%s %s: %v", field, effective, err)
			}
		}
	}
	if got := len(LeagueFranchises(b.records)); got != 32 {
		t.Fatalf("import franchises: %d", got)
	}
}

func TestFranchisesResizedRefusesASeatedTeam(t *testing.T) {
	state, err := governance.Replay(append(creationEvents(Record{Principal: "boss", Command: RawCommand{
		Args: RawArgs{Franchises: franchiseIDs(3)}}}), governance.SeatReplaced{Franchise: "franchise.03", Principal: "owner"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := governance.Apply(state, teamsEvents(2)[0]); err == nil || !strings.Contains(err.Error(), "Franchise 3 still has an owner") {
		t.Fatalf("removed a seated team: %v", err)
	}
	grown, err := governance.Apply(state, teamsEvents(4)[0])
	if err != nil {
		t.Fatal(err)
	}
	if f, ok := grown.Franchise("franchise.04"); !ok || f.Name != "Franchise 4" || f.Seat != "" {
		t.Fatalf("added team: %+v", f)
	}
	if f, _ := grown.Franchise("franchise.03"); f.Seat != "owner" {
		t.Fatalf("a kept team lost its owner: %+v", f)
	}
}

func TestAlignmentUsesCurrentNamesAndFallsBackToID(t *testing.T) {
	b := nativeLedger(t, 12, nil)
	b.add(RawCommand{Command: "commish.franchise.edit", Effective: "now", Args: RawArgs{People: &RawPeople{
		Franchise: "franchise.01", Name: "Renamed Club", Abbreviation: "RC"}}}, seasonMoment(2))
	fields := teamsPreview(t, b, "league.divisions", `4`)
	if len(fields) != 2 || !strings.Contains(fields[1].After, "Renamed Club: Division 1") {
		t.Fatalf("renamed preview: %+v", fields)
	}
	direct := teamsPreview(t, b, alignmentField, `{"franchise.01":"East"}`)
	if len(direct) != 1 || !strings.Contains(direct[0].After, "Renamed Club: East") {
		t.Fatalf("direct alignment preview: %+v", direct)
	}
	words, err := alignmentWords(b.records, leaguerules.Known(map[string]string{"missing.id": "East"}))
	if err != nil || words != "missing.id: East" {
		t.Fatalf("unknown franchise: %q, %v", words, err)
	}
}
