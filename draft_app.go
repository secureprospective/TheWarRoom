package main

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type RawDraftScope struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Seasons  int      `json:"seasons"`
	Segments []string `json:"segments"`
	NFL      []string `json:"nfl"`
	Group    string   `json:"group"`
}

func (r RawDraftScope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To || !slices.Contains([]int{2, 3, 4}, r.Seasons) {
		return fmt.Errorf("draft scope: invalid seasons or range")
	}
	for _, field := range []struct{ values, allowed []string }{{r.Segments, lab.DraftSegments()}, {r.NFL, lab.DraftNFLRounds()}} {
		seen := make(map[string]bool)
		for _, v := range field.values {
			if !slices.Contains(field.allowed, v) || seen[v] {
				return fmt.Errorf("draft scope: invalid or duplicate filter %q", v)
			}
			seen[v] = true
		}
	}
	if !slices.Contains(marketAssets()[:8], r.Group) {
		return fmt.Errorf("draft scope: invalid position %q", r.Group)
	}
	return nil
}
func draftScope(raw RawDraftScope) lab.DraftScopeFilter {
	s := lab.DraftScopeFilter{From: max(2017, raw.From), To: raw.To, Seasons: raw.Seasons}
	if len(raw.Segments) > 0 {
		s.Segments = slices.Clone(raw.Segments)
	}
	if len(raw.NFL) > 0 {
		s.NFL = slices.Clone(raw.NFL)
	}
	return s
}
func (a *App) TargetDraftLab(raw RawDraftScope) Reading[MarketSurface] {
	if err := raw.Validate(); err != nil {
		return failedDraft(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedDraft(err)
	}
	s := draftScope(raw)
	picks := lab.DraftScope(d, s)
	count := 0
	for _, p := range picks {
		if p.Judged {
			count++
		}
	}
	value := MarketSurface{Summary: fmt.Sprintf("%s with %s finished; %s profile. 100%% is the same slots in other classes.", countLabel(count, "pick"), countLabel(raw.Seasons, "season"), marketName(raw.Group)), Sections: []MarketSection{}}
	reason := ""
	if count < 40 {
		reason = "Fewer than 40 judged picks; widen the seasons, stretches or NFL rounds, or judge on fewer seasons"
	} else {
		value.Sections = draftSections(d, s, picks, raw.Group)
	}
	base := marketReading(d, lab.Scope{From: s.From, To: s.To})
	r := Reading[MarketSurface]{Sourced: snapshot.Sourced[MarketSurface]{Value: value, Provenance: base.Provenance}, Status: ReadingOK, Count: count, Threshold: 40, Baseline: "100% is wins returned by the same overall slots in other draft classes", Cutoff: base.Cutoff, Method: "slot-relative ratio estimator, 1.96 standard-error ranges and monotone nine-slot value curve", InputSHA: base.InputSHA, BuildStamp: base.BuildStamp, HowCounted: draftHowCounted()}
	if reason != "" {
		r.Status = ReadingThin
		r.AbsenceReason = &reason
	}
	return r
}
func failedDraft(err error) Reading[MarketSurface] {
	r := failedSurface(err)
	r.Threshold = 40
	r.Method = "historical draft against the slot"
	return r
}
func draftHowCounted() string {
	return "A pick's return is wins above replacement in this league's scoring. Its slot is what the same overall pick returned in the other classes, never its own class. Picks from 2017 onward are judged on their first 2, 3 or 4 finished seasons. A ratio is total returned wins divided by total slot wins; each estimate needs 8 records and 0.5 slot wins. Year 1–5 uses every class with that season recorded, including not-yet-judged picks. The likely range is the ratio plus/minus 1.96 standard errors. Best/worst only beats every other known cell beyond chance; with many cells, a colour alone may be chance. Plays like pick N reads the rounded mean slot's usual value times the return ratio onto a nine-slot-smoothed, never-rising curve. Starter hits are top or starter best seasons versus the same draft stretches; 8 judged picks are required. Kept by year 2 counts only known retention outcomes, including unjudged picks. Draft or buy uses the age-split trade market in these same seasons, not the draft filters; its usual pick round is the most common among judged picks, with stable first-seen ties. Frozen history, not a live pick forecast or ownership claim."
}
func draftLabel(id string) string {
	return map[string]string{"r1a": "1.01–1.10", "r1b": "1.11–1.32", "r2a": "2.01–2.16", "r2b": "2.17–2.32", "r3": "Round 3", "r4": "Round 4", "r5": "Round 5 on", "n1": "NFL round 1", "n2": "NFL round 2", "n3": "NFL round 3", "n4": "NFL rounds 4–7", "nx": "No NFL pick that spring", "a21": "21 or younger", "a22": "22", "a23": "23 or older"}[id]
}
func draftRatioRange(r *lab.DraftRatio) *MarketRange {
	if r == nil {
		return nil
	}
	return marketRange(&lab.Interval{Value: r.Ratio, Lo: r.Lo, Hi: r.Hi})
}
func draftCellLine(c lab.DraftCell) MarketLine {
	label := draftLabel(c.Segment + c.NFL + c.Age)
	if c.Season > 0 {
		label = fmt.Sprintf("Year %d", c.Season)
	}
	line := MarketLine{Label: label, Count: c.N, Detail: "Too few picks or less than 0.5 slot wins; not a zero"}
	if c.Ratio != nil {
		line.Range = marketRange(&lab.Interval{Value: *c.Ratio, Lo: *c.Lo, Hi: *c.Hi})
		line.Detail = slotTone(c.Tone)
	}
	if c.Best {
		line.Detail += " · best beyond chance"
	}
	if c.Worst {
		line.Detail += " · worst beyond chance"
	}
	return line
}
func draftAxis(id, title string, cells []lab.DraftCell) MarketSection {
	s := MarketSection{ID: id, Title: title, Note: "100% is the slot's return. Unavailable ranges are not zeros.", Rows: []MarketLine{}}
	for _, c := range cells {
		s.Rows = append(s.Rows, draftCellLine(c))
	}
	return s
}
func draftSections(d *lab.Data, s lab.DraftScopeFilter, picks []lab.JudgedPick, group string) []MarketSection {
	p := lab.PositionDraft(picks, group)
	sections := []MarketSection{
		draftAxis("L-15", "Return by draft stretch · "+marketName(group), p.Where),
		draftAxis("L-16", "Return by season after draft · "+marketName(group), p.When),
		draftAxis("L-17", "Return by NFL draft round", p.NFL),
		draftAxis("L-18", "Return by age at the draft", p.Age),
		draftLikeSection(p), draftHitsSection(p), draftKeptSection(p), draftBuySection(d, s, p),
	}
	// The league maps remain visible alongside the selected position's profile.
	for _, row := range lab.DraftMap(picks, marketAssets()[:8]) {
		for _, c := range row.When {
			line := draftCellLine(c)
			line.Label = marketName(row.Group) + " · " + line.Label
			sections[1].Rows = append(sections[1].Rows, line)
		}
		for _, c := range row.Where {
			sections[0].Rows = append(sections[0].Rows, MarketLine{Label: marketName(row.Group) + " · " + draftLabel(c.Segment), Count: c.N, Range: draftRatioRange(c.Reading), Detail: "Against the same slots; at least 8 judged picks and 0.5 slot wins"})
		}
	}
	return sections
}
func draftLikeSection(p lab.DraftPosition) MarketSection {
	section := MarketSection{ID: "L-19", Title: "Plays like pick #N", Note: "Historical position cohorts, not predictions for a named prospect. Each line opens that round's pick asset in the Inspector.", Rows: []MarketLine{}}
	for i, c := range p.Where {
		line := draftCellLine(c)
		round := []int{1, 1, 2, 2, 3, 3, 3}[i]
		line.Key = fmt.Sprintf("pick%d:next", round)
		if c.Like != nil {
			line.Detail = fmt.Sprintf("Taken around pick #%d · plays like pick #%d", *c.At, *c.Like)
		}
		section.Rows = append(section.Rows, line)
	}
	return section
}
func draftHitsSection(p lab.DraftPosition) MarketSection {
	line := MarketLine{Label: marketName(p.Group), Count: p.N, Detail: "Fewer than 8 judged picks; no hit-rate estimate"}
	if h := p.Hits; h != nil {
		line.Detail = percent(h.Rate) + " became starters or better, against " + percent(h.Baseline) + " for the same stretches · " + chanceTone(h.Tone)
	}
	return MarketSection{ID: "L-20", Title: "Hit rate against the stretch baseline", Note: "Best season was top or starter; the baseline matches each pick's draft stretch.", Rows: []MarketLine{line}}
}
func draftKeptSection(p lab.DraftPosition) MarketSection {
	line := MarketLine{Label: marketName(p.Group), Detail: "No known year-2 retention outcomes"}
	for _, x := range p.All {
		if x.KeptYear2 != nil {
			line.Count++
		}
	}
	if p.Kept != nil {
		line.Detail = percent(*p.Kept) + " kept by year 2 among known outcomes"
	}
	return MarketSection{ID: "L-21", Title: "Kept by year 2", Note: "Unknown retention is excluded, not counted as lost. Includes picks not yet judged on total return.", Rows: []MarketLine{line}}
}
func draftBuySection(d *lab.Data, s lab.DraftScopeFilter, p lab.DraftPosition) MarketSection {
	m := lab.LeagueMarket(d, lab.Scope{From: s.From, To: s.To, ByAge: true})
	buy := lab.DraftBuy(m, p.All, p.Group)
	section := MarketSection{ID: "L-22", Title: "Draft it, or buy it", Note: "Cost per delivered win against the going rate in the same seasons; draft stretches and NFL filters do not narrow the trade market.", Rows: []MarketLine{}}
	for _, r := range buy.Rows {
		line := MarketLine{Key: r.Kind, Label: marketName(r.Kind), Count: r.N, Range: marketRange(r.Cost), Detail: "Historical cost per delivered win"}
		if buy.Cheapest != nil && r.Kind == *buy.Cheapest {
			line.Detail += " · lowest priced age in view (not a beyond-chance claim)"
		}
		if buy.Via != nil && r.Kind == *buy.Via {
			line.Detail += " · pick route for the most common judged round in view (or first priced round if unavailable)"
		}
		section.Rows = append(section.Rows, line)
	}
	if len(section.Rows) == 0 {
		section.Note += " Too few valued trades or played-out deliveries to price these routes."
	}
	return section
}

func countLabel(n int, noun string) string {
	suffix := "s"
	if n == 1 {
		suffix = ""
	}
	return fmt.Sprintf("%d %s%s", n, noun, suffix)
}
func slotTone(tone string) string {
	return map[string]string{"down": "below its slot beyond chance", "even": "within chance of its slot", "up": "above its slot beyond chance"}[tone]
}
func chanceTone(tone string) string {
	return map[string]string{"down": "below baseline beyond chance", "even": "within chance of baseline", "up": "above baseline beyond chance"}[tone]
}
