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
	bid := catalog.Shape(reflect.TypeFor[RawBidInput](), "RawBidInput.Validate")
	plan := catalog.Shape(reflect.TypeFor[RawBidPlanScope](), "RawBidPlanScope.Validate")
	revise := bid
	revise.Fields = append([]catalog.Field{{Name: "id", Schema: catalog.Schema{Type: "string"}}}, bid.Fields...)
	revise.Validator = "RawBidPlanScope.Validate+RawBidInput.Validate"
	return []catalog.Command{
		mflCommand("mfl.pool.read", "TargetPool", "G1", catalog.Read,
			catalog.Shape(reflect.TypeFor[RawPoolScope](), "RawPoolScope.Validate"), "none", "held-free-agents,player-directory"),
		mflCommand("mfl.pool.refresh", "TargetPoolRefresh", "G1", catalog.Execute,
			catalog.Schema{Type: "object", Validator: "boolean force", Fields: []catalog.Field{{Name: "force", Schema: catalog.Schema{Type: "boolean"}}}}, "refresh-again", "read-only-MFL-freeAgents"),
		mflCommand("mfl.bid.plans", "TargetBidPlans", "G1", catalog.Read,
			catalog.Shape(reflect.TypeFor[RawBidOwner](), "RawBidOwner.Validate"), "none", "local-bid-plans"),
		mflCommand("mfl.bid.preview", "TargetBidPreview", "G1", catalog.Preview, bid, "none", "held-roster,player-directory,local-bid-plans"),
		mflCommand("bid.submit", "TargetBidDraft", "G1", catalog.Execute, bid, "withdraw-local-plan", "held-roster,player-directory,local-bid-plans"),
		mflCommand("mfl.bid.revise", "TargetBidRevise", "G1", catalog.Execute, revise, "revise-local-plan", "held-roster,player-directory,local-bid-plans"),
		mflCommand("mfl.bid.handoff", "TargetBidHandOff", "G2", catalog.Execute, plan, "withdraw-local-plan", "current-bid-rules,local-bid-plan"),
		mflCommand("mfl.bid.posted", "TargetBidPosted", "G2", catalog.Execute, plan, "withdraw-local-plan", "owner-statement,local-bid-plan"),
		mflCommand("mfl.bid.withdraw", "TargetBidWithdraw", "G2", catalog.Execute, plan, "draft-new-local-plan", "local-bid-plan"),
	}
}

func mflCommand(id, method, gravity string, mode catalog.Mode, args catalog.Schema, undo, facts string) catalog.Command {
	safeguards := "Raw-validation+local-plan-state; no MFL submissions"
	if gravity == "G2" {
		safeguards += "; explicit-owner-action"
	}
	return catalog.Command{ID: id, Method: method, Version: "1", Role: "gm,commish,admin", Gravity: gravity,
		Availability: "desktop", Arguments: args, Permission: "local-desktop-owner", Modes: []catalog.Mode{mode},
		Undo: undo, Facts: facts, Safeguards: safeguards}
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
