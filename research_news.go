package main

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

// archivedNews reads the archived playerProfile bodies once, newest first, and keeps each wanted player's newest
// answer. A body is matched by the exact members of its P list, never a substring. The pass ends when every wanted
// player is answered or when more says the remaining bodies are too old, so the cost is one scan per refresh, not
// one per player.
// archivedBodies is the history store's ArchivedBodies, as a seam for tests.
type archivedBodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)

func archivedNews(ctx context.Context, bodies archivedBodies, season int, want []string,
	more func(time.Time) bool) (map[string]heldSeasonFeed[[]researchfeeds.News], error) {
	out := map[string]heldSeasonFeed[[]researchfeeds.News]{}
	open := map[string]bool{}
	for _, id := range want {
		open[id] = true
	}
	if len(open) == 0 {
		return out, nil
	}
	_, err := bodies(ctx, "TYPE=playerProfile", func(src string, body []byte, at time.Time) bool {
		if !more(at) {
			return true
		}
		u, err := url.Parse(src)
		if err != nil || !sameExport(src, "playerProfile", season, true) {
			return false
		}
		asked := strings.Split(u.Query().Get("P"), ",")
		if !slices.ContainsFunc(asked, func(id string) bool { return open[id] }) {
			return false
		}
		rows, err := researchfeeds.ParseNews(body, asked)
		if err != nil {
			return false
		}
		for _, id := range asked {
			if news := rows[id]; open[id] {
				out[id] = heldSeasonFeed[[]researchfeeds.News]{Value: news, FetchedAt: at}
				delete(open, id)
			}
		}
		return len(open) == 0
	})
	if err != nil {
		return out, fmt.Errorf("news archive: %w", err)
	}
	return out, nil
}

// readResearchNewsBatches serves today's reads from memory or the archive, then asks MFL for the rest in batches
// of 100 with one pause per request. Omitted profiles are held as no news for the day.
func readResearchNewsBatches(ctx context.Context, held map[string]heldSeasonFeed[[]researchfeeds.News], season int, ids []string, now time.Time,
	bodies archivedBodies, pause func(context.Context, time.Duration) error,
	fetch func(context.Context, []string) (map[string][]researchfeeds.News, error), logLine func(string)) error {
	day := now.In(time.Local).Format("2006-01-02")
	today := func(at time.Time) bool { return at.In(time.Local).Format("2006-01-02") == day }
	need := []string{}
	for _, id := range ids {
		if h := held[id]; h.FetchedAt.IsZero() || !today(h.FetchedAt) {
			need = append(need, id)
		}
	}
	cached, err := archivedNews(ctx, bodies, season, need, today)
	if err != nil {
		return fmt.Errorf("news daily cache: %w", err)
	}
	pending := []string{}
	for _, id := range need {
		if h, ok := cached[id]; ok {
			held[id] = h
		} else {
			pending = append(pending, id)
		}
	}
	omitted := []string{}
	for batch := range slices.Chunk(pending, 100) {
		if err := pause(ctx, time.Second); err != nil {
			return err
		}
		rows, failures := isolatedNews(ctx, batch, pause, fetch)
		losses := map[string]error{}
		for _, id := range batch {
			news, answered := rows[id]
			switch {
			case failures[id] != nil:
				losses[id] = failures[id]
			default:
				if !answered {
					omitted = append(omitted, id)
					news = []researchfeeds.News{}
				}
				next := held[id]
				holdSeasonFeed(&next, news, liveFreshness(now), nil)
				held[id] = next
			}
		}
		fallbackResearchNews(ctx, held, season, batch, losses, bodies)
		if ctx.Err() != nil {
			return fmt.Errorf("news refresh: %w", ctx.Err())
		}
	}
	if len(omitted) > 0 {
		logLine("MFL has no profile for: " + strings.Join(omitted, ", "))
	}
	return nil
}

// fallbackResearchNews holds each lost player's newest archived news, marked stale with the loss, in one archive
// pass for the whole batch. A player with no archived news keeps the loss and nothing invented.
func fallbackResearchNews(ctx context.Context, held map[string]heldSeasonFeed[[]researchfeeds.News], season int,
	batch []string, losses map[string]error, bodies archivedBodies) {
	if len(losses) == 0 {
		return
	}
	lost := slices.DeleteFunc(slices.Clone(batch), func(id string) bool { return losses[id] == nil })
	rctx, cancel := context.WithTimeout(ctx, fallbackTimeout)
	defer cancel()
	found, err := archivedNews(rctx, bodies, season, lost, func(time.Time) bool { return true })
	for _, id := range lost {
		next, loss := held[id], losses[id]
		switch h, ok := found[id]; {
		case ok:
			holdSeasonFeed(&next, h.Value, staleFreshness(h.FetchedAt, loss), nil)
		case err != nil:
			holdSeasonFeed(&next, nil, Freshness{}, fmt.Errorf("news player %s archive: %w (live: %v)", id, err, loss))
		default:
			holdSeasonFeed(&next, nil, Freshness{}, loss)
		}
		held[id] = next
	}
}

// Only an invalid-ID rejection is split; outages and rate limits must not multiply requests.
func isolatedNews(ctx context.Context, ids []string, pause func(context.Context, time.Duration) error,
	fetch func(context.Context, []string) (map[string][]researchfeeds.News, error)) (map[string][]researchfeeds.News, map[string]error) {
	rows, err := fetch(ctx, ids)
	losses := map[string]error{}
	if err == nil {
		return rows, losses
	}
	if len(ids) == 1 && researchfeeds.InvalidPlayerIDs(err) {
		// MFL holds no profile for this one player: that is its answer (no news), not a lost read.
		return map[string][]researchfeeds.News{}, losses
	}
	if len(ids) == 1 || !researchfeeds.InvalidPlayerIDs(err) || ctx.Err() != nil {
		for _, id := range ids {
			losses[id] = err
		}
		return rows, losses
	}
	out := map[string][]researchfeeds.News{}
	for _, half := range [][]string{ids[:len(ids)/2], ids[len(ids)/2:]} {
		if err := pause(ctx, time.Second); err != nil {
			for _, id := range half {
				losses[id] = err
			}
			continue
		}
		good, lost := isolatedNews(ctx, half, pause, fetch)
		for id, news := range good {
			out[id] = news
		}
		for id, loss := range lost {
			losses[id] = loss
		}
	}
	return out, losses
}
