package main

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/weeklyresults"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
)

// realNow is a 2025 week as the league archive holds it, with that week's NFL schedule, the
// season's byes and MFL's projections for week 5.
func realNow(t *testing.T, week int) nowInput {
	t.Helper()
	schedule, err := nflschedule.Parse([]byte(seasonFixture(t, "nflschedule", "nflSchedule-2025-w05.json")))
	if err != nil {
		t.Fatal(err)
	}
	byes, err := nflschedule.ParseByes([]byte(seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")))
	if err != nil {
		t.Fatal(err)
	}
	projections, err := marketfeeds.Parse(marketfeeds.Projections,
		[]byte(seasonGzFixture(t, "marketfeeds", "projectedScores-2025-w05.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	return nowInput{race: realRace(t, "2025"), week: week, current: 5,
		at:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		held:        nowHeld{schedules: map[int]heldSeasonFeed[[]nflschedule.RawWeek]{5: held([]nflschedule.RawWeek{schedule})}, byes: held(byes)},
		projections: held([]marketfeeds.Feed{projections})}
}

// mflWeek is one week of MFL's weeklyResults export, read straight from the JSON so the reports
// are checked against MFL's own numbers, not the app's parser.
type mflWeek struct {
	Week      string
	Games     [][]mflSide
	Franchise []mflSide
}

type mflSide struct {
	ID     string `json:"id"`
	Score  string `json:"score"`
	Result string `json:"result"`
	Player json.RawMessage
}

type mflPlayer struct {
	ID     string `json:"id"`
	Score  string `json:"score"`
	Status string `json:"status"`
}

// mflList decodes MFL's list-or-single-object field.
func mflList[T any](t *testing.T, raw json.RawMessage) []T {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	var many []T
	if err := json.Unmarshal(raw, &many); err == nil {
		return many
	}
	var one T
	if err := json.Unmarshal(raw, &one); err != nil {
		t.Fatal(err)
	}
	return []T{one}
}

func mflWeeks(t *testing.T, year string) map[int]mflWeek {
	t.Helper()
	var env struct {
		All struct {
			Weeks []struct {
				Week      string          `json:"week"`
				Matchup   json.RawMessage `json:"matchup"`
				Franchise json.RawMessage `json:"franchise"`
			} `json:"weeklyResults"`
		} `json:"allWeeklyResults"`
	}
	if err := json.Unmarshal([]byte(seasonGzFixture(t, "weeklyresults", "weeklyResults-"+year+".json.gz")), &env); err != nil {
		t.Fatal(err)
	}
	out := map[int]mflWeek{}
	for _, w := range env.All.Weeks {
		n, _ := strconv.Atoi(w.Week)
		week := mflWeek{Week: w.Week, Franchise: mflList[mflSide](t, w.Franchise)}
		for _, m := range mflList[struct {
			Franchise []mflSide `json:"franchise"`
		}](t, w.Matchup) {
			week.Games = append(week.Games, m.Franchise)
		}
		out[n] = week
	}
	return out
}

// hundredthsText is MFL's score string to the hundredth, as the reports print it.
func hundredthsText(t *testing.T, score string) string {
	t.Helper()
	f, err := strconv.ParseFloat(score, 64)
	if err != nil {
		t.Fatal(err)
	}
	return scoreText(leaguefeed.Points(math.Round(f * 100)))
}

func nowBoard(t *testing.T, in nowInput) NowBoard {
	t.Helper()
	board, err := buildNow(in)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func nowReport(t *testing.T, board NowBoard, id string) Report {
	t.Helper()
	for _, r := range append(board.Board.Reports, board.Matchup) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no report %s", id)
	return Report{}
}

func franchiseNames(in nowInput) map[string]string {
	names := map[string]string{}
	for _, f := range in.race.config.Franchises {
		names[f.ID] = f.Name
	}
	return names
}

func TestNow2025WeeklyResultsMatchMFLExactly(t *testing.T) {
	t.Parallel()
	mfl := mflWeeks(t, "2025")
	in := realNow(t, 0)
	names := franchiseNames(in)
	for week := 1; week <= 17; week++ {
		in.week = week
		r := nowReport(t, nowBoard(t, in), "M-003")
		if r.Status != "ok" || len(r.Rows) != len(mfl[week].Games) {
			t.Fatalf("week %d: %s %q, %d rows for %d games", week, r.Status, r.Note, len(r.Rows), len(mfl[week].Games))
		}
		want := []string{}
		for _, g := range mfl[week].Games {
			win, lose := g[0], g[1]
			if lose.Result == "W" {
				win, lose = lose, win
			}
			ws, ls := hundredthsText(t, win.Score), hundredthsText(t, lose.Score)
			want = append(want, strings.Join([]string{names[win.ID], ws, names[lose.ID], ls}, "|"))
		}
		got := []string{}
		for _, row := range r.Rows {
			got = append(got, strings.Join(row.Cells[:4], "|"))
		}
		slices.Sort(want)
		slices.Sort(got)
		if !slices.Equal(want, got) {
			t.Fatalf("week %d results differ from MFL:\n%v\n%v", week, want, got)
		}
		plays := slices.ContainsFunc(mfl[week].Games, func(g []mflSide) bool { return g[0].ID == "0001" || g[1].ID == "0001" })
		if r.Rows[0].Mine != plays {
			t.Fatalf("week %d: my game is not first", week)
		}
	}
}

func TestNow2025WeeklySummaryMatchesMFLExactly(t *testing.T) {
	t.Parallel()
	mfl := mflWeeks(t, "2025")
	in := realNow(t, 0)
	names := franchiseNames(in)
	for week := 1; week <= 17; week++ {
		type side struct {
			name   string
			points int64
		}
		sides := []side{}
		var total int64
		var topStarter int64
		closest, biggest := int64(math.MaxInt64), int64(-1)
		for _, g := range mfl[week].Games {
			pts := [2]int64{}
			for i, s := range g {
				f, _ := strconv.ParseFloat(s.Score, 64)
				pts[i] = int64(math.Round(f * 100))
				sides = append(sides, side{names[s.ID], pts[i]})
				total += pts[i]
				for _, p := range mflList[mflPlayer](t, s.Player) {
					f, _ := strconv.ParseFloat(p.Score, 64)
					if p.Status == "starter" {
						topStarter = max(topStarter, int64(math.Round(f*100)))
					}
				}
			}
			margin := max(pts[0]-pts[1], pts[1]-pts[0])
			closest, biggest = min(closest, margin), max(biggest, margin)
		}
		for _, s := range mfl[week].Franchise {
			if s.Score != "" {
				f, _ := strconv.ParseFloat(s.Score, 64)
				sides = append(sides, side{names[s.ID], int64(math.Round(f * 100))})
				total += sides[len(sides)-1].points
			}
			for _, p := range mflList[mflPlayer](t, s.Player) {
				f, _ := strconv.ParseFloat(p.Score, 64)
				if p.Status == "starter" {
					topStarter = max(topStarter, int64(math.Round(f*100)))
				}
			}
		}
		slices.SortStableFunc(sides, func(a, b side) int { return int(b.points - a.points) })
		in.week = week
		r := nowReport(t, nowBoard(t, in), "M-007")
		summary := map[string]string{}
		for _, row := range r.Summary {
			summary[row.Cells[0]] = row.Cells[1]
		}
		n := int64(len(sides))
		checks := map[string]string{
			"High score":     sides[0].name + " · " + scoreText(leaguefeed.Points(sides[0].points)),
			"Low score":      sides[len(sides)-1].name + " · " + scoreText(leaguefeed.Points(sides[len(sides)-1].points)),
			"League average": scoreText(leaguefeed.Points((total + n/2) / n)),
		}
		for k, want := range checks {
			if summary[k] != want {
				t.Fatalf("week %d %s: %q, MFL says %q", week, k, summary[k], want)
			}
		}
		if !strings.HasSuffix(summary["Closest game"], " by "+scoreText(leaguefeed.Points(closest))) ||
			!strings.HasSuffix(summary["Biggest win"], " by "+scoreText(leaguefeed.Points(biggest))) {
			t.Fatalf("week %d margins: %q %q, MFL %d %d", week, summary["Closest game"], summary["Biggest win"], closest, biggest)
		}
		if len(r.Rows) != 10 || r.Rows[0].Cells[3] != scoreText(leaguefeed.Points(topStarter)) {
			t.Fatalf("week %d top starter: %v, MFL %d", week, r.Rows[0].Cells, topStarter)
		}
	}
}

func TestNow2025RecapsAreSentencesFromTheNumbers(t *testing.T) {
	t.Parallel()
	r := nowReport(t, nowBoard(t, realNow(t, 5)), "M-021")
	recaps := map[string]string{}
	for _, row := range r.Rows {
		recaps[row.Cells[0]] = row.Cells[1]
	}
	for game, want := range map[string]string{
		"Buffalo Bills at New York Jets": "Buffalo Bills beat New York Jets 277.03 to 179.98, a margin of 97.05. " +
			"Buffalo Bills' top starter was Fields, Justin (QB) with 31.80. New York Jets' top starter was Sweat, " +
			"Josh (DE) with 27.50. New York Jets left 26.05 on the bench; its best lineup, 206.03, would still have lost.",
		"Indianapolis Colts at Los Angeles Rams": "Indianapolis Colts beat Los Angeles Rams 195.05 to 193.20, a margin " +
			"of 1.85. Indianapolis Colts' top starter was Williams, Javonte (RB) with 31.30. Los Angeles Rams' top " +
			"starter was Stafford, Matthew (QB) with 32.60. Los Angeles Rams left 44.85 on the bench; its best " +
			"lineup, 238.05, would have won.",
		"Jacksonville Jaguars at Arizona Cardinals": "Arizona Cardinals beat Jacksonville Jaguars 281.60 to 270.95, a " +
			"margin of 10.65. Arizona Cardinals' top starter was Stroud, C.J. (QB) with 37.35. Jacksonville Jaguars' " +
			"top starter was Mayfield, Baker (QB) with 34.20. Jacksonville Jaguars started its best lineup.",
	} {
		if recaps[game] != want {
			t.Fatalf("%s:\n got %q\nwant %q", game, recaps[game], want)
		}
	}
	if !r.Rows[0].Mine || r.Rows[0].Cells[0] != "Buffalo Bills at New York Jets" {
		t.Fatal("my game is not first", r.Rows[0].Cells)
	}
	playoff := nowReport(t, nowBoard(t, realNow(t, 15)), "M-021")
	if !strings.HasPrefix(playoff.Rows[0].Cells[1], "Playoff game. Buffalo Bills beat Houston Texans 312.70 to 250.58") {
		t.Fatal(playoff.Rows[0].Cells[1])
	}
}

func TestNowMyMatchupPairsStartersAndTotals(t *testing.T) {
	t.Parallel()
	in := realNow(t, 5)
	r := nowReport(t, nowBoard(t, in), "X-03")
	if strings.Join(r.Columns, "|") != "Slot|Buffalo Bills|Points|New York Jets|Points" {
		t.Fatal(r.Columns)
	}
	if strings.Join(r.Rows[0].Cells, "|") != "QB|Fields, Justin · NYJ|31.80|Rush, Cooper · BAL|3.95" {
		t.Fatal(r.Rows[0].Cells)
	}
	var mine, theirs leaguefeed.Points
	for _, row := range r.Rows[:len(r.Rows)-1] {
		for i, sum := range map[int]*leaguefeed.Points{2: &mine, 4: &theirs} {
			if row.Cells[i] != "" {
				f, _ := strconv.ParseFloat(row.Cells[i], 64)
				*sum += hundredths(f)
			}
		}
	}
	total := r.Rows[len(r.Rows)-1].Cells
	if total[0] != "Total" || total[2] != "277.03" || total[4] != "179.98" ||
		scoreText(mine) != "277.03" || scoreText(theirs) != "179.98" {
		t.Fatal("starter points must add up to MFL's totals", total, mine, theirs)
	}
	in.race.mine = ""
	r = nowReport(t, nowBoard(t, in), "X-03")
	if r.Status != "empty" || r.Note != "Choose your franchise to see your matchup." {
		t.Fatal(r.Status, r.Note)
	}
}

func TestNowWeekIsCheckedInGo(t *testing.T) {
	t.Parallel()
	in := realNow(t, 0)
	if b := nowBoard(t, in); b.Week != 5 || len(b.Weeks) != 17 || b.Weeks[14].Label != "Week 15 · playoffs" {
		t.Fatal("the current week is the default", b.Week, b.Weeks)
	}
	in.current = 0
	if b := nowBoard(t, in); b.Week != 17 {
		t.Fatal("without a current week the latest scored week shows", b.Week)
	}
	for _, week := range []int{-1, 18, 99} {
		in.week = week
		if _, err := buildNow(in); err == nil {
			t.Fatal("accepted week", week)
		}
	}
	in.week, in.race.mine = 0, "9999"
	if _, err := buildNow(in); err == nil {
		t.Fatal("accepted an unknown franchise")
	}
}

func TestNowUnscoredWeekShowsHeldLiveScores(t *testing.T) {
	t.Parallel()
	raw, err := livescoring.Parse([]byte(seasonFixture(t, "livescoring", "liveScoring-w5.json")))
	if err != nil {
		t.Fatal(err)
	}
	lineups, err := livescoring.ToLineups(raw)
	if err != nil {
		t.Fatal(err)
	}
	in := nowInput{race: realRace(t, "2026"), current: 5, reportWeek: 5, at: raceHeldAt}
	in.race.mine = lineups.Matchups[0].Home
	before := nowBoard(t, in)
	if r := nowReport(t, before, "X-03"); r.Rows[len(r.Rows)-1].Cells[2] != "—" {
		t.Fatal("an unscored week without live scores shows no points", r.Rows)
	}
	lineups.Franchises[0].Score = 12.345
	live := held(lineups)
	live.fetchedAt = raceHeldAt.Add(time.Hour)
	in.live = live
	board := nowBoard(t, in)
	r := nowReport(t, board, "X-03")
	if r.AsOf != "As of: Oct 11, 2026, 1:00 PM" || !strings.Contains(r.Note, "Held live scores, not final") {
		t.Fatal(r.AsOf, r.Note)
	}
	// Mine is the home side; my side still reads first, as in the matchup's columns.
	mineName := r.Columns[1] // the matchup's first franchise column is mine
	for _, row := range r.Summary {
		if row.Cells[0] == "Status" && !strings.HasPrefix(row.Cells[1], "In progress: "+mineName+" ") {
			t.Fatal("my side first", row.Cells[1])
		}
	}
	totals := strings.Join(r.Rows[len(r.Rows)-1].Cells, "|")
	if !strings.Contains(totals, "12.35") {
		t.Fatal("live total not shown", totals)
	}
	if res := nowReport(t, board, "M-003"); res.Status != "empty" || !strings.Contains(res.Note, "MFL has not scored week 5 yet.") {
		t.Fatal(res.Status, res.Note)
	}
	in.live.value.Week = 4
	if r := nowReport(t, nowBoard(t, in), "X-03"); strings.Contains(r.Note, "Held live") {
		t.Fatal("another week's live scores must not stand in")
	}
}

func TestNow2025NFLScheduleMarksMyGamesAndByes(t *testing.T) {
	t.Parallel()
	board := nowBoard(t, realNow(t, 5))
	r := nowReport(t, board, "M-015")
	summary := map[string]string{}
	for _, row := range r.Summary {
		summary[row.Cells[0]] = row.Cells[1]
	}
	if summary["On bye"] != "ATL, CHI, GBP, PIT" || summary["My players on bye"] == "" || len(r.Rows) != 14 {
		t.Fatal(summary, len(r.Rows))
	}
	first := strings.Join(r.Rows[0].Cells[:3], "|")
	if first != "Oct 3, 2025, 12:15 AM|SFO at LAR|SFO won 26–23" || !r.Rows[0].Mine {
		t.Fatal(first)
	}
	box := nowReport(t, board, "M-016")
	var started leaguefeed.Points
	seen := map[string]bool{}
	for _, row := range box.Rows {
		if seen[row.Cells[0]+row.Cells[1]] {
			t.Fatal("a player shows twice", row.Cells)
		}
		seen[row.Cells[0]+row.Cells[1]] = true
		if row.Mine && row.Cells[2] == "Started" {
			f, _ := strconv.ParseFloat(row.Cells[3], 64)
			started += hundredths(f)
		}
	}
	if scoreText(started) != "277.03" || box.Rows[0].Group != "SFO at LAR · SFO won 26–23" {
		t.Fatal("my starters must add up to my score", started, box.Rows[0].Group)
	}
	later := nowBoard(t, realNow(t, 6))
	if r := nowReport(t, later, "M-015"); r.Status != "failed" || !strings.Contains(r.Note, "not held yet; refresh") {
		t.Fatal("a week whose schedule is not held must say so", r.Status, r.Note)
	}
}

func TestNow2025PreviewsSumMFLProjections(t *testing.T) {
	t.Parallel()
	var env struct {
		Projected struct {
			Scores []mflPlayer `json:"playerScore"`
		} `json:"projectedScores"`
	}
	if err := json.Unmarshal([]byte(seasonGzFixture(t, "marketfeeds", "projectedScores-2025-w05.json.gz")), &env); err != nil {
		t.Fatal(err)
	}
	projected := map[string]leaguefeed.Points{}
	for _, p := range env.Projected.Scores {
		f, _ := strconv.ParseFloat(p.Score, 64)
		projected[p.ID] = hundredths(f)
	}
	var want leaguefeed.Points
	for _, g := range mflWeeks(t, "2025")[5].Games {
		for _, s := range g {
			for _, p := range mflList[mflPlayer](t, s.Player) {
				if s.ID == "0001" && p.Status == "starter" {
					want += projected[p.ID]
				}
			}
		}
	}
	r := nowReport(t, nowBoard(t, realNow(t, 5)), "M-018")
	if r.Rows[0].Cells[0] != "Buffalo Bills" || r.Rows[0].Cells[1] != scoreText(want) || !r.Rows[0].Mine {
		t.Fatal(r.Rows[0].Cells, scoreText(want))
	}
	other := nowReport(t, nowBoard(t, realNow(t, 4)), "M-018")
	if other.Status != "empty" || other.Note != "MFL projections are held for week 5 only; choose that week." {
		t.Fatal(other.Status, other.Note)
	}
}

func archivedCopies(copies ...struct {
	src  string
	body string
	at   time.Time
}) archivedBodies {
	return func(_ context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		sorted := slices.Clone(copies)
		slices.SortFunc(sorted, func(a, b struct {
			src  string
			body string
			at   time.Time
		}) int {
			return b.at.Compare(a.at)
		})
		for _, c := range sorted {
			if strings.Contains(c.src, part) && accept(c.src, []byte(c.body), c.at) {
				return true, nil
			}
		}
		return false, nil
	}
}

func TestNowStatChangesCompareTheFirstHeldCopy(t *testing.T) {
	t.Parallel()
	body := seasonGzFixture(t, "weeklyresults", "weeklyResults-2025.json.gz")
	ytd := "/2025/export?TYPE=weeklyResults&L=" + "14432" + "&W=YTD"
	day := func(d int) time.Time { return time.Date(2026, 1, d, 12, 0, 0, 0, time.UTC) }
	type copyOf = struct {
		src  string
		body string
		at   time.Time
	}
	bodies := archivedCopies(copyOf{ytd, body, day(2)}, copyOf{ytd, body, day(3)},
		copyOf{"/2025/export?TYPE=weeklyResults&L=14432&W=5", body, day(4)},
		copyOf{"/2024/export?TYPE=weeklyResults&L=14432&W=YTD", body, day(5)})
	firsts, err := readScoreFirsts(context.Background(), scoreFirsts{}, 2025, bodies)
	if err != nil {
		t.Fatal(err)
	}
	if firsts.copies != 2 || !firsts.through.Equal(day(3)) || len(firsts.weeks) != 17 || !firsts.weeks[5].at.Equal(day(2)) {
		t.Fatal("the oldest copy of each scored week is the first", firsts.copies, firsts.through, len(firsts.weeks))
	}
	again, err := readScoreFirsts(context.Background(), firsts, 2025, bodies)
	if err != nil || again.copies != 2 {
		t.Fatal("a second read reads only newer copies", again.copies, err)
	}
	in := realNow(t, 5)
	in.held.firsts = firsts
	if r := nowReport(t, nowBoard(t, in), "M-088"); r.Status != "empty" ||
		!strings.Contains(r.Note, "No score has changed since the first held copy.") {
		t.Fatal(r.Status, r.Note)
	}
	weeks := in.race.held.weeks.value
	for i, w := range weeks {
		if w.Week != 3 {
			continue
		}
		weeks[i].Games = slices.Clone(w.Games)
		g := &weeks[i].Games[0]
		g.Home.Players = slices.Clone(g.Home.Players)
		for j, p := range g.Home.Players {
			if p.Starter {
				g.Home.Players[j].Points += 650
				break
			}
		}
		g.Home.Result, g.Away.Result = g.Away.Result, g.Home.Result
	}
	r := nowReport(t, nowBoard(t, in), "M-088")
	if r.Status != "ok" || len(r.Rows) != 1 || r.Rows[0].Cells[4] != "+6.50" ||
		r.Rows[0].Group != "Week 3 · first held Jan 2, 2026, 12:00 PM" {
		t.Fatal(r.Status, r.Rows)
	}
	flips := 0
	for _, row := range r.Summary {
		if row.Cells[0] == "Week 3 result changed" {
			flips++
		}
	}
	if flips != 2 {
		t.Fatal("both sides of a flipped result are named", r.Summary)
	}
}

func TestNowRowsNeverShowIDs(t *testing.T) {
	t.Parallel()
	franchiseID := regexp.MustCompile(`(^|[^\d.,])00\d\d([^\d.,]|$)`)
	for _, week := range []int{1, 5, 15} {
		in := realNow(t, week)
		ids := map[string]bool{}
		for _, w := range in.race.held.weeks.value {
			for _, s := range allSides(w) {
				for _, p := range s.Players {
					ids[p.ID.String()] = true
				}
			}
		}
		board := nowBoard(t, in)
		for _, r := range append(board.Board.Reports, board.Matchup) {
			for _, row := range append(slices.Clone(r.Rows), r.Summary...) {
				for _, cell := range append(slices.Clone(row.Cells), row.Group) {
					for _, word := range strings.FieldsFunc(cell, func(c rune) bool { return c == ' ' || c == '\n' }) {
						if ids[word] || franchiseID.MatchString(word) {
							t.Fatalf("week %d %s shows an id: %q", week, r.ID, cell)
						}
					}
					if strings.Contains(strings.ToLower(cell), "not held") {
						t.Fatalf("week %d %s: %q", week, r.ID, cell)
					}
				}
			}
		}
	}
}

func TestNowParsesOnlyValidWeeklyResults(t *testing.T) {
	t.Parallel()
	bodies := archivedCopies(struct {
		src  string
		body string
		at   time.Time
	}{"/2025/export?TYPE=weeklyResults&L=14432&W=YTD", `{"allWeeklyResults":{"weeklyResults":[]}}`, raceHeldAt})
	firsts, err := readScoreFirsts(context.Background(), scoreFirsts{}, 2025, bodies)
	if err != nil || firsts.copies != 0 || len(firsts.weeks) != 0 {
		t.Fatal("a body that does not parse is passed over", firsts, err)
	}
	if _, err := weeklyresults.Parse([]byte(`{}`)); err == nil {
		t.Fatal("weeklyresults accepted an empty export")
	}
}

func TestNowBindingReadsHeldAndRefreshReadsTheWeek(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	a.weekMu.Lock()
	a.week.Number = 5
	a.weekMu.Unlock()
	board, err := a.TargetNow(0, 0, "")
	if err != nil || len(transport.requests) != 0 || board.Week != 0 || board.Matchup.Status != "failed" {
		t.Fatal("reading fetched or invented a week", err, transport.requests, board.Week)
	}
	if _, err := a.TargetNow(3, 0, ""); err == nil {
		t.Fatal("accepted a week no held schedule lists")
	}
	if err := a.TargetNowRefresh(0, false); err != nil {
		t.Fatal(err)
	}
	asked := map[string]string{}
	for i, r := range transport.requests {
		asked[r] = transport.weeks[i]
	}
	if asked["nflSchedule"] != "5" || asked["weeklyResults"] != "YTD" {
		t.Fatal("refresh must read the season's results and the current week's NFL schedule", asked)
	}
	if _, ok := asked["nflByeWeeks"]; !ok {
		t.Fatal("refresh must read the bye weeks", asked)
	}
	// The test app's rulebook holds no league, so the held feeds are read with the real 2026 league.
	read := func(week int) NowBoard {
		a.seasonMu.Lock()
		in := nowInput{race: realRace(t, "2026"), held: a.now, current: 5, week: week, at: raceHeldAt}
		in.race.held = a.race
		a.seasonMu.Unlock()
		return nowBoard(t, in)
	}
	board = read(0)
	nfl := nowReport(t, board, "M-015")
	if board.Week != 5 || nfl.Status != "ok" || len(nfl.Rows) != 15 {
		t.Fatal(board.Week, nfl.Status, nfl.Note, len(nfl.Rows))
	}
	requests := len(transport.requests)
	if err := a.TargetNowRefresh(5, false); err != nil || slices.Contains(transport.requests[requests:], "nflSchedule") {
		t.Fatal("a fresh week refetched", err, transport.requests[requests:])
	}
	if err := a.TargetNowRefresh(6, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(6), "M-015"); r.Status != "failed" || !strings.Contains(r.Note, "refresh from MFL") {
		t.Fatal("a schedule for the wrong week must not stand in", r.Status, r.Note)
	}
	week7 := strings.Replace(seasonFixture(t, "nflschedule", "nflSchedule-w5.json"), `"week":"5"`, `"week":"7"`, 1)
	recordSeasonArchiveForm(t, a, "nflSchedule", "7", week7, raceHeldAt, true)
	recordSeasonArchiveForm(t, a, "nflSchedule", "8", week7, raceHeldAt.Add(time.Hour), true)
	transport.failure = "nflSchedule"
	if err := a.TargetNowRefresh(7, false); err != nil {
		t.Fatal(err)
	}
	if r := nowReport(t, read(7), "M-015"); r.Status != "stale" || len(r.Rows) != 15 {
		t.Fatal("an offline week must fall back to its own archived copy", r.Status, r.Note)
	}
	if err := a.TargetNowRefresh(99, false); err == nil {
		t.Fatal("accepted a week the league's schedule does not list")
	}
}

// Just after launch the league's weeks may not be held yet; the current NFL week's schedule is still
// fetched, because it does not depend on them. Another unlisted week is still refused.
func TestNowRefreshFetchesTheCurrentWeekBeforeLeagueWeeksAreHeld(t *testing.T) {
	t.Parallel()
	a, transport, _, _ := seasonTestApp(t)
	transport.bodies["nflSchedule"] = seasonFixture(t, "nflschedule", "nflSchedule-w5.json")
	transport.bodies["nflByeWeeks"] = seasonFixture(t, "nflschedule", "nflByeWeeks-2025.json")
	transport.failure = "weeklyResults"
	a.weekMu.Lock()
	a.week.Number = 5
	a.weekMu.Unlock()
	if err := a.TargetNowRefresh(0, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(transport.requests, "nflSchedule") {
		t.Fatal("the current week's NFL schedule was skipped", transport.requests)
	}
	if err := a.TargetNowRefresh(9, false); err == nil {
		t.Fatal("accepted an unlisted week that is not the current one")
	}
}

func TestNowWireMatchesGo(t *testing.T) {
	t.Parallel()
	board := nowBoard(t, realNow(t, 5))
	board.Matchup.Rows = board.Matchup.Rows[:3]
	for i := range board.Board.Reports {
		r := &board.Board.Reports[i]
		r.Rows = r.Rows[:min(len(r.Rows), 3)]
	}
	body, err := json.MarshalIndent(struct {
		Now NowBoard `json:"now"`
	}{board}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	file := "frontend/src/app/now/wire.testdata.json"
	body = append(body, '\n')
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(file, body, 0o644); err != nil { //nolint:gosec // a committed test fixture
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("now wire stale; TWR_WRITE_WIRE=1 go test -run TestNowWireMatchesGo .")
	}
}
