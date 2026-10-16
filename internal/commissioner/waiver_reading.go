package commissioner

import (
	"fmt"
	"slices"
	"time"

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
	ID     string `json:"id"`
	Window string `json:"window"`
	Player string `json:"player"`
	Status string `json:"status"`
}

type NativeWaivers struct {
	League      string               `json:"league"`
	Franchise   string               `json:"franchise"`
	Revision    int64                `json:"revision"`
	Season      int                  `json:"season"`
	Players     []NativeWaiverPlayer `json:"players"`
	Claims      []NativeWaiverClaim  `json:"claims"`
	Order       []string             `json:"order"`
	OrderSource string               `json:"orderSource"`
	Method      string               `json:"method"`
}

func NativeWaiversReading(league, franchise, principal string, records []Record, at time.Time,
	zone *time.Location, facts nativecheck.Facts) (NativeWaivers, error) {
	if err := AuthorizeFranchiseOwner(records, principal, franchise); err != nil {
		return NativeWaivers{}, err
	}
	assets, state, err := nativeAssetsState(league, records)
	if err != nil {
		return NativeWaivers{}, err
	}
	rules, err := RulesNow(records, at)
	if err != nil {
		return NativeWaivers{}, err
	}
	out := NativeWaivers{League: league, Franchise: franchise, Revision: int64(len(records)), Season: rules.Season,
		Players: []NativeWaiverPlayer{}, Claims: []NativeWaiverClaim{}, Order: []string{},
		Method: fieldWords(rules, "waivers.method")}
	if !assets.Seeded {
		out.OrderSource = "No starting rosters recorded"
		return out, nil
	}
	x := waiverExecution{assets: assets, state: state, rules: rules, records: records, now: at}
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
	for _, w := range state.Windows {
		closeAt, err := policy.Close(w.CutAt)
		if err != nil {
			return NativeWaivers{}, fmt.Errorf("read waiver close: %w", err)
		}
		if w.Processed {
			closeAt = w.ClosedAt
		}
		row := NativeWaiverPlayer{Window: w.ID, Player: w.Player.ID, Name: facts[w.Player.ID].Name,
			ClosesAt: closeAt.In(zone), Closes: closeAt.In(zone).Format("Jan 2, 2006 at 3:04 PM MST"), Status: "On waivers"}
		if row.Name == "" {
			row.Name = "Player " + w.Player.ID
		}
		if w.Processed {
			row.Winner = w.Winner
			row.Status = "Awarded to a roster"
			if w.Winner == "" {
				row.Status = "Available as a free agent"
			}
		} else if !at.Before(closeAt) {
			row.Status = "Closed; awaiting processing"
		}
		out.Players = append(out.Players, row)
		out.Claims = appendOwnWaiverClaims(out.Claims, w, franchise, principal)
	}
	return out, nil
}

func appendOwnWaiverClaims(out []NativeWaiverClaim, w waiverWindow, franchise, principal string) []NativeWaiverClaim {
	for _, c := range w.Claims {
		if c.Franchise != franchise || c.Principal != principal {
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
		out = append(out, NativeWaiverClaim{ID: c.ID, Window: w.ID, Player: w.Player.ID, Status: status})
	}
	return out
}

// DueWaiverWindows is shared by the clock worker and the explicit processing command.
func DueWaiverWindows(league string, records []Record, at time.Time) ([]string, error) {
	_, state, err := nativeAssetsState(league, records)
	if err != nil {
		return nil, err
	}
	rules, err := RulesNow(records, at)
	if err != nil {
		return nil, err
	}
	due := []string{}
	if len(state.Windows) == 0 {
		return due, nil
	}
	policy, err := nativecheck.NewWaiverPolicy(rules.Waivers)
	if err != nil {
		return nil, fmt.Errorf("read waiver rules: %w", err)
	}
	for _, w := range state.Windows {
		closeAt, err := policy.Close(w.CutAt)
		if err != nil {
			return nil, fmt.Errorf("date due waiver: %w", err)
		}
		if !w.Processed && !at.Before(closeAt) {
			due = append(due, w.ID)
		}
	}
	return due, nil
}
