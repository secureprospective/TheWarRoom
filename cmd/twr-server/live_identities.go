package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/sources"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Live scoring keys every ESPN, Sleeper and nflverse stat line by league player through the shared
// reference file's player directory (player_ids). An MFL import brings only MFL ids, so the server
// links the other ids itself from the same DynastyProcess crosswalk the desktop loads: at start, then
// once a day. Until the first link lands, live lines have no league player and stay pending.
const liveIdentityRefresh = 24 * time.Hour

// liveIdentityLinker is the part of history.Store the linker writes through.
type liveIdentityLinker interface {
	LinkPlayerIDs(context.Context, string, []history.PlayerIDLink) (int, error)
}

// linkLiveIdentities fetches the crosswalk from url and links every id it carries onto its MFL id.
// The server holds no MFL player names, so no name guard applies: every id links on DynastyProcess's
// word, as Links does for an id MFL does not list. An id DynastyProcess ties to two MFL ids still
// links to neither.
func linkLiveIdentities(ctx context.Context, client *http.Client, url string, store liveIdentityLinker) (int, error) {
	cw, err := crosswalk.Fetch(ctx, client, url)
	if err != nil {
		return 0, fmt.Errorf("live identities: %w", err)
	}
	links, _ := cw.Links(func(string) (string, bool) { return "", false })
	rows := make([]history.PlayerIDLink, len(links))
	for i, l := range links {
		rows[i] = history.PlayerIDLink{IDType: l.IDType, IDValue: l.IDValue, PlayerID: l.MFLID}
	}
	if _, err := store.LinkPlayerIDs(ctx, sources.DynastyProcessSource, rows); err != nil {
		return 0, fmt.Errorf("live identities: %w", err)
	}
	return len(rows), nil
}

// keepLiveIdentities links at once and then every liveIdentityRefresh until ctx ends. A failed fetch
// keeps the ids already held and tries again at the next refresh.
func keepLiveIdentities(ctx context.Context, store liveIdentityLinker) {
	client := &http.Client{Timeout: time.Minute}
	for {
		if n, err := linkLiveIdentities(ctx, client, crosswalk.SourceURL, store); err != nil {
			log.Printf("server live: %v", err)
		} else {
			log.Printf("server live: player directory linked %d outside ids", n)
		}
		timer := time.NewTimer(liveIdentityRefresh)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
