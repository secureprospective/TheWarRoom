package commissioner

import (
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func projectBid(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	if args == nil || outcome == nil {
		return fmt.Errorf("a bid change has no recorded details")
	}
	book := &state.Bids
	switch r.Command.Command {
	case BidPlace:
		return projectPlacedBid(seed, book, r)
	case BidRaise:
		return projectRaisedBid(book, r)
	case BidWithdraw:
		a, j, err := book.find(args.Bid)
		if err != nil {
			return err
		}
		if a.Decided || !a.Bids[j].live() {
			return fmt.Errorf("reverse the later bid changes before reversing this one")
		}
		a.Bids[j].Withdrawn = true
	case BidsProcess, BidsOverride:
		return projectBidAward(seed, book, r)
	}
	return nil
}

func projectPlacedBid(seed *RawNativeSnapshot, book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	if slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.ID == args.Player }) {
		return fmt.Errorf("this player already belongs to a roster")
	}
	bid := bidRecord{ID: r.Receipt.Entry, Franchise: args.Franchise, Principal: r.Principal, Salary: args.Salary,
		Years: args.Years, Points: outcome.Points, PlacedAt: r.Receipt.RecordedAt}
	if a := book.open(args.Player); a != nil {
		if _, held := a.activeFor(args.Franchise); held {
			return fmt.Errorf("this franchise already has a bid on this player")
		}
		a.Bids = append(a.Bids, bid)
		return nil
	}
	book.Auctions = append(book.Auctions, bidAuction{Player: args.Player, Bids: []bidRecord{bid}})
	return nil
}

func projectRaisedBid(book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	a, j, err := book.find(args.Bid)
	if err != nil {
		return err
	}
	if a.Decided || !a.Bids[j].live() || a.Bids[j].Franchise != args.Franchise {
		return fmt.Errorf("reverse the later bid changes before reversing this one")
	}
	a.Bids[j].Replaced = true
	a.Bids = append(a.Bids, bidRecord{ID: r.Receipt.Entry, Franchise: args.Franchise, Principal: r.Principal,
		Salary: args.Salary, Years: args.Years, Points: outcome.Points,
		PlacedAt: r.Receipt.RecordedAt})
	return nil
}

// projectBidAward decides an auction, or corrects a decision: the previous winner leaves the roster and the
// new one signs the contract his live bid named.
func projectBidAward(seed *RawNativeSnapshot, book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	var a *bidAuction
	if r.Command.Command == BidsProcess {
		if a = book.open(args.Player); a == nil {
			return fmt.Errorf("this player has no bids waiting for a decision")
		}
	} else {
		found, _, err := book.find(args.Bid)
		if err != nil {
			return err
		}
		if a = found; !a.Decided || book.latest(a.Player) != a {
			return fmt.Errorf("correct only the newest decision on a player")
		}
	}
	if err := replaceBidWinner(seed, a, outcome); err != nil {
		return err
	}
	a.Decided, a.Winner, a.WinningBid, a.Tied = true, outcome.Winner, outcome.WinningBid, slices.Clone(outcome.Tied)
	a.Results, a.Season, a.DecidedAt = slices.Clone(outcome.Results), outcome.Season, r.Receipt.RecordedAt
	return nil
}

func replaceBidWinner(seed *RawNativeSnapshot, a *bidAuction, outcome *BidOutcome) error {
	if a.Winner != "" {
		index := slices.IndexFunc(seed.Players, func(p NativePlayer) bool {
			return p.ID == a.Player && p.Franchise == a.Winner
		})
		if index < 0 {
			return fmt.Errorf("a later roster change depends on this award; reverse it first")
		}
		seed.Players = slices.Delete(seed.Players, index, index+1)
	}
	if outcome.Winner == "" {
		return nil
	}
	if slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.ID == a.Player }) {
		return fmt.Errorf("this player already belongs to a roster")
	}
	won := slices.IndexFunc(a.Bids, func(b bidRecord) bool { return b.ID == outcome.WinningBid })
	if won < 0 || a.Bids[won].Franchise != outcome.Winner {
		return fmt.Errorf("the winning bid is not a recorded bid of the winner")
	}
	bid := a.Bids[won]
	name := ""
	if pool := slices.IndexFunc(seed.Pool, func(p NativePoolEntry) bool { return p.Player == a.Player }); pool >= 0 {
		name = seed.Pool[pool].Name
	}
	seed.Players = append(seed.Players, NativePlayer{ID: a.Player, Name: name, Franchise: bid.Franchise, Status: domain.RosterActive,
		Salary: bid.Salary, ExpirationYear: outcome.Season + bid.Years - 1, ContractStatus: domain.CStatusUFA,
		ContractInfo: "Won in free-agent bidding: " + nativecheck.ContractWords(bid.Salary, bid.Years)})
	return nil
}

// bidPlayerOf follows a bid change back to the player it concerns.
func bidPlayerOf(r Record, records []Record) string {
	for range len(records) {
		args := r.Command.Args.Bid
		if args == nil {
			return ""
		}
		if args.Player != "" {
			return args.Player
		}
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == args.Bid })
		if index < 0 {
			return ""
		}
		r = records[index]
	}
	return ""
}

// privateBidRecord hides an owner's bid, and the reversal of one, from everyone else.
func privateBidRecord(r Record, records []Record) bool {
	for range len(records) {
		if r.Command.Command != Revert {
			break
		}
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == r.Command.Args.Entry })
		if index < 0 {
			return false
		}
		r = records[index]
	}
	return OwnerBidAction(r.Command.Command) != ""
}

// decidedBidRecord says whether the player's bids have been decided, which is when administrators may see
// them. A decision that is itself reversed does not count.
func decidedBidRecord(r Record, records []Record) bool {
	player := bidPlayerOf(r, records)
	if player == "" {
		return false
	}
	active := Replay(records).Active
	return slices.ContainsFunc(records, func(v Record) bool {
		return active[v.Receipt.Entry] && v.Command.Command == BidsProcess && v.Command.Args.Bid.Player == player
	})
}

func projectTrade(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	if r.Command.Args.TradeOutcome == nil {
		return fmt.Errorf("a trade action has no recorded outcome")
	}
	t := r.Command.Args.TradeOutcome.clone()
	index := slices.IndexFunc(state.Trades, func(v TradeState) bool { return v.ID == t.ID })
	before := false
	if r.Command.Command == TradeOffer {
		if index >= 0 {
			return fmt.Errorf("this trade was already offered")
		}
		t.ID = r.Receipt.Entry
	} else {
		if index < 0 {
			return fmt.Errorf("reverse the later trade actions first")
		}
		if t.Previous != tradeDigest(state.Trades[index]) {
			return fmt.Errorf("undo the later trade actions first")
		}
		before = state.Trades[index].Status == "approved"
	}
	after := t.Status == "approved"
	if before != after {
		if err := moveTradeAssets(seed, t.Assets, before); err != nil {
			return err
		}
	}
	if index < 0 {
		state.Trades = append(state.Trades, t)
	} else {
		state.Trades[index] = t
	}
	return nil
}

// Ownership checks make compensation refuse to strand a subsequent cut, award or pick transfer.
func moveTradeAssets(seed *RawNativeSnapshot, assets []TradeAsset, reverse bool) error {
	for _, a := range assets {
		from, to := a.From, a.To
		if reverse {
			from, to = to, from
		}
		if a.Player != "" {
			i := slices.IndexFunc(seed.Players, func(p NativePlayer) bool { return p.ID == a.Player && p.Franchise == from })
			if i < 0 {
				return fmt.Errorf("a traded player no longer belongs to the sending team; undo the later asset change first")
			}
			seed.Players[i].Franchise = to
		} else {
			i := slices.IndexFunc(seed.Picks, func(p NativePick) bool {
				return p.Season == a.Season && p.Round == a.Round && p.Original == a.Original && p.Owner == from
			})
			if i < 0 || seed.Picks[i].Player != "" || seed.Picks[i].Skipped || seed.Picks[i].Slot != 0 || seed.Picks[i].Compensatory {
				return fmt.Errorf("a traded pick is no longer available; undo the later pick change first")
			}
			seed.Picks[i].Owner = to
		}
	}
	return nil
}

func tradeDigest(t TradeState) string {
	body, err := json.Marshal(t)
	if err != nil {
		return "invalid"
	}
	return fmt.Sprintf("%x", sha256.Sum256(body))
}

func projectWaiver(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	args, result := r.Command.Args.Waiver, r.Command.Args.WaiverOutcome
	if args == nil {
		return fmt.Errorf("a waiver change has no details")
	}
	if r.Command.Command != WaiverClaim && result == nil {
		return fmt.Errorf("a waiver change has no recorded decision")
	}
	switch r.Command.Command {
	case RosterReserve:
		return projectReserve(seed, args, result)
	case RosterCut:
		if result.Player == nil {
			return fmt.Errorf("a cut has no recorded contract")
		}
		index := slices.IndexFunc(seed.Players, func(p NativePlayer) bool { return p.ID == args.Player })
		if index < 0 || seed.Players[index] != *result.Player {
			return fmt.Errorf("a later cut depends on the earlier roster change; reverse it first")
		}
		seed.Players = slices.Delete(seed.Players, index, index+1)
		seed.Adjustments = append(seed.Adjustments, NativeCharge{ID: r.Receipt.Entry, Franchise: args.Franchise,
			Amount: result.Dead, Season: result.Season, Description: "Dead money for released player " + cmp.Or(result.Player.Name, args.Player)})
		state.Windows = append(state.Windows, waiverWindow{ID: r.Receipt.Entry, Player: *result.Player,
			CutAt: r.Receipt.RecordedAt, Season: result.Season, Dead: result.Dead, Claims: []waiverClaim{}})
	case WaiverClaim:
		w, err := state.window(args.Window)
		if err != nil {
			return err
		}
		if w.Processed {
			return fmt.Errorf("reverse the waiver award before reversing its claims")
		}
		w.Claims = append(w.Claims, waiverClaim{ID: r.Receipt.Entry, Franchise: args.Franchise, Principal: r.Principal})
	case WaiverOrderSet, WaiverOrderAuto:
		state.Order, state.OrderSeason, state.Source = slices.Clone(result.Order), result.Season, result.Source
		state.BaseOrder = slices.Clone(result.Order)
	case WaiverProcess, WaiverOverride:
		return projectWaiverAward(seed, state, r)
	}
	return nil
}

func projectWaiverAward(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	args, result := r.Command.Args.Waiver, r.Command.Args.WaiverOutcome
	w, err := state.window(args.Window)
	if r.Command.Command == WaiverOverride {
		w, _, err = state.claim(args.Claim)
	}
	if err != nil {
		return err
	}
	if r.Command.Command == WaiverProcess && w.Processed {
		return fmt.Errorf("this waiver window was already processed")
	}
	if r.Command.Command == WaiverOverride && !w.Processed {
		return fmt.Errorf("process the waiver window before correcting its award")
	}
	if err := replaceWaiverWinner(seed, w, result.Winner); err != nil {
		return err
	}
	for i := range seed.Adjustments {
		if seed.Adjustments[i].ID == w.ID {
			seed.Adjustments[i].Amount = w.Dead
			if result.Winner != "" && result.ClearsDead {
				seed.Adjustments[i].Amount = 0
			}
		}
	}
	w.Processed, w.Winner, w.ClosedAt = true, result.Winner, result.CloseAt
	w.Results = slices.Clone(result.Claims)
	w.PriorOrder = slices.Clone(result.PriorOrder)
	if len(state.BaseOrder) == 0 {
		state.BaseOrder = slices.Clone(result.PriorOrder)
	}
	state.Order, state.OrderSeason, state.Source = slices.Clone(result.Order), result.Season, result.Source
	return nil
}

// privateWaiverRecord also hides a compensation of a claim, even from league administrators.
func privateWaiverRecord(r Record, records []Record) bool {
	if r.Command.Command == WaiverClaim {
		return true
	}
	for r.Command.Command == Revert {
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == r.Command.Args.Entry })
		if index < 0 {
			return false
		}
		r = records[index]
	}
	return r.Command.Command == WaiverClaim
}

func replaceWaiverWinner(seed *RawNativeSnapshot, w *waiverWindow, winner string) error {
	if w.Winner != "" {
		index := slices.IndexFunc(seed.Players, func(p NativePlayer) bool { return p.ID == w.Player.ID && p.Franchise == w.Winner })
		if index < 0 {
			return fmt.Errorf("a later roster change depends on this award; reverse it first")
		}
		seed.Players = slices.Delete(seed.Players, index, index+1)
	}
	if slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.ID == w.Player.ID }) {
		return fmt.Errorf("this player already belongs to a roster")
	}
	if winner != "" {
		if !slices.ContainsFunc(w.Claims, func(c waiverClaim) bool { return c.Franchise == winner }) {
			return fmt.Errorf("reverse the award before reversing its winning claim")
		}
		player := w.Player
		player.Franchise, player.Status = winner, domain.RosterActive
		seed.Players = append(seed.Players, player)
	}
	return nil
}

func processedWaiverClaim(r Record, records []Record) bool {
	if r.Command.Command != WaiverClaim {
		return false
	}
	active := Replay(records).Active
	for _, award := range records {
		if award.Command.Command == WaiverProcess && active[award.Receipt.Entry] &&
			award.Command.Args.Waiver.Window == r.Command.Args.Waiver.Window {
			return true
		}
	}
	return false
}

func (t TradeState) clone() TradeState {
	t.Details = slices.Clone(t.Details)
	for i, d := range t.Details {
		if d.Salary != nil {
			salary := *d.Salary
			t.Details[i].Salary = &salary
		}
	}
	t.Parties, t.Assets, t.Accepted = slices.Clone(t.Parties), slices.Clone(t.Assets), slices.Clone(t.Accepted)
	t.Votes, t.Recusals, t.Appeals, t.Overrides = slices.Clone(t.Votes), slices.Clone(t.Recusals), slices.Clone(t.Appeals), slices.Clone(t.Overrides)
	return t
}

func tradeByID(trades []TradeState, id string) (TradeState, error) {
	for _, t := range trades {
		if t.ID == id {
			return t.clone(), nil
		}
	}
	return TradeState{}, fmt.Errorf("choose a recorded trade")
}

func checkTradeReplay(raw RawCommand, records []Record, now time.Time) error {
	if raw.Command != Revert {
		return nil
	}
	trial := append(slices.Clone(records), Record{Command: raw, Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: now}})
	_, _, err := nativeAssetsState(raw.League, trial)
	return err
}
