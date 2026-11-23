package main

import (
	"fmt"
	"strings"
)

// Level is how a check came out.
type Level string

const (
	Green   Level = "GREEN"
	Amber   Level = "AMBER"
	Red     Level = "RED"
	Unknown Level = "UNKNOWN"
)

// Result is one check's outcome in plain words, with the lines that back it up.
type Result struct {
	Name    string
	Level   Level
	Summary string
	Detail  []string
}

// Report is every check for one league and week.
type Report struct {
	League       string
	Season, Week int
	Results      []Result
}

func (r *Report) add(name string, level Level, summary string, detail []string) {
	r.Results = append(r.Results, Result{name, level, summary, detail})
}

// HasRed is true when any check found something wrong.
func (r Report) HasRed() bool {
	for _, c := range r.Results {
		if c.Level == Red {
			return true
		}
	}
	return false
}

// Summary is one line per check.
func (r Report) Summary() string {
	var b strings.Builder
	for _, c := range r.Results {
		fmt.Fprintf(&b, "%-7s %s: %s\n", c.Level, c.Name, c.Summary)
	}
	return b.String()
}

// Markdown is the detail file.
func (r Report) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Report card: %s, %d week %d\n\n", r.League, r.Season, r.Week)
	for _, c := range r.Results {
		fmt.Fprintf(&b, "## %s: %s\n\n%s\n\n", c.Level, c.Name, c.Summary)
		for _, d := range c.Detail {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		if len(c.Detail) > 0 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
