package pulseproof

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestCounterCountsResponsesWithoutChangingBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"unchanged":true}`))
	}))
	defer server.Close()
	c := &Counter{}
	client := &http.Client{Transport: c}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			resp, err := client.Post(server.URL+"/v1/native/workflow", "application/json", strings.NewReader(`{"workflow":{"command":"week.read"}}`))
			if err != nil {
				t.Error(err)
				return
			}
			body, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil || string(body) != `{"unchanged":true}` {
				t.Errorf("response %q: %v", body, err)
			}
		})
	}
	wg.Wait()
	if got := c.Snapshot()["week.read"]; got != (Counts{Calls: 10, Unchanged: 10}) {
		t.Fatalf("counts: %+v", got)
	}
	snapshot := c.Snapshot()
	delete(snapshot, "week.read")
	if len(c.Snapshot()) != 1 {
		t.Fatal("snapshot mutated counter")
	}
}
