package commissioner

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const bidTimeLayout = "Jan 2, 2006 at 3:04 PM MST"

// NativeBidRow is one bid in plain words, with the time it was placed in the viewer's zone.
type NativeBidRow struct {
	ID        string    `json:"id"`
	Franchise string    `json:"franchise"`
	Team      string    `json:"team"`
	Contract  string    `json:"contract"`
	Points    string    `json:"points"`
	PlacedAt  time.Time `json:"placedAt"`
	Placed    string    `json:"placed"`
}

// NativeBidPlayer is a free agent with bids the viewer may see. In blind bidding that is the viewer's own bid
// alone; in open bidding it is the leading bid too, with the late-bid countdown.
type NativeBidPlayer struct {
	Player    string        `json:"player"`
	Name      string        `json:"name"`
	Mode      string        `json:"mode"`
	ModeWords string        `json:"modeWords"`
	Leader    *NativeBidRow `json:"leader,omitempty"`
	Bids      int           `json:"bids,omitempty"`
	YourBid   *NativeBidRow `json:"yourBid,omitempty"`
	YouLead   bool          `json:"youLead"`
	LateAt    time.Time     `json:"lateAt,omitempty"`
	LateWords string        `json:"lateWords,omitempty"`
	LateNow   bool          `json:"lateNow"`
	ClosesAt  *time.Time    `json:"closesAt,omitempty"`
	Closes    string        `json:"closes"`
}

// NativeBidAward is a decided player. Awards are public; a losing bid is shown only to its owner.
type NativeBidAward struct {
	Player   string    `json:"player"`
	Name     string    `json:"name"`
	Winner   string    `json:"winner,omitempty"`
	Team     string    `json:"team,omitempty"`
	Contract string    `json:"contract,omitempty"`
	Points   string    `json:"points,omitempty"`
	Decided  time.Time `json:"decided"`
	Words    string    `json:"words"`
	Yours    string    `json:"yours,omitempty"`
}

type NativeBids struct {
	League      string            `json:"league"`
	Franchise   string            `json:"franchise"`
	Revision    int64             `json:"revision"`
	Season      int               `json:"season"`
	Mode        string            `json:"mode"`
	ActiveBids  int               `json:"activeBids"`
	ActiveLimit string            `json:"activeLimit"`
	Players     []NativeBidPlayer `json:"players"`
	Awards      []NativeBidAward  `json:"awards"`
	Notes       []string          `json:"notes"`
}

func bidStandFor(rules leaguerules.RuleSet) leaguerules.Value[time.Duration] {
	return rules.Bids.StandFor
}

// bidClose dates the end of an auction from its standing bid, or says the rule is missing.
func bidClose(stand leaguerules.Value[time.Duration], standingAt time.Time, zone *time.Location) (*time.Time, string) {
	if stand.State != leaguerules.IsKnown || stand.Value <= 0 {
		return nil, "Needs a ruling: how long a bid must stand to win"
	}
	at := standingAt.Add(stand.Value).In(zone)
	return &at, "Closes " + at.Format(bidTimeLayout)
}

type bidViewer struct {
	franchise string
	zone      *time.Location
	teams     seatIndex
	facts     nativecheck.Facts
	decimals  int
	// open is whether the league shows every owner the leading bid. It is read from the rule in force now,
	// so a commissioner's change to the rule governs bids already placed.
	open bool
}

func (v bidViewer) row(b bidRecord) *NativeBidRow {
	return &NativeBidRow{ID: b.ID, Franchise: b.Franchise, Team: v.teams.of(b.Franchise).Name,
		Contract: nativecheck.ContractWords(b.Salary, b.Years),
		Points:   nativecheck.FormatPoints(b.Points, v.decimals), PlacedAt: b.PlacedAt.In(v.zone),
		Placed: b.PlacedAt.In(v.zone).Format(bidTimeLayout)}
}

func (v bidViewer) name(player string) string {
	if n := v.facts[player].Name; n != "" {
		return n
	}
	return "Player " + player
}

// NativeBidsReading shows one franchise's owner the bids they may see. A blind bid is never shown to another
// owner, and neither is the fact that it exists.
func NativeBidsReading(league, franchise, principal string, records []Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (NativeBids, error) {
	if err := AuthorizeFranchiseOwner(records, principal, franchise); err != nil {
		return NativeBids{}, err
	}
	f, err := marketFrame(league, records, at, facts, zone, false)
	if err != nil {
		return NativeBids{}, err
	}
	return nativeBidsFrame(f, franchise, principal)
}

func nativeBidsFrame(f Frame, franchise, principal string) (NativeBids, error) {
	league, records, at, zone, facts := f.League, f.Records, f.At, f.Zone, f.Facts
	assets, state := f.Assets, f.Book
	if err := f.readRules(); err != nil {
		return NativeBids{}, err
	}
	rules := f.Rules
	out := NativeBids{League: league, Franchise: franchise, Revision: RecipientRevision(records, principal, at), Season: rules.Season,
		Mode: fieldWords(rules, "bids.ufa"), ActiveLimit: fieldWords(rules, "bids.maximumActive"),
		Players: []NativeBidPlayer{}, Awards: []NativeBidAward{}, Notes: []string{}}
	if !assets.Seeded {
		out.Notes = append(out.Notes, "No starting rosters recorded")
		return out, nil
	}
	directory, err := NativeDirectoryReading(league, records, at)
	if err != nil {
		return NativeBids{}, err
	}
	v := bidViewer{franchise: franchise, zone: zone, teams: directory.seats(), facts: facts, decimals: 4}
	if d := rules.Bids.PointDecimals; d.State == leaguerules.IsKnown && d.Value >= 0 && d.Value <= 4 {
		v.decimals = d.Value
	}
	policy, policyErr := nativecheck.NewBidPolicy(rules)
	if policyErr != nil {
		out.Notes = append(out.Notes, "Needs a ruling: "+policyErr.Error())
	}
	mode, modeErr := nativecheck.BidMode(rules.Bids, leaguerules.UFAProcess)
	if modeErr != nil {
		out.Notes = append(out.Notes, "Needs a ruling: "+modeErr.Error())
	}
	v.open = modeErr == nil && mode == leaguerules.Open
	stand := bidStandFor(rules)
	out.ActiveBids = state.Bids.activeCount(franchise, "")
	for i := range state.Bids.Auctions {
		a := &state.Bids.Auctions[i]
		if a.Decided {
			out.Awards = append(out.Awards, v.award(a))
			continue
		}
		viewer := v.forPlayer(records, a.Player)
		if row, ok := viewer.player(a, at, policy, policyErr == nil, stand); ok {
			out.Players = append(out.Players, row)
		}
	}
	out.Players = append(out.Players, unbidStartingLots(assets, state, records)...)
	slices.SortFunc(out.Players, func(a, b NativeBidPlayer) int { return strings.Compare(a.Player, b.Player) })
	return out, nil
}

// player is the open auction as this viewer may see it.
func (v bidViewer) player(a *bidAuction, at time.Time, policy nativecheck.BidPolicy, haveRules bool,
	stand leaguerules.Value[time.Duration]) (NativeBidPlayer, bool) {
	own, ownLive := a.activeFor(v.franchise)
	var open []bidRecord
	if v.open {
		open = a.liveBids()
	}
	if !ownLive && len(open) == 0 {
		return NativeBidPlayer{}, false
	}
	row := NativeBidPlayer{Player: a.Player, Name: v.name(a.Player), Mode: string(leaguerules.Blind),
		ModeWords: "Blind bidding: only you can see your bid"}
	if ownLive {
		row.YourBid = v.row(own)
	}
	if lead, ok := leaderOf(open); ok {
		row.Mode, row.ModeWords = string(leaguerules.Open), "Open bidding: everyone sees the leading bid"
		row.Leader, row.Bids = v.row(lead), len(open)
		row.YouLead = lead.Franchise == v.franchise
		if haveRules {
			row.LateAt = lead.PlacedAt.Add(policy.LateAfter()).In(v.zone)
			row.LateNow = at.After(row.LateAt)
			row.LateWords = lateWords(policy, row.LateAt, row.LateNow)
		}
		row.ClosesAt, row.Closes = bidClose(stand, a.standingAt(), v.zone)
	} else {
		row.Closes = "Closing time is not shown in blind bidding"
	}
	return row, true
}

func lateWords(p nativecheck.BidPolicy, lateAt time.Time, now bool) string {
	step := p.PointsText(p.LateIncrement())
	if now {
		return fmt.Sprintf("Late bids now: a new bid must beat the leader by %s points.", step)
	}
	return fmt.Sprintf("A bid placed after %s must beat the leader by %s points.", lateAt.Format(bidTimeLayout), step)
}

func (v bidViewer) award(a *bidAuction) NativeBidAward {
	out := NativeBidAward{Player: a.Player, Name: v.name(a.Player), Decided: a.DecidedAt.In(v.zone)}
	switch {
	case a.Winner != "":
		won := a.Bids[slices.IndexFunc(a.Bids, func(b bidRecord) bool { return b.ID == a.WinningBid })]
		out.Winner, out.Team = a.Winner, v.teams.of(a.Winner).Name
		out.Contract, out.Points = nativecheck.ContractWords(won.Salary, won.Years), nativecheck.FormatPoints(won.Points, v.decimals)
		out.Words = fmt.Sprintf("Awarded to %s for %s", out.Team, out.Contract)
	case len(a.Tied) > 0:
		out.Words = "The top bids tied, and the rules name no tie-break, so the commissioner decides"
	default:
		out.Words = "No bid could be awarded, so he is a free agent again"
	}
	if own, ok := a.lastBidOf(v.franchise); ok && own.ID != a.WinningBid {
		out.Yours = "Your bid was not awarded"
		for _, r := range a.Results {
			if r.ID == own.ID && r.Refusal != "" {
				out.Yours = "Your bid could not be awarded: " + r.Refusal
			}
		}
	}
	return out
}

// lastBidOf is the franchise's final bid on the auction, standing or not.
func (a *bidAuction) lastBidOf(franchise string) (bidRecord, bool) {
	for i := len(a.Bids) - 1; i >= 0; i-- {
		if a.Bids[i].Franchise == franchise && !a.Bids[i].Withdrawn {
			return a.Bids[i], true
		}
	}
	return bidRecord{}, false
}

func unbidStartingLots(assets NativeAssets, state waiverState, records []Record) []NativeBidPlayer {
	rows := []NativeBidPlayer{}
	for _, lot := range assets.Snapshot.Pool {
		if startingCardLot(records, lot.Player) && state.Bids.latest(lot.Player) == nil {
			rows = append(rows, NativeBidPlayer{Player: lot.Player, Name: lot.Name, Mode: string(leaguerules.Open), ModeWords: "League-start open auction", Closes: "Awaiting first bid"})
		}
	}
	return rows
}

func (v bidViewer) forPlayer(records []Record, player string) bidViewer {
	if startingCardLot(records, player) {
		v.open = true
	}
	return v
}
