package manifest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/manifest"
)

func TestApprovedGapMatchesBothFileAndHashAndNeverSubstitutes(t *testing.T) {
	fetches, b := archive()
	p := append(pins(), manifest.Pin{File: "2013/assets.json", SHA256: sum("missing")})
	m, err := manifest.Build("frozen", p, fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	g := manifest.Gap{File: "2013/assets.json", SHA256: sum("missing"), Reason: "Pinned error fed no fact"}
	out, gaps, err := m.ResolveWithGaps(context.Background(), b, []manifest.Gap{g})
	if err != nil || len(out) != 3 || len(gaps) != 1 || gaps[0] != g || out[0].Body != nil {
		t.Fatalf("resolution = %+v, %+v, %v", out, gaps, err)
	}
	for _, bad := range []manifest.Gap{
		{File: g.File, SHA256: sum("other"), Reason: g.Reason},
		{File: "2014/assets.json", SHA256: g.SHA256, Reason: g.Reason},
	} {
		if _, _, err := m.ResolveWithGaps(context.Background(), b, []manifest.Gap{bad}); err == nil {
			t.Fatalf("different pin authorized a gap: %+v", bad)
		}
	}
	if _, _, err := m.ResolveWithGaps(context.Background(), b, nil); err == nil {
		t.Fatal("strict Resolve must still reject missing bytes")
	}
}

func TestGapCannotHideCorruptionOrLossOfHeldBytes(t *testing.T) {
	fetches, b := archive()
	m, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	g := manifest.Gap{File: "2026/adp.json", SHA256: sum("adp"), Reason: "Test exception"}
	b[g.SHA256] = "corrupted"
	if _, _, err := m.ResolveWithGaps(context.Background(), b, []manifest.Gap{g}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("corrupted held body: %v", err)
	}
	delete(b, g.SHA256)
	if _, _, err := m.ResolveWithGaps(context.Background(), b, []manifest.Gap{g}); err == nil || !strings.Contains(err.Error(), "no archived body") {
		t.Fatalf("lost held body: %v", err)
	}
}

func TestInvalidGapPolicyAndCancellationFailClosed(t *testing.T) {
	fetches, b := archive()
	m, err := manifest.Build("frozen", pins(), fetches, 2026, cutoffTime())
	if err != nil {
		t.Fatal(err)
	}
	g := manifest.Gap{File: "2013/assets.json", SHA256: sum("missing"), Reason: "Error pin"}
	for _, policy := range [][]manifest.Gap{
		{g, g}, {{File: g.File, SHA256: "invalid", Reason: g.Reason}},
		{{File: g.File, SHA256: g.SHA256}},
	} {
		if _, _, err := m.ResolveWithGaps(context.Background(), b, policy); err == nil {
			t.Fatalf("invalid gap policy accepted: %+v", policy)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := m.ResolveWithGaps(ctx, b, nil); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("cancellation = %v", err)
	}
}
