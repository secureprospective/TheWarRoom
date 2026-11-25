package serverhttp

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

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

func TestWriteReadingUsesIndependentGzipStreams(t *testing.T) {
	h := Hosted{gzipWriters: &sync.Pool{}}
	r := httptest.NewRequestWithContext(t.Context(), "GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	for _, owner := range []string{"first owner", "second owner", "third owner"} {
		body := []byte(`{"reading":"` + strings.Repeat(owner, gzipMinimum) + `"}`)
		w := httptest.NewRecorder()
		h.writeReading(w, r, body)
		if w.Header().Get("Content-Encoding") != "gzip" || w.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatalf("unexpected gzip headers: %v", w.Header())
		}
		zr, err := gzip.NewReader(w.Body)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(zr)
		if err != nil || !bytes.Equal(got, body) {
			t.Fatalf("%s: gzip stream did not contain exactly its reading: %v", owner, err)
		}
		if err := zr.Close(); err != nil {
			t.Fatal(err)
		}
	}
	plain := httptest.NewRecorder()
	(Hosted{}).writeReading(plain, r, []byte(`{"reading":"`+strings.Repeat("x", gzipMinimum)+`"}`))
	if plain.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("a Hosted without a writer pool did not gzip: %v", plain.Header())
	}
	w := httptest.NewRecorder()
	h.writeReading(w, r, []byte(`{"reading":"small"}`))
	if w.Header().Get("Content-Encoding") != "" || w.Header().Get("Vary") != "Accept-Encoding" || w.Body.String() != `{"reading":"small"}` {
		t.Fatalf("small reading was not sent as plain JSON with Vary: %v", w.Header())
	}
}
