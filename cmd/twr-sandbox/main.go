// Command twr-sandbox is a development-only launcher, never included in release bundles.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/hostedfixture"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: go run ./cmd/twr-sandbox <up|code|down> <throwaway-directory>")
	}
	dir, err := sandboxPath(args[1])
	if err != nil {
		return err
	}
	switch args[0] {
	case "up":
		return up(ctx, dir, out)
	case "down":
		return down(ctx, dir)
	case "code":
		state, err := readState(dir)
		if err != nil {
			return err
		}
		code, err := hostedfixture.Code(state.Seed, time.Now())
		if err != nil {
			return fmt.Errorf("compute code: %w", err)
		}
		if _, err := fmt.Fprintln(out, code); err != nil {
			return fmt.Errorf("print code: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown sandbox action %q", args[0])
	}
}

func sandboxPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("sandbox path: %w", err)
	}
	if err := rejectRealProfile(absolute); err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", fmt.Errorf("sandbox parent must exist: %w", err)
	}
	dir := filepath.Join(parent, filepath.Base(absolute))
	if err := rejectRealProfile(dir); err != nil {
		return "", err
	}
	if dir == string(filepath.Separator) {
		return "", fmt.Errorf("sandbox cannot be filesystem root")
	}
	if info, err := os.Lstat(dir); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return "", fmt.Errorf("sandbox must be a real directory, not a link")
	}
	return dir, nil
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func report(out io.Writer, dir string, state rawState) error {
	desktop := fmt.Sprintf("env HOME=%s XDG_CONFIG_HOME=%s XDG_DATA_HOME=%s XDG_CACHE_HOME=%s XDG_RUNTIME_DIR=%s DBUS_SESSION_BUS_ADDRESS=%s SSL_CERT_FILE=%s %s -native %s native.hosted",
		quote(filepath.Join(dir, "home")), quote(filepath.Join(dir, "config")), quote(filepath.Join(dir, "data")), quote(filepath.Join(dir, "cache")), quote(filepath.Join(dir, "runtime")), quote("unix:path="+filepath.Join(dir, "no-session-bus")), quote(filepath.Join(dir, "server.pem")), quote(filepath.Join(dir, "thewarroom")), quote(state.Address))
	_, err := fmt.Fprintf(out, "Server: %s\nLeague: native.hosted\nCommissioner login: commissioner\nPassword: %s\nTOTP secret: %s\nCurrent TOTP code: go run ./cmd/twr-sandbox code %s\nDesktop (inherits DISPLAY):\n%s\nStop and delete: go run ./cmd/twr-sandbox down %s\n", state.Address, state.Password, state.Seed, quote(dir), desktop, quote(dir))
	if err != nil {
		return fmt.Errorf("print sandbox instructions: %w", err)
	}
	return nil
}

func rejectRealProfile(dir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locate profile boundary: %w", err)
	}
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	for _, protected := range []string{filepath.Join(home, "league-archive"), filepath.Join(home, ".config"), config} {
		if dir == protected || strings.HasPrefix(dir, protected+string(filepath.Separator)) {
			return fmt.Errorf("sandbox cannot use real configuration or league-archive directories")
		}
	}
	return nil
}
