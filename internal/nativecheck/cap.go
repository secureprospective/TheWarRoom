package nativecheck

import (
	"fmt"
	"math/big"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// CapPlayer is a native contract, not a contract-action price or an MFL cap charge.
type CapPlayer struct {
	Salary domain.Money
	Status domain.RosterStatus
}

// CapState keeps unknown amounts absent rather than presenting them as zero. Reserve amounts are
// the counted shares; salary is the full contracted salary before reserve shares and adjustments.
type CapState struct {
	CapCents        *domain.Money       `json:"capCents"`
	SalaryCents     *domain.Money       `json:"salaryCents"`
	ActiveCents     *domain.Money       `json:"activeCents"`
	IRCents         *domain.Money       `json:"irCents"`
	TaxiCents       *domain.Money       `json:"taxiCents"`
	AdjustmentCents *domain.Money       `json:"adjustmentCents"`
	CommittedCents  *domain.Money       `json:"committedCents"`
	RoomCents       *domain.Money       `json:"roomCents"`
	QuantumCents    *domain.Money       `json:"quantumCents"`
	Mode            leaguerules.CapMode `json:"mode"`
	Deadline        *time.Time          `json:"deadline"`
	Words           string              `json:"words"`
	Enforcement     string              `json:"enforcement"`
}

// CheckCap counts each salary at its squad's share, rounds percentage math to cents, then snaps each
// resulting salary or signed charge. Already-rounded entries are summed, never rounded as one pool.
func CheckCap(rules leaguerules.CapRules, players []CapPlayer, charges []domain.Money,
	viewer *time.Location) (CapState, error) {
	out := CapState{Words: "Needs a ruling: salary cap, reserve salary shares or money rounding increment"}
	if rules.Enforcement.State == leaguerules.IsKnown {
		out.Mode = rules.Enforcement.Value
	}
	if rules.ComplianceDeadline.State == leaguerules.IsKnown {
		date := rules.ComplianceDeadline.Value
		out.Deadline = &date
	}
	if rules.Quantum.State != leaguerules.IsKnown || rules.Quantum.Value <= 0 {
		_, out.Enforcement = CapAllows(out, 0, viewer)
		return out, nil
	}
	quantum := domain.Money(rules.Quantum.Value)
	out.QuantumCents = &quantum
	if err := countCap(&out, rules, players, charges); err != nil {
		return CapState{}, err
	}
	if out.RoomCents != nil {
		room := *out.RoomCents
		switch {
		case room < 0:
			out.Words = fmt.Sprintf("Over the salary cap by %s", -room)
		case room == 0:
			out.Words = "At the salary cap; no cap room"
		default:
			out.Words = fmt.Sprintf("Within the salary cap; %s cap room", room)
		}
	}
	_, out.Enforcement = CapAllows(out, 0, viewer)
	return out, nil
}

func countCap(out *CapState, rules leaguerules.CapRules, players []CapPlayer, charges []domain.Money) error {
	quantum := *out.QuantumCents
	if rules.Amount.State == leaguerules.IsKnown {
		amount, err := capMoney(domain.Money(rules.Amount.Value), 10000, quantum)
		if err != nil {
			return err
		}
		out.CapCents = &amount
	}
	if rules.IRShare.State != leaguerules.IsKnown || rules.TaxiShare.State != leaguerules.IsKnown {
		return nil
	}
	// Totals are unbounded until conversion, so a malformed held salary cannot wrap into cap room.
	totals := make([]big.Int, 5)
	for _, p := range players {
		index, share, err := capSalaryShare(p.Status, rules)
		if err != nil {
			return err
		}
		if p.Salary < 0 {
			return fmt.Errorf("a native contract has a negative salary")
		}
		if err := addCapMoney(&totals[0], p.Salary, 10000, quantum); err != nil {
			return err
		}
		if err := addCapMoney(&totals[index], p.Salary, share, quantum); err != nil {
			return err
		}
	}
	for _, charge := range charges {
		if err := addCapMoney(&totals[4], charge, 10000, quantum); err != nil {
			return err
		}
	}
	return finishCap(out, totals)
}

func capSalaryShare(status domain.RosterStatus, rules leaguerules.CapRules) (int, leaguerules.Decimal, error) {
	switch status {
	case domain.RosterActive:
		return 1, 10000, nil
	case domain.RosterIR:
		return 2, rules.IRShare.Value, nil
	case domain.RosterTaxi:
		return 3, rules.TaxiShare.Value, nil
	}
	return 0, 0, fmt.Errorf("a native contract has an unknown roster squad")
}

func finishCap(out *CapState, totals []big.Int) error {
	fields := []**domain.Money{&out.SalaryCents, &out.ActiveCents, &out.IRCents,
		&out.TaxiCents, &out.AdjustmentCents}
	for i, field := range fields {
		amount, err := checkedCapMoney(&totals[i])
		if err != nil {
			return err
		}
		*field = &amount
	}
	var committed big.Int
	for i := 1; i < len(totals); i++ {
		committed.Add(&committed, &totals[i])
	}
	amount, err := checkedCapMoney(&committed)
	if err != nil {
		return err
	}
	out.CommittedCents = &amount
	if out.CapCents != nil {
		var room big.Int
		room.Sub(big.NewInt(int64(*out.CapCents)), &committed)
		amount, err := checkedCapMoney(&room)
		if err != nil {
			return err
		}
		out.RoomCents = &amount
	}
	return nil
}

// CapAllows takes the net change in counted salary and charges, including any conditional drop.
// Soft enforcement never refuses an over-cap move, even while its deadline needs a ruling.
func CapAllows(state CapState, delta domain.Money, zone *time.Location) (bool, string) {
	if zone == nil {
		zone = time.UTC
	}
	if state.CapCents == nil || state.CommittedCents == nil || state.QuantumCents == nil {
		return false, "Needs a ruling: the salary cap or counted salary cannot be judged"
	}
	change, err := capMoney(delta, 10000, *state.QuantumCents)
	if err != nil {
		return false, "The move's salary change is too large to judge"
	}
	var after big.Int
	after.Add(big.NewInt(int64(*state.CommittedCents)), big.NewInt(int64(change)))
	over := after.Cmp(big.NewInt(int64(*state.CapCents))) > 0
	switch state.Mode {
	case leaguerules.HardCap:
		if over {
			return false, "A move that ends over the salary cap is refused"
		}
		return true, "The move ends within the salary cap"
	case leaguerules.SoftCap:
		if !over {
			return true, "The move ends within the salary cap"
		}
		if state.Deadline == nil {
			return true, "Over the salary cap: flagged; compliance deadline needs a ruling"
		}
		return true, "Over the salary cap: flagged; comply by " + state.Deadline.In(zone).Format("2 Jan 2006 at 3:04 PM MST")
	default:
		return false, "Needs a ruling: whether over-cap moves are refused or flagged"
	}
}

func addCapMoney(total *big.Int, amount domain.Money, share leaguerules.Decimal, quantum domain.Money) error {
	counted, err := capMoney(amount, share, quantum)
	if err != nil {
		return err
	}
	total.Add(total, big.NewInt(int64(counted)))
	return nil
}

// SnapCapMoney applies the league's increment to a signed amount without changing the recorded basis.
func SnapCapMoney(amount, quantum domain.Money) (domain.Money, error) {
	return capMoney(amount, 10000, quantum)
}

func capMoney(amount domain.Money, share leaguerules.Decimal, quantum domain.Money) (domain.Money, error) {
	if quantum <= 0 || share < 0 || share > 10000 {
		return 0, fmt.Errorf("the cap needs a positive money increment and reserve shares from 0 to 100 percent")
	}
	var cents big.Int
	cents.Mul(big.NewInt(int64(amount)), big.NewInt(int64(share)))
	roundCapQuotient(&cents, big.NewInt(10000))
	roundCapQuotient(&cents, big.NewInt(int64(quantum)))
	cents.Mul(&cents, big.NewInt(int64(quantum)))
	return checkedCapMoney(&cents)
}

func roundCapQuotient(value, divisor *big.Int) {
	sign := value.Sign()
	value.Abs(value)
	var remainder big.Int
	value.QuoRem(value, divisor, &remainder)
	remainder.Lsh(&remainder, 1)
	if remainder.Cmp(divisor) >= 0 {
		value.Add(value, big.NewInt(1))
	}
	if sign < 0 {
		value.Neg(value)
	}
}

func checkedCapMoney(value *big.Int) (domain.Money, error) {
	if !value.IsInt64() || value.Int64() == -1<<63 {
		return 0, fmt.Errorf("the native salary cap totals exceed the supported money range")
	}
	return domain.Money(value.Int64()), nil
}
