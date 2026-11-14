package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/measures"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
	"github.com/secureprospective/TheWarRoom/tools/hostedfixture"
)

// twoFileHost is a hosted server after the move, with a league whose commissioner is seated.
type twoFileHost struct {
	files  stateFiles
	config string
	join   *accounts.JoinService
	code   string
}

func newTwoFileHost(t *testing.T) *twoFileHost {
	t.Helper()
	ctx := t.Context()
	dir, config := newHostFiles(t, bytes.Repeat([]byte{7}, 32))
	if err := os.Remove(filepath.Join(dir, "server.db")); err != nil {
		t.Fatal(err)
	}
	h := &twoFileHost{files: openTestServer(t, dir), config: config}
	registry, err := measures.Embedded()
	if err != nil {
		t.Fatal(err)
	}
	ledger := history.New(h.files.league, registry)
	if err := ledger.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	store := signInStore(h.files)
	if err := store.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	principal, err := hostedfixture.Bootstrap(ctx, store, accounts.RawCredentials{Username: "commissioner", Password: "private-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hostedfixture.CreateLeague(ctx, ledger, principal); err != nil {
		t.Fatal(err)
	}
	h.join = accounts.NewJoin(store, leagueservice.JoinSeats{Store: ledger}, nil)
	code, err := h.join.Rotate(ctx, "native.hosted", principal)
	if err != nil {
		t.Fatal(err)
	}
	h.code = code.Code
	return h
}

func (h *twoFileHost) seatOwner(t *testing.T) {
	t.Helper()
	if _, err := h.join.Claim(t.Context(), accounts.RawJoinClaim{RawJoin: accounts.RawJoin{League: "native.hosted", Code: h.code},
		Franchise: "franchise.01", Username: "owner", Password: "owner-private-password"}); err != nil {
		t.Fatal(err)
	}
}

func (h *twoFileHost) capture(t *testing.T, sets *memorySets, recipient string, kind offsite.SetKind) {
	t.Helper()
	pools, checkpoint := h.files.league, leagueSetCheckpoint
	if kind == offsite.SignInSet {
		pools, checkpoint = h.files.signIns, signInSetCheckpoint
	}
	captureKind(t, sets, recipient, kind, pools, h.config, checkpoint)
}

func captureKind(t *testing.T, sets *memorySets, recipient string, kind offsite.SetKind, pools *db.Pools, config string,
	checkpoint func(context.Context, string) ([]byte, error)) {
	t.Helper()
	sealed, at, err := offsite.CaptureSet(t.Context(), kind, pools, config, t.TempDir(), recipient, checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(sealed)
	if err != nil {
		t.Fatal(err)
	}
	sets.objects[kind.Prefix+at.Format("20060102T150405.000000000Z")+".tar.gz.age"] = body
	time.Sleep(time.Millisecond) // distinct object names
}

func TestRestorePairsLeagueWithItsSignIns(t *testing.T) {
	for _, tc := range []struct {
		name string
		// order of captures; "owner" seats the owner between them.
		steps   []string
		restore bool
	}{
		{"league first, sign-ins last", []string{"owner", "league", "signins"}, true},
		{"only an older sign-in set lacks the owner", []string{"signins", "owner", "league"}, false},
		{"a newer matching set wins over an older mismatch", []string{"signins", "owner", "league", "signins"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			identity, err := age.GenerateX25519Identity()
			if err != nil {
				t.Fatal(err)
			}
			h := newTwoFileHost(t)
			sets := &memorySets{objects: map[string][]byte{}}
			for _, step := range tc.steps {
				switch step {
				case "owner":
					h.seatOwner(t)
				case "league":
					h.capture(t, sets, identity.Recipient().String(), offsite.LeagueSet)
				case "signins":
					h.capture(t, sets, identity.Recipient().String(), offsite.SignInSet)
				}
			}
			target, _ := newHostFiles(t, bytes.Repeat([]byte{7}, 32))
			stage := filepath.Join(t.TempDir(), "restore")
			var output bytes.Buffer
			err = restoreSpareHost(t.Context(), sets, identity, restoreSettings(identity), filepath.Join(target, "server.db"), stage, time.Time{}, &output)
			if !tc.restore {
				if err == nil || !strings.Contains(err.Error(), "no sign-in backup matches") || !strings.Contains(output.String(), "Skipping sign-in backup") {
					t.Fatalf("mismatched pair accepted or not reported: %v\n%s", err, output.String())
				}
				return
			}
			if err != nil {
				t.Fatalf("%v\n%s", err, output.String())
			}
			for _, name := range []string{"server.db", signInFileName} {
				if !fileExists(filepath.Join(stage, "ready", name)) {
					t.Fatalf("ready lacks %s", name)
				}
			}
			if !strings.Contains(output.String(), "Signed out all 2 accounts") {
				t.Fatalf("report: %s", output.String())
			}
			// The staged pair opens as one server, with no sign-in tables in the league file.
			files, err := openSignInFiles(t.Context(), filepath.Join(stage, "ready", "server.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = files.Close() }()
			if held, err := hasTable(t.Context(), files.league, "accounts"); err != nil || held {
				t.Fatalf("restored league file holds accounts: %v", err)
			}
		})
	}
}

// A league committed in its league file with no directory row is picked up on the next start.
func TestStartRecordsLeaguesMissingFromDirectory(t *testing.T) {
	h := newTwoFileHost(t)
	store := signInStore(h.files)
	if before, err := store.Directory(t.Context()); err != nil || len(before) != 0 {
		t.Fatalf("directory before start: %v %v", before, err)
	}
	if _, err := hostedServices(t.Context(), h.files, make([]byte, 32), ""); err != nil {
		t.Fatal(err)
	}
	after, err := store.Directory(t.Context())
	if err != nil || len(after) != 1 || after[0].League != "native.hosted" || after[0].State == "" || after[0].Created == "" {
		t.Fatalf("directory after start: %+v %v", after, err)
	}
}
