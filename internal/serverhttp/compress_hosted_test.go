package serverhttp_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/serverhttp"
)

// TestHostedReadingsGzipOnlyWhenAccepted: a reading is the same bytes compressed or not; a preview,
// which may carry a confirmation, is never compressed.
func TestHostedReadingsGzipOnlyWhenAccepted(t *testing.T) {
	f := newHostedFixture(t, true)
	bearer, proof := f.directCommissioner(t)
	send := func(raw serverhttp.RawWorkflow, encoding string) (http.Header, []byte) {
		t.Helper()
		body, err := json.Marshal(serverhttp.RawNativeRequest{Protocol: 1, Workflow: &raw})
		if err != nil {
			t.Fatal(err)
		}
		request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.server.URL+serverhttp.WorkflowPath, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+bearer)
		request.Header.Set("X-TWR-Recent-Auth", proof)
		request.Header.Set("Accept-Encoding", encoding) // set by hand, so the client never decodes for us
		response, err := f.server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		got, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("status %d %v: %s", response.StatusCode, err, got)
		}
		return response.Header, got
	}
	read := workflowRequest(t, "native.assets.read", "read", `{"league":"native.hosted"}`, "")
	plainResponse, plain := send(read, "identity")
	if plainResponse.Get("Content-Encoding") != "" || len(plain) < 1024 {
		t.Fatalf("plain reading: encoding %q, %d bytes", plainResponse.Get("Content-Encoding"), len(plain))
	}
	for _, refused := range []string{"gzip;q=0", "br"} {
		if r, _ := send(read, refused); r.Get("Content-Encoding") != "" {
			t.Fatalf("%s: compressed anyway", refused)
		}
	}
	zipped, packed := send(read, "br, gzip")
	if zipped.Get("Content-Encoding") != "gzip" || !strings.Contains(zipped.Get("Vary"), "Accept-Encoding") || len(packed) >= len(plain) {
		t.Fatalf("gzip reading: encoding %q vary %q, %d of %d bytes", zipped.Get("Content-Encoding"), zipped.Get("Vary"), len(packed), len(plain))
	}
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		t.Fatal(err)
	}
	unpacked, err := io.ReadAll(zr)
	if err != nil || !bytes.Equal(unpacked, plain) {
		t.Fatalf("decompressed reading differs from the plain one (%v)", err)
	}
	var reading serverhttp.WorkflowResponse
	if err := json.Unmarshal(plain, &reading); err != nil {
		t.Fatal(err)
	}
	var assets commissioner.NativeAssets
	if err := json.Unmarshal(reading.Data, &assets); err != nil {
		t.Fatal(err)
	}
	command, err := json.Marshal(commissioner.RawCommand{League: "native.hosted", Command: commissioner.RosterReserve,
		Args:     commissioner.RawArgs{Waiver: &commissioner.RawWaiverArgs{Player: "0100", Franchise: "franchise.01", Bucket: "ir"}},
		Expected: assets.Revision, Request: "reserve.gzip", Effective: "now", Visibility: "league"})
	if err != nil {
		t.Fatal(err)
	}
	preview, body := send(workflowRequest(t, commissioner.RosterReserve, "preview", string(command), reading.Football), "gzip")
	// Vary marks the compressing writer even below its size floor, so a preview routed through it shows.
	if preview.Get("Content-Encoding") != "" || preview.Get("Vary") != "" || len(body) == 0 {
		t.Fatalf("preview went through the compressing writer: encoding %q vary %q", preview.Get("Content-Encoding"), preview.Get("Vary"))
	}
}
