package serverhttp

import (
	"context"
	"testing"
)

// A pin changed while a request runs cannot give that request two football selections: the request
// keeps the pin it first read; another league, and the next request, see the new one.
func TestRequestKeepsThePinItFirstRead(t *testing.T) {
	current := "before"
	h := Hosted{Pins: func(context.Context, string) LeaguePin { return LeaguePin{Digest: current} }}
	ctx := h.withRequestPin(t.Context(), "native.a")
	current = "after"
	for _, tc := range []struct{ league, want string }{{"native.a", "before"}, {"native.b", "after"}} {
		if got := h.pinFor(ctx, tc.league).Digest; got != tc.want {
			t.Fatalf("%s pin = %q, want %q", tc.league, got, tc.want)
		}
	}
	if got := h.pinFor(h.withRequestPin(t.Context(), "native.a"), "native.a").Digest; got != "after" {
		t.Fatalf("next request pin = %q", got)
	}
}
