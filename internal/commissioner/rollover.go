package commissioner

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RolloverState contains only recorded inputs; At is supplied by the caller's clock.
type RolloverState struct {
	Assets  RawNativeSnapshot
	At      time.Time
	Waiting []leaguerules.Change
}

type PendingExpiry struct {
	Player    string `json:"player"`
	Franchise string `json:"franchise"`
	Route     string `json:"route"`
}

// RolloverPlan is a preview, not authority to append or replace league state.
type RolloverPlan struct {
	From         int                        `json:"from"`
	To           int                        `json:"to"`
	BeforeDigest string                     `json:"beforeDigest"`
	Blocked      bool                       `json:"blocked"`
	Reasons      []string                   `json:"reasons"`
	Lines        []string                   `json:"lines"`
	After        RawNativeSnapshot          `json:"after"`
	Pending      []PendingExpiry            `json:"pending"`
	Rules        leaguerules.RuleSet        `json:"rules"`
	Calendar     leagueclock.ActionCalendar `json:"calendar"`
}

// Rollover never mutates its inputs. Missing recorded terms stay missing, not guessed.
func Rollover(state RolloverState, rules leaguerules.RuleSet, from, to int) (RolloverPlan, error) {
	if from < 1 || to > 9999 || to != from+1 || state.Assets.Season != from || rules.Season != from {
		return RolloverPlan{}, fmt.Errorf("move only from the current season to the next season")
	}
	if state.At.IsZero() || rules.LeagueID == "" {
		return RolloverPlan{}, fmt.Errorf("choose the app league and record the rollover time")
	}
	if err := state.Assets.Validate(); err != nil {
		return RolloverPlan{}, fmt.Errorf("preview rollover assets: %w", err)
	}
	body, err := json.Marshal(state.Assets)
	if err != nil {
		return RolloverPlan{}, fmt.Errorf("snapshot before rollover: %w", err)
	}
	plan := RolloverPlan{From: from, To: to, BeforeDigest: fmt.Sprintf("%x", sha256.Sum256(body)),
		Reasons: []string{}, Lines: []string{}, Pending: []PendingExpiry{}, After: cloneRolloverAssets(state.Assets)}
	plan.After.Season = to
	plan.Rules, err = rolloverRules(state, rules, to)
	if err != nil {
		return RolloverPlan{}, err
	}
	plan.agePlayers()
	plan.describeRules(rules)
	plan.dateCalendar(state.At)
	plan.describeCap()
	plan.Lines = append(plan.Lines, "No draft picks are issued at rollover.",
		"Champion archive: not built yet; no champion has been recorded in the native assets.")
	return plan, nil
}

func cloneRolloverAssets(seed RawNativeSnapshot) RawNativeSnapshot {
	seed.Franchises = slices.Clone(seed.Franchises)
	seed.Players = slices.Clone(seed.Players)
	seed.Picks = slices.Clone(seed.Picks)
	seed.Adjustments = slices.Clone(seed.Adjustments)
	return seed
}

func (p *RolloverPlan) agePlayers() {
	for i := range p.After.Players {
		player := &p.After.Players[i]
		if player.Status != domain.RosterActive {
			p.Lines = append(p.Lines, fmt.Sprintf("Player %s returns from %s to the active roster.",
				player.ID, rolloverSquad(player.Status)))
			player.Status = domain.RosterActive
		}
		if player.ExpirationYear == 0 {
			p.Lines = append(p.Lines, fmt.Sprintf("Player %s: contract years are not built yet; no end year recorded.",
				player.ID))
			continue
		}
		before, after := max(0, player.ExpirationYear-p.From+1), max(0, player.ExpirationYear-p.To+1)
		p.Lines = append(p.Lines, fmt.Sprintf("Player %s: contract years remaining change from %d to %d.",
			player.ID, before, after))
		if after == 0 {
			p.Pending = append(p.Pending, PendingExpiry{Player: player.ID, Franchise: player.Franchise,
				Route: "RFA/UFA sorting is not built yet; contract history is not recorded."})
			p.Lines = append(p.Lines, fmt.Sprintf("Player %s is pending a contract decision, not released. "+
				"RFA/UFA sorting is not built yet.", player.ID))
		}
	}
}

func rolloverSquad(status domain.RosterStatus) string {
	if status == domain.RosterIR {
		return "injured reserve"
	}
	return "taxi"
}

func (p *RolloverPlan) block(reason string) {
	p.Blocked = true
	p.Reasons = append(p.Reasons, reason)
	p.Lines = append(p.Lines, "Blocked: "+reason)
}

func (p *RolloverPlan) describeRules(before leaguerules.RuleSet) {
	for _, change := range leaguerules.Diff(before, p.Rules) {
		p.Lines = append(p.Lines, change.Label+" takes effect: "+
			leaguerules.DescribeField(change.Field, change.After).Words+".")
	}
}

func (p *RolloverPlan) describeCap() {
	if p.Rules.Cap.Amount.State != leaguerules.IsKnown {
		p.block("Salary cap needs a ruling: choose the new season's cap.")
	} else {
		value := leaguerules.DescribeField("cap.amount", p.Rules.Cap.Amount)
		p.Lines = append(p.Lines, "The new season's salary cap resets to "+value.Words+".")
	}
	p.Lines = append(p.Lines, "Recorded salary adjustments are preserved with their season and amount.",
		"Dead money carry: not built yet; recorded adjustments do not identify dead money separately.",
		"Future annual contract salaries: not built yet; recorded salaries are unchanged.")
}
