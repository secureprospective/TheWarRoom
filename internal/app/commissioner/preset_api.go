package commissioner

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TargetCommissionerPresetCompare lists every setting where the league differs from the preset.
func (a *API) TargetCommissionerPresetCompare(league, preset string) (commissioner.PresetComparison, error) {
	//nolint:wrapcheck // The shared service supplies the user-facing context.
	return a.reads.TargetCommissionerPresetCompare(league, preset)
}

// TargetCommissionerPresetApply records the chosen differences as one rules change. With dryRun it only
// returns the rules changes it would record, for the ordinary preview of each.
func (a *API) TargetCommissionerPresetApply(body string) (PresetApplyResult, error) {
	raw, err := commissioner.DecodePresetApply(body)
	if err != nil {
		return PresetApplyResult{}, fmt.Errorf("preset change: %w", err)
	}
	raws, err := a.reads.PresetChanges(raw)
	if err != nil {
		return PresetApplyResult{}, err //nolint:wrapcheck // The shared service supplies the wording.
	}
	if raw.DryRun {
		return PresetApplyResult{Changes: raws, Receipts: []commissioner.Receipt{}}, nil
	}
	receipts, err := a.reads.ExecuteRules(a.host.CommissionerContext(), raws)
	if err != nil {
		return PresetApplyResult{}, err //nolint:wrapcheck // The shared service supplies the wording.
	}
	return PresetApplyResult{Changes: raws, Receipts: receipts}, nil
}

// PresetApplyResult is the rules changes a preset apply sends and, once recorded, their receipts.
type PresetApplyResult struct {
	Changes  []commissioner.RawCommand `json:"changes"`
	Receipts []commissioner.Receipt    `json:"receipts"`
}
