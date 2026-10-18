package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

type waiverWindow struct {
	Results    []WaiverClaimResult
	PriorOrder []string
	ID         string
	Player     NativePlayer
	CutAt      time.Time
	Season     int
	Dead       domain.Money
	Claims     []waiverClaim
	Processed  bool
	Winner     string
	ClosedAt   time.Time
}

type waiverClaim struct{ ID, Franchise, Principal string }

type waiverState struct {
	Windows     []waiverWindow
	BaseOrder   []string
	Order       []string
	OrderSeason int
	Source      string
	// Bids rides with the waiver state because both fold the same ledger into the same rosters.
	Bids bidBook
}

func (s *waiverState) window(id string) (*waiverWindow, error) {
	for i := range s.Windows {
		if s.Windows[i].ID == id {
			return &s.Windows[i], nil
		}
	}
	return nil, fmt.Errorf("choose a recorded waiver window")
}

func (s *waiverState) claim(id string) (*waiverWindow, waiverClaim, error) {
	for i := range s.Windows {
		for _, c := range s.Windows[i].Claims {
			if c.ID == id {
				return &s.Windows[i], c, nil
			}
		}
	}
	return nil, waiverClaim{}, fmt.Errorf("choose a recorded waiver claim")
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
			Amount: result.Dead, Season: result.Season, Description: "Dead money for released player " + args.Player})
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
