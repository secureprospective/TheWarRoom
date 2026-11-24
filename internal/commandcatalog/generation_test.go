package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerationRegistryBuildsAndRefusesDesktopExecution(t *testing.T) {
	registry, err := GenerationRegistry()
	if err != nil {
		t.Fatal(err)
	}
	var nativeCount, mflCount, browserCount, desktopCount, cliCount int
	for _, entry := range registry.Entries() {
		if entry.Native != nil {
			nativeCount++
		}
		if entry.MFL != nil {
			mflCount++
		}
		if entry.Browser != nil {
			browserCount++
		}
		if entry.CLI {
			cliCount++
		}
		desktopCount += len(entry.Desktop)
	}
	if nativeCount == 0 || mflCount == 0 || browserCount == 0 || desktopCount == 0 || cliCount == 0 {
		t.Fatalf("empty surface counts: native=%d mfl=%d browser=%d desktop=%d cli=%d",
			nativeCount, mflCount, browserCount, desktopCount, cliCount)
	}
	if _, err := registry.Invoke(t.Context(), WailsDoor, "commissioner.TargetCommissionerLeagues", "", json.RawMessage(`[]`)); err == nil || !strings.Contains(err.Error(), "cannot execute") {
		t.Fatalf("generation desktop binding executed: err=%v", err)
	}
}
