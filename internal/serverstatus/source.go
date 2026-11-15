package serverstatus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/db"
	"github.com/secureprospective/TheWarRoom/internal/offsite"
	accountstore "github.com/secureprospective/TheWarRoom/internal/store/accounts"
)

// StatusPath is the only route on the private socket.
const StatusPath = "/v1/status"

// Chain is a hash-chained history that can verify itself.
type Chain interface {
	VerifyAudit(ctx context.Context) (db.AuditHead, error)
}

// SignIns is the sign-in history: its chain, plus the resets slice 37 records in it.
type SignIns interface {
	Chain
	LastResets(ctx context.Context) (restore, incident accountstore.Reset, err error)
}

// Source is what only the running server knows: in-memory backup coverage and its own
// view of both histories. cmd/twr-server wires it in hosted mode.
type Source struct {
	// Backup reports false when off-site backups are not configured. It is the league file's
	// coverage; SignInBackup, when set, is the sign-in file's, which a hosted server keeps apart.
	Backup       func() (offsite.Status, bool)
	SignInBackup func() (offsite.Status, bool)
	// ReferenceBackup, when set, is the football reference file's coverage. It pauses nothing.
	ReferenceBackup func() (offsite.Status, bool)
	// MissingFootball, when set, names each league whose pinned football data this server lacks.
	MissingFootball func() []string
	League          Chain
	SignIns         SignIns
	Now             func() time.Time
}

// Checks computes the server's lines afresh on every request; nothing is cached.
func (s Source) Checks(ctx context.Context) []Check {
	now := s.Now()
	status, configured := s.Backup()
	checks := []Check{backupCheck(leagueBackup, status, configured, now)}
	if s.ReferenceBackup != nil {
		status, configured := s.ReferenceBackup()
		checks = append(checks, backupCheck(referenceBackup, status, configured, now))
	}
	if s.SignInBackup != nil {
		status, configured := s.SignInBackup()
		checks = append(checks, backupCheck(signInBackup, status, configured, now))
	}
	if s.MissingFootball != nil {
		if missing := s.MissingFootball(); len(missing) > 0 {
			checks = append(checks, Check{Name: "Football data", State: Problem, Detail: "Not on this server: " + strings.Join(missing, "; ") +
				". Those leagues can be viewed, but their changes wait until the data is back; the server looks for it again every minute.",
				Action: "Restore a newer football reference backup, or pin the league to football data the server holds."})
		}
	}
	checks = append(checks, s.historyCheck(ctx))
	restore, incident, err := s.SignIns.LastResets(ctx)
	if err != nil {
		return append(checks, Check{Name: "Sign-in resets", State: Problem, Detail: "Could not be read: " + err.Error() + ".",
			Action: "Call Christopher."})
	}
	if !restore.At.IsZero() {
		checks = append(checks, Check{Name: "Last restore", State: Note,
			Detail: fmt.Sprintf("%s: %s. %d accounts were signed out.", when(restore.At, now), restore.Reason, restore.Accounts)})
	}
	if !incident.At.IsZero() {
		checks = append(checks, Check{Name: "Last sign-everyone-out", State: Note,
			Detail: fmt.Sprintf("%s, reason %q, %d accounts.", when(incident.At, now), incident.Reason, incident.Accounts)})
	}
	return checks
}

// backupPart names one file's backup line on the dashboard and what pausing it stops.
type backupPart struct{ name, paused, unconfigured string }

//nolint:gochecknoglobals // fixed words, read only
var (
	leagueBackup = backupPart{name: "Backups", paused: "League changes and sign-ins are paused",
		unconfigured: "Off-site backups are not set up, so league changes and sign-ins are paused."}
	referenceBackup = backupPart{name: "Football reference backups", paused: "Nothing is paused, but a restore needs a current football reference backup",
		unconfigured: "Off-site backups are not set up for football reference data."}
	signInBackup = backupPart{name: "Sign-in backups", paused: "Sign-ins and account changes are paused (league changes are not)",
		unconfigured: "Off-site backups are not set up, so sign-ins and account changes are paused."}
)

func backupCheck(part backupPart, status offsite.Status, configured bool, now time.Time) Check {
	check := Check{Name: part.name, State: Problem}
	what := "League changes"
	if part == signInBackup {
		what = "Sign-ins"
	}
	switch {
	case !configured:
		check.Detail = part.unconfigured
		check.Action = "Christopher finishes the backup setup in BACKUPS.md."
	case status.Writable && status.Failure == "":
		check.State = OK
		check.Detail = "Last backup saved " + when(status.Captured, now) + " and checked in Cloudflare R2."
	case status.Writable:
		check.Detail = what + " still work, but the newest backup attempt failed: " + status.Failure + ". It retries every minute."
		check.Action = "Not urgent. Look again in an hour; if this line is still here, tell Christopher."
	default:
		last := "none since the server started"
		if !status.Captured.IsZero() {
			last = when(status.Captured, now)
		}
		check.Detail = part.paused + ": no backup saved and checked in the last 14 minutes. Last good backup: " + last + "."
		if status.Failure != "" {
			check.Detail += " Newest error: " + status.Failure + "."
		}
		check.Action = "Check that the house internet works. Changes resume by themselves once a backup is checked. " +
			"If it is still paused after an hour, call Christopher. Do not replace the server."
		if part == referenceBackup {
			check.Detail = part.paused + ": no football reference backup saved and checked in the last 14 minutes. Last good backup: " + last + "."
			if status.Failure != "" {
				check.Detail += " Newest error: " + status.Failure + "."
			}
			check.Action = "Check that the house internet works. If this line is still here after an hour, tell Christopher. Do not replace the server."
		}
	}
	return check
}

func (s Source) historyCheck(ctx context.Context) Check {
	check := Check{Name: "History", State: Problem, Action: "Change nothing and call Christopher today. " +
		"Do not restore a backup or sign everyone out until he has compared it with his saved checkpoints."}
	league, err := s.League.VerifyAudit(ctx)
	if err != nil {
		check.Detail = "League history does not check out: it was changed outside the app, or the file is damaged (" + err.Error() + ")."
		return check
	}
	signIns, err := s.SignIns.VerifyAudit(ctx)
	if err != nil {
		check.Detail = "Sign-in history does not check out: it was changed outside the app, or the file is damaged (" + err.Error() + ")."
		return check
	}
	return Check{Name: "History", State: OK, Detail: fmt.Sprintf(
		"League history (%d entries) and sign-in history (%d entries) check out. Every backup carries a checkpoint of both off-site.",
		league.Size, signIns.Size)}
}

// Listen opens the private socket, owner-only. It replaces only a stale socket left by a
// stopped server, never another kind of file.
func Listen(path string) (net.Listener, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode().Type() != os.ModeSocket {
			return nil, fmt.Errorf("status socket path %s holds something else", path)
		}
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove stale status socket: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect status socket: %w", err)
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on status socket: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("protect status socket: %w", err)
	}
	return listener, nil
}

// Route is one more request the private socket answers, such as a running server's own console
// commands. The socket is owner-only, so a route is as trusted as the service account.
type Route struct {
	Pattern string
	Handler http.HandlerFunc
}

// Serve answers StatusPath, and each route, on the private socket until ctx ends.
func Serve(ctx context.Context, listener net.Listener, source Source, routes ...Route) error {
	mux := http.NewServeMux()
	for _, route := range routes {
		mux.HandleFunc(route.Pattern, route.Handler)
	}
	mux.HandleFunc("GET "+StatusPath, func(w http.ResponseWriter, r *http.Request) {
		body, err := json.Marshal(Report{Version: reportVersion, Checks: source.Checks(r.Context())})
		if err != nil {
			http.Error(w, "status unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: time.Minute}
	stop := context.AfterFunc(ctx, func() { _ = server.Close() })
	defer stop()
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve status socket: %w", err)
	}
	return nil
}
