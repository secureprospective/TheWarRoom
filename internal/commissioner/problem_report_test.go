package commissioner

import (
	"strings"
	"testing"
	"time"
)

func TestRawProblemReportBoundsAndMetadata(t *testing.T) {
	report := RawProblemReport{League: "native.test", Text: strings.Repeat("x", ProblemReportLimit),
		Route: "control/league", AppRelease: 12, ViewportWidth: 1440}
	if err := report.Validate(); err != nil {
		t.Fatalf("report at character limit refused: %v", err)
	}
	report.Text += "x"
	if err := report.Validate(); err == nil {
		t.Fatal("report over character limit accepted")
	}
	report.Text = "useful report"
	report.Route = "../settings"
	if err := report.Validate(); err == nil {
		t.Fatal("invalid screen route accepted")
	}
}

func TestProblemReportStoredRecordRequiresValidServerMetadata(t *testing.T) {
	report := ProblemReport{ID: "feedback.one", League: "native.test", Principal: "acct.member",
		Text: "useful report", Route: "home/overview", AppRelease: 12, ViewportWidth: 1440,
		UserAgent: "test browser", CreatedAt: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)}
	if err := report.Validate(); err != nil {
		t.Fatalf("valid report refused: %v", err)
	}
	report.UserAgent = "bad\nheader"
	if err := report.Validate(); err == nil {
		t.Fatal("control character in user agent accepted")
	}
}
