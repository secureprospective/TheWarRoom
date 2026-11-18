package leagueservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

const BackupNowCommand = commissioner.BackupNowCommand

const BackupInProgressMessage = "A backup is already running for this league. Try again when it finishes."

var ErrBackupInProgress = errors.New("backup already running")

type RawBackupNow struct {
	League string `json:"league"`
}

func (r RawBackupNow) Validate() error { return AppLeague(r.League) }

type BackupResult struct {
	League   string    `json:"league"`
	Captured time.Time `json:"captured"`
	Verified time.Time `json:"verified"`
	Where    string    `json:"where"`
	Checked  bool      `json:"checked"`
	Message  string    `json:"message"`
}

type BackupRunner func(context.Context, string) (BackupResult, error)

// WithBackupRunner gives the hosted service the server's existing per-league backup path.
func (a *Service) WithBackupRunner(run BackupRunner) *Service {
	a.backupRunner = run
	return a
}

func (a *Service) PreviewBackupNow(ctx context.Context, raw RawBackupNow) (commissioner.Preview, error) {
	records, err := a.admitBackup(ctx, raw, "preview")
	if err != nil {
		return commissioner.Preview{}, err
	}
	return commissioner.Preview{
		Question: "Save and verify a league backup now?",
		Target: commissioner.PreviewTarget{League: raw.League, Name: commissioner.Replay(records).Name,
			Version: commissioner.ViewRevision(a.viewContext(ctx), records, 0)},
		Fields: []commissioner.PreviewField{{Label: "Off-site league backup", Before: "No manual backup started",
			BeforeSource: "Hosted server", After: "Save and check the league backup in Cloudflare R2",
			AfterSource: "Hosted server backup service", When: "now"}},
		Affects:  commissioner.PreviewAffects{Readings: []string{}, Words: "No league settings or records change"},
		Blockers: []string{}, Safeguard: "confirm",
	}, nil
}

func (a *Service) BackupNow(ctx context.Context, raw RawBackupNow) (BackupResult, error) {
	if _, err := a.admitBackup(ctx, raw, "execute"); err != nil {
		return BackupResult{}, err
	}
	result, err := a.backupRunner(ctx, raw.League)
	if err != nil {
		return BackupResult{}, fmt.Errorf("save league backup: %w", err)
	}
	if result.League != raw.League || result.Captured.IsZero() || result.Verified.IsZero() ||
		result.Where == "" || !result.Checked || result.Message == "" {
		return BackupResult{}, fmt.Errorf("the server did not return a checked backup result")
	}
	return result, nil
}

func (a *Service) admitBackup(ctx context.Context, raw RawBackupNow, mode string) ([]commissioner.Record, error) {
	if err := raw.Validate(); err != nil {
		return nil, err
	}
	if !a.hosted || a.backupRunner == nil {
		return nil, fmt.Errorf("hosted league backups are unavailable")
	}
	ctx = a.requestContext(ctx)
	if commissioner.TokenAuditID(ctx) != "" {
		return nil, fmt.Errorf("agent tokens cannot run backups")
	}
	if err := a.host.Ready(); err != nil {
		return nil, fmt.Errorf("league backup startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return nil, err //nolint:wrapcheck // Preserve the session refusal.
	}
	ctx = a.viewContext(ctx)
	records, err := a.host.CommissionerRecords(ctx, raw.League)
	if err != nil {
		return nil, fmt.Errorf("league backup permissions: %w", err)
	}
	access := commissioner.Access{Principal: principal, League: raw.League, Operation: BackupNowCommand,
		Mode: mode, Raw: commissioner.RawCommand{League: raw.League, Command: BackupNowCommand}, Records: records}
	if err := commissioner.AdmitCredential(ctx, access); err != nil {
		return nil, fmt.Errorf("admit league backup: %w", err)
	}
	if err := commissioner.RequireCommissionerOrCo(records, principal); err != nil {
		return nil, fmt.Errorf("commissioner backup permission: %w", err)
	}
	return records, nil
}
