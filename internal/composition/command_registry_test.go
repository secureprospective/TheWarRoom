package composition

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/app/offers"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

func TestRegistryTypedDispatch(t *testing.T) {
	app := NewApp()
	inputs := newCommandAPIs(app).registryInputs(app)
	calls := 0
	inputs.Desktop["commissioner"]["TargetNativeBids"] = catalog.Bind2(func(league, franchise string) (string, error) {
		calls++
		return league + ":" + franchise, nil
	}, "league", "franchise")
	inputs.Desktop["offers"]["TargetBidRevise"] = catalog.Bind2(func(id string, bid offers.RawBidInput) (string, error) {
		calls++
		return id + ":" + bid.Player + ":" + bid.Salary, nil
	}, "id", "input")
	registry, err := catalog.NewRegistry(inputs)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		door       catalog.Door
		id         string
		mode       catalog.Mode
		body, want string
	}{
		{"native named read", catalog.NativeDoor, "bids.read", catalog.Read, `{"league":"native.test","franchise":"franchise.01"}`, `"native.test:franchise.01"`},
		{"desktop positional read", catalog.WailsDoor, "commissioner.TargetNativeBids", "", `["native.test","franchise.01"]`, `"native.test:franchise.01"`},
		{"local bid revision", catalog.MFLDoor, "mfl.bid.revise", catalog.Execute, `{"id":"plan.1","player":"0010","salary":"5.25"}`, `"plan.1:0010:5.25"`},
		{"unknown id", catalog.NativeDoor, "not.registered", catalog.Read, `{}`, ""},
		{"undeclared door", catalog.MFLDoor, "bids.read", catalog.Read, `{}`, ""},
		{"unsupported mode", catalog.NativeDoor, "bids.read", catalog.Execute, `{}`, ""},
		{"unknown native argument", catalog.NativeDoor, "bids.read", catalog.Read, `{"league":"native.test","franchise":"franchise.01","extra":1}`, ""},
		{"wrong desktop count", catalog.WailsDoor, "commissioner.TargetNativeBids", "", `["native.test"]`, ""},
		{"wrong desktop type", catalog.WailsDoor, "commissioner.TargetNativeBids", "", `[1,"franchise.01"]`, ""},
		{"wrong native id in body", catalog.NativeDoor, "commish.rules.set", catalog.Execute, `{"command":"other"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := calls
			got, err := registry.Invoke(t.Context(), tt.door, tt.id, tt.mode, json.RawMessage(tt.body))
			if tt.want == "" {
				if err == nil || calls != before {
					t.Fatalf("refusal invoked handler: result=%s err=%v calls=%d", got, err, calls-before)
				}
				return
			}
			if err != nil || string(got) != tt.want || calls != before+1 {
				t.Fatalf("dispatch changed arguments/result: got=%s err=%v", got, err)
			}
		})
	}
}

func TestRegistryRejectsUnboundAndUnregisteredMethods(t *testing.T) {
	tests := map[string]func(*catalog.RegistryInputs){
		"missing handler": func(in *catalog.RegistryInputs) { delete(in.Desktop["offers"], "TargetPool") },
		"extra method": func(in *catalog.RegistryInputs) {
			in.Desktop["offers"]["Unregistered"] = in.Desktop["offers"]["TargetPool"]
		},
		"nil handler": func(in *catalog.RegistryInputs) {
			b := in.Desktop["offers"]["TargetPool"]
			b.Handler = nil
			in.Desktop["offers"]["TargetPool"] = b
		},
		"wrong method": func(in *catalog.RegistryInputs) {
			b := in.Desktop["race"]["TargetRace"]
			delete(in.Desktop["race"], "TargetRace")
			in.Desktop["race"]["Unregistered"] = b
		},
		"extra feature": func(in *catalog.RegistryInputs) { in.Desktop["unregistered"] = map[string]catalog.Binding{} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			app := NewApp()
			inputs := newCommandAPIs(app).registryInputs(app)
			mutate(&inputs)
			if registry, err := catalog.NewRegistry(inputs); err == nil || registry != nil {
				t.Fatal("invalid executable registry admitted")
			}
		})
	}
}

func TestTypedBindingPreservesErrorsAndRejectsTrailingJSON(t *testing.T) {
	refused := errors.New("the plan stays unchanged")
	calls := 0
	binding := catalog.Bind1(func(value string) (string, error) { calls++; return value, refused }, "value")
	if _, err := binding.Handler(t.Context(), json.RawMessage(`["kept"]`)); !errors.Is(err, refused) {
		t.Fatalf("lost domain error: %v", err)
	}
	for _, body := range []string{`["kept"] {}`, `["kept",true]`, `[{}]`} {
		before := calls
		if _, err := binding.Handler(t.Context(), json.RawMessage(body)); err == nil || calls != before {
			t.Fatalf("invalid arguments reached handler: %s", body)
		}
	}
	if !strings.Contains(binding.Arguments.Validator, "Raw validation") {
		t.Fatal("binding lost validation owner")
	}
}

func TestResearchBindingsStayRetired(t *testing.T) {
	registry, err := catalog.NewRegistry(newCommandAPIs(NewApp()).registryInputs(NewApp()))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"counterparties.TargetCounterparties", "counterparties.TargetCounterpartTeam",
		"offers.TargetOffers", "draft.TargetDraftLab", "market.TargetMarket",
		"market.TargetAssetProfile", "market.TargetPickClock", "archive.TargetArchive",
	} {
		t.Run(id, func(t *testing.T) {
			if _, err := registry.Invoke(t.Context(), catalog.WailsDoor, id, "", json.RawMessage(`[]`)); err == nil {
				t.Fatal("retired research binding remains executable")
			}
		})
	}
}
