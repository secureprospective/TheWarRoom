package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
)

// A league's value revision moves with the reference file and its own file only: another league's
// commit leaves its prepared pass in force, while discovery ("") still sees every league.
func TestValueRevisionIsTheLeaguesOwn(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	files, err := openSignInFiles(ctx, newTwoLeagueServer(t).path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	v := &serverValues{pins: &leaguePins{files: files.leagues}, pools: files.leagues.reference}
	read := func(league string) int64 {
		t.Helper()
		r, err := v.revision(ctx, league)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	commit := func(pools *db.Pools, league string, sequence int) {
		t.Helper()
		if _, err := pools.Write().ExecContext(ctx, `INSERT INTO league_recovery_log(sequence,league,revision,recorded_at,message)
			VALUES(?,?,1,'2026-11-16T00:00:00Z','commit')`, sequence, league); err != nil {
			t.Fatal(err)
		}
	}
	other, _ := files.leagues.pools(secondLeague)
	own, _ := files.leagues.pools(fixtureLeague)

	mine, theirs, all := read(fixtureLeague), read(secondLeague), read("")
	commit(other, secondLeague, 2001)
	if read(fixtureLeague) != mine {
		t.Fatal("another league's commit moved this league's value revision")
	}
	if read(secondLeague) == theirs || read("") == all {
		t.Fatal("a league's commit did not move its own revision or discovery's")
	}
	commit(own, fixtureLeague, 2002)
	if read(fixtureLeague) == mine {
		t.Fatal("the league's own commit did not move its value revision")
	}
}

// Until the first input load writes player records there is nothing to prepare: the producer waits
// (it used to retry and log a refusal every second) and reads say values are being prepared.
func TestValuesWaitForPlayerRecords(t *testing.T) {
	t.Parallel()
	files, err := openSignInFiles(t.Context(), newTwoLeagueServer(t).path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	v := newServerValues(newLeaguePins(files.leagues), t.TempDir())
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	if v.waitForRecords(ctx) {
		t.Fatal("the producer started before any player record existed")
	}
	out, err := v.NativeValues(t.Context(), fixtureLeague, "", 2026)
	if err != nil || !out.Preparing || out.League != fixtureLeague {
		t.Fatalf("a read before any player record: %+v %v", out, err)
	}
	v.markRecorded()
	if !v.waitForRecords(t.Context()) {
		t.Fatal("the producer did not start once the input load wrote records")
	}
}

func TestSignalTransportArchivesBodies(t *testing.T) {
	files, err := openSignInFiles(t.Context(), newTwoLeagueServer(t).path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	v := newServerValues(newLeaguePins(files.leagues), t.TempDir())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("weekly statistics")) }))
	defer ts.Close()
	k := &signalsKeeper{values: v, client: ts.Client()}
	client := &http.Client{Transport: k.signalTransport()}
	resp, err := client.Get(ts.URL + "/weekly.csv")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil || string(body) != "weekly statistics" {
		t.Fatalf("read: %q %v", body, err)
	}
	found, err := v.reference.ExactArchivedBodies(t.Context(), ts.URL+"/weekly.csv", func(_ string, b []byte, _ time.Time) bool { return string(b) == "weekly statistics" })
	if err != nil || !found {
		t.Fatalf("archive: found=%v err=%v", found, err)
	}
}
