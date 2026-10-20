package control

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
)

func TestScoringTextReadsMFLPointsAndRanges(t *testing.T) {
	t.Parallel()
	for rule, want := range map[league.ScoringRule]string{
		{Points: "6", Range: "0-99"}:       "6 points (when it is 0 to 99)",
		{Points: "3", Range: "0-39"}:       "3 points (when it is 0 to 39)",
		{Points: "-1", Range: "30-99"}:     "-1 point (when it is 30 to 99)",
		{Points: "1", Range: "-5--1"}:      "1 point (when it is -5 to -1)",
		{Points: "*.05", Range: "-50-999"}: "0.05 points for each",
		{Points: "*-.5", Range: "0-10"}:    "-0.5 points for each",
		{Points: "*4.5", Range: "0-25"}:    "4.5 points for each",
		{Points: "*1", Range: "0-10"}:      "1 point for each",
		{Points: "2", Range: ""}:           "2 points",
	} {
		if got := scoringText(rule); got != want {
			t.Errorf("%+v = %q, want %q", rule, got, want)
		}
	}
	if !scoresNothing(league.ScoringRule{Points: "*0"}) || scoresNothing(league.ScoringRule{Points: "*.05"}) {
		t.Fatal("a zero-point rule says nothing; any other rule does")
	}
}
