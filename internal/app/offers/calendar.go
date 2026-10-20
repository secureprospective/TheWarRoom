package offers

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"slices"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/lab"
	"github.com/secureprospective/TheWarRoom/internal/labcompile"
	"github.com/secureprospective/TheWarRoom/internal/store/personal"
)

// ErrOfferYear marks an offer from a league year the archive calendar does not hold yet.
var ErrOfferYear = errors.New("offer calendar: league year unavailable")

type RawOfferMarks struct {
	NFLDraft   *string `json:"nflDraft"`
	DraftStart *string `json:"draftStart"`
	DraftEnd   *string `json:"draftEnd"`
	Kickoff    *string `json:"kickoff"`
	Deadline   *string `json:"deadline"`
	RegularEnd *string `json:"regularEnd"`
	SeasonEnd  *string `json:"seasonEnd"`
}

// RawOfferCalendar holds only the dated inputs needed to classify a new proposal.
type RawOfferCalendar struct {
	Marks map[string]*string
	Weeks map[int64]int64
	dates map[string]time.Time
}

func (r *RawOfferCalendar) Validate() error {
	r.dates = make(map[string]time.Time)
	for name, text := range r.Marks {
		if text == nil {
			continue
		}
		date, err := time.Parse(time.DateOnly, *text)
		if err != nil {
			return fmt.Errorf("offer calendar %s: %w", name, err)
		}
		r.dates[name] = date
	}
	for week, ts := range r.Weeks {
		if week < 1 || week > 18 || ts <= 0 {
			return fmt.Errorf("offer calendar: invalid week")
		}
	}
	return nil
}

func LiveOffer(d *lab.Data, year int, ts int64, sender, receiver, outcome string) (lab.Offer, error) {
	s, ok := d.Seasons[year]
	if !ok {
		return lab.Offer{}, fmt.Errorf("%w: %d", ErrOfferYear, year)
	}
	var marks RawOfferMarks
	if err := json.Unmarshal(s.Marks, &marks); err != nil {
		return lab.Offer{}, fmt.Errorf("offer marks: %w", err)
	}
	raw := RawOfferCalendar{Marks: map[string]*string{"nflDraft": marks.NFLDraft, "draftStart": marks.DraftStart,
		"draftEnd": marks.DraftEnd, "kickoff": marks.Kickoff, "deadline": marks.Deadline,
		"regularEnd": marks.RegularEnd, "seasonEnd": marks.SeasonEnd}}
	if err := json.Unmarshal(s.WeekStart, &raw.Weeks); err != nil {
		return lab.Offer{}, fmt.Errorf("offer weeks: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return lab.Offer{}, err
	}
	m := labcompile.Markers{NFLDraft: raw.date("nflDraft"), DraftStart: raw.date("draftStart"),
		DraftEnd: raw.date("draftEnd"), Kickoff: raw.date("kickoff"), Deadline: raw.date("deadline"),
		RegularEnd: raw.date("regularEnd"), SeasonEnd: raw.date("seasonEnd")}
	season, phase := labcompile.OfferTiming(int64(year), ts, m, raw.Weeks)
	return lab.Offer{Ts: int(ts), Year: year, Season: int(season), Phase: phase, By: sender, To: receiver, Outcome: outcome}, nil
}

func (r RawOfferCalendar) date(name string) *time.Time {
	d, ok := r.dates[name]
	if !ok {
		return nil
	}
	return &d
}

func Merge(d *lab.Data, rows []personal.SeenOffer, checked int64) (*lab.Data, string, time.Time, error) {
	owner := lab.OfferOwner(d)
	cutoff, err := time.Parse(time.RFC3339, d.ArchiveThrough)
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("offers: archive cutoff: %w", err)
	}
	merged := *d
	merged.Offers = slices.Clone(d.Offers)
	merged.OfferOwnerID = owner
	unplaced := 0
	for _, row := range rows {
		if row.Timestamp <= cutoff.Unix() || row.Sender != owner && row.Receiver != owner {
			continue
		}
		if slices.ContainsFunc(merged.Offers, func(o lab.Offer) bool {
			return o.Ts == int(row.Timestamp) && o.By == row.Sender && o.To == row.Receiver
		}) {
			continue
		}
		offer, err := LiveOffer(d, row.Year, row.Timestamp, row.Sender, row.Receiver, row.Outcome)
		if errors.Is(err, ErrOfferYear) {
			// Its stretch needs that year's dates; it is kept and counted once they are held.
			unplaced++
			continue
		}
		if err != nil {
			return nil, "", time.Time{}, err
		}
		merged.Offers = append(merged.Offers, offer)
	}
	source := "League archive through " + d.ArchiveThrough
	var last time.Time
	if checked != 0 {
		last = time.Unix(checked, 0)
		source += "; offers seen live since, last checked " + time.Unix(checked, 0).UTC().Format(time.RFC3339)
	} else {
		source += "; live offers not checked yet"
	}
	if unplaced > 0 {
		source += fmt.Sprintf("; %s from a league year without held dates not placed in a stretch yet",
			report.Count(unplaced, "offer"))
	}
	return &merged, source, last, nil
}
