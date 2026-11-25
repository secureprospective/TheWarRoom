package leaguerules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatLink(t *testing.T) {
	for _, c := range []struct{ value, refusal string }{
		{"", ""}, {"https://discord.com/invite/league", ""},
		{"http://example.com", "must start with https://"},
		{"javascript:alert(1)", "must start with https://"},
		{"data:text/plain,chat", "must start with https://"},
		{"ftp://example.com", "must start with https://"},
		{"https:///chat", "must include a host"},
		{"https://example.com/" + strings.Repeat("x", 300), "at most 300 characters"},
	} {
		t.Run(c.value, func(t *testing.T) {
			body, err := json.Marshal(c.value)
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecodeSetting("league.chatLink", body)
			if c.refusal == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.refusal) {
				t.Fatalf("got %v, want %s", err, c.refusal)
			}
		})
	}
}
