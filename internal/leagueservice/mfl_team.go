package leagueservice

import (
	"context"
	"fmt"
	"strings"
)

const (
	MflMovesTeamSet   = "mflmoves.team.set"
	MflMovesTeamsRead = "mflmoves.teams.read"
)

// RawMflTeamSet is the owner's choice of their own team in the MFL league. The seat names the app
// franchise; a request may not.
type RawMflTeamSet struct {
	League    string `json:"league"`
	Franchise string `json:"franchise,omitempty"`
	Team      string `json:"team"`
}

func (r RawMflTeamSet) Validate() error {
	if err := (RawMflMovesScope{League: r.League, Franchise: r.Franchise}).Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Team) == "" || len(r.Team) > 16 || strings.TrimSpace(r.Team) != r.Team {
		return fmt.Errorf("choose one of the MFL teams from the list")
	}
	return nil
}

// MflTeam is one team in the MFL league's public list; Yours marks the seat's own choice.
type MflTeam struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Yours bool   `json:"yours"`
}

type MflTeams struct {
	League    string    `json:"league"`
	MFLLeague string    `json:"mflLeague"`
	Teams     []MflTeam `json:"teams"`
	Yours     string    `json:"yours,omitempty"`
	Words     string    `json:"words"`
}

func (a *Service) MflMovesTeamSet(raw RawMflTeamSet) (MflTeams, error) {
	return a.MflMovesTeamSetContext(a.host.CommissionerContext(), raw)
}

// MflMovesTeamSetContext saves which MFL team is the seated owner's. Only the seat's own record is written.
func (a *Service) MflMovesTeamSetContext(ctx context.Context, raw RawMflTeamSet) (MflTeams, error) {
	if err := raw.Validate(); err != nil {
		return MflTeams{}, fmt.Errorf("MFL team input: %w", err)
	}
	ctx, seat, err := a.mflAccess(ctx, raw.League, MflMovesTeamSet, "execute")
	if err != nil {
		return MflTeams{}, err
	}
	defer seat.cancel()
	if err := seat.store.SetTeam(ctx, seat.franchise, seat.mflLeague, raw.Team); err != nil {
		return MflTeams{}, mflRefusal(fmt.Errorf("save MFL team: %w", err))
	}
	return a.mflTeams(ctx, seat, raw.League)
}

func (a *Service) MflMovesTeamsRead(raw RawMflMovesScope) (MflTeams, error) {
	return a.MflMovesTeamsReadContext(a.host.CommissionerContext(), raw)
}

// MflMovesTeamsReadContext lists the MFL league's public teams and marks the seat's own choice.
func (a *Service) MflMovesTeamsReadContext(ctx context.Context, raw RawMflMovesScope) (MflTeams, error) {
	if err := raw.Validate(); err != nil {
		return MflTeams{}, fmt.Errorf("MFL teams input: %w", err)
	}
	ctx, seat, err := a.mflSeatAccess(ctx, raw.League, MflMovesTeamsRead, "read", true)
	if err != nil {
		return MflTeams{}, err
	}
	defer seat.cancel()
	if seat.mflLeague == "" {
		return MflTeams{
			League: raw.League,
			Teams:  []MflTeam{},
			Words:  "This league follows no MFL league yet; the commissioner sets it in league settings.",
		}, nil
	}
	return a.mflTeams(ctx, seat, raw.League)
}

func (a *Service) mflTeams(ctx context.Context, seat mflSeat, league string) (MflTeams, error) {
	held, ok, err := seat.store.Read(ctx, seat.mflLeague)
	if err != nil {
		return MflTeams{}, fmt.Errorf("read held MFL teams: %w", err)
	}
	if !ok || len(held.Franchises) == 0 {
		return MflTeams{
			League:    league,
			MFLLeague: seat.mflLeague,
			Teams:     []MflTeam{},
			Words:     "Refresh from MFL to load the teams.",
		}, nil
	}
	yours, _, err := seat.store.Team(ctx, seat.franchise, seat.mflLeague)
	if err != nil {
		return MflTeams{}, fmt.Errorf("read your MFL team: %w", err)
	}
	out := MflTeams{League: league, MFLLeague: seat.mflLeague, Teams: make([]MflTeam, 0, len(held.Franchises)), Yours: yours}
	for _, f := range held.Franchises {
		name := f.Name
		if name == "" {
			name = f.ID
		}
		out.Teams = append(out.Teams, MflTeam{ID: f.ID, Name: name, Yours: f.ID == yours})
	}
	out.Words = "Pick which team is yours."
	if yours != "" {
		out.Words = "Your MFL team is set."
	}
	return out, nil
}
