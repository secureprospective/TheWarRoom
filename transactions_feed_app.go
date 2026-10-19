package main

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// FeedEventDTO is one activity-feed event with display names resolved server-side. StableKey
// is the React key. Money is not carried: Reason holds the audit label.
type FeedEventDTO struct {
	StableKey      string   `json:"stableKey"`
	Source         string   `json:"source"`
	ID             string   `json:"id"`
	Kind           string   `json:"kind"`
	Timestamp      string   `json:"timestamp"`
	MFLID          string   `json:"mflID"`
	PlayerName     string   `json:"playerName"`
	PlayerPosition string   `json:"playerPosition"`
	PlayerUnknown  bool     `json:"playerUnknown"`
	FranchiseIDs   []string `json:"franchiseIDs"`
	FranchiseNames []string `json:"franchiseNames"`
	Reason         string   `json:"reason"`
	Provenance     string   `json:"provenance"`
	TradeRationale string   `json:"tradeRationale,omitempty"`
	TradePicksNote string   `json:"tradePicksNote,omitempty"`
	TxID           string   `json:"txID"`
}

// FeedResult carries the events, newest first. A directory failure doesn't fail the read: the
// feed renders ids and DirectoryWarning says why.
type FeedResult struct {
	OK               bool           `json:"ok"`
	Events           []FeedEventDTO `json:"events"`
	Detail           string         `json:"detail"`
	DirectoryWarning string         `json:"directoryWarning,omitempty"`
}

// GetFeed reads recent events across the append-only ledgers, newest first. A player id that no
// longer resolves (a commissioner-created id later replaced by MFL's) shows as PlayerUnknown with
// the raw id: the history keeps the id that was live at the time.
func (a *App) GetFeed() FeedResult {
	if err := a.ready(); err != nil {
		return FeedResult{Detail: err.Error()}
	}
	if err := a.whatif.Err(); err != nil {
		return FeedResult{Detail: "state is stale after a failed reload: " + err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	events, err := a.whatif.Feed(ctx, 0) // 0 = the store's default limit
	if err != nil {
		return FeedResult{Detail: err.Error()}
	}

	dir, dirWarning := a.resolveDirectory(ctx)
	var franchiseNames map[string]string
	if a.rulebook != nil {
		franchiseNames = a.rulebook.FranchiseNames()
	}

	out := make([]FeedEventDTO, len(events))
	for i, e := range events {
		out[i] = feedEventDTO(e, dir, franchiseNames)
	}
	res := FeedResult{OK: true, Events: out}
	if dirWarning != "" {
		res.DirectoryWarning = dirWarning
	}
	return res
}

// feedEventDTO builds one event with player and franchise names joined.
func feedEventDTO(e state.FeedEvent, dir normalize.Lookup, franchiseNames map[string]string) FeedEventDTO {
	txID := e.Source + ":" + e.ID
	dto := FeedEventDTO{
		StableKey:      txID,
		Source:         e.Source,
		ID:             e.ID,
		Kind:           e.Kind,
		Timestamp:      e.Timestamp,
		MFLID:          e.MFLID,
		FranchiseIDs:   e.FranchiseIDs,
		FranchiseNames: resolveFranchiseNames(e.FranchiseIDs, franchiseNames),
		Reason:         e.Reason,
		Provenance:     e.Provenance,
		TradeRationale: e.TradeRationale,
		TradePicksNote: e.TradePicksNote,
		TxID:           txID,
	}
	// A stale commissioner-created id resolves ok=false and shows as PlayerUnknown, not an error.
	if e.MFLID != "" {
		if facts, ok := dir.Facts(e.MFLID); ok {
			dto.PlayerName = facts.Name
			dto.PlayerPosition = string(facts.Position)
		} else {
			dto.PlayerUnknown = true
		}
	}
	return dto
}

// resolveFranchiseNames labels each id. Nil for no ids, so the JSON is [].
func resolveFranchiseNames(ids []string, names map[string]string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = domain.FranchiseLabel(names, id)
	}
	return out
}
