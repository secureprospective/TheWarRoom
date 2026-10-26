package offers

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// MFLCommands describes the solo owner's local desktop plans, not hosted league actions.
// Desktop custody is the permission boundary; these declarations grant no remote access.
func MFLCommands() []catalog.Command {
	return catalog.MFLCommands(catalog.MFLShapes{
		Bid:   catalog.Shape(reflect.TypeFor[RawBidInput](), "RawBidInput.Validate"),
		Plan:  catalog.Shape(reflect.TypeFor[RawBidPlanScope](), "RawBidPlanScope.Validate"),
		Pool:  catalog.Shape(reflect.TypeFor[RawPoolScope](), "RawPoolScope.Validate"),
		Owner: catalog.Shape(reflect.TypeFor[RawBidOwner](), "RawBidOwner.Validate"),
	})
}

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
