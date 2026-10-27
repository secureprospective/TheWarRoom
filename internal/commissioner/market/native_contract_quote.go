package market

import (
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// RawContractQuote asks what one action would cost on one native contract. Zero fields are "not said":
// each action decides which it needs.
type RawContractQuote struct {
	League      string       `json:"league"`
	Player      string       `json:"player"`
	Action      string       `json:"action"`
	ViewerZone  string       `json:"viewerZone"`
	Years       int          `json:"years"`
	Level       int          `json:"level"`
	MoveCents   domain.Money `json:"moveCents"`
	AmountCents domain.Money `json:"amountCents"`
	Reason      string       `json:"reason"`
}

func (r RawContractQuote) Validate() error {
	if err := (commissioner.RawNativeCapScope{League: r.League, ViewerZone: r.ViewerZone}).Validate(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	id, err := playerid.New(r.Player)
	if err != nil || id.String() != r.Player {
		return fmt.Errorf("choose a player")
	}
	if !nativecheck.ContractAction(r.Action).Known() {
		return fmt.Errorf("choose a contract action")
	}
	if r.Years < 0 || r.Years > 99 || r.Level < 0 || r.Level > 99 || r.MoveCents < 0 || r.AmountCents < 0 ||
		len(r.Reason) > 200 {
		return fmt.Errorf("the years, tender level and amounts must be positive and reasonable")
	}
	return nil
}

// ContractCapEffect is what the action does to the franchise's counted salary this season.
type ContractCapEffect struct {
	BeforeCents *domain.Money `json:"beforeCents"`
	AfterCents  *domain.Money `json:"afterCents"`
	ChangeCents *domain.Money `json:"changeCents"`
	RoomCents   *domain.Money `json:"roomAfterCents"`
	Words       string        `json:"words"`
}

// ContractQuote is the whole answer for one contract action: allowed or refused with the plain reason, the
// new salary and years, the dead money by season, and the effect on the cap. The owner action that follows
// runs the same check, so what is quoted is what would be done.
type ContractQuote struct {
	League     string                     `json:"league"`
	Revision   int64                      `json:"revision"`
	Season     int                        `json:"season"`
	Player     string                     `json:"player"`
	Name       string                     `json:"name"`
	Franchise  string                     `json:"franchise"`
	Action     string                     `json:"action"`
	ActionName string                     `json:"actionName"`
	Allowed    bool                       `json:"allowed"`
	Reason     string                     `json:"reason"`
	Needs      []string                   `json:"needs"`
	Years      int                        `json:"years"`
	SalaryCent *domain.Money              `json:"salaryCents"`
	Salaries   []nativecheck.SeasonAmount `json:"salaries"`
	DeadMoney  []nativecheck.SeasonAmount `json:"deadMoney"`
	Relief     domain.Money               `json:"reliefCents"`
	Cap        *ContractCapEffect         `json:"cap"`
	Basis      []string                   `json:"basis"`
	Notes      []string                   `json:"notes"`
	Source     string                     `json:"source"`
}

// windowAction is the calendar's name for an action an owner can only take while a window is open. A
// retirement, a death and a cap relief are found by the commissioner, not taken by an owner, so no
// window governs them.
func windowAction(action nativecheck.ContractAction) (leaguerules.Action, bool) {
	switch action {
	case nativecheck.ActionTag, nativecheck.ActionExtension, nativecheck.ActionRestructure,
		nativecheck.ActionBuyout, nativecheck.ActionTender, nativecheck.ActionResign, nativecheck.ActionCut:
		return leaguerules.Action(action), true
	case nativecheck.ActionRetire, nativecheck.ActionDeath, nativecheck.ActionCapRelief:
		return "", false
	}
	return "", false
}

// OwnerContractWindow names the window action an owner's start of a contract action asks the gate for.
func OwnerContractWindow(action nativecheck.ContractAction) (leaguerules.Action, bool) {
	return windowAction(action)
}

// ContractActionOn is the owner's first check before an action: the league must have it on. It is the
// same switch the quote reads, so a quote and a start can never disagree about what is on.
func ContractActionOn(records []commissioner.Record, now time.Time, action nativecheck.ContractAction) error {
	rules, err := commissioner.RulesNow(records, now)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	refusal, needs := nativecheck.Availability(rules.Contracts, action)
	switch {
	case refusal != "":
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return commissioner.Refused(errors.New(refusal))
	case needs != "":
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return commissioner.Refused(errors.New(needs))
	}
	return nil
}

// QuoteContract is the one check for a native contract action. The preview reads it; the owner action
// will run it, so both ask the calendar's gate, count the same limits, price with the same rules and judge
// the same cap. A shut gate keeps the action refused even when its price is plain.
func QuoteContract(raw RawContractQuote, records []commissioner.Record, now time.Time, local *time.Location,
	facts nativecheck.Facts) (ContractQuote, error) {
	if err := raw.Validate(); err != nil {
		return ContractQuote{}, err
	}
	base, err := commissioner.ReadNativeBase(raw.League, records, now)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return ContractQuote{}, err
	}
	if !base.Assets.Seeded {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return ContractQuote{}, commissioner.Refused(fmt.Errorf("this app league has no recorded rosters to quote"))
	}
	player, ok := recordedPlayer(base.Assets.Snapshot, raw.Player)
	if !ok {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return ContractQuote{}, commissioner.Refused(fmt.Errorf("choose a player on this app league's recorded roster"))
	}
	action := nativecheck.ContractAction(raw.Action)
	gate := commissioner.Gate{Open: true}
	if window, gated := windowAction(action); gated {
		gate, err = commissioner.NativeActionGate(raw.League, records, player.Franchise, window, now, local)
		if err != nil {
			return ContractQuote{}, fmt.Errorf("ask the league calendar: %w", err)
		}
	}
	in := contractCheck{raw: raw, action: action, base: base, player: player, gate: gate, facts: facts,
		revision: int64(len(records)), zone: local}
	return checkContract(in)
}

func recordedPlayer(seed *commissioner.RawNativeSnapshot, id string) (commissioner.NativePlayer, bool) {
	for _, p := range seed.Players {
		if p.ID == id {
			return p, true
		}
	}
	return commissioner.NativePlayer{}, false
}

// contractCheck is what checkContract judges: the rules and rosters, the player, and the calendar's answer.
type contractCheck struct {
	raw      RawContractQuote
	action   nativecheck.ContractAction
	base     commissioner.NativeBase
	player   commissioner.NativePlayer
	gate     commissioner.Gate
	facts    nativecheck.Facts
	revision int64
	zone     *time.Location
}

// checkContract prices the action and then lets the calendar and the cap have their say. It is a pure
// reading: nothing is recorded.
func checkContract(in contractCheck) (ContractQuote, error) {
	fact := in.facts[in.player.ID]
	name := fact.Name
	if name == "" {
		name = "Player " + in.player.ID
	}
	// No owner contract action is recorded in the ledger yet, so nothing has been used and no contract has
	// been restructured, extended or tagged in the app league. The recording step fills this in.
	priced, err := nativecheck.PriceContract(nativecheck.Input{Rules: in.base.Rules,
		Contract: nativecheck.Contract{Player: in.player.ID, Name: name, Position: fact.Position,
			PositionHeld: fact.Position != "", Salary: in.player.Salary, Expiration: in.player.ExpirationYear,
			Status: in.player.ContractStatus},
		Ask: nativecheck.Ask{Years: in.raw.Years, Level: in.raw.Level, Move: in.raw.MoveCents,
			Amount: in.raw.AmountCents, Reason: in.raw.Reason},
		Market: positionSalaries(in.base.Assets.Snapshot, in.facts, fact.Position)}, in.action)
	if err != nil {
		return ContractQuote{}, fmt.Errorf("price the contract action: %w", err)
	}
	q := in.quoteFrom(name, priced)
	capRefused := false
	if priced.Refusal == "" && len(priced.Needs) == 0 {
		if capRefused, err = in.addCap(&q, priced); err != nil {
			return ContractQuote{}, err
		}
	}
	in.decide(&q, priced.Refusal, capRefused)
	return q, nil
}

func (in contractCheck) quoteFrom(name string, p nativecheck.Pricing) ContractQuote {
	q := ContractQuote{League: in.raw.League, Revision: in.revision, Season: in.base.Rules.Season,
		Player: in.player.ID, Name: name, Franchise: in.player.Franchise, Action: in.raw.Action,
		ActionName: in.action.Name(), Needs: append([]string{}, p.Needs...), Years: p.Years, SalaryCent: p.Salary,
		Salaries:  append([]nativecheck.SeasonAmount{}, p.Schedule...),
		DeadMoney: append([]nativecheck.SeasonAmount{}, p.Dead...), Relief: p.Relief,
		Basis: append([]string{}, p.Basis...), Notes: append([]string{}, p.Notes...),
		Source: "App league; priced from this league's own rules and recorded contract, never from MFL"}
	for _, seat := range in.base.Directory.Franchises {
		if seat.Franchise == in.player.Franchise {
			q.Franchise = seat.Name
		}
	}
	return q
}

// decide sets the verdict. A missing ruling or a refused price comes first, because no calendar or cap
// makes an unpriceable action possible; then the calendar; then the cap.
func (in contractCheck) decide(q *ContractQuote, refusal string, capRefused bool) {
	switch {
	case refusal != "":
		q.Reason = refusal
	case len(q.Needs) > 0:
		q.Reason = q.Needs[0]
	case !in.gate.Open:
		q.Reason = in.gate.Sentence
	case capRefused:
		q.Reason = q.Cap.Words
	}
	q.Allowed = q.Reason == ""
}

func positionSalaries(seed *commissioner.RawNativeSnapshot, facts nativecheck.Facts, position leaguerules.Position) []domain.Money {
	if position == "" {
		return nil
	}
	var out []domain.Money
	for _, p := range seed.Players {
		if facts[p.ID].Position == position {
			out = append(out, p.Salary)
		}
	}
	return out
}
