package nativecheck

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// WaiverPolicy consumes resolved rules; its zero value cannot date or award a waiver.
type WaiverPolicy struct {
	duration time.Duration
	method   leaguerules.WaiverMethod
}

func NewWaiverPolicy(rules leaguerules.WaiversRules) (WaiverPolicy, error) {
	if rules.Trigger.State != leaguerules.IsKnown || rules.Trigger.Value != leaguerules.OnCut {
		return WaiverPolicy{}, fmt.Errorf("waiver trigger needs a ruling")
	}
	if rules.Duration.State != leaguerules.IsKnown || rules.Duration.Value <= 0 {
		return WaiverPolicy{}, fmt.Errorf("waiver duration needs a ruling")
	}
	if rules.Method.State != leaguerules.IsKnown {
		return WaiverPolicy{}, fmt.Errorf("waiver priority needs a ruling")
	}
	switch rules.Method.Value {
	case leaguerules.ReverseStandings, leaguerules.Rolling:
		return WaiverPolicy{duration: rules.Duration.Value, method: rules.Method.Value}, nil
	default:
		return WaiverPolicy{}, fmt.Errorf("waiver priority needs a ruling")
	}
}

// Close dates an unprocessed window from its original cut, not from the latest rule change.
func (p WaiverPolicy) Close(cutAt time.Time) (time.Time, error) {
	if p.duration <= 0 || cutAt.IsZero() {
		return time.Time{}, fmt.Errorf("a recorded cut and waiver duration are needed to date this window")
	}
	closeAt := cutAt.Add(p.duration)
	if closeAt.Year() < 1 || closeAt.Year() > 9999 {
		return time.Time{}, fmt.Errorf("the waiver window closes outside the supported calendar")
	}
	return closeAt, nil
}

// WaiverPriority contains priority-first orders. Standings is already reversed by its source;
// it is never inferred from roster size, franchise ids or the order claims arrived.
type WaiverPriority struct {
	Standings []string
	Rolling   []string
}

// WaiverAward is a priority decision only. Its caller must check claim eligibility and commit the
// winning roster change atomically; no roster or salary legality is implied by this result.
type WaiverAward struct {
	Winner string   `json:"winner"`
	Order  []string `json:"order"`
}

func (p WaiverPolicy) Award(priority WaiverPriority, eligible []string) (WaiverAward, error) {
	order, err := p.priorityOrder(priority)
	if err != nil {
		return WaiverAward{}, err
	}
	if err := validateWaiverOrder(order); err != nil {
		return WaiverAward{}, err
	}
	claims := make(map[string]bool, len(eligible))
	for _, franchise := range eligible {
		if claims[franchise] || !slices.Contains(order, franchise) {
			return WaiverAward{}, fmt.Errorf("each eligible claim must name one franchise in the waiver order")
		}
		claims[franchise] = true
	}
	out := WaiverAward{Order: slices.Clone(order)}
	for i, franchise := range order {
		if !claims[franchise] {
			continue
		}
		out.Winner = franchise
		if p.method == leaguerules.Rolling {
			out.Order = append(slices.Delete(out.Order, i, i+1), franchise)
		}
		break
	}
	return out, nil
}

func (p WaiverPolicy) priorityOrder(priority WaiverPriority) ([]string, error) {
	switch p.method {
	case leaguerules.ReverseStandings:
		if len(priority.Standings) == 0 {
			return nil, fmt.Errorf("waiver order needs the standings, which are not built yet")
		}
		return priority.Standings, nil
	case leaguerules.Rolling:
		if len(priority.Rolling) == 0 {
			return nil, fmt.Errorf("set the rolling waiver order before awarding a claim")
		}
		return priority.Rolling, nil
	default:
		return nil, fmt.Errorf("waiver priority needs a ruling")
	}
}

func validateWaiverOrder(order []string) error {
	seen := make(map[string]bool, len(order))
	for _, franchise := range order {
		if strings.TrimSpace(franchise) == "" || strings.TrimSpace(franchise) != franchise || seen[franchise] {
			return fmt.Errorf("waiver order must list each franchise once, without blanks")
		}
		seen[franchise] = true
	}
	return nil
}
