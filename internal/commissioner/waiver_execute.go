package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type waiverExecution struct {
	assets  NativeAssets
	state   waiverState
	rules   leaguerules.RuleSet
	records []Record
	now     time.Time
	facts   nativecheck.Facts
	zone    *time.Location
}

// PrepareWaiverAppend computes and validates the decision against the transaction's current records.
func PrepareWaiverAppend(raw *RawCommand, records []Record, now time.Time, facts nativecheck.Facts,
	zone *time.Location) error {
	if !WaiverCommand(raw.Command) {
		return nil
	}
	assets, state, err := nativeAssetsState(raw.League, records)
	if err != nil {
		return err
	}
	if !assets.Seeded {
		return fmt.Errorf("record starting assets before waiver actions")
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return err
	}
	x := waiverExecution{assets: assets, state: state, rules: rules, records: records, now: now, facts: facts, zone: zone}
	if OwnerWaiverAction(raw.Command) != "" {
		gate, err := NativeActionGate(raw.League, records, raw.Args.Waiver.Franchise, OwnerWaiverAction(raw.Command), now, zone)
		if err != nil {
			return err
		}
		if !gate.Open {
			return fmt.Errorf("%s", gate.Sentence)
		}
	}
	return x.prepare(raw)
}

func (x *waiverExecution) prepare(raw *RawCommand) error {
	var err error
	switch raw.Command {
	case RosterCut:
		err = x.cut(raw)
	case WaiverClaim:
		err = x.claim(raw)
	case WaiverOrderSet, WaiverOrderAuto:
		err = x.setOrder(raw)
	case WaiverProcess, WaiverOverride:
		err = x.award(raw)
	}
	if err != nil {
		return err
	}
	trial := append(slices.Clone(x.records), Record{Command: *raw,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: x.now}})
	assets, err := NativeAssetsReading(raw.League, trial)
	if err != nil {
		return err
	}
	if raw.Command == WaiverOverride {
		w, _, err := x.state.claim(raw.Args.Waiver.Claim)
		if err != nil {
			return err
		}
		if w.Winner != "" && w.Winner != raw.Args.WaiverOutcome.Winner {
			return waiverRosterAllows(assets.Snapshot, x.rules, w.Winner, x.facts, x.zone)
		}
	}
	if raw.Command == RosterCut {
		return waiverRosterAllows(assets.Snapshot, x.rules, raw.Args.Waiver.Franchise, x.facts, x.zone)
	}
	return nil
}

func (x *waiverExecution) cut(raw *RawCommand) error {
	policy, err := nativecheck.NewWaiverPolicy(x.rules.Waivers)
	if err != nil {
		return fmt.Errorf("read waiver rules: %w", err)
	}
	if _, err := policy.Close(x.now); err != nil {
		return fmt.Errorf("date waiver close: %w", err)
	}
	args := raw.Args.Waiver
	for _, p := range x.assets.Snapshot.Players {
		if p.ID != args.Player || p.Franchise != args.Franchise {
			continue
		}
		dead, err := waiverDeadMoney(p, x.rules)
		if err != nil {
			return err
		}
		raw.Args.WaiverOutcome = &WaiverOutcome{Player: &p, Dead: dead, Season: x.rules.Season}
		return nil
	}
	return fmt.Errorf("choose a player on your recorded roster")
}

func (x *waiverExecution) claim(raw *RawCommand) error {
	w, err := x.state.window(raw.Args.Waiver.Window)
	if err != nil {
		return err
	}
	policy, err := nativecheck.NewWaiverPolicy(x.rules.Waivers)
	if err != nil {
		return fmt.Errorf("read waiver rules: %w", err)
	}
	closeAt, err := policy.Close(w.CutAt)
	if err != nil {
		return fmt.Errorf("date waiver close: %w", err)
	}
	if w.Processed || !x.now.Before(closeAt) {
		return fmt.Errorf("this waiver window is closed")
	}
	franchise := raw.Args.Waiver.Franchise
	for _, c := range w.Claims {
		if c.Franchise == franchise {
			return fmt.Errorf("you already claimed this player")
		}
	}
	return waiverClaimAllows(x.assets.Snapshot, x.rules, w, franchise, x.facts, x.zone)
}

func (x *waiverExecution) setOrder(raw *RawCommand) error {
	order, source := raw.Args.Waiver.Order, "Set by the commissioner"
	if raw.Command == WaiverOrderAuto {
		var err error
		order, err = RecordedStandingsOrder(raw.League, x.records, x.rules.Season, x.now)
		if err != nil {
			return err
		}
		source = "From recorded standings, lowest standing first"
	}
	if err := completeWaiverOrder(order, x.assets.Snapshot); err != nil {
		return err
	}
	raw.Args.WaiverOutcome = &WaiverOutcome{Season: x.rules.Season, Order: slices.Clone(order), Source: source}
	return nil
}

func (x *waiverExecution) priority(league string) ([]string, string, error) {
	if x.state.OrderSeason == x.rules.Season && len(x.state.Order) != 0 {
		if x.rules.Waivers.Method.Value == leaguerules.ReverseStandings {
			return slices.Clone(x.state.BaseOrder), x.state.Source, nil
		}
		return slices.Clone(x.state.Order), x.state.Source, nil
	}
	if x.rules.Waivers.Method.Value == leaguerules.Rolling {
		return nil, "", fmt.Errorf("set the rolling waiver order before awarding a claim")
	}
	order, err := RecordedStandingsOrder(league, x.records, x.rules.Season, x.now)
	return order, "From recorded standings, lowest standing first", err
}

func (x *waiverExecution) award(raw *RawCommand) error {
	w, err := x.state.window(raw.Args.Waiver.Window)
	if raw.Command == WaiverOverride {
		w, _, err = x.state.claim(raw.Args.Waiver.Claim)
	}
	if err != nil {
		return err
	}
	policy, err := nativecheck.NewWaiverPolicy(x.rules.Waivers)
	if err != nil {
		return fmt.Errorf("read waiver rules: %w", err)
	}
	closeAt, err := policy.Close(w.CutAt)
	if err != nil {
		return fmt.Errorf("date waiver close: %w", err)
	}
	if w.Processed {
		closeAt = w.ClosedAt
	}
	if x.now.Before(closeAt) {
		return fmt.Errorf("wait until this waiver window closes")
	}
	if (raw.Command == WaiverProcess && w.Processed) || (raw.Command == WaiverOverride && !w.Processed) {
		return fmt.Errorf("process an open award only once; use a correction for a processed award")
	}
	if x.rules.Contracts.ClaimClearsDeadCap.State != leaguerules.IsKnown {
		return fmt.Errorf("waiver dead money relief needs a ruling")
	}
	if err := x.chooseAward(raw, w, policy); err != nil {
		return err
	}
	raw.Args.WaiverOutcome.CloseAt = closeAt
	return nil
}

func (x *waiverExecution) chooseAward(raw *RawCommand, w *waiverWindow, policy nativecheck.WaiverPolicy) error {
	// A correction tests the incoming contract without counting the previous winner twice.
	if w.Winner != "" {
		x.assets.Snapshot.Players = slices.DeleteFunc(x.assets.Snapshot.Players,
			func(p NativePlayer) bool { return p.ID == w.Player.ID })
	}
	eligible, claims := x.eligibleClaims(w)
	order, source, err := x.awardPriority(raw.League, w, raw.Command)
	if err != nil && len(eligible) != 0 {
		return err
	}
	if err != nil {
		order, source = slices.Clone(x.state.Order), x.state.Source
	}
	result := nativecheck.WaiverAward{Order: order}
	if raw.Command == WaiverOverride {
		_, c, err := x.state.claim(raw.Args.Waiver.Claim)
		if err != nil {
			return err
		}
		if c.Franchise != raw.Args.Waiver.Franchise || !slices.Contains(eligible, c.Franchise) {
			return fmt.Errorf("the corrected winner must have a legal recorded claim")
		}
		eligible = []string{c.Franchise}
	}
	if len(eligible) != 0 {
		result, err = policy.Award(nativecheck.WaiverPriority{Standings: order, Rolling: order}, eligible)
		if err != nil {
			return fmt.Errorf("choose waiver winner: %w", err)
		}
	}
	raw.Args.WaiverOutcome = &WaiverOutcome{Season: x.rules.Season, Winner: result.Winner,
		Order: result.Order, PriorOrder: slices.Clone(order), Source: source, Claims: claims, ClearsDead: x.rules.Contracts.ClaimClearsDeadCap.Value}
	return nil
}

func (x *waiverExecution) awardPriority(league string, w *waiverWindow, command string) ([]string, string, error) {
	if command == WaiverOverride && x.rules.Waivers.Method.Value == leaguerules.Rolling {
		for _, later := range x.state.Windows {
			if later.Processed && later.ClosedAt.After(w.ClosedAt) {
				return nil, "", fmt.Errorf("reverse the later waiver awards before correcting this rolling award")
			}
		}
		return slices.Clone(w.PriorOrder), x.state.Source, nil
	}
	return x.priority(league)
}

func (x *waiverExecution) eligibleClaims(w *waiverWindow) ([]string, []WaiverClaimResult) {
	eligible := []string{}
	results := []WaiverClaimResult{}
	for _, c := range w.Claims {
		result := WaiverClaimResult{ID: c.ID}
		if err := waiverClaimAllows(x.assets.Snapshot, x.rules, w, c.Franchise, x.facts, x.zone); err != nil {
			result.Refusal = err.Error()
		} else {
			eligible = append(eligible, c.Franchise)
		}
		results = append(results, result)
	}
	return eligible, results
}
