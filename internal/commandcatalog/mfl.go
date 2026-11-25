package catalog

// DesktopCommands preserves the solo-owner MFL contracts/trades catalog views while internal/app/mflref stays.
func DesktopCommands(feature string, _ []Command) []Command {
	switch feature {
	case "contracts":
		return contractsDesktop()
	case "trades":
		return tradesDesktop()
	default:
		return nil
	}
}

func contractsDesktop() []Command {
	return []Command{
		{ID: Unset, Method: "TargetAcquisition", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetCheckLineup", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetContractActs", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "both"},
		{ID: Unset, Method: "TargetContracts", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: "mfl.lineup.set", Method: "TargetDraftLineup", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetLineup", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetPlayerContract", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetReleasePlans", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "desktop"},
		{ID: Unset, Method: "TargetRosterReports", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetWhatIf", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
	}
}

func tradesDesktop() []Command {
	return []Command{
		{ID: "mfl.trade.accept", Method: "TargetDraftTradeAccept", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "both"},
		{ID: Unset, Method: "TargetTradeBuilder", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "desktop"},
		{ID: Unset, Method: "TargetTradeBuilderPreview", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "desktop", Modes: []Mode{Preview}},
		{ID: Unset, Method: "TargetTradeBuilderRead", Version: Unset, Role: "gm,commish", Gravity: "G0", Availability: "desktop", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetTrades", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
	}
}

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

// MFLPlanCommands names the existing lineup and incoming-offer plans without granting native authority.
func MFLPlanCommands() []Command {
	text := Schema{Type: "string"}
	lineup := Schema{Type: "object", Validator: "TargetDraftLineup owner and starter checks", Fields: []Field{
		{Name: "franchiseId", Schema: text},
		{Name: "starters", Schema: Schema{Type: "array", Items: &text}},
	}}
	trade := Schema{Type: "object", Validator: "TargetDraftTradeAccept owner and pending-offer checks", Fields: []Field{
		{Name: "franchiseId", Schema: text}, {Name: "tradeId", Schema: text},
	}}
	commands := []Command{
		mflCommand("mfl.lineup.set", "TargetDraftLineup", "G1", Execute, lineup, "withdraw-local-plan", "held-lineup,local-plans"),
		mflCommand("mfl.trade.accept", "TargetDraftTradeAccept", "G2", Execute, trade, "withdraw-local-plan", "held-pending-trades,local-plans"),
	}
	for i := range commands {
		commands[i].Role = "gm,commish"
	}
	return commands
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
