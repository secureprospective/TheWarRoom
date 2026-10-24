package commissioner

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

type serviceParityHost struct{ Host }

func (serviceParityHost) CommissionerNow() time.Time {
	return time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
}
func (serviceParityHost) CommissionerZone() *time.Location { return time.UTC }

func TestWailsServiceReadParity(t *testing.T) {
	app := previewApp(t)
	host := serviceParityHost{Host: app.host}
	desktop := reflect.ValueOf(New(host))
	service := reflect.ValueOf(leagueservice.New(host))
	for _, command := range nativeReads() {
		t.Run(command.ID, func(t *testing.T) {
			left, right := desktop.MethodByName(command.Method), service.MethodByName(command.Method)
			if !left.IsValid() || !right.IsValid() {
				t.Fatalf("missing service adapter: %s", command.Method)
			}
			// Exercise every adapter's empty-input refusal as well as a real league read. Named
			// desktop Raw types deliberately retain their binding identity across the boundary.
			for _, populated := range []bool{false, true} {
				local := left.Call(parityInputs(t, left.Type(), populated))
				shared := right.Call(parityInputs(t, right.Type(), populated))
				if command.Method == "TargetCommissionerControls" {
					// Hosted metadata can refuse a credential; the offline desktop signature stays fixed.
					if len(shared) != 2 || !shared[1].IsNil() {
						t.Fatal("local controls unexpectedly refused")
					}
					shared = shared[:1]
				}
				assertParityResults(t, local, shared)
			}
		})
	}
}

func parityInputs(t *testing.T, method reflect.Type, populated bool) []reflect.Value {
	t.Helper()
	args := make([]reflect.Value, method.NumIn())
	for i := range args {
		value := reflect.New(method.In(i))
		if populated {
			if method.In(i).Kind() == reflect.String {
				if i == 0 {
					value.Elem().SetString("native.preview")
				}
			} else if method.In(i).Kind() == reflect.Struct {
				if method.In(i) == reflect.TypeFor[time.Time]() {
					value.Elem().Set(reflect.ValueOf(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)))
					args[i] = value.Elem()
					continue
				}
				if err := json.Unmarshal([]byte(`{"league":"native.preview"}`), value.Interface()); err != nil {
					t.Fatal(err)
				}
			}
		}
		args[i] = value.Elem()
	}
	return args
}

func assertParityResults(t *testing.T, left, right []reflect.Value) {
	t.Helper()
	if len(left) != len(right) {
		t.Fatal("result arity differs")
	}
	for i := range left {
		if left[i].Type().Implements(reflect.TypeFor[error]()) {
			a, b := "", ""
			if !left[i].IsNil() {
				a = left[i].Interface().(error).Error()
			}
			if !right[i].IsNil() {
				b = right[i].Interface().(error).Error()
			}
			if a != b {
				t.Fatalf("refusal differs: %q != %q", a, b)
			}
		} else if !reflect.DeepEqual(left[i].Interface(), right[i].Interface()) {
			t.Fatalf("reading differs: %+v != %+v", left[i], right[i])
		}
	}
}

func TestWailsServicePreviewParityAndNoEvents(t *testing.T) {
	app := previewApp(t)
	before, err := app.history.Events(t.Context(), "native.preview", commissioner.Stream, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, principal := range []string{"creator", "owner"} {
		app.session = principal
		host := serviceParityHost{Host: app.host}
		desktop, service := New(host), leagueservice.New(host)
		for _, expected := range []int64{0, 1} {
			body := previewRuleBody(t, "roster.limits", `{"min":20,"max":70}`, "next_rollover", "", expected)
			raw, err := commissioner.DecodePreview(body)
			if err != nil {
				t.Fatal(err)
			}
			got, gotErr := desktop.TargetCommissionerPreview(body)
			want, wantErr := service.Preview(raw)
			if gotErr != nil || wantErr != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("preview parity: %+v %v / %+v %v", got, gotErr, want, wantErr)
			}
			if (principal == "owner" || expected == 0) && len(got.Blockers) == 0 {
				t.Fatal("unauthorized/stale preview allowed")
			}
		}
	}
	after, err := app.history.Events(t.Context(), "native.preview", commissioner.Stream, 0)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("preview changed events: %v", err)
	}
	// Invalid typed Raw input must be refused before touching even an unconfigured host.
	bad, err := leagueservice.New(nil).Preview(commissioner.RawCommand{})
	if err != nil || len(bad.Blockers) == 0 {
		t.Fatalf("invalid Raw preview accepted: %+v %v", bad, err)
	}
	if _, err := leagueservice.New(nil).TargetNativeLineupCheck(commissioner.RawNativeLineup{}); err == nil {
		t.Fatal("invalid Raw lineup accepted")
	}
	if _, err := leagueservice.New(nil).TargetLeagueCalendar(leagueservice.RawLeagueCalendarScope{}); err == nil {
		t.Fatal("invalid Raw calendar accepted")
	}
}

func TestServicePublicationOnlyCompletedReadPreviewEntries(t *testing.T) {
	entries, err := ReadPreviewCommands()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(NativeCommands()) {
		t.Fatal("service catalog omitted a native family")
	}
	typ := reflect.TypeFor[*leagueservice.Service]()
	for _, entry := range entries {
		if err := entry.ValidatePublication(); err != nil {
			t.Fatal(err)
		}
		if slices.Contains(entry.Modes, catalog.Execute) {
			t.Fatalf("slice exposed execution: %s", entry.ID)
		}
		if _, ok := typ.MethodByName(entry.Method); !ok {
			t.Fatalf("registered unimplemented method: %s", entry.Method)
		}
	}
	// The legacy generic desktop execute/preview entries still have unset metadata.
	if _, err := catalog.Publication(Commands()); err == nil {
		t.Fatal("unfinished desktop doors became publishable")
	}
}
