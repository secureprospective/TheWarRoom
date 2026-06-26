package ingestion

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serveCSV(t *testing.T, status int, body string) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

// serveRaw serves an exact byte body (used to serve gzip-compressed payloads, and a
// deliberately non-gzip body to prove the gunzip step fails loud).
func serveRaw(t *testing.T, body []byte) (*http.Client, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

func gzipBytes(t *testing.T, s string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(s)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// M3 gate-proof: the byte cap is not real until a deliberate over-cap payload is
// seen to fail it. A single-column body avoids field-count noise on truncation, so
// the failure is unambiguously the cap, not a parse error.
func TestFetchCSV_ExceedsCapFailsLoud(t *testing.T) {
	t.Parallel()
	body := strings.Repeat("a", 600) + "\n"
	client, url := serveCSV(t, http.StatusOK, body)

	_, err := FetchCSV(context.Background(), client, url, 50)
	if err == nil {
		t.Fatal("expected a cap-exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("want a cap-exceeded error, got %v", err)
	}
}

// A payload at/under the cap parses normally — proves the cap rejects only genuine
// overflow, not legitimate files (the false-positive half of the gate).
func TestFetchCSV_UnderCapSucceeds(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusOK, "col_a,col_b\n1,2\n3,4\n")

	records, err := FetchCSV(context.Background(), client, url, DefaultMaxCSVBytes)
	if err != nil {
		t.Fatalf("FetchCSV: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3 (header + 2)", len(records))
	}
}

func TestFetchCSV_Non200(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusNotFound, "nope")
	if _, err := FetchCSV(context.Background(), client, url, DefaultMaxCSVBytes); err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestFetchCSV_RaggedRowFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusOK, "a,b,c\n1,2\n")
	if _, err := FetchCSV(context.Background(), client, url, DefaultMaxCSVBytes); err == nil {
		t.Fatal("expected error on inconsistent field count")
	}
}

// M3 gate-proof for the streaming path: the byte cap is not real until a deliberate
// over-cap payload is seen to fail it. A single-column body keeps truncation clean
// (no field-count noise), so the failure is unambiguously the cap, not a parse error.
func TestStreamCSV_ExceedsCapFailsLoud(t *testing.T) {
	t.Parallel()
	body := "col\n" + strings.Repeat("a", 600) + "\n"
	client, url := serveCSV(t, http.StatusOK, body)

	err := StreamCSV(context.Background(), client, url, 50, []string{"col"},
		func(_ map[string]int, _ []string) error { return nil })
	if err == nil {
		t.Fatal("expected a cap-exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("want a cap-exceeded error, got %v", err)
	}
}

// A per-row callback error aborts the stream and surfaces loudly (the malformed-row
// half of the gate) — and rows after the offending one are not processed.
func TestStreamCSV_CallbackErrorAborts(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusOK, "n\n5\nbad\n7\n")

	seen := 0
	err := StreamCSV(context.Background(), client, url, DefaultMaxCSVBytes, []string{"n"},
		func(cols map[string]int, rec []string) error {
			seen++
			_, err := IntCell(rec, cols["n"], "n")
			return err
		})
	if err == nil || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("want a malformed-cell error naming the value, got %v", err)
	}
	if seen != 2 {
		t.Fatalf("stream did not abort: saw %d rows, want 2 (5, then bad)", seen)
	}
}

// The happy path: every data row is delivered once, columns are bound by name, and a
// cell value copied out of the reused record survives (ReuseRecord copy-safety).
func TestStreamCSV_StreamsAllRows(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusOK, "a,b\n1,2\n3,4\n5,6\n")

	var got []string
	err := StreamCSV(context.Background(), client, url, DefaultMaxCSVBytes, []string{"a", "b"},
		func(cols map[string]int, rec []string) error {
			if cols["a"] != 0 || cols["b"] != 1 {
				t.Errorf("columns bound wrong: %v", cols)
			}
			got = append(got, rec[cols["a"]])
			return nil
		})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if want := []string{"1", "3", "5"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got rows %v, want %v", got, want)
	}
}

func TestStreamCSV_Non200(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusNotFound, "nope")
	err := StreamCSV(context.Background(), client, url, DefaultMaxCSVBytes, nil,
		func(_ map[string]int, _ []string) error { return nil })
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

// The gz happy path: a gzip-compressed CSV is transparently gunzipped and parsed,
// proving FetchCSVGz reads the same records FetchCSV would from the plain body.
func TestFetchCSVGz_Decompresses(t *testing.T) {
	t.Parallel()
	client, url := serveRaw(t, gzipBytes(t, "col_a,col_b\n1,2\n3,4\n"))

	records, err := FetchCSVGz(context.Background(), client, url, DefaultMaxCSVBytes)
	if err != nil {
		t.Fatalf("FetchCSVGz: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3 (header + 2)", len(records))
	}
}

// Gate-proof: a body that is NOT gzip must FAIL LOUD at the gunzip step, never get
// mis-parsed as plain CSV. Feeding plain CSV to FetchCSVGz is the deliberate
// violation; gzip.NewReader rejects the missing magic header.
func TestFetchCSVGz_NonGzipFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveRaw(t, []byte("col_a,col_b\n1,2\n"))

	_, err := FetchCSVGz(context.Background(), client, url, DefaultMaxCSVBytes)
	if err == nil {
		t.Fatal("expected a gunzip error on a non-gzip body, got nil")
	}
	if !strings.Contains(err.Error(), "gunzip") {
		t.Fatalf("want a gunzip error, got %v", err)
	}
}

// Gate-proof: the cap on the gz path bounds the DECOMPRESSED bytes. A payload that is
// small compressed but expands past the cap must fail loud — this is the gzip-bomb
// guard. A single column keeps truncation unambiguous (cap, not field-count noise).
func TestFetchCSVGz_DecompressedExceedsCapFailsLoud(t *testing.T) {
	t.Parallel()
	// 600 identical bytes compress tiny but blow a 50-byte decompressed cap.
	client, url := serveRaw(t, gzipBytes(t, "col\n"+strings.Repeat("a", 600)+"\n"))

	_, err := FetchCSVGz(context.Background(), client, url, 50)
	if err == nil {
		t.Fatal("expected a cap-exceeded error on decompressed overflow, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("want a cap-exceeded error, got %v", err)
	}
}

// The streaming gz happy path: every data row is delivered once from a gzipped source,
// columns bound by name.
func TestStreamCSVGz_StreamsAllRows(t *testing.T) {
	t.Parallel()
	client, url := serveRaw(t, gzipBytes(t, "a,b\n1,2\n3,4\n5,6\n"))

	var got []string
	err := StreamCSVGz(context.Background(), client, url, DefaultMaxCSVBytes, []string{"a", "b"},
		func(cols map[string]int, rec []string) error {
			got = append(got, rec[cols["a"]])
			return nil
		})
	if err != nil {
		t.Fatalf("StreamCSVGz: %v", err)
	}
	if want := []string{"1", "3", "5"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got rows %v, want %v", got, want)
	}
}

// Gate-proof for the streaming gz path: decompressed overflow fails loud.
func TestStreamCSVGz_DecompressedExceedsCapFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveRaw(t, gzipBytes(t, "col\n"+strings.Repeat("a", 600)+"\n"))

	err := StreamCSVGz(context.Background(), client, url, 50, []string{"col"},
		func(_ map[string]int, _ []string) error { return nil })
	if err == nil {
		t.Fatal("expected a cap-exceeded error, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("want a cap-exceeded error, got %v", err)
	}
}

func TestCSVColumns(t *testing.T) {
	t.Parallel()

	// Found, with a leading BOM on the first cell that must be stripped.
	cols, err := CSVColumns([]string{"\ufeffid", "name", "team"}, "id", "team")
	if err != nil {
		t.Fatalf("CSVColumns: %v", err)
	}
	if cols["id"] != 0 || cols["team"] != 2 {
		t.Errorf("indices wrong: %v", cols)
	}

	// Missing required column fails loud, naming the column.
	_, err = CSVColumns([]string{"id", "name"}, "id", "gsis_id")
	if err == nil || !strings.Contains(err.Error(), "gsis_id") {
		t.Fatalf("want missing-column error naming gsis_id, got %v", err)
	}
}

func TestIsMissing(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"", "NA"} {
		if !IsMissing(s) {
			t.Errorf("IsMissing(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"0", "na", "N/A", "x"} {
		if IsMissing(s) {
			t.Errorf("IsMissing(%q) = true, want false", s)
		}
	}
}
