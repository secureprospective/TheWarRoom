package serverhttp

import "testing"

func TestAcceptsGzip(t *testing.T) {
	for header, want := range map[string]bool{
		"": false, "identity": false, "br": false, "gzip": true, "GZIP": true, "x-gzip": true, "*": true,
		"br, gzip;q=0.5": true, "gzip;q=0": false, "gzip; q=0.0": false, "gzip;q=1": true, "deflate, br;q=1": false,
		"gzip;level=1": true, "gzip;q=junk": false,
	} {
		if got := acceptsGzip([]string{header}); got != want {
			t.Errorf("%q: got %v want %v", header, got, want)
		}
	}
}
