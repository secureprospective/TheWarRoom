package contracts

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

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

func (a *API) contractsInput() (contractsInput, error) {
	if err := a.host.Ready(); err != nil {
		return contractsInput{}, fmt.Errorf("contracts: startup: %w", err)
	}
	var in contractsInput
	var err error
	a.host.SerializeContractsRead(func() { in, err = a.heldContracts() })
	return in, err
}

// heldContracts runs under the refresh serializer, as the root binding's capture did.
func (a *API) heldContracts() (contractsInput, error) {
	st := a.stores()
	m, at, used := st.League.ContractsView()
	if at == "" || m.Season == 0 {
		return contractsInput{}, fmt.Errorf("contracts: MFL mirror unavailable; refresh from MFL")
	}
	directory, _ := a.host.ContractsDirectory()
	names := st.Rulebook.FranchiseNames()
	var capAmount *domain.Money
	if raw := st.Rulebook.GetSalaryCap(); raw != "" {
		parsed, err := domain.ParseMoneyMillions(raw)
		if err != nil {
			return contractsInput{}, fmt.Errorf("contracts: salary cap configuration: %w", err)
		}
		capAmount = &parsed
	}
	return contractsInput{directory: directory.Players.Provenance, mirror: m, asOf: at, used: used, rows: snapshot.ContractRows(m, directory, names), names: names, cap: capAmount, capOverride: st.Rulebook.SalaryCapOverridden()}, nil
}

func (a *API) failedContracts(err error) ContractsReadings {
	return ContractsReadings{
		Table:       contractReading(a, snapshot.ContractTable{}, "", "MFL rosters", "League contracts from MFL annual salary and expiration.", 0, err),
		Cap:         contractReading(a, snapshot.CapSummary{}, "", "MFL mirror + active league configuration", "Mirror cap usage includes configured reserve discounts and signed MFL salary adjustments.", 0, err),
		Adjustments: contractReading(a, snapshot.SalaryAdjustments{Rows: []snapshot.SalaryAdjustment{}}, "", "MFL salaryAdjustments", "MFL's salary adjustments, as MFL recorded them (dead cap from drops, retirements and commissioner changes).", 0, err),
	}
}

func (a *API) TargetContracts(raw snapshot.RawContractScope) ContractsReadings {
	if err := raw.Validate(); err != nil {
		return a.failedContracts(err)
	}
	input, err := a.contractsInput()
	if err != nil {
		return a.failedContracts(err)
	}
	table := snapshot.ContractsTable(input.rows, input.names, raw)
	table.DirectorySource = input.directory.Source
	table.DirectoryAsOf = input.directory.Freshness.FetchedAt
	out := ContractsReadings{
		Table:       contractReading(a, table, input.asOf, "MFL rosters: salary, contractYear, contractStatus, contractInfo; held players directory", "Every rostered MFL contract, including IR and taxi. Go filters by franchise and position and sorts within the my-team-first group. Names and positions come from the held player directory; unknown names show the player ID. No calculated dead cap.", len(table.Rows), nil),
		Adjustments: contractReading(a, snapshot.SalaryAdjustments{Rows: snapshot.ContractsAdjustments(input.mirror.Adjustments, input.names, raw.Mine)}, input.asOf, "MFL salaryAdjustments", "Signed amounts and descriptions exactly as MFL states them. These are MFL's recorded salary adjustments (drops, retirements and commissioner changes); credits stay negative. No description is interpreted as a computed charge.", len(input.mirror.Adjustments), nil),
	}
	used, ok := input.used[raw.Mine]
	var capErr error
	if !ok {
		capErr = fmt.Errorf("cap summary: select a franchise held in the MFL mirror")
	}
	out.Cap = contractReading(a, snapshot.ContractsCap(input.rows, raw.Mine, input.mirror.Season, used, input.cap, input.names), input.asOf, "MFL mirror + active league configuration", "Cap comes from active league configuration, not a hardcoded rulebook value. Used is the mirror's current cap total: active salary, configured IR/taxi contribution (snapped to $10k), and signed MFL adjustments. Room = cap minus used, in cents. Future commitments carry today’s annual salary through contract expiration; MFL does not report those years separately. No future discounts, adjustments or dead-cap charges are projected.", 1, capErr)
	if input.capOverride && out.Cap.Value.Cap != nil {
		out.Cap.Value.CapSource = "League salary cap: " + out.Cap.Value.Cap.Text + " (set by the commissioner)"
	}
	return out
}

func (a *API) TargetPlayerContract(rawID string) PlayerContractReading {
	id, err := playerid.New(rawID)
	if err != nil {
		return a.playerContract(snapshot.PlayerContract{}, "", "MFL rosters", "", 0, fmt.Errorf("player contract: ID: %w", err))
	}
	input, err := a.contractsInput()
	if err != nil {
		return a.playerContract(snapshot.PlayerContract{}, "", "MFL rosters", "", 0, err)
	}
	for _, row := range input.rows {
		if row.ID == id.String() {
			value := snapshot.PlayerContract{Player: row, Years: snapshot.ContractYears(row, input.mirror.Season)}
			return a.playerContract(value, input.asOf, "MFL rosters: annual salary + expiration", "The current season's annual salary is observed from MFL. Future years carry that annual salary through expiration and may change after an extension or restructure; MFL does not report those years separately. Missing or expired expiration yields only the current salary, not a guessed term. No UFA cost or dead-cap figure is computed.", len(value.Years), nil)
		}
	}
	return a.playerContract(snapshot.PlayerContract{}, input.asOf, "MFL rosters", "", 0, fmt.Errorf("player contract: player %s is not in the mirror", id.String()))
}

func (a *API) playerContract(v snapshot.PlayerContract, at, source, how string, count int, err error) PlayerContractReading {
	return PlayerContractReading(contractReading(a, v, at, source, how, count, err))
}
