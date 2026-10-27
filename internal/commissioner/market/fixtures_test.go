package market

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const roleGrant = "commish.role.grant"

func seasonMoment(day int) time.Time { return time.Date(2026, 10, day, 12, 0, 0, 0, time.UTC) }

// ledgerBuilder appends records the way the store does: ordered, each stamped with its own entry.
type ledgerBuilder struct {
	t       *testing.T
	records []commissioner.Record
}

func newLedger(t *testing.T, preset string) *ledgerBuilder {
	t.Helper()
	create, err := commissioner.RawCommand{League: "native.test", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Ours", Preset: preset,
		Season: 2026, FranchiseCount: map[string]int{commissioner.PresetLegacy: 32, commissioner.PresetEmpty: 4}[preset]}, Request: "create",
		Effective: "now", Visibility: "league"}.Prepared()
	if err != nil {
		t.Fatal(err)
	}
	b := &ledgerBuilder{t: t}
	b.add(create, seasonMoment(1))
	return b
}

func (b *ledgerBuilder) add(raw commissioner.RawCommand, at time.Time) string {
	b.t.Helper()
	entry := fmt.Sprintf("entry.%d", len(b.records)+1)
	b.records = append(b.records, commissioner.Record{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: commissioner.Receipt{Entry: entry, League: raw.League, Revision: int64(len(b.records) + 1), RecordedAt: at}})
	return entry
}

func (b *ledgerBuilder) set(field, value, effective string, at time.Time) string {
	b.t.Helper()
	return b.add(commissioner.RawCommand{League: "native.test", Command: commissioner.RulesSet, Request: fmt.Sprintf("r%d", len(b.records)),
		Args: commissioner.RawArgs{Field: field, Value: json.RawMessage(value)}, Effective: effective,
		Visibility: "league"}, at)
}

const rolloverLeague = "native.test"

// ledgerSnapshot is a seeded Legacy league's assets: franchise.02 holds one taxi player whose contract
// ends this season, one IR player and one active player; franchise.01 holds a pick owned by franchise.02.
func ledgerSnapshot() *commissioner.RawNativeSnapshot {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	native := &commissioner.RawNativeSnapshot{Season: 2026, RosterAt: at, DraftAt: at, FutureAt: at,
		Players: []commissioner.NativePlayer{
			{ID: "0001", Name: "Taxi Expiring", Franchise: "franchise.02", Status: domain.RosterTaxi, Salary: 100,
				ExpirationYear: 2026},
			{ID: "0002", Name: "Injured Veteran", Franchise: "franchise.02", Status: domain.RosterIR, Salary: 200,
				ExpirationYear: 2028},
			{ID: "0003", Name: "Starter", Franchise: "franchise.02", Status: domain.RosterActive, Salary: 300,
				ExpirationYear: 2027},
		},
		Picks:       []commissioner.NativePick{{Season: 2028, Round: 1, Original: "franchise.01", Owner: "franchise.02"}},
		Adjustments: []commissioner.NativeCharge{{ID: "held", Franchise: "franchise.02", Amount: 123}}}
	for i := 1; i <= 32; i++ {
		native.Franchises = append(native.Franchises, commissioner.NativeFranchise{ID: fmt.Sprintf("franchise.%02d", i),
			HeldID: fmt.Sprintf("%04d", i), Name: fmt.Sprintf("Team %d", i), Division: fmt.Sprintf("div.%d", (i-1)/4)})
	}
	return native
}

// rolloverLedger is a seeded Legacy league whose calendar rulings are made; time moves one day per entry.
func rolloverLedger(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := newLedger(t, commissioner.PresetLegacy)
	b.add(commissioner.RawCommand{League: rolloverLeague, Command: commissioner.NativeSeed, Args: commissioner.RawArgs{Native: ledgerSnapshot()},
		Request: "seed", Effective: "now", Visibility: "league"}, seasonMoment(2))
	b.set("calendar.boundary", `"start_of_day"`, "now", seasonMoment(3))
	b.set("calendar.tradeDeadline", `"2026-11-09T05:00:00Z"`, "now", seasonMoment(4))
	return b
}

func noChangeTime() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }

func legacyLeague(t *testing.T) []commissioner.Record {
	t.Helper()
	raw, err := (commissioner.RawCommand{League: "native.test", Command: commissioner.Create,
		Args:   commissioner.RawArgs{Name: "Ours", Preset: commissioner.PresetLegacy, FranchiseCount: 32, Season: 2026},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	return []commissioner.Record{{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: commissioner.Receipt{Entry: "create", Revision: 1, RecordedAt: noChangeTime().Add(-time.Hour)}}}
}
