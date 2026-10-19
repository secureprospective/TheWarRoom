package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Legal and illegal previews are proved before recording the exact successful request.
func proveStorePreview(t *testing.T, s *Store, raw commissioner.RawCommand, principal string) {
	t.Helper()
	before, err := s.leagueRecords(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewCommissioner(t.Context(), raw, principal, nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 || len(preview.Fields) == 0 {
		t.Fatalf("%s legal preview: %+v %v", raw.Command, preview, err)
	}
	if len(preview.Fields) == 1 && preview.Fields[0].Label == "League ledger" &&
		preview.Fields[0].When != "Already recorded" {
		t.Fatalf("%s has no current-value preview: %+v", raw.Command, preview)
	}
	for _, field := range preview.Fields {
		if field.Before == "" || field.After == "" || field.BeforeSource == "" || field.AfterSource == "" {
			t.Fatalf("%s incomplete before/after: %+v", raw.Command, field)
		}
	}
	after, err := s.leagueRecords(t.Context(), raw.League)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("%s preview wrote: %v", raw.Command, err)
	}
	illegal := raw
	illegal.Visibility = "not-an-audience"
	blocked, err := s.PreviewCommissioner(t.Context(), illegal, principal, nil, commissioner.PreviewCatalogue{})
	if err != nil {
		t.Fatal(err)
	}
	_, executeErr := s.ExecuteCommissioner(t.Context(), illegal, principal, nil)
	var refused *commissioner.Refusal
	if !errors.As(executeErr, &refused) || len(blocked.Blockers) != 1 || blocked.Blockers[0] != refused.Reason() {
		t.Fatalf("%s illegal preview differs: %+v / %v", raw.Command, blocked, executeErr)
	}
	t.Logf("preview proof: %s", raw.Command)
}

func TestPreviewReadsTheStatQueueAndScoresBeforeAndAfter(t *testing.T) {
	for _, command := range []string{commissioner.StatApprove, commissioner.StatReject, commissioner.StatCorrect} {
		t.Run(command, func(t *testing.T) {
			f := statResultsTest(t)
			f.revise(f.a, 2, 30000)
			item := f.queue(t).Open[0]
			change := commissioner.RawStatChange{Item: item.ID}
			if command == commissioner.StatCorrect {
				change = commissioner.RawStatChange{Player: item.Player, Season: 2026, Week: 2, Stat: "tackle",
					Value: pointsDelta(30000), Source: "Official gamebook"}
			}
			raw := statCommand(command, change, revision(t, f.s), "stat-preview-proof")
			preview, err := f.s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
			if err != nil || len(preview.Blockers) != 0 {
				t.Fatalf("preview: %+v %v", preview, err)
			}
			if command == commissioner.StatReject {
				requirePreviewValue(t, preview, "Statistic update", item.Words, "None recorded")
			} else {
				requirePreviewValue(t, preview, "score", "1.5 points", "4.5 points")
			}
			proveStorePreview(t, f.s, raw, "creator")
			runScoreCommand(t, f.s, raw, "creator")
		})
	}
}

func requirePreviewValue(t *testing.T, preview commissioner.Preview, label, before, after string) {
	t.Helper()
	for _, f := range preview.Fields {
		if strings.Contains(f.Label, label) && strings.Contains(f.Before, before) && strings.Contains(f.After, after) {
			return
		}
	}
	t.Fatalf("missing %s: %s -> %s: %+v", label, before, after, preview.Fields)
}

func TestPreviewStandingsUsesTheStandingRecord(t *testing.T) {
	f := resultsTest(t)
	raw := standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{
		Season: 2026, Franchise: f.a, Wins: 3}, revision(t, f.s))
	preview, err := f.s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	requirePreviewValue(t, preview, "standings", "1-2", "3-0")
}

func TestPreviewRolloverIsTheDryRunPlanAndWritesNoTables(t *testing.T) {
	f := newRolloverDB(t)
	raw := f.rollover(t, 2027, false, "rollover-preview-proof")
	before := fingerprint(t, f.s)
	preview, err := f.s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	dry := raw
	dry.Args.Rollover = &commissioner.RawRolloverArgs{To: 2027, DryRun: true}
	receipt, err := f.s.ExecuteCommissioner(t.Context(), dry, "creator", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range receipt.Plan.Lines {
		requirePreviewValue(t, preview, "Season rollover", "Season 2026", line)
	}
	if before != fingerprint(t, f.s) {
		t.Fatal("a preview wrote a table")
	}
	proveStorePreview(t, f.s, raw, "creator")
	f.do(t, raw)
}

func TestUnauthorizedPreviewDisclosesNoPrivateValues(t *testing.T) {
	s, _, _ := newStore(t)
	runScoreCommand(t, s, createCommand(), "creator")
	raw := peopleCommand("commish.seat.invite", commissioner.RawPeople{ID: "private-invite",
		Franchise: "franchise.01", Principal: "private-owner"}, 1)
	preview, err := s.PreviewCommissioner(t.Context(), raw, "visitor", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 1 || len(preview.Fields) != 0 {
		t.Fatalf("private preview leak: %+v %v", preview, err)
	}
	_, executeErr := s.ExecuteCommissioner(t.Context(), raw, "visitor", nil)
	var refused *commissioner.Refusal
	if !errors.As(executeErr, &refused) || preview.Blockers[0] != refused.Reason() {
		t.Fatalf("permission differs: %v", executeErr)
	}
}

func TestEveryBuiltScoreAndStandingsCommandHasARealPreview(t *testing.T) {
	f := resultsTest(t)
	for _, tc := range []struct {
		command string
		args    commissioner.RawScoreChange
	}{
		{commissioner.PlayerPosition, commissioner.RawScoreChange{Player: "1001", Position: "LB"}},
		{commissioner.PlayerPositionClear, commissioner.RawScoreChange{Player: "1001"}},
		{commissioner.ScorePlayer, commissioner.RawScoreChange{Player: "1001", Season: 2026, Week: 2, Delta: pointsDelta(5000)}},
		{commissioner.ScoreFranchise, commissioner.RawScoreChange{Franchise: f.a, Season: 2026, Week: 2, Delta: pointsDelta(5000)}},
	} {
		raw := scoreCommand(t, tc.command, tc.args, revision(t, f.s), tc.command)
		proveStorePreview(t, f.s, raw, "creator")
		runScoreCommand(t, f.s, raw, "creator")
	}
	standings := f.standings(t)
	order := []string{}
	for _, row := range standings.Computed {
		order = append(order, row.Franchise)
	}
	for _, tc := range []struct {
		command string
		args    commissioner.RawStandingsChange
	}{
		{commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{Season: 2026, Franchise: f.a, Wins: 3}},
		{commissioner.StandingsOverride, commissioner.RawStandingsChange{Season: 2026, Order: order}},
		{commissioner.StandingsOverrideClear, commissioner.RawStandingsChange{Season: 2026}},
	} {
		raw := standingsCommand(tc.command, tc.args, revision(t, f.s))
		proveStorePreview(t, f.s, raw, "creator")
		runScoreCommand(t, f.s, raw, "creator")
	}
}

func TestBuiltLineupCorrectionHasARealPreview(t *testing.T) {
	f := lineupTest(t)
	grantDeputy(t, f)
	raw := f.correct(t, f.a, f.legal(f.a))
	proveStorePreview(t, f.s, raw, f.cc)
	runScoreCommand(t, f.s, raw, f.cc)
}

func TestEveryBuiltBracketCommandHasARealPreview(t *testing.T) {
	f := bracketTest(t)
	shape := f.bracket(t).Rules.Shape
	for _, tc := range []struct {
		command string
		args    commissioner.RawBracketChange
	}{
		{commissioner.BracketBuild, rulesBuild()},
		{commissioner.BracketPresetSave, change{Name: "saved-shape", Shape: shape}},
		{commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2)}},
		{commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(3)}},
		{commissioner.BracketAdvance, change{Game: "AFC/wildcard_4", Team: franchiseID(8)}},
	} {
		raw := f.bracketCommand(t, tc.command, tc.args, "Set the playoffs")
		proveStorePreview(t, f.s, raw, "creator")
		f.run(t, raw)
	}
}

func TestEveryBuiltWindowScheduleAndContractActionHasARealPreview(t *testing.T) {
	f := resultsTest(t)
	apply := func(command string, args commissioner.RawArgs) {
		t.Helper()
		rev := revision(t, f.s)
		raw := commissioner.RawCommand{League: "native.test", Command: command, Args: args, Expected: rev,
			Request: fmt.Sprintf("operation.%d", rev), Reason: "Record the agreed change", Effective: "now", Visibility: "league"}
		proveStorePreview(t, f.s, raw, "creator")
		runScoreCommand(t, f.s, raw, "creator")
	}
	apply(commissioner.WindowSet, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "preview_window",
		Start: &commissioner.RawDate{Month: 1, Day: 1}, Opens: []leaguerules.Action{"cut"}}})
	apply(commissioner.WindowMove, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "preview_window",
		Start: &commissioner.RawDate{Month: 1, Day: 2}}})
	apply(commissioner.WindowRemove, commissioner.RawArgs{Window: &commissioner.RawWindowArgs{ID: "preview_window"}})
	apply(commissioner.ScheduleGenerate, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}})
	apply(commissioner.SchedulePublish, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}})
	schedule, err := commissioner.ScheduleReading("native.test", scoreRecords(t, f.s))
	if err != nil {
		t.Fatal(err)
	}
	games := schedule.Weeks[0].Games
	games[0].Home, games[0].Away = games[0].Away, games[0].Home
	apply(commissioner.ScheduleEdit, commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Week: 1, Games: games, Published: true}})
	off := false
	for _, action := range []string{"tag", "extension", "restructure", "buyout", "rfa_tender", "resign", "retire", "death", "cap_relief"} {
		apply(commissioner.ContractActionSet, commissioner.RawArgs{ContractAction: &commissioner.RawContractAction{
			Action: action, On: &off, Params: json.RawMessage(`{}`)}})
	}
}

func TestPreviewDoesNotInventScoresForAnUnheldSeason(t *testing.T) {
	f := resultsTest(t)
	for _, raw := range []commissioner.RawCommand{
		scoreCommand(t, commissioner.ScorePlayer, commissioner.RawScoreChange{Player: "1001", Season: 2025,
			Week: 2, Delta: pointsDelta(5000)}, revision(t, f.s), "past-player"),
		scoreCommand(t, commissioner.ScoreFranchise, commissioner.RawScoreChange{Franchise: f.a, Season: 2025,
			Week: 2, Delta: pointsDelta(5000)}, revision(t, f.s), "past-franchise"),
		standingsCommand(commissioner.StandingsRecordAdjust, commissioner.RawStandingsChange{Season: 2025,
			Franchise: f.a, Wins: 3}, revision(t, f.s)),
	} {
		raw.Expected = revision(t, f.s)
		preview, err := f.s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
		if err != nil || len(preview.Blockers) != 0 {
			t.Fatalf("unheld season: %+v %v", preview, err)
		}
		requirePreviewValue(t, preview, "Scoring history", "no scores", "no scores")
		proveStorePreview(t, f.s, raw, "creator")
		runScoreCommand(t, f.s, raw, "creator")
	}
}

func TestBlockedRolloverPreviewKeepsTheRealPlanAndRefusal(t *testing.T) {
	s, c, _ := newStore(t)
	runScoreCommand(t, s, createWith(commissioner.PresetLegacy, 0), "creator")
	runScoreCommand(t, s, heldSeedCommand(t), "creator")
	c.now = time.Date(2027, 3, 1, 5, 0, 0, 0, time.UTC)
	raw := commissioner.RawCommand{League: "native.test", Command: commissioner.SeasonRollover,
		Args: commissioner.RawArgs{Rollover: &commissioner.RawRolloverArgs{To: 2027}}, Expected: 2,
		Request: "blocked-plan", Reason: "Turn the league year", Effective: "now", Visibility: "league"}
	preview, err := s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 1 || len(preview.Fields) < 2 {
		t.Fatalf("blocked plan lost: %+v %v", preview, err)
	}
	_, executeErr := s.ExecuteCommissioner(t.Context(), raw, "creator", nil)
	var refused *commissioner.Refusal
	if !errors.As(executeErr, &refused) || preview.Blockers[0] != refused.Reason() {
		t.Fatalf("rollover refusal differs: %v", executeErr)
	}
	requirePreviewValue(t, preview, "Season rollover", "Season 2026", "Blocked:")
}

func TestRulePreviewCountsAffectedFranchisesWithoutChangingReadingCounts(t *testing.T) {
	s, _, _ := newStore(t)
	runScoreCommand(t, s, createCommand(), "creator")
	raw := ruleCommand("cap.amount", "200", "now", "Agree the salary cap", 1)
	preview, err := s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 || preview.Affects.Words != "All 2 franchises; the league readings and ledger change" {
		t.Fatalf("franchise count missing: %+v %v", preview, err)
	}
	if preview.Affects.Count != nil {
		t.Fatal("an owner count replaced the unknown catalogue-reading count")
	}
	proveStorePreview(t, s, raw, "creator")
	runScoreCommand(t, s, raw, "creator")
}
