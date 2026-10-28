package commissioner

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

const HistoryLink = "history.link"

// RawHistoryLink is the history.link argument (Args.Link): which franchise a historical record belongs to.
type RawHistoryLink struct {
	Record    string `json:"record"`
	Franchise string `json:"franchise"`
}

// championRecord names the record of the season's champion; seasonRecord, a season's own row.
func championRecord(season int) string { return "champion." + strconv.Itoa(season) }

// historyRecordYear is the season a champion or season record names; ok is false for any other record.
func historyRecordYear(record string) (year int, ok bool) {
	kind, rest, found := strings.Cut(record, ".")
	if !found || (kind != "champion" && kind != "season") {
		return 0, false
	}
	year, err := strconv.Atoi(rest)
	return year, err == nil && year >= 1920 && year <= 9999
}

// DecodeHistoryLink reads a link command's argument. A record is a champion or season row of a year, or an
// archive row by name.
func DecodeHistoryLink(a RawArgs) (RawHistoryLink, error) {
	other := a
	other.Link = nil
	if a.Link == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return RawHistoryLink{}, fmt.Errorf("send only the details of this history link")
	}
	raw := *a.Link
	_, dated := historyRecordYear(raw.Record)
	archive := strings.HasPrefix(raw.Record, "archive.") && len(raw.Record) > len("archive.") && len(raw.Record) <= 80 &&
		strings.Trim(raw.Record, "abcdefghijklmnopqrstuvwxyz0123456789._-") == ""
	if (!dated && !archive) || !strings.HasPrefix(raw.Franchise, "franchise.") {
		return raw, fmt.Errorf("choose a champion or season record, such as champion.2019, and an app franchise")
	}
	return raw, nil
}

// HistoryLinkRow says which franchise a record belongs to and where that came from.
type HistoryLinkRow struct {
	Record         string `json:"record"`
	Franchise      string `json:"franchise"`
	ByCommissioner bool   `json:"byCommissioner"`
	Entry          string `json:"entry,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// HistoryLinks lists every record's franchise. The champions archived with the league are the starting
// links; a link the commissioner set replaces one, and reversing it brings the starting link back.
func HistoryLinks(league string, records []Record) (map[string]HistoryLinkRow, error) {
	assets, err := NativeAssetsReading(league, records)
	if err != nil {
		return nil, err
	}
	out := map[string]HistoryLinkRow{}
	if assets.Snapshot != nil {
		for _, c := range assets.Snapshot.Champions {
			out[championRecord(c.Season)] = HistoryLinkRow{Record: championRecord(c.Season), Franchise: c.Franchise}
		}
	}
	active := Replay(records).Active
	for _, r := range records {
		if r.Command.Command != HistoryLink || !active[r.Receipt.Entry] {
			continue
		}
		l := r.Command.Args.Link
		out[l.Record] = HistoryLinkRow{Record: l.Record, Franchise: l.Franchise, ByCommissioner: true,
			Entry: r.Receipt.Entry, Reason: r.Command.Reason}
	}
	return out, nil
}

// CheckHistoryLinkAppend refuses a link to a franchise the league lacks, to a season that has not begun, or
// one that links a record to the franchise it already belongs to.
func CheckHistoryLinkAppend(raw RawCommand, records []Record, now time.Time) error {
	link, err := DecodeHistoryLink(raw.Args)
	if err != nil {
		return err
	}
	if len(records) == 0 || !slices.Contains(LeagueFranchises(records), link.Franchise) {
		return Refused(fmt.Errorf("that franchise is not in this app league"))
	}
	if year, dated := historyRecordYear(link.Record); dated {
		current, err := SeasonAt(records, ReadingInstant(records, now))
		if err != nil {
			return err
		}
		if year > current {
			return Refused(fmt.Errorf("season %d has not been played, so it has no history to link", year))
		}
	}
	held, err := HistoryLinks(raw.League, records)
	if err != nil {
		return err
	}
	if held[link.Record].Franchise == link.Franchise {
		return Refused(fmt.Errorf("that record already belongs to this franchise. Nothing would change"))
	}
	return nil
}

func linkLabel(l RawHistoryLink, name func(string) string) string {
	return fmt.Sprintf("Linked the record %s to %s", l.Record, name(l.Franchise))
}

// FranchiseHistory is everything the league's history says belongs to one franchise, read through the links.
type FranchiseHistory struct {
	Franchise string           `json:"franchise"`
	Titles    []int            `json:"titles"`
	Records   []HistoryLinkRow `json:"records"`
}

// FranchiseHistoryReading lists the records linked to a franchise, and the seasons it won the title.
func FranchiseHistoryReading(league string, records []Record, franchise string) (FranchiseHistory, error) {
	links, err := HistoryLinks(league, records)
	if err != nil {
		return FranchiseHistory{}, err
	}
	out := FranchiseHistory{Franchise: franchise, Titles: []int{}, Records: []HistoryLinkRow{}}
	for _, record := range slices.Sorted(maps.Keys(links)) {
		if links[record].Franchise != franchise {
			continue
		}
		out.Records = append(out.Records, links[record])
		if year, dated := historyRecordYear(record); dated && strings.HasPrefix(record, "champion.") {
			out.Titles = append(out.Titles, year)
		}
	}
	return out, nil
}
