package commissioner

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestScorePointsWordsAreExactAtBothLimits(t *testing.T) {
	for _, tc := range []struct {
		points leaguerules.Decimal
		want   string
	}{
		{0, "0"}, {1, "0.0001"}, {-1, "-0.0001"}, {12500, "1.25"}, {-12500, "-1.25"},
		{math.MaxInt64, "922337203685477.5807"}, {math.MinInt64, "-922337203685477.5808"},
	} {
		if got := scorePointsWords(tc.points); got != tc.want {
			t.Fatalf("%d formatted %s, want %s", tc.points, got, tc.want)
		}
	}
}

func TestPositionClearCannotCarryAReplacement(t *testing.T) {
	if err := (RawScoreChange{Player: "0099", Position: leaguerules.DT}).Validate(PlayerPositionClear); err == nil {
		t.Fatal("clear accepted a replacement position")
	}
	if err := (RawScoreChange{Player: "0099", Delta: new(leaguerules.Decimal)}).Validate(PlayerPositionClear); err == nil {
		t.Fatal("clear accepted a points payload")
	}
}

func TestScoreChangeScopeDoesNotAcceptInvalidIdentity(t *testing.T) {
	for _, player := range []string{"", "99", " 0099", "gsis.0099"} {
		if _, err := PlayerScoreChanges(nil, player, 2025, 1); err == nil {
			t.Fatalf("invalid player scope: %s", player)
		}
	}
	for _, scope := range []struct{ season, week int }{{1919, 1}, {2025, 0}, {2025, 23}} {
		if _, err := PlayerScoreChanges(nil, "0099", scope.season, scope.week); err == nil {
			t.Fatal("invalid player week")
		}
		if _, err := FranchiseScoreChanges(nil, "franchise.01", scope.season, scope.week); err == nil {
			t.Fatal("invalid franchise week")
		}
	}
}
