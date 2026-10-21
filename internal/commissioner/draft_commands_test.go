package commissioner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func draftBody(t *testing.T, command string, d RawDraftArgs, reason string) string {
	t.Helper()
	body, err := json.Marshal(RawCommand{League: "native.test", Command: command, Request: "r", Expected: 1,
		Args: RawArgs{Draft: &d}, Reason: reason, Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDraftCommandsRefuseWhatTheyDoNotTake(t *testing.T) {
	at := time.Date(2027, 7, 14, 16, 0, 0, 0, time.UTC)
	order := []string{"franchise.01", "franchise.02"}
	pick := &RawDraftSelection{Franchise: "franchise.01", Player: "0001", EntrySeason: 2027, Round: 1, Slot: 1,
		Original: "franchise.01", Terms: rookiedraft.Terms{Salary: 1, Years: 1}}
	good := map[string]RawDraftArgs{
		DraftOrderAuto:     {Season: 2027, Source: leaguerules.WorstStandingsFirst},
		DraftOrderSet:      {Season: 2027, Order: order},
		DraftOrderPattern:  {Pattern: leaguerules.Snake},
		DraftOrderTiebreak: {Tie: "Better record"},
		DraftStart:         {Season: 2027, At: at},
		DraftPick:          {Season: 2027, Pick: pick},
	}
	for command, args := range good {
		if _, err := Decode(draftBody(t, command, args, "")); err != nil {
			t.Fatalf("%s refused its own details: %v", command, err)
		}
	}
	for _, tc := range []struct {
		name, command string
		mutate        func(*RawDraftArgs)
	}{
		{"a source with a start", DraftStart, func(d *RawDraftArgs) { d.Source = leaguerules.WorstStandingsFirst }},
		{"a pattern with an order", DraftOrderSet, func(d *RawDraftArgs) { d.Pattern = leaguerules.Linear }},
		{"a tie rule with a pattern", DraftOrderPattern, func(d *RawDraftArgs) { d.Tie = "x" }},
		{"a season on a standing rule", DraftOrderPattern, func(d *RawDraftArgs) { d.Season = 2027 }},
		{"no season on a start", DraftStart, func(d *RawDraftArgs) { d.Season = 0 }},
		{"a running draft start", DraftStart, func(d *RawDraftArgs) { d.Running = true }},
		{"a running draft pick", DraftPick, func(d *RawDraftArgs) { d.Running = true }},
		{"a pick with a start", DraftStart, func(d *RawDraftArgs) { d.Pick = pick }},
		{"an unnamed franchise in an order", DraftOrderSet, func(d *RawDraftArgs) { d.Order = []string{"a", " "} }},
		{"a one franchise order", DraftOrderSet, func(d *RawDraftArgs) { d.Order = []string{"a"} }},
		{"a negative round", DraftOrderSet, func(d *RawDraftArgs) { d.Round = -1 }},
		{"another source", DraftOrderAuto, func(d *RawDraftArgs) { d.Source = "alphabet" }},
		{"another pattern", DraftOrderPattern, func(d *RawDraftArgs) { d.Pattern = "spiral" }},
		{"a blank tie rule", DraftOrderTiebreak, func(d *RawDraftArgs) { d.Tie = "  " }},
		{"a pick for the wrong season", DraftPick, func(d *RawDraftArgs) {
			d.Pick = &RawDraftSelection{
				Franchise: "f", Player: "0001", EntrySeason: 2026, Round: 1, Slot: 1, Original: "f",
				Terms: rookiedraft.Terms{Years: 1}}
		}},
		{"a pick of a player with a bare number", DraftPick, func(d *RawDraftArgs) { p := *pick; p.Player = "1"; d.Pick = &p }},
		{"a pick with no contract", DraftPick, func(d *RawDraftArgs) { p := *pick; p.Terms = rookiedraft.Terms{}; d.Pick = &p }},
	} {
		args := good[tc.command]
		tc.mutate(&args)
		if _, err := Decode(draftBody(t, tc.command, args, "")); err == nil {
			t.Fatalf("%s was accepted", tc.name)
		}
	}
}

func TestDraftChangesWhileTheClockRunsAreRecordedReasonChanges(t *testing.T) {
	set := RawCommand{Command: DraftOrderSet, Args: RawArgs{Draft: &RawDraftArgs{Season: 2027,
		Order: []string{"a", "b"}}}}
	if d, err := set.EffectiveDefinition(); err != nil || d.Gravity != "G2" {
		t.Fatalf("a regular order change: %+v, %v", d, err)
	}
	set.Args.Draft.Running = true
	if d, err := set.EffectiveDefinition(); err != nil || d.Gravity != "G3" {
		t.Fatalf("an order change while the clock runs: %+v, %v", d, err)
	}
	start := RawCommand{Command: DraftStart}
	if d, err := start.EffectiveDefinition(); err != nil || d.Gravity != "G1" {
		t.Fatalf("a draft start: %+v, %v", d, err)
	}
}

func TestDraftDescriptionsNameWhatWasRecorded(t *testing.T) {
	at := time.Date(2027, 7, 14, 16, 0, 0, 0, time.UTC)
	for command, want := range map[string]string{
		DraftStart:         "Set the 2027 rookie draft to start at 2027-07-14T16:00:00Z",
		DraftOrderAuto:     "Set the 2027 rookie draft order from the final standings",
		DraftOrderSet:      "Set the 2027 rookie draft order for round 2",
		DraftOrderPattern:  "Set the later-round draft pattern to snake",
		DraftOrderTiebreak: "Set the draft order tie rule to Better record",
	} {
		record := Record{Command: RawCommand{Command: command, Args: RawArgs{Draft: &RawDraftArgs{Season: 2027,
			Round: 2, At: at, Pattern: leaguerules.Snake, Tie: "Better record"}}}}
		if got := draftLabel(record); got != want {
			t.Fatalf("%s: %q, want %q", command, got, want)
		}
	}
}

// governsEdge is a draft setting a test sets through the real command and reads back.
type governsEdge struct {
	row, setting, proof string
}

// TestEveryDraftControlGovernsSomething holds the census rows for the draft to the edges the app tests
// prove: each row must govern at least one reading, through a setting that row really owns, and the proving
// test must exist.
func TestEveryDraftControlGovernsSomething(t *testing.T) {
	edges := []governsEdge{
		{"CM-040", "draft.clock", "TestDraftClockAndRoundsGovernTheDraft"},
		{"CM-040", "draft.rounds", "TestDraftClockAndRoundsGovernTheDraft"},
		{"CM-040", "draft.pricing", "TestDraftPricingAndLengthsGovernTheContractOnTheClock"},
		{"CM-040", "draft.lengths", "TestDraftPricingAndLengthsGovernTheContractOnTheClock"},
		{"CM-040", "draft.rfaAfter", "TestDraftPricingAndLengthsGovernTheContractOnTheClock"},
		{"CM-040", "draft.maximumPicks", "TestDraftMaximumPicksForfeitsTheExtrasWithoutAClock"},
		{"CM-041", DraftOrderSet, "TestDraftOrderCommandsGovernWhoIsOnTheClock"},
		{"CM-041", DraftOrderPattern, "TestDraftOrderCommandsGovernWhoIsOnTheClock"},
		{"CM-041", DraftOrderTiebreak, "TestDraftTieRuleIsRecordedAndReverted"},
		{"CM-041", DraftOrderAuto, "TestDraftOrderAutoAsksForTheFinalStandings"},
		{"CM-042", DraftStart, "TestDraftStartAndTheCalendarNeverDisagree"},
		{"CM-043", DraftSkip, "TestTheTraySkipsAndAssignsAndGovernsTheClockCM043"},
		{"CM-043", DraftAssign, "TestTheTraySkipsAndAssignsAndGovernsTheClockCM043"},
		{"CM-043", DraftInsert, "TestTheTrayInsertsAndChangesAPickWithoutRetargetingCM043"},
		{"CM-043", DraftChange, "TestTheTrayInsertsAndChangesAPickWithoutRetargetingCM043"},
		{"CM-043", DraftTimer, "TestTheTrayWorksTheClockAndOverridesAnExpiredPickCM043"},
		{"CM-044", DraftRevert, "TestARestoreKeepsBothStatesAndTheLedgerCM044"},
		{"CM-045", DraftArchive, "TestClosingTheDraftReleasesRookiesAndIssuesPicksCM045"},
	}
	sources := rootTestSource(t)
	proven := map[string]bool{}
	for _, e := range edges {
		owner := commandControls()[e.setting]
		if owner == "" {
			rule, err := ownerFor(leaguerules.FieldPath(e.setting))
			if err != nil {
				t.Fatalf("%s is not a setting any row owns", e.setting)
			}
			owner = rule.control
		}
		if owner != e.row {
			t.Fatalf("%s is owned by %s, not %s", e.setting, owner, e.row)
		}
		if !strings.Contains(sources, "func "+e.proof+"(t *testing.T)") {
			t.Fatalf("%s names a test that does not exist: %s", e.setting, e.proof)
		}
		proven[e.row] = true
	}
	for _, row := range []string{"CM-040", "CM-041", "CM-042", "CM-043", "CM-044", "CM-045"} {
		if !proven[row] {
			t.Fatalf("%s governs nothing the tests prove", row)
		}
	}
	for command, control := range commandControls() {
		if draftCommand(command) && command != DraftPick && !proven[control] {
			t.Fatalf("%s is owned by %s, which has no proven edge", command, control)
		}
	}
}

func rootTestSource(t *testing.T) string {
	t.Helper()
	files, err := filepath.Glob("../composition/native_draft*_test.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("the app tests for the draft are missing: %v", err)
	}
	var all strings.Builder
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		all.Write(body)
	}
	if !regexp.MustCompile(`func \(f \*draftFixture\) governs\(`).MatchString(all.String()) {
		t.Fatal("the round-trip helper is missing")
	}
	return all.String()
}

// The tray prints what a change affects only for a count it knows, so a warning has to travel with one,
// even for a change whose readings the catalogue does not list.
func TestAWarningTravelsWithACountTheTrayWillShow(t *testing.T) {
	for _, known := range [][]string{nil, {"Draft Status", "Rosters"}} {
		got := warned(PreviewAffects{Readings: known, Words: "Not known yet"}, "This swaps the rookie a pick holds.")
		if got.Count == nil || *got.Count != len(known) || got.Words != "This swaps the rookie a pick holds." {
			t.Fatalf("a warning over %d readings: %+v", len(known), got)
		}
	}
}
