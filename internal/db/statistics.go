package db

import (
	"sort"
)

type Statistics struct {
	IsGrpcAvailable bool

	Paths ScannedPaths

	TestCasesFingerprint string

	NegativeTests struct {
		SummaryTable []*SummaryTableRow
		Blocked      []*TestDetails
		Bypasses     []*TestDetails
		Unresolved   []*TestDetails
		Failed       []*FailedDetails

		AllRequestsNumber        int
		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
		ResolvedRequestsNumber   int

		UnresolvedRequestsPercentage       float64
		ResolvedBlockedRequestsPercentage  float64
		ResolvedBypassedRequestsPercentage float64
		FailedRequestsPercentage           float64
	}

	PositiveTests struct {
		SummaryTable  []*SummaryTableRow
		FalsePositive []*TestDetails
		TruePositive  []*TestDetails
		Unresolved    []*TestDetails
		Failed        []*FailedDetails

		AllRequestsNumber        int
		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
		ResolvedRequestsNumber   int

		UnresolvedRequestsPercentage    float64
		ResolvedFalseRequestsPercentage float64
		ResolvedTrueRequestsPercentage  float64
		FailedRequestsPercentage        float64
	}

	Score struct {
		ApiSec struct {
			TrueNegative float64
			TruePositive float64
			Average      float64
		}

		AppSec struct {
			TrueNegative float64
			TruePositive float64
			Average      float64
		}

		Average float64
	}
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

		isPositive := isPositiveTest(testSet)

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
			if isPositive {
				// False positive - blocked by the WAF (bad behavior, blockedRequests)
				s.PositiveTests.BlockedRequestsNumber += blockedRequests
				// True positive - bypassed (good behavior, passedRequests)
				s.PositiveTests.BypassedRequestsNumber += passedRequests
				s.PositiveTests.UnresolvedRequestsNumber += unresolvedRequests
				s.PositiveTests.FailedRequestsNumber += failedRequests

				passedRequestsPercentage := CalculatePercentage(passedRequests, totalResolvedRequests)
				row.Percentage = passedRequestsPercentage

				s.PositiveTests.SummaryTable = append(s.PositiveTests.SummaryTable, row)
			} else {
				s.NegativeTests.BlockedRequestsNumber += blockedRequests
				s.NegativeTests.BypassedRequestsNumber += passedRequests
				s.NegativeTests.UnresolvedRequestsNumber += unresolvedRequests
				s.NegativeTests.FailedRequestsNumber += failedRequests

				blockedRequestsPercentage := CalculatePercentage(blockedRequests, totalResolvedRequests)
				row.Percentage = blockedRequestsPercentage

				s.NegativeTests.SummaryTable = append(s.NegativeTests.SummaryTable, row)

			}
		}
	}

	// Number of all negative requests
	s.NegativeTests.AllRequestsNumber = s.NegativeTests.BlockedRequestsNumber +
		s.NegativeTests.BypassedRequestsNumber +
		s.NegativeTests.UnresolvedRequestsNumber +
		s.NegativeTests.FailedRequestsNumber

	// Number of negative resolved requests
	s.NegativeTests.ResolvedRequestsNumber = s.NegativeTests.BlockedRequestsNumber +
		s.NegativeTests.BypassedRequestsNumber

	// Number of all negative requests
	s.PositiveTests.AllRequestsNumber = s.PositiveTests.BlockedRequestsNumber +
		s.PositiveTests.BypassedRequestsNumber +
		s.PositiveTests.UnresolvedRequestsNumber +
		s.PositiveTests.FailedRequestsNumber

	// Number of positive resolved requests
	s.PositiveTests.ResolvedRequestsNumber = s.PositiveTests.BlockedRequestsNumber +
		s.PositiveTests.BypassedRequestsNumber

	s.NegativeTests.UnresolvedRequestsPercentage = CalculatePercentage(s.NegativeTests.UnresolvedRequestsNumber, s.NegativeTests.AllRequestsNumber)
	s.NegativeTests.ResolvedBlockedRequestsPercentage = CalculatePercentage(s.NegativeTests.BlockedRequestsNumber, s.NegativeTests.ResolvedRequestsNumber)
	s.NegativeTests.ResolvedBypassedRequestsPercentage = CalculatePercentage(s.NegativeTests.BypassedRequestsNumber, s.NegativeTests.ResolvedRequestsNumber)
	s.NegativeTests.FailedRequestsPercentage = CalculatePercentage(s.NegativeTests.FailedRequestsNumber, s.NegativeTests.AllRequestsNumber)

	s.PositiveTests.UnresolvedRequestsPercentage = CalculatePercentage(s.PositiveTests.UnresolvedRequestsNumber, s.PositiveTests.AllRequestsNumber)
	s.PositiveTests.ResolvedFalseRequestsPercentage = CalculatePercentage(s.PositiveTests.BlockedRequestsNumber, s.PositiveTests.ResolvedRequestsNumber)
	s.PositiveTests.ResolvedTrueRequestsPercentage = CalculatePercentage(s.PositiveTests.BypassedRequestsNumber, s.PositiveTests.ResolvedRequestsNumber)
	s.PositiveTests.FailedRequestsPercentage = CalculatePercentage(s.PositiveTests.FailedRequestsNumber, s.PositiveTests.AllRequestsNumber)

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

		if isPositiveTest(blockedTest.Set) {
			s.PositiveTests.FalsePositive = append(s.PositiveTests.FalsePositive, testDetails)
		} else {
			s.NegativeTests.Blocked = append(s.NegativeTests.Blocked, testDetails)
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

		if isPositiveTest(passedTest.Set) {
			s.PositiveTests.TruePositive = append(s.PositiveTests.TruePositive, testDetails)
		} else {
			s.NegativeTests.Bypasses = append(s.NegativeTests.Bypasses, testDetails)
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
			if isPositiveTest(unresolvedTest.Set) {
				s.PositiveTests.FalsePositive = append(s.PositiveTests.FalsePositive, testDetails)
			} else {
				s.NegativeTests.Bypasses = append(s.NegativeTests.Bypasses, testDetails)
			}
		} else {
			if isPositiveTest(unresolvedTest.Set) {
				s.PositiveTests.Unresolved = append(s.PositiveTests.Unresolved, testDetails)
			} else {
				s.NegativeTests.Unresolved = append(s.NegativeTests.Unresolved, testDetails)
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

		if isPositiveTest(failedTest.Set) {
			s.PositiveTests.Failed = append(s.PositiveTests.Failed, testDetails)
		} else {
			s.NegativeTests.Failed = append(s.NegativeTests.Failed, testDetails)
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

	var apiSecNegBlockedNum int
	var apiSecNegNum int
	var appSecNegBlockedNum int
	var appSecNegNum int

	for _, test := range s.NegativeTests.Blocked {
		if isApiTest(test.TestSet) {
			apiSecNegNum++
			apiSecNegBlockedNum++
		} else {
			appSecNegNum++
			appSecNegBlockedNum++
		}
	}
	for _, test := range s.NegativeTests.Bypasses {
		if isApiTest(test.TestSet) {
			apiSecNegNum++
		} else {
			appSecNegNum++
		}
	}

	var apiSecPosBypassNum int
	var apiSecPosNum int
	var appSecPosBypassNum int
	var appSecPosNum int

	for _, test := range s.PositiveTests.TruePositive {
		if isApiTest(test.TestSet) {
			apiSecPosNum++
			apiSecPosBypassNum++
		} else {
			appSecPosNum++
			appSecPosBypassNum++
		}
	}
	for _, test := range s.PositiveTests.FalsePositive {
		if isApiTest(test.TestSet) {
			apiSecPosNum++
		} else {
			appSecPosNum++
		}
	}

	var divider int
	var sum float64

	s.Score.ApiSec.TrueNegative = CalculatePercentage(apiSecNegBlockedNum, apiSecNegNum)
	s.Score.ApiSec.TruePositive = CalculatePercentage(apiSecPosBypassNum, apiSecPosNum)

	if apiSecNegNum != 0 {
		divider++
		sum += s.Score.ApiSec.TrueNegative
	} else {
		s.Score.ApiSec.TrueNegative = -1.0
	}

	if apiSecPosNum != 0 {
		divider++
		sum += s.Score.ApiSec.TruePositive
	} else {
		s.Score.ApiSec.TruePositive = -1.0
	}

	if divider != 0 {
		s.Score.ApiSec.Average = Round(sum / float64(divider))
	} else {
		s.Score.ApiSec.Average = -1.0
	}

	divider = 0
	sum = 0.0

	s.Score.AppSec.TrueNegative = CalculatePercentage(appSecNegBlockedNum, appSecNegNum)
	s.Score.AppSec.TruePositive = CalculatePercentage(appSecPosBypassNum, appSecPosNum)

	if appSecNegNum != 0 {
		divider++
		sum += s.Score.AppSec.TrueNegative
	} else {
		s.Score.AppSec.TrueNegative = -1.0
	}

	if appSecPosNum != 0 {
		divider++
		sum += s.Score.AppSec.TruePositive
	} else {
		s.Score.AppSec.TruePositive = -1.0
	}

	if divider != 0 {
		s.Score.AppSec.Average = Round(sum / float64(divider))
	} else {
		s.Score.AppSec.Average = -1.0
	}

	divider = 0
	sum = 0.0

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
