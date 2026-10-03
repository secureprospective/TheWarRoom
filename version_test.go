package main

import "testing"

func TestBuildLabel(t *testing.T) {
	saved := commit
	t.Cleanup(func() { commit = saved })

	commit = ""
	if got := buildLabel(); got != version {
		t.Errorf("unstamped: buildLabel() = %q, want %q", got, version)
	}
	commit = "abc1234"
	if got, want := buildLabel(), version+" (abc1234)"; got != want {
		t.Errorf("stamped: buildLabel() = %q, want %q", got, want)
	}
}
