// Package commstest loads the shared archived communications fixtures.
package commstest

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/comms"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/rosters"
	feedtransactions "github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func Fixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dataRoot(), "testdata", "comms", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func Real(t *testing.T) comms.Input {
	t.Helper()
	cfg, err := league.Parse(commsData(t, "league", "league-2026.json"), commsData(t, "league", "rules-2025.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := feedtransactions.Parse(commsData(t, "transactions", "transactions.json"))
	if err != nil {
		t.Fatal(err)
	}
	txs, err := feedtransactions.ToTransactions(raw)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(Fixture(t, "players-2026.json.gz")))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	rawPlayers, err := players.Parse(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	lookup, err := normalize.NewLookup(rawPlayers)
	if err != nil {
		t.Fatal(err)
	}
	in := comms.Input{Transactions: Held(txs), Lookup: lookup, Names: map[string]string{}, Zone: time.UTC, Mirror: state.MirrorSnapshot{Season: 2026}}
	for _, f := range cfg.Franchises {
		in.Names[f.ID] = f.Name
	}
	for i, export := range []string{"messageBoard", "polls"} {
		posts, err := ingestion.ParseLeagueBoard(Fixture(t, export+"-2026.json"), export)
		if err != nil {
			t.Fatal(err)
		}
		in.Boards[i] = Held(posts)
	}
	in.Mirror.Players = commsRoster(t)
	return in
}

func commsRoster(t *testing.T) []state.MirrorPlayer {
	t.Helper()
	var raw struct {
		Rosters struct {
			Franchises []struct {
				ID      string `json:"id"`
				Players []struct {
					ID     string `json:"id"`
					Salary string `json:"salary"`
					Year   string `json:"contractYear"`
					Status string `json:"status"`
				} `json:"player"`
			} `json:"franchise"`
		} `json:"rosters"`
	}
	if err := json.Unmarshal(Fixture(t, "rosters-2026.json"), &raw); err != nil {
		t.Fatal(err)
	}
	out := []state.MirrorPlayer{}
	for _, f := range raw.Rosters.Franchises {
		for _, p := range f.Players {
			rr := rosters.RawRoster{FranchiseID: f.ID, PlayerID: p.ID, Salary: p.Salary, ContractYear: p.Year, Status: p.Status}
			if err := rr.Validate(); err != nil {
				t.Fatal(err)
			}
			rec, err := normalize.Contract(rr)
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, state.MirrorPlayer{MFLID: rec.MFLID.String(), FranchiseID: f.ID, Salary: rec.Salary, ContractYear: rec.ContractYear})
		}
	}
	return out
}

func dataRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "ingestion")
}

func commsData(t *testing.T, feed, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dataRoot(), feed, "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func Held[T any](rows []T) report.Held[T] {
	return report.Held[T]{Value: rows, Feed: report.Feed{FetchedAt: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}}
}

// RepoPath keeps wire contracts and frontend samples at their canonical repository paths.
func RepoPath(parts ...string) string {
	return filepath.Join(append([]string{dataRoot(), "..", ".."}, parts...)...)
}
