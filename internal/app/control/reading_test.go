package control

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"

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

func TestParameterWords(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ key, description, label, group string }{
		{"model.k_now", "unused", "Games until production and scouting weigh equally", "Production blend · this season"},
		{"model.recency.2", "unused", "Weight of season 2 years back", "Production history"},
		{"model.prior.age.weight", "unused", "Age effect · weight", "Scouting prior"},
		{"unknown", "Unknown setting", "Unknown setting", "Other settings"},
		{"model.prior.unknown", "Unknown term", "Unknown term", "Other settings"},
	} {
		label, group := ParameterWords(tc.key, tc.description)
		if label != tc.label || group != tc.group {
			t.Errorf("%s = %q / %q", tc.key, label, group)
		}
	}
}

func TestBylawsDropMarkupButPreserveSections(t *testing.T) {
	t.Parallel()
	got := BylawSections([]byte("Preamble\n## **One**\n### Detail\n- `First`\n| Key | Value |\n|---|---|\n\n---\n## Two\nParagraph"))
	want := []BylawSection{{Title: "One", Lines: []string{"Detail:", "• First", "Key · Value"}}, {Title: "Two", Lines: []string{"Paragraph"}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bylaws=%+v", got)
	}
}

func TestAlertConstructionAndOrder(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	lock := now.Add(time.Hour)
	got := ByeAlert(lineup.ByeShortfall{Week: 6, Position: domain.PosQB, Available: 0, Required: 1},
		[]lineup.ByePlayer{{Active: true, Position: domain.PosQB, Team: "BUF"}}, []string{""}, map[string]int{"BUF": 6}, now, lock, snapshot.Provenance{})
	if got.Title != "Week 6: no quarterback available" || got.Detail != "0 available; needs 1. Checks position minimums only. On bye: Player name unavailable." || got.At != lock || got.Urgency != AlertUrgency(now, lock) {
		t.Fatal(got)
	}
	if OffersTitle(1) != "1 offer to you" || OffersTitle(2) != "2 offers to you" {
		t.Fatal("offer count wording")
	}
	for _, tc := range []struct{ a, b Alert }{
		{Alert{Urgency: "U1"}, Alert{Urgency: "U0"}},
		{Alert{Kind: "ir", Urgency: "U0"}, Alert{Kind: "bye", Urgency: "U0"}},
		{Alert{Kind: "ir", PlayerID: "0042"}, Alert{Kind: "ir", PlayerID: "0043"}},
	} {
		if CompareAlerts(tc.a, tc.b) >= 0 || CompareAlerts(tc.b, tc.a) <= 0 {
			t.Fatal("alert priority changed", tc)
		}
	}
}
