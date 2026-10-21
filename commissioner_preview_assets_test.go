package main

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func TestEveryBuiltAssetCorrectionHasARealPreview(t *testing.T) {
	t.Parallel()
	f := draftApp(t)
	assets := correctionAssets(t, f)
	p := assets.Snapshot.Players[0]
	for _, tc := range []struct {
		command string
		args    commissioner.RawCorrection
	}{
		{commissioner.CorrectRoster, commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "remove"}},
		{commissioner.CorrectRoster, commissioner.RawCorrection{Player: p.ID, Franchise: p.Franchise, Action: "add"}},
		{commissioner.CorrectProvenance, commissioner.RawCorrection{Player: p.ID, Via: "trade"}},
		{commissioner.CorrectPick, commissioner.RawCorrection{Pick: &commissioner.CorrectionPick{Season: draftSeason,
			Round: 1, Original: draftTeam(3)}, Owner: draftTeam(5)}},
		{commissioner.RuleException, commissioner.RawCorrection{Rule: "roster.limits",
			Scope: &commissioner.ExceptionScope{Franchise: draftTeam(1)}}},
	} {
		previewDraft(t, f, tc.command, commissioner.RawArgs{Correction: &tc.args})
	}
	w := waiverApp(t)
	cut := w.cut(t, "1000")
	w.app.core.CommissionerSession = testCommissionerSession{"creator"}
	previewAndExecute(t, w.app, w.raw(t, commissioner.CorrectTransaction,
		commissioner.RawArgs{Correction: &commissioner.RawCorrection{Entry: cut.Entry}}, "league"))
	previewAndExecute(t, w.app, w.raw(t, commissioner.CalendarException,
		commissioner.RawArgs{Exception: &commissioner.RawWindowException{Action: "cut", Franchise: "franchise.01",
			Until: w.now.Add(time.Hour)}}, "league"))
}
