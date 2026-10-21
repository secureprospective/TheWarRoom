package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core/coretest"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestResearchSharedReadsAreDetached(t *testing.T) {
	s := coretest.New(t).Build()
	lookup, err := normalize.NewLookup([]players.RawPlayer{{ID: "0042", Name: "Held", Position: "QB"}})
	if err != nil {
		t.Fatal(err)
	}
	s.Lookup = lookup
	_, err = s.League.Replace(t.Context(), state.MirrorSnapshot{Season: 2026,
		Players: []state.MirrorPlayer{{MFLID: "0042", FranchiseID: "0001"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Rulebook.Initialize(t.Context(), researchRules{}); err != nil {
		t.Fatal(err)
	}
	roster, config, names := s.ResearchRoster(), s.ResearchConfig(), s.ResearchNames()
	roster.Players[0].FranchiseID = "0002"
	config.Franchises[0].Name = "changed"
	config.ScoringRules[0].Rules[0].Points = "changed"
	names["0001"] = "changed"
	if s.ResearchRoster().Players[0].FranchiseID != "0001" || s.ResearchNames()["0001"] != "Held" {
		t.Fatal("roster or names alias held values")
	}
	if s.ResearchConfig().ScoringRules[0].Rules[0].Points != "1" {
		t.Fatal("nested config alias")
	}
	directory := s.ResearchDirectory()
	if name, position, ok := directory.DisplayFacts("0042"); !ok || name.Name != "Held" || position != "QB" {
		t.Fatal(name, position, ok)
	}
	if s.ResearchZone() != time.UTC || s.ResearchContext() != s.Ctx {
		t.Fatal("context or zone changed")
	}
}

type researchRules struct{}

func (researchRules) Fetch(context.Context) (league.RawConfig, error) {
	return league.RawConfig{Source: "fixture", Franchises: []league.Franchise{{ID: "0001", Name: "Held"}},
		ScoringRules: []league.PositionRuleSet{{Positions: "QB", Rules: []league.ScoringRule{{
			Event: "PY", Points: "1",
		}}}}}, nil
}
