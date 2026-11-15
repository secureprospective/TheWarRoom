package commissioner

import (
	"errors"
	"strings"
	"testing"
)

type stoppedHost struct{ Host }

func (stoppedHost) Ready() error { return errors.New("startup failed") }

func TestNativeResultsIPCScopeAndStartup(t *testing.T) {
	t.Parallel()
	a := New(stoppedHost{})
	for _, scope := range []RawNativeResultsScope{
		{League: "native.", Season: 2026, Week: 1}, {League: "imported", Season: 2026, Week: 1},
		{League: " native.test", Season: 2026, Week: 1}, {League: "native.test", Season: 0, Week: 1},
		{League: "native.test", Season: 10000, Week: 1}, {League: "native.test", Season: 2026, Week: 0},
		{League: "native.test", Season: 2026, Week: 23},
	} {
		if err := scope.Validate(); err == nil {
			t.Fatalf("scope validation accepted %+v", scope)
		}
		if _, err := a.TargetNativeResults(scope.League, scope.Season, scope.Week); err == nil {
			t.Fatalf("accepted %+v", scope)
		}
	}
	if _, err := a.TargetNativeStandings("native.test", 0); err == nil {
		t.Fatal("invalid standings season")
	}
	if _, err := a.TargetNativeResults("native.test", 2026, 1); err == nil {
		t.Fatal("results ignored startup")
	}
	if _, err := a.TargetNativeStandings("native.test", 2026); err == nil {
		t.Fatal("standings ignored startup")
	}
	if _, err := a.TargetNativeScorePreview(`{"command":"bad"}`); err == nil {
		t.Fatal("invalid preview")
	}
}
func TestNativeBracketIPCScopeAndStartup(t *testing.T) {
	t.Parallel()
	a := New(stoppedHost{})
	for _, scope := range []RawNativeResultsScope{{League: "native.", Season: 2026}, {League: "imported",
		Season: 2026},
		{League: "native.test", Season: 0}, {League: "native.test", Season: 10000}} {
		if _, err := a.TargetNativeBracket(scope.League, scope.Season); err == nil {
			t.Fatalf("accepted %+v", scope)
		}
	}
	if _, err := a.TargetNativeBracket("native.test", 2026); err == nil || !strings.Contains(err.Error(),
		"startup") {
		t.Fatalf("the bracket ignored a failed startup: %v", err)
	}
}
