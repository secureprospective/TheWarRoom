package commissioner

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueservice"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// Host retains the session, football-fact composition and transactional gate at their existing owners.
type Host interface {
	SessionHost
	ActionHost
	LeagueHost
	ResultsHost
	RosterHost
	MutationHost
	SeatHost
}
type SessionHost = leagueservice.SessionHost
type ActionHost interface {
	RunNativeOwnerAction(string, string, string, json.RawMessage) (commissioner.Receipt, error)
	CaptureNativeSeed() (commissioner.RawNativeSnapshot, error)
	CloseFinishedDrafts(string)
}

type MutationHost interface {
	ExecuteCommissioner(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer) (commissioner.Receipt, error)
	ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer, facts nativecheck.Facts,
		zone *time.Location) (commissioner.Receipt, error)
	PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
		principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue,
		playerFacts nativecheck.Facts, zone *time.Location) (commissioner.Preview, error)
}

type LeagueHost = leagueservice.LeagueHost

type ResultsHost = leagueservice.ResultsHost

type RosterHost = leagueservice.RosterHost

type API struct {
	host  Host
	reads *leagueservice.Service
}

func New(host Host) *API { return &API{host: host, reads: leagueservice.New(host)} }

const refreshTimeout = 120 * time.Second
const draftPickName = "draft.pick"

func Commands() []catalog.Command { return catalog.DesktopCommands("commissioner", NativeCommands()) }

func NativeCommands() []catalog.Command {
	return catalog.NativeCommands(catalog.Shape(reflect.TypeFor[RawNativeSeedRequest](), "RawNativeSeedRequest.Validate/ValidatePreview"))
}

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
