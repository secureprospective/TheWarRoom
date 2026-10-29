// Package contracts is the contracts and roster-action logic moved out of package main: the request
// twins' checks, a contract Act's pricing, evidence and words, a release plan's view and facts, the
// what-if's figures, rules and choices, the held lineup's layout and the roster reports. Which
// franchise a plan, player or what-if move belongs to, the window and lock checks, and every plan-ledger
// write stay in the root adapters; nothing here takes a lock, reads a global or opens a database.
package contracts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/contractacts"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// Ineligible is the reason a contract that cannot take an Act carries; the options leave such an Act out.
const Ineligible = "This contract cannot take this Act."

// Act is the twin of the root RawContractAct: the same fields in the same order.
type Act struct {
	Mine        string `json:"mine"`
	Player      string `json:"player"`
	Action      string `json:"action"`
	Intent      string `json:"intent"`
	ID          string `json:"id"`
	Level       int    `json:"level"`
	Years       int    `json:"years"`
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Move        string `json:"move"`
}

func (r Act) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("contract Act scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "options", "preview", "draft", "handoff", "posted", "withdraw", "rebase"}, r.Action) {
		return fmt.Errorf("contract Act: franchise and known action required")
	}
	if r.Action == "options" {
		if _, err := playerid.New(r.Player); err != nil {
			return fmt.Errorf("contract Act player: %w", err)
		}
	}
	if r.Action == "preview" || r.Action == "draft" {
		return r.validateTerms()
	}
	if !slices.Contains([]string{"read", "options", "preview", "draft"}, r.Action) && r.ID == "" {
		return fmt.Errorf("contract Act: plan id required")
	}
	return nil
}

func (r Act) validateTerms() error {
	if _, err := playerid.New(r.Player); err != nil {
		return fmt.Errorf("contract Act player: %w", err)
	}
	if !envelope.IsContract(r.Intent) || r.Level < 1 || r.Level > len(contractacts.TenderLevels()) || r.Years < 1 || r.Years > contractacts.MaxResignYears {
		return fmt.Errorf("contract Act: known intent, tender level 1–6 and term 1–4 required")
	}
	if r.Move != "" {
		amount, err := domain.ParseMoneyMillions(r.Move)
		if err != nil || amount < 0 || amount > 3*contractacts.Million {
			return fmt.Errorf("contract Act: move must be $0–$3M in millions")
		}
	}
	return nil
}

// Evidence is a held contract as an Act sees it: the player's cells from this season through his
// expiration at today's salary, observed only in its final season.
func Evidence(m state.MirrorSnapshot, mine, player string) envelope.ContractEvidence {
	p, found := Player(m, mine, player)
	if !found {
		return envelope.ContractEvidence{}
	}
	id, err := playerid.New(player)
	if err != nil {
		return envelope.ContractEvidence{}
	}
	cells := []contractacts.Cell{}
	for year := m.Season; year <= max(m.Season, p.ContractYear); year++ {
		cells = append(cells, contractacts.Cell{Season: year, Salary: p.Salary})
	}
	return envelope.ContractEvidence{Player: id, FranchiseID: mine, Cells: cells, Observed: p.ContractYear == m.Season, Status: p.ContractStatus, Info: p.ContractInfo}
}

// ActFacts prices an Act on a player the caller has found on the franchise's roster. window is the
// coordinator phase's refusal of the Act, if any, which the caller reads; rows are every held
// contract. priced is false when the terms could not be priced, and the reasons say why.
func ActFacts(m state.MirrorSnapshot, p state.MirrorPlayer, r Act, window string, rows []snapshot.ContractRow) (f envelope.ContractFacts, priced bool, err error) {
	evidence := Evidence(m, r.Mine, r.Player)
	row := snapshot.ContractRow{Name: r.Player}
	for _, candidate := range rows {
		if candidate.ID == r.Player {
			row = candidate
			break
		}
	}
	f = envelope.ContractFacts{Player: evidence.Player, Name: row.Name, Position: row.Position, Season: m.Season, Intent: r.Intent, Level: r.Level, Years: r.Years, Source: r.Source, Destination: r.Destination, Baseline: evidence.Cells, Status: p.ContractStatus, Info: p.ContractInfo, Reasons: []string{}}
	if window != "" {
		f.Reasons = append(f.Reasons, window)
	}
	if !contractacts.Eligible(r.Intent, f.Status, f.Baseline, f.Season, row.Position) {
		f.Reasons = append(f.Reasons, Ineligible)
	}
	if r.Move != "" {
		f.Move, err = domain.ParseMoneyMillions(r.Move)
		if err != nil {
			return f, false, fmt.Errorf("contract Act move: %w", err)
		}
	}
	terms, priceErr := Terms(f, row.Position, rows, p.Salary)
	f.Terms = terms
	if priceErr != nil {
		f.Reasons = append(f.Reasons, "Terms: "+priceErr.Error())
	}
	f.Reasons = append(f.Reasons, EvidenceReasons(f)...)
	return f, priceErr == nil, nil
}

// EvidenceReasons are what an Act's price rests on and what it could not check.
func EvidenceReasons(f envelope.ContractFacts) []string {
	reasons := []string{}
	switch f.Intent {
	case "rfa.tender", "fa.resign", "contract.tag":
		reasons = append(reasons, fmt.Sprintf("Priced from his %d salary, %s.", f.Season, snapshot.ContractAmountOf(f.Baseline[0].Salary).Text))
	}
	switch f.Intent {
	case "fa.resign":
		reasons = append(reasons, "Not checked: whether another owner has made a competing offer")
	case "contract.tag":
		reasons = append(reasons, "Not checked: earlier franchise tags and this season's team tag allowance", "Not checked: exclusions for playoff signings from the position salary pool")
		if f.Status == domain.CStatusFT2 {
			reasons = append(reasons, "He has already had two consecutive franchise tags.")
		}
		if f.Status == domain.CStatusFT1 {
			reasons = append(reasons, "Not checked: first tag price; using held salary for his second tag")
		}
	case "contract.extend":
		reasons = append(reasons, "Not checked: prior extension and this season's team extension allowance")
	case "contract.restructure":
		reasons = append(reasons, "Not checked: restructure allowance already used on this contract or team, and any extension unlock")
	}
	if len(f.Baseline) > 1 {
		reasons = append(reasons, "Not checked: future salary amounts; later years assume today's salary")
	}
	return reasons
}

// Terms prices the Act under the rulebook: position is the player's, rows every held contract (the
// tag's position pool) and salary his held salary.
func Terms(f envelope.ContractFacts, position string, rows []snapshot.ContractRow, salary domain.Money) (contractacts.Terms, error) {
	switch f.Intent {
	case "rfa.tender", "fa.resign":
		tender, err := contractacts.Tender(salary, f.Level)
		if err != nil {
			return contractacts.Terms{}, fmt.Errorf("RFA price: %w", err)
		}
		price, years, how := tender, 1, "§7: tender = greater of level price and 110% of prior BASE salary; exact cents then $10,000 snap. Prior-season BASE uses the expiring contract’s held salary."
		if f.Intent == "fa.resign" {
			price, err = contractacts.Resign(tender, f.Years)
			years = f.Years
			how = "§7 baseline re-sign with no competing offer: tender × (100% + 15% for each year after the first), 1–4 flat years; exact cents then $10,000 snap. Prior-season BASE uses the expiring contract’s held salary."
		}
		return contractacts.Flat(contractacts.NextSeason(f.Baseline, f.Season), years, price, how), err
	case "contract.tag":
		pool := []domain.Money{}
		for _, row := range rows {
			if row.Position == position {
				pool = append(pool, row.Salary.Cents)
			}
		}
		consecutive, first := 0, domain.Money(0)
		if f.Status == domain.CStatusFT1 {
			consecutive, first = 1, salary
		}
		if f.Status == domain.CStatusFT2 {
			consecutive = 2
		}
		price, err := contractacts.Tag(pool, salary, first, consecutive)
		return contractacts.Flat(contractacts.NextSeason(f.Baseline, f.Season), 1, price, "§9: average of the top five eligible BASE salaries at the position; if below prior salary use 120% of prior BASE, not max(average,120%). Second consecutive tag = 120% of first. Playoff signings excluded. Preview lacks history/exclusions and does not assert eligibility."), err
	case "contract.extend":
		terms, err := contractacts.Extend(f.Baseline, f.Season, f.Years, position, tagPrice(f, salary), false)
		if err != nil {
			return terms, fmt.Errorf("extension terms: %w", err)
		}
		return terms, nil
	case "contract.restructure":
		terms, err := contractacts.Restructure(f.Baseline, f.Source, f.Destination, f.Move, false)
		if err != nil {
			return terms, fmt.Errorf("restructure terms: %w", err)
		}
		return terms, nil
	default:
		return contractacts.Terms{}, fmt.Errorf("unsupported contract Act")
	}
}

func tagPrice(f envelope.ContractFacts, salary domain.Money) domain.Money {
	if f.Status == domain.CStatusFT1 || f.Status == domain.CStatusFT2 {
		return salary
	}
	return 0
}

// NeedsInvalidation says whether an open plan's invalidation reason moves it to Stale: a changed
// contract always does; a window that shut ("Contract: …") only for a Ready or Blocked plan.
func NeedsInvalidation(s envelope.State, reason string) bool {
	if reason == "" {
		return false
	}
	return !strings.HasPrefix(reason, "Contract:") || s == envelope.Ready || s == envelope.Blocked
}

// CapReason judges each season the Act adds money to by that season's commitments after the Act.
func CapReason(f envelope.ContractFacts, capAmount domain.Money, after map[int]domain.Money) string {
	held := map[int]domain.Money{}
	for _, c := range f.Baseline {
		held[c.Season] = c.Salary
	}
	for _, c := range f.Terms.Cells {
		if c.Salary > held[c.Season] && after[c.Season] > capAmount {
			return fmt.Sprintf("This move exceeds the team's %d salary cap.", c.Season)
		}
	}
	return ""
}
