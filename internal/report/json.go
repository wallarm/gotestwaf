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
	Score float64 `json:"score"`

	Summary    requestStats `json:"summary"`
	ApiSecStat requestStats `json:"api_sec"`
	AppSecStat requestStats `json:"app_sec"`

	TestSets testSets `json:"test_sets"`
}

type requestStats struct {
	TotalSent       int `json:"total_sent"`
	ResolvedTests   int `json:"resolved_tests"`
	BlockedTests    int `json:"blocked_tests"`
	BypassedTests   int `json:"bypassed_tests"`
	UnresolvedTests int `json:"unresolved_tests"`
	FailedTests     int `json:"failed_tests"`
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
	TruePositiveTests *testsInfo `json:"true_positive_tests,omitempty"`
	TrueNegativeTests *testsInfo `json:"true_negative_tests,omitempty"`
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
		report.Summary.TruePositiveTests = &testsInfo{
			Score: s.TruePositiveTests.ResolvedBlockedRequestsPercentage,
			Summary: requestStats{
				TotalSent:       s.TruePositiveTests.ReqStats.AllRequestsNumber,
				ResolvedTests:   s.TruePositiveTests.ReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TruePositiveTests.ReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TruePositiveTests.ReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TruePositiveTests.ReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TruePositiveTests.ReqStats.FailedRequestsNumber,
			},
			ApiSecStat: requestStats{
				TotalSent:       s.TruePositiveTests.ApiSecReqStats.AllRequestsNumber,
				ResolvedTests:   s.TruePositiveTests.ApiSecReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TruePositiveTests.ApiSecReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TruePositiveTests.ApiSecReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TruePositiveTests.ApiSecReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TruePositiveTests.ApiSecReqStats.FailedRequestsNumber,
			},
			AppSecStat: requestStats{
				TotalSent:       s.TruePositiveTests.AppSecReqStats.AllRequestsNumber,
				ResolvedTests:   s.TruePositiveTests.AppSecReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TruePositiveTests.AppSecReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TruePositiveTests.AppSecReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TruePositiveTests.AppSecReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TruePositiveTests.AppSecReqStats.FailedRequestsNumber,
			},
			TestSets: make(testSets),
		}
		for _, row := range s.TruePositiveTests.SummaryTable {
			if report.Summary.TruePositiveTests.TestSets[row.TestSet] == nil {
				report.Summary.TruePositiveTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.TruePositiveTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
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
		report.Summary.TrueNegativeTests = &testsInfo{
			Score: s.TrueNegativeTests.ResolvedBypassedRequestsPercentage,
			Summary: requestStats{
				TotalSent:       s.TrueNegativeTests.ReqStats.AllRequestsNumber,
				ResolvedTests:   s.TrueNegativeTests.ReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TrueNegativeTests.ReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TrueNegativeTests.ReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TrueNegativeTests.ReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TrueNegativeTests.ReqStats.FailedRequestsNumber,
			},
			ApiSecStat: requestStats{
				TotalSent:       s.TrueNegativeTests.ApiSecReqStats.AllRequestsNumber,
				ResolvedTests:   s.TrueNegativeTests.ApiSecReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TrueNegativeTests.ApiSecReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TrueNegativeTests.ApiSecReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TrueNegativeTests.ApiSecReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TrueNegativeTests.ApiSecReqStats.FailedRequestsNumber,
			},
			AppSecStat: requestStats{
				TotalSent:       s.TrueNegativeTests.AppSecReqStats.AllRequestsNumber,
				ResolvedTests:   s.TrueNegativeTests.AppSecReqStats.ResolvedRequestsNumber,
				BlockedTests:    s.TrueNegativeTests.AppSecReqStats.BlockedRequestsNumber,
				BypassedTests:   s.TrueNegativeTests.AppSecReqStats.BypassedRequestsNumber,
				UnresolvedTests: s.TrueNegativeTests.AppSecReqStats.UnresolvedRequestsNumber,
				FailedTests:     s.TrueNegativeTests.AppSecReqStats.FailedRequestsNumber,
			},
			TestSets: make(testSets),
		}
		for _, row := range s.TrueNegativeTests.SummaryTable {
			if report.Summary.TrueNegativeTests.TestSets[row.TestSet] == nil {
				report.Summary.TrueNegativeTests.TestSets[row.TestSet] = make(testCases)
			}
			report.Summary.TrueNegativeTests.TestSets[row.TestSet][row.TestCase] = &testCaseInfo{
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

	for _, blocked := range s.TrueNegativeTests.Blocked {
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
