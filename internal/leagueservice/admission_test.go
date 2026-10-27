package leagueservice

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type admissionFunc func(context.Context, commissioner.Access) error

func (f admissionFunc) Admit(ctx context.Context, access commissioner.Access) error {
	return f(ctx, access)
}

// Unimplemented data methods panic if a refused operation reaches the host.
type admissionHost struct{ Host }

func (admissionHost) Ready() error                           { return nil }
func (admissionHost) CommissionerPrincipal() (string, error) { return "creator", nil }
func (admissionHost) CommissionerContext() context.Context   { return context.Background() }
func (admissionHost) CommissionerNow() time.Time {
	return time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
}

func TestHostedReadsAndPreviewsRequireCredential(t *testing.T) {
	body := `{"league":"native.test","command":"commish.correction.revert","args":{"entry":"act.other"},"reason":"Correct a recorded change","request":"preview","expected":1,"effective":"now","visibility":"league"}`
	raw, err := commissioner.Decode(body)
	if err != nil {
		t.Fatal(err)
	}
	at := (admissionHost{}).CommissionerNow()
	for _, gate := range []commissioner.CredentialAdmission{
		nil,
		admissionFunc(func(_ context.Context, access commissioner.Access) error {
			if access.Principal != "creator" || (access.League != "native.test" && access.League != "") {
				return errors.New("wrong scope")
			}
			return errors.New("credential refused")
		}),
	} {
		service := NewHosted(admissionHost{}, gate)
		for _, tc := range []struct {
			name string
			args []any
		}{
			{"TargetCommissionerLedger", []any{"native.test"}},
			{"Location", []any{"native.test"}},
			{"TargetCommissionerLedgerFiltered", []any{"native.test", "{}"}},
			{"TargetCommissionerPeople", []any{"native.test"}},
			{"TargetCommissionerLeagues", nil},
			{"TargetCommissionerRules", []any{"native.test"}},
			{"TargetCommissionerJobs", []any{"native.test"}},
			{"TargetCommissionerNotices", []any{"native.test"}},
			{"TargetCommissionerDigests", []any{"native.test"}},
			{"TargetCorrections", []any{"native.test"}},
			{"TargetCommissionerMarket", []any{"native.test", commissioner.BidPlace}},
			{"TargetCommissionerActor", nil},
			{"TargetCommissionerControls", nil},
			{"TargetCommissionerSchedule", []any{"native.test"}},
			{"TargetLeagueCalendar", []any{RawLeagueCalendarScope{League: "native.test", Season: 2026, LocalZone: "UTC", Franchise: "franchise.01", Action: "lineup"}}},
			{"TargetLeagueSchedule", []any{RawLeagueScheduleScope{League: "native.test"}}},
			{"TargetNativeCapCheck", []any{"native.test", "UTC"}},
			{"TargetNativeRosterCheck", []any{"native.test"}},
			{"TargetNativeReserveCheck", []any{"native.test"}},
			{"TargetNativeLineupCheck", []any{commissioner.RawNativeLineup{League: "native.test", Franchise: "franchise.01"}}},
			{"TargetNativeHouseRules", []any{"native.test"}},
			{"TargetNativePlayerScore", []any{"native.test", "0099", 2026, 1}},
			{"TargetNativeContractQuote", []any{market.RawContractQuote{League: "native.test", Player: "0099", Action: "tag", ViewerZone: "UTC"}}},
			{"TargetNativeResults", []any{"native.test", 2026, 1}},
			{"TargetNativeStandings", []any{"native.test", 2026}},
			{"TargetNativeBracket", []any{"native.test", 2026}},
			{"TargetStatQueue", []any{"native.test", 2026}},
			{"TargetNativeScorePreview", []any{body}},
			{"TargetNativeTrades", []any{"native.test", "franchise.01"}},
			{"TargetNativeWaivers", []any{"native.test", "franchise.01", at}},
			{"TargetNativeBids", []any{"native.test", "franchise.01"}},
			{"TargetNativeDraft", []any{commissioner.RawNativeDraftScope{League: "native.test", Season: 2026, Franchise: "franchise.01", At: at}}},
			{"TargetNativeLineup", []any{"native.test", "franchise.01", 1, ""}},
			{"TargetNativeDirectory", []any{"native.test"}},
			{"TargetNativeAssets", []any{"native.test"}},
			{"TargetNativeFranchiseSchedule", []any{RawNativeScheduleScope{League: "native.test", Season: 2026, Franchise: "franchise.01", LocalZone: "UTC"}}},
			{"TargetNativeWeek", []any{RawNativeWeekScope{RawNativeScheduleScope: RawNativeScheduleScope{League: "native.test", Season: 2026, Franchise: "franchise.01", LocalZone: "UTC"}, At: at}}},
			{"Preview", []any{raw}},
			{"Execute", []any{t.Context(), raw}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				args := make([]reflect.Value, len(tc.args))
				for i, arg := range tc.args {
					args[i] = reflect.ValueOf(arg)
				}
				result := reflect.ValueOf(service).MethodByName(tc.name).Call(args)
				err, _ := result[len(result)-1].Interface().(error)
				if err == nil || (!strings.Contains(err.Error(), "session is unavailable") && !strings.Contains(err.Error(), "credential refused")) {
					t.Fatalf("missing credential denial: %v", err)
				}
			})
		}
	}
}

func TestHostedReadScopeIncludesFranchiseAndReferencedPlayer(t *testing.T) {
	for _, operation := range []string{"bids", "trades", "quote", "score", "market", "week"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			gate := admissionFunc(func(_ context.Context, access commissioner.Access) error {
				calls++
				switch operation {
				case "bids", "trades", "week":
					if access.Franchise != "franchise.other" {
						return errors.New("missing franchise")
					}
				case "quote", "score":
					if access.Player != "0099" {
						return errors.New("missing player")
					}
				case "market":
					if access.SubjectCommand != commissioner.BidsOverride {
						return errors.New("missing requested command")
					}
				}
				return errors.New("cross-franchise reference")
			})
			service := NewHosted(admissionHost{}, gate)
			var err error
			switch operation {
			case "bids":
				_, err = service.TargetNativeBids("native.test", "franchise.other")
			case "trades":
				_, err = service.TargetNativeTrades("native.test", "franchise.other")
			case "quote":
				_, err = service.TargetNativeContractQuote(market.RawContractQuote{League: "native.test", Player: "0099", Action: "tag", ViewerZone: "UTC"})
			case "score":
				_, err = service.TargetNativePlayerScore("native.test", "0099", 2026, 1)
			case "market":
				_, err = service.TargetCommissionerMarket("native.test", commissioner.BidsOverride)
			case "week":
				_, err = service.TargetNativeWeek(RawNativeWeekScope{RawNativeScheduleScope: RawNativeScheduleScope{League: "native.test", Season: 2026, Franchise: "franchise.other", LocalZone: "UTC"}, At: (admissionHost{}).CommissionerNow()})
			}
			if err == nil || !strings.Contains(err.Error(), "cross-franchise reference") || calls != 1 {
				t.Fatalf("read scope: %v calls=%d", err, calls)
			}
		})
	}
}

type admissionStoreHost struct {
	admissionHost
	store *history.Store
}

func (h admissionStoreHost) ExecuteCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, facts nativecheck.Facts, zone *time.Location) (commissioner.Receipt, error) {
	return h.store.ExecuteCommissionerWithFacts(ctx, raw, principal, auth, facts, zone)
}
func (h admissionStoreHost) PreviewCommissionerWithFacts(ctx context.Context, raw commissioner.RawCommand,
	principal string, auth commissioner.Authorizer, catalogue commissioner.PreviewCatalogue, facts nativecheck.Facts,
	zone *time.Location) (commissioner.Preview, error) {
	return h.store.PreviewCommissionerWithFacts(ctx, raw, principal, auth, catalogue, facts, zone)
}
func (admissionStoreHost) CommissionerAuthorization() commissioner.Authorizer { return nil }
func (admissionStoreHost) NativeFacts() nativecheck.Facts                     { return nil }
func (admissionStoreHost) CommissionerZone() *time.Location                   { return time.UTC }
func (admissionStoreHost) CommissionerPreviewRegistry() string {
	return "id,name,gm_question,governs\n"
}

func TestHostedServiceCarriesAdmissionIntoStoreTransaction(t *testing.T) {
	pools, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "history.db"), history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pools.Close() })
	store := history.New(pools, &measures.Registry{})
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	raw, err := commissioner.Decode(fixtureCommand)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	denyAt := 2
	gate := admissionFunc(func(_ context.Context, access commissioner.Access) error {
		calls++
		if access.Principal != "creator" || access.Raw.Command != commissioner.Create {
			return errors.New("wrong credential input")
		}
		if calls == denyAt {
			return errors.New("revoked after transport admission")
		}
		return nil
	})
	service := NewHosted(admissionStoreHost{store: store}, gate)
	if _, err := service.Execute(t.Context(), raw); err == nil || !strings.Contains(strings.ToLower(err.Error()), "revoked after transport admission") || calls != 2 {
		t.Fatalf("write recheck: %v calls=%d", err, calls)
	}
	events, err := store.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 0 {
		t.Fatalf("denied write appended: %d %v", len(events), err)
	}
	calls = 0
	preview, err := service.Preview(raw)
	if err != nil || len(preview.Blockers) != 1 || len(preview.Fields) != 0 || calls != 2 {
		t.Fatalf("preview recheck: %+v %v calls=%d", preview, err, calls)
	}
	calls, denyAt = 0, 0
	receipt, err := service.Execute(t.Context(), raw)
	if err != nil || calls != 3 {
		t.Fatalf("admitted append: %+v %v calls=%d", receipt, err, calls)
	}
	calls = 0
	retry, err := service.Execute(t.Context(), raw)
	if err != nil || receipt != retry || calls != 2 {
		t.Fatalf("admitted retry: %+v %v calls=%d", retry, err, calls)
	}
	calls, denyAt = 0, 2
	if _, err := service.Execute(t.Context(), raw); err == nil || !strings.Contains(strings.ToLower(err.Error()), "revoked after transport admission") {
		t.Fatalf("retry returned revoked receipt: %v", err)
	}
	events, err = store.Events(t.Context(), raw.League, commissioner.Stream, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("retry appended: %d %v", len(events), err)
	}
}

func TestLocationUsesSharedOwnerLabel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.db")
	pools, err := db.Open(t.Context(), path, history.SchemaVersion())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pools.Close() }()
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	store := history.New(pools, registry)
	if err := store.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	raw := commissioner.RawCommand{League: "native.location", Command: commissioner.Create, Args: commissioner.RawArgs{Name: "Location", Preset: commissioner.PresetEmpty, Season: 2026, FranchiseCount: 2}, Request: "create", Reason: "Create league", Effective: "now", Visibility: "league"}
	if _, err := store.ExecuteCommissioner(t.Context(), raw, "creator", nil); err != nil {
		t.Fatal(err)
	}
	host := StoredHost{Store: store, Context: func() context.Context { return t.Context() }, Principal: "creator"}
	service := New(host)
	want, err := store.TransferStatus(t.Context(), raw.League)
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.Location(raw.League)
	if err != nil || got != want || !got.Writable {
		t.Fatal(got, want, err)
	}
	if got, err := service.Location("native.missing"); err == nil || got.Writable {
		t.Fatal("missing league claimed authority", got, err)
	}
}
