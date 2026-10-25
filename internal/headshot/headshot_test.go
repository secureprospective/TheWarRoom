package headshot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/offline"
)

// tiny builds a small generated image; no real photo ever appears in tests (spec §7).
func tiny(t *testing.T, kind string, shade uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for x := range 48 {
		for y := range 48 {
			img.Set(x, y, color.RGBA{shade, byte((x * y * 7) & 0xff), byte((x*13 ^ y*29) & 0xff), 255})
		}
	}
	var b bytes.Buffer
	var err error
	if kind == "png" {
		err = png.Encode(&b, img)
	} else {
		err = jpeg.Encode(&b, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// hosts answers ESPN and Sleeper requests from maps of id → (status, body), counting requests.
type hosts struct {
	espn, sleeper map[string]reply
	calls         atomic.Int32
}

type reply struct {
	status int
	body   []byte
}

func (h *hosts) RoundTrip(r *http.Request) (*http.Response, error) {
	h.calls.Add(1)
	var rep reply
	var ok bool
	switch r.URL.Host {
	case "a.espncdn.com":
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Query().Get("img"), "/i/headshots/nfl/players/full/"), ".png")
		rep, ok = h.espn[id]
	case "sleepercdn.com":
		rep, ok = h.sleeper[strings.TrimSuffix(filepath.Base(r.URL.Path), ".jpg")]
	}
	if !ok {
		rep = reply{http.StatusNotFound, []byte("<html>not found</html>")}
	}
	return &http.Response{StatusCode: rep.status, Body: ioNopCloser(rep.body), Header: http.Header{}, Request: r}, nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(b []byte) nopCloser { return nopCloser{bytes.NewReader(b)} }

func service(t *testing.T, h *hosts, ids map[string][2]string) *Service {
	t.Helper()
	s := New(t.TempDir(), func(_ context.Context, id string) (string, string, error) {
		return ids[id][0], ids[id][1], nil
	}, &http.Client{Transport: h})
	clock := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { clock = clock.Add(time.Second); return clock }
	return s
}

func TestPhotoTriesESPNThenSleeperThenNone(t *testing.T) {
	espnPNG, sleeperJPG := tiny(t, "png", 10), tiny(t, "jpg", 200)
	h := &hosts{
		espn:    map[string]reply{"100": {200, espnPNG}},
		sleeper: map[string]reply{"7": {200, sleeperJPG}, "8": {200, sleeperJPG}},
	}
	s := service(t, h, map[string][2]string{"0501": {"100", "7"}, "0502": {"999", "8"}, "0503": {"", ""}})
	ctx := context.Background()

	b, kind, err := s.Photo(ctx, "0501")
	if err != nil || kind != "image/png" || !bytes.Equal(b, espnPNG) {
		t.Fatalf("ESPN first: kind %q err %v", kind, err)
	}
	b, kind, err = s.Photo(ctx, "0502")
	if err != nil || kind != "image/jpeg" || !bytes.Equal(b, sleeperJPG) {
		t.Fatalf("Sleeper when ESPN has none: kind %q err %v", kind, err)
	}
	if _, _, err := s.Photo(ctx, "0503"); !errors.Is(err, ErrNoPhoto) {
		t.Fatalf("no ids: err %v, want ErrNoPhoto", err)
	}
}

func TestSilhouetteAndHTMLAreNoPhoto(t *testing.T) {
	silhouette, sleeperJPG := tiny(t, "png", 99), tiny(t, "jpg", 40)
	html := []byte("<!doctype html><html><body>" + strings.Repeat("error ", 100) + "</body></html>")
	h := &hosts{
		espn:    map[string]reply{"1": {200, silhouette}, "2": {200, html}},
		sleeper: map[string]reply{"5": {200, sleeperJPG}},
	}
	s := service(t, h, map[string][2]string{"1001": {"1", "5"}, "1002": {"2", ""}})
	sum := sha256.Sum256(silhouette)
	s.silhouettes[hex.EncodeToString(sum[:])] = true

	b, _, err := s.Photo(context.Background(), "1001")
	if err != nil || !bytes.Equal(b, sleeperJPG) {
		t.Fatalf("a silhouette must fall back to Sleeper: err %v", err)
	}
	if _, _, err := s.Photo(context.Background(), "1002"); !errors.Is(err, ErrNoPhoto) {
		t.Fatalf("an HTML page served as a photo: err %v, want ErrNoPhoto", err)
	}
}

func TestCacheHitNegativeCacheAndHostFailure(t *testing.T) {
	h := &hosts{espn: map[string]reply{"1": {200, tiny(t, "png", 1)}, "3": {503, nil}}}
	s := service(t, h, map[string][2]string{"2001": {"1", ""}, "2002": {"2", ""}, "2003": {"3", ""}})
	ctx := context.Background()

	for range 3 {
		if _, _, err := s.Photo(ctx, "2001"); err != nil {
			t.Fatal(err)
		}
	}
	if got := h.calls.Load(); got != 1 {
		t.Fatalf("a cached photo is fetched once, got %d requests", got)
	}
	if _, _, err := s.Photo(ctx, "2002"); !errors.Is(err, ErrNoPhoto) {
		t.Fatal(err)
	}
	if _, _, err := s.Photo(ctx, "2002"); !errors.Is(err, ErrNoPhoto) || h.calls.Load() != 2 {
		t.Fatalf("a recent miss must not ask again: %d requests", h.calls.Load())
	}
	// Seven days on, the miss is asked again.
	later := time.Now().Add(RetryNone + time.Hour)
	s.now = func() time.Time { return later }
	_, _, _ = s.Photo(ctx, "2002")
	if h.calls.Load() != 3 {
		t.Fatalf("an expired miss must retry: %d requests", h.calls.Load())
	}
	// A failing host is an error, never recorded as "no photo".
	if _, _, err := s.Photo(ctx, "2003"); err == nil || errors.Is(err, ErrNoPhoto) {
		t.Fatalf("HTTP 503: err %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.dir, "2003.none")); err == nil {
		t.Fatal("a host failure was cached as no photo")
	}
	if st, files, _ := s.Stats(); st.Fetched != 1 || st.Served != 2 || files != 1 || st.LastError == "" {
		t.Fatalf("stats %+v files %d", st, files)
	}
}

func TestEvictionDropsTheLeastRecentlyServed(t *testing.T) {
	h := &hosts{espn: map[string]reply{"1": {200, tiny(t, "png", 1)}, "2": {200, tiny(t, "png", 2)},
		"3": {200, tiny(t, "png", 3)}}}
	s := service(t, h, map[string][2]string{"3001": {"1", ""}, "3002": {"2", ""}, "3003": {"3", ""}})
	ctx := context.Background()
	for _, id := range []string{"3001", "3002"} {
		if _, _, err := s.Photo(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := s.Photo(ctx, "3001"); err != nil { // 3001 served again: 3002 is now the oldest
		t.Fatal(err)
	}
	_, _, size := s.Stats()
	s.cap = size // room for two
	if _, _, err := s.Photo(ctx, "3003"); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]bool{"3001": true, "3002": false, "3003": true} {
		_, err := os.Stat(filepath.Join(s.dir, id+".png"))
		if (err == nil) != want {
			t.Errorf("%s held = %v, want %v", id, err == nil, want)
		}
	}
}

func TestHandlerStatusCodes(t *testing.T) {
	h := &hosts{espn: map[string]reply{"1": {200, tiny(t, "png", 1)}, "3": {500, nil}}}
	s := service(t, h, map[string][2]string{"4001": {"1", ""}, "4002": {"2", ""}, "4003": {"3", ""}})
	on := true
	handler := s.Handler(func() bool { return on })
	for path, want := range map[string]int{"/headshot/4001": 200, "/headshot/4002": 404, "/headshot/4003": 502,
		"/headshot/../x": 400, "/headshot/12ab": 400} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		if rec.Code != want {
			t.Errorf("%s = %d, want %d", path, rec.Code, want)
		}
	}
	on = false
	before := h.calls.Load()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/headshot/4009", nil))
	if rec.Code != http.StatusServiceUnavailable || h.calls.Load() != before {
		t.Fatalf("photos off: %d with %d new requests", rec.Code, h.calls.Load()-before)
	}
}

type refused struct{}

func (refused) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
}

// With no network the photo health row reads the offline sentence, not the dial error.
func TestPhotoFailureOfflineReadsPlainly(t *testing.T) {
	s := New(t.TempDir(), func(context.Context, string) (string, string, error) { return "1", "2", nil },
		&http.Client{Transport: refused{}})
	if _, _, err := s.Photo(context.Background(), "2001"); err == nil {
		t.Fatal("a refused photo fetch succeeded")
	}
	if st, _, _ := s.Stats(); st.LastError != offline.Sentence("a.espncdn.com") {
		t.Fatalf("last error %q", st.LastError)
	}
}
