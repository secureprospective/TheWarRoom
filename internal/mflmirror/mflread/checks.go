package mflread

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/mflmirror"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/mflref/lineup"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions/deadcap" //nolint:depguard // Charge is the pure §8 rate; nothing is written to a ledger
)

// The bid and lineup checks run on the held copy and the owner's picked team. They reuse the desktop's
// own check functions (nativecheck, lineup, envelope, deadcap) and never call MFL.

// view is the owner's side of the held copy.
type view struct{ mflmirror.Seat }

func (v view) name(id string) string {
	if v.Lookup != nil {
		if f, ok := v.Lookup.Facts(id); ok && f.Name != "" {
			return f.Name
		}
	}
	return "player " + id
}

// myRoster is the picked team's roster rows by player.
func (v view) myRoster() map[string]mflmirror.RosterRow {
	out := map[string]mflmirror.RosterRow{}
	for _, r := range v.Held.League.Rosters {
		if r.Franchise == v.Team {
			out[r.Player] = r
		}
	}
	return out
}

func rosterStatus(mfl string) domain.RosterStatus {
	switch strings.TrimSpace(mfl) {
	case "TAXI_SQUAD":
		return domain.RosterTaxi
	case "INJURED_RESERVE":
		return domain.RosterIR
	}
	return domain.RosterActive
}

// discounts reads how much of a taxi or IR salary counts against the cap from MFL's league rules.
type discounts struct{ taxi, ir float64 }

func (d discounts) TaxiCapPercent() float64 { return d.taxi }
func (d discounts) IRCapPercent() float64   { return d.ir }

// config decodes MFL's league rules from the held copy.
func (v view) config() (league.RawConfig, error) {
	var c league.RawConfig
	if len(v.Held.League.Config) == 0 {
		return c, fmt.Errorf("the held copy has no MFL league rules")
	}
	if err := json.Unmarshal(v.Held.League.Config, &c); err != nil {
		return c, fmt.Errorf("held MFL league rules: %w", err)
	}
	return c, nil
}

func (v view) discounts() discounts {
	pct := func(raw string) float64 {
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || n < 0 || n > 100 {
			return 100
		}
		return n
	}
	c, _ := v.config()
	return discounts{taxi: pct(c.IncludeTaxiWithSalary), ir: pct(c.IncludeIRWithSalary)}
}

// capUsed is what the team's salaries and literal MFL adjustments count against the cap.
func (v view) capUsed(skip map[string]bool) domain.Money {
	d := v.discounts()
	var used domain.Money
	for _, r := range v.Held.League.Rosters {
		if r.Franchise == v.Team && !skip[r.Player] {
			used += state.CapContribution(domain.Money(r.Salary), rosterStatus(r.Status), d)
		}
	}
	for _, adj := range v.Held.League.Adjustments {
		if adj.Franchise == v.Team {
			used += domain.Money(adj.Cents)
		}
	}
	return used
}

func (v view) salaryCap() *domain.Money {
	c, _ := v.config()
	raw := strings.TrimSpace(c.SalaryCapAmount)
	if raw == "" {
		return nil
	}
	m, err := domain.ParseMoneyMillions(raw)
	if err != nil || m <= 0 {
		return nil
	}
	return &m
}

// BidCheck judges one bid on the owner's MFL team from the held copy and saves nothing.
func (Fetcher) BidCheck(_ context.Context, s mflmirror.Seat, in mflmirror.BidInput) mflmirror.BidCheck {
	return view{s}.bidCheck(in)
}

func (v view) bidCheck(in mflmirror.BidInput) mflmirror.BidCheck {
	years := max(in.Years, 1)
	salary := domain.Money(math.Round(in.Amount * 100))
	out := mflmirror.BidCheck{League: v.League, Player: v.name(in.Add), Salary: salary, Years: years,
		Problems: []string{}, NotChecked: []string{}}
	facts, known := domain.PlayerFacts{}, false
	if v.Lookup != nil {
		facts, known = v.Lookup.Facts(in.Add)
	}
	roster := v.myRoster()
	if !known {
		out.Problems = append(out.Problems, "That player is not in MFL's player list.")
	}
	if _, mine := roster[in.Add]; mine {
		out.Problems = append(out.Problems, out.Player+" is already on your roster.")
	}
	for _, r := range v.Held.League.Rosters {
		if r.Player == in.Add && r.Franchise != v.Team {
			out.Problems = append(out.Problems, out.Player+" is on another team's roster, not a free agent.")
			break
		}
	}
	dropped, deadCap, dropProblems := v.drops(in.Drop, roster)
	out.Problems = append(out.Problems, dropProblems...)
	// Cap room after the bid: salaries now, minus the dropped players' salaries, plus their dead cap, plus the bid.
	before := v.capUsed(nil)
	after := v.capUsed(dropped) + deadCap + salary
	active := activeAfter(roster, dropped)
	out.CapUsedBefore, out.CapUsedAfter, out.ActiveAfter = before, after, active
	salaryCap := v.salaryCap()
	if salaryCap == nil {
		out.NotChecked = append(out.NotChecked, "MFL names no salary cap for this league, so cap room was not checked.")
	} else {
		room := *salaryCap - after
		out.CapRoomAfter = &room
	}
	bounds, boundNotes := v.rosterBounds()
	out.NotChecked = append(out.NotChecked, boundNotes...)
	out.Problems = append(out.Problems, nativecheck.AcquisitionReasons(salaryCap, after, active, bounds)...)
	out.Problems = append(out.Problems, v.reserveProblems(roster, dropped)...)
	out.Problems, out.NotChecked = v.bidRules(facts, years, salary, out.Problems, out.NotChecked)
	out.OK = len(out.Problems) == 0
	out.Words = "This bid fits your cap and roster."
	if !out.OK {
		out.Words = "This bid has problems; fix them before you plan it."
	}
	return out
}

// rosterBounds is the league's own active roster size from MFL's rules. MFL names a maximum (rosterSize,
// which leaves IR and the taxi squad out) and no minimum, so a missing figure is listed as not checked.
func (v view) rosterBounds() (nativecheck.RosterBounds, []string) {
	c, _ := v.config()
	notes := []string{"MFL names no roster minimum, so the minimum roster size was not checked."}
	size, err := strconv.Atoi(strings.TrimSpace(c.RosterSize))
	if err != nil || size <= 0 {
		return nativecheck.RosterBounds{}, append(notes, "MFL names no roster size for this league, so the roster maximum was not checked.")
	}
	return nativecheck.RosterBounds{Max: size}, notes
}

// reserveProblems flags an injured reserve or taxi squad that already holds more than the league's slots; a
// bid cannot fix it, but MFL will not accept moves from a team over its limits.
func (v view) reserveProblems(roster map[string]mflmirror.RosterRow, dropped map[string]bool) []string {
	c, _ := v.config()
	var ir, taxi int
	for id, r := range roster {
		if dropped[id] {
			continue
		}
		switch rosterStatus(r.Status) {
		case domain.RosterIR:
			ir++
		case domain.RosterTaxi:
			taxi++
		case domain.RosterActive:
		}
	}
	out := []string{}
	if slots, err := strconv.Atoi(strings.TrimSpace(c.InjuredReserve)); err == nil && slots >= 0 && ir > slots {
		out = append(out, fmt.Sprintf("Your injured reserve holds %d players but the league allows %d.", ir, slots))
	}
	if slots, err := strconv.Atoi(strings.TrimSpace(c.TaxiSquad)); err == nil && slots >= 0 && taxi > slots {
		out = append(out, fmt.Sprintf("Your taxi squad holds %d players but the league allows %d.", taxi, slots))
	}
	return out
}

// drops reads the players to drop: the ones on the roster, the dead cap their cuts cost, and what is wrong.
func (v view) drops(ids []string, roster map[string]mflmirror.RosterRow) (map[string]bool, domain.Money, []string) {
	dropped := map[string]bool{}
	var deadCap domain.Money
	problems := []string{}
	for _, id := range ids {
		row, mine := roster[id]
		switch {
		case !mine:
			problems = append(problems, v.name(id)+" is not on your roster, so he cannot be dropped.")
		case dropped[id]:
			problems = append(problems, v.name(id)+" is listed twice to drop.")
		default:
			dropped[id] = true
			deadCap += cutCharge(row, v.Held.Season)
		}
	}
	return dropped, deadCap, problems
}

// activeAfter counts the active players left after the drops, plus the one added.
func activeAfter(roster map[string]mflmirror.RosterRow, dropped map[string]bool) int {
	active := 1
	for id, r := range roster {
		if rosterStatus(r.Status) == domain.RosterActive && !dropped[id] {
			active++
		}
	}
	return active
}

// bidRules adds the minimum-salary and bid-limit problems. The minimum is the one the pool shows
// (bid.MinimumSalary); the one year limit and the years a salary needs come from the league's bid rules.
func (v view) bidRules(facts domain.PlayerFacts, years int, salary domain.Money, problems, notChecked []string) ([]string, []string) {
	exp := bid.ExperienceFrom(v.Held.Season, facts.DraftYear, facts.HasDraftYear).Years
	if minimum := bid.MinimumSalary(exp); salary < minimum {
		problems = append(problems, fmt.Sprintf("The minimum salary for this player is %s a year.", bid.MoneyText(minimum)))
	}
	if v.RulesOK {
		if policy, err := nativecheck.NewBidPolicy(v.Rules); err == nil {
			// The offseason flag keeps the strict one year limit; a playoff-week lift is not applied here.
			refusal := policy.LimitRefusal(nativecheck.BidProposal{Salary: salary, Years: years},
				nativecheck.BidFacts{Season: nativecheck.BidSeason{Offseason: true}})
			if refusal != "" {
				problems = append(problems, refusal)
			}
			return problems, notChecked
		}
	}
	return problems, append(notChecked, "This league's bid limits are not all set in its rules, so the bid length limits were not checked.")
}

// cutCharge is the unclaimed cut's dead cap for a dropped player: the desktop's §8 rule over his
// remaining paid seasons at today's flat salary.
func cutCharge(row mflmirror.RosterRow, season int) domain.Money {
	cells := []state.LedgerCell{}
	for year := season + 1; year <= row.ContractYear; year++ {
		cells = append(cells, state.LedgerCell{Year: year, Salary: domain.Money(row.Salary)})
	}
	return deadcap.Charge(cells, season, false, false)
}

// heldSnapshot is the held MFL copy in the shape the desktop lineup check reads: each team's roster and
// the players' positions.
func (v view) heldSnapshot() snapshot.Snapshot {
	var snap snapshot.Snapshot
	at := map[string]int{}
	seen := map[string]bool{}
	for _, r := range v.Held.League.Rosters {
		id, err := playerid.New(r.Player)
		if err != nil {
			continue
		}
		i, ok := at[r.Franchise]
		if !ok {
			i = len(snap.Rosters.Value)
			at[r.Franchise] = i
			snap.Rosters.Value = append(snap.Rosters.Value, snapshot.Roster{FranchiseID: r.Franchise})
		}
		roster := &snap.Rosters.Value[i]
		roster.Players = append(roster.Players, snapshot.RosterPlayer{ID: id, Salary: domain.Money(r.Salary),
			RosterStatus: rosterStatus(r.Status)})
		if v.Lookup == nil {
			continue
		}
		if f, ok := v.Lookup.Facts(id.String()); ok && !seen[id.String()] {
			seen[id.String()] = true
			snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: id, Name: f.Name, Position: f.Position, Team: f.Team})
		}
	}
	return snap
}

func starterIDs(raw []string) ([]playerid.PlayerID, error) {
	if len(raw) == 0 || len(raw) > 40 {
		return nil, fmt.Errorf("choose your starters")
	}
	ids := make([]playerid.PlayerID, 0, len(raw))
	seen := map[playerid.PlayerID]bool{}
	for _, s := range raw {
		id, err := playerid.New(s)
		if err != nil {
			return nil, fmt.Errorf("choose starters from your roster")
		}
		if seen[id] {
			return nil, fmt.Errorf("a starter is listed twice")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

// LineupCheck judges one week's lineup on the owner's MFL team from the held copy.
func (Fetcher) LineupCheck(_ context.Context, s mflmirror.Seat, in mflmirror.LineupInput) (mflmirror.LineupCheck, error) {
	ids, err := starterIDs(in.Starters)
	if err != nil {
		return mflmirror.LineupCheck{}, err
	}
	return view{s}.lineupCheck(in.Week, ids)
}

func (v view) lineupCheck(week int, ids []playerid.PlayerID) (mflmirror.LineupCheck, error) {
	out := mflmirror.LineupCheck{League: v.League, Week: week, Problems: []mflmirror.LineupProblem{}}
	config, err := v.config()
	if err != nil {
		return out, err
	}
	rules, err := lineup.ParseRules(config)
	if err != nil {
		return out, fmt.Errorf("MFL's starting lineup rules could not be read: %w", err)
	}
	snap := v.heldSnapshot()
	problems, blocked := v.starterProblems(ids, week)
	out.Problems = append(out.Problems, problems...)
	// Slot counts: the same two calls the desktop lineup check makes.
	result := lineup.Check(rules, envelope.LineupPositions(ids, snap))
	for _, p := range result.Problems {
		out.Problems = append(out.Problems, mflmirror.LineupProblem{Subject: p.Subject, Kind: p.Kind, Message: p.Message})
	}
	out.Full = result.Full && !blocked
	out.OK = result.Legal && !blocked
	switch {
	case !out.OK:
		out.Words = "This lineup has problems; MFL would not take it as it is."
	case !out.Full:
		out.Words = "This lineup is allowed but not full."
	default:
		out.Words = "This lineup is full and allowed."
	}
	return out, nil
}

// starterProblems finds the starters who cannot play: off the roster, on injured reserve or the taxi squad,
// or on bye that week. blocked is true when there is any.
func (v view) starterProblems(ids []playerid.PlayerID, week int) (out []mflmirror.LineupProblem, blocked bool) {
	roster := v.myRoster()
	for _, id := range ids {
		row, mine := roster[id.String()]
		name := v.name(id.String())
		var facts domain.PlayerFacts
		if v.Lookup != nil {
			facts, _ = v.Lookup.Facts(id.String())
		}
		switch {
		case !mine:
			out = append(out, mflmirror.LineupProblem{Subject: name, Kind: "not-on-roster",
				Message: name + " is not on your roster."})
			blocked = true
		case rosterStatus(row.Status) == domain.RosterIR:
			out = append(out, mflmirror.LineupProblem{Subject: name, Kind: "injured-reserve",
				Message: name + " is on injured reserve and cannot start."})
			blocked = true
		case rosterStatus(row.Status) == domain.RosterTaxi:
			out = append(out, mflmirror.LineupProblem{Subject: name, Kind: "taxi-squad",
				Message: name + " is on your taxi squad and cannot start."})
			blocked = true
		}
		if w, known := v.Held.League.Byes[facts.Team]; known && w == week {
			out = append(out, mflmirror.LineupProblem{Subject: name, Kind: "bye",
				Message: fmt.Sprintf("%s (%s) is on bye in week %d.", name, facts.Team, week)})
			blocked = true
		}
	}
	return out, blocked
}
