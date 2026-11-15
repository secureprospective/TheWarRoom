package commissioner

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	domain "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type releaseDenial struct{ calls int }

func (d *releaseDenial) Admit(context.Context, domain.Access) error {
	d.calls++
	return errors.New("release matrix credential revoked")
}

func TestReleaseMatrixCatalogReadAdmission(t *testing.T) {
	t.Parallel()
	app := previewApp(t)
	executeCommissionerTest(t, app, domain.RawCommand{League: "native.other", Command: domain.Create,
		Args:    domain.RawArgs{Name: "Second matrix league", Preset: domain.PresetEmpty, Season: 2026, FranchiseCount: 2},
		Request: "create.other", Reason: "Create a separate league", Effective: "now", Visibility: "league"})
	host := serviceParityHost{Host: app.host}
	otherBefore, err := app.history.Events(t.Context(), "native.other", domain.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	before, err := app.history.Events(t.Context(), "native.preview", domain.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range NativeCommands() {
		if !slices.Contains(command.Modes, catalog.Read) {
			continue
		}
		t.Run(command.ID, func(t *testing.T) {
			for _, league := range []string{"native.preview", "native.other"} {
				for _, revoked := range []bool{false, true} {
					gate := &releaseDenial{}
					var admission domain.CredentialAdmission
					if revoked {
						admission = gate
					}
					service := leagueservice.NewHosted(host, admission)
					method := reflect.ValueOf(service).MethodByName(command.Method)
					args := releaseReadInputs(t, command.Method, method.Type(), league)
					results := method.Call(args)
					failure := results[len(results)-1]
					if failure.IsNil() {
						t.Fatalf("%s revoked=%v: read admitted", league, revoked)
					}
					want := "Your session is unavailable"
					if revoked {
						want = "Release matrix credential revoked"
					}
					var refusal *domain.Refusal
					got := failure.Interface().(error)
					if !errors.As(got, &refusal) || refusal.Reason() != want {
						t.Fatalf("%s revoked=%v: did not reach credential admission: %v", league, revoked, got)
					}
					if revoked && gate.calls != 1 {
						t.Fatalf("admission calls=%d", gate.calls)
					}
				}
			}
		})
	}
	otherAfter, err := app.history.Events(t.Context(), "native.other", domain.Stream, 0)
	if err != nil || !reflect.DeepEqual(otherBefore, otherAfter) {
		t.Fatalf("denied reads changed other history: %v", err)
	}
	after, err := app.history.Events(t.Context(), "native.preview", domain.Stream, 0)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("denied reads changed history: %v", err)
	}
}

func releaseReadInputs(t *testing.T, name string, typ reflect.Type, league string) []reflect.Value {
	t.Helper()
	args := parityInputs(t, typ, true)
	if len(args) > 0 && args[0].Kind() == reflect.String {
		args[0].SetString(league)
	}
	switch name {
	case "TargetCommissionerLedgerFiltered":
		args[1].SetString("{}")
	case "TargetCommissionerMarket":
		args[1].SetString(domain.BidPlace)
	case "TargetNativeCapCheck":
		args[1].SetString("UTC")
	case "TargetNativePlayerScore":
		args[1].SetString("0099")
		args[2].SetInt(2026)
		args[3].SetInt(1)
	case "TargetNativeResults":
		args[1].SetInt(2026)
		args[2].SetInt(1)
	case "TargetNativeStandings", "TargetNativeOdds", "TargetNativeBracket", "TargetStatQueue":
		args[1].SetInt(2026)
	case "TargetNativeLineup":
		args[1].SetString("franchise.01")
		args[2].SetInt(1)
	case "TargetNativeMoves", "TargetNativeBids", "TargetNativeTrades", "TargetNativeWaivers":
		args[1].SetString("franchise.01")
	}
	if len(args) > 0 && args[0].Kind() == reflect.Struct {
		value := args[0]
		if f := value.FieldByName("League"); f.IsValid() {
			f.SetString(league)
		}
		for _, field := range []string{"Franchise", "ViewerZone", "LocalZone", "Player", "Action"} {
			f := value.FieldByName(field)
			if !f.IsValid() {
				continue
			}
			f.SetString(map[string]string{"Franchise": "franchise.01", "ViewerZone": "UTC", "LocalZone": "UTC", "Player": "0099", "Action": "lineup"}[field])
		}
		if name == "TargetNativeContractQuote" {
			value.FieldByName("Action").SetString("tag")
		}
		if f := value.FieldByName("At"); f.IsValid() {
			f.Set(reflect.ValueOf(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)))
		}
		if f := value.FieldByName("Season"); f.IsValid() {
			f.SetInt(2026)
		}
	}
	return args
}
