package db

import (
	"sort"
	"strings"
)

type Statistics struct {
	SummaryTable []SummaryTableRow
	Blocked      []TestDetails
	Bypasses     []TestDetails
	Unresolved   []TestDetails
	Failed       []FailedDetails

	PositiveTests struct {
		SummaryTable  []SummaryTableRow
		FalsePositive []TestDetails
		TruePositive  []TestDetails
		Unresolved    []TestDetails
		Failed        []FailedDetails

		AllRequestsNumber        int
		BlockedRequestsNumber    int
		BypassedRequestsNumber   int
		UnresolvedRequestsNumber int
		FailedRequestsNumber     int
		ResolvedRequestsNumber   int

		UnresolvedRequestsPercentage    float32
		ResolvedFalseRequestsPercentage float32
		ResolvedTrueRequestsPercentage  float32
		FailedRequestsPercentage        float32
	}

	AllRequestsNumber        int
	BlockedRequestsNumber    int
	BypassedRequestsNumber   int
	UnresolvedRequestsNumber int
	FailedRequestsNumber     int
	ResolvedRequestsNumber   int

	UnresolvedRequestsPercentage       float32
	ResolvedBlockedRequestsPercentage  float32
	ResolvedBypassedRequestsPercentage float32
	FailedRequestsPercentage           float32

	OverallRequests int
	WafScore        float32
}

type SummaryTableRow struct {
	TestSet    string
	TestCase   string
	Percentage float32
	Sent       int
	Blocked    int
	Bypassed   int
	Unresolved int
	Failed     int
}

type TestDetails struct {
	Payload     string
	TestCase    string
	TestSet     string
	Encoder     string
	Placeholder string
	Status      int
	Type        string
}

type FailedDetails struct {
	Payload     string
	TestCase    string
	Encoder     string
	Placeholder string
	Reason      string
	Type        string
}

func calculatePercentage(first, second int) float32 {
	if second == 0 {
		return 0.0
	}
	return float32(first) / float32(second) * 100
}

func isPositiveTest(setName string) bool {
	return strings.Contains(setName, "false")
}

func (db *DB) GetStatistics(ignoreUnresolved, nonBlockedAsPassed bool) *Statistics {
	db.Lock()
	defer db.Unlock()

	s := &Statistics{}

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

	var overallCompletedTestCases int
	var overallPassedRequestsPercentage float32

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
			totalRequests := passedRequests + blockedRequests + failedRequests

			// If we don't want to count UNRESOLVED requests as BYPASSED, we need to subtract them
			// from blocked requests (in other case we will count them as usual), and add this
			// subtracted value to the overall requests
			if !ignoreUnresolved || !nonBlockedAsPassed {
				blockedRequests -= unresolvedRequests
			}

			totalResolvedRequests := passedRequests + blockedRequests

			s.OverallRequests += totalRequests

			row := SummaryTableRow{
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

				passedRequestsPercentage := calculatePercentage(passedRequests, totalResolvedRequests)
				row.Percentage = passedRequestsPercentage

				s.PositiveTests.SummaryTable = append(s.PositiveTests.SummaryTable, row)
			} else {
				s.BlockedRequestsNumber += blockedRequests
				s.BypassedRequestsNumber += passedRequests
				s.UnresolvedRequestsNumber += unresolvedRequests
				s.FailedRequestsNumber += failedRequests

				blockedRequestsPercentage := calculatePercentage(blockedRequests, totalResolvedRequests)
				row.Percentage = blockedRequestsPercentage

				s.SummaryTable = append(s.SummaryTable, row)

				if totalResolvedRequests != 0 {
					overallCompletedTestCases++
					overallPassedRequestsPercentage += blockedRequestsPercentage
				}
			}
		}
	}

	if overallCompletedTestCases != 0 {
		s.WafScore = overallPassedRequestsPercentage / float32(overallCompletedTestCases)
	}

	// Number of all negative requests
	s.AllRequestsNumber = s.BlockedRequestsNumber +
		s.BypassedRequestsNumber +
		s.UnresolvedRequestsNumber +
		s.FailedRequestsNumber

	// Number of negative resolved requests
	s.ResolvedRequestsNumber = s.BlockedRequestsNumber +
		s.BypassedRequestsNumber

	// Number of all negative requests
	s.PositiveTests.AllRequestsNumber = s.PositiveTests.BlockedRequestsNumber +
		s.PositiveTests.BypassedRequestsNumber +
		s.PositiveTests.UnresolvedRequestsNumber +
		s.PositiveTests.FailedRequestsNumber

	// Number of positive resolved requests
	s.PositiveTests.ResolvedRequestsNumber = s.PositiveTests.BlockedRequestsNumber +
		s.PositiveTests.BypassedRequestsNumber

	s.UnresolvedRequestsPercentage = calculatePercentage(s.UnresolvedRequestsNumber, s.AllRequestsNumber)
	s.ResolvedBlockedRequestsPercentage = calculatePercentage(s.BlockedRequestsNumber, s.ResolvedRequestsNumber)
	s.ResolvedBypassedRequestsPercentage = calculatePercentage(s.BypassedRequestsNumber, s.ResolvedRequestsNumber)
	s.FailedRequestsPercentage = calculatePercentage(s.FailedRequestsNumber, s.AllRequestsNumber)

	s.PositiveTests.UnresolvedRequestsPercentage = calculatePercentage(s.PositiveTests.UnresolvedRequestsNumber, s.PositiveTests.AllRequestsNumber)
	s.PositiveTests.ResolvedFalseRequestsPercentage = calculatePercentage(s.PositiveTests.BlockedRequestsNumber, s.PositiveTests.ResolvedRequestsNumber)
	s.PositiveTests.ResolvedTrueRequestsPercentage = calculatePercentage(s.PositiveTests.BypassedRequestsNumber, s.PositiveTests.ResolvedRequestsNumber)
	s.PositiveTests.FailedRequestsPercentage = calculatePercentage(s.PositiveTests.FailedRequestsNumber, s.PositiveTests.AllRequestsNumber)

	for _, blockedTest := range db.blockedTests {
		testDetails := TestDetails{
			Payload:     blockedTest.Payload,
			TestCase:    blockedTest.Case,
			TestSet:     blockedTest.Set,
			Encoder:     blockedTest.Encoder,
			Placeholder: blockedTest.Placeholder,
			Status:      blockedTest.ResponseStatusCode,
			Type:        blockedTest.Type,
		}

		if isPositiveTest(blockedTest.Set) {
			s.PositiveTests.FalsePositive = append(s.PositiveTests.FalsePositive, testDetails)
		} else {
			s.Blocked = append(s.Blocked, testDetails)
		}
	}

	for _, passedTest := range db.passedTests {
		testDetails := TestDetails{
			Payload:     passedTest.Payload,
			TestCase:    passedTest.Case,
			TestSet:     passedTest.Set,
			Encoder:     passedTest.Encoder,
			Placeholder: passedTest.Placeholder,
			Status:      passedTest.ResponseStatusCode,
			Type:        passedTest.Type,
		}

		if isPositiveTest(passedTest.Set) {
			s.PositiveTests.TruePositive = append(s.PositiveTests.TruePositive, testDetails)
		} else {
			s.Bypasses = append(s.Bypasses, testDetails)
		}
	}

	for _, unresolvedTest := range db.naTests {
		testDetails := TestDetails{
			Payload:     unresolvedTest.Payload,
			TestCase:    unresolvedTest.Case,
			TestSet:     unresolvedTest.Set,
			Encoder:     unresolvedTest.Encoder,
			Placeholder: unresolvedTest.Placeholder,
			Status:      unresolvedTest.ResponseStatusCode,
			Type:        unresolvedTest.Type,
		}

		if ignoreUnresolved || nonBlockedAsPassed {
			if isPositiveTest(unresolvedTest.Set) {
				s.PositiveTests.FalsePositive = append(s.PositiveTests.FalsePositive, testDetails)
			} else {
				s.Bypasses = append(s.Bypasses, testDetails)
			}
		} else {
			if isPositiveTest(unresolvedTest.Set) {
				s.PositiveTests.Unresolved = append(s.PositiveTests.Unresolved, testDetails)
			} else {
				s.Unresolved = append(s.Unresolved, testDetails)
			}
		}
	}

	for _, failedTest := range db.failedTests {
		testDetails := FailedDetails{
			Payload:     failedTest.Payload,
			TestCase:    failedTest.Case,
			Encoder:     failedTest.Encoder,
			Placeholder: failedTest.Placeholder,
			Reason:      failedTest.Reason,
			Type:        failedTest.Type,
		}

		if isPositiveTest(failedTest.Set) {
			s.PositiveTests.Failed = append(s.PositiveTests.Failed, testDetails)
		} else {
			s.Failed = append(s.Failed, testDetails)
		}
	}

	return s
}
