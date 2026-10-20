package trades

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type ProjectionInput struct {
	Session            state.WhatIfSession
	Name               string
	Before, Used, Seed domain.Money
	Players            []state.MirrorPlayer
	Live               map[string]state.PlayerState
	Names              map[string]snapshot.Player
}

func DeadlineReasons(now time.Time, windows []leagueclock.Window, zone *time.Location) []string {
	for _, window := range windows {
		if window.Kind != leagueclock.TradeDeadline {
			continue
		}
		if window.Status != leagueclock.WindowDated || window.At == nil {
			return []string{"Trade deadline cannot be checked: " + window.Reason}
		}
		if !now.Before(*window.At) {
			return []string{"Trade deadline passed " + report.Time(*window.At, zone) + " · " + window.Reason}
		}
		return []string{}
	}
	return []string{"Trade deadline cannot be checked: league window unavailable"}
}

func Legs(r Input, teams []Team, player func(string) (state.PlayerState, bool)) ([]transactions.PlayerMove, []string) {
	moves := []transactions.PlayerMove{}
	blocks := []string{}
	for _, side := range []struct {
		from, to string
		tokens   []string
	}{{r.Mine, r.Other, r.Give}, {r.Other, r.Mine, r.Get}} {
		teamIndex := slices.IndexFunc(teams, func(t Team) bool { return t.ID == side.from })
		if teamIndex < 0 {
			blocks = append(blocks, "Other franchise unavailable")
			continue
		}
		for _, token := range side.tokens {
			index := slices.IndexFunc(teams[teamIndex].Assets, func(a Asset) bool { return a.Token == token })
			if index < 0 {
				blocks = append(blocks, "Selected asset is no longer held by that franchise")
				continue
			}
			if strings.HasPrefix(token, "FP_") {
				continue
			}
			p, ok := player(token)
			if !ok || p.FranchiseID != side.from {
				blocks = append(blocks, "Selected player was removed in what-if; Undo first")
				continue
			}
			moves = append(moves, transactions.PlayerMove{MFLID: token, ToFranchiseID: side.to})
		}
	}
	return moves, blocks
}

func ProjectedTeam(in ProjectionInput, figure func(string, domain.Money, domain.Money) Figure) (Projection, []string) {
	s, name := in.Session, in.Name
	team := Projection{Team: name, Figures: []Figure{}, Roster: []Asset{}}
	var blocks []string
	after := in.Before + in.Used - in.Seed
	futureBefore, futureAfter := domain.Money(0), domain.Money(0)
	active, taxi, activeBefore, taxiBefore := 0, 0, 0, 0
	for _, p := range in.Players {
		if p.FranchiseID == s.Mine {
			countRoster(p.RosterStatus, &activeBefore, &taxiBefore)
		}
		live, ok := in.Live[p.MFLID]
		if !ok {
			continue
		}
		after += CentsCorrection(p, live.FranchiseID, s)
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
		label := in.Names[p.MFLID].Name
		if label == "" {
			label = "Name unavailable"
		}
		team.Roster = append(team.Roster, Asset{Token: p.MFLID, Name: label + " · " + string(live.RosterStatus), Position: in.Names[p.MFLID].Position})
	}
	team.Figures = append(team.Figures, figure("Cap used this season", in.Before, after),
		figure(fmt.Sprintf("%d commitments at today's salaries", s.Snapshot.Season+1), futureBefore, futureAfter))
	team.Figures, blocks = tradeCapFigures(team.Figures, s, in.Before, after, futureBefore, futureAfter, name, figure)
	team.Figures = append(team.Figures, RosterFigure("Active roster · 35–48", activeBefore, active),
		RosterFigure("Taxi squad · maximum 8", taxiBefore, taxi))
	if active < 35 || active > 48 || taxi > 8 {
		blocks = append(blocks, name+": roster limits do not fit (35–48 active; at most 8 taxi)")
	}
	slices.SortFunc(team.Roster, func(a, b Asset) int { return strings.Compare(a.Name, b.Name) })
	return team, blocks
}

func DealText(r Input, teams []Team, m state.MirrorSnapshot) string {
	lines := []string{"Trade agreement for MFL and ProBoards"}
	for _, side := range []struct {
		id     string
		tokens []string
	}{{r.Mine, r.Give}, {r.Other, r.Get}} {
		index := slices.IndexFunc(teams, func(t Team) bool { return t.ID == side.id })
		if index < 0 {
			continue
		}
		lines = append(lines, teams[index].Name+" gives:")
		for _, token := range side.tokens {
			i := slices.IndexFunc(teams[index].Assets, func(a Asset) bool { return a.Token == token })
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

func SimulationPlayers(sessions []state.WhatIfSession) []state.MirrorPlayer {
	out := []state.MirrorPlayer{}
	for _, s := range sessions {
		for _, p := range s.Snapshot.Players {
			if p.FranchiseID == s.Mine {
				out = append(out, p)
			}
		}
	}
	return out
}

func CentsCorrection(p state.MirrorPlayer, afterOwner string, s state.WhatIfSession) domain.Money {
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

func tradeCapFigures(figures []Figure, s state.WhatIfSession, before, after, futureBefore, futureAfter domain.Money, name string, figure func(string, domain.Money, domain.Money) Figure) ([]Figure, []string) {
	blocks := []string{}

	if s.Cap != nil {
		figures = append(figures, figure("Cap room this season", *s.Cap-before, *s.Cap-after),
			figure("Next season’s cap room (this season’s cap carried forward)", *s.Cap-futureBefore, *s.Cap-futureAfter))
		if after > *s.Cap || futureAfter > *s.Cap {
			blocks = append(blocks, name+": deal exceeds the held salary cap")
		}
	} else {
		blocks = append(blocks, name+": salary cap unavailable")
	}
	return figures, blocks
}

func RosterFigure(label string, before, after int) Figure {
	f := Figure{Label: label, Before: strconv.Itoa(before), After: strconv.Itoa(after)}
	if d := after - before; d != 0 {
		f.Change = fmt.Sprintf("%+d", d)
	}
	return f
}

// IR counts toward neither roster limit.
func countRoster(status domain.RosterStatus, active, taxi *int) {
	switch status {
	case domain.RosterActive:
		*active++
	case domain.RosterTaxi:
		*taxi++
	case domain.RosterIR:
	}
}
