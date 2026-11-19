package commissioner

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	ProblemReportCreate = "feedback.report.create"
	ProblemReportsRead  = "commish.feedback.read"
	ProblemReportKind   = "problem_report"
	ProblemReportStream = "feedback"
	ProblemReportLimit  = 2000
)

// RawProblemReport is the member-provided part of a report. Identity, release and user agent come
// from authenticated server context rather than user-entered fields.
type RawProblemReport struct {
	League        string `json:"league"`
	Text          string `json:"text"`
	Route         string `json:"route"`
	AppRelease    int    `json:"appRelease"`
	ViewportWidth int    `json:"viewportWidth"`
}

// Validate bounds and checks each report field before it reaches the league event store.
func (r RawProblemReport) Validate() error {
	if !validProblemReportLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	if !utf8.ValidString(r.Text) || strings.TrimSpace(r.Text) == "" || utf8.RuneCountInString(r.Text) > ProblemReportLimit {
		return fmt.Errorf("write a report of up to %d characters", ProblemReportLimit)
	}
	for _, char := range r.Text {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			return fmt.Errorf("report contains an unsupported character")
		}
	}
	if !validProblemReportRoute(r.Route) {
		return fmt.Errorf("screen route is invalid")
	}
	if r.AppRelease < 1 || r.AppRelease > 999999 {
		return fmt.Errorf("app release is invalid")
	}
	if r.ViewportWidth < 1 || r.ViewportWidth > 20000 {
		return fmt.Errorf("viewport width is invalid")
	}
	return nil
}

// ProblemReport is the commissioner-visible projection stored in the league's append-only event log.
type ProblemReport struct {
	ID            string    `json:"id"`
	League        string    `json:"league"`
	Principal     string    `json:"principal"`
	Text          string    `json:"text"`
	Route         string    `json:"route"`
	AppRelease    int       `json:"appRelease"`
	ViewportWidth int       `json:"viewportWidth"`
	UserAgent     string    `json:"userAgent"`
	CreatedAt     time.Time `json:"createdAt"`
}

// ProblemReportsReading is the bounded commissioner view over one league's report stream.
type ProblemReportsReading struct {
	League  string          `json:"league"`
	Reports []ProblemReport `json:"reports"`
}

// ProblemReportReceipt is the only report-submission result shown to members.
type ProblemReportReceipt struct {
	Message string `json:"message"`
}

// Validate rejects incomplete or malformed report records read from durable history.
func (r ProblemReport) Validate() error {
	if r.ID == "" || strings.TrimSpace(r.Principal) == "" ||
		!utf8.ValidString(r.UserAgent) || len(r.UserAgent) > 512 || r.CreatedAt.IsZero() {
		return fmt.Errorf("problem report record is invalid")
	}
	if err := (RawProblemReport{League: r.League, Text: r.Text, Route: r.Route,
		AppRelease: r.AppRelease, ViewportWidth: r.ViewportWidth}).Validate(); err != nil {
		return fmt.Errorf("problem report record: %w", err)
	}
	for _, char := range r.UserAgent {
		if unicode.IsControl(char) {
			return fmt.Errorf("problem report record has an invalid user agent")
		}
	}
	return nil
}

// RawProblemReportsScope selects one league's reports for an authorized commissioner.
type RawProblemReportsScope struct {
	League string `json:"league"`
}

// Validate checks the league scope for a commissioner report-list read.
func (r RawProblemReportsScope) Validate() error {
	if !validProblemReportLeague(r.League) {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

func validProblemReportLeague(league string) bool {
	return strings.HasPrefix(league, "native.") && league != "native." && len(league) <= 128 &&
		strings.TrimSpace(league) == league
}

func validProblemReportRoute(route string) bool {
	if len(route) == 0 || len(route) > 128 || route[0] == '/' || route[len(route)-1] == '/' ||
		strings.ContainsAny(route, "?#") {
		return false
	}
	segments := strings.Split(route, "/")
	if len(segments) != 2 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for _, char := range segment {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}
