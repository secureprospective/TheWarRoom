package commissioner

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func TestNativeCapRunwayUsesRecordedContractSchedule(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	first, second := domain.Money(2_000_000_000), domain.Money(3_000_000_000)
	base := NativeBase{Assets: NativeAssets{Seeded: true, Snapshot: &RawNativeSnapshot{
		Season: 2026,
		Players: []NativePlayer{{ID: "0099", Franchise: "franchise.01", Status: domain.RosterActive,
			Salary: first, ExpirationYear: 2027, ContractSchedule: []nativecheck.SeasonAmount{
				{Season: 2026, Cents: first}, {Season: 2027, Cents: second},
			}}},
	}}, Rules: rules}
	got, err := NativeCapRunway(base, "franchise.01")
	if err != nil {
		t.Fatal(err)
	}
	want, err := nativecheck.CheckCap(rules.Cap, []nativecheck.CapPlayer{{Salary: second, Status: domain.RosterActive}}, nil, time.UTC)
	if err != nil || len(got) < 2 || got[1].CommittedCents == nil || want.CommittedCents == nil ||
		*got[1].CommittedCents != *want.CommittedCents {
		t.Fatalf("recorded schedule was not used in the runway: got %+v, want %+v, err=%v", got, want, err)
	}
}

func TestNativeCapRunwayProjectionCountsScheduledReserveSalaryAndDeadMoney(t *testing.T) {
	rules := leaguerules.LegacyNFL()
	rules.Season = 2026
	salary := domain.Money(2_000_000_000)
	dead := domain.Money(300_000_000)
	base := NativeBase{Assets: NativeAssets{Seeded: true, Snapshot: &RawNativeSnapshot{Season: 2026,
		Players: []NativePlayer{{ID: "0099", Franchise: "franchise.01", Status: domain.RosterIR,
			Salary: salary, ExpirationYear: 2027}}, Adjustments: []NativeCharge{}}}, Rules: rules}
	got, err := NativeCapRunwayProjected(base, "franchise.01", NativeCapProjection{
		Salaries: map[string][]nativecheck.SeasonAmount{"0099": {{Season: 2026, Cents: salary},
			{Season: 2027, Cents: salary + 1_000_000_000}}},
		Charges: []NativeCharge{{ID: "stack.dead", Kind: "dead_money", Franchise: "franchise.01",
			Season: 2027, Amount: dead}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 || got[1].Season != 2027 || got[1].CommittedCents == nil || got[1].DeadCents == nil {
		t.Fatalf("projected runway lacks the stacked year: %+v", got)
	}
	want, err := nativecheck.CheckCap(rules.Cap,
		[]nativecheck.CapPlayer{{Salary: salary + 1_000_000_000, Status: domain.RosterIR}}, []domain.Money{dead}, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	live, err := nativecheck.CheckCap(rules.Cap,
		[]nativecheck.CapPlayer{{Salary: salary + 1_000_000_000, Status: domain.RosterIR}}, nil, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	countedDead, hasDead, err := nativecheck.CountedChange(live, want)
	if err != nil || !hasDead || *got[1].CommittedCents != *live.CommittedCents || *got[1].DeadCents != countedDead {
		t.Fatalf("projected runway = %+v; want commitments %v and dead money %d (err %v)", got[1],
			want.CommittedCents, countedDead, err)
	}
}
