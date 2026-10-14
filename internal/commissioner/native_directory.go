package commissioner

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type NativeDirectorySeat struct {
	PeopleSeat
	Division string `json:"division"`
}

type NativeDirectory struct {
	League       string                `json:"league"`
	Name         string                `json:"name"`
	Revision     int64                 `json:"revision"`
	RosterSeason int                   `json:"rosterSeason"`
	Source       string                `json:"source"`
	Franchises   []NativeDirectorySeat `json:"franchises"`
}

// NativeDirectoryReading keeps recorded starting identities separate from the held MFL directory.
func NativeDirectoryReading(league string, records []Record, now time.Time) (NativeDirectory, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return NativeDirectory{}, err
	}
	return nativeDirectory(league, records, state, now)
}

func nativeDirectory(league string, records []Record, state governance.State, now time.Time) (NativeDirectory, error) {
	assets, err := NativeAssetsReading(league, records)
	if err != nil {
		return NativeDirectory{}, err
	}
	rules, err := RulesNow(records, now)
	if err != nil {
		return NativeDirectory{}, err
	}
	out := NativeDirectory{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		Source:     "App league; franchise identities and divisions are never synced with MFL",
		Franchises: []NativeDirectorySeat{}}
	seats := []PeopleSeat{}
	for _, r := range records {
		if r.Command.Command == Create {
			seats = readSeats(state, r.Command.Args.Franchises)
		}
	}
	alignment := rules.League.Alignment
	if assets.Seeded {
		out.RosterSeason = assets.Snapshot.Season
	}
	applyNativeNames(seats, records)
	for _, seat := range seats {
		division := "Needs a ruling"
		if value := alignment.Value[seat.Franchise]; alignment.State == leaguerules.IsKnown && value != "" {
			division = value
		}
		out.Franchises = append(out.Franchises, NativeDirectorySeat{PeopleSeat: seat, Division: division})
	}
	return out, nil
}

func applyNativeNames(seats []PeopleSeat, records []Record) {
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command == NativeSeed && r.Command.Args.Native != nil {
			for _, f := range r.Command.Args.Native.Franchises {
				setNativeName(seats, f.ID, f.Name, f.Abbreviation)
			}
		}
		if r.Command.Command == "commish.franchise.edit" && active[r.Receipt.Entry] {
			p := r.Command.Args.People
			setNativeName(seats, p.Franchise, p.Name, p.Abbreviation)
		}
	}
}

func setNativeName(seats []PeopleSeat, id, name, abbreviation string) {
	for i := range seats {
		if seats[i].Franchise == id {
			seats[i].Name, seats[i].Abbreviation = name, abbreviation
		}
	}
}
