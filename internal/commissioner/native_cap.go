package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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

type RawNativeCapScope struct {
	League     string `json:"league"`
	ViewerZone string `json:"viewerZone"`
}

func (r RawNativeCapScope) Validate() error {
	if strings.TrimSpace(r.League) != r.League || !strings.HasPrefix(r.League, "native.") ||
		r.League == "native." {
		return fmt.Errorf("choose an app league")
	}
	if r.ViewerZone == "" || strings.TrimSpace(r.ViewerZone) != r.ViewerZone || r.ViewerZone == "Local" {
		return fmt.Errorf("choose your timezone")
	}
	if _, err := time.LoadLocation(r.ViewerZone); err != nil {
		return fmt.Errorf("read viewer timezone: %w", err)
	}
	return nil
}

func NativeCapCheckReading(scope RawNativeCapScope, records []Record, now time.Time) (NativeCapCheck, error) {
	if err := scope.Validate(); err != nil {
		return NativeCapCheck{}, err
	}
	zone, err := time.LoadLocation(scope.ViewerZone)
	if err != nil {
		return NativeCapCheck{}, fmt.Errorf("read viewer timezone: %w", err)
	}
	league := scope.League
	base, err := readNativeBase(league, records, now)
	if err != nil {
		return NativeCapCheck{}, err
	}
	out := NativeCapCheck{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		RulesSeason: base.rules.Season, Seeded: base.assets.Seeded, Source: base.source(),
		Counting: "Committed salary counts active salaries, IR and taxi at the league's shares, " +
			"plus signed salary adjustments and dead money. Each counted salary and charge is rounded " +
			"to the league's money increment. Negative cap room is the amount over the cap.",
		Rules: capRuleLines(base.rules, zone), Franchises: []NativeCapFranchise{}}
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
		if c.Season != 0 && c.Season != out.RulesSeason {
			continue
		}
		charges[c.Franchise] = append(charges[c.Franchise], c.Amount)
		details[c.Franchise] = append(details[c.Franchise], c)
	}
	for _, seat := range base.directory.Franchises {
		state, err := nativecheck.CheckCap(base.rules.Cap, players[seat.Franchise], charges[seat.Franchise], zone)
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

func capRuleLines(rules leaguerules.RuleSet, zone *time.Location) []RuleLine {
	lines := ruleLines(rules, "cap.amount", "cap.enforcement", "cap.irShare", "cap.taxiShare", "cap.quantum")
	deadline := ruleLines(rules, "cap.complianceDeadline")
	if rules.Cap.ComplianceDeadline.State == leaguerules.IsKnown {
		deadline[0].Words = rules.Cap.ComplianceDeadline.Value.In(zone).Format("2 Jan 2006 at 3:04 PM MST")
	}
	return append(lines, deadline...)
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
