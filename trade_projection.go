package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type tradeSimulation struct {
	Sessions []state.WhatIfSession
	Store    *state.Store
	Before   map[string]domain.Money
}

func (a *App) tradeSimulation(ctx context.Context, mine, other string) (tradeSimulation, error) {
	out := tradeSimulation{Before: map[string]domain.Money{}}
	merged := state.MirrorSnapshot{}
	for _, fid := range []string{mine, other} {
		s, store, err := a.whatIfSession(ctx, RawWhatIfRequest{Mine: fid, Action: "read"})
		if err != nil {
			return out, err
		}
		out.Sessions = append(out.Sessions, s)
		used, _ := store.CapUsed(fid)
		out.Before[fid] = s.Used + used - s.SeedUsed
		merged.Season = s.Snapshot.Season
		for _, p := range s.Snapshot.Players {
			if p.FranchiseID != fid {
				continue
			}
			if live, ok := store.Player(p.MFLID); !ok || live.FranchiseID != fid {
				continue
			}
			merged.Players = append(merged.Players, p)
		}
		for _, adjust := range s.Snapshot.Adjustments {
			if adjust.FranchiseID == fid {
				merged.Adjustments = append(merged.Adjustments, adjust)
			}
		}
	}
	seed := out.Sessions[0]
	seed.Generation = "trade-preview:" + fmt.Sprint(time.Now().UnixNano())
	seed.Snapshot = merged
	store, err := a.loadWhatIf(ctx, seed)
	if err != nil {
		return out, err
	}
	out.Store = store
	return out, nil
}
func (a *App) tradeProjection(ctx context.Context, r RawTradeBuilder, snap snapshot.Snapshot) (TradeBuilderPlan, error) {
	out := TradeBuilderPlan{Reasons: []string{}, Teams: []TradeProjection{}}
	sim, err := a.tradeSimulation(ctx, r.Mine, r.Other)
	if err != nil {
		return out, err
	}
	stamps, err := a.league.Stamps(ctx)
	if err != nil {
		return out, fmt.Errorf("trade baseline stamps: %w", err)
	}
	for _, s := range sim.Sessions {
		if stamps.NewerThan(s.Stamps) {
			name := tradeFranchiseName(snap, s.Mine)
			out.Reasons = append(out.Reasons, "MFL changed since "+possessive(name)+" what-if started; reset it to "+
				"check this deal against MFL as it stands.")
			out.Resets = append(out.Resets, TradeReset{Franchise: s.Mine, Label: "Reset " + possessive(name) + " what-if"})
		}
	}
	out.Reasons = append(out.Reasons, a.tradeClockReasons(ctx)...)
	if reason := envelope.TradeFeedReason(snap.Rosters.Provenance); reason != "" {
		out.Reasons = append(out.Reasons, "Roster unavailable: "+reason)
	}
	if reason := a.tradePickReason(r); reason != "" {
		out.Reasons = append(out.Reasons, reason)
	}
	catalog := a.tradeBuilderTeams(ctx, snap)
	moves, blocks := tradeLegs(r, catalog, sim.Store)
	out.Reasons = append(out.Reasons, blocks...)
	before := map[string]domain.Money{}
	for _, s := range sim.Sessions {
		before[s.Mine], _ = sim.Store.CapUsed(s.Mine)
	}
	if len(blocks) == 0 && len(moves) > 0 {
		coord, err := transactions.New(sim.Store.Writer(), nil, nil)
		if err != nil {
			return out, fmt.Errorf("trade preview coordinator: %w", err)
		}
		if _, err := coord.Execute(ctx, transactions.Trade{Moves: moves, Rationale: "Local what-if trade projection"}); err != nil {
			return out, fmt.Errorf("trade preview: %w", err)
		}
	}
	players := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	for _, s := range sim.Sessions {
		team, blocks := tradeProjectedTeam(sim, s, before[s.Mine], players, tradeFranchiseName(snap, s.Mine))
		team.Roster = richTradeRoster(team.Roster, catalog)
		out.Teams, out.Reasons = append(out.Teams, team), append(out.Reasons, blocks...)
	}
	out.Post = tradeDealText(r, catalog, sim.Sessions[0].Snapshot)
	return out, nil
}
func tradeDeadlineReasons(now time.Time, windows []leagueclock.Window) []string {
	for _, window := range windows {
		if window.Kind != leagueclock.TradeDeadline {
			continue
		}
		if window.Status != leagueclock.WindowDated || window.At == nil {
			return []string{"Trade deadline cannot be checked: " + window.Reason}
		}
		if !now.Before(*window.At) {
			return []string{"Trade deadline passed " + reportTime(*window.At, time.Local) + " · " + window.Reason}
		}
		return []string{}
	}
	return []string{"Trade deadline cannot be checked: league window unavailable"}
}
func tradeLegs(r RawTradeBuilder, teams []TradeBuilderTeam, store *state.Store) ([]transactions.PlayerMove, []string) {
	moves := []transactions.PlayerMove{}
	blocks := []string{}
	for _, side := range []struct {
		from, to string
		tokens   []string
	}{{r.Mine, r.Other, r.Give}, {r.Other, r.Mine, r.Get}} {
		teamIndex := slices.IndexFunc(teams, func(t TradeBuilderTeam) bool { return t.ID == side.from })
		if teamIndex < 0 {
			blocks = append(blocks, "Other franchise unavailable")
			continue
		}
		for _, token := range side.tokens {
			index := slices.IndexFunc(teams[teamIndex].Assets, func(a TradeAsset) bool { return a.Token == token })
			if index < 0 {
				blocks = append(blocks, "Selected asset is no longer held by that franchise")
				continue
			}
			if strings.HasPrefix(token, "FP_") {
				continue
			}
			p, ok := store.Player(token)
			if !ok || p.FranchiseID != side.from {
				blocks = append(blocks, "Selected player was removed in what-if; Undo first")
				continue
			}
			moves = append(moves, transactions.PlayerMove{MFLID: token, ToFranchiseID: side.to})
		}
	}
	return moves, blocks
}
func tradeProjectedTeam(sim tradeSimulation, s state.WhatIfSession, seed domain.Money,
	players map[string]snapshot.Player, name string) (TradeProjection, []string) {
	team := TradeProjection{Team: name, Figures: []WhatIfFigure{}, Roster: []TradeAsset{}}
	var blocks []string
	used, _ := sim.Store.CapUsed(s.Mine)
	after := sim.Before[s.Mine] + used - seed
	futureBefore, futureAfter := domain.Money(0), domain.Money(0)
	active, taxi, activeBefore, taxiBefore := 0, 0, 0, 0
	for _, p := range tradeSimulationPlayers(sim) {
		if p.FranchiseID == s.Mine {
			countRoster(p.RosterStatus, &activeBefore, &taxiBefore)
		}
		live, ok := sim.Store.Player(p.MFLID)
		if !ok {
			continue
		}
		after += tradeCentsCorrection(p, live.FranchiseID, s)
		if p.ContractYear > s.Snapshot.Season {
			if p.FranchiseID == s.Mine {
				futureBefore += p.Salary
			}
			if live.FranchiseID == s.Mine {
				futureAfter += p.Salary
			}
		}
		if live.FranchiseID != s.Mine {
			continue
		}
		countRoster(live.RosterStatus, &active, &taxi)
		label := players[p.MFLID].Name
		if label == "" {
			label = "Name unavailable"
		}
		team.Roster = append(team.Roster, TradeAsset{Token: p.MFLID, Name: label + " · " + string(live.RosterStatus), Position: players[p.MFLID].Position})
	}
	team.Figures = append(team.Figures, whatIfFigure("Cap used this season", sim.Before[s.Mine], after),
		whatIfFigure(fmt.Sprintf("%d commitments · inferred", s.Snapshot.Season+1), futureBefore, futureAfter))
	team.Figures, blocks = tradeCapFigures(team.Figures, s, sim.Before[s.Mine], after, futureBefore, futureAfter, name)
	team.Figures = append(team.Figures, rosterFigure("Active roster · 35–48", activeBefore, active),
		rosterFigure("Taxi squad · maximum 8", taxiBefore, taxi))
	if active < 35 || active > 48 || taxi > 8 {
		blocks = append(blocks, name+": roster limits do not fit (35–48 active; at most 8 taxi)")
	}
	slices.SortFunc(team.Roster, func(a, b TradeAsset) int { return strings.Compare(a.Name, b.Name) })
	return team, blocks
}
func tradeDealText(r RawTradeBuilder, teams []TradeBuilderTeam, m state.MirrorSnapshot) string {
	lines := []string{"Trade agreement for MFL and ProBoards"}
	for _, side := range []struct {
		id     string
		tokens []string
	}{{r.Mine, r.Give}, {r.Other, r.Get}} {
		index := slices.IndexFunc(teams, func(t TradeBuilderTeam) bool { return t.ID == side.id })
		if index < 0 {
			continue
		}
		lines = append(lines, teams[index].Name+" gives:")
		for _, token := range side.tokens {
			i := slices.IndexFunc(teams[index].Assets, func(a TradeAsset) bool { return a.Token == token })
			if i < 0 {
				continue
			}
			label := teams[index].Assets[i].Name
			for _, p := range m.Players {
				if p.MFLID == token {
					label += fmt.Sprintf(" · %s a year through %d · %s", snapshot.ContractAmountOf(p.Salary).Text, p.ContractYear, p.ContractStatus)
				}
			}
			lines = append(lines, "- "+label)
		}
		lines = append(lines, "Cap space exchanged: none (salary-cap trading abolished).")
	}
	lines = append(lines, "My rationale: "+r.Rationale, "Other GM's rationale: add before accepting on ProBoards.",
		"Both GMs must agree on ProBoards; then propose and accept on MFL. Three DOT approvals, then commissioner approval.",
		"Once posted and accepted on ProBoards, the trade cannot be withdrawn.", "Verify no selected pick is compensatory; MFL does not identify that here.",
		"Local plan only. Nothing submitted or posted by TheWarRoom.")
	return strings.Join(lines, "\n")
}
func (a *App) tradeBuilderHandOff(ctx context.Context, r RawTradeBuilder, snap snapshot.Snapshot) error {
	plans, err := state.NewTradePlans(a.whatifPools).List(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("trade hand-off plans: %w", err)
	}
	blocks := []string{}
	for _, p := range plans {
		if p.ID != r.ID {
			continue
		}
		preview, err := a.tradeProjection(ctx, RawTradeBuilder{Mine: p.Mine, Other: p.Other, Give: p.Give, Get: p.Get, Rationale: p.Rationale}, snap)
		if err != nil {
			return err
		}
		blocks = preview.Reasons
	}
	if len(blocks) > 0 {
		return fmt.Errorf("trade hand-off blocked: %s", strings.Join(blocks, "; "))
	}
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	e, err := a.moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("trade hand-off load: %w", err)
	}
	spec := e.Receipt().Spec
	if spec.FranchiseID != r.Mine || !envelope.IsTrade(spec.Intent) || spec.Intent == "trade.accept" {
		return fmt.Errorf("trade hand-off outside scope")
	}
	if err := a.checkTradeExitPending(r.Mine, spec); err != nil {
		return err
	}
	if err := validateMoveURL(spec.Target.URL); err != nil {
		return err
	}
	next, err := e.HandOff(a.movesNow())
	if err != nil {
		return fmt.Errorf("trade hand-off: %w", err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return fmt.Errorf("trade hand-off save: %w", err)
	}
	a.openURL(ctx, spec.Target.URL)
	a.movesChanged(ctx)
	a.wakeMoves()
	return nil
}

func tradeSimulationPlayers(sim tradeSimulation) []state.MirrorPlayer {
	out := []state.MirrorPlayer{}
	for _, s := range sim.Sessions {
		for _, p := range s.Snapshot.Players {
			if p.FranchiseID == s.Mine {
				out = append(out, p)
			}
		}
	}
	return out
}
func tradeCentsCorrection(p state.MirrorPlayer, afterOwner string, s state.WhatIfSession) domain.Money {
	difference := state.CapContribution(p.Salary, p.RosterStatus, s) -
		domain.RoundToNearest10k(state.CapContribution(domain.RoundToNearest10k(p.Salary), p.RosterStatus, s))
	if p.FranchiseID == s.Mine && afterOwner != s.Mine {
		return -difference
	}
	if p.FranchiseID != s.Mine && afterOwner == s.Mine {
		return difference
	}
	return 0
}

func (a *App) checkTradeExitPending(mine string, spec envelope.Spec) error {
	if spec.Intent == "trade.propose" {
		return nil
	}
	_, held, err := a.tradeState(mine)
	if err != nil {
		return err
	}
	if envelope.TradeFeedReason(held.Provenance) != "" || !slices.ContainsFunc(held.Value, func(p leaguefeed.PendingTrade) bool {
		return p.ID == spec.Expected.Trade.TradeID && envelope.MatchesPendingTrade(*spec.Expected.Trade, p)
	}) {
		return fmt.Errorf("offer no longer available; refresh the trade desk")
	}
	return nil
}

func tradeCapFigures(figures []WhatIfFigure, s state.WhatIfSession, before, after, futureBefore, futureAfter domain.Money, name string) ([]WhatIfFigure, []string) {
	blocks := []string{}

	if s.Cap != nil {
		figures = append(figures, whatIfFigure("Cap room this season", *s.Cap-before, *s.Cap-after),
			whatIfFigure("Next-season room · held cap, inferred", *s.Cap-futureBefore, *s.Cap-futureAfter))
		if after > *s.Cap || futureAfter > *s.Cap {
			blocks = append(blocks, name+": deal exceeds the held salary cap")
		}
	} else {
		blocks = append(blocks, name+": salary cap unavailable")
	}
	return figures, blocks
}

func (a *App) tradePickReason(r RawTradeBuilder) string {
	if !slices.ContainsFunc(append(slices.Clone(r.Give), r.Get...), func(token string) bool { return strings.HasPrefix(token, "FP_") }) {
		return ""
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	held := a.seasonFuturePicks
	if held.fetchedAt.IsZero() {
		return "Draft-pick ownership unavailable; refresh before handing off."
	}
	if held.refreshError != "" {
		return "Draft-pick ownership unavailable: " + held.refreshError
	}
	return ""
}

// rosterFigure is a roster count before and after the deal, with the change when there is one.
func rosterFigure(label string, before, after int) WhatIfFigure {
	f := WhatIfFigure{Label: label, Before: strconv.Itoa(before), After: strconv.Itoa(after)}
	if d := after - before; d != 0 {
		f.Change = fmt.Sprintf("%+d", d)
	}
	return f
}

// countRoster adds a player to the active or taxi count; IR counts toward neither (rulebook).
func countRoster(status domain.RosterStatus, active, taxi *int) {
	switch status {
	case domain.RosterActive:
		*active++
	case domain.RosterTaxi:
		*taxi++
	case domain.RosterIR:
	}
}

// tradeClockReasons checks the deal against the effective trade deadline. A clock that cannot be read
// blocks the deal with its reason rather than failing the whole check.
func (a *App) tradeClockReasons(ctx context.Context) []string {
	clock, err := a.targetClock(ctx)
	if err != nil {
		return []string{"Trade deadline cannot be checked: " + err.Error()}
	}
	return tradeDeadlineReasons(time.Now(), clock.Value.Windows)
}
