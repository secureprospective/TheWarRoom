package market

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type tradeExecution struct {
	records   []commissioner.Record
	seed      *commissioner.RawNativeSnapshot
	state     governance.State
	rules     leaguerules.RuleSet
	now       time.Time
	facts     nativecheck.Facts
	zone      *time.Location
	principal string
}

func PrepareTradeAppend(raw *commissioner.RawCommand, principal string, records []commissioner.Record, now time.Time,
	facts nativecheck.Facts, zone *time.Location) error {
	if !commissioner.TradeCommand(raw.Command) {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return commissioner.CheckTradeReplay(*raw, records, now)
	}
	f, err := commissioner.NewFrame(raw.League, records, now, facts, zone)
	if err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	return prepareTradeFrame(f, raw, principal)
}

func prepareTradeFrame(f commissioner.Frame, raw *commissioner.RawCommand, principal string) error {
	records, now, facts, zone := f.Records, f.At, f.Facts, f.Zone
	var err error
	if err := f.Seeded("record starting rosters and picks before trading"); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	if err := f.ReadGovernance(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	if err := f.ReadRules(); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	x := tradeExecution{records: records, seed: f.Assets.Snapshot, state: f.Govern, rules: f.Rules, now: now, facts: facts, zone: zone, principal: principal}
	if action := commissioner.OwnerTradeAction(raw.Command); action != "" {
		if err := f.Gate(raw.Args.Trade.Franchise, action); err != nil {
			//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
			return err
		}
	}
	var t commissioner.TradeState
	if raw.Command == commissioner.TradeOffer {
		t, err = x.offer(*raw)
	} else {
		t, err = commissioner.TradeByID(f.Book.Trades, raw.Args.Trade.ID)
		if err == nil {
			prior := commissioner.TradeDigest(t)
			err = x.act(*raw, &t)
			t.Previous = prior
		}
	}
	if err != nil {
		return err
	}
	raw.Args.TradeOutcome = &t
	if _, err := f.Trial(*raw, principal); err != nil {
		//nolint:wrapcheck // Preserve the original refusal text across the package boundary.
		return err
	}
	return nil
}

func (x tradeExecution) offer(raw commissioner.RawCommand) (commissioner.TradeState, error) {
	args := raw.Args.Trade
	policy, err := resolveTradePolicy(x.rules.Trades)
	if err != nil {
		return commissioner.TradeState{}, err
	}
	t := commissioner.TradeState{Details: x.describeAssets(args.Assets), Proposer: args.Franchise, Parties: slices.Clone(args.Parties), Assets: slices.Clone(args.Assets),
		Accepted: []string{args.Franchise}, Policy: policy, OfferedAt: x.now, ExpiresAt: x.now.Add(policy.Expiry),
		Status: "offered", Words: "Waiting for every party to accept", Votes: []commissioner.TradeVote{}, Recusals: []commissioner.TradeVote{}, Appeals: []commissioner.TradeVote{}, Overrides: []commissioner.TradeVote{}}
	if len(t.Parties) > policy.MaximumParties {
		return t, fmt.Errorf("this league permits at most %d trade parties", policy.MaximumParties)
	}
	for _, id := range t.Parties {
		if _, ok := x.state.Franchise(governance.FranchiseID(id)); !ok {
			return t, fmt.Errorf("choose franchises in this league")
		}
		if !slices.ContainsFunc(t.Assets, func(a commissioner.TradeAsset) bool { return a.From == id || a.To == id }) {
			return t, fmt.Errorf("each party must send or receive an asset")
		}
	}
	if err := x.legal(t); err != nil {
		return t, err
	}
	return t, nil
}

func (x tradeExecution) act(raw commissioner.RawCommand, t *commissioner.TradeState) error {
	a := raw.Args.Trade
	switch raw.Command {
	case commissioner.TradeAccept, commissioner.TradeDecline, commissioner.TradeWithdraw:
		return x.ownerAct(raw, t)
	case commissioner.TradeReview, commissioner.TradeRecuse:
		return x.review(raw, t)
	case commissioner.TradeAppeal:
		if t.Status != "vetoed" || t.AppealedVeto == t.VetoNumber {
			return fmt.Errorf("a party may appeal this veto only once")
		}
		if !slices.Contains(t.Parties, a.Franchise) {
			return fmt.Errorf("only a trade party may appeal")
		}
		t.AppealedVeto = t.VetoNumber
		t.Appeals = append(t.Appeals, commissioner.TradeVote{Who: x.principal, Decision: "appeal", At: x.now, Reason: raw.Reason})
		t.Status, t.Words = "appealed", "Waiting for the eligible commissioner to rule on the appeal"
	case commissioner.TradeAppealRule:
		return x.ruleAppeal(raw, t)
	case commissioner.TradeOverride:
		return x.override(raw, t)
	}
	return nil
}

func (x tradeExecution) ownerAct(raw commissioner.RawCommand, t *commissioner.TradeState) error {
	a := raw.Args.Trade
	if t.Status != "offered" {
		return fmt.Errorf("this offer is no longer waiting for acceptance")
	}
	if !x.now.Before(t.ExpiresAt) {
		return fmt.Errorf("this offer has expired")
	}
	if !slices.Contains(t.Parties, a.Franchise) {
		return fmt.Errorf("only a trade party may answer this offer")
	}
	switch raw.Command {
	case commissioner.TradeWithdraw:
		if t.Proposer != a.Franchise {
			return fmt.Errorf("only the offering team may withdraw")
		}
		t.Status, t.Words = "withdrawn", "The offering team withdrew this offer"
	case commissioner.TradeDecline:
		t.Status, t.Words = "declined", "A party declined this offer"
	case commissioner.TradeAccept:
		if slices.Contains(t.Accepted, a.Franchise) {
			return fmt.Errorf("this team already accepted")
		}
		if err := x.legal(*t); err != nil {
			return err
		}
		t.Accepted = append(t.Accepted, a.Franchise)
		if len(t.Accepted) == len(t.Parties) {
			return x.beginReview(t)
		}
	}
	return nil
}

func (x tradeExecution) beginReview(t *commissioner.TradeState) error {
	t.Status, t.Words = "review", "Waiting for trade review"
	switch t.Policy.Route {
	case leaguerules.NoReview:
		return x.decide(t, "approve")
	case leaguerules.DOTReview:
		eligible, err := tradeReviewers(t, x.state, x.records, x.now)
		if err != nil && t.Status != "failed" {
			t.Words = err.Error()
		}
		if err == nil && t.Status == "review" && len(eligible) < t.Policy.DecideAt {
			t.Words = fmt.Sprintf("Only %d DOT members are not trade parties, fewer than the %d votes needed; "+
				"the commissioner decides this trade", len(eligible), t.Policy.DecideAt)
		}
	case leaguerules.CommissionerReview:
		if _, err := governance.AppealJudge(x.state, tradeParties(x.state, t.Parties)); err != nil {
			t.Status, t.Words = "failed", "No eligible commissioner remains; this trade cannot proceed"
		}
	}
	return nil
}

func (x tradeExecution) decide(t *commissioner.TradeState, decision string) error {
	if decision == "approve" {
		if t.Status != "approved" {
			if err := x.legal(*t); err != nil {
				return err
			}
		}
		t.Status, t.Words = "approved", "Approved; the assets have changed teams"
	} else {
		if t.VetoNumber == 0 {
			t.VetoNumber = 1
		}
		t.Status, t.Words = "vetoed", "Vetoed; the assets stay with their original teams"
	}
	return nil
}

func (x tradeExecution) review(raw commissioner.RawCommand, t *commissioner.TradeState) error {
	if t.Status != "review" {
		return fmt.Errorf("this trade is not waiting for review")
	}
	if t.Policy.Route == leaguerules.CommissionerReview {
		judge, err := governance.AppealJudge(x.state, tradeParties(x.state, t.Parties))
		if err != nil {
			return fmt.Errorf("trade judge: %w", err)
		}
		if string(judge) != x.principal {
			return fmt.Errorf("only the eligible commissioner may review this trade")
		}
		if raw.Command == commissioner.TradeRecuse {
			return fmt.Errorf("party commissioners are recused automatically")
		}
		t.Votes = append(t.Votes, commissioner.TradeVote{Who: x.principal, Decision: raw.Args.Trade.Decision, At: x.now, Reason: raw.Reason})
		return x.decide(t, raw.Args.Trade.Decision)
	}
	eligible, err := tradeReviewers(t, x.state, x.records, x.now)
	if err != nil {
		return err
	}
	if !slices.Contains(eligible, governance.Principal(x.principal)) {
		return fmt.Errorf("a trade party cannot review its own trade")
	}
	if raw.Command == commissioner.TradeRecuse {
		return fmt.Errorf("party DOT members are recused automatically")
	}
	if hasTradeVote(t.Votes, x.records, x.principal) {
		return fmt.Errorf("each DOT member may vote once")
	}
	t.Votes = append(t.Votes, commissioner.TradeVote{Who: x.principal, Decision: raw.Args.Trade.Decision, At: x.now, Reason: raw.Reason})
	counts := map[string]int{}
	for _, v := range t.Votes {
		if slices.Contains(eligible, governance.Principal(commissioner.PrincipalFor(x.records, v.Who))) {
			counts[v.Decision]++
		}
	}
	if counts[raw.Args.Trade.Decision] >= t.Policy.DecideAt {
		return x.decide(t, raw.Args.Trade.Decision)
	}
	return nil
}

func (x tradeExecution) ruleAppeal(raw commissioner.RawCommand, t *commissioner.TradeState) error {
	if t.Status != "appealed" {
		return fmt.Errorf("this trade has no appeal waiting for a ruling")
	}
	judge, err := governance.AppealJudge(x.state, tradeParties(x.state, t.Parties))
	if err != nil {
		return fmt.Errorf("appeal judge: %w", err)
	}
	if string(judge) != x.principal {
		return fmt.Errorf("only the eligible commissioner, or co-commissioner when the commissioner is a party, may rule")
	}
	t.Appeals = append(t.Appeals, commissioner.TradeVote{Who: x.principal, Decision: raw.Args.Trade.Decision, At: x.now, Reason: raw.Reason})
	return x.decide(t, raw.Args.Trade.Decision)
}

func (x tradeExecution) override(raw commissioner.RawCommand, t *commissioner.TradeState) error {
	a := raw.Args.Trade
	if t.Status != "approved" && t.Status != "vetoed" && t.Status != "appealed" && !x.stalled(t) {
		return fmt.Errorf("override a decided trade, or one the DOT cannot decide")
	}
	if t.Status == "approved" && a.Decision == "approve" {
		return fmt.Errorf("this trade is already approved")
	}
	if t.Status == "approved" && a.Decision == "veto" {
		t.VetoNumber++
	}
	t.Overrides = append(t.Overrides, commissioner.TradeVote{Who: x.principal, Decision: a.Decision, At: x.now, Reason: raw.Reason})
	return x.decide(t, a.Decision)
}

// stalled says a DOT review can never reach its decision count because too many DOT members are trade parties,
// so only the commissioner's override can decide it.
func (x tradeExecution) stalled(t *commissioner.TradeState) bool {
	if t.Status != "review" || t.Policy.Route != leaguerules.DOTReview {
		return false
	}
	eligible, err := tradeReviewers(t, x.state, x.records, x.now)
	return err == nil && t.Status == "review" && len(eligible) < t.Policy.DecideAt
}

func (x tradeExecution) describeAssets(assets []commissioner.TradeAsset) []commissioner.TradeAssetDetail {
	teams := map[string]string{}
	for _, f := range x.seed.Franchises {
		teams[f.ID] = f.Name
	}
	details := make([]commissioner.TradeAssetDetail, 0, len(assets))
	for _, a := range assets {
		d := commissioner.TradeAssetDetail{From: teams[a.From], To: teams[a.To]}
		if a.Player == "" {
			d.Label = fmt.Sprintf("%d round %d pick, originally %s", a.Season, a.Round, teams[a.Original])
		} else {
			for _, p := range x.seed.Players {
				if p.ID != a.Player {
					continue
				}
				d.Label = p.Name
				if d.Label == "" {
					d.Label = x.facts[p.ID].Name
				}
				if d.Label == "" {
					d.Label = "Player " + p.ID
				}
				salary := p.Salary
				d.Salary, d.Ends = &salary, p.ExpirationYear
				d.Contract = nativecheck.ContractWords(p.Salary, max(0, p.ExpirationYear-x.rules.Season+1))
			}
		}
		details = append(details, d)
	}
	return details
}
