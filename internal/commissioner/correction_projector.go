package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func applyAssetCorrection(seed *RawNativeSnapshot, records []Record, r Record) error {
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
	player, ok, err := recordedCorrectionPlayer(records, side, c.Player)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("record this player's contract before adding him; a correction cannot invent one")
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

// recordedCorrectionPlayer is the side's copy of the player as the ledger last held it, read in one pass over
// the earlier entries.
func recordedCorrectionPlayer(records []Record, side, id string) (NativePlayer, bool, error) {
	assets, err := seedNativeAssets(records[0].Command.League, records)
	if err != nil {
		return NativePlayer{}, false, fmt.Errorf("read the last recorded contract: %w", err)
	}
	var last NativePlayer
	found := false
	keep := func(a *NativeAssets) {
		if !a.Seeded {
			return
		}
		if i := a.Snapshot.PlayerOn(side, id); i >= 0 {
			last, found = a.Snapshot.Players[i], true
		}
	}
	keep(&assets)
	waivers := MarketBook{Windows: []WaiverWindow{}, Order: []string{}}
	if err := projectNativeOverrides(&assets, records, &waivers, keep); err != nil {
		return NativePlayer{}, false, fmt.Errorf("read the last recorded contract: %w", err)
	}
	return last, found, nil
}

func correctPick(seed *RawNativeSnapshot, r Record) error {
	c := r.Command.Args.Correction
	if !slices.ContainsFunc(seed.Franchises, func(f NativeFranchise) bool { return f.ID == c.Owner }) {
		return fmt.Errorf("choose an owner in this app league")
	}
	for i := range seed.Picks {
		p := &seed.Picks[i]
		if p.Season != c.Pick.Season || p.Round != c.Pick.Round || p.Original != c.Pick.Original {
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
