package offers

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// RawBidOwner selects the owner's plans on this computer, not a hosted franchise grant.
type RawBidOwner struct {
	FranchiseID string `json:"franchiseId"`
}

func (r RawBidOwner) Validate() error {
	if err := ingestion.FeedFranchise(r.FranchiseID); err != nil {
		return fmt.Errorf("choose your franchise: %w", err)
	}
	return nil
}

type RawBidPlanScope struct {
	ID string `json:"id"`
}

func (r RawBidPlanScope) Validate() error {
	if r.ID == "" || len(r.ID) > 128 || strings.TrimSpace(r.ID) != r.ID {
		return fmt.Errorf("choose a saved bid plan")
	}
	return nil
}

func (r RawBidInput) Validate() error {
	_, err := r.entry()
	return err
}
