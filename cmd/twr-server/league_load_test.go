package main

import (
	"strings"
	"testing"
)

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
