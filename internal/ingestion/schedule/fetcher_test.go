package schedule

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawScheduleMatchup_Validate(t *testing.T) {
	base := RawScheduleMatchup{
		Kickoff: "1788999600",
		Teams: []RawScheduleTeam{
			{ID: "NEP", IsHome: "0"},
			{ID: "SEA", IsHome: "1"},
		},
	}

	tests := []struct {
		name    string
		mutate  func(m *RawScheduleMatchup)
		wantErr bool
	}{
		{"valid", func(*RawScheduleMatchup) {}, false},
		{"missing kickoff", func(m *RawScheduleMatchup) { m.Kickoff = "" }, true},
		{"non-numeric kickoff", func(m *RawScheduleMatchup) { m.Kickoff = "soon" }, true},
		{"one team", func(m *RawScheduleMatchup) { m.Teams = m.Teams[:1] }, true},
		{"empty team id", func(m *RawScheduleMatchup) { m.Teams[0].ID = "" }, true},
		{"bad isHome", func(m *RawScheduleMatchup) { m.Teams[1].IsHome = "yes" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := RawScheduleMatchup{Kickoff: base.Kickoff, Teams: append([]RawScheduleTeam(nil), base.Teams...)}
			tt.mutate(&m)
			if err := m.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestDecode_RealShape decodes a trimmed copy of the live nflSchedule payload
// (captured on the Beelink) to lock the field mapping and the single-element
// collapse handling on team[].
func TestDecode_RealShape(t *testing.T) {
	const body = `{"nflSchedule":{"matchup":[` +
		`{"kickoff":"1788999600","gameSecondsRemaining":"3600","team":[` +
		`{"id":"NEP","isHome":"0","spread":"2.5","score":""},` +
		`{"id":"SEA","isHome":"1","spread":"2.5","score":""}]}` +
		`]}}`

	var env scheduleEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d matchups, want 1", len(got))
	}
	m := got[0]
	if m.Kickoff != "1788999600" || len(m.Teams) != 2 {
		t.Fatalf("matchup mapped wrong: %+v", m)
	}
	if m.Teams[0].ID != "NEP" || m.Teams[1].ID != "SEA" || m.Teams[1].IsHome != "1" {
		t.Fatalf("teams mapped wrong: %+v", m.Teams)
	}
}
