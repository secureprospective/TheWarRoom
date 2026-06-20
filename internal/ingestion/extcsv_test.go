package ingestion

import (
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
