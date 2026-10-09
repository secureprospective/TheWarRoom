package envelope

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type ExpectedTrade struct {
	TradeID        string   `json:"tradeId"`
	Offering       string   `json:"offering"`
	Accepting      string   `json:"accepting"`
	OfferingGives  []string `json:"offeringGives"`
	AcceptingGives []string `json:"acceptingGives"`
}

func tradeSubject(t ExpectedTrade) (Subject, error) {
	out := Subject{Players: []playerid.PlayerID{}, Picks: []string{}}
	for _, token := range append(slices.Clone(t.OfferingGives), t.AcceptingGives...) {
		assets, err := leaguefeed.ParseAssets(token)
		if err != nil {
			return Subject{}, fmt.Errorf("trade subject: %w", err)
		}
		if len(assets) != 1 || assets[0].BlindBidCents != nil {
			return Subject{}, fmt.Errorf("trade subject: player or pick required")
		}
		if assets[0].Player != nil {
			out.Players = append(out.Players, *assets[0].Player)
		} else {
			out.Picks = append(out.Picks, leaguefeed.AssetToken(assets[0]))
		}
	}
	return out, nil
}

func validateTradeExpected(s Spec) error {
	t := s.Expected.Trade
	if t == nil || !validTradeScope(*t, s.FranchiseID) ||
		s.Expected.Lineup != nil || !s.Expected.Player.IsZero() || s.Expected.RosterStatus != "" {
		return fmt.Errorf("envelope: trade requires both sides and accepting franchise, without other changes")
	}
	subject, err := tradeSubject(*t)
	if err != nil {
		return fmt.Errorf("envelope: %w", err)
	}
	if !samePlayers(subject.Players, s.Subject.Players) || !sameTokens(subject.Picks, s.Subject.Picks) ||
		len(subject.Players) != len(s.Subject.Players) || len(subject.Picks) != len(s.Subject.Picks) {
		return fmt.Errorf("envelope: trade subjects must equal both full asset sets")
	}
	return nil
}

func validTradeScope(t ExpectedTrade, franchiseID string) bool {
	return t.TradeID != "" && t.Offering != "" && t.Accepting != "" &&
		t.Offering != t.Accepting && franchiseID == t.Accepting &&
		len(t.OfferingGives) > 0 && len(t.AcceptingGives) > 0
}

func canonicalTokens(tokens []string) ([]string, error) {
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		assets, err := leaguefeed.ParseAssets(token)
		if err != nil {
			return nil, fmt.Errorf("canonical trade token: %w", err)
		}
		if len(assets) != 1 {
			return nil, fmt.Errorf("canonical trade token: one asset required")
		}
		out = append(out, leaguefeed.AssetToken(assets[0]))
	}
	return out, nil
}

func sameTokens(a, b []string) bool {
	left, err := canonicalTokens(a)
	if err != nil {
		return false
	}
	right, err := canonicalTokens(b)
	if err != nil {
		return false
	}
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(slices.Compact(left), slices.Compact(right))
}

// TradeFeedReason distinguishes a failed held copy from an aged but successfully fetched feed.
func TradeFeedReason(p snapshot.Provenance) string {
	f := p.Freshness
	if f.State != domain.FreshFail && (f.State != domain.FreshStale || f.Note == "") {
		return ""
	}
	if f.Note != "" {
		return f.Note
	}
	return "feed failed"
}

type TradeCheck struct {
	PendingTrades snapshot.Sourced[[]leaguefeed.PendingTrade]
	At            time.Time
	OfferingName  string
	Blocks        []string
}

func (c TradeCheck) Evaluate(s Spec, snap snapshot.Snapshot) CheckResult {
	if s.Intent != "trade.accept" || s.Expected.Trade == nil {
		return CheckResult{Blocked: true, Note: "trade check does not cover this intent"}
	}
	if reason := TradeFeedReason(c.PendingTrades.Provenance); reason != "" {
		return CheckResult{Blocked: true, Note: "pending trades unavailable: " + reason}
	}
	t := s.Expected.Trade
	for _, offer := range c.PendingTrades.Value {
		if offer.ID != t.TradeID {
			continue
		}
		return c.checkOffer(s, snap, offer)
	}
	return CheckResult{Blocked: true, Note: "offer " + t.TradeID + " is no longer pending"}
}

func (c TradeCheck) checkOffer(s Spec, snap snapshot.Snapshot, offer leaguefeed.PendingTrade) CheckResult {
	blocks := slices.Clone(c.Blocks)
	switch {
	case s.FranchiseID == offer.Offering:
		blocks = append(blocks, "you made this offer; accept is the other owner's")
	case s.FranchiseID != offer.OfferedTo:
		blocks = append(blocks, "offer is not to this franchise")
	case !offer.Expires.IsZero() && !c.At.Before(offer.Expires):
		blocks = append(blocks, "offer "+offer.ID+" expired")
	}
	t := s.Expected.Trade
	if t.Offering != offer.Offering || t.Accepting != offer.OfferedTo ||
		!sameTokens(t.OfferingGives, leaguefeed.AssetTokens(offer.Gives)) ||
		!sameTokens(t.AcceptingGives, leaguefeed.AssetTokens(offer.Gets)) {
		blocks = append(blocks, "pending offer differs from drafted trade")
	}
	blocks = append(blocks, tradeOwnership(s, snap, offer.Gets)...)
	if len(blocks) > 0 {
		return CheckResult{Blocked: true, Note: strings.Join(blocks, "; ")}
	}
	note := "offer " + offer.ID + " from " + c.OfferingName
	if slices.ContainsFunc(offer.Gets, func(a leaguefeed.Asset) bool { return a.Player == nil }) {
		note += "; MFL checks pick ownership"
	}
	return CheckResult{Note: note}
}

func tradeOwnership(s Spec, snap snapshot.Snapshot, assets []leaguefeed.Asset) []string {
	if reason := TradeFeedReason(snap.Rosters.Provenance); reason != "" {
		return []string{"roster unavailable: " + reason}
	}
	owned := make(map[string]bool)
	for _, roster := range snap.Rosters.Value {
		if roster.FranchiseID == s.FranchiseID {
			for _, p := range roster.Players {
				owned[p.ID.String()] = true
			}
		}
	}
	blocks := []string{}
	for _, asset := range assets {
		token := leaguefeed.AssetToken(asset)
		// A pending offer is MFL's own record and MFL owns pick ownership; only players are
		// checked here, against a roster that may have changed since the offer was made.
		if asset.Player != nil && !owned[token] {
			blocks = append(blocks, "asset "+token+" is not on this franchise's roster")
		}
	}
	return blocks
}

// tradeReviewWindow follows MFL league.defaultTradeExpirationDays = 7 for this league
// (docs/build-handoffs/Ring1_Gap_Closure.md); disappearance alone cannot prove execution.
const tradeReviewWindow = 7 * 24 * time.Hour

func (e Envelope) tradeReviewVerdict(observed time.Time, verdict Verdict) Verdict {
	if e.spec.Intent != "trade.accept" || e.state != DOTReview || verdict.Event != AwaitDOT ||
		!e.tradeReviewExpired(observed) {
		return verdict
	}
	return Verdict{Event: Partial,
		Note: "Not verified: no executed trade 7 days after acceptance " +
			"(declined, vetoed or still in review on MFL)"}
}

// tradeReviewExpired reports whether the window from the first entry into DOTReview has passed.
// The first entry anchors it, so a stale read that bounces through Not verified cannot extend it;
// an envelope never in DOTReview has no window, so a stale-read Not verified can still enter it.
func (e Envelope) tradeReviewExpired(observed time.Time) bool {
	for _, entry := range e.audit {
		if entry.To == DOTReview {
			return observed.After(entry.At.Add(tradeReviewWindow))
		}
	}
	return false
}

type TradePredicate struct{}

func (TradePredicate) Sources() []Source { return []Source{Transactions, PendingTrades} }

func tradeObservationReason(s Spec, obs Observation) string {
	reason := ""
	switch {
	case s.Intent != "trade.accept" || s.Expected.Trade == nil:
		reason = "trade intent missing"
	case obs.LeagueID != s.LeagueID:
		reason = "wrong league"
	case obs.HandedOffAt.IsZero():
		reason = "hand-off time missing"
	case TradeFeedReason(obs.Transactions.Provenance) != "":
		reason = "transactions unavailable: " + TradeFeedReason(obs.Transactions.Provenance)
	case TradeFeedReason(obs.PendingTrades.Provenance) != "":
		reason = "pending trades unavailable: " + TradeFeedReason(obs.PendingTrades.Provenance)
	}
	return reason
}

func (TradePredicate) Evaluate(s Spec, obs Observation) Verdict {
	if reason := tradeObservationReason(s, obs); reason != "" {
		return Verdict{Event: Partial, Note: "Not verified: " + reason}
	}
	for _, row := range obs.Transactions.Value {
		if row.Kind == "TRADE" && row.Trade != nil && !row.Unparsed &&
			!row.Time.Before(obs.HandedOffAt) && matchesTrade(*s.Expected.Trade, row) {
			return Verdict{Event: Match, Note: "public TRADE matches both franchises and full asset sets"}
		}
	}
	for _, offer := range obs.PendingTrades.Value {
		if offer.ID == s.Expected.Trade.TradeID {
			return Verdict{Event: NoChange, Note: "offer still pending on MFL"}
		}
	}
	return Verdict{Event: AwaitDOT,
		Note: "offer left MFL's pending list; awaiting DOT and commissioner (or declined there)"}
}

func matchesTrade(t ExpectedTrade, row leaguefeed.Transaction) bool {
	other := row.Trade.Counterparty
	gave := leaguefeed.AssetTokens(row.Trade.Gave)
	got := leaguefeed.AssetTokens(row.Trade.CounterpartyGave)
	return (row.Franchise == t.Offering && other == t.Accepting &&
		sameTokens(gave, t.OfferingGives) && sameTokens(got, t.AcceptingGives)) ||
		(row.Franchise == t.Accepting && other == t.Offering &&
			sameTokens(gave, t.AcceptingGives) && sameTokens(got, t.OfferingGives))
}

type TradeRequest struct {
	LeagueID    string
	FranchiseID string
	Trade       ExpectedTrade
	Target      Target
	Deadline    *time.Time
}

func DraftTrade(
	at time.Time, req TradeRequest, snap snapshot.Snapshot, check TradeCheck, generate func() string,
) (Envelope, error) {
	subject, err := tradeSubject(req.Trade)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft trade: %w", err)
	}
	spec := Spec{
		Intent: "trade.accept", LeagueID: req.LeagueID, FranchiseID: req.FranchiseID,
		Subject: subject, Expected: ExpectedChange{Trade: &req.Trade},
		Gravity: G2, Undo: Irreversible, Target: req.Target, Deadline: req.Deadline,
	}
	e, err := New(at, spec, generate)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft trade: %w", err)
	}
	check.At = at
	checked, err := e.Check(at, snap, check)
	if err != nil {
		return Envelope{}, fmt.Errorf("draft trade: check: %w", err)
	}
	return checked, nil
}
