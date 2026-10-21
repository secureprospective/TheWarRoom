package counterparties

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
)

type RawCounterpartiesScope struct {
	From     int      `json:"from"`
	To       int      `json:"to"`
	Tiers    []string `json:"tiers"`
	Windows  []string `json:"windows"`
	Postures []string `json:"postures"`
	Team     string   `json:"team"`
}

func (r RawCounterpartiesScope) Validate() error {
	if r.From < 1900 || r.To > 2200 || r.From > r.To {
		return fmt.Errorf("counterparties scope: invalid season range")
	}
	sets := []struct {
		label           string
		values, allowed []string
	}{
		{"tier", r.Tiers, lab.TeamTiers()},
		{"window", r.Windows, []string{"spring", "summer", "season"}},
		{"posture", r.Postures, lab.TeamPostures()},
	}
	for _, set := range sets {
		seen := make(map[string]bool)
		for _, v := range set.values {
			if !slices.Contains(set.allowed, v) || seen[v] {
				return fmt.Errorf("counterparties scope: invalid or duplicate %s %q", set.label, v)
			}
			seen[v] = true
		}
	}
	return nil
}

// RawCounterpartTeam opens one franchise's panel; Scope.Team is ignored because
// the panel itself is the team view.
type RawCounterpartTeam struct {
	Scope RawCounterpartiesScope `json:"scope"`
	Fid   string                 `json:"fid"`
}

// Host supplies detached frozen Lab facts. Only Trades, TeamSeasons, Franchises,
// Built and ArchiveThrough are read; there is no live refresh or authorization gate.
type Host interface {
	CounterpartiesFacts() (*lab.Data, error)
	CounterpartiesBuildStamp() string
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

func (a *API) service() Service {
	return Service{
		Load: a.host.CounterpartiesFacts, BuildStamp: a.host.CounterpartiesBuildStamp(),
		CountNumber: report.Number, CountLabel: report.Count,
		Percent: report.Percent, MarketName: report.AssetName,
	}
}

func (a *API) TargetCounterparties(raw RawCounterpartiesScope) BoardReading {
	return BoardReading(a.service().TargetCounterparties(raw))
}

func (a *API) TargetCounterpartTeam(raw RawCounterpartTeam) TeamCounterpartReading {
	return TeamCounterpartReading(a.service().TargetCounterpartTeam(raw))
}

// Commands describes the existing desktop reads, not the endpoint.open navigation verbs.
func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: catalog.Unset, Method: "TargetCounterpartTeam", Version: catalog.Unset, Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"},
		{ID: catalog.Unset, Method: "TargetCounterparties", Version: catalog.Unset, Role: catalog.Unset, Gravity: catalog.Unset, Availability: "both"},
	}
}

// Concrete wire envelopes let Wails discover the nested feature model types.
type BoardReading report.Reading[Counterparties]
type TeamCounterpartReading report.Reading[TeamCounterpart]
