package playerid

import "testing"

func TestNew(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"zero-pads short id", "99", "0099", false},
		{"collapses leading-zero variants", "099", "0099", false},
		{"already canonical", "0099", "0099", false},
		{"keeps natural width for large id", "13294", "13294", false},
		{"trims surrounding whitespace", "  531 ", "0531", false},
		{"all-zeros becomes single zero padded", "0000", "0000", false},
		{"rejects empty", "", "", true},
		{"rejects whitespace-only", "   ", "", true},
		{"rejects non-numeric", "abc", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("New(%q): expected error, got %q", tc.raw, got.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("New(%q): unexpected error: %v", tc.raw, err)
			}
			if got.String() != tc.want {
				t.Errorf("New(%q) = %q, want %q", tc.raw, got.String(), tc.want)
			}
		})
	}
}

func TestZeroValueIsZero(t *testing.T) {
	t.Parallel()
	var p PlayerID
	if !p.IsZero() {
		t.Error("zero-value PlayerID should report IsZero() == true")
	}
	id, err := New("12")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if id.IsZero() {
		t.Error("a constructed PlayerID should not report IsZero()")
	}
}
