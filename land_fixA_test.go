package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func TestLandingAAVScale(t *testing.T) {
	r := reportByID(t, buildMarketReports(realMarketHeld(t), nil, 5, time.UTC), "M-069")
	for _, want := range []string{"MFL-wide", "many leagues", "$1,000", "across all", "not this league's dollars", "average marked", "one scale"} {
		if !strings.Contains(r.How, want) {
			t.Fatalf("missing %q: %s", want, r.How)
		}
	}
	if strings.Contains(fmt.Sprint(r), "%") || strings.Contains(fmt.Sprint(r), "budget share") {
		t.Fatal("AAV still presented as percent", r.How)
	}
	if r.Rows[0].Cells[1] != "$30.1 of a $1,000 league pool · $3.4–$113.6" {
		t.Fatal(r.Rows[0].Cells)
	}
	facts := marketFacts(realMarketHeld(t))["16162"]
	if facts[3].Text != "$30.1 of a $1,000 league pool · $3.4–$113.6" {
		t.Fatal(facts[3])
	}
	if got := poolMarketLines(facts); !strings.Contains(fmt.Sprint(got), "AAV $30.1") {
		t.Fatal(got)
	}
}

func TestLandingOfferHeadlines(t *testing.T) {
	d := &lab.Data{Franchises: map[string]lab.RawFranchise{"a": {Name: "Team A"}}}
	for _, tc := range []struct {
		known []int
		want  string
	}{
		{[]int{10, 10}, "No stretch's acceptance rate stands out from the others beyond chance."},
		{[]int{10, 9}, "Too few stretches have enough decided offers to compare."},
		{[]int{0, 0}, "Too few stretches have enough decided offers to compare."},
	} {
		t.Run(fmt.Sprint(tc.known), func(t *testing.T) {
			v := lab.OfferResult{Owner: "a", Offers: []lab.Offer{{}}}
			for _, n := range tc.known {
				v.Stretches = append(v.Stretches, lab.OfferStretch{Known: n})
			}
			if got := offerSummary(d, v, lab.OfferScope{}); !strings.HasSuffix(got, tc.want) {
				t.Fatal(got)
			}
		})
	}
}

func TestLandingFutureReports(t *testing.T) {
	in := nowInput{race: realRace(t, "2026"), current: 5, at: raceHeldAt,
		held: nowHeld{schedules: map[int]heldSeasonFeed[[]nflschedule.RawWeek]{}}}
	in.current, in.reportWeek = 5, 9
	raw, err := nflschedule.Parse([]byte(weekFixture(t, "nflSchedule-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	raw.Week = "9"
	kickoff := in.at.Add(28 * 24 * time.Hour)
	for i := range raw.Matchups {
		raw.Matchups[i].Kickoff = fmt.Sprint(kickoff.Unix())
	}
	in.held.schedules[9] = heldSeasonFeed[[]nflschedule.RawWeek]{value: []nflschedule.RawWeek{raw}, fetchedAt: in.at}
	board, err := buildNow(in)
	if err != nil {
		t.Fatal(err)
	}
	if board.ReportWeek != 9 || len(board.ReportWeeks) != len(in.race.held.weeks.value) {
		t.Fatal(board.ReportWeeks)
	}
	for _, id := range []string{"M-003", "M-007", "M-021", "M-022"} {
		r := nowReport(t, board, id)
		if r.Status != "empty" || len(r.Rows) != 0 || !strings.Contains(r.Note, "Week 9 has not been played yet") ||
			!strings.Contains(r.Note, "kickoff "+kickoff.In(in.race.zone).Format("Mon Jan 2, 3:04 PM")) {
			t.Fatal(id, r)
		}
	}
	in.reportWeek = 99
	if _, err := buildNow(in); err == nil {
		t.Fatal("unlisted week accepted")
	}
}

func TestLandingColdWeekMatchesRefresh(t *testing.T) {
	a, transport, _ := weekTestApp(t, "17", weekFixture(t, "nflSchedule-w5.json"))
	at := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	a.clockNow = func() time.Time { return at }
	recordSeasonArchiveForm(t, a, "nflSchedule", "", weekFixture(t, "nflSchedule.json"), at.Add(-time.Hour), true)
	recordSeasonArchiveForm(t, a, "nflSchedule", "5", weekFixture(t, "nflSchedule-w5.json"), at, true)
	recordSeasonArchive(t, a, "playerScores", "1", string(researchSample(t, "playerScores_W01.json")), at)
	a.seedClockSchedules(a.ctx)
	a.hydrateResearch()
	source := func() string {
		v := researchView{researchInput: researchInput{held: a.research, week: a.marketWeek(), zone: time.UTC}}
		return buildAssetBoard(v, nil, at, nil).NowSource
	}
	before := source()
	if a.marketWeek() != 5 || !strings.Contains(before, "1 of 4 finished weeks held") || len(transport.requests) != 0 {
		t.Fatal(a.marketWeek(), before, transport.requests)
	}
	a.refreshWeek(a.ctx)
	a.hydrateResearch()
	if after := source(); before != after {
		t.Fatal(before, after)
	}
}

func TestLandingRefusedProfileLoggedOnce(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	held := map[string]heldSeasonFeed[[]researchfeeds.News]{}
	lines, asked := []string{}, 0
	fetch := func(context.Context, []string) (map[string][]researchfeeds.News, error) {
		asked++
		_, err := researchfeeds.ParseNews([]byte(`{"error":{"$t":"Error - No Valid Player IDs!"}}`), []string{"0042"})
		if !researchfeeds.InvalidPlayerIDs(err) {
			t.Fatal("fixture not a refusal", err)
		}
		return nil, err
	}
	for range 2 {
		err := readResearchNewsBatches(context.Background(), held, 2026, []string{"0042"}, now, noNewsArchive,
			func(context.Context, time.Duration) error { return nil }, fetch, func(s string) { lines = append(lines, s) })
		if err != nil {
			t.Fatal(err)
		}
	}
	if asked != 1 || len(lines) != 1 || lines[0] != "MFL has no profile for: 0042" ||
		!held["0042"].fetchedAt.Equal(now) || held["0042"].refreshError != "" {
		t.Fatal(asked, lines, held)
	}
}
