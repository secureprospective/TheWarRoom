package offers

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type Scope struct {
	From      int    `json:"from"`
	To        int    `json:"to"`
	Direction string `json:"direction"`
	Window    string `json:"window"`
	Team      string `json:"team"`
}

func (r Scope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To {
		return fmt.Errorf("offers scope: invalid season range")
	}
	if !slices.Contains([]string{"all", "sent", "received"}, r.Direction) {
		return fmt.Errorf("offers scope: invalid direction")
	}
	if !slices.Contains([]string{"", "spring", "summer", "season"}, r.Window) {
		return fmt.Errorf("offers scope: invalid window")
	}
	return nil
}

type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	From  int    `json:"from"`
	To    int    `json:"to"`
}
type Outcome struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Text  string  `json:"text"`
	Width float64 `json:"width"`
}
type Stretch struct {
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	CountText  string    `json:"countText"`
	Acceptance string    `json:"acceptance"`
	Finding    string    `json:"finding"`
	Outcomes   []Outcome `json:"outcomes"`
}
type Words struct {
	Heading   string `json:"heading"`
	Direction string `json:"direction"`
	Seasons   string `json:"seasons"`
	Window    string `json:"window"`
	Team      string `json:"team"`
	Retry     string `json:"retry"`
	Pending   string `json:"pending"`
}
type View struct {
	Words      Words     `json:"words"`
	Disclosure string    `json:"disclosure"`
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	CountText  string    `json:"countText"`
	Directions []Choice  `json:"directions"`
	Windows    []Choice  `json:"windows"`
	Ranges     []Choice  `json:"ranges"`
	Teams      []Team    `json:"teams"`
	Stretches  []Stretch `json:"stretches"`
}

type Team struct {
	Fid  string `json:"fid"`
	Name string `json:"name"`
}

func Empty() View {
	return View{
		Words:      Words{Heading: "Offers history", Direction: "Which offers", Seasons: "Seasons", Window: "Time of year", Team: "Counterparty", Retry: "Try again", Pending: "Updating filters; figures still describe the previous scope until Go replies."},
		Disclosure: "MFL exposes only the GM's own offers. The archive is unavailable; whose offers, the count and the cutoff cannot be verified.", Title: "How do offers end?",
		Directions: []Choice{{ID: "all", Label: "Every visible offer"}, {ID: "sent", Label: "Sent"}, {ID: "received", Label: "Received"}},
		Windows:    []Choice{{ID: "", Label: "Every stretch"}, {ID: "spring", Label: "Spring"}, {ID: "summer", Label: "Summer"}, {ID: "season", Label: "Season"}},
		Ranges:     []Choice{{ID: "all", Label: "2013–2026", From: 2013, To: 2026}, {ID: "recent", Label: "2024–2026", From: 2024, To: 2026}, {ID: "early", Label: "2013–2016", From: 2013, To: 2016}},
		Teams:      []Team{{Fid: "", Name: "Every counterparty"}}, Stretches: []Stretch{},
	}
}

func ValidateTeam(raw Scope, d *lab.Data) error {
	if err := raw.Validate(); err != nil {
		return err
	}
	if raw.Team != "" {
		if _, ok := d.Franchises[raw.Team]; !ok {
			return fmt.Errorf("offers scope: unknown team")
		}
	}
	return nil
}

func SetDisclosure(v *View, d *lab.Data) {
	owner := lab.OfferOwner(d)
	name := d.Franchises[owner].Name
	own := 0
	for _, o := range d.Offers {
		if o.By == owner || o.To == owner {
			own++
		}
	}
	v.Disclosure = fmt.Sprintf("MFL exposes only the GM's own offers. This archive holds %s visible to %s (%s), sent and received, through %s. It does not show other GMs' private offers or league-wide offer activity.", report.Count(own, "offer"), name, owner, d.ArchiveThrough)
	for _, id := range sortedTeams(d) {
		v.Teams = append(v.Teams, Team{Fid: id, Name: d.Franchises[id].Name})
	}
}

func sortedTeams(d *lab.Data) []string {
	ids := make([]string, 0, len(d.Franchises))
	for id := range d.Franchises {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func Summary(d *lab.Data, v lab.OfferResult, s lab.OfferScope) string {
	if len(v.Offers) == 0 {
		return "No offers in view."
	}
	who := "to and from " + d.Franchises[v.Owner].Name
	if v.Direction == "sent" {
		who = "sent by " + d.Franchises[v.Owner].Name
	}
	if v.Direction == "received" {
		who = "sent to " + d.Franchises[v.Owner].Name
	}
	if s.Team != "" && s.Team != v.Owner {
		who += " with " + d.Franchises[s.Team].Name
	}
	rate := "No known endings are on record."
	if v.Rate != nil {
		rate = fmt.Sprintf("%s of known endings were accepted.", fmt.Sprintf("%.1f%%", *v.Rate*100))
	}
	findings := []string{fmt.Sprintf("%s %s. %s", report.Count(len(v.Offers), "offer"), who, rate)}
	for _, st := range v.Stretches {
		if st.Best || st.Worst {
			findings = append(findings, Finding(st))
		}
	}
	if len(findings) == 1 {
		usable := 0
		for _, st := range v.Stretches {
			if st.Known >= 10 {
				usable++
			}
		}
		headline := "Too few stretches have enough decided offers to compare."
		if usable >= 2 {
			headline = "No stretch's acceptance rate stands out from the others beyond chance."
		}
		findings = append(findings, headline)
	}
	return strings.Join(findings, " ")
}

func Finding(st lab.OfferStretch) string {
	label := ""
	for _, p := range lab.OfferPhases() {
		if p.ID == st.Phase {
			label = p.Label
		}
	}
	if st.Best {
		return "Most often accepted: " + label + "; beyond chance against every other usable stretch."
	}
	if st.Worst {
		return "Least often accepted: " + label + "; beyond chance against every other usable stretch."
	}
	return ""
}

func StretchView(st lab.OfferStretch) Stretch {
	v := Stretch{ID: st.Phase, CountText: fmt.Sprintf("%s · %.1f a season", report.Count(st.N, "offer"), st.PerSeason), Acceptance: "Too few known endings to show acceptance", Finding: Finding(st), Outcomes: []Outcome{}}
	for _, p := range lab.OfferPhases() {
		if p.ID == st.Phase {
			v.Label = p.Label
		}
	}
	if st.Known >= 5 {
		v.Acceptance = fmt.Sprintf("%s accepted · %s with a known ending", fmt.Sprintf("%.1f%%", *st.Rate*100), report.Count(st.Known, "offer"))
	}
	for _, o := range []Choice{{ID: "accepted", Label: "Accepted"}, {ID: "rejected", Label: "Rejected"}, {ID: "withdrawn", Label: "Withdrawn"}, {ID: "expired", Label: "Expired"}, {ID: "unknown", Label: "Ending not on record"}, {ID: "pending", Label: "Open offers"}} {
		n := st.Counts[o.ID]
		v.Outcomes = append(v.Outcomes, Outcome{ID: o.ID, Label: o.Label, Text: o.Label + " " + report.Number(n), Width: math.Round(float64(n)/float64(max(1, st.N))*10000) / 100})
	}
	return v
}

func OpenSummary(result lab.OfferResult) string {
	open := 0
	for _, o := range result.Offers {
		if o.Outcome == "pending" {
			open++
		}
	}
	if open == 0 {
		return " No open offers."
	}
	return " " + report.Count(open, "open offer") + "."
}
