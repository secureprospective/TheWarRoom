package main

import (
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
)

func TestPrintMFLComparison(t *testing.T) {
	var output strings.Builder
	err := printMFLComparison(&output, leaguemigration.Review{Summary: "2 differences", Franchises: []leaguemigration.FranchiseCheck{
		{Name: "Mismatch", Lines: []string{"a", "b"}}, {Name: "Match", Matches: true},
	}})
	if err != nil || !strings.Contains(output.String(), "MFL comparison: 2 differences") || !strings.Contains(output.String(), "Mismatch: a; b") || strings.Contains(output.String(), "Match") {
		t.Fatalf("comparison output = %q, err = %v", output.String(), err)
	}
}

// The commissioner's password arrives on stdin even though RawCredentials hides it from JSON.
func TestConsoleCredentialsReadThePassword(t *testing.T) {
	raw, err := readConsoleCredentials(strings.NewReader(`{"username":"christopher","password":"a-long-private-password"}`))
	if err != nil || raw.Username != "christopher" || raw.Password != "a-long-private-password" {
		t.Fatalf("credentials = %q, %v", raw.Username, err)
	}
	for _, bad := range []string{`{"username":"christopher"}`, `{"username":"c","password":"a-long-private-password"}`,
		`{"username":"christopher","password":"a-long-private-password","role":"admin"}`, `{} {}`} {
		if _, err := readConsoleCredentials(strings.NewReader(bad)); err == nil {
			t.Fatalf("accepted %s", bad)
		}
	}
}
