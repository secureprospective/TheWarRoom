package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawNativeSeedRequest struct {
	League   string `json:"league"`
	Expected int64  `json:"expected"`
	Request  string `json:"request"`
	Reason   string `json:"reason"`
}

func (r RawNativeSeedRequest) Validate() error {
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." || strings.TrimSpace(r.League) != r.League ||
		r.Expected < 1 || strings.TrimSpace(r.Request) == "" || strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("choose an app league, its version, a request and a reason for recording its starting snapshot")
	}
	return nil
}

// TargetNativeSeed captures held data only. It cannot accept client-provided rosters or fetch missing inputs.
func (a *App) TargetNativeSeed(input RawNativeSeedRequest) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("starting snapshot startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	raw, err := a.nativeSeedCommand(ctx, input)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	receipt, err := a.history.ExecuteCommissioner(ctx, raw, principal, a.commissionerAuthorizer)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record starting snapshot: %w", err)
	}
	return receipt, nil
}

func (a *App) nativeSeedCommand(ctx context.Context, input RawNativeSeedRequest) (commissioner.RawCommand, error) {
	raw, found, err := a.history.NativeSeedRetry(ctx, input.League, input.Request)
	if err != nil {
		return raw, fmt.Errorf("read starting snapshot request: %w", err)
	}
	if found {
		if raw.Command != commissioner.NativeSeed || raw.Expected != input.Expected || raw.Reason != input.Reason {
			return raw, fmt.Errorf("this request was already used for a different change")
		}
		return raw, nil
	}
	seed, err := a.heldNativeSnapshot()
	if err != nil {
		return raw, err
	}
	return commissioner.RawCommand{League: input.League, Command: commissioner.NativeSeed,
		Args: commissioner.RawArgs{Native: &seed}, Expected: input.Expected, Request: input.Request,
		Reason: input.Reason, Effective: "now", Visibility: "league"}, nil
}

func (a *App) heldNativeSnapshot() (commissioner.RawNativeSnapshot, error) {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	if a.league == nil || a.rulebook == nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("hold the league rosters and settings before seeding")
	}
	snap, at, _ := a.league.ContractsView()
	if snap.Season != a.season {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("the held rosters and draft reports must be from the same season")
	}
	rosterAt, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("read held roster date: %w", err)
	}
	a.seasonMu.Lock()
	results, future := a.seasonDraftResults, a.seasonFuturePicks
	a.seasonMu.Unlock()
	seed := commissioner.RawNativeSnapshot{Season: snap.Season, RosterAt: rosterAt, DraftAt: results.fetchedAt,
		FutureAt: future.fetchedAt, Players: []commissioner.NativePlayer{}, Picks: []commissioner.NativePick{},
		Adjustments: []commissioner.NativeCharge{}}
	config := a.rulebook.ActiveConfig()
	franchises, mapping := nativeFranchises(config.Franchises)
	seed.Franchises, seed.Conferences = franchises, heldConferences(config)
	a.targetSnapshotMu.Lock()
	directory := a.targetSnapshot.Players.Value
	a.targetSnapshotMu.Unlock()
	seedPlayers(&seed, snap, mapping, directory)
	seedPicks(&seed, results.value, future.value, mapping)
	if err := seed.Validate(); err != nil {
		return commissioner.RawNativeSnapshot{}, fmt.Errorf("held starting snapshot: %w", err)
	}
	return seed, nil
}

// heldConferences names the conference of each held division, by the conference's own name.
func heldConferences(config league.RawConfig) map[string]string {
	names := map[string]string{}
	for _, c := range config.Conferences {
		names[c.ID] = cmp.Or(strings.TrimSpace(c.Name), c.ID)
	}
	out := map[string]string{}
	for _, d := range config.Divisions {
		if conference := cmp.Or(names[d.Conference], d.Conference); conference != "" {
			out[d.ID] = conference
		}
	}
	return out
}

func nativeFranchises(held []league.Franchise) ([]commissioner.NativeFranchise, map[string]string) {
	held = slices.Clone(held)
	slices.SortFunc(held, func(a, b league.Franchise) int { return strings.Compare(a.ID, b.ID) })
	out := make([]commissioner.NativeFranchise, 0, len(held))
	mapping := map[string]string{}
	for i, f := range held {
		id := fmt.Sprintf("franchise.%02d", i+1)
		mapping[f.ID] = id
		out = append(out, commissioner.NativeFranchise{ID: id, HeldID: f.ID, Name: f.Name,
			Abbreviation: f.Abbrev, Division: f.Division})
	}
	return out, mapping
}

// seedPlayers keeps each rostered player's name from the held player directory, so a plan can name them.
func seedPlayers(seed *commissioner.RawNativeSnapshot, snap state.MirrorSnapshot, mapping map[string]string,
	directory []snapshot.Player) {
	names := make(map[string]string, len(directory))
	for _, p := range directory {
		names[p.ID.String()] = p.Name
	}
	for _, p := range snap.Players {
		seed.Players = append(seed.Players, commissioner.NativePlayer{Name: names[p.MFLID], ID: p.MFLID,
			Franchise: mapping[p.FranchiseID],
			Status:    p.RosterStatus, Salary: p.Salary, ExpirationYear: p.ContractYear,
			ContractStatus: p.ContractStatus, ContractInfo: p.ContractInfo})
	}
	for _, c := range snap.Adjustments {
		seed.Adjustments = append(seed.Adjustments, commissioner.NativeCharge{ID: c.ID, Franchise: mapping[c.FranchiseID],
			Amount: c.Amount, Description: c.Description})
	}
}

func seedPicks(seed *commissioner.RawNativeSnapshot, results []draftreports.Pick,
	future []draftreports.FuturePick, mapping map[string]string) {
	for _, p := range results {
		pick := commissioner.NativePick{Season: seed.Season, Round: p.Round, Slot: p.Slot,
			Owner: mapping[p.Franchise], Skipped: p.Skipped}
		if p.Player != nil {
			pick.Player = p.Player.String()
		}
		seed.Picks = append(seed.Picks, pick)
	}
	for _, p := range future {
		seed.Picks = append(seed.Picks, commissioner.NativePick{Season: p.Year, Round: p.Round,
			Original: mapping[p.Original], Owner: mapping[p.Owner]})
	}
}

func (a *App) TargetNativeDirectory(leagueID string) (commissioner.NativeDirectory, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native directory startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	directory, err := a.history.NativeDirectory(ctx, leagueID)
	if err != nil {
		return commissioner.NativeDirectory{}, fmt.Errorf("native league directory: %w", err)
	}
	return directory, nil
}

func (a *App) TargetNativeAssets(leagueID string) (commissioner.NativeAssets, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native assets startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	assets, err := a.history.NativeAssets(ctx, leagueID)
	if err != nil {
		return commissioner.NativeAssets{}, fmt.Errorf("native league assets: %w", err)
	}
	return assets, nil
}
