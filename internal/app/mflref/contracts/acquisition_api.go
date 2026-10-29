package contracts

import (
	"context"
	"fmt"
	"slices"
	"time"

	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// AcquisitionHost reads the trade desk's held state for the acquisition preview. The composition root
// implements it over the trades feature, which this feature never imports.
type AcquisitionHost interface {
	AcquisitionSnapshot(mine string) (snapshot.Snapshot, error)
	AcquisitionTeams(ctx context.Context, snap snapshot.Snapshot) []AcquisitionTeam
	// AcquisitionCorrection is the trade desk's cap correction for a player the projection dropped.
	AcquisitionCorrection(p state.MirrorPlayer, s mflwhatif.WhatIfSession) domain.Money
}

// AcquisitionTeam is one franchise's tradeable assets, as the trade desk builds them.
type AcquisitionTeam struct {
	ID     string
	Name   string
	Assets []AcquisitionAsset
}

// AcquisitionCell is the wire twin of the trade desk's value cell.
type AcquisitionCell struct {
	Text   string `json:"text"`
	Sub    string `json:"sub"`
	Detail string `json:"detail"`
	Held   bool   `json:"held"`
}

// AcquisitionAsset is the wire twin of the trade desk's asset: the same fields in the same order.
type AcquisitionAsset struct {
	Team     string           `json:"team,omitempty"`
	Contract string           `json:"contract,omitempty"`
	Now      *AcquisitionCell `json:"now,omitempty"`
	Dynasty  *AcquisitionCell `json:"dynasty,omitempty"`
	Token    string           `json:"token"`
	Name     string           `json:"name"`
	Position domain.Position  `json:"position,omitempty"`
}

type RawAcquisition struct {
	Mine   string `json:"mine"`
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Cut    string `json:"cut"`
}

func (r RawAcquisition) Validate() error { return Acquisition(r).Validate() }

type AcquisitionChoice struct {
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Name   string `json:"name"`
}
type AcquisitionView struct {
	Incoming []AcquisitionChoice     `json:"incoming"`
	Cuts     []AcquisitionAsset      `json:"cuts"`
	Player   string                  `json:"player"`
	Seasons  []WhatIfFigure          `json:"seasons"`
	Team     []WhatIfFigure          `json:"team"`
	DeadCap  snapshot.ContractAmount `json:"deadCap"`
	CutName  string                  `json:"cutName"`
	Spot     string                  `json:"spot"`
	Reasons  []string                `json:"reasons"`
}

type AcquisitionReading report.Reading[AcquisitionView]

// TargetAcquisition previews adding one player, from a trade or a bid plan, to the what-if roster.
func (a *API) TargetAcquisition(r RawAcquisition) AcquisitionReading {
	out := AcquisitionView{Incoming: []AcquisitionChoice{}, Cuts: []AcquisitionAsset{}, Seasons: []WhatIfFigure{},
		Team: []WhatIfFigure{}, Reasons: []string{}}
	if err := r.Validate(); err != nil {
		return a.acquisitionReading(out, err)
	}
	if err := a.host.Ready(); err != nil {
		return a.acquisitionReading(out, err)
	}
	if err := a.host.RequireOwnFranchise(r.Mine); err != nil {
		return a.acquisitionReading(out, err)
	}
	var reading AcquisitionReading
	a.host.SerializeWhatIf(func() {
		ctx, cancel := context.WithTimeout(a.host.ContractsContext(), timeout)
		defer cancel()
		err := a.acquisition(ctx, r, &out)
		reading = a.acquisitionReading(out, err)
	})
	return reading
}

// acquisition fills out under the what-if and refresh serializers.
func (a *API) acquisition(ctx context.Context, r RawAcquisition, out *AcquisitionView) error {
	snap, err := a.host.AcquisitionSnapshot(r.Mine)
	if err != nil {
		return err //nolint:wrapcheck // the host names the trade-state step, as the root preview did
	}
	choices, err := a.acquisitionChoices(ctx, r.Mine, snap)
	if err != nil {
		return err
	}
	out.Incoming = choices
	st := a.stores()
	s, current, err := OpenWhatIf(ctx, st, RawWhatIfRequest{Mine: r.Mine, Action: "read"})
	if err != nil {
		return err
	}
	for _, team := range a.host.AcquisitionTeams(ctx, snap) {
		if team.ID != r.Mine {
			continue
		}
		for _, asset := range team.Assets {
			if _, ok := current.Player(asset.Token); ok {
				out.Cuts = append(out.Cuts, asset)
			}
		}
	}
	if r.Player == "" && r.Bid == "" {
		return nil
	}
	p, name, err := a.acquisitionPlayer(ctx, r, snap)
	if err != nil {
		return err
	}
	out.Player = name
	out.DeadCap = snapshot.ContractAmountOf(0)
	for year := s.Snapshot.Season; year <= p.ContractYear; year++ {
		out.Seasons = append(out.Seasons, Figure(fmt.Sprint(year), 0, p.Salary))
	}
	out.Reasons = append(out.Reasons, "Later seasons assume today's salary through expiration; MFL does not publish future salary caps.")
	return a.acquisitionProjection(ctx, r, s, current, p, out)
}

func (a *API) acquisitionReading(v AcquisitionView, err error) AcquisitionReading {
	return AcquisitionReading(contractReading(a, v, "", "whatif.db; held contracts or app-local bid terms", "The add takes one active roster spot; incoming trade IR/taxi status is retained. Each paid season's salary is shown in exact cents. Team cap uses pinned what-if with existing cuts, literal MFL adjustments and the incoming contract. A selected cut is priced by the existing coordinator's unclaimed cut engine (35% assumed, or 50% for a known restructure; final year $0). This is a preview only: no roster changed, no cut saved, nothing sent to MFL.", len(v.Seasons), err))
}

func (a *API) acquisitionChoices(ctx context.Context, mine string, snap snapshot.Snapshot) ([]AcquisitionChoice, error) {
	out := []AcquisitionChoice{}
	for _, team := range a.host.AcquisitionTeams(ctx, snap) {
		if team.ID == mine {
			continue
		}
		for _, p := range team.Assets {
			if _, err := playerid.New(p.Token); err == nil {
				out = append(out, AcquisitionChoice{Player: p.Token, Name: p.Name + " · " + team.Name})
			}
		}
	}
	st := a.stores()
	receipts, err := st.Moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return nil, fmt.Errorf("acquisition bids: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent != envelope.BidIntent {
			continue
		}
		f, err := st.Moves.BidFacts(ctx, r.CorrelationID)
		if err != nil {
			return nil, fmt.Errorf("acquisition bid terms: %w", err)
		}
		out = append(out, AcquisitionChoice{Bid: r.CorrelationID, Name: Name(snap, f.Player.String()) + " · bid plan"})
	}
	return out, nil
}

func (a *API) acquisitionPlayer(ctx context.Context, r RawAcquisition, snap snapshot.Snapshot) (state.MirrorPlayer, string, error) {
	st := a.stores()
	m, _, _ := st.League.ContractsView()
	if r.Bid != "" {
		e, err := st.Moves.Get(ctx, r.Bid)
		if err != nil {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid: %w", err)
		}
		if e.Receipt().Spec.FranchiseID != r.Mine || e.Receipt().Spec.Intent != envelope.BidIntent {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid outside scope")
		}
		f, err := st.Moves.BidFacts(ctx, r.Bid)
		if err != nil {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid facts: %w", err)
		}
		return state.MirrorPlayer{MFLID: f.Player.String(), FranchiseID: r.Mine, RosterStatus: domain.RosterActive, Salary: f.Salary, ContractYear: m.Season + f.Years - 1}, Name(snap, f.Player.String()), nil
	}
	for _, p := range m.Players {
		if p.MFLID != r.Player || p.FranchiseID == r.Mine {
			continue
		}
		for _, known := range snap.Players.Value {
			if known.ID.String() == r.Player {
				return p, known.Name, nil
			}
		}
	}
	return state.MirrorPlayer{}, "", fmt.Errorf("incoming player's held contract unavailable")
}

func (a *API) acquisitionProjection(ctx context.Context, r RawAcquisition, s mflwhatif.WhatIfSession, current *state.Store,
	incoming state.MirrorPlayer, out *AcquisitionView) error {
	st := a.stores()
	stamps, err := st.League.Stamps(ctx)
	if err != nil {
		return fmt.Errorf("acquisition stamps: %w", err)
	}
	if stamps.NewerThan(s.Stamps) {
		out.Reasons = append(out.Reasons, "MFL changed since this what-if started. Reset first.")
	}
	used, _ := current.CapUsed(r.Mine)
	before := s.Used + used - s.SeedUsed
	m := state.MirrorSnapshot{Season: s.Snapshot.Season}
	for _, p := range s.Snapshot.Players {
		if p.FranchiseID != r.Mine {
			continue
		}
		if _, ok := current.Player(p.MFLID); ok {
			m.Players = append(m.Players, p)
		}
	}
	if slices.ContainsFunc(m.Players, func(p state.MirrorPlayer) bool { return p.MFLID == incoming.MFLID }) {
		return fmt.Errorf("incoming player is already on your what-if roster")
	}
	incoming.FranchiseID = r.Mine
	m.Players = append(m.Players, incoming)
	s.Generation = "acquisition-preview:" + fmt.Sprint(time.Now().UnixNano())
	s.Snapshot = m
	store, err := LoadWhatIf(ctx, st.WhatifPools, s)
	if err != nil {
		return err
	}
	seed, _ := store.CapUsed(r.Mine)
	if err := acquisitionCut(ctx, r, current, store, incoming.MFLID, out); err != nil {
		return err
	}
	projected, _ := store.CapUsed(r.Mine)
	after := before + projected - seed + state.CapContribution(incoming.Salary, incoming.RosterStatus, s)
	active, correction := a.acquisitionRoster(store, m, s)
	after += correction
	out.Spot = fmt.Sprintf("One active roster spot · %d active players after the add (35–48 allowed)", active)
	acquisitionLimits(out, s, before, after, active)
	return nil
}

// acquisitionLimits adds the cap figures and the cap and roster-size reasons.
func acquisitionLimits(out *AcquisitionView, s mflwhatif.WhatIfSession, before, after domain.Money, active int) {
	out.Team = append(out.Team, Figure("Cap used this season", before, after))
	if s.Cap != nil {
		out.Team = append(out.Team, Figure("Cap room this season", *s.Cap-before, *s.Cap-after))
		if after > *s.Cap {
			out.Reasons = append(out.Reasons, "The add does not fit the salary cap; choose a cut and inspect its dead cap.")
		}
	}
	if active > 48 {
		out.Reasons = append(out.Reasons, "The add needs a cut to fit 48 active roster spots.")
	}
	if active < 35 {
		out.Reasons = append(out.Reasons, "The roster is below the 35-player in-season minimum.")
	}
}

func acquisitionCut(ctx context.Context, r RawAcquisition, current, store *state.Store, incomingID string, out *AcquisitionView) error {
	if r.Cut == "" {
		return nil
	}
	if r.Cut == incomingID {
		return fmt.Errorf("select a held player to cut, not the incoming player")
	}
	if _, ok := current.Player(r.Cut); !ok {
		return fmt.Errorf("cut player no longer on your what-if roster")
	}
	coord, err := transactions.New(store.Writer(), nil, nil)
	if err != nil {
		return fmt.Errorf("acquisition cut coordinator: %w", err)
	}
	receipt, err := coord.Execute(ctx, transactions.Waiver{MFLID: r.Cut})
	if err != nil {
		return fmt.Errorf("acquisition cut: %w", err)
	}
	if len(receipt.CapDeltas) == 1 {
		out.DeadCap = snapshot.ContractAmountOf(receipt.CapDeltas[0].Cents)
	}
	for _, cut := range out.Cuts {
		if cut.Token == r.Cut {
			out.CutName = cut.Name
		}
	}
	return nil
}

func (a *API) acquisitionRoster(store *state.Store, m state.MirrorSnapshot, s mflwhatif.WhatIfSession) (int, domain.Money) {
	active := 0
	var correction domain.Money
	for _, p := range m.Players {
		if _, ok := store.Player(p.MFLID); !ok {
			correction += a.host.AcquisitionCorrection(p, s)
			continue
		}
		if p.RosterStatus == domain.RosterActive {
			active++
		}
	}
	return active, correction
}
