package history

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// A Teams change made through the rule-change flow reaches every reading of the franchises.
func TestTeamsChangeReachesEveryReading(t *testing.T) {
	t.Parallel()
	s, _ := statStore(t, filepath.Join(t.TempDir(), "teams.db"))
	create := createWith(commissioner.PresetLegacy, 12)
	create.Args.Source = commissioner.SourceNative
	if _, err := s.ExecuteCommissionerWithFacts(t.Context(), create, "creator", nil, nil, time.UTC); err != nil {
		t.Fatal(err)
	}
	teams := func(t *testing.T, s *Store, want int) {
		t.Helper()
		directory, err := s.NativeDirectory(t.Context(), bracketLeague)
		if err != nil || len(directory.Franchises) != want || directory.Franchises[want-1].Franchise != franchiseID(want) {
			t.Fatalf("directory reads %d teams, want %d: %v", len(directory.Franchises), want, err)
		}
		point, err := s.RosterRestorePoint(t.Context(), bracketLeague)
		if err != nil || len(point.Snapshot.Franchises) != want {
			t.Fatalf("restore point holds %d teams, want %d: %v", len(point.Snapshot.Franchises), want, err)
		}
	}
	runScoreCommand(t, s, ruleCommand("league.teams", "14", "now", "Two expansion teams", 2), "creator")
	teams(t, s, 14)
	runScoreCommand(t, s, ruleCommand("league.teams", "12", "now", "The expansion fell through", 3), "creator")
	teams(t, s, 12)
	runScoreCommand(t, s, ruleCommand("league.teams", "14", "now", "Two expansion teams after all", 4), "creator")
	teams(t, s, 14)

	point, err := s.RosterRestorePoint(t.Context(), bracketLeague)
	if err != nil {
		t.Fatal(err)
	}
	target, _, _ := newStore(t)
	if _, err := target.StandUpRosters(t.Context(), point); err != nil {
		t.Fatal(err)
	}
	if recovered, err := target.RosterRestorePoint(t.Context(), bracketLeague); err != nil ||
		!reflect.DeepEqual(recovered.Snapshot, point.Snapshot) {
		t.Fatalf("restore round trip: %v", err)
	}
	teams(t, target, 14)

	runScoreCommand(t, s, ruleCommand("league.divisions", "2", "now", "Two divisions of seven", 5), "creator")
	directory, err := s.NativeDirectory(t.Context(), bracketLeague)
	if err != nil || directory.Franchises[6].Division != "Division 1" || directory.Franchises[7].Division != "Division 2" {
		t.Fatalf("divisions of seven: %+v %v", directory.Franchises, err)
	}
	runScoreCommand(t, s, ruleCommand("schedule.pattern", `"division_weighted"`, "now", "Choose the pattern", 6), "creator")
	runScoreCommand(t, s, ruleCommand("schedule.regularWeeks", "1", "now", "Choose the season length", 7), "creator")
	raw := commissioner.RawCommand{League: bracketLeague, Command: commissioner.ScheduleGenerate,
		Args: commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{Weeks: 3}}, Expected: 8, Request: "generate",
		Effective: "now", Visibility: "league"}
	runScoreCommand(t, s, raw, "creator")
	raw.Command, raw.Expected, raw.Request = commissioner.SchedulePublish, 9, "publish"
	raw.Args = commissioner.RawArgs{Schedule: &commissioner.RawScheduleArgs{}}
	runScoreCommand(t, s, raw, "creator")
	schedule, err := commissioner.ScheduleReading(bracketLeague, scoreRecords(t, s))
	if err != nil || len(schedule.Weeks) == 0 || len(schedule.Weeks[0].Games) != 7 {
		t.Fatalf("a published week of 14 teams holds 7 games: %+v %v", schedule.Weeks, err)
	}
	s.SetNativeLineups(&resultLineups{})
	standings, err := s.NativeStandings(t.Context(), bracketLeague, 2026)
	if err != nil || len(standings.Computed) != 14 {
		t.Fatalf("standings rank %d teams, want 14: %v (%s)", len(standings.Computed), err, standings.Words)
	}

	_, err = s.ExecuteCommissioner(t.Context(), ruleCommand("league.teams", "12", "now", "Too late", 10), "creator", nil)
	var refusal *commissioner.Refusal
	if !errors.As(err, &refusal) || !strings.Contains(err.Error(), "teams can't change once the season has games") {
		t.Fatalf("a scheduled season changed teams: %v", err)
	}
}
