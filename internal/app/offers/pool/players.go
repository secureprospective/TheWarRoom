// Package pool shapes player-pool rows: age, position order and each player's plan.
package pool

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type Plan struct{ ID, State string }
type Player struct {
	ID, Name, Position, Team, Age, Experience string
	Minimum                                   snapshot.ContractAmount
	Plan, PlanID                              string
}

func Row(id playerid.PlayerID, f normalize.PlayerFacts, season int, now time.Time, plan Plan) Player {
	exp := bid.ExperienceFrom(season, f.DraftYear, f.HasDraftYear)
	experience := "Unknown (rookie minimum)"
	switch {
	case exp.Known && exp.Years == 0:
		experience = "Rookie"
	case exp.Known && exp.Years == 1:
		experience = "1 year"
	case exp.Known:
		experience = fmt.Sprintf("%d years", exp.Years)
	}
	minimum := bid.MinimumSalary(exp.Years)
	return Player{ID: id.String(), Name: f.Name, Position: string(f.Position), Team: f.Team,
		Age: AgeText(f, now), Experience: experience,
		Minimum: snapshot.ContractAmount{Cents: minimum, Text: bid.MoneyText(minimum)},
		Plan:    plan.State, PlanID: plan.ID}
}

func AgeText(f normalize.PlayerFacts, now time.Time) string {
	if !f.HasBirthdate {
		return "—"
	}
	born, today := time.Unix(f.Birthdate, 0).UTC(), now.UTC()
	years := today.Year() - born.Year()
	if today.Month() < born.Month() || (today.Month() == born.Month() && today.Day() < born.Day()) {
		years--
	}
	return strconv.Itoa(years)
}

func PositionOrder(pos string) int {
	order := []domain.Position{domain.PosQB, domain.PosRB, domain.PosWR, domain.PosTE, domain.PosK,
		domain.PosDE, domain.PosDT, domain.PosLB, domain.PosCB, domain.PosS}
	if i := slices.Index(order, domain.Position(pos)); i >= 0 {
		return i
	}
	return len(order)
}

func Positions(seen map[string]bool) []string {
	out := []string{}
	for pos := range seen {
		out = append(out, pos)
	}
	slices.SortFunc(out, func(x, y string) int {
		return cmp.Or(cmp.Compare(PositionOrder(x), PositionOrder(y)), strings.Compare(x, y))
	})
	return out
}

func SortIDs(ids []playerid.PlayerID) {
	slices.SortFunc(ids, func(a, b playerid.PlayerID) int { return strings.Compare(a.String(), b.String()) })
}
