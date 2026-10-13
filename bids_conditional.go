package main

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type BidDropOption struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Label     string       `json:"label"`
	Salary    domain.Money `json:"salary"`
	DeadCap   domain.Money `json:"deadCap"`
	Freed     domain.Money `json:"freed"`
	Covers    bool         `json:"covers"`
	Available bool         `json:"available"`
}

// Held future salaries are flat; MFL does not provide restructure history here.
func heldCutCharge(p state.MirrorPlayer, season int) (domain.Money, bool) {
	if p.ContractYear < season {
		return 0, false
	}
	cells := []state.LedgerCell{}
	for year := season; year <= p.ContractYear; year++ {
		cells = append(cells, state.LedgerCell{Year: year, Salary: p.Salary})
	}
	return transactions.CutCharge(cells, season, false), true
}

func (a *App) bidDropOptions(mine string, salary domain.Money) []BidDropOption {
	a.lookupMu.Lock()
	lk := a.lookup
	a.lookupMu.Unlock()
	m, _, _ := a.league.ContractsView()
	room := a.capRoom(mine)
	out := []BidDropOption{}
	for _, p := range m.Players {
		if p.FranchiseID != mine {
			continue
		}
		name := p.MFLID
		if f, ok := lk.Facts(p.MFLID); ok {
			name = fmt.Sprintf("%s (%s, %s)", f.Name, f.Position, f.Team)
		}
		charge, available := heldCutCharge(p, m.Season)
		freed := p.Salary - charge
		label := fmt.Sprintf("%s · %s salary · %s · %s dead cap if dropped",
			name, bid.MoneyText(p.Salary), bid.YearsText(p.ContractYear-m.Season+1), bid.MoneyText(charge))
		if !available {
			label = name + " · contract length not held; cannot price a drop"
		}
		out = append(out, BidDropOption{ID: p.MFLID, Name: name, Label: label, Salary: p.Salary,
			DeadCap: charge, Freed: freed, Covers: available && room != nil && *room+freed >= salary,
			Available: available})
	}
	slices.SortFunc(out, func(a, b BidDropOption) int {
		if a.Covers != b.Covers {
			if a.Covers {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return out
}

func (a *App) bidConditional(mine string, f envelope.BidFacts) (*bid.ConditionalDrop, error) {
	if f.ConditionalDrop.String() == "" {
		return nil, nil
	}
	for _, option := range a.bidDropOptions(mine, f.Salary) {
		if option.ID != f.ConditionalDrop.String() {
			continue
		}
		if !option.Available {
			return nil, fmt.Errorf("conditional drop: contract length is not held")
		}
		return &bid.ConditionalDrop{Name: option.Name, Salary: option.Salary, DeadCap: option.DeadCap}, nil
	}
	return nil, fmt.Errorf("conditional drop must be on your own roster; choose another player or clear it")
}

func withConditionalName(f envelope.BidFacts, options []BidDropOption) envelope.BidFacts {
	f.ConditionalName = ""
	for _, option := range options {
		if option.ID == f.ConditionalDrop.String() {
			f.ConditionalName = option.Name
		}
	}
	return f
}
