package research

import (
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

const usageSeason = 2026

func usageRow(player string, week int, measure string, value float64) UsageFeature {
	return UsageFeature{Player: player, Season: usageSeason, Week: week, Measure: measure, Value: value}
}

func usageTeamRow(player string, week int, team string) UsageFeature {
	return UsageFeature{Player: player, Season: usageSeason, Week: week, Measure: usageTeam, Text: team}
}

func usageSnapshot(week int, teams [2]string, allowed []researchfeeds.Allowed) Snapshot {
	return Snapshot{
		Week: week,
		Held: Values{Feeds: map[string]Feed[[]researchfeeds.Feed]{
			Key("pointsAllowed", ""): {Value: []researchfeeds.Feed{{Allowed: allowed}}},
		}},
		Schedules: map[int]Feed[[]nflschedule.RawWeek]{week: {Value: []nflschedule.RawWeek{{
			Week: strconv.Itoa(week), Matchups: []nflschedule.RawMatchup{{Teams: teams}},
		}}}},
	}
}

func allowedFor(position string, points map[string]float64) []researchfeeds.Allowed {
	rows := make([]researchfeeds.Allowed, 0, len(points))
	for team, value := range points {
		rows = append(rows, researchfeeds.Allowed{Team: team, Position: position, Points: value})
	}
	return rows
}

func near(got *float64, want float64) bool {
	return got != nil && math.Abs(*got-want) < 1e-9
}

func fourWeeks(rows []UsageFeature) []UsageFeature {
	for week := 1; week <= 5; week++ {
		rows = append(rows,
			usageTeamRow("p", week, "BUF"), usageTeamRow("a", week, "BUF"),
			usageRow("p", week, usageOffenseSnap, 0.4+float64(week)*0.1),
			usageRow("p", week, usageTargets, float64(week)),
			usageRow("a", week, usageTargets, 8),
			usageRow("p", week, usageCarries, float64(week)),
			usageRow("a", week, usageCarries, 9))
	}
	return rows
}

func TestReadPlayerUsageWindowAndShares(t *testing.T) {
	t.Parallel()
	in := usageSnapshot(5, [2]string{"BUF", "NYJ"}, allowedFor("TE", map[string]float64{"NYJ": 90, "BUF": 60}))
	rows := fourWeeks(nil)
	got := ReadPlayerUsage(in, "p", "TE", usageSeason, rows)
	if got.WeeksUsed != 4 {
		t.Fatalf("weeks used = %d, want the last four of five", got.WeeksUsed)
	}
	if !near(got.SnapShare, 0.75) {
		t.Fatalf("snap share = %v, want 0.75", got.SnapShare)
	}
	if !near(got.TargetShare, 14.0/46.0) {
		t.Fatalf("target share = %v, want 14/46", got.TargetShare)
	}
	if !near(got.CarryShare, 14.0/50.0) {
		t.Fatalf("carry share = %v, want 14/50", got.CarryShare)
	}
	if got.Opponent != "NYJ" || !near(got.AllowedPoints, 90) || got.AllowedRank == nil || *got.AllowedRank != 1 || got.Label != "Soft" {
		t.Fatalf("matchup = %+v", got)
	}
}

func TestReadPlayerUsageMissingWeeks(t *testing.T) {
	t.Parallel()
	in := usageSnapshot(5, [2]string{"BUF", "NYJ"}, allowedFor("TE", map[string]float64{"NYJ": 90}))
	rows := []UsageFeature{}
	for _, week := range []int{1, 2, 3, 4, 5} {
		rows = append(rows, usageTeamRow("a", week, "BUF"), usageRow("a", week, usageTargets, 8), usageRow("a", week, usageCarries, 9))
	}
	for _, week := range []int{2, 4} {
		rows = append(rows, usageTeamRow("p", week, "BUF"),
			usageRow("p", week, usageOffenseSnap, map[int]float64{2: 0.5, 4: 0.9}[week]),
			usageRow("p", week, usageTargets, map[int]float64{2: 3, 4: 5}[week]),
			usageRow("p", week, usageCarries, map[int]float64{2: 3, 4: 5}[week]))
	}
	got := ReadPlayerUsage(in, "p", "TE", usageSeason, rows)
	if got.WeeksUsed != 2 {
		t.Fatalf("weeks used = %d, want 2", got.WeeksUsed)
	}
	if !near(got.SnapShare, 0.7) || !near(got.TargetShare, 8.0/24.0) || !near(got.CarryShare, 8.0/26.0) {
		t.Fatalf("missing-week shares = %+v", got)
	}
}

func TestReadPlayerUsageZeroTeamTotalsAreNull(t *testing.T) {
	t.Parallel()
	in := usageSnapshot(5, [2]string{"BUF", "NYJ"}, allowedFor("TE", map[string]float64{"NYJ": 90}))
	rows := []UsageFeature{
		usageTeamRow("p", 5, "BUF"), usageTeamRow("a", 5, "BUF"),
		usageRow("p", 5, usageTargets, 0), usageRow("a", 5, usageTargets, 0),
		usageRow("p", 5, usageCarries, 0), usageRow("a", 5, usageCarries, 0),
	}
	got := ReadPlayerUsage(in, "p", "TE", usageSeason, rows)
	if got.TargetShare != nil || got.CarryShare != nil {
		t.Fatalf("zero team totals must stay null: %+v", got)
	}
	if !strings.Contains(got.Reason, "zero") {
		t.Fatalf("reason does not name the zero totals: %q", got.Reason)
	}
}

func TestReadPlayerUsageRankTiesUseTeamCode(t *testing.T) {
	t.Parallel()
	allowed := allowedFor("TE", map[string]float64{"ARI": 100, "BUF": 100, "CIN": 80, "DEN": 60})
	rows := []UsageFeature{usageTeamRow("p", 5, "BUF"), usageRow("p", 5, usageOffenseSnap, 0.5)}
	for _, opponent := range []struct {
		team string
		rank int
	}{{"ARI", 1}, {"BUF", 2}, {"CIN", 3}, {"DEN", 4}} {
		got := ReadPlayerUsage(usageSnapshot(5, [2]string{"BUF", opponent.team}, allowed), "p", "TE", usageSeason, rows)
		if got.AllowedRank == nil || *got.AllowedRank != opponent.rank {
			t.Fatalf("%s rank = %v, want %d", opponent.team, got.AllowedRank, opponent.rank)
		}
	}
}

func TestReadPlayerUsageLabelsFromTeamCount(t *testing.T) {
	t.Parallel()
	allowed := allowedFor("TE", map[string]float64{"ARI": 120, "BUF": 100, "CIN": 90, "DEN": 80, "JAX": 60, "LV": 40})
	rows := []UsageFeature{usageTeamRow("p", 5, "BUF"), usageRow("p", 5, usageOffenseSnap, 0.5)}
	for _, tc := range []struct {
		opponent, label string
		rank            int
	}{
		{"ARI", "Soft", 1}, {"BUF", "Soft", 2}, {"CIN", "Neutral", 3},
		{"DEN", "Neutral", 4}, {"JAX", "Tough", 5}, {"LV", "Tough", 6},
	} {
		got := ReadPlayerUsage(usageSnapshot(5, [2]string{"BUF", tc.opponent}, allowed), "p", "TE", usageSeason, rows)
		if got.Teams != 6 || got.AllowedRank == nil || *got.AllowedRank != tc.rank || got.Label != tc.label {
			t.Fatalf("%s = teams %d rank %v label %q, want 6/%d/%s", tc.opponent, got.Teams, got.AllowedRank, got.Label, tc.rank, tc.label)
		}
	}
}

func TestNextOpponentReadsHeldSchedule(t *testing.T) {
	t.Parallel()
	schedules := map[int]Feed[[]nflschedule.RawWeek]{
		4: {Value: []nflschedule.RawWeek{{Week: "4", Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"BUF", "MIA"}}}}}},
		5: {Value: []nflschedule.RawWeek{{Week: "5", Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"LA", "BUF"}}}}}},
		6: {Value: []nflschedule.RawWeek{{Week: "6", Matchups: []nflschedule.RawMatchup{{Teams: [2]string{"BUF", "ARI"}}}}}},
	}
	if got := nextOpponent(schedules, 5, "BUF"); got != "LAR" {
		t.Fatalf("next opponent = %q, want the current week's LAR", got)
	}
	if got := nextOpponent(schedules, 6, "BUF"); got != "ARI" {
		t.Fatalf("next opponent = %q, want week 6", got)
	}
}
