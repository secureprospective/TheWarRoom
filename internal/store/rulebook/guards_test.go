package rulebook

import (
	"context"
	"testing"
)

func TestReload_NoChangeEmptyChangeSet(t *testing.T) {
	src := &fakeSource{cfg: baseConfig()}
	s := newStore(t, src)

	cs, err := s.Reload(context.Background())
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if !cs.Empty() {
		t.Errorf("ChangeSet = %+v, want empty for identical config", cs.Deltas)
	}
}

func TestGetScoringRules_NoAlias(t *testing.T) {
	s := newStore(t, &fakeSource{cfg: baseConfig()})

	got := s.GetScoringRules()
	got[0].Rules[0].Points = "MUTATED"
	got[0].Positions = "WRECKED"

	again := s.GetScoringRules()
	if again[0].Rules[0].Points != "*0.5" || again[0].Positions != "CB|S" {
		t.Errorf("active config corrupted via returned slice: %+v", again[0])
	}
	if cfg := s.ActiveConfig(); cfg.ScoringRules[0].Rules[0].Points != "*0.5" {
		t.Errorf("ActiveConfig leaked an alias: %+v", cfg.ScoringRules[0])
	}
}
