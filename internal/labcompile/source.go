package labcompile

import (
	_ "embed"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/facts"
)

//go:embed testdata/check-exceptions.csv
var checkExceptions []byte

type SourceCheck struct {
	Name      string `json:"name"`
	Checked   int    `json:"checked"`
	Agreed    int    `json:"agreed"`
	Explained int    `json:"explained"`
	Open      int    `json:"open"`
	Detail    string `json:"detail"`
}
type Source struct {
	Checks []SourceCheck `json:"checks"`
	Pages  PageExam      `json:"pages"`
}

func sourceChecks(f facts.Facts, pages PageExam) (Source, error) {
	pinned, err := facts.ParseExceptions(checkExceptions)
	if err != nil {
		return Source{}, fmt.Errorf("lab source exceptions: %w", err)
	}
	checks, err := sourceFacts(f).Check(pinned)
	if err != nil {
		return Source{}, fmt.Errorf("lab source checks: %w", err)
	}
	out := Source{Pages: pages, Checks: make([]SourceCheck, 0, len(checks))}
	for _, c := range checks {
		out.Checks = append(out.Checks, SourceCheck{c.Name, c.Checked, c.Agreed, c.Explained, c.Review, sourceDetail(c)})
	}
	return out, nil
}

func sourceDetail(c facts.CheckResult) string {
	switch c.Name {
	case "Win-loss records match MFL standings":
		return fmt.Sprintf("%d in 2013, when MFL counted playoff games in the standings.", c.Explained)
	case "Team scores equal their starters":
		return fmt.Sprintf("%d games carry MFL's 3-point home bonus (2014-2015 regular season, and playoff games).", c.Explained)
	}
	return c.Detail
}

// A franchise directory alone is not evidence of a scored team-season.
// Keep partially observed seasons: their discrepancies must still pass the identity pins.
func sourceFacts(f facts.Facts) facts.Facts {
	observed := map[teamYear]bool{}
	for i, y := range f.Game.Year {
		observed[teamYear{y, f.Game.Fid[i]}] = true
	}
	for i, y := range f.Standings.Year {
		observed[teamYear{y, f.Standings.Fid[i]}] = true
	}
	original := f.FranchiseSeason
	scoped := facts.FranchiseSeasonColumns{}
	for i, y := range original.Year {
		if !observed[teamYear{y, original.Fid[i]}] {
			continue
		}
		scoped.Year = append(scoped.Year, y)
		scoped.Fid = append(scoped.Fid, original.Fid[i])
		scoped.Name = append(scoped.Name, original.Name[i])
		scoped.Abbrev = append(scoped.Abbrev, original.Abbrev[i])
		scoped.Division = append(scoped.Division, original.Division[i])
		scoped.DivisionName = append(scoped.DivisionName, original.DivisionName[i])
		scoped.Conference = append(scoped.Conference, original.Conference[i])
		scoped.ConferenceName = append(scoped.ConferenceName, original.ConferenceName[i])
		scoped.Src = append(scoped.Src, original.Src[i])
	}
	f.FranchiseSeason = scoped
	return f
}
