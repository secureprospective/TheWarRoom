package archive

import (
	"bytes"
	"compress/gzip"
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
	body, err := Gunzip(final.Gzip, final.Size)
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

func TestAPIKEYURLAndEchoRedaction(t *testing.T) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		"https://example.test/export?APIKEY=private-test-value&L=league", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := originURL(req)
	if strings.Contains(got, "private-test-value") || !strings.Contains(got, "APIKEY=REDACTED") {
		t.Fatalf("URL not redacted: %s", got)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "prefix private-test-value suffix")
	}))
	defer srv.Close()
	sink := &memSink{}
	client := &http.Client{Transport: &Transport{Sink: sink}}
	resp, err := client.Get(srv.URL + "?APIKEY=private-test-value")
	if err != nil {
		t.Fatal(err)
	}
	// One-byte reads prove that matching works across body read boundaries.
	buffer := make([]byte, 1)
	for {
		_, readErr := resp.Body.Read(buffer)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			t.Fatal(readErr)
		}
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if len(sink.got) != 1 {
		t.Fatal("missing fetch")
	}
	f := sink.got[0]
	if f.Err == "" || len(f.Gzip) != 0 || f.SHA256 != "" || f.Size != 0 {
		t.Fatal("echoed credential body archived")
	}
}

// A judged 200 refusal is recorded as a failed fetch with its body kept; data, large bodies and non-200
// answers are never judged.
func TestJudgeMarksRefusals(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/refused", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "NO login") })
	mux.HandleFunc("/data", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "rows") })
	mux.HandleFunc("/big", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "NO"+strings.Repeat("x", judgedBytes))
	})
	mux.HandleFunc("/gone", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "NO")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	sink := &memSink{}
	judge := func(_ string, body []byte) string {
		if strings.HasPrefix(string(body), "NO") {
			return "refused: " + string(body)
		}
		return ""
	}
	client := &http.Client{Transport: &Transport{Sink: sink, Judge: judge}}
	for _, path := range []string{"/refused?APIKEY=s3cret", "/data", "/big", "/gone"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}
	got := map[string]string{}
	for _, f := range sink.got {
		got[strings.SplitN(strings.TrimPrefix(f.URL, srv.URL), "?", 2)[0]] = f.Err
		if f.SHA256 == "" {
			t.Errorf("%s: body not kept", f.URL)
		}
	}
	want := map[string]string{"/refused": "refused: NO login", "/data": "", "/big": "", "/gone": ""}
	for path, err := range want {
		if got[path] != err {
			t.Errorf("%s: error %q, want %q", path, got[path], err)
		}
	}
}

func TestGunzipBodySize(t *testing.T) {
	const plain = "archived body"
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := io.WriteString(zw, plain); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		body   []byte
		size   int64
		refuse bool
	}{
		{name: "larger than recorded", body: compressed.Bytes(), size: int64(len(plain)) - 1, refuse: true},
		{name: "shorter than recorded", body: compressed.Bytes(), size: int64(len(plain)) + 1, refuse: true},
		{name: "over cap before opening gzip", body: []byte("not gzip"), size: maxArchivedBody + 1, refuse: true},
		{name: "negative size", body: compressed.Bytes(), size: -1, refuse: true},
		{name: "exact size", body: compressed.Bytes(), size: int64(len(plain))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Gunzip(tt.body, tt.size)
			if tt.refuse {
				if !errors.Is(err, ErrBodySize) || got != nil {
					t.Fatalf("Gunzip = %q, %v; want nil, ErrBodySize", got, err)
				}
				return
			}
			if err != nil || string(got) != plain {
				t.Fatalf("Gunzip = %q, %v; want %q, nil", got, err, plain)
			}
		})
	}
}
