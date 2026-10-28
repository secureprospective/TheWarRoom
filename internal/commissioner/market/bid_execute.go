package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// BidEnvironment is what the NFL season says at the moment of a bid: the week in play and which franchises
// are in the playoffs. The app supplies it from held data; a franchise it does not list is unknown.
type BidEnvironment struct {
	Season  nativecheck.BidSeason
	Playoff map[string]nativecheck.PlayoffStanding
}

type bidExecution struct {
	assets  commissioner.NativeAssets
	state   commissioner.MarketBook
	rules   leaguerules.RuleSet
	records []commissioner.Record
	now     time.Time
	facts   nativecheck.Facts
	zone    *time.Location
	env     BidEnvironment
}

// PrepareBidAppend judges a bid action against the transaction's current records and writes the app's
// decision onto the command. Owner actions are checked purely against the bid rules, the cap and roster
// legality; the commissioner's commands decide a player's bids by those same rules.
func PrepareBidAppend(raw *commissioner.RawCommand, records []commissioner.Record, now time.Time, facts nativecheck.Facts,
	zone *time.Location, env BidEnvironment) error {
	if !commissioner.BidCommand(raw.Command) {
		return nil
	}
	f, err := commissioner.NewFrame(raw.League, records, now, facts, zone)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	return prepareBidFrame(f, raw, env)
}

func prepareBidFrame(f commissioner.Frame, raw *commissioner.RawCommand, env BidEnvironment) error {
	records, now, facts, zone := f.Records, f.At, f.Facts, f.Zone
	if err := f.Seeded("record starting assets before bid actions"); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	x := bidExecution{assets: f.Assets, state: f.Book, rules: f.Rules, records: records, now: now, facts: facts, zone: zone, env: env}
	if action := commissioner.OwnerBidAction(raw.Command); action != "" && !commissioner.StartingCardLot(records, commissioner.BidPlayerOf(commissioner.Record{Command: *raw}, records)) {
		if err := f.Gate(raw.Args.Bid.Franchise, action); err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
	}
	if err := x.prepare(raw); err != nil {
		return err
	}
	if _, err := f.Trial(*raw, "proposed"); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	return nil
}

func (x *bidExecution) prepare(raw *commissioner.RawCommand) error {
	if _, card := nativecheck.LineCardTeam(commissioner.BidPlayerOf(commissioner.Record{Command: *raw}, x.records)); x.state.OLine != nil && card {
		return fmt.Errorf("the starting O-Line auction is closed")
	}
	switch raw.Command {
	case commissioner.BidPlace:
		return x.place(raw)
	case commissioner.BidRaise:
		return x.raise(raw)
	case commissioner.BidWithdraw:
		return x.withdraw(raw)
	}
	return x.decide(raw)
}

func (x *bidExecution) name(player string) string {
	if n := x.facts[player].Name; n != "" {
		return n
	}
	return "Player " + player
}

func (x *bidExecution) place(raw *commissioner.RawCommand) error {
	args := raw.Args.Bid
	if _, known := x.facts[args.Player]; !known {
		return fmt.Errorf("choose a player the app holds")
	}
	if !slices.ContainsFunc(x.assets.Snapshot.Franchises, func(f commissioner.NativeFranchise) bool { return f.ID == args.Franchise }) {
		return fmt.Errorf("choose a franchise in this app league")
	}
	if err := x.bidBar(args.Player, args.Franchise); err != nil {
		return err
	}
	auction := x.state.Bids.Open(x.assets.Snapshot.SideOf(args.Franchise), args.Player)
	if auction != nil {
		if _, held := auction.ActiveFor(args.Franchise); held {
			return fmt.Errorf("you already have a bid on %s; raise it instead", x.name(args.Player))
		}
	}
	outcome, err := x.judge(args.Player, args.Franchise, auction, nativecheck.BidProposal{Salary: args.Salary, Years: args.Years})
	if err != nil {
		return err
	}
	raw.Args.BidOutcome = outcome
	return nil
}

func (x *bidExecution) raise(raw *commissioner.RawCommand) error {
	args := raw.Args.Bid
	auction, err := x.standingBid(args)
	if err != nil {
		return err
	}
	outcome, err := x.judge(auction.Player, args.Franchise, auction, nativecheck.BidProposal{Salary: args.Salary, Years: args.Years})
	if err != nil {
		return err
	}
	raw.Args.BidOutcome = outcome
	return nil
}

func (x *bidExecution) withdraw(raw *commissioner.RawCommand) error {
	args := raw.Args.Bid
	auction, err := x.standingBid(args)
	if err != nil {
		return err
	}
	if err := x.auctionOpen(auction); err != nil {
		return err
	}
	raw.Args.BidOutcome = &commissioner.BidOutcome{Player: auction.Player, Season: x.rules.Season}
	return nil
}

func (x *bidExecution) standingBid(args *commissioner.RawBidArgs) (*commissioner.BidAuction, error) {
	a, j, err := x.state.Bids.Find(args.Bid)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	if a.Decided || !a.Bids[j].Live() || a.Bids[j].Franchise != args.Franchise {
		return nil, fmt.Errorf("choose your own bid that is still standing")
	}
	return a, nil
}

// bidBar refuses a player who cannot be bid on: one on a roster, one still on waivers, or one this franchise
// released this season when the league bars re-signing him.
func (x *bidExecution) bidBar(player, franchise string) error {
	side := x.assets.Snapshot.SideOf(franchise)
	if x.assets.Snapshot.OnSide(side, player) {
		return fmt.Errorf("%s is on a roster; only free agents can be bid on", x.name(player))
	}
	for _, w := range x.state.Windows {
		if w.Player.ID != player || w.Side != side {
			continue
		}
		if !w.Processed {
			return fmt.Errorf("%s is on waivers, and bids open once he clears them", x.name(player))
		}
		if w.Player.Franchise != franchise || w.Season != x.rules.Season {
			continue
		}
		if x.rules.Contracts.ReleaseReSignSameSeason.State != leaguerules.IsKnown {
			return fmt.Errorf("re-signing a released player needs a ruling")
		}
		if !x.rules.Contracts.ReleaseReSignSameSeason.Value {
			return fmt.Errorf("you released %s this season, so you cannot sign him again", x.name(player))
		}
	}
	return nil
}

// judge runs the bid rules, then the roster and the cap, and returns what the app records about the bid.
func (x *bidExecution) judge(player, franchise string, auction *commissioner.BidAuction, p nativecheck.BidProposal) (*commissioner.BidOutcome, error) {
	if err := x.auctionOpen(auction); err != nil {
		return nil, err
	}
	rules := x.rules
	startCard := commissioner.StartingCardLot(x.records, player)
	if startCard {
		if p.Years != 1 {
			return nil, fmt.Errorf("a starting O-Line card has a 1-year contract")
		}
		rules.Bids.UFA = leaguerules.Known(leaguerules.Open)
	}
	policy, err := nativecheck.NewBidPolicy(rules)
	if err != nil {
		return nil, fmt.Errorf("read bid rules: %w", err)
	}
	mode, err := nativecheck.BidMode(rules.Bids, leaguerules.UFAProcess)
	if err != nil {
		return nil, fmt.Errorf("read bid mode: %w", err)
	}
	facts := nativecheck.BidFacts{Now: x.now, ExperienceYears: experienceYears(x.rules.Season, x.facts[player]),
		Season: x.env.Season, Playoff: x.env.Playoff[franchise], LateRule: mode == leaguerules.Open}
	if startCard {
		facts.Season = nativecheck.BidSeason{Offseason: true}
	}
	facts.Active = x.state.Bids.ActiveCount(franchise, player)
	if auction != nil {
		facts.Standing = standingBid(auction, mode, franchise)
	}
	verdict := policy.Check(p, facts)
	if verdict.Refusal != "" {
		return nil, fmt.Errorf("%s", verdict.Refusal)
	}
	if err := x.rosterAllowsContract(player, franchise, p.Salary, p.Years); err != nil {
		return nil, err
	}
	return &commissioner.BidOutcome{Player: player, Season: x.rules.Season, Points: verdict.Points, Late: verdict.Late}, nil
}

// standingBid is the bid a new bid must beat. Open bidding shows the leader to everyone; a blind bid can only
// be held to the owner's own earlier bid, since nothing else may be revealed.
func standingBid(a *commissioner.BidAuction, mode leaguerules.BidMode, franchise string) *nativecheck.StandingBid {
	if mode == leaguerules.Blind {
		if own, ok := a.ActiveFor(franchise); ok {
			return &nativecheck.StandingBid{Points: own.Points, PlacedAt: own.PlacedAt}
		}
		return nil
	}
	if lead, ok := commissioner.LeaderOf(a.LiveBids()); ok {
		return &nativecheck.StandingBid{Points: lead.Points, PlacedAt: lead.PlacedAt}
	}
	return nil
}

func (x *bidExecution) rosterAllowsContract(player, franchise string, salary domain.Money, years int) error {
	candidate := *x.assets.Snapshot
	candidate.Players = append(slices.Clone(candidate.Players), commissioner.NativePlayer{ID: player, Franchise: franchise,
		Status: domain.RosterActive, Salary: salary, ExpirationYear: x.rules.Season + years - 1,
		ContractStatus: domain.CStatusUFA})
	rules := x.rules
	if commissioner.StartingCardLot(x.records, player) {
		rules.Roster.Limits.Value.Min = leaguerules.Known(0)
	}
	return rosterCapAllows(&candidate, rules, franchise, x.facts, x.zone, commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: franchise, Player: player}, x.now)...)
}

// experienceYears is the player's NFL seasons so far. Without a real draft year the player is a rookie.
func experienceYears(season int, f nativecheck.PlayerFacts) int {
	const longestCareer = 30
	if f.DraftYearHeld && f.DraftYear >= season-longestCareer && f.DraftYear <= season {
		return season - f.DraftYear
	}
	return 0
}

// decide chooses the winner of a player's bids, or corrects the choice. A bid is judged again at the moment of
// the decision, because the franchise's roster and cap may have changed since it was made.
func (x *bidExecution) decide(raw *commissioner.RawCommand) error {
	args := raw.Args.Bid
	var auction *commissioner.BidAuction
	if raw.Command == commissioner.BidsProcess {
		if err := x.sideNamed(args.Side); err != nil {
			return err
		}
		if auction = x.state.Bids.Open(args.Side, args.Player); auction == nil {
			return fmt.Errorf("%s has no bids waiting for a decision", x.name(args.Player))
		}
	} else {
		found, _, err := x.state.Bids.Find(args.Bid)
		if err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
		if auction = found; !auction.Decided || x.state.Bids.Latest(auction.Side, auction.Player) != auction {
			return fmt.Errorf("correct only the newest decision on a player")
		}
		// The corrected decision is tested without counting the previous winner twice.
		seed := x.assets.Snapshot
		x.assets.Snapshot.Players = slices.DeleteFunc(slices.Clone(seed.Players), func(p commissioner.NativePlayer) bool {
			return p.ID == auction.Player && seed.SideOf(p.Franchise) == auction.Side
		})
	}
	if err := x.clearsWaivers(auction.Side, auction.Player); err != nil {
		return err
	}
	live := auction.LiveBids()
	if len(live) == 0 {
		return fmt.Errorf("every bid on %s has been withdrawn", x.name(auction.Player))
	}
	eligible, results := x.eligibleBids(auction.Player, live)
	outcome := &commissioner.BidOutcome{Player: auction.Player, Season: x.rules.Season, Results: results}
	if raw.Command == commissioner.BidsOverride {
		return x.overrideAward(raw, auction, eligible, outcome)
	}
	award := nativecheck.ChooseBidWinner(eligible)
	if award.Won {
		outcome.Winner = franchiseOf(live, award.Winner.ID)
		outcome.WinningBid = award.Winner.ID
	}
	for _, tied := range award.Tied {
		outcome.Tied = append(outcome.Tied, tied.ID)
	}
	raw.Args.BidOutcome = outcome
	return nil
}

// sideNamed holds a decision in a league with a pool per side to one of its sides, and one with one pool
// to none.
func (x *bidExecution) sideNamed(side string) error {
	sides := x.assets.Snapshot.SideNames()
	if side == "" && len(sides) != 0 {
		return fmt.Errorf("each side has its own copy of this player: name the side whose bids are decided")
	}
	if side != "" && !slices.Contains(sides, side) {
		return fmt.Errorf("choose a side of this league")
	}
	return nil
}

func (x *bidExecution) overrideAward(raw *commissioner.RawCommand, a *commissioner.BidAuction, eligible []nativecheck.AwardBid, outcome *commissioner.BidOutcome) error {
	winner := raw.Args.Bid.Franchise
	bid, held := a.ActiveFor(winner)
	if !held || !slices.ContainsFunc(eligible, func(e nativecheck.AwardBid) bool { return e.ID == bid.ID }) {
		return fmt.Errorf("the corrected winner must have a legal bid on this player")
	}
	if a.Winner == winner {
		return fmt.Errorf("that franchise already holds this award")
	}
	outcome.Winner, outcome.WinningBid = winner, bid.ID
	raw.Args.BidOutcome = outcome
	return nil
}

// clearsWaivers refuses to decide bids on a player who is still on waivers.
func (x *bidExecution) clearsWaivers(side, player string) error {
	for _, w := range x.state.Windows {
		if w.Player.ID == player && w.Side == side && !w.Processed {
			return fmt.Errorf("%s is still on waivers, and bids are decided once he clears them", x.name(player))
		}
	}
	return nil
}

func (x *bidExecution) eligibleBids(player string, live []commissioner.BidRecord) ([]nativecheck.AwardBid, []commissioner.BidResult) {
	eligible := []nativecheck.AwardBid{}
	results := make([]commissioner.BidResult, 0, len(live))
	for _, b := range live {
		result := commissioner.BidResult{ID: b.ID}
		if err := x.rosterAllowsContract(player, b.Franchise, b.Salary, b.Years); err != nil {
			result.Refusal = err.Error()
		} else {
			eligible = append(eligible, nativecheck.AwardBid{ID: b.ID, Franchise: b.Franchise, Points: b.Points})
		}
		results = append(results, result)
	}
	return eligible, results
}

func franchiseOf(bids []commissioner.BidRecord, id string) string {
	for _, b := range bids {
		if b.ID == id {
			return b.Franchise
		}
	}
	return ""
}

func (x *bidExecution) auctionOpen(auction *commissioner.BidAuction) error {
	if auction == nil {
		return nil
	}
	stand := bidStandFor(x.rules)
	if stand.State != leaguerules.IsKnown || stand.Value <= 0 {
		return fmt.Errorf("the bid closing time needs a ruling")
	}
	if !x.now.Before(auction.StandingAt().Add(stand.Value)) {
		return fmt.Errorf("bidding has closed; this auction is waiting for a decision")
	}
	return nil
}
