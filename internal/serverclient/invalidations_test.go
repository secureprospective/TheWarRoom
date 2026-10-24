package serverclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestInvalidationProtocolRejectsCursorsAndBoundsFrames(t *testing.T) {
	for _, body := range []string{
		"id: 99\nevent: invalidate\ndata: {\"protocol\":1}\n\n",
		"event: invalidate\ndata: {\"protocol\":2}\n\n",
		"event: ledger\ndata: {\"protocol\":1}\n\n",
		"data: " + strings.Repeat("x", 5000) + "\n\n",
	} {
		calls := 0
		response := &http.Response{Body: io.NopCloser(strings.NewReader(body))}
		_, err := consumeInvalidations(t.Context(), response, func(context.Context) error { calls++; return nil })
		if err == nil || calls != 0 {
			t.Fatalf("unvalidated event reached reread: calls=%d err=%v", calls, err)
		}
	}
}

func TestInvalidationsRereadOnlyCompleteValidatedEvents(t *testing.T) {
	response := &http.Response{Body: io.NopCloser(strings.NewReader(": keepalive\n\nevent: invalidate\ndata: {\"protocol\":1}\n\nevent: invalidate\ndata: {\"protocol\":1}"))}
	calls := 0
	retry, err := consumeInvalidations(t.Context(), response, func(context.Context) error { calls++; return nil })
	if !retry || err == nil || calls != 1 {
		t.Fatalf("calls=%d retry=%v err=%v", calls, retry, err)
	}
}
