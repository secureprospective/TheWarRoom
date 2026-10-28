package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/rookiedraft"
)

// applyDraftPick gives a drafted rookie his rookie contract, and marks the pick he was taken with.
func applyDraftPick(seed *RawNativeSnapshot, record Record) {
	d := record.Command.Args.Draft
	p := d.Pick
	if d.Startup {
		signStartup(seed, d.Season, p.Player, p.Franchise, p.Round, p.Slot, p.Terms)
		return
	}
	signRookie(seed, d.Season, p.Player, p.Franchise, p.Round, p.Slot, p.Terms)
	markPick(seed, d.Season, p.Round, p.Original, p.Slot, p.Player)
}

// signStartup puts a player taken in the startup draft on a roster under the startup draft's contract. He is
// no rookie, and his pick was the draft's own, so no recorded pick is marked.
func signStartup(seed *RawNativeSnapshot, season int, player, franchise string, round, slot int, terms rookiedraft.Terms) {
	seed.Players = append(seed.Players, NativePlayer{ID: player, Franchise: franchise, Status: domain.RosterActive,
		Salary: terms.Salary, ExpirationYear: season + terms.Years - 1, ContractStatus: domain.CStatusUFA,
		ContractInfo: fmt.Sprintf("Round %d pick %d of the %d startup draft", round, slot, season)})
}

// signRookie puts a drafted rookie on a roster under the contract his slot fixed.
func signRookie(seed *RawNativeSnapshot, season int, player, franchise string, round, slot int, terms rookiedraft.Terms) {
	status := domain.CStatusUFA
	if terms.RFAAfter {
		status = domain.CStatusRFA
	}
	seed.Players = append(seed.Players, NativePlayer{ID: player, Franchise: franchise, RookieRound: round,
		Status: domain.RosterActive, Salary: terms.Salary, ExpirationYear: season + terms.Years - 1,
		ContractStatus: status, ContractInfo: fmt.Sprintf("Rookie contract, round %d pick %d of the %d draft",
			round, slot, season)})
}

// markPick records which rookie a scheduled pick was used on, and where it fell.
func markPick(seed *RawNativeSnapshot, season, round int, original string, slot int, player string) {
	for i, pick := range seed.Picks {
		if pick.Season == season && pick.Round == round && pick.Original == original {
			seed.Picks[i].Slot, seed.Picks[i].Player = slot, player
		}
	}
}

// applyDraftTray gives the rookie a commissioner chose, or added, his contract; a chosen pick is marked used.
func applyDraftTray(seed *RawNativeSnapshot, record Record) {
	d, o := record.Command.Args.Draft, record.Command.Args.DraftOutcome
	if o == nil || o.Terms == nil {
		return
	}
	if d.Startup {
		signStartup(seed, d.Season, d.Act.Player, o.Franchise, d.Act.Round, o.Slot, *o.Terms)
		return
	}
	signRookie(seed, d.Season, d.Act.Player, o.Franchise, d.Act.Round, o.Slot, *o.Terms)
	if record.Command.Command == DraftAssign && d.Act.Extra == 0 {
		markPick(seed, d.Season, d.Act.Round, d.Act.Original, o.Slot, d.Act.Player)
	}
}

// applyDraftChange swaps the rookie a pick holds: the new one takes the contract, and the old one leaves the
// roster to be undrafted.
func applyDraftChange(seed *RawNativeSnapshot, record Record) {
	d := record.Command.Args.Draft
	for i := range seed.Players {
		if seed.Players[i].ID == d.Act.Replaces {
			seed.Players[i].ID = d.Act.Player
		}
	}
	for i := range seed.Picks {
		if seed.Picks[i].Player == d.Act.Replaces {
			seed.Picks[i].Player = d.Act.Player
		}
	}
}

// applyDraftClose puts a closed draft's outcome into the native assets: the released rookies join the
// free-agent pool and the issued picks join the franchises' picks.
func applyDraftClose(seed *RawNativeSnapshot, record Record) {
	outcome := record.Command.Args.DraftOutcome
	if outcome == nil {
		return
	}
	seed.Pool = append(seed.Pool, outcome.Pool...)
	seed.Picks = append(seed.Picks, outcome.Issued...)
}
