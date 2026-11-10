package commissioner

import (
	"errors"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestRecordedStartersExplainTheWeekWithoutChangingIt(t *testing.T) {
	t.Parallel()
	total, points := leaguerules.Decimal(1917000), leaguerules.Decimal(306000)
	week := func() NativeResults {
		return NativeResults{Week: 8, Scores: []FranchiseWeekScore{
			{Franchise: "franchise.01", RecordedResult: "W", Status: "Complete", Points: &total, Starters: []StarterScore{}},
			{Franchise: "franchise.02", Status: "Incomplete", Starters: []StarterScore{}},
		}}
	}
	lineup := WeekLineup{Set: true, Starters: []Starter{{Player: "0001", Name: "Purdy, Brock"}}}
	got := week()
	RecordedStarters(t.Context(), nil, scoreSource{lineup: lineup,
		score: NativePlayerScore{Status: "Complete", Points: &points}}, &got)
	recorded := got.Scores[0]
	if len(recorded.Starters) != 1 || *recorded.Starters[0].Points != points || *recorded.Points != total {
		t.Fatalf("recorded week: %+v", recorded)
	}
	if len(got.Scores[1].Starters) != 0 {
		t.Fatal("an app-scored week was touched")
	}
	for name, src := range map[string]scoreSource{"no lineup": {}, "unreadable": {lineup: lineup, err: errors.New("down")}} {
		got := week()
		RecordedStarters(t.Context(), nil, src, &got)
		if len(got.Scores[0].Starters) != 0 || *got.Scores[0].Points != total {
			t.Fatalf("%s: %+v", name, got.Scores[0])
		}
	}
}
