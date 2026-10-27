package commissioner

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// SeasonRollover moves the league into its next season in one recorded act (census CM-027).
const SeasonRollover = "season.rollover"

// RawRolloverArgs name the season to move to; a dry run only previews the move.
type RawRolloverArgs struct {
	To     int  `json:"to"`
	DryRun bool `json:"dry_run"`
}

func (a RawArgs) validateRollover() error {
	if a.Rollover == nil || a.Rollover.To < 2 || a.Rollover.To > lastSeason {
		return fmt.Errorf("choose the next season")
	}
	rest := a
	rest.Rollover = nil
	if !reflect.DeepEqual(rest, RawArgs{}) {
		return fmt.Errorf("send only the destination season and preview choice")
	}
	return nil
}

// RolloverWaiting stores one waiting rule change whole, so replay never decodes an interface value.
type RolloverWaiting struct {
	Setting    SeedValue `json:"setting"`
	RecordedAt time.Time `json:"recordedAt"`
}

// RolloverBefore is everything the plan was made from: the native assets, the rules in force with their
// calendar, and the waiting rule changes the rollover applies. Its digest is the plan's BeforeDigest.
type RolloverBefore struct {
	NoAssets bool                `json:"noAssets,omitempty"`
	Assets   RawNativeSnapshot   `json:"assets"`
	Rules    leaguerules.RuleSet `json:"rules"`
	At       time.Time           `json:"at"`
	Waiting  []RolloverWaiting   `json:"waiting"`
}

func (b RolloverBefore) state() (RolloverState, error) {
	state := RolloverState{Assets: b.Assets, At: b.At, NoAssets: b.NoAssets}
	for _, waiting := range b.Waiting {
		field := leaguerules.FieldPath(waiting.Setting.Field)
		value, err := leaguerules.DecodeStored(field, waiting.Setting.Value)
		if err != nil {
			return RolloverState{}, fmt.Errorf("read waiting rollover rule: %w", err)
		}
		state.Waiting = append(state.Waiting, leaguerules.Change{LeagueID: b.Rules.LeagueID,
			Season: b.Rules.Season, Field: field, New: value, RecordedAt: waiting.RecordedAt,
			Effective: leaguerules.Effective{Kind: leaguerules.NextRollover}})
	}
	return state, nil
}

func (b RolloverBefore) plan() (RolloverPlan, error) {
	state, err := b.state()
	if err != nil {
		return RolloverPlan{}, err
	}
	return Rollover(state, b.Rules, b.Rules.Season, b.Rules.Season+1)
}

// waitingRollover lists, in ledger order, the in-force rule changes still waiting for the next rollover.
func waitingRollover(records []Record, starts []time.Time, at time.Time) ([]RolloverWaiting, error) {
	active := Replay(records).Active
	waiting := []RolloverWaiting{}
	for _, record := range records {
		if record.Command.Command != RulesSet || record.Command.Effective != "next_rollover" ||
			!active[record.Receipt.Entry] {
			continue
		}
		if start, ok := nextRollover(starts, record.Receipt.RecordedAt); ok && !at.Before(start) {
			continue
		}
		field := leaguerules.FieldPath(record.Command.Args.Field)
		value, err := leaguerules.DecodeSetting(field, record.Command.Args.Value)
		if err != nil {
			return nil, fmt.Errorf("read waiting rule: %w", err)
		}
		stored, err := leaguerules.EncodeStored(value)
		if err != nil {
			return nil, fmt.Errorf("snapshot waiting rule: %w", err)
		}
		waiting = append(waiting, RolloverWaiting{Setting: SeedValue{Field: string(field), Value: stored},
			RecordedAt: record.Receipt.RecordedAt})
	}
	return waiting, nil
}

// PrepareRollover builds the before-state and the plan for a season.rollover command. It runs under the
// ledger transaction, after authorization and the version check, and writes nothing. It is nil for every
// other command. A dry run may return a blocked plan; a real run never does.
func PrepareRollover(raw RawCommand, records []Record, now time.Time, champion ...*BracketChampion) (*RolloverBefore, *RolloverPlan, error) {
	if raw.Command != SeasonRollover {
		return nil, nil, nil
	}
	at := rolloverInstant(records, now)
	from, err := SeasonAt(records, at)
	if err != nil {
		return nil, nil, Refused(err)
	}
	if raw.Args.Rollover.To != from+1 {
		return nil, nil, Refused(fmt.Errorf("move only from season %d to season %d", from, from+1))
	}
	before, err := rolloverBefore(raw.League, records, at, from)
	if err != nil {
		return nil, nil, err
	}
	if len(champion) != 0 {
		before.Assets.Champion = ""
		if champion[0] != nil {
			before.Assets.Champion = champion[0].Franchise
		}
	}
	plan, err := before.plan()
	if err != nil {
		return nil, nil, err
	}
	if plan.Blocked && !raw.Args.Rollover.DryRun {
		return before, &plan, Refused(fmt.Errorf("the rollover is blocked. %s", strings.Join(plan.Reasons, " ")))
	}
	return before, &plan, nil
}

// rolloverInstant is when a rollover is recorded: the clock, but never at or before an earlier entry, so
// every rule change made before it is plainly waiting for it and every one after it is not.
func rolloverInstant(records []Record, now time.Time) time.Time {
	at := ReadingInstant(records, now)
	for _, r := range records {
		if !at.After(r.Receipt.RecordedAt) {
			at = r.Receipt.RecordedAt.Add(time.Microsecond)
		}
	}
	return at
}

func rolloverBefore(league string, records []Record, at time.Time, from int) (*RolloverBefore, error) {
	assets, err := NativeAssetsReading(league, records)
	if err != nil {
		return nil, err
	}
	if assets.Seeded && assets.Snapshot.Season != from {
		return nil, Refused(fmt.Errorf("the recorded assets are for season %d but the league is in season %d",
			assets.Snapshot.Season, from))
	}
	rules, err := RulesInForce(records, from, at)
	if err != nil {
		return nil, err
	}
	starts, err := RolloverInstants(records)
	if err != nil {
		return nil, err
	}
	waiting, err := waitingRollover(records, starts, at)
	if err != nil {
		return nil, err
	}
	before := &RolloverBefore{NoAssets: !assets.Seeded, Rules: rules, At: at, Waiting: waiting}
	if assets.Seeded {
		before.Assets = *assets.Snapshot
	} else {
		before.Assets = RawNativeSnapshot{Season: from}
	}
	return before, nil
}

// validateRolloverRecord checks that a recorded rollover still holds the before-state its plan was made
// from. It does not rebuild the plan: replay applies the recorded plan, so later code cannot change what
// an earlier rollover did.
func (r Record) validateRolloverRecord() error {
	if r.Command.Command != SeasonRollover {
		if r.Before != nil || r.Receipt.Plan != nil {
			return fmt.Errorf("league ledger holds a rollover snapshot on another change")
		}
		return nil
	}
	args, plan := r.Command.Args.Rollover, r.Receipt.Plan
	if r.Before == nil || plan == nil || args == nil || args.DryRun {
		return fmt.Errorf("league ledger rollover has no before-state and plan")
	}
	state, err := r.Before.state()
	if err != nil {
		return err
	}
	digest, err := rolloverDigest(state, r.Before.Rules)
	if err != nil {
		return err
	}
	if digest != plan.BeforeDigest || !r.Before.At.Equal(r.Receipt.RecordedAt) || !r.Before.matches(*plan) ||
		plan.To != args.To || plan.Rules.LeagueID != r.Command.League {
		return fmt.Errorf("league ledger rollover does not match its before-state")
	}
	return nil
}

// matches says whether the plan moves exactly the seasons this before-state is in.
func (b RolloverBefore) matches(plan RolloverPlan) bool {
	if plan.Blocked || plan.From != b.Rules.Season || plan.To != plan.From+1 || plan.Rules.Season != plan.To {
		return false
	}
	return b.NoAssets || (b.Assets.Season == plan.From && plan.After.Season == plan.To)
}

// rolloverKind says whether a recorded change is one a rollover captures in its before-state, or one that
// is made inside the season it opens. Neither can be reversed around the rollover.
func rolloverKind(command string) bool {
	return correctionCommand(command) || slices.Contains([]string{ContractEdit, CapAdjust, CapRelief, ReserveException, RulesSet,
		WindowSet, WindowMove, WindowRemove, SeasonRollover}, command)
}

// CheckRolloverAppend refuses a reversal that would leave a season rollover and the changes around it
// disagreeing about which season they belong to. Everything else about a rollover is checked as it is
// prepared.
func CheckRolloverAppend(raw RawCommand, records []Record) error {
	if raw.Command != Revert {
		return nil
	}
	if reason := rolloverReversalRefusal(records, raw.Args.Entry); reason != "" {
		return Refused(errors.New(reason))
	}
	return nil
}

// rolloverReversalRefusal says why the entry cannot be reversed (or restored by reversing its reversal)
// without breaking a rollover: a rollover cannot be reversed while later changes depend on its season,
// and a change a rollover captured cannot be reversed once the league has moved past it. It is empty when
// the reversal is fine.
func rolloverReversalRefusal(records []Record, entry string) string {
	target, found := reversalRoots(records)[entry]
	if !found {
		return ""
	}
	active := Replay(records).Active
	later := records[target+1:]
	if records[target].Command.Command == SeasonRollover {
		if slices.ContainsFunc(later, func(r Record) bool {
			return active[r.Receipt.Entry] && rolloverKind(r.Command.Command)
		}) {
			return "Reverse the later changes first: rules, contracts, windows or another rollover were " +
				"recorded after this season rollover."
		}
		return ""
	}
	if rolloverKind(records[target].Command.Command) && slices.ContainsFunc(later, func(r Record) bool {
		return active[r.Receipt.Entry] && r.Command.Command == SeasonRollover
	}) {
		return "This change is already part of a season the league has moved past; reverse the season " +
			"rollover first."
	}
	return ""
}
