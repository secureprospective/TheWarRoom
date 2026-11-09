package ingestion

import (
	"context"
	"net/http"
	"testing"
)

func TestFetchCSV_RaggedRowFailsLoud(t *testing.T) {
	t.Parallel()
	client, url := serveCSV(t, http.StatusOK, "a,b,c\n1,2\n")
	if _, err := FetchCSV(context.Background(), client, url, DefaultMaxCSVBytes); err == nil {
		t.Fatal("expected error on inconsistent field count")
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
