package market

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguearchive"
	"github.com/secureprospective/TheWarRoom/internal/pickvalue"
	"github.com/secureprospective/TheWarRoom/internal/playerrecord"
)

// HeldDraftBodies offers archived exports newest first, including imported copies; never fetches.
type HeldDraftBodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error)

// HeldPickHistory extends the frozen v1 sample only after its last season. Each season uses its
// newest valid held draft, so corrections replace rather than duplicate picks. Unknown draft
// capital is omitted, not relabelled undrafted. Missing seasons retain the original sample.
func HeldPickHistory(ctx context.Context, bodies HeldDraftBodies, league string, season int,
	records []playerrecord.Record) ([]pickvalue.Pick, error) {
	if season <= 2026 || bodies == nil {
		return nil, nil
	}
	if league == "" {
		return nil, fmt.Errorf("pickvalue: draft history needs a league")
	}
	byMFL := make(map[string]playerrecord.Draft, len(records))
	for _, r := range records {
		byMFL[r.MFL] = r.Draft
	}
	seen := map[int]bool{}
	var out []pickvalue.Pick
	_, err := bodies(ctx, "draftResults", func(url string, body []byte, _ time.Time) bool {
		export, id, err := leaguearchive.ParseURL(url)
		if err != nil || id != league || export.Type != draftreports.ResultsExport ||
			export.Season <= 2026 || export.Season > season || seen[export.Season] {
			return false
		}
		picks, err := draftreports.ParseResults(body)
		if err != nil {
			return false
		}
		seen[export.Season] = true
		out = append(out, draftSlots(export.Season, picks, byMFL)...)
		return false
	})
	if err != nil {
		return nil, fmt.Errorf("pickvalue: read held draft slots: %w", err)
	}
	slices.SortFunc(out, func(a, b pickvalue.Pick) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return a.Selection - b.Selection
	})
	return out, nil
}

func draftSlots(year int, picks []draftreports.Pick, byMFL map[string]playerrecord.Draft) []pickvalue.Pick {
	var out []pickvalue.Pick
	teams := 0
	for _, p := range picks {
		teams = max(teams, p.Slot)
	}
	for _, p := range picks {
		if p.Player == nil || p.Skipped {
			continue
		}
		d, ok := byMFL[p.Player.String()]
		if !ok || d.Year != year || (d.Status != playerrecord.Undrafted &&
			(d.Status != playerrecord.Drafted || d.Overall < 1)) {
			continue
		}
		out = append(out, pickvalue.Pick{Year: year, Round: p.Round, Slot: p.Slot,
			Selection: (p.Round-1)*teams + p.Slot, NFLOverall: d.Overall})
	}
	return out
}
