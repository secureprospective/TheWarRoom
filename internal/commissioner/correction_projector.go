package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func applyAssetCorrection(seed *RawNativeSnapshot, contracts map[string]NativePlayer, r Record) error {
	if seed == nil {
		return fmt.Errorf("record the starting assets before correcting them")
	}
	c := r.Command.Args.Correction
	switch r.Command.Command {
	case CorrectTransaction, RuleException:
		return nil
	case CorrectPick:
		return correctPick(seed, r)
	}
	if r.Command.Command == CorrectRoster && (c.Action == "adjustment" || c.Action == "remove_adjustment") {
		return correctRosterAdjustment(seed, r)
	}
	side, err := seed.TargetSide(c.Franchise)
	if err != nil {
		return err
	}
	index := seed.PlayerOn(side, c.Player)
	if r.Command.Command == CorrectProvenance {
		return correctProvenance(seed, index, r)
	}
	return applyRosterCorrection(seed, contracts, side, index, r)
}

func applyRosterCorrection(seed *RawNativeSnapshot, contracts map[string]NativePlayer, side string, index int, r Record) error {
	c := r.Command.Args.Correction
	if !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Franchise }) {
		return fmt.Errorf("choose a franchise in this app league")
	}
	switch c.Action {
	case "status":
		return correctRosterStatus(seed, index, r)
	case "contract":
		return correctRosterContract(seed, index, r)
	case "remove":
		if index < 0 || seed.Players[index].Franchise != c.Franchise {
			return fmt.Errorf("this player is not on that franchise's roster")
		}
		seed.Players = slices.Delete(seed.Players, index, index+1)
		return nil
	}
	if index >= 0 {
		return fmt.Errorf("remove the player from his current roster before adding him elsewhere")
	}
	player, ok := contracts[side+"/"+c.Player]
	if !ok {
		player = NativePlayer{ID: c.Player, ContractMissing: true}
	}
	if c.Contract != nil {
		applyCorrectionContract(&player, *c.Contract)
	}
	player.Franchise, player.Status = c.Franchise, domain.RosterActive
	player.Correction = correctionWords(r)
	seed.Players = append(seed.Players, player)
	return nil
}

func applyCorrectionContract(player *NativePlayer, c CorrectionContract) {
	player.ContractMissing = c.ContractMissing
	player.Salary = c.Salary
	player.ExpirationYear = c.ExpirationYear
	player.ContractStatus = c.ContractStatus
	player.ContractInfo = c.ContractInfo
	player.HeldContractStatus = c.HeldContractStatus
	player.ContractSchedule = slices.Clone(c.ContractSchedule)
	player.Pending = false
	player.Restructured = false
}

func correctRosterContract(seed *RawNativeSnapshot, index int, r Record) error {
	c := r.Command.Args.Correction
	if index < 0 || seed.Players[index].Franchise != c.Franchise {
		return fmt.Errorf("choose a player on that franchise's recorded roster")
	}
	applyCorrectionContract(&seed.Players[index], *c.Contract)
	seed.Players[index].Correction = correctionWords(r)
	return nil
}

func correctRosterAdjustment(seed *RawNativeSnapshot, r Record) error {
	c := r.Command.Args.Correction
	if !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Franchise }) {
		return fmt.Errorf("choose a franchise in this app league")
	}
	idx := slices.IndexFunc(seed.Adjustments, func(a NativeCharge) bool { return a.ID == c.Charge.ID })
	if c.Action == "remove_adjustment" {
		if idx < 0 || seed.Adjustments[idx].Franchise != c.Franchise {
			return fmt.Errorf("choose a recorded salary adjustment")
		}
		seed.Adjustments = slices.Delete(seed.Adjustments, idx, idx+1)
		return nil
	}
	if idx >= 0 {
		seed.Adjustments[idx] = *c.Charge
		return nil
	}
	seed.Adjustments = append(seed.Adjustments, *c.Charge)
	return nil
}

// correctProvenance records how the side's copy of the player was acquired.
func correctProvenance(seed *RawNativeSnapshot, index int, r Record) error {
	c := r.Command.Args.Correction
	if index < 0 || (c.Franchise != "" && seed.Players[index].Franchise != c.Franchise) {
		return fmt.Errorf("choose a recorded roster player")
	}
	seed.Players[index].AcquiredVia = c.Via
	seed.Players[index].Correction = correctionWords(r)
	return nil
}

func correctPick(seed *RawNativeSnapshot, r Record) error {
	c := r.Command.Args.Correction
	if !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Owner }) {
		return fmt.Errorf("choose an owner in this app league")
	}
	if c.History != nil {
		return correctHistoricalPick(seed, c)
	}
	idx := slices.IndexFunc(seed.Picks, func(p NativePick) bool {
		return p.Season == c.Pick.Season && p.Round == c.Pick.Round && p.Original == c.Pick.Original && p.Slot == c.Pick.Slot
	})
	if c.Pick.Add {
		return addCorrectedPick(seed, idx, r)
	}
	if idx < 0 {
		return fmt.Errorf("choose a recorded draft pick")
	}
	if c.Pick.Remove {
		seed.Picks = slices.Delete(seed.Picks, idx, idx+1)
		return nil
	}
	return updateCorrectedPick(&seed.Picks[idx], r)
}

func addCorrectedPick(seed *RawNativeSnapshot, existing int, r Record) error {
	if existing >= 0 {
		return fmt.Errorf("this draft pick is already recorded")
	}
	c := r.Command.Args.Correction
	if c.Pick.Original != "" && !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Pick.Original }) {
		return fmt.Errorf("choose an original franchise in this league")
	}
	seed.Picks = append(seed.Picks, NativePick{
		Season:     c.Pick.Season,
		Round:      c.Pick.Round,
		Slot:       c.Pick.Slot,
		Original:   c.Pick.Original,
		Owner:      c.Owner,
		Player:     c.Pick.Player,
		Skipped:    c.Pick.Skipped,
		Correction: correctionWords(r),
	})
	return nil
}

func updateCorrectedPick(p *NativePick, r Record) error {
	c := r.Command.Args.Correction
	if c.Pick.Clear {
		if p.Player == "" && !p.Skipped && p.Owner == c.Owner {
			return fmt.Errorf("this pick already matches the correction")
		}
		p.Player, p.Skipped, p.Owner, p.Correction = "", false, c.Owner, correctionWords(r)
		return nil
	}
	if c.Pick.Player != "" || c.Pick.Skipped {
		if p.Player == c.Pick.Player && p.Skipped == c.Pick.Skipped && p.Owner == c.Owner {
			return fmt.Errorf("this pick already matches the correction")
		}
		p.Player, p.Skipped, p.Owner, p.Correction = c.Pick.Player, c.Pick.Skipped, c.Owner, correctionWords(r)
		return nil
	}
	if p.Player != "" || p.Skipped {
		return fmt.Errorf("reverse the recorded selection before changing its owner")
	}
	if p.Owner == c.Owner {
		return fmt.Errorf("this franchise already owns the pick")
	}
	p.Owner, p.Correction = c.Owner, correctionWords(r)
	return nil
}

func correctionWords(r Record) string { return "Commissioner correction: " + r.Command.Reason }

func correctionLabel(r Record) string {
	c := r.Command.Args.Correction
	switch r.Command.Command {
	case CorrectRoster:
		return rosterCorrectionLabel(c)
	case CorrectTransaction:
		return "Reversed recorded transaction " + c.Entry
	case CorrectProvenance:
		return fmt.Sprintf("Corrected how player %s was acquired: %s", c.Player, c.Via)
	case CorrectPick:
		return pickCorrectionLabel(c)
	case RuleException:
		end := "without an expiry"
		if c.Until != "" {
			end = "until " + c.Until
		}
		return "Allowed an exception to " + c.Rule + " " + end
	}
	return "Corrected a recorded asset"
}

func rosterCorrectionLabel(c *RawCorrection) string {
	switch c.Action {
	case "adjustment":
		return fmt.Sprintf("Corrected salary adjustment %s on %s", c.Charge.ID, c.Franchise)
	case "remove_adjustment":
		return fmt.Sprintf("Removed salary adjustment %s from %s", c.Charge.ID, c.Franchise)
	default:
		return fmt.Sprintf("Corrected player %s on %s: %s", c.Player, c.Franchise, c.Action)
	}
}

func pickCorrectionLabel(c *RawCorrection) string {
	base := fmt.Sprintf("season %d round %d pick from %s", c.Pick.Season, c.Pick.Round, c.Pick.Original)
	if c.Pick.Original == "" {
		base = fmt.Sprintf("season %d round %d slot %d", c.Pick.Season, c.Pick.Round, c.Pick.Slot)
	}
	switch {
	case c.Pick.Remove:
		return fmt.Sprintf("Removed %s", base)
	case c.Pick.Player != "":
		return fmt.Sprintf("Corrected %s: owned by %s, selected player %s", base, c.Owner, c.Pick.Player)
	case c.Pick.Skipped:
		return fmt.Sprintf("Corrected %s: owned by %s, skipped", base, c.Owner)
	case c.Pick.Clear:
		return fmt.Sprintf("Corrected %s: owned by %s, selection cleared", base, c.Owner)
	default:
		return fmt.Sprintf("Corrected %s: owned by %s", base, c.Owner)
	}
}

func correctRosterStatus(seed *RawNativeSnapshot, index int, r Record) error {
	c := r.Command.Args.Correction
	if index < 0 || seed.Players[index].Franchise != c.Franchise {
		return fmt.Errorf("choose a player on that franchise's recorded roster")
	}
	seed.Players[index].Status, seed.Players[index].Correction = c.Status, correctionWords(r)
	return nil
}
