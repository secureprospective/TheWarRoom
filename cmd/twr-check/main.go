// Command twr-check is the Tuesday report card: it reads a league's week over a read-only token and
// says in plain words whether the published scores and standings add up. It never changes a league.
//
//	twr-check -server https://host -token-file F -league native.x -season 2026 -week 14 [-out detail.md]
//
// Exit 0: nothing red. Exit 1: at least one red check. Exit 2: could not connect or the token was refused.
package main

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverclient"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, nil))
}

// run is main with its outside world passed in; roots nil trusts the system's certificates.
func run(args []string, stdout, stderr io.Writer, roots *x509.CertPool) int {
	fs := flag.NewFlagSet("twr-check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	server := fs.String("server", "", "the league server's https:// address")
	tokenFile := fs.String("token-file", "", "file holding a read-only connection token")
	league := fs.String("league", "", "league ID (native.…)")
	season := fs.Int("season", 0, "season, for example 2026")
	week := fs.Int("week", 0, "week to check")
	out := fs.String("out", "", "write the markdown detail here")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *server == "" || *tokenFile == "" || *league == "" || *season == 0 || *week < 1 {
		_, _ = fmt.Fprintln(stderr, "usage: twr-check -server https://host -token-file F -league native.x -season 2026 -week 14 [-out detail.md]")
		return 2
	}
	raw, err := os.ReadFile(*tokenFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "can't read the token file:", err)
		return 2
	}
	agent, err := serverclient.NewAgent(*server, *league, strings.TrimSpace(string(raw)), roots)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	report, err := Check(ctx, agent, *season, *week)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	report.League, report.Season, report.Week = *league, *season, *week
	_, _ = fmt.Fprint(stdout, report.Summary())
	if *out != "" {
		if err := os.WriteFile(*out, []byte(report.Markdown()), 0o600); err != nil { //nolint:gosec // the operator chose this path
			_, _ = fmt.Fprintln(stderr, "can't write the detail file:", err)
			return 2
		}
	}
	if report.HasRed() {
		return 1
	}
	return 0
}
