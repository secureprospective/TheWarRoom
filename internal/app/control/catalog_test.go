package control

import "testing"

func TestMFLKeysRemainDesktopOnly(t *testing.T) {
	remaining := map[string]bool{"MFLKeyStatus": true, "SetMFLKey": true, "DeleteMFLKey": true}
	for _, command := range Commands() {
		if !remaining[command.Method] {
			continue
		}
		if command.Availability != "desktop" {
			t.Fatalf("MFL key operation exposed remotely: %+v", command)
		}
		delete(remaining, command.Method)
	}
	if len(remaining) != 0 {
		t.Fatalf("missing key operations: %v", remaining)
	}
}
