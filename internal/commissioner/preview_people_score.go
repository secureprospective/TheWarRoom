package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func peoplePreviewValues(raw RawCommand, records []Record) (map[string]string, error) {
	out := map[string]string{}
	reading, err := PeopleReading(raw.League, Replay(records).Creator, records)
	if err != nil {
		return nil, err
	}
	for _, s := range reading.Seats {
		out[s.Franchise+" name"] = s.Name + " (" + s.Abbreviation + ")"
		out[s.Franchise+" owner"] = personWords(s.Owner)
	}
	for _, i := range reading.Invites {
		out["Invitation "+i.ID] = i.Person + ": " + i.Status
	}
	out["Commissioner succession"] = reading.Succession
	for _, v := range reading.Votes {
		out["Commissioner vote "+v.ID] = fmt.Sprintf("%s; %d ballots cast; closes %s", v.Status,
			v.BallotsCast, v.EndsAt.Format(time.RFC3339))
		for _, c := range v.Counts {
			out["Vote "+v.ID+" for "+personWords(c.Choice)] = fmt.Sprintf("%d ballots", c.Ballots)
		}
	}
	state, err := GovernanceState(records)
	if err != nil {
		return nil, err
	}
	order := []string{}
	for _, person := range state.CoCommissioners() {
		order = append(order, personWords(string(person)))
	}
	out["Co-commissioner priority"] = strings.Join(order, ", ")
	p := raw.Args.People
	inSeason, offseason, election, runoff := state.SuccessionPeriods()
	out["In-season commissioner inactivity period"] = leaguerules.DurationWords(inSeason)
	out["Offseason commissioner inactivity period"] = leaguerules.DurationWords(offseason)
	out["Commissioner election voting period"] = leaguerules.DurationWords(election)
	out["Commissioner runoff voting period"] = leaguerules.DurationWords(runoff)
	if raw.Command == "commish.succession.activity" {
		out["Last activity of "+personWords(p.Principal)] = state.LastActivity(governance.Principal(p.Principal)).Format(time.RFC3339)
	}

	peoplePermissionValues(raw, reading, state, out)
	for _, r := range reading.Roles {
		out[personWords(r.Person)+" — "+r.Role] = "Holds the role"
	}
	return out, nil
}

func scorePreviewValues(raw RawCommand, records []Record) (map[string]string, error) {
	c := raw.Args.Score
	season, week := c.Season, c.Week
	changes, err := scoreChanges(records, c.Player, c.Franchise, season, week)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	if raw.Command == PlayerPosition || raw.Command == PlayerPositionClear {
		position := string(changes.Position)
		if position == "" {
			position = "Use the position reported with the NFL statistics"
		}
		out["Player "+c.Player+" scoring position"] = position
	} else {
		target := c.Franchise
		if target == "" {
			target = "Player " + c.Player
		}
		out[fmt.Sprintf("%s, season %d, week %d points adjustments", target, season, week)] =
			scorePointsWords(changes.Delta) + " points"
	}
	return out, nil
}

func peoplePermissionValues(raw RawCommand, reading People, state governance.State, out map[string]string) {
	p := raw.Args.People
	if raw.Command == "commish.abilities.set" {
		for _, s := range reading.Seats {
			if !p.All && s.Franchise != p.Franchise {
				continue
			}
			seat, _ := state.Franchise(governance.FranchiseID(s.Franchise))
			words := "Not allowed"
			if seat.Abilities[p.Ability] {
				words = "Allowed"
			}
			out[s.Name+" — "+p.Ability] = words
		}
	}
	if raw.Command == "commish.visibility.set" {
		audience := state.Audience(p.Report)
		if audience == "" {
			audience = "Default league audience"
		}
		out[p.Report+" audience"] = audience
	}
}
