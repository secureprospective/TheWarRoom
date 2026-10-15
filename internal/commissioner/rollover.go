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
	Assets       RawNativeSnapshot
	At           time.Time
	Waiting      []leaguerules.Change
	PlayerNames  map[string]string
	WindowLabels map[string]string
	// NoAssets is a league with no recorded starting assets: only its rules and calendar move.
	NoAssets bool
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

// rolloverDigest fingerprints the complete before-state a plan is made from.
func rolloverDigest(state RolloverState, rules leaguerules.RuleSet) (string, error) {
	body, err := json.Marshal(struct {
		State RolloverState
		Rules leaguerules.RuleSet
	}{state, rules})
	if err != nil {
		return "", fmt.Errorf("snapshot before rollover: %w", err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(body)), nil
}

// Rollover never mutates its inputs. Missing recorded terms stay missing, not guessed.
func Rollover(state RolloverState, rules leaguerules.RuleSet, from, to int) (RolloverPlan, error) {
	if from < 1 || to > lastSeason || to != from+1 || state.Assets.Season != from || rules.Season != from {
		return RolloverPlan{}, fmt.Errorf("move only from the current season to the next season")
	}
	if state.At.IsZero() || rules.LeagueID == "" {
		return RolloverPlan{}, fmt.Errorf("choose the app league and record the rollover time")
	}
	if !state.NoAssets {
		if err := state.Assets.validate(true); err != nil {
			return RolloverPlan{}, fmt.Errorf("preview rollover assets: %w", err)
		}
	}
	digest, err := rolloverDigest(state, rules)
	if err != nil {
		return RolloverPlan{}, err
	}
	plan := RolloverPlan{From: from, To: to, BeforeDigest: digest,
		Reasons: []string{}, Lines: []string{}, Pending: []PendingExpiry{}, After: cloneRolloverAssets(state.Assets)}
	plan.After.Season = to
	plan.Rules, err = leaguerules.Resolve(leaguerules.SeedChanges(rules, rules.LeagueID, from, state.At),
		from, state.At)
	if err != nil {
		return RolloverPlan{}, fmt.Errorf("copy before rollover rules: %w", err)
	}
	plan.redateCalendar(state)
	plan.Rules, err = rolloverRules(state, plan.Rules, to)
	if err != nil {
		return RolloverPlan{}, err
	}
	if state.NoAssets {
		plan.Lines = append(plan.Lines, "No starting assets are recorded, so no players, contracts or picks move.")
	} else {
		plan.agePlayers(state.PlayerNames)
		plan.carrySeasonFacts()
	}
	plan.describeRules(rules)
	plan.dateCalendar(state.At, state.WindowLabels)
	plan.describeCap(state.NoAssets)
	plan.Lines = append(plan.Lines, "No draft picks are issued at rollover.",
		fmt.Sprintf("The league schedule starts empty for season %d; the season %d schedule stays in the ledger.",
			to, from))
	return plan, nil
}

func cloneRolloverAssets(seed RawNativeSnapshot) RawNativeSnapshot {
	seed.Champions = slices.Clone(seed.Champions)
	seed.Franchises = slices.Clone(seed.Franchises)
	seed.Players = slices.Clone(seed.Players)
	seed.Picks = slices.Clone(seed.Picks)
	seed.Adjustments = slices.Clone(seed.Adjustments)
	return seed
}

func (p *RolloverPlan) agePlayers(names map[string]string) {
	for i := range p.After.Players {
		player := &p.After.Players[i]
		name := rolloverPlayerName(*player, names)
		if player.Status != domain.RosterActive {
			p.Lines = append(p.Lines, fmt.Sprintf("%s returns from %s to the active roster.",
				name, rolloverSquad(player.Status)))
			player.Status = domain.RosterActive
		}
		if player.ExpirationYear == 0 {
			p.Lines = append(p.Lines, fmt.Sprintf("%s: contract years are not built yet; no end year recorded.",
				name))
			continue
		}
		before, after := max(0, player.ExpirationYear-p.From+1), max(0, player.ExpirationYear-p.To+1)
		if before == 0 {
			p.Lines = append(p.Lines, fmt.Sprintf("%s: the contract ended in %d and is unchanged.",
				name, player.ExpirationYear))
			continue
		}
		p.Lines = append(p.Lines, fmt.Sprintf("%s: contract years remaining change from %d to %d.",
			name, before, after))
		if after == 0 {
			route := expiryRoute(player.RookieRound)
			player.Pending = true
			p.Pending = append(p.Pending, PendingExpiry{Player: player.ID, Franchise: player.Franchise, Route: route})
			p.Lines = append(p.Lines, name+" is pending a contract decision, not released. "+route)
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
	if slices.Contains(p.Reasons, reason) {
		return
	}
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

func (p *RolloverPlan) describeCap(noAssets bool) {
	if p.Rules.Cap.Amount.State != leaguerules.IsKnown {
		p.block("Salary cap needs a ruling: choose the new season's cap.")
	} else {
		value := leaguerules.DescribeField("cap.amount", p.Rules.Cap.Amount)
		p.Lines = append(p.Lines, "The new season's salary cap resets to "+value.Words+".")
	}
	if !noAssets {
		p.Lines = append(p.Lines, "Future annual contract salaries: not built yet; recorded salaries are unchanged.")
	}
}
