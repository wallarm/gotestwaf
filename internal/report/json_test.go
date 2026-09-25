package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wallarm/gotestwaf/internal/db"
)

// Regression test for https://github.com/wallarm/gotestwaf/issues/285:
// the placeholder field of every payload must come from the placeholder,
// not from the encoder.
func TestJSONReportPayloadPlaceholder(t *testing.T) {
	detail := &db.TestDetails{
		Payload: "p", TestCase: "case", TestSet: "set",
		Encoder: "URL", Placeholder: "URLParam", ResponseStatusCode: 200,
	}
	failed := &db.FailedDetails{
		Payload: "p", TestCase: "case", TestSet: "set",
		Encoder: "URL", Placeholder: "URLParam", Reason: []string{"r"},
	}

	s := &db.Statistics{}
	s.TruePositiveTests.Bypasses = []*db.TestDetails{detail}
	s.TruePositiveTests.Unresolved = []*db.TestDetails{detail}
	s.TruePositiveTests.Failed = []*db.FailedDetails{failed}
	s.TrueNegativeTests.Blocked = []*db.TestDetails{detail}
	s.TrueNegativeTests.Unresolved = []*db.TestDetails{detail}
	s.TrueNegativeTests.Failed = []*db.FailedDetails{failed}

	reportFile := filepath.Join(t.TempDir(), "report.json")
	err := printFullReportToJson(s, reportFile, time.Now(), "waf", "http://example.com", nil, false)
	if err != nil {
		t.Fatalf("got an error while writing the report: %v", err)
	}

	data, err := os.ReadFile(reportFile)
	if err != nil {
		t.Fatal(err)
	}

	var report jsonReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}

	for _, payloads := range []*testPayloads{report.TruePositiveTestsPayloads, report.TrueNegativeTestsPayloads} {
		if payloads == nil {
			t.Fatal("payloads section is missing in the report")
		}
		for _, list := range [][]*payloadDetails{payloads.Blocked, payloads.Bypassed, payloads.Unresolved, payloads.Failed} {
			for _, p := range list {
				if p.Encoder != "URL" || p.Placeholder != "URLParam" {
					t.Fatalf("got encoder=%q placeholder=%q, want encoder=%q placeholder=%q", p.Encoder, p.Placeholder, "URL", "URLParam")
				}
			}
		}
	}
}
