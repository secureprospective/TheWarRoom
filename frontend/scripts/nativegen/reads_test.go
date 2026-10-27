package main

import (
	"slices"
	"strings"
	"testing"

	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
	"github.com/secureprospective/TheWarRoom/internal/composition"
)

func TestQueryCatalogUsesOnlyAuditedRegistryReadAdapters(t *testing.T) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	entries := registry.Entries()
	body, err := generateReadCommands(entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"archive.TargetArchive", "comms.TargetComms", "draft.TargetDraftReports", "market.TargetMarketReports", "race.TargetRace",
		"control.TargetLeagueSettings", "control.TargetBylaws", "control.TargetAccounting", "control.TargetSiteNews",
		"contracts.TargetLineup", "trades.TargetTrades", "control.TargetAlerts",
		"commissioner.TargetNativeLineup",
		"offers.TargetPersonal", "offers.TargetOffers", "commissioner.TargetNativeDraft",
		"market.TargetAssetBoard", "now.TargetNow",
		"sources.TargetDataSources", "sources.TargetCrosswalk", "control.GetParams",
		"sources.TargetCacheScopes", "draft.TargetDraftLab", "market.TargetMarketLists", "market.TargetPickClock",
		"market.TargetMarketStretches", "market.TargetAssetProfile",
		"counterparties.TargetCounterparties", "counterparties.TargetCounterpartTeam"} {
		if !strings.Contains(string(body), `"`+key+`":`) {
			t.Fatal("missing read", key)
		}
	}
	for _, method := range []string{"TargetCommsRefresh", "TargetOpenComms", "TargetDraftReportsRefresh", "TargetOpenMFLDraftRoom", "TargetRaceRefresh", "TargetCacheClear", "TargetDataRefresh", "TargetOpenMFLAccounting"} {
		if strings.Contains(string(body), method) {
			t.Fatal("action admitted to query catalog", method)
		}
	}
	for i, entry := range entries {
		for j, d := range entry.Desktop {
			if d.Command.Method != "TargetComms" {
				continue
			}
			entries[i].Desktop = slices.Clone(entry.Desktop)
			entries[i].Desktop[j].Command.Modes = []catalog.Mode{catalog.Execute}
		}
	}
	if _, err := generateReadCommands(entries); err == nil || !strings.Contains(err.Error(), "not a held reading") {
		t.Fatal(err)
	}
	if _, err := generateReadCommands(nil); err == nil || !strings.Contains(err.Error(), "absent from registry") {
		t.Fatal(err)
	}
}

func TestMigratedReadAdaptersRefuseEffectModesAndMissingEntries(t *testing.T) {
	registry, err := composition.CommandRegistry()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"control.TargetLeagueSettings", "control.TargetBylaws", "control.TargetAccounting", "control.TargetSiteNews",
		"contracts.TargetLineup", "trades.TargetTrades", "control.TargetAlerts",
		"commissioner.TargetNativeLineup",
		"offers.TargetPersonal", "offers.TargetOffers", "commissioner.TargetNativeDraft",
		"market.TargetAssetBoard", "now.TargetNow",
		"sources.TargetDataSources", "sources.TargetCrosswalk", "control.GetParams",
		"sources.TargetCacheScopes", "draft.TargetDraftLab", "market.TargetMarketLists", "market.TargetPickClock",
		"market.TargetMarketStretches", "market.TargetAssetProfile",
		"counterparties.TargetCounterparties", "counterparties.TargetCounterpartTeam",
	} {
		for _, change := range []string{"mode", "gravity", "missing"} {
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
						case "gravity":
							entries[i].Desktop[j].Command.Gravity = "G1"
						case "missing":
							entries[i].Desktop = slices.Delete(entries[i].Desktop, j, j+1)
						}
					}
				}
				_, err := generateReadCommands(entries)
				want := "not a held reading"
				if change == "missing" {
					want = "absent from registry"
				}
				if err == nil || !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), want) {
					t.Fatal("wrong query admission", key, change, err)
				}
			})
		}
	}
}
