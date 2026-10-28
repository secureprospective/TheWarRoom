package main

import (
	"slices"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/composition"
)

// auditedReads are the adapters audited onto the shared read tool before the catalog was generated.
func auditedReads() []string {
	return []string{"archive.TargetArchive", "comms.TargetComms", "draft.TargetDraftReports", "market.TargetMarketReports", "race.TargetRace",
		"control.TargetLeagueSettings", "control.TargetBylaws", "control.TargetAccounting", "control.TargetSiteNews",
		"contracts.TargetLineup", "trades.TargetTrades", "control.TargetAlerts", "research.TargetResearch",
		"commissioner.TargetNativeLineup",
		"offers.TargetPersonal", "commissioner.TargetNativeDraft",
		"market.TargetAssetBoard", "now.TargetNow",
		"sources.TargetDataSources", "sources.TargetCrosswalk", "control.GetParams",
		"sources.TargetCacheScopes",
		"commissioner.TargetCommissionerLeagues", "commissioner.TargetCommissionerLedger",
		"commissioner.TargetCommissionerRules", "commissioner.TargetCommissionerPeople",
		"commissioner.TargetCommissionerJobs", "commissioner.TargetCommissionerNotices",
		"commissioner.TargetCommissionerDigests",
	}
}

func TestQueryCatalogIsTheRegistrysHeldDesktopReads(t *testing.T) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	entries := registry.Entries()
	body, err := generateReadCommands(entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range auditedReads() {
		if !strings.Contains(string(body), `"`+key+`":`) {
			t.Fatal("missing read", key)
		}
	}
	// Actions, refreshes, browser openers, unmarked legacy reads and the G1 free-agent pool stay out.
	for _, key := range []string{"comms.TargetCommsRefresh", "comms.TargetOpenComms", "draft.TargetDraftReportsRefresh", "draft.TargetOpenMFLDraftRoom",
		"race.TargetRaceRefresh", "sources.TargetCacheClear", "sources.TargetDataRefresh", "control.TargetOpenMFLAccounting",
		"commissioner.TargetCommissionerExecute", "commissioner.TargetCommissionerPreview", "trades.TargetTradeBuilder",
		"market.TargetMarket", "offers.TargetPool", "offers.TargetBidPlans"} {
		if strings.Contains(string(body), `"`+key+`":`) {
			t.Fatal("non-read admitted to query catalog", key)
		}
	}
	if _, err := generateReadCommands(nil); err == nil || !strings.Contains(err.Error(), "no held desktop reads") {
		t.Fatal(err)
	}
}

func TestReadCatalogDropsAdaptersThatLoseTheirReadMark(t *testing.T) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range auditedReads() {
		for _, change := range []string{"mode", "unmarked", "gravity"} {
			t.Run(key+"/"+change, func(t *testing.T) {
				entries := registry.Entries()
				for i, entry := range entries {
					for j, desktop := range entry.Desktop {
						if desktop.Feature+"."+desktop.Command.Method != key {
							continue
						}
						entries[i].Desktop = slices.Clone(entry.Desktop)
						switch change {
						case "mode":
							entries[i].Desktop[j].Command.Modes = []catalog.Mode{catalog.Execute}
						case "unmarked":
							entries[i].Desktop[j].Command.Modes = nil
						case "gravity":
							entries[i].Desktop[j].Command.Gravity = "G1"
						}
					}
				}
				body, err := generateReadCommands(entries)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(body), `"`+key+`":`) {
					t.Fatal("query catalog kept", key, change)
				}
			})
		}
	}
}
