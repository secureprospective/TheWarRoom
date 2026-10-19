package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func calendarPreviewValues(raw RawCommand, records []Record, now time.Time) (map[string]string, error) {
	e := raw.Args.Exception
	reading, err := CalendarReading(raw.League, records, now, now.Year(), e.Action, e.Franchise, time.UTC)
	if err != nil {
		return nil, err
	}
	out := map[string]string{e.Franchise + " — " + string(e.Action): reading.Decision.Reason,
		"Window exception": "No active exception"}
	exceptions, err := readCalendarExceptions(records)
	if err != nil {
		return nil, err
	}
	for _, v := range exceptions {
		if v.Action != e.Action || v.Franchise != e.Franchise || !v.Until.After(now) {
			continue
		}
		out["Window exception"] = fmt.Sprintf("Allowed until %s: %s", v.Until.Format(time.RFC3339), v.Reason)
	}
	return out, nil
}

func ReservePreviewValues(raw RawCommand, records []Record, now time.Time, facts nativecheck.Facts, out map[string]string) error {
	if raw.Command != ReserveException {
		return nil
	}
	view, err := NativeReserveCheckReading(raw.League, records, now, facts)
	if err != nil {
		return err
	}
	for _, f := range view.Franchises {
		for _, p := range f.Players {
			if p.Player != raw.Args.Override.Player {
				continue
			}
			out["Player "+p.Player+" — "+p.Squad+" eligibility"] = p.Words
		}
	}
	return nil
}
