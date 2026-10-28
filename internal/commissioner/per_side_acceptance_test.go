package commissioner_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// The rev-2 acceptance fixture for per-side pools (F8). Player 1000 is on both sides of a 16-Team Mirror
// league with different contracts: franchise.01 (AFC) holds him active at $1, franchise.09 (NFC) on IR at $9.
// Player 5001 is a free agent on both sides.

const (
	afcTeam, afcOther = "franchise.01", "franchise.02"
	nfcTeam, nfcOther = "franchise.09", "franchise.10"
	samePlayer        = "1000"
)

func perSideSeed() *commissioner.RawNativeSnapshot {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	native := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []commissioner.NativePlayer{
			{ID: samePlayer, Name: "Same Player", Franchise: afcTeam, Status: domain.RosterActive, Salary: 100,
				ExpirationYear: 2027},
			{ID: samePlayer, Name: "Same Player", Franchise: nfcTeam, Status: domain.RosterIR, Salary: 900,
				ExpirationYear: 2029},
		}, Picks: []commissioner.NativePick{}, Adjustments: []commissioner.NativeCharge{}}
	divisions := []string{"AFC East", "AFC West", "NFC East", "NFC West"}
	for i := 1; i <= 16; i++ {
		id := fmt.Sprintf("franchise.%02d", i)
		native.Franchises = append(native.Franchises, commissioner.NativeFranchise{ID: id, HeldID: id,
			Name: fmt.Sprintf("Team %d", i), Division: divisions[(i-1)/4]})
		native.StandingsOrder = append(native.StandingsOrder, id)
	}
	return native
}

// reachedBy is how an acceptance league gets its per-side pools: from the 16-Team Mirror preset, or as an
// empty 16-team league whose commissioner sets every one of the preset's values by hand, each through the
// store's own rule check.
func reachedBy() []string { return []string{"preset", "by hand"} }

func mirrorBuilder(t *testing.T, how string) *ledgerBuilder {
	t.Helper()
	if how == "preset" {
		return newLedger(t, commissioner.PresetMirror16)
	}
	create, err := commissioner.RawCommand{League: rolloverLeague, Command: commissioner.Create, Args: commissioner.RawArgs{
		Name: "Ours", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 16}, Request: "create",
		Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	bundle := leaguerules.Mirror16()
	values, current := leaguerules.ReadSettings(bundle), leaguerules.Current(bundle)
	for i, field := range leaguerules.Catalogue() {
		if values[i].State == "Needs a ruling" || field.Path == "league.sides" {
			continue
		}
		raw := commissioner.RawCommand{League: rolloverLeague, Command: commissioner.RulesSet,
			Request: fmt.Sprintf("hand.%d", i), Args: commissioner.RawArgs{Field: string(field.Path), Value: current[i]},
			Reason: "Set by hand", Effective: "now", Visibility: "league"}
		if err := commissioner.CheckRulesAppend(raw, b.records, seasonMoment(1)); err != nil {
			if strings.Contains(err.Error(), "Nothing would change") {
				continue
			}
			t.Fatalf("by hand %s: %v", field.Path, err)
		}
		b.add(raw, seasonMoment(1))
	}
	return b
}

// perSideMarket is a per-side league reached one way or the other, seeded with perSideSeed, its market open.
func perSideMarket(t *testing.T, how string) *marketFixture {
	t.Helper()
	b := mirrorBuilder(t, how)
	b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: perSideSeed()}, Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	for _, rule := range [][2]string{
		{"calendar.boundary", `"start_of_day"`},
		{"roster.limits", `{"min":0,"max":100}`},
		{"cap.amount", `100000000000`},
		{"calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid","trade","ir"]}}`},
		{"calendar.tradeDeadline", `"2026-12-01T00:00:00Z"`},
		{"trades.dotSeats", `4`},
	} {
		b.set(rule[0], rule[1], "now", seasonMoment(5))
	}
	return &marketFixture{b: b, now: seasonMoment(13), setup: true,
		facts: nativecheck.Facts{samePlayer: {Name: "Same Player"}, "5001": {Name: "Free Agent"}}}
}

// try prepares a market command as the store does and returns the refusal, recording it when accepted.
func (f *marketFixture) try(t *testing.T, command string, args commissioner.RawArgs, principal string) (string, error) {
	t.Helper()
	visibility := "league"
	if commissioner.OwnerBidAction(command) != "" || command == commissioner.WaiverClaim {
		visibility = "private"
	}
	raw := commissioner.RawCommand{League: rolloverLeague, Command: command, Args: args,
		Request: fmt.Sprintf("request.%d", len(f.b.records)), Reason: "Recorded change", Effective: "now", Visibility: visibility}
	if err := raw.Args.Validate(command); err != nil {
		return "", err
	}
	var err error
	switch {
	case commissioner.BidCommand(command):
		err = market.PrepareBidAppend(&raw, f.b.records, f.now, f.facts, time.UTC, market.BidEnvironment{Season: nativecheck.BidSeason{Week: 5, WeekKnown: true}})
	case commissioner.WaiverCommand(command):
		err = market.PrepareWaiverAppend(&raw, f.b.records, f.now, f.facts, time.UTC)
	case commissioner.TradeCommand(command):
		err = market.PrepareTradeAppend(&raw, principal, f.b.records, f.now, f.facts, time.UTC)
	default:
		_, err = commissioner.NativeAssetsReading(rolloverLeague, append(slices.Clone(f.b.records),
			commissioner.Record{Command: raw, Receipt: commissioner.Receipt{Entry: "trial", RecordedAt: f.now}}))
	}
	if err != nil {
		return "", err
	}
	entry := f.b.add(raw, f.now)
	f.b.records[len(f.b.records)-1].Principal = principal
	return entry, nil
}

func (f *marketFixture) must(t *testing.T, command string, args commissioner.RawArgs, principal string) string {
	t.Helper()
	entry, err := f.try(t, command, args, principal)
	if err != nil {
		t.Fatalf("%s: %v", command, err)
	}
	return entry
}

func (f *marketFixture) book(t *testing.T) commissioner.MarketBook {
	t.Helper()
	frame, err := commissioner.NewFrame(rolloverLeague, f.b.records, f.now, f.facts, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	return frame.Book
}

func (f *marketFixture) assets(t *testing.T) *commissioner.RawNativeSnapshot {
	t.Helper()
	assets, err := commissioner.NativeAssetsReading(rolloverLeague, f.b.records, f.now)
	if err != nil {
		t.Fatal(err)
	}
	return assets.Snapshot
}

// sideState is everything a side holds of its players: its rosters and its free-agent lots, as JSON.
func sideState(t *testing.T, seed *commissioner.RawNativeSnapshot, side string) string {
	t.Helper()
	var players []commissioner.NativePlayer
	for _, p := range seed.Players {
		if seed.SideOf(p.Franchise) == side {
			players = append(players, p)
		}
	}
	var pool []commissioner.NativePoolEntry
	for _, p := range seed.Pool {
		if p.Side == side {
			pool = append(pool, p)
		}
	}
	body, err := json.Marshal([]any{players, pool})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func wantRefusal(t *testing.T, err error, words string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), words) {
		t.Fatalf("refusal %v, want %q", err, words)
	}
}

func TestPerSidePoolsHoldIndependentPlayerCopies(t *testing.T) {
	for _, how := range reachedBy() {
		t.Run(how, func(t *testing.T) { perSidePoolsHoldIndependentPlayerCopies(t, perSideMarket(t, how)) })
	}
}

func perSidePoolsHoldIndependentPlayerCopies(t *testing.T, f *marketFixture) {
	seed := f.assets(t)
	afc, nfc := seed.PlayerOn("AFC", samePlayer), seed.PlayerOn("NFC", samePlayer)
	if afc < 0 || nfc < 0 || seed.Players[afc].Salary != 100 || seed.Players[nfc].Status != domain.RosterIR {
		t.Fatalf("copies: %+v", seed.Players)
	}
	twice := *seed
	twice.Players = append(slices.Clone(seed.Players), commissioner.NativePlayer{ID: samePlayer, Franchise: afcOther,
		Status: domain.RosterActive, ExpirationYear: 2027})
	if err := twice.Validate(); err == nil {
		t.Fatal("two copies on one side were accepted")
	}
}

func TestPerSideMarketsAreIndependent(t *testing.T) {
	for _, how := range reachedBy() {
		t.Run(how, func(t *testing.T) { perSideMarketsAreIndependent(t, perSideMarket(t, how)) })
	}
}

func perSideMarketsAreIndependent(t *testing.T, f *marketFixture) {
	bid := func(franchise string, salary domain.Money) commissioner.RawArgs {
		return commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Franchise: franchise, Salary: salary, Years: 1}}
	}
	f.must(t, commissioner.BidPlace, bid(afcTeam, 500000000), "owner.afc")
	f.must(t, commissioner.BidPlace, bid(nfcTeam, 700000000), "owner.nfc")
	_, err := f.try(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: samePlayer,
		Franchise: afcOther, Salary: 500000000, Years: 1}}, "owner.afc2")
	wantRefusal(t, err, "is on a roster")
	book := f.book(t)
	if a, n := book.Bids.Open("AFC", "5001"), book.Bids.Open("NFC", "5001"); a == nil || n == nil || a == n ||
		len(a.Bids) != 1 || len(n.Bids) != 1 {
		t.Fatalf("auctions: %+v", book.Bids.Auctions)
	}
	f.now = f.now.Add(10 * 24 * time.Hour)
	_, err = f.try(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001"}}, "creator")
	wantRefusal(t, err, "name the side")
	before := sideState(t, f.assets(t), "NFC")
	f.must(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5001", Side: "AFC"}}, "creator")
	seed := f.assets(t)
	if i := seed.PlayerOn("AFC", "5001"); i < 0 || seed.Players[i].Franchise != afcTeam {
		t.Fatalf("AFC award: %+v", seed.Players)
	}
	if seed.OnSide("NFC", "5001") || sideState(t, seed, "NFC") != before {
		t.Fatal("deciding the AFC auction changed the NFC side")
	}
	if book := f.book(t); book.Bids.Open("NFC", "5001") == nil {
		t.Fatal("the NFC auction closed with the AFC one")
	}

	// Waivers: an AFC cut opens an AFC window; the NFC side neither claims him nor loses its copy.
	f.now = seasonMoment(13)
	window := f.must(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: samePlayer,
		Franchise: afcTeam}}, "owner.afc")
	_, err = f.try(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window,
		Franchise: nfcOther}}, "owner.nfc2")
	wantRefusal(t, err, "own side of the league")
	f.must(t, commissioner.WaiverClaim, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window,
		Franchise: afcOther}}, "owner.afc2")
	if i := f.assets(t).PlayerOn("NFC", samePlayer); i < 0 || f.assets(t).Players[i].Franchise != nfcTeam {
		t.Fatal("the AFC cut touched the NFC copy")
	}
	f.now = f.now.Add(7 * 24 * time.Hour)
	f.must(t, commissioner.WaiverProcess, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Window: window}}, "creator")
	seed = f.assets(t)
	if i := seed.PlayerOn("AFC", samePlayer); i < 0 || seed.Players[i].Franchise != afcOther || seed.Players[i].Salary != 100 {
		t.Fatalf("AFC claim: %+v", seed.Players)
	}
	if i := seed.PlayerOn("NFC", samePlayer); seed.Players[i].Salary != 900 || seed.Players[i].Status != domain.RosterIR {
		t.Fatalf("NFC copy: %+v", seed.Players[i])
	}
}

func TestPerSideChangesStayOnTheirSide(t *testing.T) {
	for _, how := range reachedBy() {
		t.Run(how, func(t *testing.T) { perSideChangesStayOnTheirSide(t, perSideMarket(t, how)) })
	}
}

func perSideChangesStayOnTheirSide(t *testing.T, f *marketFixture) {
	before := sideState(t, f.assets(t), "NFC")
	correction := func(franchise string) commissioner.RawArgs {
		return commissioner.RawArgs{Correction: &commissioner.RawCorrection{Player: samePlayer, Franchise: franchise,
			Via: "trade"}}
	}
	_, err := f.try(t, commissioner.CorrectProvenance, correction(""), "creator")
	wantRefusal(t, err, "name the franchise or side")
	f.must(t, commissioner.CorrectProvenance, correction(afcTeam), "creator")
	years, salary := 3, domain.Money(250)
	edit := func(franchise string) commissioner.RawArgs {
		return commissioner.RawArgs{Override: &commissioner.RawNativeOverride{Player: samePlayer, Franchise: franchise,
			Years: &years, Salary: &salary}}
	}
	_, err = f.try(t, commissioner.ContractEdit, edit(""), "creator")
	wantRefusal(t, err, "name the franchise or side")
	f.must(t, commissioner.ContractEdit, edit(afcTeam), "creator")
	if i := f.assets(t).PlayerOn("AFC", samePlayer); f.assets(t).Players[i].Salary != 250 {
		t.Fatal("the AFC contract edit did not apply")
	}
	f.must(t, commissioner.RosterCut, commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: samePlayer,
		Franchise: afcTeam}}, "owner.afc")
	seed := f.assets(t)
	if seed.OnSide("AFC", samePlayer) || sideState(t, seed, "NFC") != before {
		t.Fatalf("AFC changes reached the NFC side: %s", sideState(t, seed, "NFC"))
	}
	_, err = f.try(t, commissioner.TradeOffer, commissioner.RawArgs{Trade: &commissioner.RawTradeArgs{Franchise: nfcTeam,
		Parties: []string{nfcTeam, afcOther}, Assets: []commissioner.TradeAsset{{From: nfcTeam, To: afcOther,
			Player: samePlayer}}}}, "owner.nfc")
	wantRefusal(t, err, "own side of the league")
}

func TestPerSideQuoteNamesAmbiguousFranchises(t *testing.T) {
	f := perSideMarket(t, "preset")
	_, err := market.QuoteContract(market.RawContractQuote{League: rolloverLeague, Player: samePlayer,
		Action: "cut", ViewerZone: "UTC"}, f.b.records, f.now, time.UTC, f.facts)
	if err == nil || !strings.Contains(err.Error(), afcTeam) || !strings.Contains(err.Error(), nfcTeam) {
		t.Fatalf("ambiguous quote should name both franchises: %v", err)
	}
	for _, tc := range []struct{ franchise, name string }{{afcTeam, "Team 1"}, {nfcTeam, "Team 9"}} {
		q, err := market.QuoteContract(market.RawContractQuote{League: rolloverLeague, Player: samePlayer,
			Franchise: tc.franchise, Action: "cut", ViewerZone: "UTC"}, f.b.records, f.now, time.UTC, f.facts)
		if err != nil || q.Franchise != tc.name {
			t.Fatalf("quote %s: %+v, %v", tc.franchise, q, err)
		}
	}
}

func TestPerSideReserveExceptionOnlyAffectsNamedFranchise(t *testing.T) {
	f := perSideMarket(t, "preset")
	for i := range f.b.records {
		if f.b.records[i].Command.Command == commissioner.NativeSeed {
			f.b.records[i].Command.Args.Native.Players[0].Status = domain.RosterIR
		}
	}
	f.must(t, commissioner.ReserveException, commissioner.RawArgs{Override: &commissioner.RawNativeOverride{
		Player: samePlayer, Franchise: afcTeam, Bucket: "ir"}}, "commissioner")
	view, err := commissioner.NativeReserveCheckReading(rolloverLeague, f.b.records, f.now, f.facts)
	if err != nil {
		t.Fatal(err)
	}
	for _, seat := range view.Franchises {
		if seat.Franchise != afcTeam && seat.Franchise != nfcTeam {
			continue
		}
		if len(seat.Players) != 1 {
			t.Fatalf("%s: %+v", seat.Franchise, seat.Players)
		}
		got := strings.Contains(seat.Players[0].Words, "commissioner exception")
		if got != (seat.Franchise == afcTeam) {
			t.Fatalf("%s: %+v", seat.Franchise, seat.Players[0])
		}
	}
}

func TestPerSidePlayerDecisionExceptionOnlyAffectsNamedCopy(t *testing.T) {
	f := perSideMarket(t, "preset")
	scope := &commissioner.ExceptionScope{Franchise: afcTeam, Player: samePlayer}
	raw := commissioner.RawCommand{League: rolloverLeague, Command: commissioner.RuleException,
		Args:    commissioner.RawArgs{Correction: &commissioner.RawCorrection{Rule: "ir.eligible", Scope: scope}},
		Request: "copy.exception", Reason: "Exception", Effective: "now", Visibility: "league"}
	if err := commissioner.CheckCorrectionAppend(raw, f.b.records, f.now); err != nil {
		t.Fatal(err)
	}
	f.b.add(raw, f.now)
	for _, franchise := range []string{afcTeam, nfcTeam} {
		got := commissioner.ScopedDecisionExceptions(f.b.records,
			commissioner.ExceptionScope{Franchise: franchise, Player: samePlayer}, f.now)
		if (len(got) == 1) != (franchise == afcTeam) {
			t.Fatalf("%s: %+v", franchise, got)
		}
	}
}

func TestPerSideIgnoresLegacyUnscopedPlayerException(t *testing.T) {
	f := perSideMarket(t, "preset")
	f.b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.RuleException,
		Args: commissioner.RawArgs{Correction: &commissioner.RawCorrection{Rule: "ir.eligible",
			Scope: &commissioner.ExceptionScope{Player: samePlayer}}}, Request: "legacy.exception",
		Reason: "Old exception", Effective: "now", Visibility: "league"}, f.now)
	for _, franchise := range []string{afcTeam, nfcTeam} {
		got := commissioner.ScopedDecisionExceptions(f.b.records,
			commissioner.ExceptionScope{Franchise: franchise, Player: samePlayer}, f.now)
		if len(got) != 0 {
			t.Fatalf("legacy exception affected %s: %+v", franchise, got)
		}
	}
}
