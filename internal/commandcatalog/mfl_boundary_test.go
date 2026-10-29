package catalog

import (
	"fmt"
	"strings"
	"testing"
)

func checkMFLNames(native, mfl []Command) error {
	names := make(map[string]bool, len(native))
	for _, command := range native {
		names[command.ID] = true
	}
	for _, command := range mfl {
		if names[command.ID] {
			return fmt.Errorf("command %q appears in native and MFL", command.ID)
		}
		if !strings.HasPrefix(command.ID, "mfl.") {
			return fmt.Errorf("MFL command %q is not namespaced", command.ID)
		}
	}
	return nil
}

func TestMFLCommandNamesAreSeparate(t *testing.T) {
	native := append(NativeCommands(Schema{}), DesktopCommands("commissioner", nil)...)
	mfl := append(MFLCommands(MFLShapes{}), MFLPlanCommands()...)
	for _, door := range []struct{ feature, method, id string }{
		{"contracts", "TargetDraftLineup", "mfl.lineup.set"},
		{"trades", "TargetDraftTradeAccept", "mfl.trade.accept"},
	} {
		found := false
		for _, command := range DesktopCommands(door.feature, nil) {
			if command.Method == door.method {
				if command.ID != door.id {
					t.Errorf("%s ID = %q, want %q", door.method, command.ID, door.id)
				}
				mfl = append(mfl, command)
				found = true
			}
		}
		if !found {
			t.Errorf("missing MFL desktop method %s", door.method)
		}
	}
	if err := checkMFLNames(native, mfl); err != nil {
		t.Fatal(err)
	}
}

func TestMFLNameGateRejectsCollisionAndUnqualifiedName(t *testing.T) {
	for _, test := range []struct {
		name        string
		native, mfl []Command
	}{
		{"collision", []Command{{ID: "mfl.test"}}, []Command{{ID: "mfl.test"}}},
		{"unqualified", nil, []Command{{ID: "test"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := checkMFLNames(test.native, test.mfl); err == nil {
				t.Fatal("invalid names accepted")
			}
		})
	}
}
