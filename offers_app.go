package main

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type RawOfferScope struct {
	From      int    `json:"from"`
	To        int    `json:"to"`
	Direction string `json:"direction"`
	Window    string `json:"window"`
	Team      string `json:"team"`
}

func (r RawOfferScope) Validate() error {
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

type OfferChoice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	From  int    `json:"from"`
	To    int    `json:"to"`
}
type OfferOutcomeView struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Text  string  `json:"text"`
	Width float64 `json:"width"`
}
type OfferStretchView struct {
	ID         string             `json:"id"`
	Label      string             `json:"label"`
	CountText  string             `json:"countText"`
	Acceptance string             `json:"acceptance"`
	Finding    string             `json:"finding"`
	Outcomes   []OfferOutcomeView `json:"outcomes"`
}
type OfferWords struct {
	Heading   string `json:"heading"`
	Direction string `json:"direction"`
	Seasons   string `json:"seasons"`
	Window    string `json:"window"`
	Team      string `json:"team"`
	Retry     string `json:"retry"`
	Pending   string `json:"pending"`
}
type OffersView struct {
	Words      OfferWords         `json:"words"`
	Disclosure string             `json:"disclosure"`
	Title      string             `json:"title"`
	Summary    string             `json:"summary"`
	CountText  string             `json:"countText"`
	Directions []OfferChoice      `json:"directions"`
	Windows    []OfferChoice      `json:"windows"`
	Ranges     []OfferChoice      `json:"ranges"`
	Teams      []TeamOption       `json:"teams"`
	Stretches  []OfferStretchView `json:"stretches"`
}

func offerWords() OffersView {
	return OffersView{
		Words:      OfferWords{Heading: "Offers history", Direction: "Which offers", Seasons: "Seasons", Window: "Time of year", Team: "Counterparty", Retry: "Try again", Pending: "Updating filters; figures still describe the previous scope until Go replies."},
		Disclosure: "MFL exposes only the GM's own offers. The archive is unavailable; whose offers, the count and the cutoff cannot be verified.", Title: "How do offers end?",
		Directions: []OfferChoice{{ID: "all", Label: "Every visible offer"}, {ID: "sent", Label: "Sent"}, {ID: "received", Label: "Received"}},
		Windows:    []OfferChoice{{ID: "", Label: "Every stretch"}, {ID: "spring", Label: "Spring"}, {ID: "summer", Label: "Summer"}, {ID: "season", Label: "Season"}},
		Ranges:     []OfferChoice{{ID: "all", Label: "2013–2026", From: 2013, To: 2026}, {ID: "recent", Label: "2024–2026", From: 2024, To: 2026}, {ID: "early", Label: "2013–2016", From: 2013, To: 2016}},
		Teams:      []TeamOption{{Fid: "", Name: "Every counterparty"}}, Stretches: []OfferStretchView{},
	}
}
func (a *App) TargetOffers(raw RawOfferScope) Reading[OffersView] {
	r := Reading[OffersView]{Sourced: snapshot.Sourced[OffersView]{Value: offerWords()}, Status: ReadingFailed, Threshold: 10, Method: "owner-scoped offerStretches; 1.96 standard-error comparisons", InputSHA: lab.DataSHA, BuildStamp: buildLabel() + " " + buildDate, HowCounted: "Only offers visible to the archived GM are available. No league-wide offer history is supplied."}
	d, err := a.lab()
	if err != nil {
		reason := err.Error()
		r.AbsenceReason = &reason
		return r
	}
	setOfferDisclosure(&r.Value, d)
	merged, source, checked, err := a.offersWithLive(d)
	if err != nil {
		reason := err.Error()
		r.AbsenceReason = &reason
		return r
	}
	d = merged
	r.Cutoff = source
	r.Provenance = counterProvenance(d)
	r.Provenance.Source = source
	if !checked.IsZero() {
		// Live offers are added since the archive, so the reading is as fresh as the last pending-offer read.
		r.Provenance.Kind, r.Provenance.Freshness = "live", liveFreshness(checked)
	}
	if err := validateOfferTeam(raw, d); err != nil {
		reason := err.Error()
		r.AbsenceReason = &reason
		return r
	}
	s := lab.OfferScope{From: raw.From, To: raw.To, Direction: raw.Direction, Team: raw.Team}
	if raw.Window != "" {
		s.Windows = []string{raw.Window}
	}
	result := lab.OfferStretches(d, s)
	r.Status = ReadingOK
	r.Count = len(result.Offers)
	r.Value.CountText = countNumber(r.Count)
	r.Baseline = "Other stretches with at least 10 known endings in this same scope"
	r.HowCounted = "Seasons use the league-year season field, not calendar year. Direction is relative to the archive owner (the busiest endpoint, stable first-appearance ties). Team and time-of-year filters apply before counts. Accepted, rejected, withdrawn and expired are known endings; unknown endings are excluded from acceptance. Per-season volume divides by the number of seasons with any recorded offer in the selected years, before direction, team or window filtering (at least one). Rates display only at 5 known endings; a most/least often accepted finding requires at least 10 known endings in each compared stretch, two usable stretches, and a strict difference greater than 1.96 × the combined standard error against every other usable stretch. Unknown endings are never rejections. Open offers are counted separately and excluded from endings and acceptance. Ending not on record means MFL no longer lists the offer and no matching completed trade is held; it is not a rejection. Live coverage contains only offers seen by this app, not all offers since the archive. This is owner coverage, not league-wide activity."
	r.Value.Summary = offerSummary(d, result, s)
	r.Value.Summary += offerOpenSummary(result)
	for _, st := range result.Stretches {
		r.Value.Stretches = append(r.Value.Stretches, offerStretchView(st))
	}
	if r.Count < 10 {
		r.Status = ReadingThin
		reason := "Fewer than 10 offers in this scope; individual stretches may have too few known endings to compare"
		r.AbsenceReason = &reason
	}
	return r
}
func validateOfferTeam(raw RawOfferScope, d *lab.Data) error {
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
func setOfferDisclosure(v *OffersView, d *lab.Data) {
	owner := lab.OfferOwner(d)
	name := d.Franchises[owner].Name
	own := 0
	for _, o := range d.Offers {
		if o.By == owner || o.To == owner {
			own++
		}
	}
	v.Disclosure = fmt.Sprintf("MFL exposes only the GM's own offers. This archive holds %s visible to %s (%s), sent and received, through %s. It does not show other GMs' private offers or league-wide offer activity.", countLabel(own, "offer"), name, owner, d.ArchiveThrough)
	for _, id := range sortedOfferTeams(d) {
		v.Teams = append(v.Teams, TeamOption{Fid: id, Name: d.Franchises[id].Name})
	}
}
func sortedOfferTeams(d *lab.Data) []string {
	ids := make([]string, 0, len(d.Franchises))
	for id := range d.Franchises {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}
func offerSummary(d *lab.Data, v lab.OfferResult, s lab.OfferScope) string {
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
		rate = fmt.Sprintf("%s of known endings were accepted.", percent(*v.Rate))
	}
	findings := []string{fmt.Sprintf("%s %s. %s", countLabel(len(v.Offers), "offer"), who, rate)}
	for _, st := range v.Stretches {
		if st.Best || st.Worst {
			findings = append(findings, offerFinding(st))
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
func offerFinding(st lab.OfferStretch) string {
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
func offerStretchView(st lab.OfferStretch) OfferStretchView {
	v := OfferStretchView{ID: st.Phase, CountText: fmt.Sprintf("%s · %.1f a season", countLabel(st.N, "offer"), st.PerSeason), Acceptance: "Too few known endings to show acceptance", Finding: offerFinding(st), Outcomes: []OfferOutcomeView{}}
	for _, p := range lab.OfferPhases() {
		if p.ID == st.Phase {
			v.Label = p.Label
		}
	}
	if st.Known >= 5 {
		v.Acceptance = fmt.Sprintf("%s accepted · %s with a known ending", percent(*st.Rate), countLabel(st.Known, "offer"))
	}
	for _, o := range []OfferChoice{{ID: "accepted", Label: "Accepted"}, {ID: "rejected", Label: "Rejected"}, {ID: "withdrawn", Label: "Withdrawn"}, {ID: "expired", Label: "Expired"}, {ID: "unknown", Label: "Ending not on record"}, {ID: "pending", Label: "Open offers"}} {
		n := st.Counts[o.ID]
		v.Outcomes = append(v.Outcomes, OfferOutcomeView{ID: o.ID, Label: o.Label, Text: o.Label + " " + countNumber(n), Width: math.Round(float64(n)/float64(max(1, st.N))*10000) / 100})
	}
	return v
}

func offerOpenSummary(result lab.OfferResult) string {
	open := 0
	for _, o := range result.Offers {
		if o.Outcome == "pending" {
			open++
		}
	}
	if open == 0 {
		return " No open offers."
	}
	return " " + countLabel(open, "open offer") + "."
}
