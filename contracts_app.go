package main

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type ContractsReadings struct {
	Table       Reading[snapshot.ContractTable]     `json:"table"`
	Cap         Reading[snapshot.CapSummary]        `json:"cap"`
	Adjustments Reading[snapshot.SalaryAdjustments] `json:"adjustments"`
}

type contractsInput struct {
	directory   snapshot.Provenance
	mirror      state.MirrorSnapshot
	asOf        string
	used        map[string]domain.Money
	rows        []snapshot.ContractRow
	names       map[string]string
	cap         *domain.Money
	capOverride bool
}

func (a *App) contractsInput() (contractsInput, error) {
	if err := a.ready(); err != nil {
		return contractsInput{}, fmt.Errorf("contracts: startup: %w", err)
	}
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	m, at, used := a.core.League.ContractsView()
	if at == "" || m.Season == 0 {
		return contractsInput{}, fmt.Errorf("contracts: MFL mirror unavailable; refresh from MFL")
	}
	a.core.TargetSnapshotMu.Lock()
	directory := a.core.TargetSnapshot
	a.core.TargetSnapshotMu.Unlock()
	names := a.core.Rulebook.FranchiseNames()
	var capAmount *domain.Money
	if raw := a.core.Rulebook.GetSalaryCap(); raw != "" {
		parsed, err := domain.ParseMoneyMillions(raw)
		if err != nil {
			return contractsInput{}, fmt.Errorf("contracts: salary cap configuration: %w", err)
		}
		capAmount = &parsed
	}
	return contractsInput{directory: directory.Players.Provenance, mirror: m, asOf: at, used: used, rows: snapshot.ContractRows(m, directory, names), names: names, cap: capAmount, capOverride: a.core.Rulebook.SalaryCapOverridden()}, nil
}

func contractReading[T any](value T, at, source, how string, count int, err error) Reading[T] {
	r := Reading[T]{Sourced: snapshot.Sourced[T]{Value: value, Provenance: snapshot.Provenance{Source: source, Kind: "live", Freshness: domain.Freshness{State: domain.FreshLive, FetchedAt: at, Note: "Held MFL mirror; no network or what-if state"}}}, Status: ReadingOK, Count: count, Threshold: 0, Baseline: "Current MFL mirror revision", Cutoff: at, Method: "Integer cents; Go-authored display", BuildStamp: buildLabel() + " " + buildDate, HowCounted: how}
	if err != nil {
		reason := err.Error()
		r.Status, r.AbsenceReason = ReadingFailed, &reason
		r.Provenance.Freshness.State = domain.FreshFail
	}
	return r
}

func failedContracts(err error) ContractsReadings {
	return ContractsReadings{
		Table:       contractReading(snapshot.ContractTable{}, "", "MFL rosters", "League contracts from MFL annual salary and expiration.", 0, err),
		Cap:         contractReading(snapshot.CapSummary{}, "", "MFL mirror + active league configuration", "Mirror cap usage includes configured reserve discounts and signed MFL salary adjustments.", 0, err),
		Adjustments: contractReading(snapshot.SalaryAdjustments{Rows: []snapshot.SalaryAdjustment{}}, "", "MFL salaryAdjustments", "MFL's salary adjustments, as MFL recorded them (dead cap from drops, retirements and commissioner changes).", 0, err),
	}
}

func (a *App) TargetContracts(raw snapshot.RawContractScope) ContractsReadings {
	if err := raw.Validate(); err != nil {
		return failedContracts(err)
	}
	input, err := a.contractsInput()
	if err != nil {
		return failedContracts(err)
	}
	table := snapshot.ContractsTable(input.rows, input.names, raw)
	table.DirectorySource = input.directory.Source
	table.DirectoryAsOf = input.directory.Freshness.FetchedAt
	out := ContractsReadings{
		Table:       contractReading(table, input.asOf, "MFL rosters: salary, contractYear, contractStatus, contractInfo; held players directory", "Every rostered MFL contract, including IR and taxi. Go filters by franchise and position and sorts within the my-team-first group. Names and positions come from the held player directory; unknown names show the player ID. No calculated dead cap.", len(table.Rows), nil),
		Adjustments: contractReading(snapshot.SalaryAdjustments{Rows: snapshot.ContractsAdjustments(input.mirror.Adjustments, input.names, raw.Mine)}, input.asOf, "MFL salaryAdjustments", "Signed amounts and descriptions exactly as MFL states them. These are MFL's recorded salary adjustments (drops, retirements and commissioner changes); credits stay negative. No description is interpreted as a computed charge.", len(input.mirror.Adjustments), nil),
	}
	used, ok := input.used[raw.Mine]
	var capErr error
	if !ok {
		capErr = fmt.Errorf("cap summary: select a franchise held in the MFL mirror")
	}
	out.Cap = contractReading(snapshot.ContractsCap(input.rows, raw.Mine, input.mirror.Season, used, input.cap, input.names), input.asOf, "MFL mirror + active league configuration", "Cap comes from active league configuration, not a hardcoded rulebook value. Used is the mirror's current cap total: active salary, configured IR/taxi contribution (snapped to $10k), and signed MFL adjustments. Room = cap minus used, in cents. Future commitments carry today’s annual salary through contract expiration; MFL does not report those years separately. No future discounts, adjustments or dead-cap charges are projected.", 1, capErr)
	if input.capOverride && out.Cap.Value.Cap != nil {
		out.Cap.Value.CapSource = "League salary cap: " + out.Cap.Value.Cap.Text + " (set by the commissioner)"
	}
	return out
}

func (a *App) TargetPlayerContract(rawID string) Reading[snapshot.PlayerContract] {
	id, err := playerid.New(rawID)
	if err != nil {
		return contractReading(snapshot.PlayerContract{}, "", "MFL rosters", "", 0, fmt.Errorf("player contract: ID: %w", err))
	}
	input, err := a.contractsInput()
	if err != nil {
		return contractReading(snapshot.PlayerContract{}, "", "MFL rosters", "", 0, err)
	}
	for _, row := range input.rows {
		if row.ID == id.String() {
			value := snapshot.PlayerContract{Player: row, Years: snapshot.ContractYears(row, input.mirror.Season)}
			return contractReading(value, input.asOf, "MFL rosters: annual salary + expiration", "The current season's annual salary is observed from MFL. Future years carry that annual salary through expiration and may change after an extension or restructure; MFL does not report those years separately. Missing or expired expiration yields only the current salary, not a guessed term. No UFA cost or dead-cap figure is computed.", len(value.Years), nil)
		}
	}
	return contractReading(snapshot.PlayerContract{}, input.asOf, "MFL rosters", "", 0, fmt.Errorf("player contract: player %s is not in the mirror", id.String()))
}
