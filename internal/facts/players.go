package facts

import (
	"context"
	"fmt"
	"strconv"
	"time"
	_ "time/tzdata" // Birthdates must not depend on the host timezone database.
)

type playerIdentity struct {
	fields map[string]*string
	first  int64
	last   int64
	src    string
}

func (c *compiler) players(ctx context.Context) error {
	best := map[string]playerIdentity{}
	for _, year := range c.years {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("compile players: %w", err)
		}
		src := seasonFile(year, "players_DETAILS1")
		d, err := c.read(src)
		if err != nil {
			return err
		}
		if d.Object == nil {
			src = seasonFile(year, "players")
			d, err = c.read(src)
			if err != nil {
				return err
			}
		}
		for _, p := range d.get("players", "player").rows() {
			pid, err := requiredID(p, "id", src)
			if err != nil {
				return err
			}
			c.facts.PlayerSeason = append(c.facts.PlayerSeason, PlayerSeason{
				Year: year, Pid: pid, Name: p.get("name").str(), Position: p.get("position").str(), NflTeam: p.get("team").str(), Src: src,
			})
			c.mergePlayer(best, pid, p, year, src)
		}
	}
	for pid, rec := range best {
		if rec.fields["name"] == nil {
			return fmt.Errorf("player %s has no name", pid)
		}
		c.facts.Player = append(c.facts.Player, playerRow(pid, rec, c.eastern))
	}
	return nil
}

func (c *compiler) mergePlayer(best map[string]playerIdentity, pid string, p RawValue, year int64, src string) {
	fields := []string{"name", "position", "birthdate", "draft_year", "draft_team", "draft_round", "draft_pick", "college", "height", "weight"}
	conflict := map[string]bool{"name": true, "birthdate": true, "draft_year": true, "draft_team": true, "draft_round": true, "draft_pick": true, "college": true}
	old, exists := best[pid]
	rec := playerIdentity{fields: make(map[string]*string, len(fields)), first: year, last: year, src: src}
	if exists {
		rec.first = old.first
	}
	for _, field := range fields {
		v := nonempty(p.get(field).str())
		previous := old.fields[field]
		if conflict[field] && previous != nil && v != nil && *previous != *v {
			c.facts.PlayerConflict = append(c.facts.PlayerConflict, PlayerConflict{Pid: pid, Field: field, Year: old.last, Value: previous, Kept: v})
		}
		if v == nil {
			v = previous
		}
		rec.fields[field] = v
	}
	best[pid] = rec
}

func playerRow(pid string, r playerIdentity, eastern *time.Location) Player {
	f := r.fields
	whole := func(key string) *int64 { return (RawValue{Scalar: f[key]}).whole() }
	draftYear := whole("draft_year")
	if draftYear != nil && *draftYear == 0 {
		draftYear = nil
	}
	return Player{
		Pid: pid, Name: value(f["name"]), Position: f["position"], Birthdate: birthdate(f["birthdate"], eastern),
		BirthdateTs: whole("birthdate"), DraftYear: draftYear, DraftTeam: f["draft_team"], DraftRound: whole("draft_round"),
		DraftPick: whole("draft_pick"), College: f["college"], Height: f["height"], Weight: f["weight"],
		FirstYear: ptr(r.first), LastYear: ptr(r.last), Src: r.src,
	}
}

func birthdate(s *string, eastern *time.Location) *string {
	ts, err := strconv.ParseInt(value(s), 10, 64)
	if err != nil || ts == 0 {
		return nil
	}
	return ptr(time.Unix(ts, 0).In(eastern).Format("2006-01-02"))
}
