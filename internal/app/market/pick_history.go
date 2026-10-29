package market

import (
	"context"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/m2service"
	"github.com/secureprospective/TheWarRoom/internal/model"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/internal/store/params"

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
		if !ok || !rookieCapital(d, year) {
			continue
		}
		out = append(out, pickvalue.Pick{Year: year, Round: p.Round, Slot: p.Slot,
			Selection: (p.Round-1)*teams + p.Slot, NFLOverall: d.Overall})
	}
	return out
}

// PickTotals reads the next two drafts without substituting zero for unreadable inventory or
// an unreadable value. One bad pick blanks its owner's total, not any other franchise's.
func PickTotals(model Model, inventory []draftreports.FuturePick, season int, owners []string, at time.Time) map[string]float64 {
	return pickCapital(model, inventory, season, owners, at).Totals
}

func pickCapital(values Model, inventory []draftreports.FuturePick, season int, owners []string, at time.Time) m2service.PickCapital {
	capital := m2service.PickCapital{}
	if values == nil || inventory == nil {
		return capital
	}
	capital.Totals = make(map[string]float64, len(owners))
	capital.Status = make(map[string]model.InputStatus, len(owners))
	for _, id := range owners {
		capital.Totals[id] = 0
	}
	for _, p := range inventory {
		if p.Year <= season || p.Year > season+2 {
			continue
		}
		v := values.Pick(p.Year, p.Round, p.Original, at)
		status := capital.Status[p.Owner]
		status.Merge(model.InputStatus{ReducedInputs: v.ReducedInputs, Reason: v.Reason})
		if !v.Valued || math.IsNaN(v.Discounted) || math.IsInf(v.Discounted, 0) || v.Discounted < 0 {
			status.Unavailable = true
			status.ReducedInputs = true
			if status.Reason == "" {
				status.Reason = "A future pick has no usable value"
			}
		} else {
			capital.Totals[p.Owner] += v.Discounted
		}
		capital.Status[p.Owner] = status
	}
	for id, status := range capital.Status {
		if status.Unavailable {
			delete(capital.Totals, id)
		}
	}
	return capital
}

// NativePickHistory uses the replayed rookie selections, not raw ledger commands: corrections
// replace players and restored/skipped picks cannot train the curve. Orders follow originals.
func NativePickHistory(drafts []commissioner.NativeDraft, season int, records []playerrecord.Record) ([]pickvalue.Pick, map[PickKey]int) {
	byMFL := make(map[string]playerrecord.Draft, len(records))
	for _, r := range records {
		byMFL[r.MFL] = r.Draft
	}
	var out []pickvalue.Pick
	slots := map[PickKey]int{}
	for _, draft := range drafts {
		if draft.Startup {
			continue
		}
		teams := 0
		for _, round := range draft.Order {
			teams = max(teams, len(round.Picks))
			for _, p := range round.Picks {
				slots[PickKey{Year: draft.Season, Round: p.Round, Original: p.Original}] = p.Slot
			}
		}
		if draft.Season > season {
			continue
		}
		for _, p := range draft.Draft.Picks {
			d, ok := byMFL[p.Player]
			if !ok || p.Player == "" || p.Skipped || p.Forfeited || p.Extra > 0 || !rookieCapital(d, draft.Season) {
				continue
			}
			out = append(out, pickvalue.Pick{Year: draft.Season, Round: p.Round, Slot: p.Slot, Selection: (p.Round-1)*teams + p.Slot, NFLOverall: d.Overall})
		}
	}
	return out, slots
}

func rookieCapital(d playerrecord.Draft, year int) bool {
	return d.Year == year && (d.Status == playerrecord.Undrafted || (d.Status == playerrecord.Drafted && d.Overall > 0))
}

// PowerPickSource prices current inventory against one named model run and the live settings.
type PowerPickSource struct {
	Native    NativeSource
	Params    *params.Store
	Inventory []draftreports.FuturePick
	Owners    []string
	At        time.Time
}

func PowerPicks(ctx context.Context, run history.Run, src PowerPickSource) m2service.PickCapital {
	capital := m2service.PickCapital{}
	if src.Params == nil {
		return capital
	}
	capital.Weight, _ = src.Params.Snapshot().GetGlobal(params.KeyPowerPickWeight)
	src.Native.LatestRun = func(context.Context) (history.Run, bool, error) { return run, true, nil }
	model, ok, err := LoadNative(ctx, src.Native)
	if err != nil || !ok {
		return capital
	}
	priced := pickCapital(model, src.Inventory, run.Season, src.Owners, src.At)
	capital.Totals, capital.Status = priced.Totals, priced.Status
	return capital
}
