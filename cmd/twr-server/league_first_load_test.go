package main

import (
	"context"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/players"
	"github.com/secureprospective/TheWarRoom/internal/leagueload"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration"
	"github.com/secureprospective/TheWarRoom/internal/leaguemigration/migrationtest"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
)

// heldLegacy is a small held league for the loader: the Legacy NFL export plus a matching player list.
func heldLegacy(t *testing.T) heldLeague {
	t.Helper()
	export, err := migrationtest.Legacy()
	if err != nil {
		t.Fatal(err)
	}
	list := make([]players.RawPlayer, 0, len(export.Players))
	for id := range export.Players {
		list = append(list, players.RawPlayer{ID: id, Name: "Player " + id, Position: "WR", Team: "DAL", DraftYear: "2020"})
	}
	return heldLeague{MFLLeague: "14432",
		Source: func(leaguerules.RuleSet) (leaguemigration.Source, error) {
			return &migrationtest.Source{Export: export}, nil
		},
		Football: func(_ context.Context, season int) ([]byte, string, error) {
			at := time.Date(season, 9, 1, 0, 0, 0, 0, time.UTC)
			weeks := []leagueweek.Week{{Number: 1, Games: []leagueweek.Game{{Teams: [2]string{"DAL", "NYG"}, Kickoff: at.Add(240 * time.Hour)}}}}
			return leagueload.BuildFootball(list, weeks, season, at)
		}}
}

// A brand-new server loads its first league without the service ever starting: the empty state
// directory gets an empty server database, the load lands, and a second load never replaces it.
func TestFirstLoadPreparesAnEmptyStateDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.db")
	if err := prepareFirstLoad(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 || info.Mode().Perm() != 0o600 {
		t.Fatalf("prepared database = %v, %v", info, err)
	}
	if err := checkHostedDatabase(path, 0); err != nil {
		t.Fatal(err)
	}
	raw := accounts.RawCredentials{Username: "christopher", Password: "a-long-private-password"}
	var out strings.Builder
	if err := loadHeld(t.Context(), path, heldLegacy(t), "", raw, &out); err != nil {
		t.Fatalf("first load: %v", err)
	}
	if !strings.Contains(out.String(), "with 32 franchises") {
		t.Fatalf("report = %q", out.String())
	}
	// The database now exists: preparing again changes nothing and loading again is refused.
	if err := prepareFirstLoad(path); err != nil {
		t.Fatal(err)
	}
	if err := loadHeld(t.Context(), path, heldLegacy(t), "", raw, io.Discard); err == nil || !strings.Contains(err.Error(), "already on this server") {
		t.Fatalf("second load = %v, want the existing league kept", err)
	}
}

func TestStateDirectoryOwnerUsesDirectoryMetadata(t *testing.T) {
	dir := t.TempDir()
	owner, err := stateDirectoryOwner(dir)
	if err != nil {
		t.Fatal(err)
	}
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	uid, err := strconv.Atoi(current.Uid)
	if err != nil {
		t.Fatal(err)
	}
	gid, err := strconv.Atoi(current.Gid)
	if err != nil {
		t.Fatal(err)
	}
	if owner != (stateFileOwner{uid: uid, gid: gid}) {
		t.Fatalf("owner = %+v, want current directory owner %d:%d", owner, uid, gid)
	}
}

func TestFirstLoadStatePathsUseStateDirectoryOwner(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("changing a temporary state directory owner requires root")
	}
	dir := t.TempDir()
	const uid, gid = 12345, 12346
	if err := os.Chown(dir, uid, gid); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "server.db")
	raw := accounts.RawCredentials{Username: "christopher", Password: "a-long-private-password"}
	var out strings.Builder
	err := withStateDirectoryOwnership(dir, func() error {
		if err := prepareFirstLoad(path); err != nil {
			return err
		}
		if err := checkHostedDatabase(path, 0); err != nil {
			return err
		}
		return loadHeld(t.Context(), path, heldLegacy(t), "", raw, &out)
	})
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if err := filepath.WalkDir(dir, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		actual, err := stateOwnerFromFileInfo(info)
		if err != nil {
			return err
		}
		if actual != (stateFileOwner{uid: uid, gid: gid}) {
			t.Errorf("created path %s owner = %+v, want %d:%d", path, actual, uid, gid)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// An absent database beside other files is never created: the directory is not a fresh install.
func TestFirstLoadRefusesANonEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "leftover"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "server.db")
	if err := prepareFirstLoad(path); err == nil {
		t.Fatal("created a server database beside other files")
	}
	if _, err := os.Lstat(path); err == nil {
		t.Fatal("server database was created")
	}
	if err := checkHostedDatabase(path, 0); err == nil {
		t.Fatal("hosted check accepted a missing database")
	}
}
