package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type envelopeDemo struct {
	Kind    string           `json:"kind"`
	Receipt envelope.Receipt `json:"receipt"`
}

// demoURL is fixture-only: the .invalid host can never resolve, so the demo cannot dispatch.
const demoURL = "https://fixture.invalid/options?O=18"

// demo runs one roster.ir envelope for a real rostered player through draft → ready → handed
// off → not yet done → landed. Times are relative to the held snapshot, not the machine clock.
// The "after" observation is the same snapshot with that one player moved to IR.
func demo(snap snapshot.Snapshot) (envelopeDemo, error) {
	at, err := time.Parse(time.RFC3339, snap.Rosters.Provenance.Freshness.FetchedAt)
	if err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: observation time: %w", err)
	}
	fid, pid, err := demoPlayer(snap)
	if err != nil {
		return envelopeDemo{}, err
	}
	e, err := envelope.New(at.Add(-3*time.Minute), envelope.Spec{
		Intent: "roster.ir", LeagueID: ingestion.LeagueID, FranchiseID: fid,
		Subject:  envelope.Subject{Players: []playerid.PlayerID{pid}},
		Expected: envelope.ExpectedChange{Player: pid, RosterStatus: domain.RosterIR},
		Gravity:  envelope.G2, Undo: envelope.Reversible, Target: envelope.Target{Kind: envelope.Mapped, URL: demoURL},
	}, func() string { return "fixture-roster-ir-001" })
	if err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: draft: %w", err)
	}
	if e, err = e.Check(at.Add(-2*time.Minute), snap, envelope.IRCheck{}); err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: check: %w", err)
	}
	if e, err = e.HandOff(at.Add(-time.Minute)); err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: handoff: %w", err)
	}
	before := envelope.Observation{LeagueID: ingestion.LeagueID, Snapshot: snap}
	if e, err = e.Observe(at, before, envelope.IRPredicate{}); err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: before: %w", err)
	}
	after := envelope.Observation{LeagueID: ingestion.LeagueID,
		Snapshot: irSnapshot(snap, fid, pid, at.Add(time.Minute))}
	if e, err = e.Observe(at.Add(time.Minute), after, envelope.IRPredicate{}); err != nil {
		return envelopeDemo{}, fmt.Errorf("fixtures: after: %w", err)
	}
	return envelopeDemo{Kind: "fixture", Receipt: e.Receipt()}, nil
}

func demoPlayer(snap snapshot.Snapshot) (string, playerid.PlayerID, error) {
	for _, r := range snap.Rosters.Value {
		for _, p := range r.Players {
			if p.RosterStatus != domain.RosterIR {
				return r.FranchiseID, p.ID, nil
			}
		}
	}
	return "", playerid.PlayerID{}, fmt.Errorf("fixtures: no rostered non-IR player")
}

func irSnapshot(snap snapshot.Snapshot, fid string, pid playerid.PlayerID, at time.Time) snapshot.Snapshot {
	snap.Rosters.Value = append([]snapshot.Roster{}, snap.Rosters.Value...)
	for i, r := range snap.Rosters.Value {
		if r.FranchiseID != fid {
			continue
		}
		snap.Rosters.Value[i].Players = append([]snapshot.RosterPlayer{}, r.Players...)
		for j, p := range r.Players {
			if p.ID == pid {
				snap.Rosters.Value[i].Players[j].RosterStatus = domain.RosterIR
			}
		}
	}
	snap.Rosters.Provenance.Freshness.FetchedAt = at.UTC().Format(time.RFC3339)
	return snap
}

func writeEnvelopeDemo(out string, snap snapshot.Snapshot) error {
	d, err := demo(snap)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(out, "envelope-demo.json"), d)
}

func writeClock(out string, clock snapshot.Sourced[leagueclock.Reading]) error {
	return writeJSON(filepath.Join(out, "clock.json"), clock)
}

func writeJSON[T any](path string, value T) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("fixtures: encode %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("fixtures: write %s: %w", filepath.Base(path), err)
	}
	return nil
}
