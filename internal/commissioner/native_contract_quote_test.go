package commissioner

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const quoteMillion = domain.Money(100_000_000)

// quoteBase is one franchise with an active player (0099) and a taxi player (0100), each $10M through 2029.
func quoteBase() nativeBase {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	seed := &RawNativeSnapshot{Season: 2026, Players: []NativePlayer{
		{ID: "0099", Franchise: "franchise.01", Status: domain.RosterActive, Salary: 10 * quoteMillion,
			ExpirationYear: 2029, ContractStatus: domain.CStatusUFA},
		{ID: "0100", Franchise: "franchise.01", Status: domain.RosterTaxi, Salary: 10 * quoteMillion,
			ExpirationYear: 2029, ContractStatus: domain.CStatusUFA}}}
	seat := NativeDirectorySeat{PeopleSeat: PeopleSeat{Franchise: "franchise.01", Name: "Franchise One"}}
	return nativeBase{assets: NativeAssets{Seeded: true, Snapshot: seed},
		directory: NativeDirectory{Franchises: []NativeDirectorySeat{seat}}, rules: rules}
}

func quoteOf(t *testing.T, base nativeBase, player, action string, gate Gate) ContractQuote {
	t.Helper()
	held, ok := recordedPlayer(base.assets.Snapshot, player)
	if !ok {
		t.Fatal("no such player")
	}
	q, err := checkContract(contractCheck{raw: RawContractQuote{League: "native.test", Player: player, Action: action},
		action: nativecheck.ContractAction(action), base: base, player: held, gate: gate, revision: 7})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func openGate() Gate { return Gate{Open: true} }

func TestContractQuoteWaitsForTheCalendarButStillShowsThePrice(t *testing.T) {
	base := quoteBase()
	open := quoteOf(t, base, "0099", "buyout", openGate())
	if !open.Allowed || open.Reason != "" || len(open.DeadMoney) != 1 || open.DeadMoney[0].Cents != 75*quoteMillion/10 ||
		open.Franchise != "Franchise One" || open.Cap == nil {
		t.Fatalf("open quote: %+v", open)
	}
	shut := quoteOf(t, base, "0099", "buyout", Gate{Sentence: "This action opens with the cut_buyout window on Aug 25, 2026 at 12:00 AM UTC."})
	if shut.Allowed || shut.Reason != "This action opens with the cut_buyout window on Aug 25, 2026 at 12:00 AM UTC." ||
		len(shut.DeadMoney) != 1 {
		t.Fatalf("a shut window must refuse and still show the price: %+v", shut)
	}
	// A refusal about the contract itself outranks a shut window: the window would not help.
	base.rules.Contracts.Buyout.Value.Enabled = leaguerules.Known(false)
	off := quoteOf(t, base, "0099", "buyout", Gate{Sentence: "shut"})
	if off.Allowed || !strings.Contains(off.Reason, "turned off") || len(off.DeadMoney) != 0 || off.Cap != nil {
		t.Fatalf("a switched-off action: %+v", off)
	}
}

func TestContractQuoteCapEffectAndHardCap(t *testing.T) {
	base := quoteBase()
	q := quoteOf(t, base, "0099", "buyout", openGate())
	// $20M counted before; the buyout removes $10M and charges $7.5M.
	if *q.Cap.BeforeCents != 20*quoteMillion || *q.Cap.AfterCents != 175*quoteMillion/10 ||
		*q.Cap.ChangeCents != -25*quoteMillion/10 || !strings.Contains(q.Cap.Words, "no tighter") {
		t.Fatalf("cap effect: %+v", q.Cap)
	}
	// A taxi player counts at 0 percent here, so his buyout charge is new counted salary.
	base.rules.Cap.TaxiShare = leaguerules.Known(leaguerules.Decimal(0))
	base.rules.Cap.Amount = leaguerules.Known(leaguerules.Money(15 * quoteMillion))
	hard := quoteOf(t, base, "0100", "buyout", openGate())
	if hard.Allowed || *hard.Cap.ChangeCents != 75*quoteMillion/10 ||
		hard.Reason != hard.Cap.Words || !strings.Contains(hard.Reason, "ends over the salary cap is refused") {
		t.Fatalf("a hard cap must refuse the buyout: %+v", hard)
	}
	base.rules.Cap.Amount = leaguerules.Known(leaguerules.Money(18 * quoteMillion))
	if fits := quoteOf(t, base, "0100", "buyout", openGate()); !fits.Allowed {
		t.Fatalf("a buyout that ends at $17.5M fits an $18M cap: %+v", fits)
	}
}

func TestContractQuoteSoftCapFlagsAndMissingCapRulingStops(t *testing.T) {
	base := quoteBase()
	base.rules.Cap.TaxiShare = leaguerules.Known(leaguerules.Decimal(0))
	base.rules.Cap.Amount = leaguerules.Known(leaguerules.Money(15 * quoteMillion))
	base.rules.Cap.Enforcement = leaguerules.Known(leaguerules.SoftCap)
	soft := quoteOf(t, base, "0100", "buyout", openGate())
	if !soft.Allowed || !strings.Contains(soft.Cap.Words, "Over the salary cap: flagged") {
		t.Fatalf("a soft cap flags and allows: %+v", soft)
	}
	base.rules.Cap.Amount = leaguerules.Needs[leaguerules.Money]("test")
	needs := quoteOf(t, base, "0100", "buyout", openGate())
	if needs.Allowed || len(needs.Needs) != 1 || !strings.HasPrefix(needs.Reason, "Needs a ruling: the salary cap") {
		t.Fatalf("an unruled cap must stop the quote: %+v", needs)
	}
}

func TestContractQuoteNamesAreHeldFactsOrPlain(t *testing.T) {
	base := quoteBase()
	q := quoteOf(t, base, "0099", "retire", openGate())
	if q.Name != "Player 0099" || q.ActionName != "retirement" || !q.Allowed || q.Revision != 7 || q.Season != 2026 {
		t.Fatalf("quote identity: %+v", q)
	}
}

func TestContractQuoteInputIsChecked(t *testing.T) {
	good := RawContractQuote{League: "native.test", Player: "0099", Action: "tag", ViewerZone: "UTC"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*RawContractQuote){
		func(r *RawContractQuote) { r.League = "mfl" }, func(r *RawContractQuote) { r.Player = "99" },
		func(r *RawContractQuote) { r.Action = "trade" }, func(r *RawContractQuote) { r.ViewerZone = "Local" },
		func(r *RawContractQuote) { r.Years = -1 }, func(r *RawContractQuote) { r.MoveCents = -1 },
		func(r *RawContractQuote) { r.Level = 100 },
	} {
		bad := good
		change(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}
