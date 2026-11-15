package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// An MFL import holds only MFL ids, so live ESPN/Sleeper lines matched no league player (CT 257,
// 11-15: "0 players … 0 ESPN ids held" with a game live). Linking the crosswalk fills the directory
// the live collector reads.
func TestLiveIdentitiesLinkTheCrosswalkTheCollectorReads(t *testing.T) {
	const csv = "name,mfl_id,gsis_id,espn_id,sleeper_id\n" +
		"Jared Goff,12345,00-0033106,3046779,3163\n" +
		"Amon-Ra St. Brown,15281,00-0036963,4374302,7547\n" +
		// One ESPN id under two MFL ids links to neither.
		"Shared One,16001,00-0099901,9999999,9001\n" +
		"Shared Two,16002,00-0099902,9999999,9002\n"
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(csv))
	}))
	defer source.Close()

	files := openTestServer(t, t.TempDir())
	store := files.leagues.family.Over(files.reference)
	collector := &liveCollector{identities: store, packets: &livePackets{held: rawLiveCheckpoint{IDs: map[string]map[string]string{}}}}
	before, err := collector.SourceIDs(t.Context(), "espn")
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != 0 {
		t.Fatalf("fresh reference already holds ESPN ids: %v", before)
	}

	n, err := linkLiveIdentities(t.Context(), source.Client(), source.URL, store)
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no links written")
	}
	want := map[string]map[string]string{
		"espn":    {"12345": "3046779", "15281": "4374302"},
		"sleeper": {"12345": "3163", "15281": "7547", "16001": "9001", "16002": "9002"},
		"gsis":    {"12345": "00-0033106", "15281": "00-0036963", "16001": "00-0099901", "16002": "00-0099902"},
	}
	for kind, ids := range want {
		got, err := collector.SourceIDs(t.Context(), kind)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(ids) {
			t.Fatalf("%s ids %v, want %v", kind, got, ids)
		}
		for player, id := range ids {
			if got[player] != id {
				t.Fatalf("%s id for %s = %q, want %q (all %v)", kind, player, got[player], id, got)
			}
		}
	}

	// A second run (the daily refresh) changes nothing.
	if _, err := linkLiveIdentities(t.Context(), source.Client(), source.URL, store); err != nil {
		t.Fatal(err)
	}
	again, err := collector.SourceIDs(t.Context(), "espn")
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 2 {
		t.Fatalf("refresh changed ESPN ids: %v", again)
	}
}
