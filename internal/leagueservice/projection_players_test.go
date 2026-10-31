package leagueservice

import (
	"slices"
	"strings"
	"testing"
)

func TestProjectionPlayers(t *testing.T) {
	many := strings.Repeat("0816,", maxProjectionPlayers) // duplicates collapse to one
	var tooMany []string
	for i := range maxProjectionPlayers + 1 {
		tooMany = append(tooMany, itoa(10000+i))
	}
	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{"list keeps order", "15711,0816", []string{"15711", "0816"}, false},
		{"spaces and blanks ignored", " 0816 , ,15711,", []string{"0816", "15711"}, false},
		{"duplicates collapse", many, []string{"0816"}, false},
		{"empty", "", nil, true},
		{"only commas", ",,", nil, true},
		{"short id refused", "816", nil, true},
		{"injection refused", "0816;DROP", nil, true},
		{"too many", strings.Join(tooMany, ","), nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := projectionPlayers(tt.raw)
			if (err != nil) != tt.wantErr || !slices.Equal(got, tt.want) {
				t.Fatalf("got %v err %v, want %v wantErr %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func itoa(n int) string {
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
