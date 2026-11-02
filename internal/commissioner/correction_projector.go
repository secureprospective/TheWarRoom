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
	if r.Command.Command == CorrectTransaction || r.Command.Command == RuleException {
		return nil
	}
	if r.Command.Command == CorrectPick {
		return correctPick(seed, r)
	}
	side, err := seed.TargetSide(c.Franchise)
	if err != nil {
		return err
	}
	index := seed.PlayerOn(side, c.Player)
	if r.Command.Command == CorrectProvenance {
		return correctProvenance(seed, index, r)
	}
	if !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Franchise }) {
		return fmt.Errorf("choose a franchise in this app league")
	}
	if c.Action == "status" {
		return correctRosterStatus(seed, index, r)
	}

	if c.Action == "remove" {
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
	player.Franchise, player.Status = c.Franchise, domain.RosterActive
	player.Correction = correctionWords(r)
	seed.Players = append(seed.Players, player)
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
	for i := range seed.Picks {
		p := &seed.Picks[i]
		if p.Season != c.Pick.Season || p.Round != c.Pick.Round || p.Original != c.Pick.Original || p.Slot != c.Pick.Slot {
			continue
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
	return fmt.Errorf("choose a recorded draft pick")
}

func correctionWords(r Record) string { return "Commissioner correction: " + r.Command.Reason }

func correctionLabel(r Record) string {
	c := r.Command.Args.Correction
	switch r.Command.Command {
	case CorrectRoster:
		return fmt.Sprintf("Corrected player %s on %s: %s", c.Player, c.Franchise, c.Action)
	case CorrectTransaction:
		return "Reversed recorded transaction " + c.Entry
	case CorrectProvenance:
		return fmt.Sprintf("Corrected how player %s was acquired: %s", c.Player, c.Via)
	case CorrectPick:
		if c.Pick.Original == "" {
			return fmt.Sprintf("Corrected season %d round %d slot %d: owned by %s", c.Pick.Season, c.Pick.Round, c.Pick.Slot, c.Owner)
		}
		return fmt.Sprintf("Corrected season %d round %d pick from %s: owned by %s",
			c.Pick.Season, c.Pick.Round, c.Pick.Original, c.Owner)
	case RuleException:
		end := "without an expiry"
		if c.Until != "" {
			end = "until " + c.Until
		}
		return "Allowed an exception to " + c.Rule + " " + end
	}
	return "Corrected a recorded asset"
}

func correctRosterStatus(seed *RawNativeSnapshot, index int, r Record) error {
	c := r.Command.Args.Correction
	if index < 0 || seed.Players[index].Franchise != c.Franchise {
		return fmt.Errorf("choose a player on that franchise's recorded roster")
	}
	seed.Players[index].Status, seed.Players[index].Correction = c.Status, correctionWords(r)
	return nil
}
