package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/measures"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestReferenceRestoreReportsMissingOwnedPlayer(t *testing.T) {
	h := newTwoFileHost(t)
	lacks, err := referenceLacks(t.Context(), referencePath(h.dir+"/server.db"), referenceNeed{
		players: []string{"player." + strings.Repeat("f", 32)},
	})
	if err != nil || lacks != "it lacks 1 player(s) the leagues own" {
		t.Fatalf("missing player not reported: %q (%v)", lacks, err)
	}
}

func TestMissingPinnedReferenceStopsHostedSelection(t *testing.T) {
	h := newTwoFileHost(t)
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, league := range []string{"native.hosted", "native.other"} {
		t.Run(league, func(t *testing.T) {
			digest := strings.Repeat("e", 64)
			pins := newLeaguePins(history.New(h.files.league, registry))
			err := pins.selectPins(t.Context(), "", []accountstore.DirectoryEntry{{League: league, Football: digest}})
			if !errors.Is(err, history.ErrNoBody) || !strings.Contains(err.Error(), "select football for league "+league) {
				t.Fatalf("missing pin did not identify the blocking read: %v", err)
			}
		})
	}
}
