package normalize

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
)

// TestNewLookup_MalformedBirthdateFailsLoud proves a corrupt DETAILS=1 birthdate
// cannot ride into the typed lookup (the fetcher validates too — defense in depth).
func TestNewLookup_MalformedBirthdateFailsLoud(t *testing.T) {
	_, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR", Birthdate: "1979-08-03"},
	})
	if err == nil {
		t.Fatal("NewLookup should error on a non-epoch birthdate, got nil")
	}
}

// TestNewLookup_MalformedDraftYearFailsLoud proves a corrupt DETAILS=1 draft year cannot ride into
// the typed lookup (the fetcher validates too — defense in depth).
func TestNewLookup_MalformedDraftYearFailsLoud(t *testing.T) {
	_, err := NewLookup([]players.RawPlayer{
		{ID: "0099", Name: "Young, Bryce", Position: "QB", Team: "CAR", DraftYear: "twenty-twenty"},
	})
	if err == nil {
		t.Fatal("NewLookup should error on a non-numeric draft year, got nil")
	}
}
