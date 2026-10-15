package nativecheck

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ContractAction is one thing a contract can go through. The names are the ones the league's rules and
// windows use, so a rule, a window and a price all speak of the same action.
type ContractAction string

const (
	ActionTag         ContractAction = "tag"
	ActionExtension   ContractAction = "extension"
	ActionRestructure ContractAction = "restructure"
	ActionBuyout      ContractAction = "buyout"
	ActionTender      ContractAction = "rfa_tender"
	ActionResign      ContractAction = "resign"
	ActionRetire      ContractAction = "retire"
	ActionDeath       ContractAction = "death"
	ActionCapRelief   ContractAction = "cap_relief"
	ActionCut         ContractAction = "cut"
)

// ContractActions lists every priced action in the order a GM meets them.
func ContractActions() []ContractAction {
	return []ContractAction{ActionTag, ActionExtension, ActionRestructure, ActionBuyout, ActionTender,
		ActionResign, ActionCut, ActionRetire, ActionDeath, ActionCapRelief}
}

// Known reports whether the action is one the league prices.
func (a ContractAction) Known() bool {
	for _, known := range ContractActions() {
		if a == known {
			return true
		}
	}
	return false
}

// Name is the action in a GM's words.
func (a ContractAction) Name() string {
	switch a {
	case ActionTag:
		return "franchise tag"
	case ActionExtension:
		return "contract extension"
	case ActionRestructure:
		return "contract restructure"
	case ActionBuyout:
		return "contract buyout"
	case ActionTender:
		return "restricted free agent tender"
	case ActionResign:
		return "restricted free agent re-sign"
	case ActionCut:
		return "release"
	case ActionRetire:
		return "retirement"
	case ActionDeath:
		return "player death"
	case ActionCapRelief:
		return "cap relief"
	}
	return string(a)
}

// Switch is whether the league has an action on. Unknown is its own answer, never "off".
type Switch struct {
	On    bool
	Known bool
}

// ActionSwitch reads the one on/off switch for an action from the rules in force. A release has no
// switch: a team can always release a player, and only the price is the league's to set.
func ActionSwitch(rules leaguerules.ContractsRules, action ContractAction) Switch {
	switch action {
	case ActionTag:
		return pick(rules.Tag.State, rules.Tag.Value.Enabled)
	case ActionExtension:
		return pick(rules.Extension.State, rules.Extension.Value.Enabled)
	case ActionRestructure:
		return pick(rules.Restructure.State, rules.Restructure.Value.Enabled)
	case ActionBuyout:
		return pick(rules.Buyout.State, rules.Buyout.Value.Enabled)
	case ActionTender:
		return pick(rules.Tender.State, rules.Tender.Value.Enabled)
	case ActionResign:
		return pick(rules.ReSign.State, rules.ReSign.Value.Enabled)
	case ActionRetire:
		return pick(rules.Retirement.State, rules.Retirement.Value.Enabled)
	case ActionDeath:
		return pick(rules.Death.State, rules.Death.Value.Enabled)
	case ActionCapRelief:
		return pick(rules.CapRelief.State, rules.CapRelief.Value.Enabled)
	case ActionCut:
		return Switch{On: true, Known: true}
	}
	return Switch{}
}

// Availability says why an action cannot start in this league at all: the league has it turned off
// (refusal), or has not yet ruled on its switch (needs). Both are empty when the action is on. Pricing and
// the owner's start of an action both ask here, so neither can disagree about what is on.
func Availability(rules leaguerules.ContractsRules, action ContractAction) (refusal, needs string) {
	switch sw := ActionSwitch(rules, action); {
	case !sw.Known:
		return "", fmt.Sprintf("Needs a ruling: whether the %s is on or off", action.Name())
	case !sw.On:
		return fmt.Sprintf("This league has the %s turned off.", action.Name()), ""
	}
	return "", ""
}

func pick(outer leaguerules.State, enabled leaguerules.Value[bool]) Switch {
	if outer != leaguerules.IsKnown || enabled.State != leaguerules.IsKnown {
		return Switch{}
	}
	return Switch{On: enabled.Value, Known: true}
}

// Contract is one native contract as the league records it: a flat salary through its last season.
type Contract struct {
	Player       string
	Name         string
	Position     leaguerules.Position
	PositionHeld bool
	Salary       domain.Money
	Expiration   int
	Status       domain.ContractStatus
}

// Trail is what this franchise and this contract have already been through. It counts only what the app
// league has recorded; anything done before the league began is not in it.
type Trail struct {
	// Used counts this franchise's uses of each action in the season being priced.
	Used map[ContractAction]int
	// Restructures and Extensions count what this contract has had so far.
	Restructures int
	Extensions   int
	// ConsecutiveTags is how many years in a row the player has been tagged; FirstTag is the first price.
	ConsecutiveTags int
	FirstTag        domain.Money
}

// Ask is what the GM asked for. Zero means "not said": the action decides whether it needs it.
type Ask struct {
	Years  int
	Level  int
	Move   domain.Money
	Amount domain.Money
	Reason string
}

// SeasonAmount is one season's money.
type SeasonAmount struct {
	Season int          `json:"season"`
	Cents  domain.Money `json:"cents"`
}

// Pricing is the numbers for one action on one contract, or the plain reason it cannot be priced.
type Pricing struct {
	// Refusal is why the action cannot go ahead; Needs names each ruling the league has yet to make.
	Refusal string
	Needs   []string
	// Years and Salary are the new contract's length and yearly price, where one price applies.
	Years  int
	Salary *domain.Money
	// Schedule is the contract's salary by season after the action; Dead is the money that stays on the
	// cap after the player is gone, by the season it lands in; Removed says the player leaves the roster.
	Schedule []SeasonAmount
	Dead     []SeasonAmount
	Removed  bool
	// Relief is cap room the commissioner gives back this season.
	Relief domain.Money
	// Basis says how each number was reached; Notes carry anything else a GM should know.
	Basis []string
	Notes []string
}

func (p *Pricing) refuse(format string, args ...any) { p.Refusal = fmt.Sprintf(format, args...) }

func (p *Pricing) need(what string) { p.Needs = append(p.Needs, "Needs a ruling: "+what) }

func (p *Pricing) basis(format string, args ...any) {
	p.Basis = append(p.Basis, fmt.Sprintf(format, args...))
}

func (p *Pricing) note(format string, args ...any) {
	p.Notes = append(p.Notes, fmt.Sprintf(format, args...))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func percentWords(rate leaguerules.Decimal) string {
	whole, frac := int64(rate)/100, int64(rate)%100
	if frac == 0 {
		return fmt.Sprintf("%d%%", whole)
	}
	return strings.TrimRight(fmt.Sprintf("%d.%02d", whole, frac), "0") + "%"
}

// sumMoney adds amounts without wrapping past the supported range.
func sumMoney(amounts []domain.Money) (domain.Money, error) {
	var total big.Int
	for _, amount := range amounts {
		total.Add(&total, big.NewInt(int64(amount)))
	}
	return checkedCapMoney(&total)
}

// shareOfAverage is rate × the mean of the amounts: exact cents first, half away from zero, then the
// league's increment. The mean is never rounded on its own.
func shareOfAverage(amounts []domain.Money, rate leaguerules.Decimal, quantum domain.Money) (domain.Money, error) {
	if len(amounts) == 0 || quantum <= 0 || rate < 0 {
		return 0, nil
	}
	var cents big.Int
	for _, amount := range amounts {
		cents.Add(&cents, big.NewInt(int64(amount)))
	}
	cents.Mul(&cents, big.NewInt(int64(rate)))
	roundCapQuotient(&cents, big.NewInt(int64(len(amounts))*10000))
	roundCapQuotient(&cents, big.NewInt(int64(quantum)))
	cents.Mul(&cents, big.NewInt(int64(quantum)))
	return checkedCapMoney(&cents)
}
