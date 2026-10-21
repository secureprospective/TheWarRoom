package research

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/researchfeeds"
)

type fakeHost struct {
	in           Snapshot
	calls        []string
	names        map[string]string
	readyErr     error
	host, opened string
	refresh      func(bool) error
}

func (h *fakeHost) Ready() error { h.calls = append(h.calls, "ready"); return h.readyErr }
func (h *fakeHost) ResearchNames() map[string]string {
	h.calls = append(h.calls, "names")
	return h.names
}
func (h *fakeHost) HydrateResearch(ids ...string) {
	h.calls = append(h.calls, "hydrate:"+strings.Join(ids, ","))
}
func (h *fakeHost) PrepareResearchDepth()            { h.calls = append(h.calls, "depth") }
func (h *fakeHost) ResearchSnapshot() Snapshot       { h.calls = append(h.calls, "snapshot"); return h.in }
func (h *fakeHost) RefreshResearch(force bool) error { return h.refresh(force) }
func (h *fakeHost) RefreshResearchNews(mine string) error {
	h.calls = append(h.calls, "news-refresh:"+mine)
	return nil
}
func (h *fakeHost) ResearchNews(id string) Feed[[]researchfeeds.News] {
	h.calls = append(h.calls, "news:"+id)
	return h.in.Held.News[id]
}
func (h *fakeHost) ResearchDestination() (string, int) {
	h.calls = append(h.calls, "destination")
	return h.host, 2026
}
func (h *fakeHost) OpenBrowser(target string) { h.opened = target; h.calls = append(h.calls, "open") }

func TestAPIValidationAndReadOrder(t *testing.T) {
	h := &fakeHost{in: realResearch(t), names: map[string]string{"0001": "Mine"}}
	a := New(h, &Held{})
	if _, err := a.TargetResearch(RawResearchScope{Mine: "bad"}); err == nil || len(h.calls) != 0 {
		t.Fatal(err, h.calls)
	}
	if _, err := a.TargetResearchPlayer("bad"); err == nil || len(h.calls) != 0 {
		t.Fatal(err, h.calls)
	}
	if _, err := a.TargetResearch(RawResearchScope{Mine: "9999"}); err == nil || !reflect.DeepEqual(h.calls, []string{"ready", "names"}) {
		t.Fatal(err, h.calls)
	}
	h.calls = nil
	if _, err := a.TargetResearch(RawResearchScope{Mine: "0001"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.calls, []string{"ready", "names", "hydrate:", "snapshot"}) {
		t.Fatal(h.calls)
	}
	h.calls = nil
	if _, err := a.TargetResearchPlayer("16162"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(h.calls, []string{"ready", "hydrate:16162", "depth", "snapshot", "names"}) {
		t.Fatal(h.calls)
	}
	h.calls, h.readyErr = nil, errors.New("fixture startup failed")
	if _, err := a.TargetResearch(RawResearchScope{}); err == nil || err.Error() != "research: fixture startup failed" || !reflect.DeepEqual(h.calls, []string{"ready"}) {
		t.Fatal(err, h.calls)
	}
}

func TestAPIArticleAndRefreshPorts(t *testing.T) {
	h := &fakeHost{in: realResearch(t), host: "fixture.invalid"}
	held := &Held{}
	a := New(h, held)
	if err := a.TargetOpenResearchArticle("16162", "forged"); err == nil || !reflect.DeepEqual(h.calls, []string{"ready", "news:16162"}) {
		t.Fatal(err, h.calls)
	}
	h.calls = nil
	if err := a.TargetOpenResearchArticle("16162", "rotowire640841"); err != nil {
		t.Fatal(err)
	}
	if h.opened != "https://fixture.invalid/2026/view_news_article?ID=rotowire640841" || !reflect.DeepEqual(h.calls, []string{"ready", "news:16162", "destination", "open"}) {
		t.Fatal(h.opened, h.calls)
	}
	h.refresh = func(force bool) error {
		if !force {
			t.Error("force lost")
		}
		return errors.New("fixture refresh")
	}
	if err := a.TargetResearchRefresh(true); err == nil || err.Error() != "fixture refresh" {
		t.Fatal(err)
	}
	if err := a.TargetResearchNewsRefresh("0001"); err != nil {
		t.Fatal(err)
	}
	held.PublishStatus("Reading fantasy points")
	if a.TargetResearchStatus() != "Reading fantasy points" {
		t.Fatal(a.TargetResearchStatus())
	}
}

func TestAPIStatusReadCompletesDuringBlockedRefresh(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	h := &fakeHost{refresh: func(bool) error { close(entered); <-release; return nil }}
	held := &Held{}
	held.PublishStatus("waiting")
	a := New(h, held)
	done := make(chan error, 1)
	go func() { done <- a.TargetResearchRefresh(false) }()
	<-entered
	read := make(chan string, 1)
	go func() { read <- a.TargetResearchStatus() }()
	select {
	case got := <-read:
		if got != "waiting" {
			t.Error(got)
		}
	case <-time.After(5 * time.Second):
		t.Error("status blocked behind refresh")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
