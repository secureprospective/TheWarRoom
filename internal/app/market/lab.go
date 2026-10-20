package market

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Line is one row of a Lab surface; it is main.MarketLine.
type Line struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Detail string        `json:"detail"`
	Count  int           `json:"count"`
	Range  *report.Range `json:"range"`
	Mark   string        `json:"mark"`
}

// Breakdown is a titled run of lines inside a section; it is main.MarketBreakdown.
type Breakdown struct {
	Title string `json:"title"`
	Count int    `json:"count"`
	Rows  []Line `json:"rows"`
}

// Section is one panel of a Lab surface; it is main.MarketSection.
type Section struct {
	Breakdowns []Breakdown `json:"breakdowns,omitempty"`
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Note       string      `json:"note"`
	Rows       []Line      `json:"rows"`
}

// Surface is a Lab screen's value: a summary and its sections; it is main.MarketSurface.
type Surface struct {
	Summary  string    `json:"summary"`
	Sections []Section `json:"sections"`
}

// Scope is the seasons, stretches and assets a Market surface reads; it is main.RawMarketScope.
type Scope struct {
	From   int
	To     int
	Phases []string
	Assets []string
	ByAge  bool
}

// Validate checks a scope that came over IPC: a real season range, known stretches and assets,
// none repeated.
func (r Scope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To {
		return fmt.Errorf("market scope: invalid season range")
	}
	allowed := Phases()
	seen := make(map[string]bool)
	for _, p := range r.Phases {
		if !slices.Contains(allowed, p) || seen[p] {
			return fmt.Errorf("market scope: invalid or duplicate phase %q", p)
		}
		seen[p] = true
	}
	seen = make(map[string]bool)
	for _, asset := range r.Assets {
		if !slices.Contains(Assets(), asset) || seen[asset] {
			return fmt.Errorf("market scope: invalid or duplicate asset %q", asset)
		}
		seen[asset] = true
	}
	return nil
}

// Lab is what a Market surface reads: the frozen Lab decode, and the build stamp its readings
// carry.
type Lab struct {
	Data  *lab.Data
	Stamp string
}

// Phases are the league-year stretches, in calendar order.
func Phases() []string {
	return []string{"winter", "opening", "between", "rookieDraft", "summer", "early", "midseason", "deadline"}
}

// PhaseName is a stretch's plain name.
func PhaseName(phase string) string {
	return map[string]string{"winter": "After the season", "opening": "Before the NFL draft", "between": "NFL draft to rookie draft", "rookieDraft": "During the rookie draft", "summer": "Rookie draft to kickoff", "early": "Weeks 1–4", "midseason": "Midseason", "deadline": "Deadline run-in"}[phase]
}

// Assets are the traded asset kinds: the eight positions, then the picks by round and draft.
func Assets() []string {
	return []string{"QB", "RB", "WR", "TE", "PK", "DL", "LB", "DB", "pick1:next", "pick2:next", "pick3:next", "pick1:later", "pick2:later", "pick3:later"}
}

// LabScope is the Lab's scope for a request, empty filters left unset.
func LabScope(raw Scope) lab.Scope {
	s := lab.Scope{From: raw.From, To: raw.To, ByAge: raw.ByAge}
	if len(raw.Phases) > 0 {
		s.Phases = slices.Clone(raw.Phases)
	}
	if len(raw.Assets) > 0 {
		s.Assets = slices.Clone(raw.Assets)
	}
	return s
}

// Name is an asset kind's plain name: "Linebackers, 25 to 27", "Next-draft 1sts".
func Name(kind string) string {
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

// Percent writes a ratio as a percentage to one decimal.
func Percent(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }

// RangeOf words a ratio's interval as a range; nil stays nil.
func RangeOf(r *lab.Interval) *report.Range {
	if r == nil {
		return nil
	}
	return &report.Range{Interval: *r, Text: Percent(r.Value) + "; likely " + Percent(r.Lo) + " to " + Percent(r.Hi)}
}

// FailedMarket is the reading of a market that could not be read, saying why.
func FailedMarket(err error) report.Reading[lab.Market] {
	reason := err.Error()
	return report.Reading[lab.Market]{Status: report.Failed, AbsenceReason: &reason, Threshold: lab.MinWhole, Method: "ridge market fit; frozen compiled facts", InputSHA: lab.DataSHA}
}

// Market is the league market for a scope, with the evidence it rests on; thin below 80 trades.
func (l Lab) Market(s lab.Scope) report.Reading[lab.Market] {
	d := l.Data
	value := lab.LeagueMarket(d, s)
	r := report.Reading[lab.Market]{Sourced: snapshot.Sourced[lab.Market]{Value: value, Provenance: snapshot.Provenance{Source: "Lab compiled facts sha256:" + lab.DataSHA, Kind: "archive", Freshness: domain.Freshness{State: domain.FreshStale, FetchedAt: d.Built, Note: "Frozen historical input; not current league truth"}}}, Status: report.OK, Count: value.Trades, Threshold: lab.MinWhole, Baseline: "Median traded position or pick cost per delivered win; age splits keep this yardstick", Cutoff: d.ArchiveThrough, Method: "ridge least-squares with partial-pivot Gauss and sandwich covariance", InputSHA: lab.DataSHA, BuildStamp: l.Stamp, HowCounted: "Valued two-sided trades in the selected seasons and league-year stretches. Prices are anchored to a next-draft first-round pick. At least 80 trades for the whole view, 60 per half or window, and 8 delivery records with at least 0.5 forecast wins. Delivered wins are compared with the forecast at trade time. Ranges carry price and delivery uncertainty. The going rate is the asset-count-weighted median across positions and picks, never a live quote."}
	if value.Rows == nil {
		reason := "Fewer than 80 valued trades in this scope"
		r.Status = report.Thin
		r.AbsenceReason = &reason
	}
	return r
}

func (l Lab) surfaceReading(raw Scope, value Surface, count int, reason string) report.Reading[Surface] {
	base := l.Market(LabScope(raw))
	r := report.Reading[Surface]{Sourced: snapshot.Sourced[Surface]{Value: value, Provenance: base.Provenance}, Status: report.OK, Count: count, Threshold: lab.MinWhole, Baseline: base.Baseline, Cutoff: base.Cutoff, Method: base.Method, InputSHA: base.InputSHA, BuildStamp: base.BuildStamp, HowCounted: base.HowCounted}
	if reason != "" {
		r.Status = report.Thin
		r.AbsenceReason = &reason
	}
	return r
}

// FailedSurface is the reading of a surface that could not be read, saying why.
func FailedSurface(err error) report.Reading[Surface] {
	base := FailedMarket(err)
	return report.Reading[Surface]{Status: base.Status, AbsenceReason: base.AbsenceReason, Threshold: base.Threshold, InputSHA: base.InputSHA, Method: base.Method}
}
func listSections() []Section {
	return []Section{
		{ID: "L-02", Title: "Buy list · paid", Note: "Forecast wins priced below the typical asset; the whole range clears the baseline.", Rows: []Line{}},
		{ID: "L-03", Title: "Sell list · paid", Note: "Forecast wins priced above the typical asset; the whole range clears the baseline.", Rows: []Line{}},
		{ID: "L-04", Title: "Buy list · delivered", Note: "Delivered wins cost less than the going rate.", Rows: []Line{}},
		{ID: "L-05", Title: "Sell list · delivered", Note: "Delivered wins cost more than the going rate.", Rows: []Line{}},
		{ID: "L-01", Title: "Within chance or not enough delivery evidence", Note: "An absent delivery range is not a zero. At least 8 records and 0.5 forecast wins are required.", Rows: []Line{}},
	}
}
func appendListRow(sections []Section, r lab.MarketRow) {
	line := Line{Key: r.Kind, Label: Name(r.Kind), Count: r.N, Detail: "Asset legs in valued trades"}
	if r.Kind != lab.Anchor && r.Paid != nil {
		paid := line
		paid.Range = RangeOf(r.Paid)
		if r.Paid.Hi < 1 {
			sections[0].Rows = append(sections[0].Rows, paid)
		}
		if r.Paid.Lo > 1 {
			sections[1].Rows = append(sections[1].Rows, paid)
		}
	}
	line.Range = RangeOf(r.Cost)
	line.Detail = trend(r)
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
func trend(r lab.MarketRow) string {
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

// Lists are the buy and sell lists: assets priced clearly below or above the typical asset,
// paid and delivered.
func (l Lab) Lists(raw Scope) report.Reading[Surface] {
	d := l.Data
	m := lab.LeagueMarket(d, LabScope(raw))
	value := Surface{Summary: "Nothing is priced until this scope has 80 valued trades.", Sections: listSections()}
	reason := ""
	if m.Rows == nil {
		reason = "Fewer than 80 valued trades; add neighbouring stretches or widen the seasons"
	}
	if m.Going != nil {
		value.Summary = fmt.Sprintf("Going rate: %.3f next-draft firsts per delivered win. Asset chips never change this league-wide yardstick.", *m.Going)
	}
	for _, row := range m.Rows {
		if len(raw.Assets) == 0 || slices.Contains(raw.Assets, lab.AssetOf(row.Kind)) {
			appendListRow(value.Sections, row)
		}
	}
	for i := range value.Sections {
		slices.SortStableFunc(value.Sections[i].Rows, func(a, b Line) int {
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
	return l.surfaceReading(raw, value, m.Trades, reason)
}

// PickClock compares a first a year before its draft with its next-draft price, per half of
// the history.
func (l Lab) PickClock(raw Scope) report.Reading[Surface] {
	d := l.Data
	s := LabScope(raw)
	count := len(lab.ValuedTrades(d, lab.InScope(s)))
	value := Surface{Summary: "A first-round pick a year before its draft, compared with its next-draft price, per forecast win.", Sections: []Section{{ID: "L-14", Title: "Pick clock", Note: "100% is the next-draft first. Each half needs 60 trades; the range, not a grade, shows uncertainty.", Rows: []Line{}}}}
	reason := "Fewer than 80 valued trades for the pick clock"
	if clock := lab.PickClock(d, s); clock != nil {
		reason = ""
		for _, h := range clock.Halves {
			line := Line{Label: fmt.Sprintf("%d–%d", h.From, h.To), Count: h.Trades, Detail: "Fewer than 60 trades or no later-first price in this half"}
			if h.Clock != nil {
				line.Range = RangeOf(&lab.Interval{Value: h.Clock.Value, Lo: h.Clock.Lo, Hi: h.Clock.Hi})
				line.Detail = "Within chance of its next-draft price"
				if h.Clock.Clear {
					line.Detail = "A later-draft first costs clearly less than its next-draft price"
				}
			}
			value.Sections[0].Rows = append(value.Sections[0].Rows, line)
		}
	}
	return l.surfaceReading(raw, value, count, reason)
}

// Stretches counts completed trades per season in each league-year stretch.
func (l Lab) Stretches(raw Scope) report.Reading[Surface] {
	d := l.Data
	s := LabScope(raw)
	// Completed seasons make the per-season denominator comparable across stretches.
	s.To = min(s.To, d.LastFinished)
	totals, err := lab.TradesByPhase(d, s)
	if err != nil {
		return FailedSurface(err)
	}
	section := Section{ID: "L-35", Title: "Trades per season by stretch", Note: "Completed two-sided trades, not only valued trades; completed seasons only. Asset chips narrow these counts; stretch selection leaves the calendar context visible.", Rows: []Line{}}
	for _, phase := range Phases() {
		p := totals.Phases[phase]
		section.Rows = append(section.Rows, Line{Key: phase, Label: PhaseName(phase), Count: p.N, Detail: fmt.Sprintf("%.1f trades per season across %s", p.PerSeason, report.Count(totals.Seasons, "completed season"))})
	}
	reason := ""
	if totals.Total == 0 {
		reason = "No completed two-sided trades in the chosen seasons and assets"
	}
	r := l.surfaceReading(raw, Surface{Summary: fmt.Sprintf("Completed trade history, %d–%d", s.From, s.To), Sections: []Section{section}}, totals.Total, reason)
	r.Threshold = 1
	r.Method = "Two-sided trade counts by each season's league-year calendar; Eastern time"
	r.Baseline = "Completed trades divided by the number of completed seasons with trades in this asset scope"
	r.HowCounted = section.Note
	return r
}
