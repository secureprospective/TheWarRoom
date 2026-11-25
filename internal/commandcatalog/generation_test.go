package catalog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerationRegistryBuildsAndRefusesUnboundExecution(t *testing.T) {
	registry, err := GenerationRegistry()
	if err != nil {
		t.Fatal(err)
	}
	var nativeCount, mflCount, browserCount, cliCount int
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
	}
	if nativeCount == 0 || mflCount == 0 || browserCount == 0 || cliCount == 0 {
		t.Fatalf("empty surface counts: native=%d mfl=%d browser=%d cli=%d",
			nativeCount, mflCount, browserCount, cliCount)
	}
	if _, err := registry.Invoke(t.Context(), NativeDoor, "commish.leagues.read", Read, json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "cannot execute") {
		t.Fatalf("generation unbound handler executed: err=%v", err)
	}
}
