package composition

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// quoteClock is inside the test window below and at the store's clock, so rules and gate read one instant.
func quoteClock() time.Time { return time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC) }

// quoteApp seeds one native league (franchise.02 holds the players named below) with the Legacy NFL rules:
//
//	0099  QB, $10M through 2029   0098  QB, $8M through 2026   0097  WR, $0.5M through 2026, restricted
//	0094..0096  three more quarterbacks on franchise.03, so a tag has five salaries to average
func quoteApp(t *testing.T) *legalityFixture {
	t.Helper()
	a := nativeSeedTestApp(t)
	path := filepath.Join(t.TempDir(), "quote.db")
	store, pools := openDirectoryHistory(t, path)
	a.core.History = store
	snap := state.MirrorSnapshot{Season: 2026}
	for _, p := range []struct {
		id, franchise string
		salary        domain.Money
		year          int
		status        domain.ContractStatus
	}{{"0099", "0002", 1_000_000_000, 2029, domain.CStatusUFA}, {"0098", "0002", 800_000_000, 2026, domain.CStatusUFA},
		{"0097", "0002", 50_000_000, 2026, domain.CStatusRFA}, {"0096", "0003", 500_000_000, 2028, domain.CStatusUFA},
		{"0095", "0003", 400_000_000, 2028, domain.CStatusUFA}, {"0094", "0003", 300_000_000, 2028, domain.CStatusUFA}} {
		snap.Players = append(snap.Players, state.MirrorPlayer{MFLID: p.id, FranchiseID: p.franchise,
			RosterStatus: domain.RosterActive, Salary: p.salary, ContractYear: p.year, ContractStatus: p.status})
	}
	if _, err := a.core.League.Replace(t.Context(), snap); err != nil {
		t.Fatal(err)
	}
	var raws []players.RawPlayer
	for id, position := range map[string]string{"0099": "QB", "0098": "QB", "0097": "WR", "0096": "QB",
		"0095": "QB", "0094": "QB"} {
		raws = append(raws, players.RawPlayer{ID: id, Name: "Player, " + id, Position: position, Team: "AAA", DraftYear: "2018"})
	}
	lookup, err := normalize.NewLookup(raws)
	if err != nil {
		t.Fatal(err)
	}
	a.core.Lookup, a.core.HasLookup = lookup, true
	seedNativeTest(t, a, createNativeSeedLeague(t, a, commissioner.PresetLegacy))
	f := &legalityFixture{app: a, path: path, pools: pools}
	f.set(t, "calendar.boundary", `"start_of_day"`, "now")
	f.openWindow(t, "contract-test", []leaguerules.Action{"cut", "buyout", "tag", "extension", "restructure",
		"rfa_tender", "resign"})
	return f
}

// openWindow gives the league a window that is open at quoteClock for the actions named.
func (f *legalityFixture) openWindow(t *testing.T, id string, opens []leaguerules.Action) {
	t.Helper()
	houses, err := newCommissionerAPI(f.app).TargetNativeHouseRules(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	args := commissioner.RawWindowArgs{ID: id, Start: &commissioner.RawDate{Month: 10, Day: 1},
		End: &commissioner.RawDate{Month: 10, Day: 31}, Opens: opens}
	executeCommissionerTest(t, f.app, commissioner.RawCommand{League: legalityLeague,
		Command: commissioner.WindowSet, Args: commissioner.RawArgs{Window: &args}, Expected: houses.Revision,
		Request: fmt.Sprintf("window.%d", houses.Revision), Reason: "Open the contract desk",
		Effective: "now", Visibility: "league"})
}

// quote reads the actual quote at the fixed clock, for player and action, asking whatever ask sets.
func (f *legalityFixture) quote(t *testing.T, player, action string, ask func(*market.RawContractQuote)) market.ContractQuote {
	t.Helper()
	f.app.core.ClockNow = quoteClock
	raw := market.RawContractQuote{League: legalityLeague, Player: player, Action: action, ViewerZone: "UTC"}
	if ask != nil {
		ask(&raw)
	}
	q, err := newCommissionerAPI(f.app).TargetNativeContractQuote(raw)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// legacyParams are one action's Legacy parameters as the named command takes them: plain JSON without the
// on/off switch, with the given keys replaced.
func legacyParams(t *testing.T, field string, replace map[string]string) string {
	t.Helper()
	fields, values := leaguerules.Catalogue(), leaguerules.Current(leaguerules.LegacyNFL())
	for i, f := range fields {
		if string(f.Path) != field {
			continue
		}
		var params map[string]json.RawMessage
		if err := json.Unmarshal(values[i], &params); err != nil {
			t.Fatal(err)
		}
		delete(params, "Enabled")
		for key, value := range replace {
			params[key] = json.RawMessage(value)
		}
		body, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	t.Fatalf("no such rule %s", field)
	return ""
}

// setAction records the named command for one action: its parameters, and whether it is on.
func (f *legalityFixture) setAction(t *testing.T, action, params string, on bool) commissioner.Receipt {
	t.Helper()
	return executeCommissionerTest(t, f.app, nativeActionCommand(t, f, action, params, "now", on))
}

// priceOf is the one figure a case watches, as plain text.
type priceOf func(market.ContractQuote) string

func cents(m *domain.Money) string {
	if m == nil {
		return "no price"
	}
	return fmt.Sprint(int64(*m))
}

func deadCents(q market.ContractQuote) string {
	if len(q.DeadMoney) == 0 {
		return "none"
	}
	return fmt.Sprint(int64(q.DeadMoney[0].Cents))
}

func verdict(q market.ContractQuote, figure string) string {
	if !q.Allowed {
		return "refused: " + q.Reason
	}
	return figure
}

type quoteCase struct {
	name, action, player, field string
	ask                         func(*market.RawContractQuote)
	patch                       map[string]string
	price                       priceOf
	before, after               string
}

func quoteCases() []quoteCase {
	asLevel := func(level, years int) func(*market.RawContractQuote) {
		return func(r *market.RawContractQuote) { r.Level, r.Years = level, years }
	}
	return []quoteCase{
		{name: "buyout rates", action: "buyout", player: "0099", field: "contracts.buyout",
			patch: map[string]string{"Rates": `{"2":6000,"3":9000,"4":9000}`}, price: func(q market.ContractQuote) string {
				return verdict(q, deadCents(q))
			}, before: "750000000", after: "900000000"},
		{name: "retirement rate", action: "retire", player: "0099", field: "contracts.retirement",
			patch: map[string]string{"Rate": "5000"}, price: func(q market.ContractQuote) string {
				return verdict(q, deadCents(q))
			}, before: "900000000", after: "1500000000"},
		{name: "death rate", action: "death", player: "0099", field: "contracts.death",
			patch: map[string]string{"Rate": "1000"}, price: func(q market.ContractQuote) string {
				return verdict(q, deadCents(q))
			}, before: "none", after: "300000000"},
		{name: "restructure tiers", action: "restructure", player: "0099", field: "contracts.restructure",
			patch: map[string]string{"Tiers": `[{"At":300000000,"Amount":100000000},{"At":600000000,"Amount":250000000},` +
				`{"At":1200000000,"Amount":300000000}]`},
			price: func(q market.ContractQuote) string {
				if len(q.Salaries) == 0 {
					return verdict(q, "none")
				}
				return verdict(q, fmt.Sprint(int64(q.Salaries[0].Cents)))
			}, before: "800000000", after: "750000000"},
		{name: "extension multiplier", action: "extension", player: "0099", field: "contracts.extension",
			patch: map[string]string{"Multiplier": "20000"}, price: func(q market.ContractQuote) string {
				return verdict(q, cents(q.SalaryCent))
			}, before: "1500000000", after: "2000000000"},
		{name: "tag top salaries", action: "tag", player: "0098", field: "contracts.tag",
			patch: map[string]string{"TopPlayers": "2"}, price: func(q market.ContractQuote) string {
				return verdict(q, cents(q.SalaryCent))
			}, before: "960000000", after: "900000000"},
		{name: "tender price", action: "rfa_tender", player: "0097", field: "contracts.tender", ask: asLevel(1, 0),
			patch: map[string]string{"Prices": `[{"Price":70000000,"Rounds":[5]},{"Price":100000000,"Rounds":[4]},` +
				`{"Price":150000000,"Rounds":[3]},{"Price":250000000,"Rounds":[2]},{"Price":500000000,"Rounds":[1]},` +
				`{"Price":650000000,"Rounds":[1,2]}]`},
			price:  func(q market.ContractQuote) string { return verdict(q, cents(q.SalaryCent)) },
			before: "60000000", after: "70000000"},
		{name: "re-sign premium", action: "resign", player: "0097", field: "contracts.reSign", ask: asLevel(1, 3),
			patch:  map[string]string{"Premiums": `{"1":0,"2":1500,"3":5000,"4":4500}`},
			price:  func(q market.ContractQuote) string { return verdict(q, cents(q.SalaryCent)) },
			before: "78000000", after: "90000000"},
		{name: "cap relief reasons", action: "cap_relief", player: "0099", field: "contracts.capRelief",
			ask:    func(r *market.RawContractQuote) { r.Reason, r.AmountCents = "Career-ending injury", 100_000_000 },
			patch:  map[string]string{"Reasons": `["Recurring injury"]`},
			price:  func(q market.ContractQuote) string { return verdict(q, fmt.Sprint(int64(q.Relief))) },
			before: "100000000", after: "refused: Cap relief is for one of: Recurring injury."},
	}
}

// TestNativeContractActionSettingsGovernTheQuote is the CM-009 governs proof: the named command sets an
// action's number and the quote's price follows; turning the action off refuses the quote; the compensating
// reverts bring each price back; and after a real reopen of the database the same round trip repeats.
func TestNativeContractActionSettingsGovernTheQuote(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	parent := t // a reopened database must outlive the case that reopened it
	for _, tc := range quoteCases() {
		t.Run(tc.name, func(t *testing.T) {
			read := func() string { return tc.price(f.quote(t, tc.player, tc.action, tc.ask)) }
			for pass, name := range []string{"live", "after reopening the database"} {
				if pass == 1 {
					f.reopen(parent)
				}
				if got := read(); got != tc.before {
					t.Fatalf("%s: before the ruling: %q, want %q", name, got, tc.before)
				}
				number := f.setAction(t, tc.action, legacyParams(t, tc.field, tc.patch), true)
				if got := read(); got != tc.after {
					t.Fatalf("%s: after the ruling: %q, want %q", name, got, tc.after)
				}
				off := f.setAction(t, tc.action, legacyParams(t, tc.field, tc.patch), false)
				q := f.quote(t, tc.player, tc.action, tc.ask)
				if q.Allowed || !strings.Contains(q.Reason, "turned off") || q.SalaryCent != nil ||
					len(q.DeadMoney) != 0 || len(q.Salaries) != 0 || q.Relief != 0 || q.Cap != nil {
					t.Fatalf("%s: a switched-off action must be refused with no price: %+v", name, q)
				}
				f.revert(t, off)
				if got := read(); got != tc.after {
					t.Fatalf("%s: after turning it back on: %q, want %q", name, got, tc.after)
				}
				f.revert(t, number)
				if got := read(); got != tc.before {
					t.Fatalf("%s: after the compensating revert: %q, want %q", name, got, tc.before)
				}
			}
		})
	}
}

// TestNativeReleaseChargeFollowsTheDeadMoneyRates covers the release, which has no switch of its own: the
// league's two dead-money rates are plain rules, and the quote follows both doors.
func TestNativeReleaseChargeFollowsTheDeadMoneyRates(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	read := func(t *testing.T, f *legalityFixture) string {
		q := f.quote(t, "0099", "cut", nil)
		return verdict(q, deadCents(q))
	}
	// Three remaining $10M salaries: 35% is $10.5M, 50% is $15M.
	f.governs(t, "contracts.deadCap", `5000`, read, "1050000000", "1500000000")
}

// TestNativeQuoteFollowsWhicheverDoorSetTheRuleLast shows the rules desk and the named command write one
// setting: whichever came last prices the quote, a compensation exposes the other, and a reopen agrees.
func TestNativeQuoteFollowsWhicheverDoorSetTheRuleLast(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	read := func() string {
		q := f.quote(t, "0098", "tag", nil)
		return verdict(q, cents(q.SalaryCent))
	}
	tag := func(top string) string { return legacyParams(t, "contracts.tag", map[string]string{"TopPlayers": top}) }
	full := func(top string) string {
		return strings.Replace(tag(top), `{`, `{"Enabled":true,`, 1)
	}
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		if got := read(); got != "960000000" {
			t.Fatalf("pass %d: Legacy tag price %s", pass, got)
		}
		viaRules := f.set(t, "contracts.tag", full("2"), "now")
		if got := read(); got != "900000000" {
			t.Fatalf("pass %d: rules desk did not price the quote: %s", pass, got)
		}
		viaCommand := f.setAction(t, "tag", tag("1"), true)
		if got := read(); got != "1000000000" {
			t.Fatalf("pass %d: named command did not price the quote: %s", pass, got)
		}
		f.reopen(t)
		f.revert(t, viaCommand)
		if got := read(); got != "900000000" {
			t.Fatalf("pass %d: reverting the command must expose the rules desk's value: %s", pass, got)
		}
		f.revert(t, viaRules)
		if got := read(); got != "960000000" {
			t.Fatalf("pass %d: reverting both must restore Legacy: %s", pass, got)
		}
	}
}

// TestNativeQuoteAsksTheCalendarGate keeps a priced action refused until the league's window opens it, and
// names the window in the owner's time.
func TestNativeQuoteAsksTheCalendarGate(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	open := f.quote(t, "0099", "buyout", nil)
	if !open.Allowed || deadCents(open) != "750000000" {
		t.Fatalf("buyout inside its window: %+v", open)
	}
	f.app.core.ClockNow = func() time.Time { return quoteClock().AddDate(0, 0, 30) }
	shut, err := newCommissionerAPI(f.app).TargetNativeContractQuote(market.RawContractQuote{League: legalityLeague,
		Player: "0099", Action: "buyout", ViewerZone: "UTC"})
	if err != nil {
		t.Fatal(err)
	}
	if shut.Allowed || shut.Reason != "No league window opens this action now." || deadCents(shut) != "750000000" {
		t.Fatalf("buyout after its window must be refused and still show its price: %+v", shut)
	}
	// A retirement is found by the commissioner, not taken in a window, so no window governs it.
	if q := f.quote(t, "0099", "retire", nil); !q.Allowed {
		t.Fatalf("retirement: %+v", q)
	}
}

// TestNativeOwnerStartFollowsTheSameSwitch runs the registered owner action through the gate: it starts
// while the league has the action on and the window is open, and is refused once the commissioner turns it
// off, by the same sentence the quote gives.
func TestNativeOwnerStartFollowsTheSameSwitch(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	f.app.core.ClockNow = quoteClock
	f.app.core.LocalZone = time.UTC
	registry := nativeOwnerActions()
	start := func() error {
		_, err := f.app.runNativeAction(registry, legalityLeague, "franchise.02", "contract.buyout", nil)
		return err
	}
	if err := start(); err != nil {
		t.Fatal(err)
	}
	off := f.setAction(t, "buyout", legacyParams(t, "contracts.buyout", nil), false)
	err := start()
	if err == nil || !strings.Contains(err.Error(), "This league has the contract buyout turned off.") {
		t.Fatalf("a switched-off action started: %v", err)
	}
	if q := f.quote(t, "0099", "buyout", nil); q.Allowed || !strings.Contains(err.Error(), q.Reason) {
		t.Fatalf("the start and the quote disagree: %q vs %+v", err, q)
	}
	f.revert(t, off)
	if err := start(); err != nil {
		t.Fatal(err)
	}
}

func TestNativeQuoteRefusesWhatItCannotPrice(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	for _, tc := range []struct {
		player, action, reason string
		ask                    func(*market.RawContractQuote)
	}{
		{"0099", "tag", "last year of his contract", nil},
		{"0099", "rfa_tender", "Only a restricted free agent", nil},
		{"0097", "rfa_tender", "Choose a tender level from 1 to 6.", nil},
		{"0099", "extension", "adds at most 3 years", func(r *market.RawContractQuote) { r.Years = 4 }},
		{"0099", "restructure", "At most", func(r *market.RawContractQuote) { r.MoveCents = 900_000_000 }},
	} {
		q := f.quote(t, tc.player, tc.action, tc.ask)
		if q.Allowed || !strings.Contains(q.Reason, tc.reason) {
			t.Fatalf("%s %s: %+v", tc.player, tc.action, q)
		}
	}
	for _, bad := range []market.RawContractQuote{
		{League: legalityLeague, Player: "0099", Action: "tender", ViewerZone: "UTC"},
		{League: legalityLeague, Player: "0001", Action: "buyout", ViewerZone: "UTC"},
		{League: "native.unmade", Player: "0099", Action: "buyout", ViewerZone: "UTC"},
	} {
		if _, err := newCommissionerAPI(f.app).TargetNativeContractQuote(bad); err == nil {
			t.Fatalf("quoted %+v", bad)
		}
	}
}

// TestNativeQuoteNeedsARulingNotAGuess keeps an unruled parameter out of the price: the named command
// replaces all of an action's parameters, so one it leaves out is a ruling still to make.
func TestNativeQuoteNeedsARulingNotAGuess(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	for pass := range 2 {
		if pass == 1 {
			f.reopen(t)
		}
		receipt := f.setAction(t, "buyout", `{"AnnualLimit":2}`, true)
		q := f.quote(t, "0099", "buyout", nil)
		if q.Allowed || len(q.Needs) != 1 || q.Reason != q.Needs[0] ||
			!strings.HasPrefix(q.Reason, "Needs a ruling: the buyout rates") || len(q.DeadMoney) != 0 {
			t.Fatalf("pass %d: an omitted rate must be a ruling, not a guess: %+v", pass, q)
		}
		f.revert(t, receipt)
		if q := f.quote(t, "0099", "buyout", nil); !q.Allowed || deadCents(q) != "750000000" {
			t.Fatalf("pass %d: reverting restores the price: %+v", pass, q)
		}
	}
}

func TestNativeContractQuoteWire(t *testing.T) {
	t.Parallel()
	f := quoteApp(t)
	f.setAction(t, "extension", legacyParams(t, "contracts.extension", nil), false)
	readings := map[string]market.ContractQuote{
		"buyout":       f.quote(t, "0099", "buyout", nil),
		"tag":          f.quote(t, "0098", "tag", nil),
		"restructure":  f.quote(t, "0099", "restructure", nil),
		"resign":       f.quote(t, "0097", "resign", func(r *market.RawContractQuote) { r.Level, r.Years = 1, 3 }),
		"extensionOff": f.quote(t, "0099", "extension", nil),
		"cutAtGate": func() market.ContractQuote {
			f.app.core.ClockNow = func() time.Time { return quoteClock().AddDate(0, 0, 30) }
			q, err := newCommissionerAPI(f.app).TargetNativeContractQuote(market.RawContractQuote{League: legalityLeague,
				Player: "0099", Action: "cut", ViewerZone: "America/Los_Angeles"})
			if err != nil {
				t.Fatal(err)
			}
			return q
		}(),
	}
	body, err := json.MarshalIndent(readings, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	checkWire(t, "internal/commissioner/testdata/contract-quote.wire.json", body, "contract quote")
}
