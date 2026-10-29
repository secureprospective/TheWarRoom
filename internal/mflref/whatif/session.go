// Package mflwhatif persists frozen MFL simulations without changing the source mirror.
package mflwhatif

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type WhatIfMove struct {
	ID          string       `json:"id"`
	Kind        string       `json:"kind"`
	Player      string       `json:"player"`
	Name        string       `json:"name"`
	Charge      domain.Money `json:"charge"`
	Explanation string       `json:"explanation"`
}

type WhatIfSession struct {
	Generation  string               `json:"generation"`
	Mine        string               `json:"mine"`
	Stamps      state.MirrorStamps   `json:"stamps"`
	Snapshot    state.MirrorSnapshot `json:"snapshot"`
	SeedUsed    domain.Money         `json:"seedUsed"`
	TaxiPercent float64              `json:"taxiPercent"`
	IRPercent   float64              `json:"irPercent"`
	Used        domain.Money         `json:"used"`
	Cap         *domain.Money        `json:"cap"`
	Moves       []WhatIfMove         `json:"moves"`
}

// WhatIfSessions publishes one complete session per franchise. Failed replays leave the
// previous session intact; generation state is isolated from the legacy transaction screens.
type WhatIfSessions struct{ pools *db.Pools }

func NewWhatIfSessions(pools *db.Pools) *WhatIfSessions { return &WhatIfSessions{pools: pools} }

func (s *WhatIfSessions) Load(ctx context.Context, mine string) (WhatIfSession, bool, error) {
	if err := state.InitializeWhatIf(ctx, s.pools); err != nil {
		return WhatIfSession{}, false, fmt.Errorf("what-if schema: %w", err)
	}
	raw, found, err := state.NewWhatIfRecords(s.pools).Load(ctx, mine)
	if err != nil {
		return WhatIfSession{}, false, fmt.Errorf("what-if load: %w", err)
	}
	if !found {
		return WhatIfSession{}, false, nil
	}
	var out WhatIfSession
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return out, false, fmt.Errorf("what-if decode: %w", err)
	}
	return out, true, nil
}
func (s *WhatIfSessions) Save(ctx context.Context, session WhatIfSession) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("what-if encode: %w", err)
	}
	if err := state.NewWhatIfRecords(s.pools).Save(ctx, session.Mine, string(raw)); err != nil {
		return fmt.Errorf("what-if save: %w", err)
	}
	return nil
}

func (s WhatIfSession) TaxiCapPercent() float64 { return s.TaxiPercent }
func (s WhatIfSession) IRCapPercent() float64   { return s.IRPercent }
