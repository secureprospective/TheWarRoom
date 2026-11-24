// Command twr-canary walks a seeded subset of owner journeys against an allowlisted Lab league.
package main

import (
	"context"
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
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

type journey struct {
	id, name, command string
	scope             serverhttp.RawWorkflowScope
	reason            string
}

func journeys() []journey {
	return []journey{
		{"1", "connect", "", serverhttp.RawWorkflowScope{}, "pre-minted token is supplied; sign-in is human-only"},
		{"2", "week and lineup", "week.read", serverhttp.RawWorkflowScope{}, ""},
		{"3", "set lineup", "commish.lineup.set", serverhttp.RawWorkflowScope{}, "no safe fixture-independent lineup payload"},
		{"4", "live score", "nfl.games.read", serverhttp.RawWorkflowScope{}, ""},
		{"5", "trade propose", "trade.offer", serverhttp.RawWorkflowScope{}, "no safe fixture-independent trade payload"},
		{"6", "trade accept", "trade.accept", serverhttp.RawWorkflowScope{}, "requires second owner token"},
		{"7", "trade review", "trade.review", serverhttp.RawWorkflowScope{}, "commissioner confirmation is human-only; preview arguments unavailable"},
		{"8", "waiver claim", "waivers.claim", serverhttp.RawWorkflowScope{}, "preview payload requires a seeded player identity"},
		{"9", "waiver processing", "waivers.process", serverhttp.RawWorkflowScope{}, "timed commissioner operation"},
		{"10", "free-agent bid", "bid.place", serverhttp.RawWorkflowScope{}, "no safe fixture-independent bid payload"},
		{"11", "roster change", "roster.cut", serverhttp.RawWorkflowScope{}, "no safe fixture-independent roster payload"},
		{"12", "week final", "results.read", serverhttp.RawWorkflowScope{}, ""},
		{"13", "standings", "standings.read", serverhttp.RawWorkflowScope{}, ""},
		{"14", "playoff bracket", "bracket.read", serverhttp.RawWorkflowScope{}, "read door not confirmed available"},
		{"15", "weekly recap", "", serverhttp.RawWorkflowScope{}, "no command exists"},
		{"16", "league feed", "league.activity.read", serverhttp.RawWorkflowScope{}, "read door not confirmed available"},
		{"17", "chat", "", serverhttp.RawWorkflowScope{}, "no command exists"},
		{"18", "notification delivery", "", serverhttp.RawWorkflowScope{}, "no command exists"},
		{"19", "commissioner act", "commish.notice.publish", serverhttp.RawWorkflowScope{}, "preview payload unavailable; never execute commissioner changes"},
		{"20", "rule read-back", "commish.rules.read", serverhttp.RawWorkflowScope{}, "read door not confirmed available"},
		{"21", "stat correction", "commish.stat.correct", serverhttp.RawWorkflowScope{}, "preview payload unavailable"},
		{"22", "invite and join", "", serverhttp.RawWorkflowScope{}, "human-only token boundary"},
		{"23", "draft pick trade", "trade.draft.read", serverhttp.RawWorkflowScope{}, "read door not confirmed available"},
		{"24", "rookie draft", "draft.queue.read", serverhttp.RawWorkflowScope{}, "read door not confirmed available"},
	}
}
func alwaysJourneys() []string { return []string{"1", "2", "3", "4", "8", "13", "20"} }

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, nil)) }

//nolint:gocyclo,funlen // The CLI boundary owns option validation, safety refusals, run and exit codes.
func run(args []string, stdout, stderr io.Writer, roots *x509.CertPool) int {
	fs := flag.NewFlagSet("twr-canary", flag.ContinueOnError)
	fs.SetOutput(stderr)
	server := fs.String("server", "", "Lab league server https:// address")
	tokenFile := fs.String("token-file", "", "connection token file")
	league := fs.String("league", "", "canary league ID (native.…)")
	season := fs.Int("season", 0, "season")
	week := fs.Int("week", 0, "week")
	seedFlag := fs.String("seed", "", "replay seed YEARWEEK-hexnonce")
	out := fs.String("out", "", "markdown report path")
	allow := fs.String("allow-file", "", "one allowlisted native league ID per line")
	target := fs.String("target", "", "must be lab")
	if fs.Parse(args) != nil {
		return 2
	}
	if *target != "lab" {
		fmt.Fprintln(stderr, "refusing: -target must be lab")
		return 2
	}
	if *server == "" || *tokenFile == "" || *league == "" || *season < 1 || *week < 1 || *allow == "" {
		fmt.Fprintln(stderr, "usage: twr-canary -target lab -server https://host -token-file F -league native.id -season N -week N -allow-file F")
		return 2
	}
	if !strings.HasPrefix(*league, "native.") {
		fmt.Fprintln(stderr, "refusing: only native leagues are allowed")
		return 2
	}
	allowData, err := os.ReadFile(*allow)
	if err != nil {
		fmt.Fprintln(stderr, "can't read allowlist:", err)
		return 2
	}
	listed := false
	for _, line := range strings.Split(string(allowData), "\n") {
		if strings.TrimSpace(line) == *league {
			listed = true
		}
	}
	if !listed {
		fmt.Fprintln(stderr, "refusing: league is not on the allowlist")
		return 2
	}
	seed, nonce, rng, err := makeSeed(*seedFlag, *season, *week)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	first := fmt.Sprintf("seed=%d-%04x", seed, nonce)
	fmt.Fprintln(stdout, first)
	raw, err := os.ReadFile(*tokenFile)
	if err != nil {
		fmt.Fprintln(stderr, "can't read token file:", err)
		return 2
	}
	bearer := strings.TrimSpace(string(raw))
	if _, parsed, e := serverclient.ParseHelperLink(bearer); e == nil {
		bearer = parsed
	}
	agent, err := serverclient.NewAgent(*server, *league, bearer, roots)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	chosen := selectJourneys(rng)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	lines := []string{first}
	red := false
	for _, j := range journeys() {
		if !chosen[j.id] {
			continue
		}
		started := time.Now()
		status, detail := "SKIPPED", j.reason
		if j.id == "1" {
			status, detail = "PASS", "token accepted by agent constructor"
		} else if j.reason == "" && j.command != "" {
			data, e := agent.Read(ctx, j.command, serverhttp.RawWorkflowScope{Season: *season, Week: *week})
			switch {
			case e != nil:
				status, detail, red = "FAIL", e.Error(), true
			case len(data) == 0 || string(data) == "null":
				status, detail, red = "FAIL", "empty response", true
			default:
				status, detail = "PASS", "read response received"
			}
		}
		line := fmt.Sprintf("%s %s [%s] %dms %s", status, j.name, j.command, time.Since(started).Milliseconds(), detail)
		lines = append(lines, line)
		fmt.Fprintln(stdout, line)
	}
	if *out != "" {
		body := "# Lab canary\n\n" + strings.Join(lines, "\n") + "\n"
		//nolint:gosec // The operator explicitly chooses the report destination.
		if e := os.WriteFile(*out, []byte(body), 0o600); e != nil {
			fmt.Fprintln(stderr, "can't write report:", e)
			return 2
		}
	}
	if red {
		return 1
	}
	return 0
}
func selectJourneys(rng *rand.Rand) map[string]bool {
	chosen := map[string]bool{}
	for _, id := range alwaysJourneys() {
		chosen[id] = true
	}
	var pool []journey
	for _, j := range journeys() {
		if !chosen[j.id] && j.id != "15" && j.id != "17" && j.id != "18" && j.id != "22" {
			pool = append(pool, j)
		}
	}
	for n := 0; n < 5 && n < len(pool); n++ {
		idx := rng.IntN(len(pool))
		chosen[pool[idx].id] = true
		pool = append(pool[:idx], pool[idx+1:]...)
	}
	return chosen
}

func makeSeed(input string, season, week int) (int64, uint16, *rand.Rand, error) {
	var seed int64
	var nonce uint64
	if input == "" {
		seed = int64(season*100 + week)
		nonce = uint64(time.Now().UnixNano()) & 0xffff
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
