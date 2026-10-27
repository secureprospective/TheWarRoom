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
