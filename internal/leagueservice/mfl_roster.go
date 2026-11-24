package leagueservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

const MflMovesRosterRead = "mflmoves.roster.read"

// MflRosterPlayer is one player the owner's MFL team holds, from the held copy.
type MflRosterPlayer struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	Position        string       `json:"position"`
	Team            string       `json:"team"`
	Salary          domain.Money `json:"salaryCents"`
	ContractThrough int          `json:"contractThrough"`
	Status          string       `json:"status"` // active, IR or taxi
}

type MflRoster struct {
	League    string            `json:"league"`
	MFLLeague string            `json:"mflLeague"`
	Team      string            `json:"team"`
	Players   []MflRosterPlayer `json:"players"`
	Words     string            `json:"words"`
}

func (a *Service) MflMovesRosterRead(raw RawMflMovesScope) (MflRoster, error) {
	return a.MflMovesRosterReadContext(a.host.CommissionerContext(), raw)
}

// MflMovesRosterReadContext lists the players on the seated owner's picked MFL team from the held copy. It
// never calls MFL; the team comes from the seat's own pick, never from the request.
func (a *Service) MflMovesRosterReadContext(ctx context.Context, raw RawMflMovesScope) (MflRoster, error) {
	if err := raw.Validate(); err != nil {
		return MflRoster{}, fmt.Errorf("MFL roster input: %w", err)
	}
	_, seat, done, err := a.mflOwnerSeat(ctx, raw.League, MflMovesRosterRead, "read", true)
	if err != nil {
		return MflRoster{}, err
	}
	defer done()
	out := MflRoster{League: raw.League, MFLLeague: seat.MFLLeague, Team: seat.Team, Players: []MflRosterPlayer{}}
	for _, r := range seat.Held.League.Rosters {
		if r.Franchise != seat.Team {
			continue
		}
		p := MflRosterPlayer{ID: r.Player, Name: "player " + r.Player, Salary: domain.Money(r.Salary),
			ContractThrough: r.ContractYear, Status: mflRosterStatus(r.Status)}
		if seat.Lookup != nil {
			if f, ok := seat.Lookup.Facts(r.Player); ok {
				if f.Name != "" {
					p.Name = f.Name
				}
				p.Position, p.Team = string(f.Position), f.Team
			}
		}
		out.Players = append(out.Players, p)
	}
	out.Words = fmt.Sprintf("%d players on your MFL team, from the held copy.", len(out.Players))
	return out, nil
}

func mflRosterStatus(mfl string) string {
	switch strings.TrimSpace(mfl) {
	case "TAXI_SQUAD":
		return "taxi"
	case "INJURED_RESERVE":
		return "IR"
	}
	return "active"
}
