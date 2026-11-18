package market

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// PrepareContractDecisionAppend repeats ownership-independent legality under the append lock and fixes
// the engine-priced effect that replay will apply.
func PrepareContractDecisionAppend(raw *commissioner.RawCommand, records []commissioner.Record, now time.Time,
	facts nativecheck.Facts, zone *time.Location) error {
	if raw.Command != commissioner.ContractDecisionSet {
		return nil
	}
	choice := raw.Args.ContractDecision
	if choice == nil || raw.Args.ContractDecisionOutcome != nil {
		return fmt.Errorf("send one owner contract choice, not its result")
	}
	if err := choice.Validate(); err != nil {
		return fmt.Errorf("contract decision: %w", err)
	}
	if zone == nil {
		return fmt.Errorf("choose the owner's time zone before deciding a contract")
	}
	window, err := openDecisionWindow(raw.League, *choice, records, now, zone)
	if err != nil {
		return err
	}
	quote, err := QuoteContract(RawContractQuote{League: raw.League, Franchise: choice.Franchise,
		Player: choice.Player, Action: choice.Action, ViewerZone: zone.String(), Years: choice.Years,
		Level: choice.Level, MoveCents: choice.MoveCents, AmountCents: choice.AmountCents},
		records, now, zone, facts)
	if err != nil {
		return err
	}
	if !quote.Allowed || !quote.Priceable {
		return fmt.Errorf("contract decision: %w", commissioner.Refused(fmt.Errorf("%s", quote.Reason)))
	}
	before, err := decisionBefore(raw.League, *choice, quote.Season, records, now)
	if err != nil {
		return err
	}
	pricing := nativecheck.Pricing{Years: quote.Years, Salary: quote.SalaryCent,
		Schedule: slices.Clone(quote.Salaries), Dead: slices.Clone(quote.DeadMoney), Relief: quote.Relief,
		Basis: slices.Clone(quote.Basis), Notes: slices.Clone(quote.Notes)}
	if len(pricing.Schedule) == 0 {
		return fmt.Errorf("contract decision: %w",
			commissioner.Refused(fmt.Errorf("the contract engine returned no future salary schedule")))
	}
	after := decisionAfter(before, pricing, quote.Season)
	if nativecheck.ContractAction(choice.Action) == nativecheck.ActionTag {
		after.ContractStatus = domain.CStatusFT1
		if contractDecisionTrail(records, choice.Franchise, choice.Player, quote.Season).ConsecutiveTags > 0 {
			after.ContractStatus = domain.CStatusFT2
		}
	}
	raw.Args.ContractDecisionOutcome = &commissioner.ContractDecisionOutcome{Before: before, After: after,
		Pricing: pricing, Season: quote.Season, Deadline: window.deadline}
	return nil
}

// openDecisionWindow is the league calendar's open window for this choice, or the calendar's own refusal.
func openDecisionWindow(league string, choice commissioner.RawContractDecision, records []commissioner.Record,
	now time.Time, zone *time.Location) (decisionWindowDetails, error) {
	action := nativecheck.ContractAction(choice.Action)
	if _, ok := OwnerContractWindow(action); !ok {
		return decisionWindowDetails{}, fmt.Errorf("the league calendar does not open this contract action")
	}
	rules, err := commissioner.RulesNow(records, now)
	if err != nil {
		return decisionWindowDetails{}, fmt.Errorf("read contract decision season: %w", err)
	}
	window, ok := decisionWindow(records, league, choice.Franchise, action, rules.Season, now, zone)
	if ok && window.open && now.Before(window.deadline) {
		return window, nil
	}
	gate, err := commissioner.NativeActionGate(league, records, choice.Franchise, leaguerules.Action(action), now, zone)
	if err != nil {
		return decisionWindowDetails{}, fmt.Errorf("read the contract deadline: %w", err)
	}
	sentence := gate.Sentence
	if sentence == "" {
		sentence = "the contract decision deadline passed"
	}
	return decisionWindowDetails{}, fmt.Errorf("contract decision: %w", commissioner.Refused(errors.New(sentence)))
}

// decisionBefore is the recorded contract the choice replaces: the season's original terms when an earlier
// choice already changed them, so a replacement applies against the same contract exactly once.
func decisionBefore(league string, choice commissioner.RawContractDecision, season int,
	records []commissioner.Record, now time.Time) (commissioner.NativePlayer, error) {
	refuse := func(words string) (commissioner.NativePlayer, error) {
		return commissioner.NativePlayer{}, fmt.Errorf("contract decision: %w", commissioner.Refused(errors.New(words)))
	}
	base, err := commissioner.ReadNativeBase(league, records, now)
	if err != nil {
		return commissioner.NativePlayer{}, fmt.Errorf("read contract decision base: %w", err)
	}
	if season != base.Rules.Season || base.Assets.Snapshot == nil {
		return refuse("choose a player expiring in the current league season")
	}
	before, found := commissioner.ContractDecisionBaseline(records, choice.Franchise, choice.Player, season)
	if !found {
		index := base.Assets.Snapshot.PlayerOf(choice.Franchise, choice.Player)
		if index < 0 {
			return refuse("choose a player on your franchise")
		}
		before = base.Assets.Snapshot.Players[index]
	}
	if before.ExpirationYear != season || before.Pending || before.ContractMissing {
		return refuse("choose a player whose recorded contract expires this season")
	}
	return before, nil
}

// decisionAfter lays the engine's priced schedule over the recorded contract, keeping this season's salary
// when the new terms start next season.
func decisionAfter(before commissioner.NativePlayer, pricing nativecheck.Pricing, season int) commissioner.NativePlayer {
	after := before
	after.ExpirationYear = pricing.Schedule[len(pricing.Schedule)-1].Season
	after.ContractSchedule = slices.Clone(pricing.Schedule)
	if pricing.Schedule[0].Season <= season {
		return after
	}
	currentSalary := before.Salary
	for _, year := range before.ContractSchedule {
		if year.Season == season {
			currentSalary = year.Cents
			break
		}
	}
	after.ContractSchedule = append([]nativecheck.SeasonAmount{{Season: season, Cents: currentSalary}},
		after.ContractSchedule...)
	return after
}
