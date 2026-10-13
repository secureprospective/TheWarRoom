package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type MarketRange struct {
	Max float64 `json:"max,omitempty"`
	lab.Interval
	Text string `json:"text"`
}
type MarketLine struct {
	Key    string       `json:"key"`
	Label  string       `json:"label"`
	Detail string       `json:"detail"`
	Count  int          `json:"count"`
	Range  *MarketRange `json:"range"`
	Mark   string       `json:"mark"`
}
type MarketBreakdown struct {
	Title string       `json:"title"`
	Count int          `json:"count"`
	Rows  []MarketLine `json:"rows"`
}

type MarketSection struct {
	Breakdowns []MarketBreakdown `json:"breakdowns,omitempty"`
	ID         string            `json:"id"`
	Title      string            `json:"title"`
	Note       string            `json:"note"`
	Rows       []MarketLine      `json:"rows"`
}
type MarketSurface struct {
	Summary  string          `json:"summary"`
	Sections []MarketSection `json:"sections"`
}

func marketPhases() []string {
	return []string{"winter", "opening", "between", "rookieDraft", "summer", "early", "midseason", "deadline"}
}
func marketPhaseName(phase string) string {
	return map[string]string{"winter": "After the season", "opening": "Before the NFL draft", "between": "NFL draft to rookie draft", "rookieDraft": "During the rookie draft", "summer": "Rookie draft to kickoff", "early": "Weeks 1–4", "midseason": "Midseason", "deadline": "Deadline run-in"}[phase]
}
func marketAssets() []string {
	return []string{"QB", "RB", "WR", "TE", "PK", "DL", "LB", "DB", "pick1:next", "pick2:next", "pick3:next", "pick1:later", "pick2:later", "pick3:later"}
}
func marketScope(raw RawMarketScope) lab.Scope {
	s := lab.Scope{From: raw.From, To: raw.To, ByAge: raw.ByAge}
	if len(raw.Phases) > 0 {
		s.Phases = slices.Clone(raw.Phases)
	}
	if len(raw.Assets) > 0 {
		s.Assets = slices.Clone(raw.Assets)
	}
	return s
}
func marketName(kind string) string {
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
func percent(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }
func marketRange(r *lab.Interval) *MarketRange {
	if r == nil {
		return nil
	}
	return &MarketRange{Interval: *r, Text: percent(r.Value) + "; likely " + percent(r.Lo) + " to " + percent(r.Hi)}
}
func surfaceReading(d *lab.Data, raw RawMarketScope, value MarketSurface, count int, reason string) Reading[MarketSurface] {
	base := marketReading(d, marketScope(raw))
	r := Reading[MarketSurface]{Sourced: snapshot.Sourced[MarketSurface]{Value: value, Provenance: base.Provenance}, Status: ReadingOK, Count: count, Threshold: lab.MinWhole, Baseline: base.Baseline, Cutoff: base.Cutoff, Method: base.Method, InputSHA: base.InputSHA, BuildStamp: base.BuildStamp, HowCounted: base.HowCounted}
	if reason != "" {
		r.Status = ReadingThin
		r.AbsenceReason = &reason
	}
	return r
}
func failedSurface(err error) Reading[MarketSurface] {
	base := failedMarket(err)
	return Reading[MarketSurface]{Status: base.Status, AbsenceReason: base.AbsenceReason, Threshold: base.Threshold, InputSHA: base.InputSHA, Method: base.Method}
}
func (a *App) marketInput(raw RawMarketScope) (*lab.Data, error) {
	if err := raw.Validate(); err != nil {
		return nil, fmt.Errorf("read Market: %w", err)
	}
	return a.lab()
}
func marketSections() []MarketSection {
	return []MarketSection{
		{ID: "L-02", Title: "Buy list · paid", Note: "Forecast wins priced below the typical asset; the whole range clears the baseline.", Rows: []MarketLine{}},
		{ID: "L-03", Title: "Sell list · paid", Note: "Forecast wins priced above the typical asset; the whole range clears the baseline.", Rows: []MarketLine{}},
		{ID: "L-04", Title: "Buy list · delivered", Note: "Delivered wins cost less than the going rate.", Rows: []MarketLine{}},
		{ID: "L-05", Title: "Sell list · delivered", Note: "Delivered wins cost more than the going rate.", Rows: []MarketLine{}},
		{ID: "L-01", Title: "Within chance or not enough delivery evidence", Note: "An absent delivery range is not a zero. At least 8 records and 0.5 forecast wins are required.", Rows: []MarketLine{}},
	}
}
func appendMarketRow(sections []MarketSection, r lab.MarketRow) {
	line := MarketLine{Key: r.Kind, Label: marketName(r.Kind), Count: r.N, Detail: "Asset legs in valued trades"}
	if r.Kind != lab.Anchor && r.Paid != nil {
		paid := line
		paid.Range = marketRange(r.Paid)
		if r.Paid.Hi < 1 {
			sections[0].Rows = append(sections[0].Rows, paid)
		}
		if r.Paid.Lo > 1 {
			sections[1].Rows = append(sections[1].Rows, paid)
		}
	}
	line.Range = marketRange(r.Cost)
	line.Detail = marketTrend(r)
	switch {
	case r.Side != nil && *r.Side == "cheap":
		sections[2].Rows = append(sections[2].Rows, line)
	case r.Side != nil && *r.Side == "dear":
		sections[3].Rows = append(sections[3].Rows, line)
	default:
		if r.Cost == nil {
			line.Detail = "Not enough played-out delivery evidence to judge; this is not a zero"
		}
		sections[4].Rows = append(sections[4].Rows, line)
	}
}
func marketTrend(r lab.MarketRow) string {
	switch {
	case r.Holds:
		return "Clear in both halves of the history"
	case r.Trend != nil && *r.Trend == "new":
		return "Clear only lately"
	case r.Trend != nil && *r.Trend == "fading":
		return "Fading; clear only in the earlier half"
	default:
		return "Not clear in both halves independently"
	}
}
func (a *App) TargetMarketLists(raw RawMarketScope) Reading[MarketSurface] {
	d, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	m := lab.LeagueMarket(d, marketScope(raw))
	value := MarketSurface{Summary: "Nothing is priced until this scope has 80 valued trades.", Sections: marketSections()}
	reason := ""
	if m.Rows == nil {
		reason = "Fewer than 80 valued trades; add neighbouring stretches or widen the seasons"
	}
	if m.Going != nil {
		value.Summary = fmt.Sprintf("Going rate: %.3f next-draft firsts per delivered win. Asset chips never change this league-wide yardstick.", *m.Going)
	}
	for _, row := range m.Rows {
		if len(raw.Assets) == 0 || slices.Contains(raw.Assets, lab.AssetOf(row.Kind)) {
			appendMarketRow(value.Sections, row)
		}
	}
	for i := range value.Sections {
		slices.SortStableFunc(value.Sections[i].Rows, func(a, b MarketLine) int {
			if a.Range == nil || b.Range == nil {
				return strings.Compare(a.Key, b.Key)
			}
			sign := 1
			if i == 1 || i == 3 {
				sign = -1
			}
			if a.Range.Value < b.Range.Value {
				return -sign
			}
			if a.Range.Value > b.Range.Value {
				return sign
			}
			return 0
		})
	}
	return surfaceReading(d, raw, value, m.Trades, reason)
}
func (a *App) TargetPickClock(raw RawMarketScope) Reading[MarketSurface] {
	d, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	s := marketScope(raw)
	count := len(lab.ValuedTrades(d, lab.InScope(s)))
	value := MarketSurface{Summary: "A first-round pick a year before its draft, compared with its next-draft price, per forecast win.", Sections: []MarketSection{{ID: "L-14", Title: "Pick clock", Note: "100% is the next-draft first. Each half needs 60 trades; the range, not a grade, shows uncertainty.", Rows: []MarketLine{}}}}
	reason := "Fewer than 80 valued trades for the pick clock"
	if clock := lab.PickClock(d, s); clock != nil {
		reason = ""
		for _, h := range clock.Halves {
			line := MarketLine{Label: fmt.Sprintf("%d–%d", h.From, h.To), Count: h.Trades, Detail: "Fewer than 60 trades or no later-first price in this half"}
			if h.Clock != nil {
				line.Range = marketRange(&lab.Interval{Value: h.Clock.Value, Lo: h.Clock.Lo, Hi: h.Clock.Hi})
				line.Detail = "Within chance of its next-draft price"
				if h.Clock.Clear {
					line.Detail = "A later-draft first costs clearly less than its next-draft price"
				}
			}
			value.Sections[0].Rows = append(value.Sections[0].Rows, line)
		}
	}
	return surfaceReading(d, raw, value, count, reason)
}
func (a *App) TargetMarketStretches(raw RawMarketScope) Reading[MarketSurface] {
	d, err := a.marketInput(raw)
	if err != nil {
		return failedSurface(err)
	}
	s := marketScope(raw)
	// Completed seasons make the per-season denominator comparable across stretches.
	s.To = min(s.To, d.LastFinished)
	totals, err := lab.TradesByPhase(d, s)
	if err != nil {
		return failedSurface(err)
	}
	section := MarketSection{ID: "L-35", Title: "Trades per season by stretch", Note: "Completed two-sided trades, not only valued trades; completed seasons only. Asset chips narrow these counts; stretch selection leaves the calendar context visible.", Rows: []MarketLine{}}
	for _, phase := range marketPhases() {
		p := totals.Phases[phase]
		section.Rows = append(section.Rows, MarketLine{Key: phase, Label: marketPhaseName(phase), Count: p.N, Detail: fmt.Sprintf("%.1f trades per season across %s", p.PerSeason, countLabel(totals.Seasons, "completed season"))})
	}
	reason := ""
	if totals.Total == 0 {
		reason = "No completed two-sided trades in the chosen seasons and assets"
	}
	r := surfaceReading(d, raw, MarketSurface{Summary: fmt.Sprintf("Completed trade history, %d–%d", s.From, s.To), Sections: []MarketSection{section}}, totals.Total, reason)
	r.Threshold = 1
	r.Method = "Two-sided trade counts by each season's league-year calendar; Eastern time"
	r.Baseline = "Completed trades divided by the number of completed seasons with trades in this asset scope"
	r.HowCounted = section.Note
	return r
}
