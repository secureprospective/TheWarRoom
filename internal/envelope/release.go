package envelope

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

const CutIntent = "roster.drop"
const BuyoutIntent = "roster.buyout"

// ReleaseFacts holds the priced move and the mirror revisions it started from.
type ReleaseFacts struct {
	Player        playerid.PlayerID `json:"player" ts_type:"string"`
	Name          string            `json:"name"`
	Season        int               `json:"season"`
	Charge        domain.Money      `json:"charge"`
	Salary        domain.Money      `json:"salary"`
	Expiration    int               `json:"expiration"`
	ContractInfo  string            `json:"contractInfo"`
	RostersStamp  string            `json:"rostersStamp"`
	ContractStamp string            `json:"contractStamp"`
	DeadCapStamp  string            `json:"deadCapStamp"`
	Adjustments   string            `json:"adjustments"`
}

type ReleaseCheck struct {
	OnRoster     bool
	Offseason    bool
	OtherBuyouts int
	Remaining    int
	ChargeText   string
}

func IsRelease(intent string) bool { return intent == CutIntent || intent == BuyoutIntent }
func ReleaseOpen(state State) bool {
	return state != Landed && state != Failed && state != Withdrawn && state != AwardedOffApp
}
func (c ReleaseCheck) Evaluate(s Spec, _ snapshot.Snapshot) CheckResult {
	if !IsRelease(s.Intent) {
		return CheckResult{Blocked: true, Note: "Release check: unsupported intent"}
	}
	reasons := []string{}
	if !c.OnRoster {
		reasons = append(reasons, "Roster: player is not on my roster in the MFL mirror")
	}
	if s.Intent == BuyoutIntent {
		if !c.Offseason {
			reasons = append(reasons, "Offseason only: can't be done until the offseason.")
		}
		if c.OtherBuyouts >= 2 {
			reasons = append(reasons, "Two buyouts per season: landed and open plans use both slots")
		}
		if c.Remaining < 2 || c.Remaining > 4 {
			reasons = append(reasons, "Buyout term: requires 2–4 years left; a 1-year-left buyout is refused")
		}
	}
	if len(reasons) > 0 {
		return CheckResult{Blocked: true, Note: strings.Join(reasons, "; ") + "; planned dead cap this season: " + c.ChargeText}
	}
	return CheckResult{Note: "Roster checked; planned dead cap this season: " + c.ChargeText}
}
func DraftRelease(at time.Time, spec Spec, check ReleaseCheck, generate func() string) (Envelope, error) {
	if !IsRelease(spec.Intent) {
		return Envelope{}, fmt.Errorf("draft release: unsupported intent")
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft release: %w", err)
	}
	next, err := e.Check(at, snapshot.Snapshot{}, check)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft release check: %w", err)
	}
	return next, nil
}
func (e Envelope) HandOffRelease(at time.Time, confirmed bool) (Envelope, error) {
	if !IsRelease(e.spec.Intent) {
		return Envelope{}, fmt.Errorf("release hand-off: unsupported intent")
	}
	if e.spec.Intent == CutIntent && !confirmed {
		return Envelope{}, fmt.Errorf("release hand-off: confirmation required")
	}
	if e.spec.Intent == BuyoutIntent {
		return e.HandOffManual(at, "Buyout request copied for the commissioner on ProBoards; nothing posted; acceptance not verified")
	}
	return e.handOff(at, "Confirmed irreversible release; MFL drop page opened; owner submits on MFL")
}

type ReleasePredicate struct{}

func (ReleasePredicate) Sources() []Source { return []Source{Rosters} }
func (ReleasePredicate) Evaluate(s Spec, obs Observation) Verdict {
	if !IsRelease(s.Intent) || obs.LeagueID != s.LeagueID || obs.Rosters.Provenance.Freshness.State == domain.FreshFail {
		return Verdict{Event: Partial, Note: "Not verified: authoritative scoped roster unavailable"}
	}
	for _, r := range obs.Rosters.Value {
		if r.FranchiseID != s.FranchiseID {
			continue
		}
		_, found := rosterStatus(s, obs.Rosters.Value)
		if !found {
			return Verdict{Event: Match, Note: "Landed: player gone from my roster in the rosters export. Dead-cap charge not yet verified."}
		}
		return Verdict{Event: NoChange, Note: "No matching MFL release observed"}
	}
	return Verdict{Event: Partial, Note: "Not verified: franchise roster missing"}
}

// ReleaseInvalidation names each independent reason; roster disappearance after hand-off is
// left to the authoritative landing predicate, not mistaken for failure.
func ReleaseInvalidation(intent string, handed, onRoster, offseason, contractChanged, adjustmentChanged bool) string {
	switch {
	case !onRoster && !handed:
		return "Roster: player left my roster by another route"
	case contractChanged && onRoster:
		return "Contract: newer change for this player"
	case adjustmentChanged && onRoster:
		return "Dead cap / adjustments: new row or change for my franchise"
	case intent == BuyoutIntent && !offseason && onRoster:
		return "Offseason only: buyout window closed"
	default:
		return ""
	}
}

func (e Envelope) WithdrawRelease(at time.Time, note string) (Envelope, error) {
	if !IsRelease(e.spec.Intent) {
		return Envelope{}, ErrIllegalTransition{From: e.state, Event: Withdraw}
	}
	return e.move(at, Withdraw, note)
}
