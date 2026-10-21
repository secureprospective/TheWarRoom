package core

import (
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// PendingTrades copies the held pendingTrades feed under SeasonMu. The value is never nil, as
// before, and every offer's assets are detached from the held feed.
func (s *State) PendingTrades() snapshot.Sourced[[]leaguefeed.PendingTrade] {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	held := s.SeasonPendingTrades
	out := make([]leaguefeed.PendingTrade, len(held.Value))
	for i, t := range held.Value {
		t.Gives, t.Gets = cloneAssets(t.Gives), cloneAssets(t.Gets)
		out[i] = t
	}
	return snapshot.Sourced[[]leaguefeed.PendingTrade]{
		Value:      out,
		Provenance: held.Provenance("pendingTrades", "pending trades"),
	}
}

// TradeFuturePicks copies the held futureDraftPicks value under SeasonMu, keeping nil nil.
func (s *State) TradeFuturePicks() []draftreports.FuturePick {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return slices.Clone(s.SeasonFuturePicks.Value)
}

// TradeFuturePicksFeed reads when futureDraftPicks was last fetched and its last refresh error.
func (s *State) TradeFuturePicksFeed() (time.Time, string) {
	s.SeasonMu.Lock()
	defer s.SeasonMu.Unlock()
	return s.SeasonFuturePicks.FetchedAt, s.SeasonFuturePicks.RefreshError
}

// TradeRequest is the local plan's request for offer, mapped to MFL's trade desk (O=05) once the
// league host is known. MFLClient and Season are set at startup and never change.
func (s *State) TradeRequest(franchiseID string, offer leaguefeed.PendingTrade) envelope.TradeRequest {
	req := envelope.TradeRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID,
		Trade: envelope.ExpectedTrade{
			TradeID: offer.ID, Offering: offer.Offering, Accepting: offer.OfferedTo,
			OfferingGives: leaguefeed.AssetTokens(offer.Gives), AcceptingGives: leaguefeed.AssetTokens(offer.Gets),
		},
		Target: envelope.Target{Kind: envelope.Unmapped},
	}
	if !offer.Expires.IsZero() {
		req.Deadline = &offer.Expires
	}
	if s.MFLClient != nil && s.MFLClient.LeagueHost() != "" {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/options?L=%s&O=05", s.MFLClient.LeagueHost(), s.Season,
			url.QueryEscape(req.LeagueID))}
	}
	return req
}
