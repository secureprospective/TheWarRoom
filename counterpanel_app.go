package main

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// RawCounterpartTeam opens one franchise's panel; Scope.Team is ignored because
// the panel itself is the team view.
type RawCounterpartTeam struct {
	Scope RawCounterpartiesScope `json:"scope"`
	Fid   string                 `json:"fid"`
}

type SeasonCellView struct {
	NowText     string   `json:"nowText"`
	LaterText   string   `json:"laterText"`
	SwapText    string   `json:"swapText"`
	AgainstText string   `json:"againstText"`
	Season      int      `json:"season"`
	N           int      `json:"n"`
	Now         int      `json:"now"`
	Later       int      `json:"later"`
	Swap        int      `json:"swap"`
	Against     int      `json:"against"`
	Flagged     bool     `json:"flagged"`
	MostlyText  string   `json:"mostlyText"`
	Net         *float64 `json:"net"`
	Finish      *int     `json:"finish"`
	Label       string   `json:"label"`
	Title       string   `json:"title"`
}
type KindFlowView struct {
	Kind string  `json:"kind"`
	Name string  `json:"name"`
	InN  int     `json:"inN"`
	OutN int     `json:"outN"`
	InW  float64 `json:"inW"`
	OutW float64 `json:"outW"`
	Text string  `json:"text"`
}
type PartnerView struct {
	Fid       string       `json:"fid"`
	Name      string       `json:"name"`
	N         int          `json:"n"`
	Net       lab.TeamEdge `json:"net"`
	TradesTxt string       `json:"tradesText"`
	NetTxt    string       `json:"netText"`
}
type AsTierView struct {
	Tier  string       `json:"tier"`
	N     int          `json:"n"`
	Edge  lab.TeamEdge `json:"edge"`
	Label string       `json:"label"`
	Text  string       `json:"text"`
}
type TeamCounterpart struct {
	PartnersText string            `json:"partnersText"`
	Words        CounterWords      `json:"words"`
	CountText    string            `json:"countText"`
	Fid          string            `json:"fid"`
	Name         string            `json:"name"`
	AsOf         string            `json:"asOf"`
	FlowScale    float64           `json:"flowScale"`
	Sides        int               `json:"sides"`
	Empty        bool              `json:"empty"`
	Note         string            `json:"note"`
	Seasons      []SeasonCellView  `json:"seasons"`
	SeasonSay    string            `json:"seasonSay"`
	Posture      []PostureCell     `json:"posture"`
	Kinds        []KindFlowView    `json:"kinds"`
	KindSay      string            `json:"kindSay"`
	Partners     []PartnerView     `json:"partners"`
	PartnersSay  string            `json:"partnersSay"`
	Behaviour    lab.TeamBehaviour `json:"behaviour"`
	AsTier       []AsTierView      `json:"asTier"`
	Day          lab.TeamEdge      `json:"day"`
	Since        lab.TeamEdge      `json:"since"`
	DayTxt       string            `json:"dayText"`
	SinceTxt     string            `json:"sinceText"`
}

func (a *App) TargetCounterpartTeam(raw RawCounterpartTeam) Reading[TeamCounterpart] {
	if err := raw.Scope.Validate(); err != nil {
		return failedTeamCounterpart(err)
	}
	d, err := a.lab()
	if err != nil {
		return failedTeamCounterpart(err)
	}
	if _, ok := d.Franchises[raw.Fid]; !ok {
		return failedTeamCounterpart(fmt.Errorf("counterparties team: unknown franchise %q", raw.Fid))
	}
	s := counterScope(raw.Scope)
	s.Team = ""
	league := lab.SidesInScope(d, lab.TeamScope{From: s.From, To: s.To})
	sides := lab.SidesInScope(d, s)
	profile := lab.TeamProfile(d, sides, league, raw.Fid)
	b := profile.Behaviour
	value := TeamCounterpart{
		PartnersText: countNumber(len(profile.Partners)),
		FlowScale:    counterFlowScale(profile),
		Fid:          raw.Fid,
		Name:         franchiseName(d, raw.Fid),
		AsOf:         "Standings as of the archive cutoff " + strings.Split(d.ArchiveThrough, "T")[0] + ", not today's live standings.",
		Sides:        len(profile.Sides),
		Seasons:      seasonCells(profile),
		SeasonSay:    seasonSay(b),
		Posture:      postureCells(b),
		Kinds:        kindFlows(profile),
		KindSay:      "Bars are forecast wins sent and taken in for each kind of asset; the figures count the assets.",
		Partners:     partnerViews(d, profile),
		Behaviour:    b,
		AsTier:       asTierViews(profile),
		Day:          b.Now,
		Since:        b.Real,
	}
	value.Words, value.CountText = counterWords(), countNumber(len(profile.Sides))
	if b.Now.Total != nil {
		value.DayTxt = fmt.Sprintf("%s expected wins on the day, all trades", wins(*b.Now.Total, 2))
	} else {
		value.DayTxt = "Too few trades for a total on the day"
	}
	if b.Real.N > 0 {
		value.SinceTxt = fmt.Sprintf("%s wins since, over %s with 2+ seasons played", wins(*b.Real.Total, 2), countLabel(b.Real.N, "trade"))
	}
	r := Reading[TeamCounterpart]{Sourced: snapshot.Sourced[TeamCounterpart]{Value: value, Provenance: counterProvenance(d)}, Status: ReadingOK, Count: len(profile.Sides), Threshold: 1, Baseline: "The league's own posture and window shares in the same seasons; a share is coloured only where it clears chance", Cutoff: d.ArchiveThrough, Method: "this team's sides of the valued trades against the league in the same seasons; 1.96 standard-error ranges", InputSHA: lab.DataSHA, BuildStamp: buildLabel() + " " + buildDate, HowCounted: counterpartHowCounted(d)}
	if len(profile.Sides) == 0 {
		reason := "No trade sides for this team in the seasons and filters in view"
		value.Empty = true
		value.Note = "This team shows no valued trades in view. Widen the seasons, time of year or what the trades did."
		value.PartnersSay = "No partners in view."
		r.Status, r.AbsenceReason = ReadingThin, &reason
		r.Value = value
		return r
	}
	more := len(profile.Partners) - len(value.Partners)
	if more > 0 {
		value.PartnersSay = "And " + countLabel(more, "more partner") + " in view."
	}
	r.Value = value
	return r
}
func failedTeamCounterpart(err error) Reading[TeamCounterpart] {
	reason := err.Error()
	return Reading[TeamCounterpart]{Status: ReadingFailed, AbsenceReason: &reason, Threshold: 1, Method: "this team's sides of the valued trades against the league", InputSHA: lab.DataSHA}
}
func seasonCells(p lab.TeamDetail) []SeasonCellView {
	out := []SeasonCellView{}
	for _, c := range p.BySeason {
		cell := SeasonCellView{NowText: countNumber(c.Now), LaterText: countNumber(c.Later), SwapText: countNumber(c.Swap), AgainstText: countNumber(c.Against), Season: c.Season, N: c.N, Now: c.Now, Later: c.Later, Swap: c.Swap, Against: c.Against, Flagged: c.Flagged(), Net: c.Net, Finish: c.Finish, MostlyText: ""}
		if c.Mostly != nil {
			cell.MostlyText = tierOne(*c.Mostly)
		}
		if c.Finish != nil {
			cell.Label = fmt.Sprintf("%d · finished %d", c.Season, *c.Finish)
		} else {
			cell.Label = fmt.Sprintf("%d", c.Season)
		}
		cell.Title = fmt.Sprintf("%d: %s; %s for now, %s for later, %s like for like", c.Season, countLabel(c.N, "trade side"), countNumber(c.Now), countNumber(c.Later), countNumber(c.Swap))
		if c.Finish != nil {
			cell.Title += fmt.Sprintf("; finished %d", *c.Finish)
		}
		if c.Flagged() {
			cell.Title += fmt.Sprintf(". %s went against its standing that day.", countNumber(c.Against))
		}
		out = append(out, cell)
	}
	return out
}
func seasonSay(b lab.TeamBehaviour) string {
	return fmt.Sprintf("%s, %.1f a season. Bars: bought for now, built for later, like for like. An outlined season is one where at least two trades and at least 30%% of its trades went against the team's standing that day: buying for now while rebuilding, or building while contending.", countLabel(b.N, "trade side"), b.PerSeason)
}
func postureCells(b lab.TeamBehaviour) []PostureCell {
	out := []PostureCell{}
	for _, p := range lab.TeamPostures() {
		out = append(out, postureCell(p, b.Posture[p]))
	}
	return out
}
func kindFlows(p lab.TeamDetail) []KindFlowView {
	out := []KindFlowView{}
	for _, k := range p.Kinds {
		out = append(out, KindFlowView{Kind: k.Kind, Name: marketName(k.Kind), InN: k.InN, OutN: k.OutN, InW: k.InW, OutW: k.OutW, Text: fmt.Sprintf("%s out · %s in", countNumber(k.OutN), countNumber(k.InN))})
	}
	return out
}
func partnerViews(d *lab.Data, p lab.TeamDetail) []PartnerView {
	out := []PartnerView{}
	for _, x := range p.Partners {
		if len(out) == 12 {
			break
		}
		net := "—"
		v := x.Net.Total
		if v == nil {
			v = x.Net.Mean
		}
		if v != nil {
			net = fmt.Sprintf("%s on the day", wins(*v, 2))
		}
		out = append(out, PartnerView{Fid: x.Fid, Name: franchiseName(d, x.Fid), N: x.N, Net: x.Net, TradesTxt: countLabel(x.N, "trade"), NetTxt: net})
	}
	return out
}
func asTierViews(p lab.TeamDetail) []AsTierView {
	out := []AsTierView{}
	for _, t := range p.AsTier {
		out = append(out, AsTierView{Tier: t.Tier, N: t.N, Edge: t.TeamEdge, Label: fmt.Sprintf("As %s (%s)", tierOne(t.Tier), countNumber(t.N)), Text: edgeWords(t.TeamEdge, "expected wins a trade")})
	}
	return out
}

func counterFlowScale(p lab.TeamDetail) float64 {
	scale := 0.01
	for _, kind := range p.Kinds {
		scale = max(scale, kind.InW, kind.OutW)
	}
	return scale
}
