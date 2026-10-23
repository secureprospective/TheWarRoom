package governance

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestCheckpointResumeMatchesReplayAndOwnsState(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	events := []Event{
		LeagueCreated{Creator: "creator", Franchises: []FranchiseID{"one", "two"}},
		InviteIssued{ID: "invite", Franchise: "one", Redeemer: "owner"},
		InviteRedeemed{ID: "invite", Principal: "owner"},
		RoleGranted{Principal: "owner", Role: DOT},
		ActivityObserved{Principal: "creator", At: at},
	}
	state, err := Replay(events)
	if err != nil {
		t.Fatal(err)
	}
	raw := state.Checkpoint()
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RawCheckpoint
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil {
		t.Fatal(err)
	}
	suffix := []Event{ActivityObserved{Principal: "owner", At: at.Add(time.Hour)}, InviteIssued{ID: "next", Franchise: "two", Redeemer: "new"}}
	got, err := Resume(decoded, suffix, len(events))
	if err != nil {
		t.Fatal(err)
	}
	want, err := Replay(append(events, suffix...))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("resumed state differs from replay")
	}
	if _, ok := decoded.Invites["next"]; ok {
		t.Fatal("resume mutated persisted checkpoint")
	}
	if _, ok := state.Invite("next"); ok {
		t.Fatal("checkpoint shared original state")
	}
	decoded.Roles["creator"][Commissioner] = false
	if !got.HasRole("creator", Commissioner) || !state.HasRole("creator", Commissioner) {
		t.Fatal("checkpoint ownership leaked")
	}
	bad := []Event{SeatRemoved{Franchise: "absent"}}
	_, resumedErr := Resume(raw, bad, len(events))
	_, fullErr := Replay(append(events, bad...))
	if resumedErr == nil || fullErr == nil || resumedErr.Error() != fullErr.Error() {
		t.Fatalf("errors differ: %v / %v", resumedErr, fullErr)
	}
	if _, err := Resume(RawCheckpoint{}, nil, 0); err == nil {
		t.Fatal("invalid checkpoint accepted")
	}
}
