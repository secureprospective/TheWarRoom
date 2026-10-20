package counterparties

import (
	"fmt"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"encoding/json"

	"regexp"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func ptrFloat(v float64) *float64 { return &v }

func TestTargetCounterpartiesReading(t *testing.T) {
	t.Parallel()
	a := testService()
	r := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	if r.Status != report.ReadingOK || r.Count < 40 || r.Threshold != 40 {
		t.Fatalf("missing Counterparties reading: status %s count %d", r.Status, r.Count)
	}
	v := r.Value
	if len(v.Picks) != 3 || len(v.Teams) != 32 || len(v.Cards) != 3 || len(v.Pairs) != 9 || len(v.Rows) != 32 {
		t.Fatalf("surface shape: picks %d teams %d cards %d pairs %d rows %d", len(v.Picks), len(v.Teams), len(v.Cards), len(v.Pairs), len(v.Rows))
	}
	if !strings.Contains(v.AsOf, "archive cutoff 2026-10-06") || !strings.Contains(v.AsOf, "not today's live standings") {
		t.Fatalf("as-of wording must name the cutoff and deny live standings: %q", v.AsOf)
	}
	if !strings.Contains(r.HowCounted, "archive cutoff 2026-10-06") {
		t.Fatal("how counted must name the archive cutoff")
	}
	if r.InputSHA != lab.DataSHA || r.BuildStamp != "dev " || r.Cutoff != "2026-10-06T01:42:57+00:00" {
		t.Fatal("missing provenance")
	}
	if r.Provenance.Freshness.State == "live" || !strings.Contains(r.Provenance.Source, r.InputSHA) {
		t.Fatal("archive labelled live")
	}
	if v.Headline == "" || v.Summary == "" || len(v.TrendSay) == 0 || len(v.Trends.Tiers) != 3 {
		t.Fatal("missing headline, summary or trends")
	}
	for _, card := range v.Cards {
		if card.Thin && len(card.Rows) != 0 {
			t.Fatalf("thin card %s carries estimates", card.ID)
		}
		if !card.Thin && len(card.Posture) != 3 || card.PerSeasonTxt == "" {
			t.Fatalf("card %s missing posture or per-season text", card.ID)
		}
	}
	for _, card := range v.Cards {
		if !card.Thin && (len(card.Rows) < 2 || card.Rows[0].Range == nil || card.Rows[1].Range == nil) {
			t.Fatalf("card %s lost its supplied ranges", card.ID)
		}
	}
	for _, row := range v.Rows {
		if row.Name == "" || row.StandingTitle == "" {
			t.Fatalf("row %s missing name or standing title", row.Fid)
		}
		if row.Partner != nil && (row.PartnerName == "" || row.PartnerTrades == "") {
			t.Fatalf("row %s partner without name or trades text", row.Fid)
		}
	}
}

func TestCounterpartiesWireAndWording(t *testing.T) {
	t.Parallel()
	a := testService()
	r := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(b, &wire); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"value", "provenance", "status", "absenceReason", "count", "threshold", "baseline", "cutoff", "method", "howCounted", "inputSHA", "buildStamp"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	// Raw tone codes never reach the screen: no caption carries "· up" style enum tails.
	leaks := regexp.MustCompile(`· (up|down|even|swap|now|later|contender|hunt|rebuild)\b`)
	if where := leaks.FindString(string(b)); where != "" {
		t.Errorf("tone enum leaked into display text: %q", where)
	}
	for tone, want := range map[string]string{
		"up":   "above the league's share beyond chance",
		"down": "below the league's share beyond chance",
		"even": "within chance of the league's share",
	} {
		if testService().shareWords(lab.TeamShare{Tone: tone}) != want {
			t.Errorf("share tone %s rendered %q, want %q", tone, testService().shareWords(lab.TeamShare{Tone: tone}), want)
		}
	}

}

func TestCounterpartiesStates(t *testing.T) {
	t.Parallel()
	a := testService()
	thin := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2017, Tiers: []string{"hunt"}, Windows: []string{"summer"}, Postures: []string{"later"}})
	if thin.Status != report.ReadingThin || thin.AbsenceReason == nil || thin.Count >= 40 {
		t.Fatalf("narrow scope should read thin, got %s count %d", thin.Status, thin.Count)
	}
	if len(thin.Value.Cards) != 0 || len(thin.Value.Rows) != 0 {
		t.Fatal("thin reading fabricated analytics")
	}
	if len(thin.Value.Picks) != 3 || len(thin.Value.Teams) != 32 {
		t.Fatal("thin reading lost its picker")
	}
	for _, raw := range []RawCounterpartiesScope{
		{From: 2026, To: 2017},
		{From: 2017, To: 2026, Tiers: []string{"bogus"}},
		{From: 2017, To: 2026, Windows: []string{"spring", "spring"}},
		{From: 2017, To: 2026, Postures: []string{"whenever"}},
		{From: 2017, To: 2026, Team: "0099"},
	} {
		failed := a.TargetCounterparties(raw)
		if failed.Status != report.ReadingFailed || failed.AbsenceReason == nil {
			t.Fatalf("invalid IPC scope accepted: %+v -> %s", raw, failed.Status)
		}
	}
}

func TestCounterpartTeamPanel(t *testing.T) {
	t.Parallel()
	a := testService()
	r := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0001"})
	if r.Status != report.ReadingOK || r.Count == 0 || r.Threshold != 1 {
		t.Fatalf("missing team panel: status %s count %d", r.Status, r.Count)
	}
	v := r.Value
	if v.Name == "" || v.DayTxt == "" || v.SeasonSay == "" || len(v.Seasons) == 0 || len(v.Kinds) == 0 || len(v.Partners) == 0 || len(v.AsTier) != 3 {
		t.Fatalf("panel shape wrong: %+v", v)
	}
	if len(v.Partners) > 12 {
		t.Fatalf("partners not capped: %d", len(v.Partners))
	}
	for _, s := range v.Seasons {
		if s.Flagged && (s.Against < 2 || float64(s.Against)/float64(s.N) < 0.3) {
			t.Fatalf("season %d flagged without the rule holding", s.Season)
		}
	}
	flagged := 0
	for _, option := range a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026}).Value.Teams {
		p := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: option.Fid})
		for _, s := range p.Value.Seasons {
			if s.Flagged {
				flagged++
			}
		}
		if flagged > 0 {
			break
		}
	}
	if flagged == 0 {
		t.Fatal("no team ever traded against its standing in the frozen facts; the L-27 mark could not be exercised")
	}
	empty := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2017, Windows: []string{"summer"}, Postures: []string{"later"}}, Fid: "0002"})
	if empty.Status != report.ReadingThin || !empty.Value.Empty || empty.Value.Note == "" {
		t.Fatalf("empty team panel must be honest: %s empty %t", empty.Status, empty.Value.Empty)
	}
	unknown := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0099"})
	if unknown.Status != report.ReadingFailed || unknown.AbsenceReason == nil {
		t.Fatal("unknown franchise accepted")
	}
}

func TestCounterpartyDisplayThresholds(t *testing.T) {
	t.Parallel()
	rows := []lab.TierTrendRow{{N: 9}, {N: 10}}
	view := testService().counterTrends(lab.TeamTrends{Tiers: []lab.TierTrend{{Tier: "hunt", Rows: rows}}})
	if view.Tiers[0].Rows[0].ShowShares || !view.Tiers[0].Rows[1].ShowShares {
		t.Fatal("season share threshold must be 10, as in the reference display")
	}
	panel := testService().TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0001"})
	peak := 0.01
	for _, kind := range panel.Value.Kinds {
		peak = max(peak, kind.InW, kind.OutW)
	}
	if panel.Value.FlowScale != peak {
		t.Fatal("all asset-flow bars must share the same Go-supplied scale")
	}
}

func TestCounterCountFormatting(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		n    int
		want string
	}{
		{0, "0"}, {999, "999"}, {1000, "1,000"}, {3810, "3,810"}, {1000000, "1,000,000"}, {-3810, "-3,810"},
	} {
		if got := testService().CountNumber(c.n); got != c.want {
			t.Errorf("testService().CountNumber(%d)=%s, want %s", c.n, got, c.want)
		}
	}
	if got := testService().counterShare(lab.TeamShare{K: 1000, N: 3810}).CountText; got != "1,000 of 3,810" {
		t.Fatal(got)
	}
	if got := testService().CountLabel(3810, "trade side"); got != "3,810 trade sides" {
		t.Fatal(got)
	}
	a := testService()
	r := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	if !strings.Contains(r.Value.Summary, "3,810 trade sides") || r.Value.CountText != "3,810" {
		t.Fatal("unformatted archive count")
	}
	for _, row := range r.Value.Rows {
		if row.NText != testService().CountNumber(row.N) || row.PartnersText != testService().CountNumber(row.PartnersCount) {
			t.Fatal("unformatted team counts")
		}
	}
	for _, tier := range r.Value.Trends.Tiers {
		for _, row := range tier.Rows {
			if row.NText != testService().CountNumber(row.N) {
				t.Fatal("unformatted trend count")
			}
		}
	}
}
func TestCounterRoundedZero(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		v    float64
		dp   int
		want string
	}{
		{-0.0001, 2, "0.00"}, {-0.0049, 2, "0.00"}, {-0.0051, 2, "-0.01"}, {-0.04, 1, "0.0"}, {0, 2, "0.00"}, {0.0049, 2, "0.00"}, {0.0051, 2, "+0.01"},
	} {
		if got := testService().wins(c.v, c.dp); got != c.want {
			t.Errorf("testService().wins(%v,%d)=%s, want %s", c.v, c.dp, got, c.want)
		}
	}
	e := lab.TeamEdge{Mean: ptrFloat(-0.0001), Lo: ptrFloat(-0.1), Hi: ptrFloat(0.1), Tone: "even"}
	if got := testService().edgeWords(e, "wins"); got != "0.00 wins, within chance of zero" {
		t.Fatal("negative zero in prose")
	}
}
func TestCounterVisibleWording(t *testing.T) {
	t.Parallel()
	w := testService().counterWords()
	if w.Day != "On the day" || w.Since != "Since" || w.Up != "↑" || w.Down != "↓" {
		t.Fatal("bar labels or non-colour marks missing")
	}
	for _, word := range []string{"red", "below", "beyond chance", "Unmarked", "losses", "gains"} {
		if !strings.Contains(w.Legend, word) {
			t.Errorf("legend missing %s", word)
		}
	}
	for _, tier := range lab.TeamTiers() {
		if testService().tierOne(tier) != testService().tierLabel(tier) {
			t.Fatal("tier case drift")
		}
	}
	a := testService()
	panel := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0001"}).Value
	if panel.Words != w {
		t.Fatal("panel wording drift")
	}
	for _, cell := range panel.Seasons {
		if cell.MostlyText != "" && cell.MostlyText != "Contenders" && cell.MostlyText != "In the hunt" && cell.MostlyText != "Rebuilding" {
			t.Fatal(cell.MostlyText)
		}
	}
	for _, tier := range panel.AsTier {
		if !strings.Contains(tier.Label, testService().tierLabel(tier.Tier)) {
			t.Fatal(tier.Label)
		}
	}
}

func TestCounterMatrixPhrases(t *testing.T) {
	t.Parallel()
	picks := testService().tierPicks(nil, 1)
	for i, want := range []string{"With contenders", "With teams in the hunt", "With rebuilding teams"} {
		if picks[i].With != want || picks[i].TradingAs != testService().tierLabel(picks[i].ID) {
			t.Fatalf("matrix phrase: %+v, want %q", picks[i], want)
		}
	}
}
func TestCounterTableTotals(t *testing.T) {
	t.Parallel()
	e := lab.TeamEdge{Total: ptrFloat(4.59), Mean: ptrFloat(0.05), Lo: ptrFloat(-0.02), Hi: ptrFloat(0.12)}
	if got := testService().counterTotalWords(e); got != "+4.59 total · +0.05 a trade (-0.02 to +0.12)" {
		t.Fatal(got)
	}
	if testService().counterTotalWords(lab.TeamEdge{}) != "—" {
		t.Fatal("absent total fabricated")
	}
	for _, row := range testService().TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026}).Value.Rows {
		if row.DayText != testService().counterTotalWords(row.Day) || row.SinceText != testService().counterTotalWords(row.Since) {
			t.Fatal("row total text missing")
		}
	}
}

func TestCounterNarrowHeadingsAndPanelPartners(t *testing.T) {
	t.Parallel()
	words := testService().counterWords()
	if words.Sides != "Sides" || words.Now != "For now" || words.Later != "For later" || words.Day != "On the day" || words.Since != "Since" {
		t.Fatalf("unexpected compact headings: %+v", words)
	}
	a := testService()
	surface := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	for _, row := range surface.Value.Rows {
		panel := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: row.Fid})
		if panel.Value.PartnersText != row.PartnersText || panel.Value.Behaviour.PerSeason != row.PerSeason {
			t.Fatalf("panel omitted row values for %s", row.Fid)
		}
		if row.Partner != nil && (len(panel.Value.Partners) == 0 || panel.Value.Partners[0].Fid != row.Partner.Fid) {
			t.Fatalf("panel most frequent partner differs for %s", row.Fid)
		}
	}
}

// TestTierPicksParity recounts the per-tier "trades a season" figure from the Lab data:
// one team's sides per tier, over the seasons in which the league traded at all.
func TestTierPicksParity(t *testing.T) {
	t.Parallel()
	d, err := lab.Load()
	if err != nil {
		t.Fatal(err)
	}
	scope := RawCounterpartiesScope{From: 2023, To: 2024, Team: "0001"}
	tiers, seasons := map[string]int{}, map[int]bool{}
	for _, s := range lab.SidesInScope(d, lab.TeamScope{From: 2023, To: 2024}) {
		seasons[s.Season] = true
		if s.Fid == scope.Team {
			tiers[s.Tier]++
		}
	}
	want := map[string]float64{"contender": 15.5, "hunt": 0, "rebuild": 26}
	picks := testService().counterpartiesReading(d, testService().counterScope(scope)).Value.Picks
	if len(picks) != 3 || len(seasons) != 2 {
		t.Fatalf("scope shape: %d picks over %d seasons", len(picks), len(seasons))
	}
	for _, p := range picks {
		independent := float64(tiers[p.ID]) / float64(len(seasons))
		if p.Count != independent || p.Count != want[p.ID] {
			t.Fatalf("%s trades a season = %v, recounted %v, pinned %v", p.ID, p.Count, independent, want[p.ID])
		}
	}
}

func TestFilteredPostureIsNotAChanceFinding(t *testing.T) {
	t.Parallel()
	a := testService()
	for _, selected := range [][]string{{"now"}, {"later"}, {"swap"}, {"now", "later"}} {
		raw := RawCounterpartiesScope{From: 2017, To: 2026, Postures: selected}
		r := a.TargetCounterparties(raw)
		v := r.Value
		if r.Status != report.ReadingOK || !strings.Contains(v.Summary, "Showing only trades that") ||
			strings.Contains(v.Headline, "buy for now in") || strings.Contains(v.Words.Legend, "↑") ||
			len(v.TrendSay) != 0 {
			t.Fatalf("filter presented as a finding: %+v", v)
		}
		for _, card := range v.Cards {
			for _, cell := range card.Posture {
				if cell.Tone != "" || strings.Contains(cell.Title, "chance") {
					t.Fatal("posture card marked")
				}
			}
		}
		for _, row := range v.Rows {
			if row.ForNow.Tone != "" || row.ForLater.Tone != "" {
				t.Fatal("team shares marked")
			}
			if row.DayText == "" {
				t.Fatal("on-the-day result removed")
			}
		}
		for _, tier := range v.Trends.Tiers {
			for _, row := range tier.Rows {
				if row.Now.Tone != "" || row.Later.Tone != "" {
					t.Fatal("trend shares marked")
				}
			}
		}
		panel := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: raw, Fid: "0014"}).Value
		if !strings.Contains(panel.AsOf, "Showing only trades that") {
			t.Fatal("Inspector scope missing")
		}
		for _, cell := range panel.Posture {
			if cell.Tone != "" {
				t.Fatal("Inspector posture marked")
			}
		}
	}
	thin := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2017, Postures: []string{"later"}, Windows: []string{"summer"}})
	if !strings.Contains(thin.Value.Summary, "Showing only trades that") {
		t.Fatal("thin scope lost filter words")
	}
}

func TestExcludedTiersAreNotThin(t *testing.T) {
	t.Parallel()
	a := testService()
	raw := RawCounterpartiesScope{From: 2017, To: 2026, Tiers: []string{"contender"}}
	r := a.TargetCounterparties(raw)
	for _, card := range r.Value.Cards {
		excluded := card.ID != "contender"
		if card.Excluded != excluded || excluded && card.Thin {
			t.Fatalf("excluded card: %+v", card)
		}
	}
	if len(r.Value.Trends.Tiers) != 1 {
		t.Fatal("excluded trend columns remain")
	}
	for _, pair := range r.Value.Pairs {
		if pair.Excluded != (pair.tier != "contender") || pair.Excluded && pair.Trades != "Not in the current filter" {
			t.Fatalf("standing against standing, excluded row: %+v", pair)
		}
	}
	panel := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: raw, Fid: "0014"})
	for _, tier := range panel.Value.AsTier {
		if tier.Tier != "contender" && (!tier.Excluded || tier.Text != "Not in the current filter") {
			t.Fatalf("Inspector excluded tier: %+v", tier)
		}
	}
	sides := []lab.TeamSide{{Tier: "contender", Window: "spring"}, {Tier: "hunt", Window: "summer"}}
	s := lab.TeamScope{Windows: []string{"spring"}}
	shown := sides[:1]
	if !testService().tierExcluded(s, "hunt", shown, sides) || testService().tierExcluded(s, "contender", shown, sides) ||
		testService().tierExcluded(lab.TeamScope{}, "hunt", nil, sides) {
		t.Fatal("window exclusion confused with thin history")
	}
	empty := testService().emptyTeamReading(report.Reading[TeamCounterpart]{}, TeamCounterpart{Fid: "0014"}, s,
		[]lab.TeamSide{{Fid: "0014", Tier: "hunt", Window: "summer"}})
	if empty.Status != report.ReadingOK || !empty.Value.Empty || empty.Value.Note != "Not in the current filter" || empty.AbsenceReason != nil {
		t.Fatal("filtered-out Inspector still claims thin history")
	}
	if !testService().teamExcluded(s, nil, sides) || testService().teamExcluded(lab.TeamScope{}, nil, sides) {
		t.Fatal("empty filtered Inspector confused with thin history")
	}
}

func testService() Service {
	return Service{Load: lab.Load, BuildStamp: "dev ", CountNumber: testCountNumber, CountLabel: func(n int, noun string) string {
		suffix := "s"
		if n == 1 {
			suffix = ""
		}
		return fmt.Sprintf("%s %s%s", testCountNumber(n), noun, suffix)
	}, Percent: func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }, MarketName: testMarketName}
}

func testCountNumber(n int) string {
	text := strconv.Itoa(n)
	start := 0
	if n < 0 {
		start = 1
	}
	for i := len(text) - 3; i > start; i -= 3 {
		text = text[:i] + "," + text[i:]
	}
	return text
}

func testMarketName(kind string) string {
	names := map[string]string{"QB": "Quarterbacks", "RB": "Running backs", "WR": "Wide receivers", "TE": "Tight ends", "PK": "Kickers", "DL": "Defensive linemen", "LB": "Linebackers", "DB": "Defensive backs"}
	parts := strings.Split(kind, ":")
	if strings.HasPrefix(parts[0], "pick") && len(parts) == 2 {
		round := map[string]string{"pick1": "1sts", "pick2": "2nds", "pick3": "3rds and later"}[parts[0]]
		return strings.ToUpper(parts[1][:1]) + parts[1][1:] + "-draft " + round
	}
	label := names[parts[0]]
	if len(parts) == 2 {
		label += ", " + map[string]string{"young": "24 or younger", "prime": "25 to 27", "older": "28 or older"}[parts[1]]
	}
	return label
}

func TestServiceReadFailureAndValidationOrder(t *testing.T) {
	t.Parallel()
	svc := testService()
	calls := 0
	svc.Load = func() (*lab.Data, error) {
		calls++
		return nil, fmt.Errorf("load lab facts: unavailable")
	}
	bad := RawCounterpartiesScope{From: 2026, To: 2017}
	svc.TargetCounterparties(bad)
	svc.TargetCounterpartTeam(RawCounterpartTeam{Scope: bad})
	if calls != 0 {
		t.Fatal("invalid scope reached the reader")
	}
	scope := RawCounterpartiesScope{From: 2017, To: 2026}
	board := svc.TargetCounterparties(scope)
	team := svc.TargetCounterpartTeam(RawCounterpartTeam{Scope: scope, Fid: "0001"})
	if calls != 2 || board.Status != report.ReadingFailed || team.Status != report.ReadingFailed {
		t.Fatal("reader failure did not produce failed readings")
	}
	if board.AbsenceReason == nil || team.AbsenceReason == nil || *board.AbsenceReason != "load lab facts: unavailable" || *team.AbsenceReason != *board.AbsenceReason {
		t.Fatal("reader error text changed")
	}
	if board.Threshold != 40 || team.Threshold != 1 {
		t.Fatal("failure thresholds changed")
	}
}
