package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

// GenerationRegistry builds the catalog registry for code generators without desktop wiring.
// Desktop bindings carry argument shapes only and refuse execution if invoked.
func GenerationRegistry() (*Registry, error) {
	return NewRegistry(RegistryInputs{
		Seed:    Shape(reflect.TypeFor[seedRequestShape](), "RawNativeSeedRequest.Validate/ValidatePreview"),
		MFL:     generationMFLShapes(),
		Service: new(leagueservice.Service),
		Desktop: generationDesktopBindings(),
	})
}

func generationMFLShapes() MFLShapes {
	return MFLShapes{
		Bid:   Shape(reflect.TypeFor[bidInputShape](), "RawBidInput.Validate"),
		Plan:  Shape(reflect.TypeFor[bidPlanScopeShape](), "RawBidPlanScope.Validate"),
		Pool:  Shape(reflect.TypeFor[poolScopeShape](), "RawPoolScope.Validate"),
		Owner: Shape(reflect.TypeFor[bidOwnerShape](), "RawBidOwner.Validate"),
	}
}

func generationDesktopBindings() map[string]map[string]Binding {
	return map[string]map[string]Binding{
		"commissioner": commissionerGenBindings(),
		"contracts":    contractsGenBindings(),
		"control":      controlGenBindings(),
		"draft":        bindSpecs(map[string]string{"TargetDraftReportsRefresh": "force:boolean", "TargetDraftReports": "", "TargetOpenMFLDraftRoom": "", "TargetNFLDraft": ""}),
		"market":       bindSpecs(map[string]string{"TargetAssetBoard": "", "TargetAgeArc": "id:string", "TargetMarketReportsRefresh": "force:boolean", "TargetMarketReports": ""}),
		"now":          bindSpecs(map[string]string{"TargetNowRefresh": "week:integer force:boolean", "TargetNow": "week:integer reportWeek:integer mine:string", "TargetLiveGame": "open:boolean follow:string mine:string"}),
		"offers":       offersGenBindings(),
		"platform":     platformGenBindings(),
		"race":         bindSpecs(map[string]string{"TargetRace": "week:integer team:string mine:string aggMode:string", "TargetRaceRefresh": "force:boolean"}),
		"research":     researchGenBindings(),
		"sources":      sourcesGenBindings(),
		"trades":       tradesGenBindings(),
	}
}

func commissionerGenBindings() map[string]Binding {
	out := bindSpecs(map[string]string{
		"TargetNativeCapCheck":             "leagueID:string viewerZone:string",
		"TargetNativeRosterCheck":          "leagueID:string",
		"TargetNativeReserveCheck":         "leagueID:string",
		"TargetNativeHouseRules":           "leagueID:string",
		"TargetNativePlayerScore":          "league:string player:string season:integer week:integer",
		"TargetNativeProjections":          "league:string players:string season:integer week:integer",
		"TargetNativeResults":              "league:string season:integer week:integer",
		"TargetNativeStandings":            "league:string season:integer",
		"TargetNativeOdds":                 "league:string season:integer",
		"TargetNativeScorePreview":         "body:string",
		"TargetNativeBracket":              "league:string season:integer",
		"TargetStatQueue":                  "league:string season:integer request:string",
		"TargetCancelRead":                 "request:string",
		"TargetNativeLineupSet":            "body:string",
		"TargetNativeLineup":               "league:string franchise:string week:integer at:string",
		"TargetNativeDirectory":            "leagueID:string",
		"TargetNativeAssets":               "leagueID:string",
		"TargetCommissionerExecute":        "body:string",
		"TargetCommissionerLedger":         "league:string",
		"TargetCommissionerLedgerFiltered": "league:string body:string",
		"TargetCommissionerPeople":         "league:string",
		"TargetCommissionerLeagues":        "",
		"TargetCommissionerRules":          "league:string",
		"TargetCommissionerPresetCompare":  "league:string preset:string",
		"TargetCommissionerPresetApply":    "body:string",
		"TargetCommissionerJobs":           "league:string",
		"TargetCommissionerControls":       "",
		"TargetCommissionerNotices":        "league:string",
		"TargetCommissionerDigests":        "league:string",
		"TargetCorrections":                "league:string",
		"TargetCommissionerMarket":         "league:string command:string",
		"TargetCommissionerActor":          "",
		"TargetCommissionerSchedule":       "league:string",
		"TargetCommissionerSeat":           "",
		"TargetSignInFacts":                "",
		"TargetCommissionerSeatSignIn":     "username:string password:string",
		"TargetWorkflow":                   "body:string",
		"TargetBackupNow":                  "",
		"TargetCommissionerPreview":        "body:string",
		"TargetNativeMoves":                "league:string franchise:string",
		"TargetNativeTrades":               "league:string viewer:string",
		"TargetNativeWaivers":              "league:string franchise:string at:time",
		"TargetNativeBids":                 "league:string franchise:string",
		"TargetNFLGame":                    "league:string game:string",
		"TargetNFLGames":                   "league:string",
	})
	maps.Copy(out, commissionerTypedGenBindings())
	return out
}

func commissionerTypedGenBindings() map[string]Binding {
	return map[string]Binding{
		"TargetNativeLineupCheck":       bindType[commissioner.RawNativeLineup]("input"),
		"TargetNativeLineupDelta":       bindType[leagueservice.RawLineupDeltaScope]("raw"),
		"TargetNativeValues":            bindType[leagueservice.RawValueScope]("raw"),
		"TargetNativeTradeLens":         bindType[leagueservice.RawTradeLensScope]("raw"),
		"TargetNativePlayerUsage":       bindType[leagueservice.RawPlayerReadScope]("raw"),
		"TargetNativePlayerAgeArc":      bindType[leagueservice.RawPlayerReadScope]("raw"),
		"TargetNativeContractQuote":     bindType[market.RawContractQuote]("raw"),
		"TargetNativeDraftPick":         bindType[commissioner.DraftPickInput]("input"),
		"TargetNativeDraft":             bindType[commissioner.RawNativeDraftScope]("scope"),
		"TargetNativeSeed":              bindType[seedRequestShape]("input"),
		"TargetNativeTradeAction":       bindType[commissioner.RawCommand]("raw"),
		"TargetFranchiseMovesHistory":   bindType[commissioner.RawFranchiseMovesHistory]("scope"),
		"TargetNativeWaiverAction":      bindType[commissioner.RawCommand]("raw"),
		"TargetNativeBidAction":         bindType[commissioner.RawCommand]("raw"),
		"TargetLeagueCalendar":          bindType[leagueservice.RawLeagueCalendarScope]("raw"),
		"TargetLeagueSchedule":          bindType[leagueservice.RawLeagueScheduleScope]("raw"),
		"TargetNativeFranchiseSchedule": bindType[leagueservice.RawNativeScheduleScope]("raw"),
		"TargetNativeWeek":              bindType[leagueservice.RawNativeWeekScope]("raw"),
	}
}

func contractsGenBindings() map[string]Binding {
	out := bindSpecs(map[string]string{
		"TargetLineup":         "franchiseID:string",
		"TargetCheckLineup":    "franchiseID:string starters:strings",
		"TargetDraftLineup":    "franchiseID:string starters:strings",
		"TargetRosterReports":  "mine:string",
		"TargetPlayerContract": "rawID:string",
	})
	out["TargetAcquisition"] = bindType[acquisitionShape]("r")
	out["TargetContractActs"] = bindType[contractActShape]("raw")
	out["TargetContracts"] = bindType[contractScopeShape]("raw")
	out["TargetWhatIf"] = bindType[whatIfRequestShape]("raw")
	out["TargetReleasePlans"] = bindType[releaseRequestShape]("raw")
	return out
}

func controlGenBindings() map[string]Binding {
	return bindSpecs(map[string]string{
		"TargetMFLMigration":      "command:string body:string",
		"TargetLeagueSettings":    "",
		"TargetSiteNews":          "",
		"TargetAccounting":        "",
		"TargetOpenMFLAccounting": "",
		"TargetBylaws":            "",
		"GetParams":               "",
		"SetParam":                "key:string position:string value:number",
		"ResetParam":              "key:string position:string",
		"MFLKeyStatus":            "",
		"SetMFLKey":               "key:string",
		"DeleteMFLKey":            "",
		"TargetAlerts":            "franchiseID:string",
		"TargetSeason":            "",
		"TargetShellStatus":       "",
		"TargetSetWindow":         "kind:string at:string",
	})
}

func offersGenBindings() map[string]Binding {
	out := bindSpecs(map[string]string{
		"TargetPersonal":        "mine:string",
		"TargetPersonalNotes":   "mine:string playerID:string",
		"TargetPersonalHandOff": "kind:string",
		"TargetPoolRefresh":     "force:boolean",
		"TargetBidPlans":        "franchiseID:string",
		"TargetBidHandOff":      "id:string",
		"TargetBidPosted":       "id:string",
		"TargetBidWithdraw":     "id:string",
	})
	out["TargetPersonalChange"] = bindType[personalChangeShape]("raw")
	out["TargetPool"] = bindType[poolScopeShape]("raw")
	out["TargetBidPreview"] = bindType[bidInputShape]("raw")
	out["TargetBidDraft"] = bindType[bidInputShape]("raw")
	out["TargetBidRevise"] = shapeBinding(
		Field{Name: "id", Schema: Schema{Type: "string"}},
		Field{Name: "raw", Schema: shape(reflect.TypeFor[bidInputShape]())},
	)
	return out
}

func platformGenBindings() map[string]Binding {
	return bindSpecs(map[string]string{
		"TargetSnapshot":       "",
		"TargetLeagues":        "",
		"TargetClock":          "",
		"TargetInjuries":       "",
		"TargetDraftIR":        "franchiseID:string playerID:string",
		"TargetDraftTaxi":      "franchiseID:string playerID:string",
		"TargetMoves":          "franchiseID:string",
		"TargetHandOff":        "correlationID:string",
		"TargetCheckMoves":     "",
		"TargetWithdrawRoster": "correlationID:string",
		"TargetPhoto":          "mflID:string",
	})
}

func researchGenBindings() map[string]Binding {
	out := bindSpecs(map[string]string{
		"TargetResearchRefresh":     "force:boolean",
		"TargetResearchNewsRefresh": "mine:string",
		"TargetPlayerUsage":         "playerID:string",
		"TargetResearchPlayer":      "id:string",
		"TargetResearchStatus":      "",
		"TargetOpenResearchArticle": "id:string article:string",
	})
	out["TargetResearch"] = bindType[researchScopeShape]("raw")
	return out
}

func sourcesGenBindings() map[string]Binding {
	return bindSpecs(map[string]string{
		"LoadCrosswalk":       "",
		"GetCrosswalkReport":  "",
		"TargetCrosswalk":     "",
		"TargetDataSources":   "",
		"TargetDataRefresh":   "scope:string",
		"SetLeagueArchiveDir": "dir:string",
		"SetPhotos":           "on:boolean",
		"TargetCacheScopes":   "",
		"TargetCacheClear":    "scope:string",
	})
}

func tradesGenBindings() map[string]Binding {
	out := bindSpecs(map[string]string{
		"TargetTrades":           "franchiseID:string",
		"TargetDraftTradeAccept": "franchiseID:string tradeID:string",
		"TargetTradeBuilderRead": "franchiseID:string",
	})
	out["TargetTradeBuilder"] = bindType[tradeBuilderShape]("r")
	out["TargetTradeBuilderPreview"] = bindType[tradeBuilderShape]("r")
	return out
}

func bindSpecs(specs map[string]string) map[string]Binding {
	out := make(map[string]Binding, len(specs))
	for method, spec := range specs {
		out[method] = bindSpec(spec)
	}
	return out
}

func bindSpec(spec string) Binding {
	tokens := strings.Fields(spec)
	fields := make([]Field, 0, len(tokens))
	for _, token := range tokens {
		name, kind, _ := strings.Cut(token, ":")
		fields = append(fields, Field{Name: name, Schema: scalarSchema(kind)})
	}
	return shapeBinding(fields...)
}

func scalarSchema(kind string) Schema {
	switch kind {
	case "string", "integer", "number", "boolean":
		return Schema{Type: kind}
	case "time":
		return Schema{Type: "string", Format: "date-time"}
	case "strings":
		item := Schema{Type: "string"}
		return Schema{Type: "array", Items: &item}
	default:
		return Schema{Type: Unset}
	}
}

func bindType[T any](name string) Binding {
	return shapeBinding(Field{Name: name, Schema: shape(reflect.TypeFor[T]())})
}

func shapeBinding(fields ...Field) Binding {
	if fields == nil {
		fields = []Field{}
	}
	return Binding{
		Arguments: Schema{Type: "object", Validator: "desktop adapter Raw validation", Fields: fields},
		Handler: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			return nil, fmt.Errorf("generation registry cannot execute desktop bindings")
		},
	}
}

type seedRequestShape struct {
	League   string `json:"league"`
	Expected int64  `json:"expected"`
	Request  string `json:"request"`
	Reason   string `json:"reason"`
}

type bidInputShape struct {
	ConditionalDrop string `json:"conditionalDrop"`
	FranchiseID     string `json:"franchiseId"`
	Player          string `json:"player"`
	Salary          string `json:"salary"`
	Years           int    `json:"years"`
	Snipe           bool   `json:"snipe"`
	Beat            string `json:"beat"`
	Evidence        string `json:"evidence"`
	Closes          string `json:"closes"`
}

type poolScopeShape struct {
	ShowNoTeam  bool   `json:"showNoTeam"`
	FranchiseID string `json:"franchiseId"`
	Position    string `json:"position"`
	Sort        string `json:"sort"`
	Desc        bool   `json:"desc"`
}

type bidOwnerShape struct {
	FranchiseID string `json:"franchiseId"`
}

type bidPlanScopeShape struct {
	ID string `json:"id"`
}

type acquisitionShape struct {
	Mine   string `json:"mine"`
	Player string `json:"player"`
	Bid    string `json:"bid"`
	Cut    string `json:"cut"`
}

type contractActShape struct {
	Mine        string `json:"mine"`
	Player      string `json:"player"`
	Action      string `json:"action"`
	Intent      string `json:"intent"`
	ID          string `json:"id"`
	Level       int    `json:"level"`
	Years       int    `json:"years"`
	Source      int    `json:"source"`
	Destination int    `json:"destination"`
	Move        string `json:"move"`
}

type contractScopeShape struct {
	Mine     string `json:"mine"`
	Team     string `json:"team"`
	Position string `json:"position"`
	Sort     string `json:"sort"`
	Desc     bool   `json:"desc"`
}

type whatIfRequestShape struct {
	Mine   string `json:"mine"`
	Action string `json:"action"`
	Player string `json:"player"`
	Move   string `json:"move"`
}

type releaseRequestShape struct {
	Mine      string `json:"mine"`
	Action    string `json:"action"`
	Move      string `json:"move"`
	ID        string `json:"id"`
	Confirmed bool   `json:"confirmed"`
}

type personalChangeShape struct {
	Mine   string `json:"mine"`
	Kind   string `json:"kind"`
	Entity string `json:"entity"`
	ID     string `json:"id"`
	Text   string `json:"text"`
	Action string `json:"action"`
	At     string `json:"at,omitempty"`
	With   string `json:"with,omitempty"`
}

type researchScopeShape struct {
	Mine     string `json:"mine"`
	Search   string `json:"search"`
	Position string `json:"position"`
	Team     string `json:"team"`
	Group    string `json:"group"`
	Trend    string `json:"trend"`
	Depth    string `json:"depth"`
	Injuries string `json:"injuries"`
}

type tradeBuilderShape struct {
	Mine      string   `json:"mine"`
	Other     string   `json:"other"`
	Action    string   `json:"action"`
	ID        string   `json:"id"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Rationale string   `json:"rationale"`
}
