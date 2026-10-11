package main

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/lab"
)

func ptrFloat(v float64) *float64 { return &v }

func TestTargetCounterpartiesReading(t *testing.T) {
	a := &App{}
	r := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	if r.Status != ReadingOK || r.Count < 40 || r.Threshold != 40 {
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
	if r.InputSHA != lab.DataSHA || r.BuildStamp != buildLabel()+" "+buildDate || r.Cutoff != "2026-10-06T01:42:57+00:00" {
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
	a := &App{}
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
		if shareWords(lab.TeamShare{Tone: tone}) != want {
			t.Errorf("share tone %s rendered %q, want %q", tone, shareWords(lab.TeamShare{Tone: tone}), want)
		}
	}
	for tone, want := range map[string]string{
		"up":   "above its slot beyond chance",
		"down": "below its slot beyond chance",
		"even": "within chance of its slot",
	} {
		if slotTone(tone) != want {
			t.Errorf("slot tone %s rendered %q", tone, slotTone(tone))
		}
	}
}

func TestCounterpartiesStates(t *testing.T) {
	a := &App{}
	thin := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2017, Tiers: []string{"hunt"}, Windows: []string{"summer"}, Postures: []string{"later"}})
	if thin.Status != ReadingThin || thin.AbsenceReason == nil || thin.Count >= 40 {
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
		if failed.Status != ReadingFailed || failed.AbsenceReason == nil {
			t.Fatalf("invalid IPC scope accepted: %+v -> %s", raw, failed.Status)
		}
	}
}

func TestCounterpartTeamPanel(t *testing.T) {
	a := &App{}
	r := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0001"})
	if r.Status != ReadingOK || r.Count == 0 || r.Threshold != 1 {
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
	if empty.Status != ReadingThin || !empty.Value.Empty || empty.Value.Note == "" {
		t.Fatalf("empty team panel must be honest: %s empty %t", empty.Status, empty.Value.Empty)
	}
	unknown := a.TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0099"})
	if unknown.Status != ReadingFailed || unknown.AbsenceReason == nil {
		t.Fatal("unknown franchise accepted")
	}
}

func TestDraftToneWordsOnScreen(t *testing.T) {
	for tone, want := range map[string]string{
		"up":   "above its slot beyond chance",
		"down": "below its slot beyond chance",
		"even": "within chance of its slot",
	} {
		line := draftCellLine(lab.DraftCell{Tone: tone, Ratio: ptrFloat(1.1), Lo: ptrFloat(1.0), Hi: ptrFloat(1.2)})
		if line.Detail != want {
			t.Fatalf("tone %s rendered %q, want %q", tone, line.Detail, want)
		}
	}
	for tone, want := range map[string]string{
		"up":   "above baseline beyond chance",
		"down": "below baseline beyond chance",
		"even": "within chance of baseline",
	} {
		h := lab.DraftHits{Rate: 0.4, Baseline: 0.2, Tone: tone}
		section := draftHitsSection(lab.DraftPosition{Hits: &h})
		if !strings.Contains(section.Rows[0].Detail, want) {
			t.Fatalf("hits tone %s missing words %q in %q", tone, want, section.Rows[0].Detail)
		}
		if strings.Contains(section.Rows[0].Detail, "· "+tone) {
			t.Fatalf("raw tone token reached the screen: %q", section.Rows[0].Detail)
		}
	}
}

func TestCountLabelPlural(t *testing.T) {
	for _, c := range []struct {
		n          int
		noun, want string
	}{
		{1, "record", "1 record"}, {2, "record", "2 records"},
		{1, "pick", "1 pick"}, {0, "trade", "0 trades"},
		{1, "trade", "1 trade"}, {1, "trade side", "1 trade side"},
	} {
		if got := countLabel(c.n, c.noun); got != c.want {
			t.Errorf("countLabel(%d, %q) = %q, want %q", c.n, c.noun, got, c.want)
		}
	}
}

func TestCounterpartyDisplayThresholds(t *testing.T) {
	rows := []lab.TierTrendRow{{N: 9}, {N: 10}}
	view := counterTrends(lab.TeamTrends{Tiers: []lab.TierTrend{{Tier: "hunt", Rows: rows}}})
	if view.Tiers[0].Rows[0].ShowShares || !view.Tiers[0].Rows[1].ShowShares {
		t.Fatal("season share threshold must be 10, as in the reference display")
	}
	panel := (&App{}).TargetCounterpartTeam(RawCounterpartTeam{Scope: RawCounterpartiesScope{From: 2017, To: 2026}, Fid: "0001"})
	peak := 0.01
	for _, kind := range panel.Value.Kinds {
		peak = max(peak, kind.InW, kind.OutW)
	}
	if panel.Value.FlowScale != peak {
		t.Fatal("all asset-flow bars must share the same Go-supplied scale")
	}
}

func TestCounterCountFormatting(t *testing.T) {
	for _, c := range []struct {
		n    int
		want string
	}{
		{0, "0"}, {999, "999"}, {1000, "1,000"}, {3810, "3,810"}, {1000000, "1,000,000"}, {-3810, "-3,810"},
	} {
		if got := countNumber(c.n); got != c.want {
			t.Errorf("countNumber(%d)=%s, want %s", c.n, got, c.want)
		}
	}
	if got := counterShare(lab.TeamShare{K: 1000, N: 3810}).CountText; got != "1,000 of 3,810" {
		t.Fatal(got)
	}
	if got := countLabel(3810, "trade side"); got != "3,810 trade sides" {
		t.Fatal(got)
	}
	a := &App{}
	r := a.TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026})
	if !strings.Contains(r.Value.Summary, "3,810 trade sides") || r.Value.CountText != "3,810" {
		t.Fatal("unformatted archive count")
	}
	for _, row := range r.Value.Rows {
		if row.NText != countNumber(row.N) || row.PartnersText != countNumber(row.PartnersCount) {
			t.Fatal("unformatted team counts")
		}
	}
	for _, tier := range r.Value.Trends.Tiers {
		for _, row := range tier.Rows {
			if row.NText != countNumber(row.N) {
				t.Fatal("unformatted trend count")
			}
		}
	}
}
func TestCounterRoundedZero(t *testing.T) {
	for _, c := range []struct {
		v    float64
		dp   int
		want string
	}{
		{-0.0001, 2, "0.00"}, {-0.0049, 2, "0.00"}, {-0.0051, 2, "-0.01"}, {-0.04, 1, "0.0"}, {0, 2, "0.00"}, {0.0049, 2, "0.00"}, {0.0051, 2, "+0.01"},
	} {
		if got := wins(c.v, c.dp); got != c.want {
			t.Errorf("wins(%v,%d)=%s, want %s", c.v, c.dp, got, c.want)
		}
	}
	e := lab.TeamEdge{Mean: ptrFloat(-0.0001), Lo: ptrFloat(-0.1), Hi: ptrFloat(0.1), Tone: "even"}
	if got := edgeWords(e, "wins"); got != "0.00 wins, within chance of zero" {
		t.Fatal("negative zero in prose")
	}
}
func TestCounterVisibleWording(t *testing.T) {
	w := counterWords()
	if w.Day != "On the day" || w.Since != "Since" || w.Up != "↑" || w.Down != "↓" {
		t.Fatal("bar labels or non-colour marks missing")
	}
	for _, word := range []string{"red", "below", "beyond chance", "Unmarked", "losses", "gains"} {
		if !strings.Contains(w.Legend, word) {
			t.Errorf("legend missing %s", word)
		}
	}
	for _, tier := range lab.TeamTiers() {
		if tierOne(tier) != tierLabel(tier) {
			t.Fatal("tier case drift")
		}
	}
	a := &App{}
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
		if !strings.Contains(tier.Label, tierLabel(tier.Tier)) {
			t.Fatal(tier.Label)
		}
	}
}

func TestCounterMatrixPhrases(t *testing.T) {
	picks := tierPicks(nil, 1)
	for i, want := range []string{"With contenders", "With teams in the hunt", "With rebuilding teams"} {
		if picks[i].With != want || picks[i].TradingAs != tierLabel(picks[i].ID) {
			t.Fatalf("matrix phrase: %+v, want %q", picks[i], want)
		}
	}
}
func TestCounterTableTotals(t *testing.T) {
	e := lab.TeamEdge{Total: ptrFloat(4.59), Mean: ptrFloat(0.05), Lo: ptrFloat(-0.02), Hi: ptrFloat(0.12)}
	if got := counterTotalWords(e); got != "+4.59 total · +0.05 a trade (-0.02 to +0.12)" {
		t.Fatal(got)
	}
	if counterTotalWords(lab.TeamEdge{}) != "—" {
		t.Fatal("absent total fabricated")
	}
	for _, row := range (&App{}).TargetCounterparties(RawCounterpartiesScope{From: 2017, To: 2026}).Value.Rows {
		if row.DayText != counterTotalWords(row.Day) || row.SinceText != counterTotalWords(row.Since) {
			t.Fatal("row total text missing")
		}
	}
}

func TestCounterNarrowHeadingsAndPanelPartners(t *testing.T) {
	words := counterWords()
	if words.Sides != "Sides" || words.Now != "For now" || words.Later != "For later" || words.Day != "On the day" || words.Since != "Since" {
		t.Fatalf("unexpected compact headings: %+v", words)
	}
	a := &App{}
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
	picks := counterpartiesReading(d, counterScope(scope)).Value.Picks
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
