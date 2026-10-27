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

// NewFrame folds the ledger and reads its current rules and governance.
func NewFrame(league string, records []Record, at time.Time, facts nativecheck.Facts, zone *time.Location) (Frame, error) {
	f, err := marketFrame(league, records, at, facts, zone, false)
	if err != nil {
		return f, err
	}
	if err = f.readRules(); err != nil {
		return f, err
	}
	if err = f.readGovernance(); err != nil {
		return f, err
	}
	return f, nil
}

func marketFrame(league string, records []Record, at time.Time, facts nativecheck.Facts, zone *time.Location, correctionsAt bool) (Frame, error) {
	f := Frame{League: league, Records: records, At: at, Facts: facts, Zone: zone}
	var err error
	if correctionsAt {
		f.Assets, f.Book, err = nativeAssetsState(league, records, at)
	} else {
		f.Assets, f.Book, err = nativeAssetsState(league, records)
	}
	return f, err
}

func (f *Frame) readRules() error {
	var err error
	f.Rules, err = RulesNow(f.Records, f.At)
	return err
}
func (f *Frame) readGovernance() error {
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
