package history

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func killRetryNote() commissioner.RawCommand {
	return commissioner.RawCommand{League: "native.test", Command: commissioner.NotePost,
		Args: commissioner.RawArgs{Note: &commissioner.RawNote{Text: "League note"}}, Expected: 1,
		Request: "kill.retry.note", Reason: "Post a note", Effective: "now", Visibility: "league"}
}

// An execute whose process is SIGKILLed mid-flight, or after commit but before any reply, and is then
// retried with the same request id lands exactly once, and the retry returns that entry's receipt.
func TestExecuteKilledMidFlightRetriesExactlyOnce(t *testing.T) {
	if path := os.Getenv("TWR_KILL_RETRY_DB"); path != "" {
		s, _ := openMoveTest(t, path)
		_, _ = fmt.Fprintln(os.Stdout, "go")
		if _, err := s.ExecuteCommissioner(t.Context(), killRetryNote(), "creator", nil); err != nil {
			_, _ = fmt.Fprintln(os.Stdout, "error", err)
		}
		_, _ = fmt.Fprintln(os.Stdout, "committed")
		time.Sleep(time.Minute) // the parent kills before any reply leaves
		return
	}
	for _, at := range []string{"go", "committed"} {
		t.Run("kill-after-"+at, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "league.db")
			s, p := openMoveTest(t, path)
			if _, err := s.ExecuteCommissioner(t.Context(), createCommand(), "creator", nil); err != nil {
				t.Fatal(err)
			}
			if err := p.Close(); err != nil {
				t.Fatal(err)
			}
			child := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestExecuteKilledMidFlightRetriesExactlyOnce$") //nolint:gosec // Fixed test binary and selector, no shell.
			child.Env = append(os.Environ(), "TWR_KILL_RETRY_DB="+path)
			out, err := child.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := child.Start(); err != nil {
				t.Fatal(err)
			}
			lines := bufio.NewScanner(out)
			for lines.Scan() && lines.Text() != at {
				if len(lines.Text()) > 5 && lines.Text()[:5] == "error" {
					t.Fatal(lines.Text())
				}
			}
			if err := child.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = child.Wait()
			s, _ = openMoveTest(t, path)
			receipt, err := s.ExecuteCommissioner(t.Context(), killRetryNote(), "creator", nil)
			if err != nil {
				t.Fatal(err)
			}
			again, err := s.ExecuteCommissioner(t.Context(), killRetryNote(), "creator", nil)
			if err != nil || again != receipt {
				t.Fatalf("second retry: %v %+v != %+v", err, again, receipt)
			}
			events, err := s.Events(t.Context(), "native.test", commissioner.Stream, 0)
			if err != nil {
				t.Fatal(err)
			}
			landed := 0
			for _, e := range events {
				if e.Payload.Commissioner.Command.Request == "kill.retry.note" {
					landed++
					if e.Payload.Commissioner.Receipt != receipt {
						t.Fatalf("retry receipt %+v, ledger %+v", receipt, e.Payload.Commissioner.Receipt)
					}
				}
			}
			if landed != 1 || len(events) != 2 {
				t.Fatalf("note landed %d times in %d events, want once in 2", landed, len(events))
			}
		})
	}
}
