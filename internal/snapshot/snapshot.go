// Package snapshot builds the target UI contract without fetching or writing data.
package snapshot

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type Provenance struct {
	Source    string           `json:"source"`
	Freshness domain.Freshness `json:"freshness"`
	Kind      string           `json:"kind"`
}

type Sourced[T any] struct {
	Value      T          `json:"value"`
	Provenance Provenance `json:"provenance"`
}

type Snapshot struct {
	League     Sourced[League]      `json:"league"`
	Franchises Sourced[[]Franchise] `json:"franchises"`
	Rosters    Sourced[[]Roster]    `json:"rosters"`
	Players    Sourced[[]Player]    `json:"players"`
}

type League struct {
	Season         int           `json:"season"`
	FranchiseCount int           `json:"franchiseCount"`
	SalaryCap      *domain.Money `json:"salaryCap,omitempty"`
}

type Franchise struct {
	ID      string        `json:"id"`
	Name    string        `json:"name,omitempty"`
	CapUsed *domain.Money `json:"capUsed,omitempty"`
	CapRoom *domain.Money `json:"capRoom,omitempty"`
}

type Roster struct {
	FranchiseID string         `json:"franchiseId"`
	Players     []RosterPlayer `json:"players"`
}

type RosterPlayer struct {
	ID             playerid.PlayerID     `json:"id"`
	Salary         domain.Money          `json:"salary"`
	YearsRemaining *int                  `json:"yearsRemaining,omitempty"`
	ContractStatus domain.ContractStatus `json:"contractStatus"`
	RosterStatus   domain.RosterStatus   `json:"rosterStatus"`
}

type Player struct {
	ID        playerid.PlayerID `json:"id"`
	Name      string            `json:"name,omitempty"`
	Position  domain.Position   `json:"position,omitempty"`
	Team      string            `json:"team,omitempty"`
	Birthdate *int64            `json:"birthdate,omitempty"`
	IsRookie  *bool             `json:"isRookie,omitempty"`
}

type Directory struct {
	Lookup     normalize.Lookup
	Provenance Provenance
}

type Metadata struct {
	Season     int
	SalaryCap  *domain.Money
	Names      map[string]string
	MirrorAsOf string
}

type Source interface {
	Franchises() []string
	Roster(string) ([]state.PlayerState, bool)
	CapUsed(string) (domain.Money, bool)
	Metadata() (Metadata, error)
}

type Mirror interface {
	Reader() state.Reader
	Season() int
	AsOf() string
}

type Rules interface {
	FranchiseNames() map[string]string
	GetSalaryCap() string
}

type storeSource struct {
	state.Reader
	mirror Mirror
	rules  Rules
}

func NewSource(m Mirror, r Rules) Source {
	return storeSource{Reader: m.Reader(), mirror: m, rules: r}
}

func (s storeSource) Metadata() (Metadata, error) {
	meta := Metadata{Season: s.mirror.Season(), Names: s.rules.FranchiseNames(), MirrorAsOf: s.mirror.AsOf()}
	if raw := strings.TrimSpace(s.rules.GetSalaryCap()); raw != "" {
		capAmount, err := domain.ParseMoneyMillions(raw)
		if err != nil {
			return meta, fmt.Errorf("snapshot: salary cap: %w", err)
		}
		meta.SalaryCap = &capAmount
	}
	return meta, nil
}

func provenance(asOf, note string, present bool) Provenance {
	f := domain.Freshness{State: domain.FreshLive, FetchedAt: asOf, Note: note}
	if !present {
		f.State = domain.FreshFail
	}
	return Provenance{Source: "mfl-mirror", Freshness: f, Kind: "live"}
}

func Build(ctx context.Context, src Source, dir Directory) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: build: %w", err)
	}
	meta, err := src.Metadata()
	if err != nil {
		return Snapshot{}, fmt.Errorf("snapshot: metadata: %w", err)
	}
	ids := src.Franchises()
	for id := range meta.Names {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	out := emptySnapshot(meta, dir)
	if meta.Season > 0 {
		out.League.Value = League{Season: meta.Season, FranchiseCount: len(ids), SalaryCap: meta.SalaryCap}
	}
	seen := map[string]bool{}
	for _, fid := range ids {
		if len(meta.Names) > 0 {
			out.Franchises.Value = append(out.Franchises.Value, franchise(src, meta, fid))
		}
		if meta.MirrorAsOf == "" {
			continue
		}
		roster, err := buildRoster(src, meta.Season, fid)
		if err != nil {
			return Snapshot{}, err
		}
		out.Rosters.Value = append(out.Rosters.Value, roster)
		for _, p := range roster.Players {
			if seen[p.ID.String()] {
				return Snapshot{}, fmt.Errorf("snapshot: duplicate player %s", p.ID.String())
			}
			seen[p.ID.String()] = true
			if out.Players.Provenance.Freshness.State != domain.FreshFail {
				out.Players.Value = append(out.Players.Value, directoryPlayer(p.ID, dir.Lookup))
			}
		}
	}
	finishPlayers(&out, len(seen))
	return out, nil
}

func emptySnapshot(meta Metadata, dir Directory) Snapshot {
	pp := dir.Provenance
	if pp.Source == "" {
		pp = provenance("", "player directory unavailable", false)
		pp.Source = "mfl-players"
	}
	if meta.MirrorAsOf == "" {
		pp.Freshness.State = domain.FreshFail
		pp.Freshness.Note = "mirror unavailable"
	}
	return Snapshot{
		League:     Sourced[League]{Provenance: provenance(meta.MirrorAsOf, "stored mirror; league identity not held", meta.Season > 0)},
		Franchises: Sourced[[]Franchise]{Value: []Franchise{}, Provenance: provenance(meta.MirrorAsOf, "stored rulebook and mirror", len(meta.Names) > 0)},
		Rosters:    Sourced[[]Roster]{Value: []Roster{}, Provenance: provenance(meta.MirrorAsOf, "stored mirror", meta.MirrorAsOf != "")},
		Players:    Sourced[[]Player]{Value: []Player{}, Provenance: pp},
	}
}

func directoryPlayer(id playerid.PlayerID, lookup normalize.Lookup) Player {
	p := Player{ID: id}
	if f, ok := lookup.Facts(id.String()); ok {
		p.Name, p.Position, p.Team, p.IsRookie = f.Name, f.Position, f.Team, &f.IsRookie
		if f.HasBirthdate {
			p.Birthdate = &f.Birthdate
		}
	}
	return p
}

func finishPlayers(out *Snapshot, rostered int) {
	slices.SortFunc(out.Players.Value, func(a, b Player) int { return strings.Compare(a.ID.String(), b.ID.String()) })
	unnamed := 0
	for _, p := range out.Players.Value {
		if p.Name == "" {
			unnamed++
		}
	}
	if out.Players.Provenance.Freshness.State == domain.FreshFail {
		unnamed = rostered
	}
	out.Players.Provenance.Freshness.Note += fmt.Sprintf("; %d unnamed rostered players", unnamed)
}

func franchise(src Source, meta Metadata, fid string) Franchise {
	f := Franchise{ID: fid, Name: meta.Names[fid]}
	if used, ok := src.CapUsed(fid); ok {
		f.CapUsed = &used
		if meta.SalaryCap != nil {
			room := *meta.SalaryCap - used
			f.CapRoom = &room
		}
	}
	return f
}

func buildRoster(src Source, season int, fid string) (Roster, error) {
	out := Roster{FranchiseID: fid, Players: []RosterPlayer{}}
	players, _ := src.Roster(fid)
	for _, p := range players {
		id, err := playerid.New(p.MFLID)
		if err != nil {
			return out, fmt.Errorf("snapshot: franchise %s player id: %w", fid, err)
		}
		var years *int
		if season > 0 && p.ExpirationYear > 0 {
			n := max(0, p.ExpirationYear-season+1)
			years = &n
		}
		out.Players = append(out.Players, RosterPlayer{ID: id, Salary: p.Salary,
			YearsRemaining: years, ContractStatus: p.ContractStatus, RosterStatus: p.RosterStatus})
	}
	slices.SortFunc(out.Players, func(a, b RosterPlayer) int { return strings.Compare(a.ID.String(), b.ID.String()) })
	return out, nil
}
