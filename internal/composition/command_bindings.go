package composition

import (
	"reflect"

	archiveapi "github.com/secureprospective/TheWarRoom/internal/app/archive"
	commissionerapi "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	commsapi "github.com/secureprospective/TheWarRoom/internal/app/comms"
	contractsapi "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	controlapi "github.com/secureprospective/TheWarRoom/internal/app/control"
	counterpartiesapi "github.com/secureprospective/TheWarRoom/internal/app/counterparties"
	draftapi "github.com/secureprospective/TheWarRoom/internal/app/draft"
	marketapi "github.com/secureprospective/TheWarRoom/internal/app/market"
	nowapi "github.com/secureprospective/TheWarRoom/internal/app/now"
	offersapi "github.com/secureprospective/TheWarRoom/internal/app/offers"
	platformapi "github.com/secureprospective/TheWarRoom/internal/app/platform"
	raceapi "github.com/secureprospective/TheWarRoom/internal/app/race"
	researchapi "github.com/secureprospective/TheWarRoom/internal/app/research"
	sourcesapi "github.com/secureprospective/TheWarRoom/internal/app/sources"
	tradesapi "github.com/secureprospective/TheWarRoom/internal/app/trades"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func archiveCommandBindings(api *archiveapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetArchive": catalog.Bind3(api.TargetArchive, "season", "team", "other"),
	}
}
func commissionerCommandBindings(api *commissionerapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetNativeCapCheck":             catalog.Bind2(api.TargetNativeCapCheck, "leagueID", "viewerZone"),
		"TargetNativeRosterCheck":          catalog.Bind1(api.TargetNativeRosterCheck, "leagueID"),
		"TargetNativeReserveCheck":         catalog.Bind1(api.TargetNativeReserveCheck, "leagueID"),
		"TargetNativeLineupCheck":          catalog.Bind1(api.TargetNativeLineupCheck, "input"),
		"TargetNativeHouseRules":           catalog.Bind1(api.TargetNativeHouseRules, "leagueID"),
		"TargetNativePlayerScore":          catalog.Bind4(api.TargetNativePlayerScore, "league", "player", "season", "week"),
		"TargetNativeContractQuote":        catalog.Bind1(api.TargetNativeContractQuote, "raw"),
		"TargetNativeResults":              catalog.Bind3(api.TargetNativeResults, "league", "season", "week"),
		"TargetNativeStandings":            catalog.Bind2(api.TargetNativeStandings, "league", "season"),
		"TargetNativeScorePreview":         catalog.Bind1(api.TargetNativeScorePreview, "body"),
		"TargetNativeBracket":              catalog.Bind2(api.TargetNativeBracket, "league", "season"),
		"TargetStatQueue":                  catalog.Bind2(api.TargetStatQueue, "league", "season"),
		"TargetNativeTradeAction":          catalog.Bind1(api.TargetNativeTradeAction, "raw"),
		"TargetNativeTrades":               catalog.Bind2(api.TargetNativeTrades, "league", "viewer"),
		"TargetNativeWaiverAction":         catalog.Bind1(api.TargetNativeWaiverAction, "raw"),
		"TargetNativeWaivers":              catalog.Bind3(api.TargetNativeWaivers, "league", "franchise", "at"),
		"TargetNativeBidAction":            catalog.Bind1(api.TargetNativeBidAction, "raw"),
		"TargetNativeBids":                 catalog.Bind2(api.TargetNativeBids, "league", "franchise"),
		"TargetNativeDraftPick":            catalog.Bind1(api.TargetNativeDraftPick, "input"),
		"TargetNativeDraft":                catalog.Bind1(api.TargetNativeDraft, "scope"),
		"TargetNativeLineupSet":            catalog.Bind1(api.TargetNativeLineupSet, "body"),
		"TargetNativeLineup":               catalog.Bind4(api.TargetNativeLineup, "league", "franchise", "week", "at"),
		"TargetNativeSeed":                 catalog.Bind1(api.TargetNativeSeed, "input"),
		"TargetNativeDirectory":            catalog.Bind1(api.TargetNativeDirectory, "leagueID"),
		"TargetNativeAssets":               catalog.Bind1(api.TargetNativeAssets, "leagueID"),
		"TargetCommissionerExecute":        catalog.Bind1(api.TargetCommissionerExecute, "body"),
		"TargetCommissionerLedger":         catalog.Bind1(api.TargetCommissionerLedger, "league"),
		"TargetCommissionerLedgerFiltered": catalog.Bind2(api.TargetCommissionerLedgerFiltered, "league", "body"),
		"TargetCommissionerPeople":         catalog.Bind1(api.TargetCommissionerPeople, "league"),
		"TargetCommissionerLeagues":        catalog.Bind0(api.TargetCommissionerLeagues),
		"TargetCommissionerRules":          catalog.Bind1(api.TargetCommissionerRules, "league"),
		"TargetCommissionerJobs":           catalog.Bind1(api.TargetCommissionerJobs, "league"),
		"TargetCommissionerControls":       catalog.Bind0Value(api.TargetCommissionerControls),
		"TargetCommissionerNotices":        catalog.Bind1(api.TargetCommissionerNotices, "league"),
		"TargetCommissionerDigests":        catalog.Bind1(api.TargetCommissionerDigests, "league"),
		"TargetCorrections":                catalog.Bind1(api.TargetCorrections, "league"),
		"TargetCommissionerMarket":         catalog.Bind2(api.TargetCommissionerMarket, "league", "command"),
		"TargetCommissionerActor":          catalog.Bind0(api.TargetCommissionerActor),
		"TargetCommissionerSchedule":       catalog.Bind1(api.TargetCommissionerSchedule, "league"),
		"TargetCommissionerSeat":           catalog.Bind0(api.TargetCommissionerSeat),
		"TargetCommissionerSeatSignIn":     catalog.Bind2(api.TargetCommissionerSeatSignIn, "username", "password"),
		"TargetBackupNow":                  catalog.Bind0(api.TargetBackupNow),
		"TargetCommissionerPreview":        catalog.Bind1(api.TargetCommissionerPreview, "body"),
		"TargetLeagueCalendar":             catalog.Bind1(api.TargetLeagueCalendar, "raw"),
		"TargetLeagueSchedule":             catalog.Bind1(api.TargetLeagueSchedule, "raw"),
		"TargetNativeFranchiseSchedule":    catalog.Bind1(api.TargetNativeFranchiseSchedule, "raw"),
		"TargetNativeWeek":                 catalog.Bind1(api.TargetNativeWeek, "raw"),
	}
}
func commsCommandBindings(api *commsapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetComms":        catalog.Bind1(api.TargetComms, "raw"),
		"TargetCommsRefresh": catalog.Bind1Error(api.TargetCommsRefresh, "force"),
		"TargetOpenComms":    catalog.Bind1Error(api.TargetOpenComms, "destination"),
	}
}
func contractsCommandBindings(api *contractsapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetAcquisition":    catalog.Bind1Value(api.TargetAcquisition, "r"),
		"TargetLineup":         catalog.Bind1(api.TargetLineup, "franchiseID"),
		"TargetCheckLineup":    catalog.Bind2(api.TargetCheckLineup, "franchiseID", "starters"),
		"TargetDraftLineup":    catalog.Bind2(api.TargetDraftLineup, "franchiseID", "starters"),
		"TargetRosterReports":  catalog.Bind1(api.TargetRosterReports, "mine"),
		"TargetContractActs":   catalog.Bind1Value(api.TargetContractActs, "raw"),
		"TargetContracts":      catalog.Bind1Value(api.TargetContracts, "raw"),
		"TargetPlayerContract": catalog.Bind1Value(api.TargetPlayerContract, "rawID"),
		"TargetWhatIf":         catalog.Bind1Value(api.TargetWhatIf, "raw"),
		"TargetReleasePlans":   catalog.Bind1Value(api.TargetReleasePlans, "raw"),
	}
}
func controlCommandBindings(api *controlapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetLeagueSettings":    catalog.Bind0(api.TargetLeagueSettings),
		"TargetSiteNews":          catalog.Bind0(api.TargetSiteNews),
		"TargetAccounting":        catalog.Bind0Value(api.TargetAccounting),
		"TargetOpenMFLAccounting": catalog.Bind0Error(api.TargetOpenMFLAccounting),
		"TargetBylaws":            catalog.Bind0Value(api.TargetBylaws),
		"GetParams":               catalog.Bind0Value(api.GetParams),
		"SetParam":                catalog.Bind3Value(api.SetParam, "key", "position", "value"),
		"ResetParam":              catalog.Bind2Value(api.ResetParam, "key", "position"),
		"MFLKeyStatus":            catalog.Bind0(api.MFLKeyStatus),
		"SetMFLKey":               catalog.Bind1(api.SetMFLKey, "key"),
		"DeleteMFLKey":            catalog.Bind0(api.DeleteMFLKey),
		"TargetAlerts":            catalog.Bind1(api.TargetAlerts, "franchiseID"),
		"TargetSeason":            catalog.Bind0(api.TargetSeason),
		"TargetShellStatus":       catalog.Bind0(api.TargetShellStatus),
		"TargetSetWindow":         catalog.Bind2Error(api.TargetSetWindow, "kind", "at"),
	}
}
func counterpartiesCommandBindings(api *counterpartiesapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetCounterparties":  catalog.Bind1Value(api.TargetCounterparties, "raw"),
		"TargetCounterpartTeam": catalog.Bind1Value(api.TargetCounterpartTeam, "raw"),
	}
}
func draftCommandBindings(api *draftapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetDraftReportsRefresh": catalog.Bind1Error(api.TargetDraftReportsRefresh, "force"),
		"TargetDraftReports":        catalog.Bind0(api.TargetDraftReports),
		"TargetOpenMFLDraftRoom":    catalog.Bind0Error(api.TargetOpenMFLDraftRoom),
		"TargetNFLDraft":            catalog.Bind0(api.TargetNFLDraft),
		"TargetDraftLab":            catalog.Bind1Value(api.TargetDraftLab, "raw"),
	}
}
func marketCommandBindings(api *marketapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetAssetBoard":           catalog.Bind0(api.TargetAssetBoard),
		"TargetMarket":               catalog.Bind1Value(api.TargetMarket, "raw"),
		"TargetMarketLists":          catalog.Bind1Value(api.TargetMarketLists, "raw"),
		"TargetPickClock":            catalog.Bind1Value(api.TargetPickClock, "raw"),
		"TargetMarketStretches":      catalog.Bind1Value(api.TargetMarketStretches, "raw"),
		"TargetAssetProfile":         catalog.Bind1Value(api.TargetAssetProfile, "raw"),
		"TargetMarketReportsRefresh": catalog.Bind1Error(api.TargetMarketReportsRefresh, "force"),
		"TargetMarketReports":        catalog.Bind0(api.TargetMarketReports),
	}
}
func nowCommandBindings(api *nowapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetNowRefresh": catalog.Bind2Error(api.TargetNowRefresh, "week", "force"),
		"TargetNow":        catalog.Bind3(api.TargetNow, "week", "reportWeek", "mine"),
		"TargetLiveGame":   catalog.Bind3(api.TargetLiveGame, "open", "follow", "mine"),
	}
}
func offersCommandBindings(api *offersapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetOffers":          catalog.Bind1Value(api.TargetOffers, "raw"),
		"TargetPersonal":        catalog.Bind1(api.TargetPersonal, "mine"),
		"TargetPersonalChange":  catalog.Bind1Error(api.TargetPersonalChange, "raw"),
		"TargetPersonalHandOff": catalog.Bind1Error(api.TargetPersonalHandOff, "kind"),
		"TargetPool":            catalog.Bind1Value(api.TargetPool, "raw"),
		"TargetPoolRefresh":     catalog.Bind1Error(api.TargetPoolRefresh, "force"),
		"TargetBidPreview":      catalog.Bind1Value(api.TargetBidPreview, "raw"),
		"TargetBidDraft":        catalog.Bind1(api.TargetBidDraft, "raw"),
		"TargetBidPlans":        catalog.Bind1Value(api.TargetBidPlans, "franchiseID"),
		"TargetBidHandOff":      catalog.Bind1(api.TargetBidHandOff, "id"),
		"TargetBidPosted":       catalog.Bind1(api.TargetBidPosted, "id"),
		"TargetBidWithdraw":     catalog.Bind1(api.TargetBidWithdraw, "id"),
		"TargetBidRevise":       catalog.Bind2(api.TargetBidRevise, "id", "raw"),
	}
}
func platformCommandBindings(api *platformapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetSnapshot":       catalog.Bind0(api.TargetSnapshot),
		"TargetClock":          catalog.Bind0(api.TargetClock),
		"TargetDraftIR":        catalog.Bind2(api.TargetDraftIR, "franchiseID", "playerID"),
		"TargetDraftTaxi":      catalog.Bind2(api.TargetDraftTaxi, "franchiseID", "playerID"),
		"TargetMoves":          catalog.Bind1(api.TargetMoves, "franchiseID"),
		"TargetHandOff":        catalog.Bind1(api.TargetHandOff, "correlationID"),
		"TargetCheckMoves":     catalog.Bind0Error(api.TargetCheckMoves),
		"TargetWithdrawRoster": catalog.Bind1(api.TargetWithdrawRoster, "correlationID"),
		"TargetPhoto":          catalog.Bind1Value(api.TargetPhoto, "mflID"),
	}
}
func raceCommandBindings(api *raceapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetRace":        catalog.Bind3(api.TargetRace, "week", "team", "mine"),
		"TargetRaceRefresh": catalog.Bind1Error(api.TargetRaceRefresh, "force"),
	}
}
func researchCommandBindings(api *researchapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetResearch":            catalog.Bind1(api.TargetResearch, "raw"),
		"TargetResearchRefresh":     catalog.Bind1Error(api.TargetResearchRefresh, "force"),
		"TargetResearchNewsRefresh": catalog.Bind1Error(api.TargetResearchNewsRefresh, "mine"),
		"TargetResearchPlayer":      catalog.Bind1(api.TargetResearchPlayer, "id"),
		"TargetResearchStatus":      catalog.Bind0Value(api.TargetResearchStatus),
		"TargetOpenResearchArticle": catalog.Bind2Error(api.TargetOpenResearchArticle, "id", "article"),
	}
}
func sourcesCommandBindings(api *sourcesapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"LoadCrosswalk":       catalog.Bind0Value(api.LoadCrosswalk),
		"GetCrosswalkReport":  catalog.Bind0Value(api.GetCrosswalkReport),
		"TargetCrosswalk":     catalog.Bind0Value(api.TargetCrosswalk),
		"TargetDataSources":   catalog.Bind0(api.TargetDataSources),
		"TargetDataRefresh":   catalog.Bind1Value(api.TargetDataRefresh, "scope"),
		"SetLeagueArchiveDir": catalog.Bind1Value(api.SetLeagueArchiveDir, "dir"),
		"SetPhotos":           catalog.Bind1Value(api.SetPhotos, "on"),
		"TargetCacheScopes":   catalog.Bind0Value(api.TargetCacheScopes),
		"TargetCacheClear":    catalog.Bind1Value(api.TargetCacheClear, "scope"),
	}
}
func tradesCommandBindings(api *tradesapi.API) map[string]catalog.Binding {
	return map[string]catalog.Binding{
		"TargetTrades":           catalog.Bind1(api.TargetTrades, "franchiseID"),
		"TargetDraftTradeAccept": catalog.Bind2(api.TargetDraftTradeAccept, "franchiseID", "tradeID"),
		"TargetTradeBuilder":     catalog.Bind1Value(api.TargetTradeBuilder, "r"),
	}
}

type commandAPIs struct {
	archive        *archiveapi.API
	commissioner   *commissionerapi.API
	comms          *commsapi.API
	contracts      *contractsapi.API
	control        *controlapi.API
	counterparties *counterpartiesapi.API
	draft          *draftapi.API
	market         *marketapi.API
	now            *nowapi.API
	offers         *offersapi.API
	platform       *platformapi.API
	race           *raceapi.API
	research       *researchapi.API
	sources        *sourcesapi.API
	trades         *tradesapi.API
}

func newCommandAPIs(app *App) commandAPIs {
	return commandAPIs{
		archive:        archiveapi.New(app.core),
		commissioner:   newCommissionerAPI(app),
		comms:          commsapi.New(app.core),
		contracts:      contractsapi.New(contractsHost{State: app.core, app: app}),
		control:        controlapi.New(controlHost{State: app.core, app: app}, app.controlHeld),
		counterparties: counterpartiesapi.New(app.core),
		draft:          draftapi.New(draftHost{State: app.core, app: app}),
		market:         newMarketAPI(app),
		now:            nowapi.New(nowHost{State: app.core, app: app}, app.nowHeld),
		offers:         offersapi.New(offersHost{State: app.core, app: app}),
		platform:       app.platformAPI(),
		race:           raceapi.New(app.core, app.raceHeld),
		research:       researchapi.New(researchHost{app: app}, app.researchHeld),
		sources:        app.sourcesAPI(),
		trades:         tradesapi.New(tradesHost{State: app.core, app: app}),
	}
}
func (a commandAPIs) registryInputs(app *App) catalog.RegistryInputs {
	return catalog.RegistryInputs{
		Seed: catalog.Shape(reflect.TypeFor[commissionerapi.RawNativeSeedRequest](), "RawNativeSeedRequest.Validate/ValidatePreview"),
		MFL: catalog.MFLShapes{
			Bid:   catalog.Shape(reflect.TypeFor[offersapi.RawBidInput](), "RawBidInput.Validate"),
			Plan:  catalog.Shape(reflect.TypeFor[offersapi.RawBidPlanScope](), "RawBidPlanScope.Validate"),
			Pool:  catalog.Shape(reflect.TypeFor[offersapi.RawPoolScope](), "RawPoolScope.Validate"),
			Owner: catalog.Shape(reflect.TypeFor[offersapi.RawBidOwner](), "RawBidOwner.Validate"),
		},
		Service: leagueservice.New(commissionerHost{State: app.core, app: app}),
		Desktop: map[string]map[string]catalog.Binding{
			"archive":        archiveCommandBindings(a.archive),
			"commissioner":   commissionerCommandBindings(a.commissioner),
			"comms":          commsCommandBindings(a.comms),
			"contracts":      contractsCommandBindings(a.contracts),
			"control":        controlCommandBindings(a.control),
			"counterparties": counterpartiesCommandBindings(a.counterparties),
			"draft":          draftCommandBindings(a.draft),
			"market":         marketCommandBindings(a.market),
			"now":            nowCommandBindings(a.now),
			"offers":         offersCommandBindings(a.offers),
			"platform":       platformCommandBindings(a.platform),
			"race":           raceCommandBindings(a.race),
			"research":       researchCommandBindings(a.research),
			"sources":        sourcesCommandBindings(a.sources),
			"trades":         tradesCommandBindings(a.trades),
		},
	}
}
