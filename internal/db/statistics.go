package db

import (
	"sort"
)

type Statistics struct {
	IsGrpcAvailable    bool
	IsGraphQLAvailable bool

	Paths ScannedPaths

	TestCasesFingerprint string

	TruePositiveTests TestsSummary
	TrueNegativeTests TestsSummary

	Score struct {
		ApiSec  Score
		AppSec  Score
		Average float64
	}
}

type TestsSummary struct {
	SummaryTable []*SummaryTableRow
	Blocked      []*TestDetails
	Bypasses     []*TestDetails
	Unresolved   []*TestDetails
	Failed       []*FailedDetails

	ReqStats       RequestStats
	ApiSecReqStats RequestStats
	AppSecReqStats RequestStats

	UnresolvedRequestsPercentage       float64
	ResolvedBlockedRequestsPercentage  float64
	ResolvedBypassedRequestsPercentage float64
	FailedRequestsPercentage           float64
}

type SummaryTableRow struct {
	TestSet    string  `json:"test_set" validate:"required,printascii,max=256"`
	TestCase   string  `json:"test_case" validate:"required,printascii,max=256"`
	Percentage float64 `json:"percentage" validate:"min=0,max=100"`
	Sent       int     `json:"sent" validate:"min=0"`
	Blocked    int     `json:"blocked" validate:"min=0"`
	Bypassed   int     `json:"bypassed" validate:"min=0"`
	Unresolved int     `json:"unresolved" validate:"min=0"`
	Failed     int     `json:"failed" validate:"min=0"`
}

type TestDetails struct {
	Payload            string
	TestCase           string
	TestSet            string
	Encoder            string
	Placeholder        string
	ResponseStatusCode int
	AdditionalInfo     []string
	Type               string
}

type FailedDetails struct {
	Payload     string   `json:"payload" validate:"required"`
	TestCase    string   `json:"test_case" validate:"required,printascii"`
	TestSet     string   `json:"test_set" validate:"required,printascii"`
	Encoder     string   `json:"encoder" validate:"required,printascii"`
	Placeholder string   `json:"placeholder" validate:"required,printascii"`
	Reason      []string `json:"reason" validate:"omitempty,dive,required"`
	Type        string   `json:"type" validate:"omitempty"`
}

type RequestStats struct {
	AllRequestsNumber        int
	BlockedRequestsNumber    int
	BypassedRequestsNumber   int
	UnresolvedRequestsNumber int
	FailedRequestsNumber     int
	ResolvedRequestsNumber   int
}

type Score struct {
	TruePositive float64
	TrueNegative float64
	Average      float64
}

type Path struct {
	Method string `json:"method" validate:"required,printascii,max=32"`
	Path   string `json:"path" validate:"required,printascii,max=1024"`
}

type ScannedPaths []*Path

var _ sort.Interface = (ScannedPaths)(nil)

func (sp ScannedPaths) Len() int {
	return len(sp)
}

func (sp ScannedPaths) Less(i, j int) bool {
	if sp[i].Path > sp[j].Path {
		return false
	} else if sp[i].Path < sp[j].Path {
		return true
	}

	return sp[i].Method < sp[j].Method
}

func (sp ScannedPaths) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

func (sp ScannedPaths) Sort() {
	sort.Sort(sp)
}

func (db *DB) GetStatistics(ignoreUnresolved, nonBlockedAsPassed bool) *Statistics {
	db.Lock()
	defer db.Unlock()

	s := &Statistics{
		IsGrpcAvailable:      db.IsGrpcAvailable,
		IsGraphQLAvailable:   db.IsGraphQLAvailable,
		TestCasesFingerprint: db.Hash,
	}

	unresolvedRequestsNumber := make(map[string]map[string]int)

	for _, unresolvedTest := range db.naTests {
		if unresolvedRequestsNumber[unresolvedTest.Set] == nil {
			unresolvedRequestsNumber[unresolvedTest.Set] = make(map[string]int)
		}

		// If we want to count UNRESOLVED as BYPASSED, we shouldn't count UNRESOLVED at all
		// set it to zero by default
		if ignoreUnresolved || nonBlockedAsPassed {
			unresolvedRequestsNumber[unresolvedTest.Set][unresolvedTest.Case] = 0
		} else {
			unresolvedRequestsNumber[unresolvedTest.Set][unresolvedTest.Case]++
		}
	}

	// Sort all test sets by name
	var sortedTestSets []string
	for testSet := range db.counters {
		sortedTestSets = append(sortedTestSets, testSet)
	}
	sort.Strings(sortedTestSets)

	for _, testSet := range sortedTestSets {
		// Sort all test cases by name
		var sortedTestCases []string
		for testCase := range db.counters[testSet] {
			sortedTestCases = append(sortedTestCases, testCase)
		}
		sort.Strings(sortedTestCases)

		isFalsePositive := isFalsePositiveTest(testSet)

		for _, testCase := range sortedTestCases {
			// Number of requests for all request types for the selected testCase
			unresolvedRequests := unresolvedRequestsNumber[testSet][testCase]
			passedRequests := db.counters[testSet][testCase]["passed"]
			blockedRequests := db.counters[testSet][testCase]["blocked"]
			failedRequests := db.counters[testSet][testCase]["failed"]

			// passedRequests or blockedRequests already contains unresolvedRequests
			totalRequests := passedRequests + blockedRequests + failedRequests

			// If we don't want to count UNRESOLVED requests as BYPASSED, we need to subtract them
			// from blocked requests (in other case we will count them as usual), and add this
			// subtracted value to the overall requests
			if !ignoreUnresolved || !nonBlockedAsPassed {
				blockedRequests -= unresolvedRequests
			}

			totalResolvedRequests := passedRequests + blockedRequests

			row := &SummaryTableRow{
				TestSet:    testSet,
				TestCase:   testCase,
				Percentage: 0.0,
				Sent:       totalRequests,
				Blocked:    blockedRequests,
				Bypassed:   passedRequests,
				Unresolved: unresolvedRequests,
				Failed:     failedRequests,
			}

			// If positive set - move to another table (remove from general cases)
			if isFalsePositive {
				// False positive - blocked by the WAF (bad behavior, blockedRequests)
				s.TrueNegativeTests.ReqStats.BlockedRequestsNumber += blockedRequests
				// True positive - bypassed (good behavior, passedRequests)
				s.TrueNegativeTests.ReqStats.BypassedRequestsNumber += passedRequests
				s.TrueNegativeTests.ReqStats.UnresolvedRequestsNumber += unresolvedRequests
				s.TrueNegativeTests.ReqStats.FailedRequestsNumber += failedRequests

				passedRequestsPercentage := CalculatePercentage(passedRequests, totalResolvedRequests)
				row.Percentage = passedRequestsPercentage

				s.TrueNegativeTests.SummaryTable = append(s.TrueNegativeTests.SummaryTable, row)
			} else {
				s.TruePositiveTests.ReqStats.BlockedRequestsNumber += blockedRequests
				s.TruePositiveTests.ReqStats.BypassedRequestsNumber += passedRequests
				s.TruePositiveTests.ReqStats.UnresolvedRequestsNumber += unresolvedRequests
				s.TruePositiveTests.ReqStats.FailedRequestsNumber += failedRequests

				blockedRequestsPercentage := CalculatePercentage(blockedRequests, totalResolvedRequests)
				row.Percentage = blockedRequestsPercentage

				s.TruePositiveTests.SummaryTable = append(s.TruePositiveTests.SummaryTable, row)
			}
		}
	}

	for _, blockedTest := range db.blockedTests {
		sort.Strings(blockedTest.AdditionalInfo)

		testDetails := &TestDetails{
			Payload:            blockedTest.Payload,
			TestCase:           blockedTest.Case,
			TestSet:            blockedTest.Set,
			Encoder:            blockedTest.Encoder,
			Placeholder:        blockedTest.Placeholder,
			ResponseStatusCode: blockedTest.ResponseStatusCode,
			AdditionalInfo:     blockedTest.AdditionalInfo,
			Type:               blockedTest.Type,
		}

		if isFalsePositiveTest(blockedTest.Set) {
			s.TrueNegativeTests.Blocked = append(s.TrueNegativeTests.Blocked, testDetails)

			if isApiTest(blockedTest.Set) {
				s.TrueNegativeTests.ApiSecReqStats.BlockedRequestsNumber += 1
			} else {
				s.TrueNegativeTests.AppSecReqStats.BlockedRequestsNumber += 1
			}
		} else {
			s.TruePositiveTests.Blocked = append(s.TruePositiveTests.Blocked, testDetails)

			if isApiTest(blockedTest.Set) {
				s.TruePositiveTests.ApiSecReqStats.BlockedRequestsNumber += 1
			} else {
				s.TruePositiveTests.AppSecReqStats.BlockedRequestsNumber += 1
			}
		}
	}

	for _, passedTest := range db.passedTests {
		sort.Strings(passedTest.AdditionalInfo)

		testDetails := &TestDetails{
			Payload:            passedTest.Payload,
			TestCase:           passedTest.Case,
			TestSet:            passedTest.Set,
			Encoder:            passedTest.Encoder,
			Placeholder:        passedTest.Placeholder,
			ResponseStatusCode: passedTest.ResponseStatusCode,
			AdditionalInfo:     passedTest.AdditionalInfo,
			Type:               passedTest.Type,
		}

		if isFalsePositiveTest(passedTest.Set) {
			s.TrueNegativeTests.Bypasses = append(s.TrueNegativeTests.Bypasses, testDetails)

			if isApiTest(passedTest.Set) {
				s.TrueNegativeTests.ApiSecReqStats.BypassedRequestsNumber += 1
			} else {
				s.TrueNegativeTests.AppSecReqStats.BypassedRequestsNumber += 1
			}
		} else {
			s.TruePositiveTests.Bypasses = append(s.TruePositiveTests.Bypasses, testDetails)

			if isApiTest(passedTest.Set) {
				s.TruePositiveTests.ApiSecReqStats.BypassedRequestsNumber += 1
			} else {
				s.TruePositiveTests.AppSecReqStats.BypassedRequestsNumber += 1
			}
		}
	}

	for _, unresolvedTest := range db.naTests {
		sort.Strings(unresolvedTest.AdditionalInfo)

		testDetails := &TestDetails{
			Payload:            unresolvedTest.Payload,
			TestCase:           unresolvedTest.Case,
			TestSet:            unresolvedTest.Set,
			Encoder:            unresolvedTest.Encoder,
			Placeholder:        unresolvedTest.Placeholder,
			ResponseStatusCode: unresolvedTest.ResponseStatusCode,
			AdditionalInfo:     unresolvedTest.AdditionalInfo,
			Type:               unresolvedTest.Type,
		}

		if ignoreUnresolved || nonBlockedAsPassed {
			if isFalsePositiveTest(unresolvedTest.Set) {
				s.TrueNegativeTests.Blocked = append(s.TrueNegativeTests.Blocked, testDetails)

				if isApiTest(unresolvedTest.Set) {
					s.TrueNegativeTests.ApiSecReqStats.BlockedRequestsNumber += 1
				} else {
					s.TrueNegativeTests.AppSecReqStats.BlockedRequestsNumber += 1
				}
			} else {
				s.TruePositiveTests.Bypasses = append(s.TruePositiveTests.Bypasses, testDetails)

				if isApiTest(unresolvedTest.Set) {
					s.TruePositiveTests.ApiSecReqStats.BypassedRequestsNumber += 1
				} else {
					s.TruePositiveTests.AppSecReqStats.BypassedRequestsNumber += 1
				}
			}
		} else {
			if isFalsePositiveTest(unresolvedTest.Set) {
				s.TrueNegativeTests.Unresolved = append(s.TrueNegativeTests.Unresolved, testDetails)

				if isApiTest(unresolvedTest.Set) {
					s.TrueNegativeTests.ApiSecReqStats.UnresolvedRequestsNumber += 1
				} else {
					s.TrueNegativeTests.AppSecReqStats.UnresolvedRequestsNumber += 1
				}
			} else {
				s.TruePositiveTests.Unresolved = append(s.TruePositiveTests.Unresolved, testDetails)

				if isApiTest(unresolvedTest.Set) {
					s.TruePositiveTests.ApiSecReqStats.UnresolvedRequestsNumber += 1
				} else {
					s.TruePositiveTests.AppSecReqStats.UnresolvedRequestsNumber += 1
				}
			}
		}
	}

	for _, failedTest := range db.failedTests {
		testDetails := &FailedDetails{
			Payload:     failedTest.Payload,
			TestCase:    failedTest.Case,
			TestSet:     failedTest.Set,
			Encoder:     failedTest.Encoder,
			Placeholder: failedTest.Placeholder,
			Reason:      failedTest.AdditionalInfo,
			Type:        failedTest.Type,
		}

		if isFalsePositiveTest(failedTest.Set) {
			s.TrueNegativeTests.Failed = append(s.TrueNegativeTests.Failed, testDetails)

			if isApiTest(failedTest.Set) {
				s.TrueNegativeTests.ApiSecReqStats.FailedRequestsNumber += 1
			} else {
				s.TrueNegativeTests.AppSecReqStats.FailedRequestsNumber += 1
			}
		} else {
			s.TruePositiveTests.Failed = append(s.TruePositiveTests.Failed, testDetails)

			if isApiTest(failedTest.Set) {
				s.TruePositiveTests.ApiSecReqStats.FailedRequestsNumber += 1
			} else {
				s.TruePositiveTests.AppSecReqStats.FailedRequestsNumber += 1
			}
		}
	}

	if db.scannedPaths != nil {
		var paths ScannedPaths
		for path, methods := range db.scannedPaths {
			for method := range methods {
				paths = append(paths, &Path{
					Method: method,
					Path:   path,
				})
			}
		}

		paths.Sort()

		s.Paths = paths
	}

	calculateTestsSummaryStat(&s.TruePositiveTests)
	calculateTestsSummaryStat(&s.TrueNegativeTests)

	calculateScorePercentage(
		&s.Score.ApiSec,
		s.TruePositiveTests.ApiSecReqStats.BlockedRequestsNumber,
		s.TruePositiveTests.ApiSecReqStats.ResolvedRequestsNumber,
		s.TrueNegativeTests.ApiSecReqStats.BypassedRequestsNumber,
		s.TrueNegativeTests.ApiSecReqStats.ResolvedRequestsNumber,
	)
	calculateScorePercentage(
		&s.Score.AppSec,
		s.TruePositiveTests.AppSecReqStats.BlockedRequestsNumber,
		s.TruePositiveTests.AppSecReqStats.ResolvedRequestsNumber,
		s.TrueNegativeTests.AppSecReqStats.BypassedRequestsNumber,
		s.TrueNegativeTests.AppSecReqStats.ResolvedRequestsNumber,
	)

	var divider int
	var sum float64

	if s.Score.ApiSec.Average != -1.0 {
		divider++
		sum += s.Score.ApiSec.Average
	}
	if s.Score.AppSec.Average != -1.0 {
		divider++
		sum += s.Score.AppSec.Average
	}

	if divider != 0 {
		s.Score.Average = Round(sum / float64(divider))
	} else {
		s.Score.Average = -1.0
	}

	return s
}

func calculateTestsSummaryStat(s *TestsSummary) {
	// All requests stat
	s.ReqStats.AllRequestsNumber = s.ReqStats.BlockedRequestsNumber +
		s.ReqStats.BypassedRequestsNumber +
		s.ReqStats.UnresolvedRequestsNumber +
		s.ReqStats.FailedRequestsNumber

	s.ReqStats.ResolvedRequestsNumber = s.ReqStats.BlockedRequestsNumber +
		s.ReqStats.BypassedRequestsNumber

	// ApiSec requests stat
	s.ApiSecReqStats.AllRequestsNumber = s.ApiSecReqStats.BlockedRequestsNumber +
		s.ApiSecReqStats.BypassedRequestsNumber +
		s.ApiSecReqStats.UnresolvedRequestsNumber +
		s.ApiSecReqStats.FailedRequestsNumber

	s.ApiSecReqStats.ResolvedRequestsNumber = s.ApiSecReqStats.BlockedRequestsNumber +
		s.ApiSecReqStats.BypassedRequestsNumber

	// AppSec requests stat
	s.AppSecReqStats.AllRequestsNumber = s.AppSecReqStats.BlockedRequestsNumber +
		s.AppSecReqStats.BypassedRequestsNumber +
		s.AppSecReqStats.UnresolvedRequestsNumber +
		s.AppSecReqStats.FailedRequestsNumber

	s.AppSecReqStats.ResolvedRequestsNumber = s.AppSecReqStats.BlockedRequestsNumber +
		s.AppSecReqStats.BypassedRequestsNumber

	s.UnresolvedRequestsPercentage = CalculatePercentage(s.ReqStats.UnresolvedRequestsNumber, s.ReqStats.AllRequestsNumber)
	s.ResolvedBlockedRequestsPercentage = CalculatePercentage(s.ReqStats.BlockedRequestsNumber, s.ReqStats.ResolvedRequestsNumber)
	s.ResolvedBypassedRequestsPercentage = CalculatePercentage(s.ReqStats.BypassedRequestsNumber, s.ReqStats.ResolvedRequestsNumber)
	s.FailedRequestsPercentage = CalculatePercentage(s.ReqStats.FailedRequestsNumber, s.ReqStats.AllRequestsNumber)
}

func calculateScorePercentage(s *Score, truePosBlockedNum, truePosNum, trueNegBypassNum, trueNegNum int) {
	var (
		divider int
		sum     float64
	)

	s.TruePositive = CalculatePercentage(truePosBlockedNum, truePosNum)
	s.TrueNegative = CalculatePercentage(trueNegBypassNum, trueNegNum)

	if truePosNum != 0 {
		divider++
		sum += s.TruePositive
	} else {
		s.TruePositive = -1.0
	}

	if trueNegNum != 0 {
		divider++
		sum += s.TrueNegative
	} else {
		s.TrueNegative = -1.0
	}

	if divider != 0 {
		// If all malicious request were passed then grade is 0.
		if truePosBlockedNum == 0 {
			s.Average = 0.0
		} else {
			s.Average = Round(sum / float64(divider))
		}
	} else {
		s.Average = -1.0
	}
}
