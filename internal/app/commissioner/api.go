package commissioner

import (
	"context"
	"encoding/json"
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
