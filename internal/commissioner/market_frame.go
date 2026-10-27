package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Frame holds the market's view of a transaction's records.
type Frame struct {
	League  string
	Records []Record
	At      time.Time
	Assets  NativeAssets
	Book    waiverState
	Rules   leaguerules.RuleSet
	Govern  governance.State
	Facts   nativecheck.Facts
	Zone    *time.Location
}

// NewFrame folds only the ledger. Callers authorize before construction and load
// rules and governance at their operation's original stage. correctionAt, when
// present, dates correction readings rather than using the last record's time.
func NewFrame(league string, records []Record, at time.Time, facts nativecheck.Facts, zone *time.Location, correctionAt ...time.Time) (Frame, error) {
	f := Frame{League: league, Records: records, At: at, Facts: facts, Zone: zone}
	var err error
	f.Assets, f.Book, err = nativeAssetsState(league, records, correctionAt...)
	return f, err
}

// ReadRules loads rules only when the operation reaches its rules check.
func (f *Frame) ReadRules() error {
	var err error
	f.Rules, err = RulesNow(f.Records, f.At)
	return err
}

// ReadGovernance loads governance independently, so trade checks can precede rules.
func (f *Frame) ReadGovernance() error {
	var err error
	f.Govern, err = GovernanceState(f.Records)
	return err
}

// Seeded refuses the caller's action when starting assets are absent.
func (f Frame) Seeded(what string) error {
	if !f.Assets.Seeded {
		return fmt.Errorf("%s", what)
	}
	return nil
}

// Gate applies the native action window.
func (f Frame) Gate(franchise string, action leaguerules.Action) error {
	gate, err := NativeActionGate(f.League, f.Records, franchise, action, f.At, f.Zone)
	if err != nil {
		return err
	}
	if !gate.Open {
		return fmt.Errorf("%s", gate.Sentence)
	}
	return nil
}

// Trial projects a proposed command without changing the held records.
func (f Frame) Trial(raw RawCommand, principal string) (NativeAssets, error) {
	trial := append(slices.Clone(f.Records), Record{Command: raw, Principal: principal,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: f.At}})
	return NativeAssetsReading(f.League, trial)
}

// Exceptions returns the applicable recorded decisions.
func (f Frame) Exceptions(scope ExceptionScope) []nativecheck.DecisionException {
	return scopedDecisionExceptions(f.Records, scope, f.At)
}
