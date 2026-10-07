// Package leaguefeed holds source-independent season feed records.
package leaguefeed

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// CurrentPick is a current-year draft pick, actual round and pick (MFL's token is zero-based).
type CurrentPick struct {
	Round int `json:"round"`
	Pick  int `json:"pick"`
}

// FuturePick is a later year's pick originally owned by Franchise; Round is actual.
type FuturePick struct {
	Franchise string `json:"franchise"`
	Year      int    `json:"year"`
	Round     int    `json:"round"`
}

// ParseAssets populates exactly one variant; money is exact cents, not floating point.
type Asset struct {
	Player        *playerid.PlayerID `json:"player,omitempty"`
	CurrentPick   *CurrentPick       `json:"currentPick,omitempty"`
	FuturePick    *FuturePick        `json:"futurePick,omitempty"`
	BlindBidCents *int64             `json:"blindBidCents,omitempty"`
}

// Trade is from the transaction's Franchise side: Gave is what it gave up.
type Trade struct {
	Counterparty     string    `json:"counterparty"`
	Gave             []Asset   `json:"gave"`
	CounterpartyGave []Asset   `json:"counterpartyGave"`
	Comments         string    `json:"comments"`
	Expires          time.Time `json:"expires"`
}

// RosterChange: In is added, activated or promoted; Out is dropped, deactivated or demoted.
type RosterChange struct {
	In  []Asset `json:"in"`
	Out []Asset `json:"out"`
}

// Transaction keeps a kind it does not know with Unparsed set, so the feed never fails on one.
type Transaction struct {
	Kind      string        `json:"kind"`
	Time      time.Time     `json:"time"`
	Franchise string        `json:"franchise"`
	ByCommish bool          `json:"byCommish"`
	Trade     *Trade        `json:"trade,omitempty"`
	AddsDrops *RosterChange `json:"addsDrops,omitempty"`
	IR        *RosterChange `json:"ir,omitempty"`
	Taxi      *RosterChange `json:"taxi,omitempty"`
	Unparsed  bool          `json:"unparsed"`
}

type Lineup struct {
	Franchise        string              `json:"franchise"`
	Starters         []playerid.PlayerID `json:"starters"`
	NonStarters      []playerid.PlayerID `json:"nonStarters"`
	Score            float64             `json:"score"`
	SecondsRemaining int                 `json:"secondsRemaining"`
}

type Lineups struct {
	Week       int      `json:"week"`
	Franchises []Lineup `json:"franchises"`
}

// PendingTrade is from the offering team's side: Gives is what Offering gives up.
type PendingTrade struct {
	ID          string    `json:"id"`
	Offering    string    `json:"offering"`
	OfferedTo   string    `json:"offeredTo"`
	Gives       []Asset   `json:"gives"`
	Gets        []Asset   `json:"gets"`
	Proposed    time.Time `json:"proposed"`
	Expires     time.Time `json:"expires"`
	Comments    string    `json:"comments"`
	Description string    `json:"description"`
}
