// Package pendingtrades verifies authentication without inferring an owner's franchise.
package pendingtrades

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// ErrRejected marks MFL's own verdict against a key (error envelope, non-200, redirect, no
// pendingTrades object). Any other error means MFL could not be asked or answered unreadably.
var ErrRejected = errors.New("MFL rejected the key")

// Verified reports only what the export proves, not the identity of a trade participant.
type Verified struct {
	Franchise       string
	FranchiseProven bool
	Detail          string
}

// RawEnvelope is the documented pendingTrades object or MFL error envelope.
type RawEnvelope struct {
	PendingTrades json.RawMessage `json:"pendingTrades"`
	Error         *struct {
		Text string `json:"$t"`
	} `json:"error"`
}

func (r RawEnvelope) Validate() error {
	if r.Error != nil {
		return fmt.Errorf("%w: %s", ErrRejected, r.Error.Text)
	}
	if len(r.PendingTrades) == 0 || r.PendingTrades[0] != '{' {
		return fmt.Errorf("%w: response has no pendingTrades object", ErrRejected)
	}
	return nil
}

// Verify makes one keyed export; discovery is an unkeyed lookup only if not already cached.
func Verify(ctx context.Context, c *mfl.Client, year, leagueID string) (Verified, error) {
	if _, err := fetchEnvelope(ctx, c, year, leagueID); err != nil {
		return Verified{}, err
	}
	return Verified{Detail: "Export accepted; authenticated franchise not proved by this body"}, nil
}

func fetchEnvelope(ctx context.Context, c *mfl.Client, year, leagueID string) (RawEnvelope, error) {
	if err := c.DiscoverHost(ctx, year, leagueID); err != nil {
		return RawEnvelope{}, fmt.Errorf("pendingTrades: discover host: %w", err)
	}
	resp, err := c.Do(ctx, mfl.Request{
		Type: "pendingTrades", Year: year, Params: map[string]string{"L": leagueID}, Keyed: true,
	})
	if err != nil {
		return RawEnvelope{}, fmt.Errorf("pendingTrades: verify: %w", err)
	}
	return decodeEnvelope(resp.Body, resp.StatusCode)
}

func decodeEnvelope(body []byte, status int) (RawEnvelope, error) {
	var raw RawEnvelope
	decodeErr := json.Unmarshal(body, &raw)
	if status != http.StatusOK && raw.Error == nil {
		return RawEnvelope{}, fmt.Errorf("%w: HTTP status %d", ErrRejected, status)
	}
	if decodeErr != nil {
		return RawEnvelope{}, fmt.Errorf("pendingTrades: decode response: %w", decodeErr)
	}
	// Validate rejects an MFL error envelope as ErrRejected, the verdict Verify needs.
	if err := raw.Validate(); err != nil {
		return RawEnvelope{}, err
	}
	return raw, nil
}
