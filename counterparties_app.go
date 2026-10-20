package main

import (
	counterpartiesapp "github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type RawCounterpartiesScope struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Tiers    []string `json:"tiers"`
	Windows  []string `json:"windows"`
	Postures []string `json:"postures"`
	Team     string   `json:"team"`
}

type TierPick struct {
	With      string  `json:"with"`
	TradingAs string  `json:"tradingAs"`
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Hint      string  `json:"hint"`
	Count     float64 `json:"count"`
}

type TeamOption struct {
	Fid  string `json:"fid"`
	Name string `json:"name"`
}

type PostureCell struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	P     *float64 `json:"p"`
	Tone  string   `json:"tone"`
	Title string   `json:"title"`
}

type CardRow struct {
	Label string        `json:"label"`
	Text  string        `json:"text"`
	Tone  string        `json:"tone"`
	Range *lab.TeamEdge `json:"range,omitempty"`
}

type TierCard struct {
	ID           string        `json:"id"`
	Label        string        `json:"label"`
	Hint         string        `json:"hint"`
	Excluded     bool          `json:"excluded"`
	Thin         bool          `json:"thin"`
	PerSeason    float64       `json:"perSeason"`
	PerSeasonTxt string        `json:"perSeasonText"`
	Posture      []PostureCell `json:"posture"`
	Rows         []CardRow     `json:"rows"`
}

type PairView struct {
	Other    string       `json:"other"`
	Excluded bool         `json:"excluded"`
	N        int          `json:"n"`
	Now      lab.TeamEdge `json:"now"`
	Real     lab.TeamEdge `json:"real"`
	Trades   string       `json:"trades"`
	Title    string       `json:"title"`
}

type CounterShare struct {
	lab.TeamShare
	CountText string `json:"countText"`
}

type CounterRow struct {
	DayText       string             `json:"dayText"`
	SinceText     string             `json:"sinceText"`
	NText         string             `json:"nText"`
	PartnersText  string             `json:"partnersText"`
	Fid           string             `json:"fid"`
	Name          string             `json:"name"`
	N             int                `json:"n"`
	PerSeason     float64            `json:"perSeason"`
	Tiers         map[string]float64 `json:"tiers"`
	ForNow        CounterShare       `json:"forNow"`
	ForLater      CounterShare       `json:"forLater"`
	Day           lab.TeamEdge       `json:"day"`
	Since         lab.TeamEdge       `json:"since"`
	Picks         lab.TeamEdge       `json:"picks"`
	Age           lab.TeamEdge       `json:"age"`
	Partner       *lab.TeamPartner   `json:"partner"`
	PartnerName   string             `json:"partnerName"`
	PartnersCount int                `json:"partnersCount"`
	StandingTitle string             `json:"standingTitle"`
	PartnerTrades string             `json:"partnerTrades"`
}

type CounterTrendRow struct {
	Season     int          `json:"season"`
	N          int          `json:"n"`
	Net        *float64     `json:"net"`
	Now        CounterShare `json:"now"`
	Later      CounterShare `json:"later"`
	ShowShares bool         `json:"showShares"`
	NText      string       `json:"nText"`
}

type CounterShift lab.TierShift

type CounterShifts struct {
	Now   *CounterShift `json:"now"`
	Later *CounterShift `json:"later"`
}

type CounterTrend struct {
	Tier  string            `json:"tier"`
	Rows  []CounterTrendRow `json:"rows"`
	Shift CounterShifts     `json:"shift"`
}

type CounterTrends struct {
	Seasons []int          `json:"seasons"`
	Tiers   []CounterTrend `json:"tiers"`
}

type CounterWords struct {
	Team      string `json:"team"`
	Sides     string `json:"sides"`
	PerSeason string `json:"perSeason"`
	Now       string `json:"now"`
	Later     string `json:"later"`
	Picks     string `json:"picks"`
	Age       string `json:"age"`
	Partners  string `json:"partners"`
	Partner   string `json:"partner"`
	Day       string `json:"day"`
	Since     string `json:"since"`
	Legend    string `json:"legend"`
	Up        string `json:"up"`
	Down      string `json:"down"`
}

type Counterparties struct {
	Words     CounterWords  `json:"words"`
	CountText string        `json:"countText"`
	Summary   string        `json:"summary"`
	AsOf      string        `json:"asOf"`
	Headline  string        `json:"headline"`
	Note      string        `json:"note"`
	Picks     []TierPick    `json:"picks"`
	Teams     []TeamOption  `json:"teams"`
	Cards     []TierCard    `json:"cards"`
	TrendSay  []string      `json:"trendSay"`
	Trends    CounterTrends `json:"trends"`
	Pairs     []PairView    `json:"pairs"`
	Rows      []CounterRow  `json:"rows"`
}

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
	Excluded bool         `json:"excluded"`
	Tier     string       `json:"tier"`
	N        int          `json:"n"`
	Edge     lab.TeamEdge `json:"edge"`
	Label    string       `json:"label"`
	Text     string       `json:"text"`
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

func (r RawCounterpartiesScope) Validate() error {
	return counterpartiesapp.RawCounterpartiesScope(r).Validate() //nolint:wrapcheck // validation text is already contextual
}
func (a *App) counterpartiesService() counterpartiesapp.Service {
	return counterpartiesapp.Service{Load: a.lab, BuildStamp: buildLabel() + " " + buildDate, CountNumber: countNumber, CountLabel: countLabel, Percent: percent, MarketName: marketName}
}
func (a *App) TargetCounterparties(raw RawCounterpartiesScope) Reading[Counterparties] {
	r := a.counterpartiesService().TargetCounterparties(counterpartiesapp.RawCounterpartiesScope(raw))
	return wireCounterReading(r, wireCounterparties(r.Value))
}
func (a *App) TargetCounterpartTeam(raw RawCounterpartTeam) Reading[TeamCounterpart] {
	r := a.counterpartiesService().TargetCounterpartTeam(counterpartiesapp.RawCounterpartTeam{Scope: counterpartiesapp.RawCounterpartiesScope(raw.Scope), Fid: raw.Fid})
	return wireCounterReading(r, wireTeamCounterpart(r.Value))
}
func wireCounterReading[S, W any](r report.Reading[S], value W) Reading[W] {
	return Reading[W]{Sourced: snapshot.Sourced[W]{Value: value, Provenance: r.Provenance}, Status: ReadingStatus(r.Status), AbsenceReason: r.AbsenceReason, Count: r.Count, Threshold: r.Threshold, Baseline: r.Baseline, Cutoff: r.Cutoff, Method: r.Method, InputSHA: r.InputSHA, BuildStamp: r.BuildStamp, HowCounted: r.HowCounted}
}
func wireCounterSlice[S, W any](src []S, convert func(S) W) []W {
	if src == nil {
		return nil
	}
	dst := make([]W, len(src))
	for i := range src {
		dst[i] = convert(src[i])
	}
	return dst
}

func wireTierPick(v counterpartiesapp.TierPick) TierPick { return TierPick(v) }

func wireTeamOption(v counterpartiesapp.TeamOption) TeamOption { return TeamOption(v) }

func wirePostureCell(v counterpartiesapp.PostureCell) PostureCell { return PostureCell(v) }

func wireCardRow(v counterpartiesapp.CardRow) CardRow { return CardRow(v) }

func wireTierCard(v counterpartiesapp.TierCard) TierCard {
	return TierCard{ID: v.ID, Label: v.Label, Hint: v.Hint, Excluded: v.Excluded, Thin: v.Thin, PerSeason: v.PerSeason, PerSeasonTxt: v.PerSeasonTxt, Posture: wireCounterSlice(v.Posture, wirePostureCell), Rows: wireCounterSlice(v.Rows, wireCardRow)}
}

func wirePairView(v counterpartiesapp.PairView) PairView {
	return PairView{Other: v.Other, Excluded: v.Excluded, N: v.N, Now: v.Now, Real: v.Real, Trades: v.Trades, Title: v.Title}
}

func wireCounterShare(v counterpartiesapp.CounterShare) CounterShare { return CounterShare(v) }

func wireCounterRow(v counterpartiesapp.CounterRow) CounterRow {
	return CounterRow{DayText: v.DayText, SinceText: v.SinceText, NText: v.NText, PartnersText: v.PartnersText, Fid: v.Fid, Name: v.Name, N: v.N, PerSeason: v.PerSeason, Tiers: v.Tiers, ForNow: wireCounterShare(v.ForNow), ForLater: wireCounterShare(v.ForLater), Day: v.Day, Since: v.Since, Picks: v.Picks, Age: v.Age, Partner: v.Partner, PartnerName: v.PartnerName, PartnersCount: v.PartnersCount, StandingTitle: v.StandingTitle, PartnerTrades: v.PartnerTrades}
}

func wireCounterTrendRow(v counterpartiesapp.CounterTrendRow) CounterTrendRow {
	return CounterTrendRow{Season: v.Season, N: v.N, Net: v.Net, Now: wireCounterShare(v.Now), Later: wireCounterShare(v.Later), ShowShares: v.ShowShares, NText: v.NText}
}

func wireCounterShifts(v counterpartiesapp.CounterShifts) CounterShifts {
	return CounterShifts{Now: (*CounterShift)(v.Now), Later: (*CounterShift)(v.Later)}
}

func wireCounterTrend(v counterpartiesapp.CounterTrend) CounterTrend {
	return CounterTrend{Tier: v.Tier, Rows: wireCounterSlice(v.Rows, wireCounterTrendRow), Shift: wireCounterShifts(v.Shift)}
}

func wireCounterTrends(v counterpartiesapp.CounterTrends) CounterTrends {
	return CounterTrends{Seasons: v.Seasons, Tiers: wireCounterSlice(v.Tiers, wireCounterTrend)}
}

func wireCounterWords(v counterpartiesapp.CounterWords) CounterWords { return CounterWords(v) }

func wireCounterparties(v counterpartiesapp.Counterparties) Counterparties {
	return Counterparties{Words: wireCounterWords(v.Words), CountText: v.CountText, Summary: v.Summary, AsOf: v.AsOf, Headline: v.Headline, Note: v.Note, Picks: wireCounterSlice(v.Picks, wireTierPick), Teams: wireCounterSlice(v.Teams, wireTeamOption), Cards: wireCounterSlice(v.Cards, wireTierCard), TrendSay: v.TrendSay, Trends: wireCounterTrends(v.Trends), Pairs: wireCounterSlice(v.Pairs, wirePairView), Rows: wireCounterSlice(v.Rows, wireCounterRow)}
}

func wireSeasonCellView(v counterpartiesapp.SeasonCellView) SeasonCellView { return SeasonCellView(v) }

func wireKindFlowView(v counterpartiesapp.KindFlowView) KindFlowView { return KindFlowView(v) }

func wirePartnerView(v counterpartiesapp.PartnerView) PartnerView { return PartnerView(v) }

func wireAsTierView(v counterpartiesapp.AsTierView) AsTierView { return AsTierView(v) }

func wireTeamCounterpart(v counterpartiesapp.TeamCounterpart) TeamCounterpart {
	return TeamCounterpart{PartnersText: v.PartnersText, Words: wireCounterWords(v.Words), CountText: v.CountText, Fid: v.Fid, Name: v.Name, AsOf: v.AsOf, FlowScale: v.FlowScale, Sides: v.Sides, Empty: v.Empty, Note: v.Note, Seasons: wireCounterSlice(v.Seasons, wireSeasonCellView), SeasonSay: v.SeasonSay, Posture: wireCounterSlice(v.Posture, wirePostureCell), Kinds: wireCounterSlice(v.Kinds, wireKindFlowView), KindSay: v.KindSay, Partners: wireCounterSlice(v.Partners, wirePartnerView), PartnersSay: v.PartnersSay, Behaviour: v.Behaviour, AsTier: wireCounterSlice(v.AsTier, wireAsTierView), Day: v.Day, Since: v.Since, DayTxt: v.DayTxt, SinceTxt: v.SinceTxt}
}

func counterProvenance(d *lab.Data) snapshot.Provenance {
	return (counterpartiesapp.Service{}).Provenance(d)
}
