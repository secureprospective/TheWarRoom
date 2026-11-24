package mflmirror

import (
	"context"
	"encoding/json"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// Seat is everything one owner's bid, lineup and plan work runs on: the held copy, the owner's picked MFL
// team, the league's rules and the league database the plans live in. The league service builds it from
// the authenticated seat; the engine never sees a request's franchise.
type Seat struct {
	League    string // the app league
	Franchise string // the seated app franchise; every plan is keyed by League and Franchise
	MFLLeague string
	Team      string // the owner's picked MFL team; empty when plans are only listed
	Held      Held
	Lookup    Directory // nil when no player directory is held
	Pools     *db.Pools
	Now       time.Time
	Rules     leaguerules.RuleSet
	RulesOK   bool
}

// Engine judges bids and lineups and keeps plans. Its implementation uses the MFL reference packages, which
// the native league service may not import, so it is reached through the fetcher the host supplies.
type Engine interface {
	BidCheck(ctx context.Context, s Seat, in BidInput) BidCheck
	LineupCheck(ctx context.Context, s Seat, in LineupInput) (LineupCheck, error)
	PlanSave(ctx context.Context, s Seat, kind string, body json.RawMessage) (Plan, error)
	PlansRead(ctx context.Context, s Seat) (Plans, error)
	PlanLink(ctx context.Context, s Seat, plan string) (PlanLink, error)
	PlanDone(ctx context.Context, s Seat, plan string) (Plan, error)
	PlanWithdraw(ctx context.Context, s Seat, plan string) (Plan, error)
}

// BidInput is one bid to judge: the player to add, the one annual salary in cents, the years, and the
// players to drop to make room.
type BidInput struct {
	Add         string
	AmountCents domain.Money
	Years       int
	Drop        []string
}

// LineupInput is one week's starters as MFL player ids.
type LineupInput struct {
	Week     int
	Starters []string
}

// BidCheck is the plain-word verdict on one bid. OK is true only when Problems is empty.
type BidCheck struct {
	League        string        `json:"league"`
	Player        string        `json:"player"`
	Salary        domain.Money  `json:"salaryCents"`
	Years         int           `json:"years"`
	OK            bool          `json:"ok"`
	Problems      []string      `json:"problems"`
	NotChecked    []string      `json:"notChecked"`
	CapUsedBefore domain.Money  `json:"capUsedBeforeCents"`
	CapUsedAfter  domain.Money  `json:"capUsedAfterCents"`
	CapRoomAfter  *domain.Money `json:"capRoomAfterCents,omitempty"`
	ActiveAfter   int           `json:"activeAfter"`
	Words         string        `json:"words"`
}

// LineupProblem is one thing wrong with a lineup, in words.
type LineupProblem struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// LineupCheck is the plain-word verdict on a lineup. OK is true when MFL would accept it: nothing over a
// limit, nobody on bye, injured reserve, taxi squad or off your roster. A partial lineup (a slot short) is
// OK but not Full.
type LineupCheck struct {
	League   string          `json:"league"`
	Week     int             `json:"week"`
	OK       bool            `json:"ok"`
	Full     bool            `json:"full"`
	Problems []LineupProblem `json:"problems"`
	Words    string          `json:"words"`
}

// Plan is one saved plan in words. Nothing here was sent to MFL.
type Plan struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	State       string       `json:"state"`
	Words       string       `json:"words"`
	Player      string       `json:"player,omitempty"`
	PlayerName  string       `json:"playerName,omitempty"`
	Salary      domain.Money `json:"salaryCents,omitempty"`
	Years       int          `json:"years,omitempty"`
	Points      string       `json:"points,omitempty"`
	Drop        string       `json:"drop,omitempty"`
	DropName    string       `json:"dropName,omitempty"`
	Snipe       bool         `json:"snipe,omitempty"`
	Week        int          `json:"week,omitempty"`
	Starters    []string     `json:"starters,omitempty"`
	CanDone     bool         `json:"canDone"`
	CanWithdraw bool         `json:"canWithdraw"`
}

type Plans struct {
	League string `json:"league"`
	Plans  []Plan `json:"plans"`
	Words  string `json:"words"`
}

// PlanLink is the MFL page for a plan. The browser opens it; the server never calls it.
type PlanLink struct {
	League string `json:"league"`
	Plan   string `json:"plan"`
	URL    string `json:"url"`
	Words  string `json:"words"`
}

// PlanNotFound is what a missing plan and another seat's plan both read.
const PlanNotFound = "That plan was not found."

// Refusal marks a safe, owner-facing domain refusal (not an internal storage or host failure).
type Refusal struct{ msg string }

func (r Refusal) Error() string { return r.msg }

// Refused returns a safe, owner-facing refusal carrying msg.
func Refused(msg string) error { return Refusal{msg: msg} }
