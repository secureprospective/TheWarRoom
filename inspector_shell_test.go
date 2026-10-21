package main

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/secureprospective/TheWarRoom/internal/archive"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestInspectorPointsAgreeWithCardHeldWeeks(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	delete(in.held.feeds, researchKey("playerScores", "YTD"))
	v := researchView{researchInput: in}
	card, _, _ := v.onFieldNow()
	p := v.playerReading("16162", nil, "How")
	fact := func(label string) string {
		for _, row := range p.Report.Rows {
			if row.Cells[0] == label {
				return row.Cells[1]
			}
		}
		t.Fatalf("missing %s", label)
		return ""
	}
	if !strings.HasPrefix(fact("Points per game"), strings.TrimSuffix(card["16162"].Text, " PPG")) {
		t.Fatal("card and research disagree", card["16162"], fact("Points per game"))
	}
	sum, games := 0.0, 0
	for w := 1; w < in.week; w++ {
		for _, row := range researchRows(v.feed("playerScores", fmt.Sprint(w))) {
			if row.ID.String() == "16162" {
				sum, games = sum+row.Value, games+1
			}
		}
	}
	if !strings.HasPrefix(fact("Season fantasy points"), fmt.Sprintf("%.2f · %d", sum, games)) {
		t.Fatal("season did not use held finished weeks", fact("Season fantasy points"))
	}
	for _, row := range p.Report.Rows {
		if slices.Contains(row.Cells, "not read yet") {
			t.Fatal("completed reading claimed an in-flight read", row)
		}
	}
}

func TestInspectorOpenReadsDepthThroughFixtureClient(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	a.core.Lookup, a.core.HasLookup, a.core.Week = in.lookup, true, leagueweek.Week{Number: 5}
	a.research.feeds = in.held.feeds
	a.core.FallbackLog = func(string) {}
	var body bytes.Buffer
	if err := parquet.Write(&body, in.held.depth.Value); err != nil {
		t.Fatal(err)
	}
	events := []string{}
	a.core.Fetches = &archive.Transport{Sink: a.core.History,
		Base: researchFixtureTransport{t: t, events: &events, depth: body.Bytes()}}
	p, err := a.TargetResearchPlayer("16162")
	if err != nil {
		t.Fatal(err)
	}
	if a.research.depth.FetchedAt.IsZero() || len(events) != 1 {
		t.Fatal("opening did not read depth", events, a.research.depth)
	}
	for _, row := range p.Report.Rows {
		if row.Cells[0] == "NFL depth" && strings.Contains(row.Cells[2], "not read yet") {
			t.Fatal(row)
		}
	}
	if _, err := a.TargetResearchPlayer("16162"); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatal("fresh depth fetched again", events)
	}
}

func TestNowNamesCarryHeldInspectorIdentities(t *testing.T) {
	t.Parallel()
	in := realNow(t, 5)
	in.race.mine = "0001"
	board, err := buildNow(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Matchup.Entities) == 0 {
		t.Fatal("matchup has no inspectable names")
	}
	for _, r := range append(board.Board.Reports, board.Matchup) {
		for _, row := range r.Rows {
			for _, cell := range append(slices.Clone(row.Cells), row.Group) {
				for _, e := range board.Matchup.Entities {
					if strings.Contains(cell, e.Text) && !slices.Contains(r.Entities, e) {
						t.Fatalf("%s lost identity for %s", r.ID, e.Text)
					}
				}
			}
		}
	}
	if got := researchHeldTime(time.Time{}, "fixture feed failed", time.UTC); got != "fixture feed failed" {
		t.Fatal("missing feed hid the reason", got)
	}
}

func TestInspectorWaitsForHeldArchivePublication(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	a := targetTestApp(t)
	a.core.Lookup, a.core.HasLookup, a.core.Week = in.lookup, true, leagueweek.Week{Number: 5}
	a.core.ResearchArchiveMu.Lock()
	locked := true
	defer func() {
		if locked {
			a.core.ResearchArchiveMu.Unlock()
		}
	}()
	result := make(chan ResearchPlayerReading, 1)
	failed := make(chan error, 1)
	go func() {
		p, err := a.TargetResearchPlayer("16162")
		if err != nil {
			failed <- err
			return
		}
		result <- p
	}()
	select {
	case p := <-result:
		t.Fatalf("Inspector returned before held archive publication: %+v", p.Report.Rows)
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(50 * time.Millisecond):
	}
	a.core.SeasonMu.Lock()
	a.research.feeds = in.held.feeds
	a.core.SeasonMu.Unlock()
	a.core.ResearchArchiveMu.Unlock()
	locked = false
	select {
	case p := <-result:
		if !slices.ContainsFunc(p.Report.Rows, func(row ReportRow) bool {
			return row.Cells[0] == "Points per game" && strings.HasPrefix(row.Cells[1], "32.25")
		}) {
			t.Fatal("published weekly scores were not read", p.Report.Rows)
		}
	case err := <-failed:
		t.Fatal(err)
	case <-time.After(15 * time.Second):
		t.Fatal("Inspector remained blocked after publication")
	}
}

func TestLiveBoxKeepsCrosswalkIdentity(t *testing.T) {
	t.Parallel()
	a := targetTestApp(t)
	v := liveWire(t)
	var name, source string
	for _, box := range v.Followed.Box {
		if len(box.Rows) > 0 {
			name, source = box.Rows[0].Name, box.Rows[0].ID
			break
		}
	}
	if source == "" {
		t.Fatal("validated ESPN athlete identity was discarded")
	}
	if _, err := a.core.History.LinkPlayerIDs(a.core.Ctx, dynastyProcessSource, []history.PlayerIDLink{
		{IDType: "espn", IDValue: source, PlayerID: "13130"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := a.inspectLiveBox(a.core.Ctx, &v); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(v.Box.Entities, ReportEntity{Text: name, Kind: "player", ID: "13130"}) {
		t.Fatal("box score did not expose its held MFL identity", v.Box.Entities)
	}
}

func TestInspectorMissingFeedsGiveCompletedReadReasons(t *testing.T) {
	t.Parallel()
	in := realResearch(t)
	v := researchView{researchInput: researchInput{lookup: in.lookup, zone: time.UTC, week: 5}}
	p := v.playerReading("16162", nil, "How")
	for _, row := range p.Report.Rows[1:] {
		if row.Cells[2] != "No held copy available" || strings.Contains(row.Cells[1], "not read yet") {
			t.Fatal("completed read has no terminal reason", row.Cells)
		}
	}
	if got := researchHeldTime(time.Time{}, "fixture depth failed", time.UTC); got != "fixture depth failed" {
		t.Fatal("failure reason was hidden", got)
	}
}
