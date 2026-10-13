package main

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

type RawAcquisition struct {
	Mine   string `json:"mine"`
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Cut    string `json:"cut"`
}

func (r RawAcquisition) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("acquisition scope: %w", err)
	}
	if r.Mine == "" || (r.Player != "" && r.Bid != "") {
		return fmt.Errorf("acquisition: choose franchise and one source")
	}
	for _, id := range []string{r.Player, r.Cut} {
		if id == "" {
			continue
		}
		if _, err := playerid.New(id); err != nil {
			return fmt.Errorf("acquisition player: %w", err)
		}
	}
	return nil
}

type AcquisitionChoice struct {
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Name   string `json:"name"`
}
type AcquisitionView struct {
	Incoming []AcquisitionChoice     `json:"incoming"`
	Cuts     []TradeAsset            `json:"cuts"`
	Player   string                  `json:"player"`
	Seasons  []WhatIfFigure          `json:"seasons"`
	Team     []WhatIfFigure          `json:"team"`
	DeadCap  snapshot.ContractAmount `json:"deadCap"`
	CutName  string                  `json:"cutName"`
	Spot     string                  `json:"spot"`
	Reasons  []string                `json:"reasons"`
}

func (a *App) TargetAcquisition(r RawAcquisition) Reading[AcquisitionView] {
	out := AcquisitionView{Incoming: []AcquisitionChoice{}, Cuts: []TradeAsset{}, Seasons: []WhatIfFigure{}, Team: []WhatIfFigure{}, Reasons: []string{}}
	if err := r.Validate(); err != nil {
		return acquisitionReading(out, err)
	}
	if err := a.ready(); err != nil {
		return acquisitionReading(out, err)
	}
	a.whatifMu.Lock()
	defer a.whatifMu.Unlock()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	snap, _, err := a.tradeState(r.Mine)
	if err != nil {
		return acquisitionReading(out, err)
	}
	choices, err := a.acquisitionChoices(ctx, r.Mine, snap)
	if err != nil {
		return acquisitionReading(out, err)
	}
	out.Incoming = choices
	s, current, err := a.whatIfSession(ctx, RawWhatIfRequest{Mine: r.Mine, Action: "read"})
	if err != nil {
		return acquisitionReading(out, err)
	}
	for _, team := range a.tradeBuilderTeams(ctx, snap) {
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
		return acquisitionReading(out, nil)
	}
	p, name, err := a.acquisitionPlayer(ctx, r, snap)
	if err != nil {
		return acquisitionReading(out, err)
	}
	out.Player = name
	out.DeadCap = snapshot.ContractAmountOf(0)
	for year := s.Snapshot.Season; year <= p.ContractYear; year++ {
		out.Seasons = append(out.Seasons, whatIfFigure(fmt.Sprint(year), 0, p.Salary))
	}
	out.Reasons = append(out.Reasons, "Future salaries are flat-filled from held salary and expiration; future season caps are not held.")
	if err := a.acquisitionProjection(ctx, r, s, current, p, &out); err != nil {
		return acquisitionReading(out, err)
	}
	return acquisitionReading(out, nil)
}
func acquisitionReading(v AcquisitionView, err error) Reading[AcquisitionView] {
	return contractReading(v, "", "whatif.db; held contracts or app-local bid terms", "The add takes one active roster spot; incoming trade IR/taxi status is retained. Each paid season's salary is shown in exact cents. Team cap uses pinned what-if with existing cuts, literal MFL adjustments and the incoming contract. A selected cut is priced by the existing coordinator's unclaimed cut engine (35% assumed, or 50% for a known restructure; final year $0). This is a preview only: no roster changed, no cut saved, nothing sent to MFL.", len(v.Seasons), err)
}
func (a *App) acquisitionChoices(ctx context.Context, mine string, snap snapshot.Snapshot) ([]AcquisitionChoice, error) {
	out := []AcquisitionChoice{}
	for _, team := range a.tradeBuilderTeams(ctx, snap) {
		if team.ID == mine {
			continue
		}
		for _, p := range team.Assets {
			if _, err := playerid.New(p.Token); err == nil {
				out = append(out, AcquisitionChoice{Player: p.Token, Name: p.Name + " · " + team.Name})
			}
		}
	}
	receipts, err := a.moves.List(ctx, ingestion.LeagueID, mine)
	if err != nil {
		return nil, fmt.Errorf("acquisition bids: %w", err)
	}
	for _, r := range receipts {
		if r.Spec.Intent != envelope.BidIntent {
			continue
		}
		f, err := a.moves.BidFacts(ctx, r.CorrelationID)
		if err != nil {
			return nil, fmt.Errorf("acquisition bid terms: %w", err)
		}
		out = append(out, AcquisitionChoice{Bid: r.CorrelationID, Name: acquisitionName(snap, f.Player.String()) + " · bid plan"})
	}
	return out, nil
}
func (a *App) acquisitionPlayer(ctx context.Context, r RawAcquisition, snap snapshot.Snapshot) (state.MirrorPlayer, string, error) {
	m, _, _ := a.league.ContractsView()
	if r.Bid != "" {
		e, err := a.moves.Get(ctx, r.Bid)
		if err != nil {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid: %w", err)
		}
		if e.Receipt().Spec.FranchiseID != r.Mine || e.Receipt().Spec.Intent != envelope.BidIntent {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid outside scope")
		}
		f, err := a.moves.BidFacts(ctx, r.Bid)
		if err != nil {
			return state.MirrorPlayer{}, "", fmt.Errorf("acquisition bid facts: %w", err)
		}
		return state.MirrorPlayer{MFLID: f.Player.String(), FranchiseID: r.Mine, RosterStatus: domain.RosterActive, Salary: f.Salary, ContractYear: m.Season + f.Years - 1}, acquisitionName(snap, f.Player.String()), nil
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
func (a *App) acquisitionProjection(ctx context.Context, r RawAcquisition, s state.WhatIfSession, current *state.Store,
	incoming state.MirrorPlayer, out *AcquisitionView) error {
	stamps, err := a.league.Stamps(ctx)
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
	store, err := a.loadWhatIf(ctx, s)
	if err != nil {
		return err
	}
	seed, _ := store.CapUsed(r.Mine)
	if err := acquisitionCut(ctx, r, current, store, incoming.MFLID, out); err != nil {
		return err
	}
	projected, _ := store.CapUsed(r.Mine)
	after := before + projected - seed + state.CapContribution(incoming.Salary, incoming.RosterStatus, s)
	active, correction := acquisitionRoster(store, m, s)
	after += correction
	out.Spot = fmt.Sprintf("One %s roster spot · %d active players after the add (35–48 allowed)", incoming.RosterStatus, active)
	out.Team = append(out.Team, whatIfFigure("Cap used this season", before, after))
	if s.Cap != nil {
		out.Team = append(out.Team, whatIfFigure("Cap room this season", *s.Cap-before, *s.Cap-after))
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
	return nil
}

func acquisitionName(snap snapshot.Snapshot, id string) string {
	for _, p := range snap.Players.Value {
		if p.ID.String() == id && p.Name != "" {
			return p.Name
		}
	}
	return "Name unavailable"
}

func acquisitionCut(ctx context.Context, r RawAcquisition, current, store *state.Store, incomingID string, out *AcquisitionView) error {

	if r.Cut != "" {
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
	}
	return nil
}

func acquisitionRoster(store *state.Store, m state.MirrorSnapshot, s state.WhatIfSession) (int, domain.Money) {
	active := 0
	var correction domain.Money
	for _, p := range m.Players {
		if _, ok := store.Player(p.MFLID); !ok {
			correction += tradeCentsCorrection(p, "", s)
			continue
		}
		if p.RosterStatus == domain.RosterActive {
			active++
		}
	}
	return active, correction
}
