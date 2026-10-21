package race

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/season"
)

// Host captures the private and shared season data together and owns refresh serialization.
type Host interface {
	Ready() error
	RaceInput(func() season.Held) season.Snapshot
	RefreshRace(bool, func() season.Held) error
}

type API struct {
	host Host
	held *Held
}

func New(host Host, held *Held) *API { return &API{host: host, held: held} }

//nolint:revive // Preserve the existing exported wire name.
type RaceBoard struct {
	Standings report.Report   `json:"standings"`
	Board     report.Board    `json:"board"`
	Week      int             `json:"week"`
	Weeks     []report.Choice `json:"weeks"`
	Team      string          `json:"team"`
	Teams     []report.Choice `json:"teams"`
}

func (a *API) TargetRace(week int, team, mine string) (RaceBoard, error) {
	if err := a.host.Ready(); err != nil {
		return RaceBoard{}, fmt.Errorf("race: startup: %w", err)
	}
	in := a.host.RaceInput(a.held.Snapshot)
	in.Week, in.Team, in.Mine = week, team, mine
	b, err := Build(in)
	return wireRace(b), err
}

func (a *API) TargetRaceRefresh(force bool) error {
	return a.host.RefreshRace(force, a.held.Snapshot) //nolint:wrapcheck // Preserve the existing refresh error.
}

func wireRace(b Board) RaceBoard {
	return RaceBoard(b)
}

func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: catalog.Unset, Method: "TargetRace", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetRaceRefresh", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
	}
}
