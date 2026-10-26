package research

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/store/state"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

func ReadArchivedFeed(
	ctx context.Context, bodies ArchivedBodies, season int, kind, week string,
) Feed[[]researchfeeds.Feed] {
	h := Feed[[]researchfeeds.Feed]{}
	found, err := bodies(ctx, "TYPE="+kind, func(src string, body []byte, at time.Time) bool {
		if !researchExport(src, kind, season) {
			return false
		}
		feed, err := researchfeeds.Parse(body, kind, week)
		if err != nil {
			return false
		}
		h.Value, h.FetchedAt, h.RefreshError = []researchfeeds.Feed{feed}, at, "Archived copy; refresh pending"
		return true
	})
	if err != nil {
		h.RefreshError = "Archive unreadable: " + err.Error()
	}
	if !found && err == nil {
		h.RefreshError = "No archived copy held"
	}
	return h
}

func researchExport(src, kind string, season int) bool {
	if researchfeeds.IsTrend(kind) {
		return ingestion.SameLeaguelessExport(src, kind, season)
	}
	return ingestion.SameExport(src, kind, season)
}

func ReadDepthArchive(ctx context.Context, season int, bodies ArchivedBodies, fetchErr error) ([]researchfeeds.RawDepth, domain.Freshness, error) {
	rctx, cancel := context.WithTimeout(ctx, report.FallbackTimeout)
	defer cancel()
	var fresh domain.Freshness
	var rows []researchfeeds.RawDepth
	found, aerr := bodies(rctx, researchfeeds.DepthURL(season), func(src string, body []byte, at time.Time) bool {
		if src != researchfeeds.DepthURL(season) {
			return false
		}
		parsed, perr := researchfeeds.ParseDepth(body)
		if perr != nil {
			return false
		}
		rows = parsed
		fresh = report.StaleFreshness(at, fetchErr)
		return true
	})
	if aerr != nil {
		return nil, domain.Freshness{}, fmt.Errorf("depth chart archive: %w", aerr)
	}
	if !found {
		return nil, domain.Freshness{}, fmt.Errorf("depth charts: no archived copy: %w", fetchErr)
	}
	return rows, fresh, nil
}

// FirstArchiveRead marks a key before attempting its read, including absent archive copies.
// The caller owns serialization and publication of the map.
func FirstArchiveRead(reads *map[string]bool, key string) bool {
	if (*reads)[key] {
		return false
	}
	if *reads == nil {
		*reads = map[string]bool{}
	}
	(*reads)[key] = true
	return true
}

func HydrateFeed(feeds map[string]Feed[[]researchfeeds.Feed], kind, week string, first func(string) bool, read func(string, string) Feed[[]researchfeeds.Feed]) {
	key := Key(kind, week)
	held := feeds[key]
	if held.FetchedAt.IsZero() && first("feed:"+key) {
		archived := read(kind, week)
		if held.RefreshError != "" {
			archived.RefreshError = held.RefreshError
		}
		feeds[key] = archived
	}
}

// ArchiveFeeds selects the existing scoped store callback for weekly reads.
type ArchiveFeeds struct {
	Bodies ArchivedBodies
	Scoped func(string, string) ArchivedBodies
}

func (s ArchiveFeeds) Read(ctx context.Context, season int, kind, week string) Feed[[]researchfeeds.Feed] {
	bodies := s.Bodies
	if week != "" {
		bodies = s.Scoped("W", week)
	}
	return ReadArchivedFeed(ctx, bodies, season, kind, week)
}

type ArchiveInput struct {
	Feeds    map[string]Feed[[]researchfeeds.Feed]
	Depth    Feed[[]researchfeeds.RawDepth]
	Injuries Feed[[]leaguefeed.Injury]
	News     map[string]Feed[[]researchfeeds.News]
}

// ArchiveSource supplies local reads only. The caller serializes hydration and publication.
type ArchiveSource struct {
	Feeds     ArchiveFeeds
	Season    int
	Week      func() int
	FirstRead func(string) bool
	Roster    func() state.MirrorSnapshot
	Injuries  func() Feed[[]leaguefeed.Injury]
	Log       func(string)
}

func HydrateArchive(ctx context.Context, in ArchiveInput, extraIDs []string, source ArchiveSource) ArchiveInput {
	in.Feeds = maps.Clone(in.Feeds)
	in.News = maps.Clone(in.News)
	if in.Feeds == nil {
		in.Feeds = map[string]Feed[[]researchfeeds.Feed]{}
	}
	read := func(kind, week string) Feed[[]researchfeeds.Feed] {
		return source.Feeds.Read(ctx, source.Season, kind, week)
	}
	for _, kind := range Kinds() {
		week := ""
		if kind == "playerScores" {
			week = "YTD"
		}
		HydrateFeed(in.Feeds, kind, week, source.FirstRead, read)
	}
	for week := 1; week <= source.Week(); week++ {
		HydrateFeed(in.Feeds, "playerScores", strconv.Itoa(week), source.FirstRead, read)
	}
	if in.Depth.FetchedAt.IsZero() && source.FirstRead("depth") {
		HydrateDepth(ctx, source.Season, source.Feeds.Bodies, &in.Depth, source.Log)
	}
	if in.Injuries.FetchedAt.IsZero() && source.FirstRead("injuries") {
		in.Injuries = source.Injuries()
		if !in.Injuries.FetchedAt.IsZero() {
			in.Injuries.RefreshError = "Archived copy; refresh pending"
		}
	}
	mirror := source.Roster()
	missing := []string{}
	for _, id := range append(RosterIDs(mirror, ""), extraIDs...) {
		if in.News[id].FetchedAt.IsZero() && source.FirstRead("news:"+id) {
			missing = append(missing, id)
		}
	}
	cached, err := ArchivedNews(ctx, source.Feeds.Bodies, source.Season, missing, func(time.Time) bool { return true })
	if err != nil {
		source.Log(err.Error())
	}
	in.News = cached
	return in
}

func HydrateDepth(ctx context.Context, season int, bodies ArchivedBodies, depth *Feed[[]researchfeeds.RawDepth], log func(string)) {
	rows, fresh, err := ReadDepthArchive(ctx, season, bodies, fmt.Errorf("archived copy; refresh pending"))
	if err != nil {
		if depth.RefreshError == "" {
			log(err.Error())
			depth.RefreshError = "No archived copy held"
		}
		return
	}
	failed := depth.RefreshError
	report.HoldFeed(depth, rows, fresh, nil)
	if failed != "" {
		depth.RefreshError = failed
	}
}

func ArchiveLinks(links map[string]string, heldVersion, loadedVersion string, read func() (map[string]string, error)) (map[string]string, error) {
	if links == nil || heldVersion != loadedVersion {
		return read()
	}
	return links, nil
}

// MergeArchive fills absent data only, against the latest detached capture under the caller's lock.
func MergeArchive(current Values, injury Feed[[]leaguefeed.Injury], archived ArchiveInput, links map[string]string, version string, linkErr error, now func() time.Time) (Values, Feed[[]leaguefeed.Injury]) {
	if current.Feeds == nil {
		current.Feeds = map[string]Feed[[]researchfeeds.Feed]{}
	}
	for key, h := range archived.Feeds {
		if current.Feeds[key].FetchedAt.IsZero() {
			current.Feeds[key] = h
		}
	}
	if current.Depth.FetchedAt.IsZero() {
		current.Depth = archived.Depth
	}
	if injury.FetchedAt.IsZero() {
		injury = archived.Injuries
	}
	if current.News == nil {
		current.News = map[string]Feed[[]researchfeeds.News]{}
	}
	for id, h := range archived.News {
		if current.News[id].FetchedAt.IsZero() {
			if h.FetchedAt.In(time.Local).Format("2006-01-02") != now().In(time.Local).Format("2006-01-02") {
				h.RefreshError = "Archived copy; refresh pending"
			}
			current.News[id] = h
		}
	}
	if linkErr != nil {
		current.LinkError = linkErr.Error()
	} else {
		current.Links, current.LinkError, current.LinkVersion = links, "", version
	}
	return current, injury
}
