package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Correction is a public explanation of an active compensating change, never a replacement ledger body.
type Correction struct {
	Entry     string          `json:"entry"`
	Who       string          `json:"who"`
	When      time.Time       `json:"when"`
	Why       string          `json:"why"`
	What      string          `json:"what"`
	Player    string          `json:"player,omitempty"`
	Franchise string          `json:"franchise,omitempty"`
	Pick      *CorrectionPick `json:"pick,omitempty"`
	Rule      string          `json:"rule,omitempty"`
	Scope     *ExceptionScope `json:"scope,omitempty"`
	Until     string          `json:"until,omitempty"`
}

type Corrections struct {
	League   string       `json:"league"`
	Revision int64        `json:"revision"`
	Items    []Correction `json:"items"`
}

func CorrectionsReading(league string, records []Record, now time.Time) (Corrections, error) {
	if len(records) == 0 || records[0].Command.League != league {
		return Corrections{}, fmt.Errorf("choose an existing app league")
	}
	out := Corrections{League: league, Revision: int64(len(records)), Items: []Correction{}}
	active := Replay(records).Active
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if !correctionCommand(r.Command.Command) || !active[r.Receipt.Entry] || now.Before(r.Receipt.RecordedAt) {
			continue
		}
		c, err := decodeCorrection(r.Command.Args, r.Command.Command)
		if err != nil {
			return Corrections{}, err
		}
		if c.Until != "" {
			end, err := time.Parse(time.RFC3339Nano, c.Until)
			if err != nil {
				return Corrections{}, fmt.Errorf("read correction expiry: %w", err)
			}
			if !now.Before(end) {
				continue
			}
		}
		out.Items = append(out.Items, Correction{Entry: r.Receipt.Entry, Who: r.Principal, When: r.Receipt.RecordedAt,
			Why: r.Command.Reason, What: correctionDescription(records, r), Player: c.Player, Franchise: c.Franchise,
			Pick: c.Pick, Rule: c.Rule, Scope: c.Scope, Until: c.Until})
	}
	return out, nil
}

// decisionException is shared by the calendar's actual enforcement point and the public reading.
func decisionException(records []Record, rule string, scope ExceptionScope, now time.Time) (string, bool) {
	active := Replay(records).Active
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Command.Command != RuleException || !active[r.Receipt.Entry] || now.Before(r.Receipt.RecordedAt) {
			continue
		}
		c := r.Command.Args.Correction
		if c == nil || c.Rule != rule || c.Scope == nil || !exceptionScopeMatches(*c.Scope, scope) {
			continue
		}
		if c.Until != "" {
			until, err := time.Parse(time.RFC3339Nano, c.Until)
			if err != nil || !now.Before(until) {
				continue
			}
		}
		return "Commissioner exception: " + r.Command.Reason, true
	}
	return "", false
}

func supportedExceptionRule(rule string) bool {
	return rule == "roster.limits" || rule == "roster.positionBounds" || rule == "cap.enforcement" ||
		rule == "ir.eligible" || rule == "taxi.maximumYears" ||
		(strings.HasPrefix(rule, "calendar.") && len(rule) > len("calendar."))
}

func ScopedDecisionExceptions(records []Record, scope ExceptionScope, now time.Time) []nativecheck.DecisionException {
	out := []nativecheck.DecisionException{}
	for _, rule := range []string{"roster.limits", "roster.positionBounds", "cap.enforcement", "ir.eligible", "taxi.maximumYears"} {
		if why, allowed := decisionException(records, rule, scope, now); allowed {
			out = append(out, nativecheck.DecisionException{Rule: rule, Reason: why})
		}
	}
	return out
}

func exceptionScopeMatches(want, decision ExceptionScope) bool {
	switch {
	case want.Franchise != "" && want.Player != "":
		return want.Franchise == decision.Franchise && want.Player == decision.Player
	case want.Franchise != "":
		return want.Franchise == decision.Franchise
	case want.Player != "":
		return want.Player == decision.Player
	case want.Transaction != "":
		return want.Transaction == decision.Transaction
	}
	return false
}
