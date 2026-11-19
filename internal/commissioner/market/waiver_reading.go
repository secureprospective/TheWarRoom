package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type NativeWaiverPlayer struct {
	Window   string    `json:"window"`
	Player   string    `json:"player"`
	Name     string    `json:"name"`
	ClosesAt time.Time `json:"closesAt"`
	Closes   string    `json:"closes"`
	Status   string    `json:"status"`
	Winner   string    `json:"winner,omitempty"`
}

type NativeWaiverClaim struct {
	ID       string `json:"id"`
	Window   string `json:"window"`
	Player   string `json:"player"`
	Status   string `json:"status"`
	Position int    `json:"position,omitempty"`
}

type NativeWaivers struct {
	Corrections []commissioner.Correction `json:"corrections,omitempty"`
	League      string                    `json:"league"`
	Franchise   string                    `json:"franchise"`
	Revision    int64                     `json:"revision"`
	Season      int                       `json:"season"`
	Players     []NativeWaiverPlayer      `json:"players"`
	Claims      []NativeWaiverClaim       `json:"claims"`
	Order       []string                  `json:"order"`
	OrderSource string                    `json:"orderSource"`
	Method      string                    `json:"method"`
}

func NativeWaiversReading(league, franchise, principal string, records []commissioner.Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (NativeWaivers, error) {
	if err := commissioner.AuthorizeFranchiseOwner(records, principal, franchise); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return NativeWaivers{}, err
	}
	f, err := commissioner.NewFrame(league, records, at, facts, zone, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return NativeWaivers{}, err
	}
	return nativeWaiversFrame(f, franchise, principal)
}

func nativeWaiversFrame(f commissioner.Frame, franchise, principal string) (NativeWaivers, error) {
	league, records, at := f.League, f.Records, f.At
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return NativeWaivers{}, err
	}
	rules := f.Rules
	changes, err := commissioner.CorrectionsReading(league, records, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return NativeWaivers{}, err
	}
	out := NativeWaivers{Corrections: changes.Items, League: league, Franchise: franchise, Revision: int64(len(records)), Season: rules.Season,
		Players: []NativeWaiverPlayer{}, Claims: []NativeWaiverClaim{}, Order: []string{},
		Method: commissioner.RuleValueWords(rules, "waivers.method")}
	if !f.Assets.Seeded {
		out.OrderSource = "No starting rosters recorded"
		return out, nil
	}
	x := waiverExecution{assets: f.Assets, state: f.Book, rules: rules, records: records, now: at}
	order, source, err := x.priority(league)
	if err != nil {
		out.OrderSource = err.Error()
	} else {
		out.Order, out.OrderSource = slices.Clone(order), source
	}
	policy, err := nativecheck.NewWaiverPolicy(rules.Waivers)
	if err != nil {
		return NativeWaivers{}, fmt.Errorf("read waiver rules: %w", err)
	}
	positions := make(map[string]int)
	for position, id := range f.Book.PendingClaimIDsFor(franchise, principal, records) {
		positions[id] = position + 1
	}
	if err := appendNativeWaiverRows(&out, f, franchise, principal, positions, policy); err != nil {
		return NativeWaivers{}, err
	}
	return out, nil
}

func appendNativeWaiverRows(out *NativeWaivers, f commissioner.Frame, franchise, principal string,
	positions map[string]int, policy nativecheck.WaiverPolicy) error {
	for _, w := range f.Book.Windows {
		closeAt, err := policy.Close(w.CutAt)
		if err != nil {
			return fmt.Errorf("read waiver close: %w", err)
		}
		if w.Processed {
			closeAt = w.ClosedAt
		}
		row := NativeWaiverPlayer{Window: w.ID, Player: w.Player.ID, Name: f.Facts[w.Player.ID].Name,
			ClosesAt: closeAt.In(f.Zone), Closes: closeAt.In(f.Zone).Format("Jan 2, 2006 at 3:04 PM MST"), Status: "On waivers"}
		if row.Name == "" {
			row.Name = "Player " + w.Player.ID
		}
		if w.Processed {
			row.Winner = w.Winner
			row.Status = "Awarded to a roster"
			if w.Winner == "" {
				row.Status = "Available as a free agent"
			}
		} else if !f.At.Before(closeAt) {
			row.Status = "Closed; awaiting processing"
		}
		out.Players = append(out.Players, row)
		out.Claims = appendOwnWaiverClaims(out.Claims, w, franchise, principal, f.Records, positions)
	}
	slices.SortStableFunc(out.Claims, func(a, b NativeWaiverClaim) int {
		if (a.Position > 0) != (b.Position > 0) {
			if a.Position > 0 {
				return -1
			}
			return 1
		}
		return a.Position - b.Position
	})
	return nil
}

func appendOwnWaiverClaims(out []NativeWaiverClaim, w commissioner.WaiverWindow, franchise, principal string,
	records []commissioner.Record, positions map[string]int) []NativeWaiverClaim {
	for _, c := range w.Claims {
		if c.Franchise != franchise || !commissioner.SamePrincipal(records, c.Principal, principal) {
			continue
		}
		status := "Claim recorded privately"
		if w.Processed {
			status = "Not awarded to your franchise"
			for _, result := range w.Results {
				if result.ID == c.ID && result.Refusal != "" {
					status = "Not awarded: " + result.Refusal
				}
			}
			if w.Winner == franchise {
				status = "Awarded to your franchise"
			}
		}
		out = append(out, NativeWaiverClaim{ID: c.ID, Window: w.ID, Player: w.Player.ID, Status: status, Position: positions[c.ID]})
	}
	return out
}

// DueWaiverWindows is shared by the clock worker and the explicit processing command.
func DueWaiverWindows(league string, records []commissioner.Record, at time.Time) ([]string, error) {
	f, err := commissioner.NewFrame(league, records, at, nil, nil, at)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	return dueWaiverWindowsFrame(f)
}

func dueWaiverWindowsFrame(f commissioner.Frame) ([]string, error) {
	at := f.At
	assets, state := f.Assets, f.Book
	// A league with no starting assets has no waivers; its season may not even be named yet.
	due := []string{}
	if !assets.Seeded || len(state.Windows) == 0 {
		return due, nil
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return nil, err
	}
	rules := f.Rules
	policy, err := nativecheck.NewWaiverPolicy(rules.Waivers)
	if err != nil {
		return nil, fmt.Errorf("read waiver rules: %w", err)
	}
	duePosition := make(map[string]int)
	for _, w := range state.Windows {
		closeAt, err := policy.Close(w.CutAt)
		if err != nil {
			return nil, fmt.Errorf("date due waiver: %w", err)
		}
		if !w.Processed && !at.Before(closeAt) {
			duePosition[w.ID] = len(due)
			due = append(due, w.ID)
		}
	}
	orderDueWaiverWindows(due, duePosition, state)
	return due, nil
}

func orderDueWaiverWindows(due []string, duePosition map[string]int, state commissioner.MarketBook) {
	claimRank := map[string]int{}
	for _, order := range state.ClaimOrders {
		for rank, id := range order {
			claimRank[id] = rank
		}
	}
	rankByWindow := map[string]int{}
	for _, window := range state.Windows {
		if _, isDue := duePosition[window.ID]; !isDue {
			continue
		}
		for _, claim := range window.Claims {
			rank, ordered := claimRank[claim.ID]
			current, ranked := rankByWindow[window.ID]
			if ordered && (!ranked || rank < current) {
				rankByWindow[window.ID] = rank
			}
		}
	}
	slices.SortStableFunc(due, func(a, b string) int {
		aRank, aOrdered := rankByWindow[a]
		bRank, bOrdered := rankByWindow[b]
		if aOrdered != bOrdered {
			if aOrdered {
				return -1
			}
			return 1
		}
		if aOrdered && aRank != bRank {
			return aRank - bRank
		}
		return duePosition[a] - duePosition[b]
	})
}
