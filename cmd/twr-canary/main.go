// Command twr-canary walks a seeded subset of owner journeys against an allowlisted Lab league.
package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/serverclient"
)

type cliOptions struct {
	server, tokenFile, league, franchise, seedFlag, out, allow, target, confirmation string
	season, week                                                                     int
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, nil)) }

func parseOptions(args []string, stderr io.Writer) (cliOptions, bool) {
	fs := flag.NewFlagSet("twr-canary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o cliOptions
	fs.StringVar(&o.server, "server", "", "Lab league server https:// address")
	fs.StringVar(&o.tokenFile, "token-file", "", "connection token file")
	fs.StringVar(&o.league, "league", "", "canary league ID (native.…)")
	fs.StringVar(&o.franchise, "franchise", "", "canary franchise ID (franchise.…)")
	fs.IntVar(&o.season, "season", 0, "season")
	fs.IntVar(&o.week, "week", 0, "week")
	fs.StringVar(&o.seedFlag, "seed", "", "replay seed YEARWEEK-hexnonce")
	fs.StringVar(&o.out, "out", "", "markdown report path")
	fs.StringVar(&o.allow, "allow-file", "", "one allowlisted native league ID per line")
	fs.StringVar(&o.target, "target", "", "must be lab")
	fs.StringVar(&o.confirmation, "confirmation", "", "optional human confirmation code for G2/G3 execute")
	if fs.Parse(args) != nil {
		return o, false
	}
	return o, true
}

func validateTargetAndAllowlist(o cliOptions, stderr io.Writer) bool {
	if o.target != "lab" {
		_, _ = fmt.Fprintln(stderr, "refusing: -target must be lab")
		return false
	}
	if o.server == "" || o.tokenFile == "" || o.league == "" || o.season < 1 || o.week < 1 || o.allow == "" {
		_, _ = fmt.Fprintln(stderr, "usage: twr-canary -target lab -server https://host -token-file F -league native.id [-franchise franchise.01] -season N -week N -allow-file F")
		return false
	}
	if !strings.HasPrefix(o.league, "native.") {
		_, _ = fmt.Fprintln(stderr, "refusing: only native leagues are allowed")
		return false
	}
	allowData, err := os.ReadFile(o.allow)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "can't read allowlist:", err)
		return false
	}
	for line := range strings.SplitSeq(string(allowData), "\n") {
		if strings.TrimSpace(line) == o.league {
			return true
		}
	}
	_, _ = fmt.Fprintln(stderr, "refusing: league is not on the allowlist")
	return false
}

func loadAgent(o cliOptions, roots *x509.CertPool, stderr io.Writer) (*serverclient.Agent, bool) {
	raw, err := os.ReadFile(o.tokenFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "can't read token file:", err)
		return nil, false
	}
	bearer := strings.TrimSpace(string(raw))
	if _, parsed, e := serverclient.ParseHelperLink(bearer); e == nil {
		bearer = parsed
	}
	agent, err := serverclient.NewAgent(o.server, o.league, bearer, roots)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return nil, false
	}
	return agent, true
}

func run(args []string, stdout, stderr io.Writer, roots *x509.CertPool) int {
	opts, ok := parseOptions(args, stderr)
	if !ok || !validateTargetAndAllowlist(opts, stderr) {
		return 2
	}
	seed, nonce, rng, err := makeSeed(opts.seedFlag, opts.season, opts.week)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	seedText := fmt.Sprintf("%d-%04x", seed, nonce)
	first := "seed=" + seedText
	_, _ = fmt.Fprintln(stdout, first)
	agent, ok := loadAgent(opts, roots, stderr)
	if !ok {
		return 2
	}
	cfg := canaryConfig{
		target:       opts.target,
		league:       opts.league,
		franchise:    opts.franchise,
		seedLabel:    seedText,
		confirmation: opts.confirmation,
		season:       opts.season,
		week:         opts.week,
		allowlisted:  true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	selected := selectJourneys(rng)
	lines, red := executeJourneys(ctx, agent, cfg, selected, first, stdout)
	if opts.out != "" {
		body := "# Lab canary\n\n" + strings.Join(lines, "\n") + "\n"
		if e := os.WriteFile(opts.out, []byte(body), 0o600); e != nil {
			_, _ = fmt.Fprintln(stderr, "can't write report:", e)
			return 2
		}
	}
	if red {
		return 1
	}
	return 0
}

func executeJourneys(ctx context.Context, agent *serverclient.Agent, cfg canaryConfig, selected []string, first string, stdout io.Writer) ([]string, bool) {
	byID := map[string]journey{}
	all := journeys()
	for _, j := range all {
		byID[j.id] = j
	}
	chosen := map[string]bool{}
	lines := []string{first}
	red := false
	for _, id := range selected {
		chosen[id] = true
		j := byID[id]
		started := time.Now()
		status, detail := runOneJourney(ctx, agent, cfg, j)
		if status == "FAIL" {
			red = true
		}
		line := fmt.Sprintf("%s %s [%s] %dms %s", status, j.name, j.command, time.Since(started).Milliseconds(), detail)
		lines = append(lines, line)
		_, _ = fmt.Fprintln(stdout, line)
	}
	for _, j := range all {
		if chosen[j.id] {
			continue
		}
		detail := "not selected by this seed"
		if j.excluded && j.reason != "" {
			detail = j.reason
		}
		line := fmt.Sprintf("SKIPPED %s [%s] 0ms %s", j.name, j.command, detail)
		lines = append(lines, line)
		_, _ = fmt.Fprintln(stdout, line)
	}
	return lines, red
}

func selectJourneys(rng *rand.Rand) []string {
	selected := append([]string{}, alwaysJourneys()...)
	chosen := map[string]bool{}
	for _, id := range selected {
		chosen[id] = true
	}
	var pool []journey
	for _, j := range journeys() {
		if !chosen[j.id] && !j.excluded {
			pool = append(pool, j)
		}
	}
	for n := 0; n < 5 && len(pool) > 0; n++ {
		idx := rng.IntN(len(pool))
		selected = append(selected, pool[idx].id)
		pool = append(pool[:idx], pool[idx+1:]...)
	}
	return selected
}

func makeSeed(input string, season, week int) (int64, uint16, *rand.Rand, error) {
	var seed int64
	var nonce uint64
	if input == "" {
		seed = int64(season*100 + week)
		var random [2]byte
		if _, err := cryptorand.Read(random[:]); err != nil {
			return 0, 0, nil, fmt.Errorf("generate run nonce: %w", err)
		}
		nonce = uint64(random[0])<<8 | uint64(random[1])
	} else {
		parts := strings.Split(input, "-")
		if len(parts) != 2 {
			return 0, 0, nil, fmt.Errorf("seed must be YEARWEEK-hexnonce")
		}
		base, e := strconv.ParseInt(parts[0], 10, 64)
		if e != nil {
			return 0, 0, nil, fmt.Errorf("invalid seed")
		}
		n, e := strconv.ParseUint(parts[1], 16, 16)
		if e != nil {
			return 0, 0, nil, fmt.Errorf("invalid seed nonce")
		}
		seed, nonce = base, n
	}
	//nolint:gosec // Seed is a positive YEARWEEK; preserving its bits is intentional.
	mixed := uint64(seed) ^ nonce
	//nolint:gosec // Deterministic selection is required for exact replay.
	return seed, uint16(nonce), rand.New(rand.NewPCG(mixed, mixed^0x9e3779b97f4a7c15)), nil
}
