package market

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func contractDecisionLedger(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := newLedger(t, commissioner.PresetLegacy)
	seed := ledgerSnapshot()
	seed.Players[0].ContractStatus = domain.CStatusRFA
	b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: seed}, Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	b.set("calendar.windows", `{"owner_decisions":{"start":{"month":10,"day":1},"end":{"month":10,"day":30},"opens":["rfa_tender","resign"]}}`, "now", seasonMoment(4))
	return b
}

func contractDecisionRaw(b *ledgerBuilder, action string, level, years int) commissioner.RawCommand {
	return commissioner.RawCommand{League: rolloverLeague, Command: commissioner.ContractDecisionSet,
		Args: commissioner.RawArgs{ContractDecision: &commissioner.RawContractDecision{Franchise: "franchise.02",
			Player: "0001", Action: action, Level: level, Years: years}},
		Expected: int64(len(b.records)), Request: "decision." + action, Reason: "Owner contract choice",
		Effective: "now", Visibility: "private"}
}

func prepareContractDecision(t *testing.T, b *ledgerBuilder, raw commissioner.RawCommand, now time.Time) commissioner.RawCommand {
	t.Helper()
	prepared, err := raw.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	if err := prepared.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := PrepareContractDecisionAppend(&prepared, b.records, now, nil, time.UTC); err != nil {
		t.Fatal(err)
	}
	if err := prepared.Validate(); err != nil {
		t.Fatalf("prepared event is invalid: %v", err)
	}
	return prepared
}

func TestContractDecisionsReadQuotesRuleEnabledOptionsAndDeadline(t *testing.T) {
	b := contractDecisionLedger(t)
	now := seasonMoment(13)
	reading, err := ContractDecisionsReading(RawContractDecisionsScope{League: rolloverLeague, ViewerZone: "UTC"},
		"franchise.02", b.records, now, time.UTC, nil)
	if err != nil {
		t.Fatal(err)
	}
	if reading.Season != 2026 || len(reading.Players) != 1 || reading.Players[0].Player != "0001" ||
		reading.Players[0].Decision != nil {
		t.Fatalf("expiring owner contracts: %+v", reading)
	}
	options := map[string]ContractDecisionOption{}
	for _, option := range reading.Players[0].Options {
		options[option.Action] = option
	}
	for _, action := range []string{"rfa_tender", "resign"} {
		option, ok := options[action]
		if !ok || !option.Open || option.Past || option.Deadline.In(time.UTC).Day() != 31 ||
			len(option.Choices) == 0 || !option.Choices[0].Quote.Priceable || option.Choices[0].Quote.Cap == nil {
			t.Fatalf("rule-priced %s option: %+v", action, option)
		}
	}
	if _, ok := options["walk"]; ok || !strings.Contains(reading.Words, "No walk action is defined") {
		t.Fatalf("invented a walk action: %+v", reading)
	}
}

func TestContractDecisionReplayReplacesChoiceExactlyOnce(t *testing.T) {
	b := contractDecisionLedger(t)
	now := seasonMoment(13)
	first := prepareContractDecision(t, b, contractDecisionRaw(b, "rfa_tender", 1, 0), now)
	firstEntry := b.add(first, now)
	assets, err := commissioner.NativeAssetsReading(rolloverLeague, b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	player := assets.Snapshot.Players[assets.Snapshot.PlayerOf("franchise.02", "0001")]
	if player.ExpirationYear != 2027 || len(player.ContractSchedule) != 2 || player.ContractSchedule[0].Season != 2026 ||
		player.ContractSchedule[1].Season != 2027 || player.Salary != 100 {
		t.Fatalf("tender was not projected once: %+v", player)
	}

	secondRaw := contractDecisionRaw(b, "resign", 1, 1)
	secondRaw.Request = "decision.resign"
	second := prepareContractDecision(t, b, secondRaw, now.Add(time.Minute))
	secondEntry := b.add(second, now.Add(time.Minute))
	assets, err = commissioner.NativeAssetsReading(rolloverLeague, b.records, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	player = assets.Snapshot.Players[assets.Snapshot.PlayerOf("franchise.02", "0001")]
	if player.ExpirationYear != second.Args.ContractDecisionOutcome.After.ExpirationYear ||
		player.ContractSchedule[1].Cents != second.Args.ContractDecisionOutcome.Pricing.Schedule[0].Cents {
		t.Fatalf("replacement choice was not applied once: %+v", player)
	}
	latest, ok := commissioner.LatestContractDecision(b.records, "franchise.02", "0001", 2026)
	if !ok || latest.Receipt.Entry != secondEntry || firstEntry == secondEntry {
		t.Fatalf("latest choice: %+v %v", latest, ok)
	}
	reading, err := ContractDecisionsReading(RawContractDecisionsScope{League: rolloverLeague, ViewerZone: "UTC"},
		"franchise.02", b.records, now.Add(time.Minute), time.UTC, nil)
	if err != nil || len(reading.Players) != 1 || reading.Players[0].Decision == nil || reading.Players[0].Decision.Action != "resign" {
		t.Fatalf("owner reread: %+v %v", reading, err)
	}

	reverse := func(entry, request string, at time.Time) {
		t.Helper()
		b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.Revert,
			Args: commissioner.RawArgs{Entry: entry}, Expected: int64(len(b.records)), Request: request,
			Reason: "Reverse a contract choice", Effective: "now", Visibility: "private"}, at)
	}
	reverse(secondEntry, "reverse.resign", now.Add(2*time.Minute))
	assets, err = commissioner.NativeAssetsReading(rolloverLeague, b.records, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	player = assets.Snapshot.Players[assets.Snapshot.PlayerOf("franchise.02", "0001")]
	latest, ok = commissioner.LatestContractDecision(b.records, "franchise.02", "0001", 2026)
	if player.ExpirationYear != 2027 || !ok || latest.Receipt.Entry != firstEntry {
		t.Fatalf("reversing replacement did not restore prior active choice: %+v, %+v %v", player, latest, ok)
	}
	reverse(firstEntry, "reverse.tender", now.Add(3*time.Minute))
	assets, err = commissioner.NativeAssetsReading(rolloverLeague, b.records, now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	player = assets.Snapshot.Players[assets.Snapshot.PlayerOf("franchise.02", "0001")]
	if player.ExpirationYear != 2026 || player.Salary != 100 {
		t.Fatalf("reversing final active choice did not restore original contract: %+v", player)
	}
	reading, err = ContractDecisionsReading(RawContractDecisionsScope{League: rolloverLeague, ViewerZone: "UTC"},
		"franchise.02", b.records, now.Add(3*time.Minute), time.UTC, nil)
	if err != nil || len(reading.Players) != 1 || reading.Players[0].Decision != nil {
		t.Fatalf("reversal remained visible as a current choice: %+v %v", reading, err)
	}
}

func TestContractDecisionReadKeepsSavedChoiceWhenRulesDisableAllOptions(t *testing.T) {
	b := contractDecisionLedger(t)
	now := seasonMoment(13)
	choice := prepareContractDecision(t, b, contractDecisionRaw(b, "rfa_tender", 1, 0), now)
	b.add(choice, now)
	for i, action := range []string{"extension", "tag", "rfa_tender", "resign"} {
		off := false
		b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.ContractActionSet,
			Args: commissioner.RawArgs{ContractAction: &commissioner.RawContractAction{Action: action, On: &off,
				Params: json.RawMessage(`{}`)}}, Request: "disable." + action, Reason: "Policy changed",
			Effective: "now", Visibility: "league"}, seasonMoment(14+i))
	}
	reading, err := ContractDecisionsReading(RawContractDecisionsScope{League: rolloverLeague, ViewerZone: "UTC"},
		"franchise.02", b.records, seasonMoment(19), time.UTC, nil)
	if err != nil || len(reading.Players) != 1 || reading.Players[0].Decision == nil ||
		reading.Players[0].Decision.Action != "rfa_tender" || len(reading.Players[0].Options) != 0 {
		t.Fatalf("saved choice disappeared when policy removed every current option: %+v %v", reading, err)
	}
}

func TestContractDecisionRefusesAfterDeadlineAndWhenRuleTurnsOff(t *testing.T) {
	b := contractDecisionLedger(t)
	var refused *commissioner.Refusal
	closed := seasonMoment(31)
	raw, err := contractDecisionRaw(b, "rfa_tender", 1, 0).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	err = PrepareContractDecisionAppend(&raw, b.records, closed, nil, time.UTC)
	if !errors.As(err, &refused) {
		t.Fatalf("past deadline accepted: %v", err)
	}

	b = contractDecisionLedger(t)
	off := false
	b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.ContractActionSet,
		Args: commissioner.RawArgs{ContractAction: &commissioner.RawContractAction{Action: "rfa_tender", On: &off,
			Params: json.RawMessage(`{}`)}}, Request: "tender.off", Reason: "League policy changed",
		Effective: "now", Visibility: "league"}, seasonMoment(5))
	raw, err = contractDecisionRaw(b, "rfa_tender", 1, 0).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	err = PrepareContractDecisionAppend(&raw, b.records, seasonMoment(13), nil, time.UTC)
	if !errors.As(err, &refused) || !strings.Contains(refused.Reason(), "turned off") {
		t.Fatalf("rule-disabled tender accepted: %v", err)
	}
}

func TestContractDecisionInputRejectsUnruledWalk(t *testing.T) {
	choice := commissioner.RawContractDecision{Franchise: "franchise.02", Player: "0001", Action: "walk"}
	if err := choice.Validate(); err == nil {
		t.Fatal("accepted a walk action that the contract rules do not define")
	}
	if len(OwnerContractDecisionActions()) != 4 || !slices.Contains(OwnerContractDecisionActions(), nativecheck.ActionTender) {
		t.Fatalf("owner actions diverged from rule-priced reads: %v", OwnerContractDecisionActions())
	}
}
