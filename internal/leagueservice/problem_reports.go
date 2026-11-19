package leagueservice

import (
	"context"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type problemReportUserAgentKey struct{}

// WithProblemReportUserAgent adds only the transport's User-Agent header to the report request.
func WithProblemReportUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, problemReportUserAgentKey{}, boundedUserAgent(userAgent))
}

func boundedUserAgent(value string) string {
	value = strings.ToValidUTF8(value, "")
	var bounded strings.Builder
	for _, char := range value {
		if unicode.IsControl(char) || bounded.Len()+utf8.RuneLen(char) > 512 {
			continue
		}
		bounded.WriteRune(char)
	}
	return bounded.String()
}

func reportUserAgent(ctx context.Context) string {
	value, _ := ctx.Value(problemReportUserAgentKey{}).(string)
	return value
}

// WithProblemReportStore binds the current request's league-owned history file. The UI never chooses a
// destination file; hosted composition supplies the store already selected by authenticated admission.
func (a *Service) WithProblemReportStore(store *history.Store) *Service {
	a.problemReports = store
	return a
}

// SubmitProblemReport admits a current league member, stores one private report event and returns no
// report contents to that member.
func (a *Service) SubmitProblemReport(ctx context.Context, raw commissioner.RawProblemReport) (commissioner.ProblemReportReceipt, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("validate problem report: %w", err)
	}
	if a.problemReports == nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report storage is unavailable")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report identity: %w", err)
	}
	requestContext := a.requestContext(ctx)
	access := commissioner.Access{Principal: principal, League: raw.League,
		Operation: commissioner.ProblemReportCreate, Mode: "execute"}
	if err := commissioner.AdmitCredential(requestContext, access); err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report admission: %w", err)
	}
	ctx, cancel := context.WithTimeout(requestContext, refreshTimeout)
	defer cancel()
	records, err := a.host.CommissionerRecords(ctx, raw.League)
	if err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report league membership: %w", err)
	}
	policy, err := commissioner.GovernanceState(records)
	if err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report league policy: %w", err)
	}
	if !commissioner.IsLeagueMember(policy, principal, records) {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("league membership required")
	}
	report := commissioner.ProblemReport{League: raw.League, Principal: commissioner.PrincipalFor(records, principal),
		Text: raw.Text, Route: raw.Route, AppRelease: raw.AppRelease, ViewportWidth: raw.ViewportWidth,
		UserAgent: reportUserAgent(requestContext)}
	if _, err := a.problemReports.AppendProblemReport(ctx, report); err != nil {
		return commissioner.ProblemReportReceipt{}, fmt.Errorf("problem report: %w", err)
	}
	return commissioner.ProblemReportReceipt{Message: "Sent — thanks"}, nil
}

// TargetProblemReports returns at most the newest 100 reports after checking the caller's current role.
func (a *Service) TargetProblemReports(requestContext context.Context,
	scope commissioner.RawProblemReportsScope) (commissioner.ProblemReportsReading, error) {
	if err := scope.Validate(); err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("validate problem reports scope: %w", err)
	}
	if a.problemReports == nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem report storage is unavailable")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem reports startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(requestContext), refreshTimeout)
	defer cancel()
	if err := a.admitReadingContext(ctx, commissioner.Access{League: scope.League,
		Operation: commissioner.ProblemReportsRead, Mode: "read"}); err != nil {
		return commissioner.ProblemReportsReading{}, err
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem reports identity: %w", err)
	}
	records, err := a.host.CommissionerRecords(ctx, scope.League)
	if err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem reports permissions: %w", err)
	}
	if err := commissioner.RequireCommissionerOrCo(records, principal); err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem reports permission: %w", err)
	}
	reports, err := a.problemReports.ProblemReports(ctx, scope.League)
	if err != nil {
		return commissioner.ProblemReportsReading{}, fmt.Errorf("problem reports: %w", err)
	}
	return commissioner.ProblemReportsReading{League: scope.League, Reports: reports}, nil
}
