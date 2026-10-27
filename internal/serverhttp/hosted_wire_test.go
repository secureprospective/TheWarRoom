package serverhttp_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// hostedWireVolatile matches what differs between two runs of a hosted workflow: clock instants (also in words), SHA-256
// digests, account ids, store revisions and the countdowns to the trade deadline and the pick clock.
func hostedWireVolatile() []struct {
	pattern *regexp.Regexp
	with    string
} {
	return []struct {
		pattern *regexp.Regexp
		with    string
	}{
		{regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})`), "<instant>"},
		{regexp.MustCompile(`\d{1,2} [A-Z][a-z]{2} \d{4} \d{2}:\d{2} [A-Z]{2,5}`), "<instant>"},
		{regexp.MustCompile(`([A-Z][a-z]{2}, )?[A-Z][a-z]{2} \d{1,2}, \d{4} at \d{1,2}:\d{2} [AP]M [A-Z]{2,5}`), "<instant>"},
		{regexp.MustCompile(`[0-9a-f]{64}`), "<sha256>"},
		{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "<uuid>"},
		{regexp.MustCompile(`"revision": \d+`), `"revision": <n>`},
		{regexp.MustCompile(`"secondsToTradeDeadline": \d+`), `"secondsToTradeDeadline": <n>`},
		{regexp.MustCompile(`"remaining": \d{10,}`), `"remaining": <nanoseconds>`},
	}
}

func normaliseHostedWire(body []byte) string {
	for _, v := range hostedWireVolatile() {
		body = v.pattern.ReplaceAll(body, []byte(v.with))
	}
	return string(body)
}

// checkHostedWire holds a frontend fixture to its writer: TWR_UPDATE_HOSTED_WIRE=1 rewrites it, and
// otherwise the held file must match this run's output once the run-to-run values are normalised.
func checkHostedWire(t *testing.T, path string, body []byte) {
	t.Helper()
	if os.Getenv("TWR_UPDATE_HOSTED_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	held, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want, got := strings.Split(normaliseHostedWire(held), "\n"), strings.Split(normaliseHostedWire(body), "\n")
	for i := range max(len(want), len(got)) {
		if i >= len(want) || i >= len(got) || want[i] != got[i] {
			t.Fatalf("%s differs from its writer's output at line %d (held %q, written %q); rerun with "+
				"TWR_UPDATE_HOSTED_WIRE=1", path, i+1, line(want, i), line(got, i))
		}
	}
}

func line(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "<end>"
}
