package platform

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/accounts"
	catalog "github.com/secureprospective/TheWarRoom/internal/commandcatalog"
)

type EntryPoints struct {
	Smoke          func(string) int
	Native         func(string, string) int
	Desktop, Probe func() int
	Backups        func(io.Writer, io.Writer) int
	Restore        func(string, io.Writer, io.Writer) int
	Label          string
}

func (s EntryPoints) Run(args []string, stdout, stderr io.Writer) int {
	switch len(args) {
	case 0:
		return s.Desktop()
	case 2:
		if args[0] == "-package-smoke" {
			return s.Smoke(args[1]) //nolint:gosec // The exact two-argument case bounds this index.
		}
	case 3:
		if args[0] == "-native" {
			return s.Native(args[1], args[2]) //nolint:gosec // The exact three-argument case bounds both indices.
		}
	}
	return s.RunFlag(args, stdout, stderr)
}

func (s EntryPoints) RunFlag(args []string, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 1 && args[0] == "-probe":
		return s.Probe()
	case len(args) == 1 && args[0] == "-version":
		_, _ = fmt.Fprintln(stdout, s.Label)
		return 0
	case len(args) == 1 && args[0] == "-backups":
		return s.Backups(stdout, stderr)
	case len(args) == 2 && args[0] == "-restore":
		return s.Restore(args[1], stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "thewarroom: unknown arguments %q\nusage: thewarroom [-probe | -version | -backups | -restore <set> | -native <https-origin> <league-id>]\n", args)
		return 2
	}
}

func Probe(start, shutdown func(context.Context), failure func() error) int {
	start(context.Background())
	defer shutdown(context.Background())
	if failure() != nil {
		return 1
	}
	return 0
}

func SmokeWindow(directory string, stderr io.Writer, run func() error) int {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() || !filepath.IsAbs(directory) {
		_, _ = fmt.Fprintln(stderr, "package smoke requires an existing absolute directory")
		return 2
	}
	lock, err := AcquireInstanceLock(filepath.Join(directory, "smoke.db"))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	defer ReleaseInstanceLock(lock)
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "package smoke: %v\n", err)
		return 1
	}
	return 0
}

func DelegableCommands(commands []catalog.Command) []string {
	var out []string
	for _, command := range commands {
		if accounts.DelegableCommand(command.ID) {
			out = append(out, command.ID)
		}
	}
	return out
}

type NativeDesktop interface {
	Close()
	SetHelperCommands([]string)
	ConfigureUpdater(string, string, func()) error
	PracticeUpdate()
}

type NativeWindow struct {
	Open      func() (NativeDesktop, error)
	Commands  func() []string
	Directory func() (string, error)
	Lock      func(string) (*os.File, error)
	Run       func() error
	Quit      func()
	Version   string
	Practice  func() string
}

func (s NativeWindow) RunWindow(stderr io.Writer) int {
	desktop, err := s.Open()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "native startup requires an HTTPS origin and native league ID")
		return 2
	}
	defer desktop.Close()
	desktop.SetHelperCommands(s.Commands())
	directory, err := s.Directory()
	if err != nil {
		return 2
	}
	directory = filepath.Join(directory, "TheWarRoom", "updates")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return 2
	}
	lock, err := s.Lock(directory)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "Another native window is open. Close it and try again.")
		return 2
	}
	defer ReleaseInstanceLock(lock)
	// Missing install trust keeps the screen reachable, never an unsigned update.
	_ = desktop.ConfigureUpdater(directory, s.Version, s.Quit)
	if s.Version == "dev" && s.Practice() == "app" {
		desktop.PracticeUpdate()
	}
	if err := s.Run(); err != nil {
		_, _ = fmt.Fprintln(stderr, "native window could not start")
		return 1
	}
	return 0
}

func NativeUpdateLock(directory string, acquire func(string) (*os.File, error), pause func(time.Duration)) (*os.File, error) {
	var last error
	for range 100 {
		lock, err := acquire(filepath.Join(directory, "desktop"))
		if err == nil {
			return lock, nil
		}
		last = err
		pause(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("open desktop update lock: %w", last)
}
