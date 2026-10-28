package commissioner

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// RawBidArgs names a recorded player or bid and the contract an owner offers. It never carries a decision,
// a price the app works out or an owner identity.
type RawBidArgs struct {
	Player    string `json:"player,omitempty"`
	Franchise string `json:"franchise,omitempty"`
	// Side names whose copy of the player has its bids decided, in a league with a pool per side.
	Side   string       `json:"side,omitempty"`
	Bid    string       `json:"bid,omitempty"`
	Salary domain.Money `json:"salaryCents,omitempty"`
	Years  int          `json:"years,omitempty"`
}

// BidResult is how one bid fared when its player's bids were decided.
type BidResult struct {
	ID      string `json:"id"`
	Refusal string `json:"refusal,omitempty"`
}

// BidOutcome is worked out by the app under the append lock; a request cannot supply it.
type BidOutcome struct {
	Player     string              `json:"player,omitempty"`
	Season     int                 `json:"season"`
	Points     leaguerules.Decimal `json:"points,omitempty"`
	Late       bool                `json:"late,omitempty"`
	Winner     string              `json:"winner,omitempty"`
	WinningBid string              `json:"winningBid,omitempty"`
	Tied       []string            `json:"tied,omitempty"`
	Results    []BidResult         `json:"results,omitempty"`
}

// Params replaces the action's parameters; omitted values explicitly need a ruling.
// On is separate so there cannot be two competing enable switches in one request.
type RawContractAction struct {
	Action string          `json:"action"`
	On     *bool           `json:"on"`
	Params json.RawMessage `json:"params"`
}

// TradeAssetDetail preserves the names and full contract offered, even if the player later leaves a roster.
type TradeAssetDetail struct {
	From     string        `json:"fromTeam"`
	To       string        `json:"toTeam"`
	Label    string        `json:"label"`
	Contract string        `json:"contract,omitempty"`
	Salary   *domain.Money `json:"yearlySalaryCents,omitempty"`
	Ends     int           `json:"contractEndsInSeason,omitempty"`
}

// TradeAsset identifies exactly one player or an unspent pick, moving between two parties.
type TradeAsset struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Player   string `json:"player,omitempty"`
	Season   int    `json:"season,omitempty"`
	Round    int    `json:"round,omitempty"`
	Original string `json:"original,omitempty"`
}

type RawTradeArgs struct {
	ID        string       `json:"id,omitempty"`
	Franchise string       `json:"franchise,omitempty"`
	Parties   []string     `json:"parties,omitempty"`
	Assets    []TradeAsset `json:"assets,omitempty"`
	Decision  string       `json:"decision,omitempty"`
}

type TradeVote struct {
	Who      string    `json:"who"`
	Decision string    `json:"decision"`
	At       time.Time `json:"at"`
	Reason   string    `json:"reason,omitempty"`
}

type TradeState struct {
	VetoNumber   int                `json:"vetoNumber"`
	AppealedVeto int                `json:"appealedVetoNumber"`
	Details      []TradeAssetDetail `json:"assetDetails"`
	Previous     string             `json:"previous,omitempty"`
	ID           string             `json:"id"`
	Proposer     string             `json:"proposer"`
	Parties      []string           `json:"parties"`
	Assets       []TradeAsset       `json:"assets"`
	Accepted     []string           `json:"accepted"`
	Policy       TradePolicy        `json:"policy"`
	OfferedAt    time.Time          `json:"offeredAt"`
	ExpiresAt    time.Time          `json:"expiresAt"`
	Status       string             `json:"status"`
	Words        string             `json:"words"`
	Votes        []TradeVote        `json:"votes"`
	Recusals     []TradeVote        `json:"recusals"`
	Appeals      []TradeVote        `json:"appeals"`
	Overrides    []TradeVote        `json:"overrides"`
}

type TradePolicy struct {
	Route          leaguerules.ReviewRoute `json:"route"`
	Seats          int                     `json:"seats"`
	DecideAt       int                     `json:"votesToDecide"`
	FutureYears    int                     `json:"futureYears"`
	MinimumRound   int                     `json:"minimumRound"`
	MaximumRound   int                     `json:"maximumRound"`
	RosterGate     bool                    `json:"refuseIllegalRoster"`
	Expiry         time.Duration           `json:"offerLifetimeNanoseconds"`
	MaximumParties int                     `json:"maximumParties"`
}

// RawWaiverArgs names recorded native assets, never prices or an owner identity.
type RawWaiverArgs struct {
	Bucket    string   `json:"bucket,omitempty"`
	Player    string   `json:"player,omitempty"`
	Franchise string   `json:"franchise,omitempty"`
	Window    string   `json:"window,omitempty"`
	Claim     string   `json:"claim,omitempty"`
	Order     []string `json:"order,omitempty"`
}

// WaiverOutcome is computed under the append lock; IPC cannot supply a decision.
type WaiverClaimResult struct {
	ID      string `json:"id"`
	Refusal string `json:"refusal,omitempty"`
}

type WaiverOutcome struct {
	Claims     []WaiverClaimResult `json:"claims,omitempty"`
	PriorOrder []string            `json:"priorOrder,omitempty"`
	CloseAt    time.Time           `json:"closeAt,omitempty"`
	Player     *NativePlayer       `json:"player,omitempty"`
	Dead       domain.Money        `json:"deadCents,omitempty"`
	Season     int                 `json:"season"`
	Winner     string              `json:"winner,omitempty"`
	Order      []string            `json:"order,omitempty"`
	Source     string              `json:"source,omitempty"`
	ClearsDead bool                `json:"clearsDead"`
}

func tradeLabel(r Record) string {
	labels := map[string]string{TradeOffer: "Offered a trade", TradeAccept: "Accepted a trade", TradeDecline: "Declined a trade",
		TradeWithdraw: "Withdrew a trade offer", TradeReview: "Reviewed a trade", TradeRecuse: "Sat out a trade review",
		TradeAppeal: "Appealed a trade veto", TradeAppealRule: "Ruled on a trade appeal", TradeOverride: "Overrode a trade decision"}
	label := labels[r.Command.Command]
	if r.Command.Reason != "" {
		label = fmt.Sprintf("%s: %s", label, r.Command.Reason)
	}
	return label
}
func (p TradePolicy) Validate(r leaguerules.TradesRules) error {
	if r.Rounds.Value.Min.State != leaguerules.IsKnown || r.Rounds.Value.Max.State != leaguerules.IsKnown ||
		p.invalidLimits() {
		return fmt.Errorf("the trade rules need a valid ruling")
	}
	if !slices.Contains([]leaguerules.ReviewRoute{leaguerules.DOTReview, leaguerules.CommissionerReview, leaguerules.NoReview}, p.Route) {
		return fmt.Errorf("choose who reviews trades")
	}
	if p.Route == leaguerules.DOTReview && (r.DOTSeats.State != leaguerules.IsKnown || p.Seats < 1 || p.DecideAt > p.Seats) {
		return fmt.Errorf("%s: choose at least one DOT seat and a decision count within it", "Needs a ruling")
	}
	return nil
}
func (p TradePolicy) invalidLimits() bool {
	return p.DecideAt < 1 || p.FutureYears < 0 || p.MinimumRound < 1 || p.MaximumRound < p.MinimumRound ||
		p.Expiry <= 0 || p.MaximumParties < 2
}

const ReserveClockPrincipal = "native.reserve.clock"
