package facts

import "fmt"

func (c *compiler) addNote(step, src, note string, n int64) {
	if n > 0 {
		c.facts.LoadNote = append(c.facts.LoadNote, LoadNote{Step: step, Src: ptr(src), Note: note, N: n})
	}
}

func (c *compiler) loadNotes(year int64) error {
	if err := c.resultsNotes(year); err != nil {
		return err
	}
	for week := 0; week <= 17; week++ {
		src := seasonFile(year, "rosters")
		if week > 0 {
			src = fmt.Sprintf("%d/weekly/rosters_W%02d.json", year, week)
		}
		d, err := c.read(src)
		if err != nil {
			return err
		}
		seen := map[[2]string]bool{}
		var doubled int64
		for _, f := range d.get("rosters", "franchise").rows() {
			for _, p := range f.get("player").rows() {
				key := [2]string{value(f.get("id").str()), value(p.get("id").str())}
				if seen[key] {
					doubled++
				}
				seen[key] = true
			}
		}
		c.addNote("rosters", src, "player listed twice on one roster (ROSTER and IR or taxi): kept once, with the specific status", doubled)
	}
	return nil
}

func (c *compiler) resultsNotes(year int64) error {
	src := seasonFile(year, "weeklyResults_WYTD")
	d, err := c.read(src)
	if err != nil {
		return err
	}
	var dupes, byes int64
	for _, week := range d.get("allWeeklyResults", "weeklyResults").rows() {
		scores := map[string]*float64{}
		for _, matchup := range week.get("matchup").rows() {
			for _, f := range matchup.get("franchise").rows() {
				id := value(f.get("id").str())
				if _, exists := scores[id]; !exists {
					scores[id] = f.get("score").num()
				}
				if id == "BYE" {
					byes++
				}
			}
		}
		for _, f := range week.get("franchise").rows() {
			id := value(f.get("id").str())
			score, exists := scores[id]
			if exists {
				if !sameNumber(score, f.get("score").num()) {
					return fmt.Errorf("%s: standalone score differs for %s", src, id)
				}
				dupes++
			} else if id == "BYE" {
				byes++
			}
		}
	}
	c.addNote("results", src, "standalone copy of a team already in a game that week: identical score, skipped", dupes)
	c.addNote("results", src, "MFL's BYE stand-in team: not a franchise, skipped", byes)
	return nil
}

func sameNumber(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
