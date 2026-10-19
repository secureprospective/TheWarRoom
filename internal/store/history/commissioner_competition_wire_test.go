package history

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type competitionWire struct {
	Request   commissioner.RawCommand       `json:"request"`
	Preview   commissioner.Preview          `json:"preview"`
	Bracket   *commissioner.NativeBracket   `json:"bracket,omitempty"`
	Schedule  *commissioner.ManagedSchedule `json:"schedule,omitempty"`
	Directory commissioner.NativeDirectory  `json:"directory"`
}

func competitionEntry(t *testing.T, s *Store, raw commissioner.RawCommand) competitionWire {
	t.Helper()
	records := scoreRecords(t, s)
	preview, err := s.PreviewCommissioner(t.Context(), raw, "creator", nil, commissioner.PreviewCatalogue{})
	if err != nil || len(preview.Blockers) != 0 {
		t.Fatal("competition preview refused", preview, err)
	}
	directory, err := commissioner.NativeDirectoryReading(raw.League, records, s.now())
	if err != nil {
		t.Fatal(err)
	}
	return competitionWire{Request: raw, Preview: preview, Directory: directory}
}

func TestCommissionerCompetitionWire(t *testing.T) {
	entries := []competitionWire{}
	f := bracketTest(t)
	shape := f.bracket(t).Rules.Shape
	for _, tc := range []struct {
		command string
		args    commissioner.RawBracketChange
	}{
		{commissioner.BracketBuild, rulesBuild()},
		{commissioner.BracketPresetSave, change{Name: "saved-layout", Shape: shape}},
		{commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: franchiseID(2)}},
		{commissioner.BracketSeed, change{Slot: 2, Team: franchiseID(3)}},
		{commissioner.BracketAdvance, change{Game: "AFC/wildcard_4", Team: franchiseID(8)}},
	} {
		raw := f.bracketCommand(t, tc.command, tc.args, "Set the playoffs")
		entry := competitionEntry(t, f.s, raw)
		bracket := f.bracket(t)
		entry.Bracket = &bracket
		entries = append(entries, entry)
		f.run(t, raw)
	}
	schedule := resultsTest(t)
	apply := func(command string, args commissioner.RawScheduleArgs) {
		t.Helper()
		raw := commissioner.RawCommand{League: "native.test", Command: command,
			Args: commissioner.RawArgs{Schedule: &args}, Expected: revision(t, schedule.s),
			Request: fmt.Sprintf("competition.%d", revision(t, schedule.s)), Reason: "Set the schedule",
			Effective: "now", Visibility: "league"}
		entry := competitionEntry(t, schedule.s, raw)
		held, err := commissioner.ManagedScheduleReading(raw.League, "creator", scoreRecords(t, schedule.s))
		if err != nil {
			t.Fatal(err)
		}
		entry.Schedule = &held
		entries = append(entries, entry)
		runScoreCommand(t, schedule.s, raw, "creator")
	}
	apply(commissioner.ScheduleGenerate, commissioner.RawScheduleArgs{Weeks: 3, Pattern: leaguerules.DivisionWeighted})
	held, err := commissioner.ManagedScheduleReading("native.test", "creator", scoreRecords(t, schedule.s))
	if err != nil {
		t.Fatal(err)
	}
	game := held.Draft[0].Games[0]
	game.Home, game.Away = game.Away, game.Home
	apply(commissioner.ScheduleEdit, commissioner.RawScheduleArgs{Week: 1, Games: []commissioner.ScheduleGame{game}})
	apply(commissioner.SchedulePublish, commissioner.RawScheduleArgs{})
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := "../../../frontend/src/app/commissioner/competition.wire.testdata.json"
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	heldBody, err := os.ReadFile(path)
	if err != nil || string(heldBody) != string(body) {
		t.Fatal("competition wire stale", err)
	}
}
