package commissioner

import (
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// draftStartFor is the start recorded for one draft, if the commissioner has set one.
func draftStartFor(records []Record, season int) (time.Time, bool) {
	active := Replay(records).Active
	var start time.Time
	found := false
	for _, r := range records {
		if r.Command.Command == DraftStart && active[r.Receipt.Entry] && ofDraft(r, rookieDraft(season)) {
			start, found = r.Command.Args.Draft.At, true
		}
	}
	return start, found
}

// projectDraftCalendar gives the calendar the draft's own start. The calendar and the draft both read the
// start through here, so a draft start can never disagree with the window that opens its picks.
func projectDraftCalendar(clock *leagueclock.ActionCalendar, records []Record, season int, at time.Time) {
	defer projectStartupCalendar(clock, records, season, at)
	start, found := draftStartFor(records, season)
	if !found {
		return
	}
	window := leagueclock.ActionWindow{ID: rookieDraftWindow, Opens: []leaguerules.Action{draftPickAction},
		Start: start.UTC()}
	for i, w := range clock.Windows {
		if w.ID == rookieDraftWindow && w.Close != nil && w.Close.After(start) {
			window.Close = w.Close
		}
		clock.Windows[i].Opens = slices.DeleteFunc(slices.Clone(w.Opens), func(a leaguerules.Action) bool {
			return a == draftPickAction
		})
	}
	clock.Windows = slices.DeleteFunc(clock.Windows, func(w leagueclock.ActionWindow) bool {
		return w.ID == rookieDraftWindow
	})
	window.Live = !at.Before(window.Start) && (window.Close == nil || at.Before(*window.Close))
	if window.Live && window.Close != nil {
		seconds := int64(window.Close.Sub(at) / time.Second)
		window.Seconds = &seconds
	}
	clock.Windows = append(clock.Windows, window)
	slices.SortFunc(clock.Windows, func(a, b leagueclock.ActionWindow) int { return strings.Compare(a.ID, b.ID) })
}

// draftSeasonPending is the earliest season with a pick still waiting to be ordered and no close recorded.
func draftSeasonPending(assets NativeAssets, through []Record) (int, bool) {
	seasons := []int{}
	for _, p := range assets.Snapshot.Picks {
		if p.Slot == 0 && !slices.Contains(seasons, p.Season) {
			seasons = append(seasons, p.Season)
		}
	}
	slices.Sort(seasons)
	for _, season := range seasons {
		if !rookieClosed(through, season) {
			return season, true
		}
	}
	return 0, false
}

// closeRecord is the close in force for one draft, if there is one.
func closeRecord(records []Record, key DraftKey) (Record, bool) {
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command == DraftArchive && active[r.Receipt.Entry] && ofDraft(r, key) {
			return r, true
		}
	}
	return Record{}, false
}

// projectStartupCalendar opens owners' picks while the startup draft runs, from its opening until it closes,
// in whichever calendar year is read: the draft is dated by those instants, not by a season's windows.
func projectStartupCalendar(clock *leagueclock.ActionCalendar, records []Record, _ int, at time.Time) {
	if len(records) == 0 {
		return
	}
	key := startupKey(records)
	open, found := startupOpened(records, key)
	if !found {
		return
	}
	window := leagueclock.ActionWindow{ID: startupDraftWindow, Opens: []leaguerules.Action{draftPickAction},
		Start: open.Receipt.RecordedAt.UTC()}
	// The window closes when the last side's startup draft does; with one pool, when the draft does.
	if startupClosed(records) {
		var end time.Time
		for _, k := range startupKeys(records) {
			closed, _ := closeRecord(records, k)
			if at := closed.Receipt.RecordedAt.UTC(); at.After(end) {
				end = at
			}
		}
		window.Close = &end
	}
	window.Live = !at.Before(window.Start) && (window.Close == nil || at.Before(*window.Close))
	clock.Windows = append(clock.Windows, window)
	slices.SortFunc(clock.Windows, func(a, b leagueclock.ActionWindow) int { return strings.Compare(a.ID, b.ID) })
}
