package main

import (
	"context"
	"fmt"
	"time"

	offersapp "github.com/secureprospective/TheWarRoom/internal/app/offers"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

type RawOfferScope struct {
	From      int    `json:"from"`
	To        int    `json:"to"`
	Direction string `json:"direction"`
	Window    string `json:"window"`
	Team      string `json:"team"`
}

func (r RawOfferScope) Validate() error { return offersapp.Scope(r).Validate() } //nolint:wrapcheck // preserves offer scope error text

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
		// Live offers are added to the archive, so the reading is as fresh as the last pending-offer read. The
		// kind stays "archive" (the screen's contract): the history still rests on it.
		r.Provenance.Freshness = liveFreshness(checked)
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

func offerWords() OffersView { return wireOffers(offersapp.Empty()) }
func validateOfferTeam(raw RawOfferScope, d *lab.Data) error {
	return offersapp.ValidateTeam(offersapp.Scope(raw), d) //nolint:wrapcheck // preserves offer scope error text
}
func setOfferDisclosure(v *OffersView, d *lab.Data) {
	service := offersapp.View{Teams: make([]offersapp.Team, len(v.Teams))}
	for i, t := range v.Teams {
		service.Teams[i] = offersapp.Team(t)
	}
	offersapp.SetDisclosure(&service, d)
	v.Disclosure = service.Disclosure
	v.Teams = make([]TeamOption, len(service.Teams))
	for i, t := range service.Teams {
		v.Teams[i] = TeamOption(t)
	}
}
func offerSummary(d *lab.Data, v lab.OfferResult, s lab.OfferScope) string {
	return offersapp.Summary(d, v, s)
}
func offerOpenSummary(v lab.OfferResult) string { return offersapp.OpenSummary(v) }
func offerStretchView(st lab.OfferStretch) OfferStretchView {
	return wireOfferStretch(offersapp.StretchView(st))
}
func wireOfferStretch(st offersapp.Stretch) OfferStretchView {
	out := OfferStretchView{ID: st.ID, Label: st.Label, CountText: st.CountText, Acceptance: st.Acceptance, Finding: st.Finding}
	if st.Outcomes != nil {
		out.Outcomes = make([]OfferOutcomeView, len(st.Outcomes))
		for i, o := range st.Outcomes {
			out.Outcomes[i] = OfferOutcomeView(o)
		}
	}
	return out
}
func wireOffers(v offersapp.View) OffersView {
	out := OffersView{Words: OfferWords(v.Words), Disclosure: v.Disclosure, Title: v.Title, Summary: v.Summary, CountText: v.CountText}
	if v.Directions != nil {
		out.Directions = make([]OfferChoice, len(v.Directions))
		for i, c := range v.Directions {
			out.Directions[i] = OfferChoice(c)
		}
	}
	if v.Windows != nil {
		out.Windows = make([]OfferChoice, len(v.Windows))
		for i, c := range v.Windows {
			out.Windows[i] = OfferChoice(c)
		}
	}
	if v.Ranges != nil {
		out.Ranges = make([]OfferChoice, len(v.Ranges))
		for i, c := range v.Ranges {
			out.Ranges[i] = OfferChoice(c)
		}
	}
	if v.Teams != nil {
		out.Teams = make([]TeamOption, len(v.Teams))
		for i, c := range v.Teams {
			out.Teams[i] = TeamOption(c)
		}
	}
	if v.Stretches != nil {
		out.Stretches = make([]OfferStretchView, len(v.Stretches))
		for i, c := range v.Stretches {
			out.Stretches[i] = wireOfferStretch(c)
		}
	}
	return out
}

func liveOffer(d *lab.Data, year int, ts int64, sender, receiver, outcome string) (lab.Offer, error) {
	return offersapp.LiveOffer(d, year, ts, sender, receiver, outcome) //nolint:wrapcheck // preserves the existing calendar error text
}

// The caller holds seasonMu, so the ending evidence and the pending read form one observation.
func (a *App) recordOffers(ctx context.Context, pending []leaguefeed.PendingTrade, at time.Time, readErr error) error {
	if readErr != nil || a.pools == nil {
		return readErr
	}
	d, err := a.lab()
	if err != nil {
		return fmt.Errorf("offers: archive owner: %w", err)
	}
	if err := personal.New(a.pools).ObserveOffers(ctx, lab.OfferOwner(d), a.season, pending,
		a.seasonTransactions.value, at); err != nil {
		return fmt.Errorf("offers: save live read: %w", err)
	}
	return nil
}

// offersWithLive also returns when the app last read MFL's pending offers (zero before the first read).
func (a *App) offersWithLive(d *lab.Data) (*lab.Data, string, time.Time, error) {
	if a.pools == nil {
		return d, d.ArchiveThrough, time.Time{}, nil
	}
	ctx, cancel := context.WithTimeout(a.fallbackParent(), 10*time.Second)
	defer cancel()
	owner := lab.OfferOwner(d)
	rows, checked, err := personal.New(a.pools).Offers(ctx, owner)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("offers: local history: %w", err)
	}
	return offersapp.Merge(d, rows, checked) //nolint:wrapcheck // preserves the existing offer history errors
}
