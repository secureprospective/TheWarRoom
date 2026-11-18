package market

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestWhatIfStackMatchesSequentialContractQuotesAndRunway(t *testing.T) {
	b := rolloverLedger(t)
	b.set("calendar.timezone", `"UTC"`, "now", seasonMoment(5))
	b.set("calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut"]}}`, "now", seasonMoment(6))
	raw := RawWhatIfStack{League: rolloverLeague, ViewerZone: "UTC", Moves: []RawWhatIfMove{
		{Player: "0002", Action: "cut"}, {Player: "0003", Action: "cut"},
	}}
	now := b.records[len(b.records)-1].Receipt.RecordedAt
	got, err := QuoteWhatIfStack(raw, b.records, now, time.UTC, nil, "franchise.02")
	if err != nil {
		t.Fatal(err)
	}
	base, err := commissioner.ReadNativeBase(raw.League, b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	manual := base
	for i, move := range raw.Moves {
		input := RawContractQuote{League: raw.League, Franchise: "franchise.02", ViewerZone: "UTC",
			Player: move.Player, Action: move.Action}
		quote, err := quoteContractOnBase(input, b.records, base.Now, time.UTC, nil, manual,
			nativecheck.Trail{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !quote.Allowed || !got.Moves[i].Allowed || quote.Reason != got.Moves[i].Reason ||
			!reflect.DeepEqual(quote.DeadMoney, got.Moves[i].DeadMoney) {
			t.Fatalf("move %d quote differs from sequential engine: got %+v want %+v", i, got.Moves[i], quote)
		}
		seed := *manual.Assets.Snapshot
		seed.Players = make([]commissioner.NativePlayer, 0, len(manual.Assets.Snapshot.Players)-1)
		for _, player := range manual.Assets.Snapshot.Players {
			if player.ID != move.Player || player.Franchise != "franchise.02" {
				seed.Players = append(seed.Players, player)
			}
		}
		for _, dead := range quote.DeadMoney {
			seed.Adjustments = append(seed.Adjustments, commissioner.NativeCharge{ID: "manual.dead." + move.Player,
				Kind: "dead_money", Season: dead.Season, Franchise: "franchise.02", Amount: dead.Cents})
		}
		assets := manual.Assets
		assets.Snapshot = &seed
		manual.Assets = assets
	}
	want, err := commissioner.NativeCapRunway(manual, "franchise.02")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.After, want) || len(got.Before) == 0 || len(got.Moves) != len(raw.Moves) {
		t.Fatalf("stack runway differs from sequential application: got %+v want %+v", got.After, want)
	}
}

func TestWhatIfStackCarriesRestructureScheduleIntoLaterCut(t *testing.T) {
	b := rolloverLedger(t)
	player := &b.records[1].Command.Args.Native.Players[2]
	player.Salary, player.ExpirationYear, player.ContractStatus = 10*quoteMillion, 2028, domain.CStatusUFA
	b.set("calendar.timezone", `"UTC"`, "now", seasonMoment(5))
	b.set("calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["restructure","cut"]}}`, "now", seasonMoment(6))
	raw := RawWhatIfStack{League: rolloverLeague, ViewerZone: "UTC", Moves: []RawWhatIfMove{
		{Player: "0003", Action: "restructure", MoveCents: quoteMillion},
		{Player: "0003", Action: "cut"},
	}}
	now := b.records[len(b.records)-1].Receipt.RecordedAt
	got, err := QuoteWhatIfStack(raw, b.records, now, time.UTC, nil, "franchise.02")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Moves) != 2 || !got.Moves[0].Allowed || !got.Moves[1].Allowed || len(got.Moves[1].DeadMoney) != 1 {
		t.Fatalf("sequential restructure and cut: %+v", got.Moves)
	}
	base, err := commissioner.ReadNativeBase(raw.League, b.records, now)
	if err != nil {
		t.Fatal(err)
	}
	pricing, err := nativecheck.PriceContract(nativecheck.Input{Rules: base.Rules,
		Trail: nativecheck.Trail{Restructures: 1}, Contract: nativecheck.Contract{Player: "0003",
			Name: "Starter", Salary: got.Moves[0].Salaries[0].Cents, Expiration: 2028,
			Status: domain.CStatusUFA, Schedule: got.Moves[0].Salaries}}, nativecheck.ActionCut)
	if err != nil || len(pricing.Dead) != 1 || pricing.Dead[0].Cents != got.Moves[1].DeadMoney[0].Cents {
		t.Fatalf("stacked cut does not use the reshaped contract: %+v err=%v", pricing, err)
	}
	want, err := commissioner.NativeCapRunwayProjected(base, "franchise.02", commissioner.NativeCapProjection{
		Salaries: map[string][]nativecheck.SeasonAmount{"0003": got.Moves[0].Salaries},
		Removed:  map[string]bool{"0003": true},
		Charges: []commissioner.NativeCharge{{ID: "sequence.dead", Kind: "dead_money", Season: got.Moves[1].DeadMoney[0].Season,
			Franchise: "franchise.02", Amount: got.Moves[1].DeadMoney[0].Cents}},
	})
	if err != nil || !reflect.DeepEqual(got.After, want) {
		t.Fatalf("runway does not match the applied contract schedule: got %+v want %+v err=%v", got.After, want, err)
	}
}

func TestWhatIfStackRejectsAnotherFranchisePlayer(t *testing.T) {
	b := rolloverLedger(t)
	raw := RawWhatIfStack{League: rolloverLeague, ViewerZone: "UTC", Moves: []RawWhatIfMove{{Player: "0002", Action: "cut"}}}
	_, err := QuoteWhatIfStack(raw, b.records, seasonMoment(5), time.UTC, nil, "franchise.01")
	if err == nil {
		t.Fatal("quoted a player outside the requested owner's franchise")
	}
}

func TestWhatIfStackBoundsAndValidatesMoves(t *testing.T) {
	good := RawWhatIfStack{League: rolloverLeague, ViewerZone: "UTC"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.Moves = []RawWhatIfMove{{Player: "2", Action: "cut"}}
	if err := bad.Validate(); err == nil {
		t.Fatal("accepted a malformed player ID")
	}
	bad = good
	bad.Moves = make([]RawWhatIfMove, maxWhatIfMoves+1)
	if err := bad.Validate(); err == nil {
		t.Fatal("accepted an unbounded move stack")
	}
}
