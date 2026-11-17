package leagueservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

const (
	TradeDraftRead   = "trade.draft.read"
	TradeDraftSave   = "trade.draft.save"
	TradeDraftDelete = "trade.draft.delete"
)

type RawTradeDraftScope struct {
	League string `json:"league"`
}

func (r RawTradeDraftScope) Validate() error {
	if !validDraftQueueLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

type RawTradeDraftSave struct {
	League  string   `json:"league"`
	Partner string   `json:"partner"`
	Give    []string `json:"give"`
	Get     []string `json:"get"`
	Label   string   `json:"label,omitempty"`
}

func (r RawTradeDraftSave) Validate() error {
	if err := (RawTradeDraftScope{League: r.League}).Validate(); err != nil {
		return err
	}
	if !validTradeDraftFranchise(r.Partner) {
		return fmt.Errorf("choose a valid trade partner")
	}
	if len(r.Label) > 160 || strings.IndexByte(r.Label, 0) >= 0 {
		return fmt.Errorf("draft label must be under 161 bytes")
	}
	seen := make(map[string]bool, len(r.Give)+len(r.Get))
	for _, group := range [][]string{r.Give, r.Get} {
		for _, asset := range group {
			if !validTradeDraftAsset(asset) {
				return fmt.Errorf("use valid player or pick ids in the proposal")
			}
			if seen[asset] {
				return fmt.Errorf("list each proposal asset once")
			}
			seen[asset] = true
		}
	}
	if len(seen) == 0 {
		return fmt.Errorf("choose at least one proposal asset")
	}
	encoded, err := json.Marshal(TradeDraftProposal{Partner: r.Partner, Give: r.Give, Get: r.Get, Label: r.Label})
	if err != nil {
		return fmt.Errorf("encode saved proposal: %w", err)
	}
	if len(encoded) >= personal.MaxTradeProposalBytes {
		return fmt.Errorf("saved proposal must be under 16 KB")
	}
	return nil
}

type RawTradeDraftDelete struct {
	League string `json:"league"`
	ID     string `json:"id"`
}

func (r RawTradeDraftDelete) Validate() error {
	if err := (RawTradeDraftScope{League: r.League}).Validate(); err != nil {
		return err
	}
	if !validTradeDraftID(r.ID) {
		return fmt.Errorf("choose a saved proposal")
	}
	return nil
}

type TradeDraftProposal struct {
	Partner string   `json:"partner"`
	Give    []string `json:"give"`
	Get     []string `json:"get"`
	Label   string   `json:"label,omitempty"`
}

type SavedTradeDraft struct {
	ID string `json:"id"`
	TradeDraftProposal
	At string `json:"at"`
}

type TradeDraftList struct {
	Drafts []SavedTradeDraft `json:"drafts"`
}

type TradeDraftDeleted struct {
	ID string `json:"id"`
}

// TargetTradeDrafts lists only the authenticated seated owner's saved proposals.
func (a *Service) TargetTradeDrafts(ctx context.Context, raw RawTradeDraftScope) (TradeDraftList, error) {
	if err := raw.Validate(); err != nil {
		return TradeDraftList{}, fmt.Errorf("trade draft scope: %w", err)
	}
	ctx, owner, store, cancel, err := a.tradeDraftAccess(ctx, raw.League, TradeDraftRead, "read")
	if err != nil {
		return TradeDraftList{}, err
	}
	defer cancel()
	entries, err := store.TradeProposalsForFranchise(ctx, owner)
	if err != nil {
		return TradeDraftList{}, fmt.Errorf("read saved trade proposals: %w", err)
	}
	return decodeTradeDraftEntries(entries)
}

// SaveTradeDraft writes the authenticated owner's current proposal; it is not a trade offer.
func (a *Service) SaveTradeDraft(ctx context.Context, raw RawTradeDraftSave) (SavedTradeDraft, error) {
	if err := raw.Validate(); err != nil {
		return SavedTradeDraft{}, fmt.Errorf("trade draft input: %w", err)
	}
	ctx, owner, store, cancel, err := a.tradeDraftAccess(ctx, raw.League, TradeDraftSave, "execute")
	if err != nil {
		return SavedTradeDraft{}, err
	}
	defer cancel()
	if raw.Partner == owner {
		return SavedTradeDraft{}, fmt.Errorf("choose a different trade partner")
	}
	if err := a.tradeDraftPartner(ctx, raw.League, raw.Partner); err != nil {
		return SavedTradeDraft{}, err
	}
	proposal := TradeDraftProposal{Partner: raw.Partner, Give: raw.Give, Get: raw.Get, Label: raw.Label}
	encoded, err := json.Marshal(proposal)
	if err != nil {
		return SavedTradeDraft{}, fmt.Errorf("encode saved trade proposal: %w", err)
	}
	id, err := newTradeDraftID()
	if err != nil {
		return SavedTradeDraft{}, err
	}
	at := a.host.CommissionerNow().UTC()
	if err := store.SaveTradeProposalForFranchise(ctx, owner, id, string(encoded), at); err != nil {
		return SavedTradeDraft{}, fmt.Errorf("save private trade proposal: %w", err)
	}
	return SavedTradeDraft{ID: id, TradeDraftProposal: proposal, At: at.Format(time.RFC3339Nano)}, nil
}

// DeleteTradeDraft removes only a saved proposal belonging to the authenticated owner.
func (a *Service) DeleteTradeDraft(ctx context.Context, raw RawTradeDraftDelete) (TradeDraftDeleted, error) {
	if err := raw.Validate(); err != nil {
		return TradeDraftDeleted{}, fmt.Errorf("trade draft deletion: %w", err)
	}
	ctx, owner, store, cancel, err := a.tradeDraftAccess(ctx, raw.League, TradeDraftDelete, "execute")
	if err != nil {
		return TradeDraftDeleted{}, err
	}
	defer cancel()
	if err := store.DeleteTradeProposalForFranchise(ctx, owner, raw.ID); err != nil {
		return TradeDraftDeleted{}, fmt.Errorf("delete private trade proposal: %w", err)
	}
	return TradeDraftDeleted{ID: raw.ID}, nil
}

func (a *Service) tradeDraftAccess(requestCtx context.Context, league, operation, mode string) (
	context.Context, string, *personal.Store, context.CancelFunc, error) {
	if !a.hosted {
		return nil, "", nil, nil, fmt.Errorf("saved trade proposals require the hosted league")
	}
	return a.playerPrivateAccess(requestCtx, league, operation, mode)
}

func (a *Service) tradeDraftPartner(ctx context.Context, league, partner string) error {
	records, err := a.host.CommissionerRecords(ctx, league)
	if err != nil {
		return fmt.Errorf("trade draft partner: %w", err)
	}
	for _, id := range commissioner.LeagueFranchises(records) {
		if id == partner {
			return nil
		}
	}
	return fmt.Errorf("choose a franchise in this league")
}

func decodeTradeDraftEntries(entries []personal.Entry) (TradeDraftList, error) {
	drafts := make([]SavedTradeDraft, 0, len(entries))
	for _, entry := range entries {
		var proposal TradeDraftProposal
		if err := json.Unmarshal([]byte(entry.Text), &proposal); err != nil {
			return TradeDraftList{}, fmt.Errorf("decode saved trade proposal: %w", err)
		}
		drafts = append(drafts, SavedTradeDraft{ID: entry.ID, TradeDraftProposal: proposal, At: entry.At})
	}
	return TradeDraftList{Drafts: drafts}, nil
}

func newTradeDraftID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate saved proposal id: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

func validTradeDraftID(value string) bool {
	if len(value) != 32 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validTradeDraftFranchise(value string) bool {
	return value != "" && len(value) <= 128 && strings.TrimSpace(value) == value &&
		strings.IndexByte(value, 0) < 0 && !strings.Contains(value, ":")
}

func validTradeDraftAsset(value string) bool {
	if player, err := playerid.New(value); err == nil && player.String() == value {
		return true
	}
	parts := strings.Split(value, ":")
	if len(parts) != 4 || parts[0] != "pick" || !validTradeDraftFranchise(parts[3]) {
		return false
	}
	season, seasonErr := strconv.Atoi(parts[1])
	round, roundErr := strconv.Atoi(parts[2])
	return seasonErr == nil && season >= 1 && season <= 9999 && roundErr == nil && round >= 1
}
