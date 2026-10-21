package trades

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// The wire types keep the names, fields, JSON tags and nil behaviour they had in package main.

type TradeAsset struct {
	Team     string          `json:"team,omitempty"`
	Contract string          `json:"contract,omitempty"`
	Now      *Cell           `json:"now,omitempty"`
	Dynasty  *Cell           `json:"dynasty,omitempty"`
	Token    string          `json:"token"`
	Name     string          `json:"name"`
	Position domain.Position `json:"position,omitempty"`
}

type TradeOffer struct {
	TradeID   string       `json:"tradeId"`
	Direction string       `json:"direction"`
	OtherID   string       `json:"otherId"`
	OtherName string       `json:"otherName"`
	Give      []TradeAsset `json:"give"`
	Get       []TradeAsset `json:"get"`
	Expires   time.Time    `json:"expires"`
	Comments  string       `json:"comments"`
}

type TradeReading struct {
	Offers     []TradeOffer        `json:"offers"`
	Provenance snapshot.Provenance `json:"provenance"`
}

type RawTradeBuilder struct {
	Mine      string   `json:"mine"`
	Other     string   `json:"other"`
	Action    string   `json:"action"`
	ID        string   `json:"id"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Rationale string   `json:"rationale"`
}

func (r RawTradeBuilder) Validate() error { return Input(r).Validate() }

type TradeBuilderTeam struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Assets []TradeAsset `json:"assets"`
}
type TradeProjection struct {
	Team    string       `json:"team"`
	Figures []Figure     `json:"figures"`
	Roster  []TradeAsset `json:"roster"`
}
type TradeBuilderPlan struct {
	ID      string            `json:"id"`
	Post    string            `json:"post"`
	Reasons []string          `json:"reasons"`
	Resets  []TradeReset      `json:"resets,omitempty"`
	Teams   []TradeProjection `json:"teams"`
	Receipt *envelope.Receipt `json:"receipt,omitempty"`
}

// TradeReset offers the Reset for a what-if MFL has moved past, so the deal is checked afresh.
type TradeReset struct {
	Franchise string `json:"franchise"`
	Label     string `json:"label"`
}

type TradeBuilderView struct {
	Teams   []TradeBuilderTeam `json:"teams"`
	Plans   []TradeBuilderPlan `json:"plans"`
	Preview *TradeBuilderPlan  `json:"preview,omitempty"`
}

// TradeBuilderReading is concrete so Wails discovers the nested builder models.
type TradeBuilderReading report.Reading[TradeBuilderView]

func WireAssets(in []Asset) []TradeAsset {
	out := make([]TradeAsset, len(in))
	for i, v := range in {
		out[i] = TradeAsset{Team: v.Team, Contract: v.Contract, Now: cloneCell(v.Now), Dynasty: cloneCell(v.Dynasty),
			Token: v.Token, Name: v.Name, Position: v.Position}
	}
	return keepNil(in, out)
}
func serviceAssets(in []TradeAsset) []Asset {
	out := make([]Asset, len(in))
	for i, v := range in {
		out[i] = Asset{Team: v.Team, Contract: v.Contract, Now: cloneCell(v.Now), Dynasty: cloneCell(v.Dynasty),
			Token: v.Token, Name: v.Name, Position: v.Position}
	}
	return keepNil(in, out)
}
func WireTeams(in []Team) []TradeBuilderTeam {
	out := make([]TradeBuilderTeam, len(in))
	for i, v := range in {
		out[i] = TradeBuilderTeam{ID: v.ID, Name: v.Name, Assets: WireAssets(v.Assets)}
	}
	return keepNil(in, out)
}
func ServiceTeams(in []TradeBuilderTeam) []Team {
	out := make([]Team, len(in))
	for i, v := range in {
		out[i] = Team{ID: v.ID, Name: v.Name, Assets: serviceAssets(v.Assets)}
	}
	return keepNil(in, out)
}
func WireProjection(in Projection) TradeProjection {
	figures := make([]Figure, len(in.Figures))
	copy(figures, in.Figures)
	return TradeProjection{Team: in.Team, Figures: keepNil(in.Figures, figures), Roster: WireAssets(in.Roster)}
}
func WireReading(in Reading) TradeReading {
	offers := make([]TradeOffer, len(in.Offers))
	for i, v := range in.Offers {
		offers[i] = TradeOffer{TradeID: v.TradeID, Direction: v.Direction, OtherID: v.OtherID, OtherName: v.OtherName,
			Give: WireAssets(v.Give), Get: WireAssets(v.Get), Expires: v.Expires, Comments: v.Comments}
	}
	return TradeReading{Offers: keepNil(in.Offers, offers), Provenance: in.Provenance}
}

func cloneCell(c *Cell) *Cell {
	if c == nil {
		return nil
	}
	cell := *c
	return &cell
}

// keepNil keeps a nil slice nil, so it still encodes as null rather than [].
func keepNil[S, T any](in []S, out []T) []T {
	if in == nil {
		return nil
	}
	return out
}
