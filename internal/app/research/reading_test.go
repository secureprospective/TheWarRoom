package research

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestResearchWeekRunsNameEachDifferingRunOnce(t *testing.T) {
	t.Parallel()
	got := weekRuns([]string{"", "as of A · Archived copy", "as of A · Archived copy", "as of A · Archived copy", "as of B"})
	if got != " Weeks 2–4 · as of A · Archived copy. Week 5 · as of B." {
		t.Fatal(got)
	}
	if weekRuns([]string{"", ""}) != "" {
		t.Fatal("matching weeks named")
	}
}

func TestNewsPostedPrecision(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 13, 5, 48, 37, 0, time.UTC)
	zone := time.FixedZone("CDT", -5*60*60)
	for _, tc := range []struct{ wording, want string }{
		{"26 minutes", "Oct 13, 2026, 12:22 AM"},
		{"3 hours", "Oct 12, 2026, about 9 PM"},
		{"1 day", "Oct 12, 2026"},
		{"", "Posted time not supplied"},
		{"recently", "Posted time unknown: MFL wording not recognized"},
	} {
		t.Run(tc.wording, func(t *testing.T) {
			got, order := newsPosted(tc.wording, at, zone)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			age, known := researchNewsAge(tc.wording)
			if known && order != float64(at.Add(-age).Unix()) || !known && order != 0 {
				t.Fatal("incorrect posted ordering", order)
			}
		})
	}
	if got, order := newsPosted("3 hours", time.Time{}, zone); order != 0 ||
		got != "Posted time unknown: fetch time not held" {
		t.Fatal(got, order)
	}
}

func TestNewsAgeValidation(t *testing.T) {
	for _, word := range []string{"6 days", "15 hours", "1 day", "1 hour", "2 minutes"} {
		if _, ok := researchNewsAge(word); !ok {
			t.Fatal(word)
		}
	}
	for _, word := range []string{"", "-1 day", "1 day ago", "recently", "1.5 days"} {
		if _, ok := researchNewsAge(word); ok {
			t.Fatal(word)
		}
	}
}

func TestShortDepthNames(t *testing.T) {
	if shortDepthNames([]string{"▸ Josh Sweat · MFL identity not linked"}) != "▸ J. Sweat*" {
		t.Fatal("grid name is not short and labelled")
	}
}

func TestConcurrentConstructionLeavesSnapshotUntouched(t *testing.T) {
	in := realResearch(t)
	before, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		after, err := json.Marshal(in)
		if err != nil {
			t.Error(err)
		}
		if !bytes.Equal(before, after) {
			t.Error("construction mutated the detached snapshot")
		}
	})
	for i := 0; i < 4; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			out := Build(in, Scope{Mine: "0001"})
			player := ReadPlayer(in, "16162", nil, "Counting")
			if len(out.Board.Reports) != 10 || player.ID != "16162" || len(player.Report.Rows) == 0 {
				t.Fatal("reading lost held inputs")
			}
		})
	}
}
