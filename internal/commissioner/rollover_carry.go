package commissioner

import (
	"fmt"
	"strings"
)

// The rookie draft rounds whose contracts expire as restricted free agents.
const (
	firstRFARound = 2
	lastRFARound  = 5
	deadMoney     = "dead_money"
)

func rolloverPlayerName(player NativePlayer, held map[string]string) string {
	if name := strings.TrimSpace(held[player.ID]); name != "" {
		return name
	}
	if name := strings.TrimSpace(player.Name); name != "" {
		return name
	}
	return "A player with no name recorded (" + player.ID + ")"
}

// expiryRoute says which way an expiring player goes: restricted for a rookie contract from rounds 2 to 5,
// unrestricted otherwise. Without the recorded rookie round it says so and guesses nothing.
func expiryRoute(rookieRound int) string {
	switch {
	case rookieRound == 0:
		return "RFA or UFA cannot be determined: the rookie contract's draft round is not recorded."
	case rookieRound >= firstRFARound && rookieRound <= lastRFARound:
		return fmt.Sprintf("Restricted free agent (RFA): a round %d rookie contract.", rookieRound)
	default:
		return fmt.Sprintf("Unrestricted free agent (UFA): a round %d rookie contract.", rookieRound)
	}
}

func franchiseName(franchises []NativeFranchise, id string) string {
	for _, f := range franchises {
		if f.ID == id {
			return f.Name
		}
	}
	return id
}

// carrySeasonFacts archives the season's champion and settles the recorded charges for the new season.
func (p *RolloverPlan) carrySeasonFacts() {
	p.archiveChampion()
	p.carryCharges()
}

func (p *RolloverPlan) archiveChampion() {
	champion := p.After.Champion
	if champion == "" {
		p.Lines = append(p.Lines, fmt.Sprintf("No champion is decided for season %d, so none is archived.", p.From))
		return
	}
	p.After.Champions = append(p.After.Champions, NativeChampion{Season: p.From, Franchise: champion})
	p.After.Champion = ""
	p.Lines = append(p.Lines, fmt.Sprintf("%s, the season %d champion, is archived.",
		franchiseName(p.After.Franchises, champion), p.From))
}

// carryCharges keeps dead money counting in the new season and ends charges dated to the season just
// played. A charge with no season is standing and stays; one dated to the new season or later is already
// placed. Only the dead-money charges dated to the old season are re-dated, with their amount unchanged.
func (p *RolloverPlan) carryCharges() {
	for i := range p.After.Adjustments {
		charge := &p.After.Adjustments[i]
		who := franchiseName(p.After.Franchises, charge.Franchise)
		switch {
		case charge.Season == 0 || charge.Season >= p.To:
			p.Lines = append(p.Lines, fmt.Sprintf("%s: %s keeps counting in season %d.", who, chargeWords(*charge), p.To))
		case charge.Kind == deadMoney:
			charge.Season = p.To
			p.Lines = append(p.Lines, fmt.Sprintf("%s: %s carries into season %d.", who, chargeWords(*charge), p.To))
		default:
			p.Lines = append(p.Lines, fmt.Sprintf("%s: %s ends with season %d; the cap resets.",
				who, chargeWords(*charge), charge.Season))
		}
	}
	if len(p.After.Adjustments) == 0 {
		p.Lines = append(p.Lines, "No salary adjustments or dead money are recorded, so none carries.")
	}
}

func chargeWords(charge NativeCharge) string {
	kind := "the salary adjustment"
	if charge.Kind == deadMoney {
		kind = "dead money"
	}
	return fmt.Sprintf("%s (%s, %s)", kind, charge.Description, charge.Amount)
}
