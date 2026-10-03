package archive

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type memSink struct {
	mu   sync.Mutex
	got  []Fetch
	fail error
}

func (m *memSink) Record(_ context.Context, f Fetch) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.got = append(m.got, f)
	return m.fail
}

func TestRedirectIsRecordedUnderTheURLAskedFor(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/release/file.csv", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/objects/abc123", http.StatusFound)
	})
	mux.HandleFunc("/objects/abc123", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "a,b\n1,2\n")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	sink := &memSink{}
	client := &http.Client{Transport: &Transport{Sink: sink}}

	resp, err := client.Get(srv.URL + "/release/file.csv?token=s3cret&year=2025")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()

	var final Fetch
	for _, f := range sink.got {
		if strings.Contains(f.URL, "s3cret") {
			t.Errorf("credential recorded: %s", f.URL)
		}
		if f.Status == http.StatusOK {
			final = f
		}
	}
	if !strings.Contains(final.URL, "/release/file.csv") || !strings.Contains(final.URL, "year=2025") {
		t.Errorf("final body recorded under %q, want the URL first requested", final.URL)
	}
	body, err := Gunzip(final.Gzip)
	if err != nil || string(body) != "a,b\n1,2\n" || final.Size != int64(len(body)) || len(final.SHA256) != 64 {
		t.Errorf("recorded body %q (size %d, sha %q), %v", body, final.Size, final.SHA256, err)
	}
}

func TestSinkFailureFailsTheRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "data")
	}))
	defer srv.Close()
	sink := &memSink{fail: errors.New("disk full")}
	client := &http.Client{Transport: &Transport{Sink: sink}}
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("read error = %v, want the archive failure", err)
	}
}
