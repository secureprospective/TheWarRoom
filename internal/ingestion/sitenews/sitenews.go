// Package sitenews reads MFL's site news (What's New): MFL-wide articles such as official stat
// changes and strange plays, served as RSS from the api host.
package sitenews

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// Export is MFL's export name.
const Export = "siteNews"

type text struct {
	T string `json:"$t"`
}

// RawItem is one article as MFL sends it: HTML text and an RFC 1123 date.
type RawItem struct {
	Title       text `json:"title"`
	Description text `json:"description"`
	Link        text `json:"link"`
	PubDate     text `json:"pubDate"`
}

// Item is an article in plain words.
type Item struct {
	Headline  string
	Published time.Time
	Lines     []string // the article's text: one sentence or list entry per line
	Link      string
}

// Fetch reads the site news from the api host.
func Fetch(ctx context.Context, c *mfl.Client, year string) ([]RawItem, error) {
	resp, err := c.Do(ctx, mfl.Request{Type: Export, Year: year})
	if err != nil {
		return nil, fmt.Errorf("sitenews: fetch: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitenews: unexpected status %d", resp.StatusCode)
	}
	return Parse(resp.Body)
}

// Parse reads MFL's RSS envelope.
func Parse(body []byte) ([]RawItem, error) {
	if err := ingestion.CheckAPIError(body); err != nil {
		return nil, fmt.Errorf("sitenews: %w", err)
	}
	var env struct {
		RSS struct {
			Channel struct {
				Items ingestion.MFLList[RawItem] `json:"item"`
			} `json:"channel"`
		} `json:"rss"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("sitenews: decode: %w", err)
	}
	return env.RSS.Channel.Items, nil
}

// Read turns raw items into plain words, newest first. An item without a headline or a readable
// date is an error: MFL always sends both.
func Read(raw []RawItem) ([]Item, error) {
	out := make([]Item, 0, len(raw))
	for _, r := range raw {
		headline := strings.TrimSpace(html.UnescapeString(r.Title.T))
		at, err := time.Parse(time.RFC1123, strings.TrimSpace(r.PubDate.T))
		if headline == "" || err != nil {
			return nil, fmt.Errorf("sitenews: item %q has no headline or date", headline)
		}
		out = append(out, Item{Headline: headline, Published: at, Lines: PlainLines(r.Description.T), Link: r.Link.T})
	}
	slicesSortNewest(out)
	return out, nil
}

var (
	// A tag runs to the first > outside quotes: MFL's links carry "<league_id>" inside their href.
	breakTag = regexp.MustCompile(`(?i)</?(p|br|ul|ol|li|div|h[1-6]|tr)\b(?:"[^"]*"|'[^']*'|[^'">])*>`)
	anyTag   = regexp.MustCompile(`<(?:"[^"]*"|'[^']*'|[^'">])*>`)
	spaces   = regexp.MustCompile(`[ \t\r\f\v]+`)
)

// PlainLines reads MFL's article HTML as plain lines: block tags and list items break lines, every
// other tag is dropped, entities are decoded and spacing is collapsed.
func PlainLines(s string) []string {
	s = breakTag.ReplaceAllString(s, "\n")
	s = html.UnescapeString(anyTag.ReplaceAllString(s, ""))
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(spaces.ReplaceAllString(line, " ")); line != "" {
			out = append(out, line)
		}
	}
	return out
}

func slicesSortNewest(items []Item) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Published.After(items[j-1].Published); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
