package report

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/wallarm/gotestwaf/internal/db"
)

// jsonReport represents a data required to render a full report in JSON format.
type jsonReport struct {
	Date        string  `json:"date"`
	ProjectName string  `json:"project_name"`
	URL         string  `json:"url"`
	Score       float64 `json:"score,omitempty"`
	TestCasesFP string  `json:"fp"`
	Args        string  `json:"args"`

	// fields for console report in JSON format
	TruePositiveTests *testsInfo `json:"true_positive_tests,omitempty"`
	TrueNegativeTests *testsInfo `json:"true_negative_tests,omitempty"`

	// fields for full report in JSON format
	Summary                   *summary      `json:"summary,omitempty"`
	TruePositiveTestsPayloads *testPayloads `json:"true_positive_tests_payloads,omitempty"`
	TrueNegativeTestsPayloads *testPayloads `json:"true_negative_tests_payloads,omitempty"`
}

type testsInfo struct {
	Score           float64  `json:"score"`
	TotalSent       int      `json:"total_sent"`
	ResolvedTests   int      `json:"resolved_tests"`
	BlockedTests    int      `json:"blocked_tests"`
	BypassedTests   int      `json:"bypassed_tests"`
	UnresolvedTests int      `json:"unresolved_tests"`
	FailedTests     int      `json:"failed_tests"`
	TestSets        testSets `json:"test_sets"`
}

type testSets map[string]testCases

type testCases map[string]*testCaseInfo

type testCaseInfo struct {
	Percentage float64 `json:"percentage"`
	Sent       int     `json:"sent"`
	Blocked    int     `json:"blocked"`
	Bypassed   int     `json:"bypassed"`
	Unresolved int     `json:"unresolved"`
	Failed     int     `json:"failed"`
}

type summary struct {
	NegativeTests *testsInfo `json:"negative,omitempty"`
	PositiveTests *testsInfo `json:"positive,omitempty"`
}

type testPayloads struct {
	Blocked    []*payloadDetails `json:"blocked,omitempty"`
	Bypassed   []*payloadDetails `json:"bypassed,omitempty"`
	Unresolved []*payloadDetails `json:"unresolved,omitempty"`
	Failed     []*payloadDetails `json:"failed,omitempty"`
}

type payloadDetails struct {
	Payload     string `json:"payload"`
	TestSet     string `json:"test_set"`
	TestCase    string `json:"test_case"`
	Encoder     string `json:"encoder"`
	Placeholder string `json:"placeholder"`
	Status      int    `json:"status,omitempty"`
	TestResult  string `json:"test_result"`

	// Used for non-failed payloads
	AdditionalInformation []string `json:"additional_info,omitempty"`

	// Used for failed payloads
	Reason []string `json:"reason,omitempty"`
}

// printFullReportToJson prepares and prints a full report in JSON format to the file.
func printFullReportToJson(
	s *db.Statistics, reportFile string, reportTime time.Time,
	wafName string, url string, args []string, ignoreUnresolved bool,
) error {
	report := jsonReport{
		Date:        reportTime.Format(time.ANSIC),
		ProjectName: wafName,
		URL:         url,
		Score:       s.Score.Average,
		TestCasesFP: s.TestCasesFingerprint,
		Args:        strings.Join(args, " "),
	}

	report.Summary = &summary{}

	if len(s.TruePositiveTests.SummaryTable) != 0 {
		report.Summary.NegativeTests = &testsInfo{
			Score:           s.TruePositiveTests.ResolvedBlockedRequestsPercentage,
			TotalSent:       s.TruePositiveTests.AllRequestsNumber,
			ResolvedTests:   s.TruePositiveTests.ResolvedRequestsNumber,
			BlockedTests:    s.TruePositiveTests.BlockedRequestsNumber,
			BypassedTests:   s.TruePositiveTests.BypassedRequestsNumber,
			UnresolvedTests: s.TruePositiveTests.UnresolvedRequestsNumber,
			FailedTests:     s.TruePositiveTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.TruePositiveTests.SummaryTable {
			if report.Summary.NegativeTests.TestSets[row.TestSet] == nil {
				report.Summary.NegativeTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.NegativeTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	if len(s.TrueNegativeTests.SummaryTable) != 0 {
		report.Summary.PositiveTests = &testsInfo{
			Score:           s.TrueNegativeTests.ResolvedTrueRequestsPercentage,
			TotalSent:       s.TrueNegativeTests.AllRequestsNumber,
			ResolvedTests:   s.TrueNegativeTests.ResolvedRequestsNumber,
			BlockedTests:    s.TrueNegativeTests.BlockedRequestsNumber,
			BypassedTests:   s.TrueNegativeTests.BypassedRequestsNumber,
			UnresolvedTests: s.TrueNegativeTests.UnresolvedRequestsNumber,
			FailedTests:     s.TrueNegativeTests.FailedRequestsNumber,
			TestSets:        make(testSets),
		}
		for _, row := range s.TrueNegativeTests.SummaryTable {
			if report.Summary.PositiveTests.TestSets[row.TestSet] == nil {
				report.Summary.PositiveTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.PositiveTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
				Percentage: row.Percentage,
				Sent:       row.Sent,
				Blocked:    row.Blocked,
				Bypassed:   row.Bypassed,
				Unresolved: row.Unresolved,
				Failed:     row.Failed,
			}
		}
	}

	report.TruePositiveTestsPayloads = &testPayloads{}

	for _, bypass := range s.TruePositiveTests.Bypasses {
		bypassDetail := &payloadDetails{
			Payload:               bypass.Payload,
			TestSet:               bypass.TestSet,
			TestCase:              bypass.TestCase,
			Encoder:               bypass.Encoder,
			Placeholder:           bypass.Encoder,
			Status:                bypass.ResponseStatusCode,
			TestResult:            "failed",
			AdditionalInformation: bypass.AdditionalInfo,
		}

		report.TruePositiveTestsPayloads.Bypassed = append(report.TruePositiveTestsPayloads.Bypassed, bypassDetail)
	}
	if !ignoreUnresolved {
		for _, unresolved := range s.TruePositiveTests.Unresolved {
			unresolvedDetail := &payloadDetails{
				Payload:               unresolved.Payload,
				TestSet:               unresolved.TestSet,
				TestCase:              unresolved.TestCase,
				Encoder:               unresolved.Encoder,
				Placeholder:           unresolved.Encoder,
				Status:                unresolved.ResponseStatusCode,
				TestResult:            "unknown",
				AdditionalInformation: unresolved.AdditionalInfo,
			}

			report.TruePositiveTestsPayloads.Unresolved = append(report.TruePositiveTestsPayloads.Unresolved, unresolvedDetail)
		}
	}
	for _, failed := range s.TruePositiveTests.Failed {
		failedDetail := &payloadDetails{
			Payload:     failed.Payload,
			TestSet:     failed.TestSet,
			TestCase:    failed.TestCase,
			Encoder:     failed.Encoder,
			Placeholder: failed.Encoder,
			Reason:      failed.Reason,
		}

		report.TruePositiveTestsPayloads.Failed = append(report.TruePositiveTestsPayloads.Failed, failedDetail)
	}

	report.TrueNegativeTestsPayloads = &testPayloads{}

	for _, blocked := range s.TrueNegativeTests.FalsePositive {
		blockedDetails := &payloadDetails{
			Payload:               blocked.Payload,
			TestSet:               blocked.TestSet,
			TestCase:              blocked.TestCase,
			Encoder:               blocked.Encoder,
			Placeholder:           blocked.Encoder,
			Status:                blocked.ResponseStatusCode,
			TestResult:            "failed",
			AdditionalInformation: blocked.AdditionalInfo,
		}

		report.TrueNegativeTestsPayloads.Blocked = append(report.TrueNegativeTestsPayloads.Blocked, blockedDetails)
	}
	if !ignoreUnresolved {
		for _, unresolved := range s.TrueNegativeTests.Unresolved {
			unresolvedDetail := &payloadDetails{
				Payload:               unresolved.Payload,
				TestSet:               unresolved.TestSet,
				TestCase:              unresolved.TestCase,
				Encoder:               unresolved.Encoder,
				Placeholder:           unresolved.Encoder,
				Status:                unresolved.ResponseStatusCode,
				TestResult:            "unknown",
				AdditionalInformation: unresolved.AdditionalInfo,
			}

			report.TrueNegativeTestsPayloads.Unresolved = append(report.TrueNegativeTestsPayloads.Unresolved, unresolvedDetail)
		}
	}
	for _, failed := range s.TrueNegativeTests.Failed {
		failedDetail := &payloadDetails{
			Payload:     failed.Payload,
			TestSet:     failed.TestSet,
			TestCase:    failed.TestCase,
			Encoder:     failed.Encoder,
			Placeholder: failed.Encoder,
			Reason:      failed.Reason,
		}

		report.TrueNegativeTestsPayloads.Failed = append(report.TrueNegativeTestsPayloads.Failed, failedDetail)
	}

	jsonBytes, err := json.MarshalIndent(report, "", "    ")
	if err != nil {
		return errors.Wrap(err, "couldn't dump report to JSON")
	}

	file, err := os.Create(reportFile)
	if err != nil {
		return errors.Wrap(err, "couldn't create file")
	}
	defer file.Close()

	_, err = file.Write(jsonBytes)
	if err != nil {
		return errors.Wrap(err, "couldn't write report to file")
	}

	return nil
}
