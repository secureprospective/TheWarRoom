package catalog

// DesktopCommands is the legacy binding view; availability describes the old adapter, not remote admission.
func DesktopCommands(feature string, related []Command) []Command {
	views := map[string]func([]Command) []Command{"archive": archiveDesktop, "commissioner": commissionerDesktop, "comms": commsDesktop, "contracts": contractsDesktop, "control": controlDesktop, "counterparties": counterpartiesDesktop, "draft": draftDesktop, "market": marketDesktop, "now": nowDesktop, "offers": offersDesktop, "platform": platformDesktop, "race": raceDesktop, "research": researchDesktop, "sources": sourcesDesktop, "trades": tradesDesktop}
	if view, ok := views[feature]; ok {
		return view(related)
	}
	return nil
}

// heldRead marks an audited legacy adapter as a held reading; the shared frontend read catalog
// is generated from these marks (frontend/scripts/nativegen/reads.go).
func heldRead(c Command) Command {
	c.Modes = []Mode{Read}
	return c
}

func archiveDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetArchive", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
	}
}

func commissionerDesktop(related []Command) []Command {
	methods := []struct{ method, role, gravity string }{
		{"TargetBackupNow", "gm,commish,admin", "G1"},
		{"TargetCommissionerActor", "gm,commish", "G0"},
		{"TargetCommissionerControls", "gm,commish", "G0"},
		{"TargetCommissionerDigests", "gm,commish", "G0"},
		{"TargetCommissionerExecute", "commish", "G3"},
		{"TargetCommissionerJobs", "gm,commish", "G0"},
		{"TargetCommissionerLeagues", "gm,commish", "G0"},
		{"TargetCommissionerLedger", "gm,commish", "G0"},
		{"TargetCommissionerLedgerFiltered", "gm,commish", "G0"},
		{"TargetCommissionerMarket", "gm,commish", "G0"},
		{"TargetCommissionerNotices", "gm,commish", "G0"},
		{"TargetCommissionerPeople", "gm,commish", "G0"},
		{"TargetCommissionerPreview", "commish", "G0"},
		{"TargetCommissionerRules", "gm,commish", "G0"},
		{"TargetCommissionerSchedule", "commish", "G0"},
		{"TargetCommissionerSeat", "gm,commish,admin", "G0"},
		{"TargetCommissionerSeatSignIn", "gm,commish,admin", "G1"},
		{"TargetCorrections", "gm,commish", "G0"},
		{"TargetLeagueCalendar", "gm,commish", "G0"},
		{"TargetLeagueSchedule", "gm,commish", "G0"},
		{"TargetNativeAssets", "gm,commish", "G0"},
		{"TargetNativeBidAction", "gm,commish", "G3"},
		{"TargetNativeBids", "gm,commish", "G0"},
		{"TargetNativeBracket", "gm,commish", "G0"},
		{"TargetNativeCapCheck", "gm,commish", "G0"},
		{"TargetNativeContractQuote", "gm,commish,admin", "G0"},
		{"TargetNativeDirectory", "gm,commish,admin", "G0"},
		{"TargetNativeDraft", "gm,commish,admin", "G0"},
		{"TargetNativeDraftPick", "gm,commish", "G3"},
		{"TargetNativeFranchiseSchedule", "gm,commish,admin", "G0"},
		{"TargetNativeHouseRules", "gm,commish,admin", "G0"},
		{"TargetNativeLineup", "gm,commish,admin", "G0"},
		{"TargetNativeLineupCheck", "gm,commish,admin", "G0"},
		{"TargetNativeLineupSet", "gm,commish", "G2"},
		{"TargetNativePlayerScore", "gm,commish,admin", "G0"},
		{"TargetNativeReserveCheck", "gm,commish,admin", "G0"},
		{"TargetNativeResults", "gm,commish,admin", "G0"},
		{"TargetNativeRosterCheck", "gm,commish,admin", "G0"},
		{"TargetNativeScorePreview", "commish", "G0"},
		{"TargetNativeSeed", "commish", "G3"},
		{"TargetNativeStandings", "gm,commish,admin", "G0"},
		{"TargetNativeTradeAction", "gm,commish", "G3"},
		{"TargetNativeTrades", "gm,commish", "G0"},
		{"TargetNativeWaiverAction", "gm,commish", "G3"},
		{"TargetNativeWaivers", "gm,commish", "G0"},
		{"TargetNativeWeek", "gm,commish,admin", "G0"},
		{"TargetStatQueue", "commish", "G0"},
	}
	return desktopCommands(methods, related)
}
func desktopCommands(methods []struct{ method, role, gravity string }, related []Command) []Command {
	reads := map[string]Command{}
	for _, read := range related {
		if len(read.Modes) != 1 || read.Modes[0] != Read {
			continue
		}
		reads[read.Method] = read
	}
	commands := make([]Command, 0, len(methods))
	for _, metadata := range methods {
		command := Command{ID: Unset, Method: metadata.method, Version: Unset,
			Role: metadata.role, Gravity: metadata.gravity, Availability: "both"}
		switch metadata.method {
		case "TargetCommissionerLedger", "TargetCommissionerLedgerFiltered":
			command.ID = "commish.ledger.read"
		case "TargetNativeLineupSet":
			command.ID = "lineup.set"
		case "TargetCommissionerSeat", "TargetCommissionerSeatSignIn":
			command.Availability = "desktop" // a hosted league signs in on the server
		case "TargetBackupNow":
			command.ID, command.Availability = "backup.now", "desktop" // a hosted league backs up on the server
		}
		if read, ok := reads[command.Method]; ok {
			read.Role = command.Role
			command = read
		}
		commands = append(commands, command)
	}
	return commands
}

func commsDesktop(_ []Command) []Command {
	return []Command{
		{ID: "comms.read", Method: "TargetComms", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: "comms.refresh", Method: "TargetCommsRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: "comms.open", Method: "TargetOpenComms", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "desktop"},
	}
}

func contractsDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetAcquisition", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetCheckLineup", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetContractActs", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "both"},
		{ID: Unset, Method: "TargetContracts", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: "lineup.set", Method: "TargetDraftLineup", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetLineup", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetPlayerContract", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetReleasePlans", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "desktop"},
		{ID: Unset, Method: "TargetRosterReports", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetWhatIf", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "both"},
	}
}

func controlDesktop(_ []Command) []Command {
	c := func(method, role, gravity, availability string) Command {
		return Command{ID: Unset, Method: method, Version: Unset, Role: role,
			Gravity: gravity, Availability: availability}
	}
	return []Command{
		c("DeleteMFLKey", "gm,commish,admin", "G2", "desktop"), heldRead(c("GetParams", "gm,commish,admin", "G0", "both")), c("MFLKeyStatus", "gm,commish,admin", "G0", "desktop"), c("ResetParam", "admin", "G2", "both"),
		c("SetMFLKey", "gm,commish,admin", "G2", "desktop"), c("SetParam", "admin", "G2", "both"), heldRead(c("TargetAccounting", "gm,commish,admin", "G0", "both")), heldRead(c("TargetAlerts", "gm,commish,admin", "G0", "both")),
		heldRead(c("TargetBylaws", "gm,commish,admin", "G0", "both")), heldRead(c("TargetLeagueSettings", "gm,commish,admin", "G0", "both")), c("TargetOpenMFLAccounting", "gm,commish,admin", "G0", "desktop"),
		c("TargetSeason", "gm,commish,admin", "G0", "both"), c("TargetShellStatus", "gm,commish,admin", "G0", "both"), heldRead(c("TargetSiteNews", "gm,commish,admin", "G0", "both")),
		{ID: "window.set", Method: "TargetSetWindow", Version: Unset, Role: "commish,admin",
			Gravity: "G2", Availability: "both"},
	}
}

func counterpartiesDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetCounterpartTeam", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetCounterparties", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
	}
}

func draftDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetDraftLab", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetDraftReports", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetDraftReportsRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetNFLDraft", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: "draft.reports", Method: "TargetOpenMFLDraftRoom", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "desktop"},
	}
}

func marketDesktop(_ []Command) []Command {
	read := func(method, gravity string) Command {
		return Command{ID: Unset, Method: method, Version: Unset, Role: "gm,commish,admin", Gravity: gravity, Availability: "both"}
	}
	return []Command{
		heldRead(read("TargetAssetBoard", "G0")),
		heldRead(read("TargetAssetProfile", "G0")),
		read("TargetMarket", "G0"),
		heldRead(read("TargetMarketLists", "G0")),
		heldRead(read("TargetMarketReports", "G0")),
		read("TargetMarketReportsRefresh", "G1"),
		heldRead(read("TargetMarketStretches", "G0")),
		heldRead(read("TargetPickClock", "G0")),
	}
}

func nowDesktop(_ []Command) []Command {
	return []Command{
		{ID: "game.follow", Method: "TargetLiveGame", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetNow", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetNowRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
	}
}

func offersDesktop(related []Command) []Command {
	return append(related, []Command{
		{ID: Unset, Method: "TargetOffers", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetPersonal", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetPersonalChange", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetPersonalHandOff", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "desktop"},
	}...)
}

func platformDesktop(_ []Command) []Command {
	c := func(method, role, gravity, availability string) Command {
		return Command{ID: Unset, Method: method, Version: Unset, Role: role,
			Gravity: gravity, Availability: availability}
	}
	return []Command{
		c("TargetCheckMoves", "gm,commish,admin", "G1", "both"), c("TargetClock", "gm,commish,admin", "G0", "both"), c("TargetDraftIR", "gm,commish", "G2", "both"),
		c("TargetDraftTaxi", "gm,commish", "G2", "both"), c("TargetHandOff", "gm,commish", "G2", "desktop"), c("TargetMoves", "gm,commish,admin", "G0", "both"),
		c("TargetPhoto", "gm,commish,admin", "G0", "both"), c("TargetSnapshot", "gm,commish,admin", "G0", "both"), c("TargetWithdrawRoster", "gm,commish", "G1", "both"),
	}
}

func raceDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetRace", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetRaceRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
	}
}

func researchDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "TargetResearch", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetResearchRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetResearchNewsRefresh", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetResearchPlayer", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetResearchStatus", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "TargetOpenResearchArticle", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "desktop"},
	}
}

func sourcesDesktop(_ []Command) []Command {
	return []Command{
		{ID: Unset, Method: "GetCrosswalkReport", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
		{ID: Unset, Method: "LoadCrosswalk", Version: Unset, Role: "admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "SetLeagueArchiveDir", Version: Unset, Role: "admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "SetPhotos", Version: Unset, Role: "admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetCacheClear", Version: Unset, Role: "gm,commish,admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetCacheScopes", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: Unset, Method: "TargetCrosswalk", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
		{ID: "refresh.run", Method: "TargetDataRefresh", Version: Unset, Role: "admin", Gravity: "G1", Availability: "both"},
		{ID: Unset, Method: "TargetDataSources", Version: Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both", Modes: []Mode{Read}},
	}
}

func tradesDesktop(_ []Command) []Command {
	return []Command{
		{ID: "trade.accept", Method: "TargetDraftTradeAccept", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "both"},
		{ID: Unset, Method: "TargetTradeBuilder", Version: Unset, Role: "gm,commish", Gravity: "G2", Availability: "desktop"},
		{ID: Unset, Method: "TargetTradeBuilderPreview", Version: Unset, Role: "gm,commish", Gravity: "G1", Availability: "desktop", Modes: []Mode{Preview}},
		heldRead(Command{ID: Unset, Method: "TargetTradeBuilderRead", Version: Unset, Role: "gm,commish", Gravity: "G0", Availability: "desktop"}),
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
func mflCommand(id, method, gravity string, mode Mode, args Schema, undo, facts string) Command {
	safeguards := "Raw-validation+local-plan-state; no MFL submissions"
	if gravity == "G2" {
		safeguards += "; explicit-owner-action"
	}
	return Command{ID: id, Method: method, Version: "1", Role: "gm,commish,admin", Gravity: gravity,
		Availability: "desktop", Arguments: args, Permission: "local-desktop-owner", Modes: []Mode{mode},
		Undo: undo, Facts: facts, Safeguards: safeguards}
}
