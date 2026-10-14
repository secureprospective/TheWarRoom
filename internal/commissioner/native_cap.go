package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type NativeCapCharge struct {
	ID          string        `json:"id"`
	Description string        `json:"description"`
	AmountCents *domain.Money `json:"amountCents"`
}

type NativeCapFranchise struct {
	nativecheck.CapState
	Franchise    string            `json:"franchise"`
	Name         string            `json:"name"`
	Abbreviation string            `json:"abbreviation"`
	Charges      []NativeCapCharge `json:"charges"`
}

type NativeCapCheck struct {
	League       string               `json:"league"`
	Name         string               `json:"name"`
	Revision     int64                `json:"revision"`
	RosterSeason int                  `json:"rosterSeason"`
	RulesSeason  int                  `json:"rulesSeason"`
	Seeded       bool                 `json:"seeded"`
	Source       string               `json:"source"`
	Counting     string               `json:"counting"`
	Rules        []RuleLine           `json:"rules"`
	Franchises   []NativeCapFranchise `json:"franchises"`
}

func NativeCapCheckReading(league string, records []Record, now time.Time) (NativeCapCheck, error) {
	base, err := readNativeBase(league, records, now)
	if err != nil {
		return NativeCapCheck{}, err
	}
	out := NativeCapCheck{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		RulesSeason: base.rules.Season, Seeded: base.assets.Seeded, Source: base.source(),
		Counting: "Committed salary counts active salaries, IR and taxi at the league's shares, " +
			"plus signed salary adjustments and dead money. Each counted salary and charge is rounded " +
			"to the league's money increment. Negative cap room is the amount over the cap.",
		Rules: ruleLines(base.rules, "cap.amount", "cap.enforcement", "cap.irShare", "cap.taxiShare",
			"cap.quantum", "cap.complianceDeadline"), Franchises: []NativeCapFranchise{}}
	if !base.assets.Seeded {
		return out, nil
	}
	out.RosterSeason = base.assets.Snapshot.Season
	players := map[string][]nativecheck.CapPlayer{}
	charges := map[string][]domain.Money{}
	details := map[string][]NativeCharge{}
	for _, p := range base.assets.Snapshot.Players {
		players[p.Franchise] = append(players[p.Franchise], nativecheck.CapPlayer{Salary: p.Salary, Status: p.Status})
	}
	for _, c := range base.assets.Snapshot.Adjustments {
		charges[c.Franchise] = append(charges[c.Franchise], c.Amount)
		details[c.Franchise] = append(details[c.Franchise], c)
	}
	for _, seat := range base.directory.Franchises {
		state, err := nativecheck.CheckCap(base.rules.Cap, players[seat.Franchise], charges[seat.Franchise])
		if err != nil {
			return NativeCapCheck{}, fmt.Errorf("judge salary cap for %s: %w", seat.Name, err)
		}
		list, err := nativeCapCharges(details[seat.Franchise], state.QuantumCents)
		if err != nil {
			return NativeCapCheck{}, err
		}
		out.Franchises = append(out.Franchises, NativeCapFranchise{CapState: state, Franchise: seat.Franchise,
			Name: seat.Name, Abbreviation: seat.Abbreviation, Charges: list})
	}
	return out, nil
}

func nativeCapCharges(charges []NativeCharge, quantum *domain.Money) ([]NativeCapCharge, error) {
	out := make([]NativeCapCharge, 0, len(charges))
	for _, charge := range charges {
		row := NativeCapCharge{ID: charge.ID, Description: charge.Description}
		if quantum != nil {
			amount, err := nativecheck.SnapCapMoney(charge.Amount, *quantum)
			if err != nil {
				return nil, fmt.Errorf("round salary adjustment %s: %w", charge.ID, err)
			}
			row.AmountCents = &amount
		}
		out = append(out, row)
	}
	return out, nil
}
