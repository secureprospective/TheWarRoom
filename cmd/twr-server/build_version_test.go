package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestServerBuildVersion(t *testing.T) {
	cases := []struct{ name, flags, stamp string }{
		{"default", "", "Server build dev (commit: , date: )"},
		{"tagged", "-X main.version=v0.10.0 -X main.commit=abc123 -X main.buildDate=2026-11-16T00:00:00Z", "Server build 0.10.0 (commit: abc123, date: 2026-11-16T00:00:00Z)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binary := filepath.Join(t.TempDir(), "twr-server")
			cmd := exec.Command("go", "build", "-o", binary, "-ldflags", tc.flags, ".")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("build: %v: %s", err, out)
			}
			out, err := exec.Command(binary, "version").CombinedOutput()
			if err != nil {
				t.Fatalf("version: %v: %s", err, out)
			}
			lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
			if len(lines) != 2 || lines[0] != "League server release 8: league file version 15; serves The War Room protocol 1.0 to 1.1; durable commands 1." || lines[1] != tc.stamp {
				t.Fatalf("unexpected version output: %q", out)
			}
			start, _ := exec.Command(binary, "hosted").CombinedOutput()
			if strings.Count(string(start), tc.stamp) != 1 {
				t.Fatalf("hosted startup stamp: %q", start)
			}
		})
	}
}
