package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type waiverExecution struct {
	assets  commissioner.NativeAssets
	state   commissioner.MarketBook
	rules   leaguerules.RuleSet
	records []commissioner.Record
	now     time.Time
	facts   nativecheck.Facts
	zone    *time.Location
}

// PrepareWaiverAppend computes and validates the decision against the transaction's current records.
func PrepareWaiverAppend(raw *commissioner.RawCommand, records []commissioner.Record, now time.Time, facts nativecheck.Facts,
	zone *time.Location) error {
	if !commissioner.WaiverCommand(raw.Command) {
		return nil
	}
	f, err := commissioner.NewFrame(raw.League, records, now, facts, zone)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	return prepareWaiverFrame(f, raw)
}

func prepareWaiverFrame(f commissioner.Frame, raw *commissioner.RawCommand) error {
	records, now, facts, zone := f.Records, f.At, f.Facts, f.Zone
	if err := f.Seeded("record starting assets before waiver actions"); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	x := waiverExecution{assets: f.Assets, state: f.Book, rules: f.Rules, records: records, now: now, facts: facts, zone: zone}
	if commissioner.OwnerWaiverAction(raw.Command) != "" {
		if err := f.Gate(raw.Args.Waiver.Franchise, x.reserveAction(*raw)); err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
	}
	return x.prepare(raw)
}

func (x *waiverExecution) prepare(raw *commissioner.RawCommand) error {
	var err error
	switch raw.Command {
	case commissioner.RosterReserve:
		err = x.reserve(raw)
	case commissioner.RosterCut:
		err = x.cut(raw)
	case commissioner.WaiverClaim:
		err = x.claim(raw)
	case commissioner.WaiverOrderSet, commissioner.WaiverOrderAuto:
		err = x.setOrder(raw)
	case commissioner.WaiverProcess, commissioner.WaiverOverride:
		err = x.award(raw)
	}
	if err != nil {
		return err
	}
	f := commissioner.Frame{League: raw.League, Records: x.records, At: x.now}
	assets, err := f.Trial(*raw, "")
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	if raw.Command == commissioner.WaiverOverride {
		w, _, err := x.state.Claim(raw.Args.Waiver.Claim)
		if err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
		if w.Winner != "" && w.Winner != raw.Args.WaiverOutcome.Winner {
			return rosterCapAllows(assets.Snapshot, x.rules, w.Winner, x.facts, x.zone, commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: w.Winner}, x.now)...)
		}
	}
	if raw.Command == commissioner.RosterCut || raw.Command == commissioner.RosterReserve {
		rules := commissioner.AllowDebtReduction(x.assets.Snapshot, assets.Snapshot, x.records, x.rules, raw.Args.Waiver.Franchise)
		if raw.Command == commissioner.RosterCut {
			rules = commissioner.PendingOLineCutRules(x.state, x.assets.Snapshot, assets.Snapshot, rules, raw.Args.Waiver.Franchise)
		}
		return rosterCapAllows(assets.Snapshot, rules, raw.Args.Waiver.Franchise, x.facts, x.zone, commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: raw.Args.Waiver.Franchise}, x.now)...)
	}
	return nil
}

func (x *waiverExecution) cut(raw *commissioner.RawCommand) error {
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
		raw.Args.WaiverOutcome = &commissioner.WaiverOutcome{Player: &p, Dead: dead, Season: x.rules.Season}
		return nil
	}
	return fmt.Errorf("choose a player on your recorded roster")
}

func (x *waiverExecution) claim(raw *commissioner.RawCommand) error {
	w, err := x.state.Window(raw.Args.Waiver.Window)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
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
	if x.assets.Snapshot.SideOf(franchise) != w.Side {
		return fmt.Errorf("players stay on their own side of the league: claim only players cut on your side")
	}
	for _, c := range w.Claims {
		if c.Franchise == franchise {
			return fmt.Errorf("you already claimed this player")
		}
	}
	return waiverClaimAllows(x.assets.Snapshot, x.rules, w, franchise, x.facts, x.zone, commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: franchise, Player: w.Player.ID, Transaction: w.ID}, x.now)...)
}

func (x *waiverExecution) setOrder(raw *commissioner.RawCommand) error {
	order, source := raw.Args.Waiver.Order, "Set by the commissioner"
	if raw.Command == commissioner.WaiverOrderAuto {
		var err error
		order, err = RecordedStandingsOrder(raw.League, x.records, x.rules.Season, x.now)
		if err != nil {
			return err
		}
		source = "From recorded standings, lowest standing first"
	}
	if err := commissioner.CompleteWaiverOrder(order, x.assets.Snapshot); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	raw.Args.WaiverOutcome = &commissioner.WaiverOutcome{Season: x.rules.Season, Order: slices.Clone(order), Source: source}
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

func (x *waiverExecution) award(raw *commissioner.RawCommand) error {
	w, err := x.state.Window(raw.Args.Waiver.Window)
	if raw.Command == commissioner.WaiverOverride {
		w, _, err = x.state.Claim(raw.Args.Waiver.Claim)
	}
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
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
	if (raw.Command == commissioner.WaiverProcess && w.Processed) || (raw.Command == commissioner.WaiverOverride && !w.Processed) {
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

func (x *waiverExecution) chooseAward(raw *commissioner.RawCommand, w *commissioner.WaiverWindow, policy nativecheck.WaiverPolicy) error {
	// A correction tests the incoming contract without counting the previous winner twice.
	if w.Winner != "" {
		seed := x.assets.Snapshot
		seed.Players = slices.DeleteFunc(seed.Players,
			func(p commissioner.NativePlayer) bool {
				return p.ID == w.Player.ID && seed.SideOf(p.Franchise) == w.Side
			})
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
	if raw.Command == commissioner.WaiverOverride {
		_, c, err := x.state.Claim(raw.Args.Waiver.Claim)
		if err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
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
	raw.Args.WaiverOutcome = &commissioner.WaiverOutcome{Season: x.rules.Season, Winner: result.Winner,
		Order: result.Order, PriorOrder: slices.Clone(order), Source: source, Claims: claims, ClearsDead: x.rules.Contracts.ClaimClearsDeadCap.Value}
	return nil
}

func (x *waiverExecution) awardPriority(league string, w *commissioner.WaiverWindow, command string) ([]string, string, error) {
	if command == commissioner.WaiverOverride && x.rules.Waivers.Method.Value == leaguerules.Rolling {
		for _, later := range x.state.Windows {
			if later.Processed && later.ClosedAt.After(w.ClosedAt) {
				return nil, "", fmt.Errorf("reverse the later waiver awards before correcting this rolling award")
			}
		}
		return slices.Clone(w.PriorOrder), x.state.Source, nil
	}
	return x.priority(league)
}

func (x *waiverExecution) eligibleClaims(w *commissioner.WaiverWindow) ([]string, []commissioner.WaiverClaimResult) {
	eligible := []string{}
	results := []commissioner.WaiverClaimResult{}
	for _, c := range w.Claims {
		result := commissioner.WaiverClaimResult{ID: c.ID}
		if err := waiverClaimAllows(x.assets.Snapshot, x.rules, w, c.Franchise, x.facts, x.zone, commissioner.ScopedDecisionExceptions(x.records, commissioner.ExceptionScope{Franchise: c.Franchise, Player: w.Player.ID, Transaction: c.ID}, x.now)...); err != nil {
			result.Refusal = err.Error()
		} else {
			eligible = append(eligible, c.Franchise)
		}
		results = append(results, result)
	}
	return eligible, results
}
