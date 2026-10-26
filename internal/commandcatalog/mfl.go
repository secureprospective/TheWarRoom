package catalog

type MFLShapes struct{ Bid, Plan, Pool, Owner Schema }

func MFLCommands(shapes MFLShapes) []Command {
	bid := shapes.Bid
	plan := shapes.Plan
	revise := bid
	revise.Fields = append([]Field{{Name: "id", Schema: Schema{Type: "string"}}}, bid.Fields...)
	revise.Validator = "RawBidPlanScope.Validate+RawBidInput.Validate"
	return []Command{
		mflCommand("mfl.pool.read", "TargetPool", "G1", Read,
			shapes.Pool, "none", "held-free-agents,player-directory"),
		mflCommand("mfl.pool.refresh", "TargetPoolRefresh", "G1", Execute,
			Schema{Type: "object", Validator: "boolean force", Fields: []Field{{Name: "force", Schema: Schema{Type: "boolean"}}}}, "refresh-again", "read-only-MFL-freeAgents"),
		mflCommand("mfl.bid.plans", "TargetBidPlans", "G1", Read,
			shapes.Owner, "none", "local-bid-plans"),
		mflCommand("mfl.bid.preview", "TargetBidPreview", "G1", Preview, bid, "none", "held-roster,player-directory,local-bid-plans"),
		mflCommand("mfl.bid.submit", "TargetBidDraft", "G1", Execute, bid, "withdraw-local-plan", "held-roster,player-directory,local-bid-plans"),
		mflCommand("mfl.bid.revise", "TargetBidRevise", "G1", Execute, revise, "revise-local-plan", "held-roster,player-directory,local-bid-plans"),
		mflCommand("mfl.bid.handoff", "TargetBidHandOff", "G2", Execute, plan, "withdraw-local-plan", "current-bid-rules,local-bid-plan"),
		mflCommand("mfl.bid.posted", "TargetBidPosted", "G2", Execute, plan, "withdraw-local-plan", "owner-statement,local-bid-plan"),
		mflCommand("mfl.bid.withdraw", "TargetBidWithdraw", "G2", Execute, plan, "draft-new-local-plan", "local-bid-plan"),
	}
}
func mflCommand(id, method, gravity string, mode Mode, args Schema, undo, facts string) Command {
	safeguards := "Raw-validation+local-plan-state; no MFL submissions"
	if gravity == "G2" {
		safeguards += "; explicit-owner-action"
	}
	return Command{ID: id, Method: method, Version: "1", Role: "gm,commish,admin", Gravity: gravity,
		Availability: "desktop", Arguments: args, Permission: "local-desktop-owner", Modes: []Mode{mode},
		Undo: undo, Facts: facts, Safeguards: safeguards}
}
